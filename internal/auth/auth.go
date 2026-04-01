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

// had to plugin Claude and solve the auth issues here
// since datadome was actually detecting the login attempts
// don't mind the AI comments, they are very useful
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
// Strategy:
//  1. Return cached session if still valid.
//  2. Try browser login for each configured account (bypasses DataDome).
//  3. Fall back to direct API login.
//  4. If rate-limited (429), rotate through proxies with exponential backoff.
func (a *Auth) EnsureSession(ctx context.Context) error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	// Re-check after acquiring the lock — another goroutine may have
	// already refreshed the session while we were waiting.
	user, err := a.loadUserData(ctx)
	if err == nil && !a.isExpired(user) {
		a.UserData = *user
		a.logger.Println("Using cached Ubisoft session")
		return nil
	}

	a.logger.Println("Cached session missing or expired, logging in...")

	// 1. Try refreshing with the rememberMeTicket (no browser needed).
	if a.UserData.RememberMeTicket != "" {
		a.logger.Println("Attempting session refresh via rememberMeTicket...")
		if err := a.refreshSession(ctx); err == nil {
			a.logger.Println("Session refreshed successfully!")
			return nil
		} else {
			a.logger.Printf("Session refresh failed: %v", err)
		}
	}

	// 2. Browser login (handles DataDome JS challenges).
	if chromiumPath := a.config.GetChromiumCommand(); chromiumPath != "" {
		accounts := a.config.GetUbisoftAccounts()
		for i, account := range accounts {
			a.logger.Printf("Browser login attempt %d/%d (account: %s)", i+1, len(accounts), account["email"])
			if err := a.browserLogin(ctx, account["email"], account["password"]); err == nil {
				a.logger.Println("Browser login successful!")
				return nil
			} else {
				a.logger.Printf("Browser login failed: %v", err)
			}
		}
		a.logger.Println("All browser login attempts failed.")
	}

	return fmt.Errorf("all login methods exhausted")

	// regular posts to the auth endpoint stopped workig
	// since datadome started fucking people in the ass
	// now, we're using browser automation
	// leaving this here for future reference
	// // 2. Load any cached DataDome cookie so the first API attempt may succeed without
	// // needing to launch a browser at all.
	// if cookie := a.loadDataDomeCookie(ctx); cookie != "" {
	// 	a.logger.Println("Using cached DataDome cookie")
	// 	a.dataDomeCookie = cookie
	// }

	// // 3. Direct API login with rate-limit fallback.
	// if err := a.UbiLogin(ctx); err == nil {
	// 	a.logger.Println("API login successful!")
	// 	return nil
	// } else if strings.Contains(err.Error(), "429") {
	// 	a.logger.Println("Rate limited (429). Falling back to proxy rotation...")
	// 	return a.loginWithProxiesBackoff(ctx)
	// } else {
	// 	return fmt.Errorf("API login failed: %w", err)
	// }
}

// loginWithProxiesBackoff attempts proxy-based login with exponential backoff.
func (a *Auth) loginWithProxiesBackoff(ctx context.Context) error {
	const maxRounds = 3
	backoff := 30 * time.Second

	for round := 1; round <= maxRounds; round++ {
		if err := a.loginWithProxies(ctx); err == nil {
			return nil
		}
		if round < maxRounds {
			a.logger.Printf("All proxies rate-limited. Waiting %v before retry (round %d/%d)...", backoff, round, maxRounds)
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return fmt.Errorf("all accounts and proxies rate-limited after %d backoff rounds", maxRounds)
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

	rand.Shuffle(len(proxies), func(i, j int) { proxies[i], proxies[j] = proxies[j], proxies[i] })

	for i, proxy := range proxies {
		account := accounts[i%len(accounts)]
		authHeader := encodeCredentials(account["email"], account["password"])
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
	if user.Ticket == "" || user.Expiration == "" {
		return true
	}
	expTime, err := time.Parse(time.RFC3339Nano, user.Expiration)
	if err != nil {
		expTime, err = time.Parse(time.RFC3339, user.Expiration)
		if err != nil {
			return true
		}
	}
	// Treat the session as expired 60 seconds early to avoid handing out a near-dead token.
	return time.Now().UTC().After(expTime.Add(-60 * time.Second))
}

func (a *Auth) UbiLogin(ctx context.Context) error {
	authHeader, err := a.randomAccount()
	if err != nil {
		return err
	}
	return a.doLogin(ctx, a.client, authHeader, a.userAgent)
}

func (a *Auth) doLogin(ctx context.Context, client *http.Client, authHeader, userAgent string) error {
	body, status, err := a.doLoginRequest(ctx, client, authHeader, userAgent)
	if err != nil {
		return err
	}

	// DataDome intercept: JSON with a captcha-delivery.com URL.
	// Solve the interstitial in the browser, save the cookie, then retry once.
	if status == http.StatusForbidden {
		var challenge struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(body, &challenge) == nil && strings.Contains(challenge.URL, "captcha-delivery.com") {
			if a.config.GetChromiumCommand() != "" {
				a.logger.Println("DataDome challenge detected, launching browser to solve interstitial...")
				if cookie, solveErr := a.solveDataDomeInterstitial(challenge.URL); solveErr == nil {
					a.dataDomeCookie = cookie
					_ = a.saveDataDomeCookie(ctx, cookie)
					body, status, err = a.doLoginRequest(ctx, client, authHeader, userAgent)
					if err != nil {
						return err
					}
				} else {
					a.logger.Printf("DataDome interstitial solve failed: %v", solveErr)
				}
			}
		}
	}

	if status != http.StatusOK {
		return fmt.Errorf("login failed (%d): %s", status, string(body))
	}

	if err := json.Unmarshal(body, &a.UserData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if a.UserData.Ticket == "" {
		return fmt.Errorf("login response missing ticket: %s", string(body))
	}

	if err := a.saveUserData(ctx, &a.UserData); err != nil {
		return fmt.Errorf("failed to save user data: %w", err)
	}

	a.logger.Println("Ubisoft API login successful. Session cached.")
	return nil
}

// doLoginRequest fires the session POST and returns (body, statusCode, error).
func (a *Auth) doLoginRequest(ctx context.Context, client *http.Client, authHeader, userAgent string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginPlatFormUrl, strings.NewReader(`{"rememberMe":true}`))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Ubi-AppId", ubisoftAppId)
	req.Header.Set("Ubi-RequestedPlatformType", defaultPlatformType)
	req.Header.Set("Ubi-LocaleCode", "en-US")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://connect.ubisoft.com")
	req.Header.Set("Referer", "https://connect.ubisoft.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	if a.dataDomeCookie != "" {
		req.Header.Set("Cookie", "datadome="+a.dataDomeCookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func (a *Auth) saveUserData(ctx context.Context, data *UserData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	if a.cache != nil {
		ttl := a.sessionTTLSeconds(data)
		if _, err := a.cache.Put(ctx, ubisoftAccountCacheKey, string(jsonData), ttl); err == nil {
			a.logger.Printf("Session cached in Redis (TTL: %ds)", ttl)
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
	if a.UserData.Ticket == "" {
		return "", fmt.Errorf("no active session ticket")
	}
	return fmt.Sprintf("Ubi_v1 t=%s", a.UserData.Ticket), nil
}

func (a *Auth) GetSessionId() (string, error) {
	if a.UserData.SessionId == "" {
		return "", fmt.Errorf("no active session id")
	}
	return a.UserData.SessionId, nil
}

// ForceExpire expires the session everywhere — memory, Redis, and file.
// Used for testing the rememberMeTicket flow.
func (a *Auth) ForceExpire() {
	a.UserData.Expiration = time.Now().UTC().Format(time.RFC3339)
	if a.cache != nil {
		a.cache.Delete(context.Background(), ubisoftAccountCacheKey)
	}
	os.Remove(userCachePath)
	a.logger.Println("Session forcibly expired for testing (memory + Redis + file cleared)")
}

func (a *Auth) randomAccount() (string, error) {
	accounts := a.config.GetUbisoftAccounts()
	if len(accounts) == 0 {
		return "", fmt.Errorf("no ubisoft accounts configured in env")
	}
	selected := accounts[rand.Intn(len(accounts))]
	return encodeCredentials(selected["email"], selected["password"]), nil
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

// refreshSession uses the rememberMeTicket to get a new session without a browser.
// This is a lightweight API call that avoids DataDome because the ticket proves
// the user already authenticated.
func (a *Auth) refreshSession(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginPlatFormUrl, strings.NewReader(`{"rememberMe":true}`))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "rm_v1 t="+a.UserData.RememberMeTicket)
	req.Header.Set("Ubi-AppId", ubisoftAppId)
	req.Header.Set("Ubi-RequestedPlatformType", defaultPlatformType)
	req.Header.Set("Ubi-LocaleCode", "en-US")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", a.userAgent)
	if a.dataDomeCookie != "" {
		req.Header.Set("Cookie", "datadome="+a.dataDomeCookie)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &a.UserData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if a.UserData.Ticket == "" {
		return fmt.Errorf("refresh response missing ticket")
	}

	if err := a.saveUserData(ctx, &a.UserData); err != nil {
		return fmt.Errorf("failed to save user data: %w", err)
	}

	a.logger.Printf("Session refreshed. New expiration: %s", a.UserData.Expiration)
	return nil
}

// sessionTTLSeconds computes the Redis TTL from Ubisoft's actual session expiration.
// Falls back to 5 minutes if the expiration can't be parsed.
func (a *Auth) sessionTTLSeconds(data *UserData) int {
	if data.Expiration == "" {
		return sessionTTL
	}
	expTime, err := time.Parse(time.RFC3339Nano, data.Expiration)
	if err != nil {
		expTime, err = time.Parse(time.RFC3339, data.Expiration)
		if err != nil {
			return sessionTTL
		}
	}
	// Expire from Redis 60 seconds before the actual expiry (same margin as isExpired).
	ttl := int(time.Until(expTime.Add(-60 * time.Second)).Seconds())
	a.logger.Printf("[TTL] expiration=%s now=%s ttl=%ds", data.Expiration, time.Now().UTC().Format(time.RFC3339), ttl)
	if ttl < 60 {
		return sessionTTL
	}
	return ttl
}

// encodeCredentials returns the Base64-encoded "email:password" string used in Basic auth.
func encodeCredentials(email, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(email + ":" + password))
}
