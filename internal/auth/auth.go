package auth

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewAuth(logger *log.Logger, cache *cache.Cache, config *config.Config) *Auth {
	client, err := utils.NewBrowserClient("")
	if err != nil {
		logger.Printf("Warning: failed to create browser client: %v", err)
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Auth{
		logger:    logger,
		cache:     cache,
		config:    config,
		UserData:  UserData{},
		client:    client,
		userAgent: utils.GetRandomUserAgent(),
	}
}

// EnsureSession checks for a cached session and logs in if needed.
// On 429 (rate limited), it falls back to proxies from proxy.txt,
// and if all proxies are also rate-limited, waits and retries with backoff.
func (a *Auth) EnsureSession(ctx context.Context) error {
	const maxRetries = 1
	const retryDelay = time.Second * 2
	const maxBackoffRounds = 3
	backoffDuration := 30 * time.Second

	user, err := a.loadUserData(ctx)
	if err == nil && !a.isExpired(user) {
		a.UserData = *user
		a.logger.Println("Using cached Ubisoft session")
		return nil
	}

	a.logger.Println("Cached session missing or expired, logging in...")

	// headless browser login
	accounts := a.config.GetUbisoftAccounts()
	if len(accounts) > 0 {
		for i, account := range accounts {
			a.logger.Printf("Browser login attempt %d/%d (account: %s)", i+1, len(accounts), account["email"])
			if err := a.browserLogin(ctx, account["email"], account["password"], a.config); err != nil {
				a.logger.Printf("Browser login failed: %v", err)
				continue
			}
			a.logger.Println("Login successful via browser!")
			return nil
		}
		a.logger.Println("All browser login attempts failed. Falling back to API login...")
	}

	// direct api login
	var lastError error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := a.UbiLogin(ctx)
		if err == nil {
			a.logger.Println("Login successful!")
			return nil
		}
		lastError = err
		a.logger.Printf("API login attempt %d/%d failed: %v", attempt, maxRetries, err)

		// 429 is the rate limiting error code
		// i was lazy, should have return the error code in the above function
		// but it is what it is
		if strings.Contains(err.Error(), "429") {
			a.logger.Println("Rate limited (429). Falling back to proxy rotation...")

			for round := 1; round <= maxBackoffRounds; round++ {
				if err := a.loginWithProxies(ctx); err == nil {
					return nil
				}
				if round < maxBackoffRounds {
					a.logger.Printf("All proxies/accounts rate-limited. Waiting %v before retry (round %d/%d)...",
						backoffDuration, round, maxBackoffRounds)
					time.Sleep(backoffDuration)
					backoffDuration *= 2 // increase backoff
				}
			}
			return fmt.Errorf("failed to login: all accounts and proxies rate-limited after %d backoff rounds", maxBackoffRounds)
		}

		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to login after %d attempts: %w", maxRetries, lastError)
}

func (a *Auth) loginWithProxies(ctx context.Context) error {
	proxies := a.config.GetProxies()
	if len(proxies) == 0 {
		return fmt.Errorf("rate limited and no proxies configured in proxy.txt")
	}

	accounts := a.config.GetUbisoftAccounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no ubisoft accounts configured")
	}

	rand.Shuffle(len(proxies), func(i, j int) {
		proxies[i], proxies[j] = proxies[j], proxies[i]
	})

	for i, proxy := range proxies {
		account := accounts[i%len(accounts)]
		creds := fmt.Sprintf("%s:%s", account["email"], account["password"])
		authHeader := base64.StdEncoding.EncodeToString([]byte(creds))

		ua := utils.GetRandomUserAgent()

		a.logger.Printf("Trying proxy %d/%d: %s (account: %s)", i+1, len(proxies), proxy, account["email"])

		proxyClient, err := utils.NewBrowserClient(proxy)
		if err != nil {
			a.logger.Printf("Failed to create proxy client: %v", err)
			continue
		}

		if err := a.doLogin(ctx, proxyClient, authHeader, ua); err != nil {
			a.logger.Printf("Proxy login failed: %v", err)

			time.Sleep(time.Second)
			continue
		}

		a.client = proxyClient
		a.userAgent = ua
		a.logger.Printf("Login successful via proxy: %s", proxy)
		return nil
	}

	return fmt.Errorf("failed to login through all %d proxies", len(proxies))
}

func (a *Auth) isExpired(user *UserData) bool {
	expTime, err := time.Parse(time.RFC3339Nano, user.Expiration)
	if err != nil {
		return true
	}
	return time.Since(expTime) > sessionTTL*time.Second
}

func (a *Auth) UbiLogin(ctx context.Context) error {
	authHeader := a.randomAccount()
	if authHeader == "" {
		return fmt.Errorf("no valid account credentials found")
	}
	return a.doLogin(ctx, a.client, authHeader, a.userAgent)
}

func (a *Auth) doLogin(ctx context.Context, client *http.Client, authHeader, userAgent string) error {
	reqBody := strings.NewReader(`{"rememberMe":true}`)
	req, err := http.NewRequest("POST", loginPlatFormUrl, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authHeader))
	req.Header.Set("Ubi-AppId", ubisoftAppId)
	req.Header.Set("Ubi-RequestedPlatformType", defaultPlatformType)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed (%d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &a.UserData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	a.UserData.Expiration = time.Now().UTC().Format(time.RFC3339Nano)

	if err := a.saveUserData(ctx, &a.UserData); err != nil {
		return fmt.Errorf("failed to save user data: %w", err)
	}

	a.logger.Println("Ubisoft login successful. Session cached for 5 minutes.")
	return nil
}

func (a *Auth) saveUserData(ctx context.Context, data *UserData) error {
	jsonData, _ := json.Marshal(data)

	if a.cache != nil {
		if _, err := a.cache.Put(ctx, ubisoftAccountCacheKey, string(jsonData), sessionTTL); err == nil {
			return nil
		}
		a.logger.Println("Redis unavailable; falling back to file system")
	}

	if err := os.MkdirAll(filepath.Dir(userCachePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(userCachePath, jsonData, 0644)
}

func (a *Auth) loadUserData(ctx context.Context) (*UserData, error) {
	if a.cache != nil {
		if val, err := a.cache.Get(ctx, ubisoftAccountCacheKey); err == nil && val.Data != "" {
			var user UserData
			if err := json.Unmarshal([]byte(val.Data), &user); err == nil {
				return &user, nil
			}
		}
		a.logger.Println("Redis unavailable or empty; falling back to file system")
	}

	data, err := os.ReadFile(userCachePath)
	if err != nil {
		return nil, err
	}

	var user UserData
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (a *Auth) EnsureValidSession(ctx context.Context) error {
	if a.UserData.Ticket != "" && !a.isExpired(&a.UserData) {
		return nil
	}
	a.logger.Println("Session expired or missing, re-authenticating...")
	a.EnsureRedisConsistency(ctx)
	return a.EnsureSession(ctx)
}

func (a *Auth) GetTicket() (string, error) {
	if a.UserData.Ticket != "" {
		return fmt.Sprintf("Ubi_v1 t=%s", a.UserData.Ticket), nil
	}
	return "", fmt.Errorf("unable to read user ticket")
}

func (a *Auth) GetSessionId() (string, error) {
	if a.UserData.Ticket != "" {
		return a.UserData.SessionId, nil
	}
	return "", fmt.Errorf("unable to read session id")
}

func (a *Auth) randomAccount() string {
	accounts := a.config.GetUbisoftAccounts()

	if len(accounts) == 0 {
		a.logger.Fatalln("Unable to load any ubisoft accounts from env.")
	}

	randomIndex := rand.Intn(len(accounts))
	selected := accounts[randomIndex]
	creds := fmt.Sprintf("%s:%s", selected["email"], selected["password"])
	return base64.StdEncoding.EncodeToString([]byte(creds))
}

func (a *Auth) EnsureRedisConsistency(ctx context.Context) {
	if a.cache == nil || a.UserData.Ticket == "" || a.isExpired(&a.UserData) {
		return
	}
	if val, err := a.cache.Get(ctx, ubisoftAccountCacheKey); err != nil || val.Data == "" {
		a.logger.Println("Redis session missing, repopulating from in-memory session...")
		a.saveUserData(ctx, &a.UserData)
	}
}
