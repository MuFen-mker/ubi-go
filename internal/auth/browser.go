package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

const (
	loginPageUrl = "https://connect.ubisoft.com/login?" +
		"appId=" + ubisoftAppId +
		"&lang=en-GB" +
		"&nextUrl=https://connect.ubisoft.com/logged-in.html"

	// browserLoginTimeout is the per-account hard deadline.
	// Allows time for DataDome to solve its interstitial challenge and retry the XHR.
	browserLoginTimeout = 35 * time.Second

	// chromeCleanupTimeout is the maximum time we wait for Chrome to exit gracefully.
	chromeCleanupTimeout = 4 * time.Second
)

// newBrowser launches a Rod-controlled Chrome instance.
// The Delete("enable-automation") call removes the flag that DataDome
// and other bot-detection systems use to identify automated browsers.
func (a *Auth) newBrowser(ctx context.Context, headless bool) (*rod.Browser, func(), error) {
	l := launcher.New().
		Headless(headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-infobars", "").
		Set("window-size", "1280,800").
		Delete("enable-automation").
		Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	if chromiumPath := a.config.GetChromiumCommand(); chromiumPath != "" {
		l = l.Bin(chromiumPath)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("browser launch failed: %w", err)
	}

	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("browser connect failed: %w", err)
	}

	cleanup := func() {
		go func() {
			done := make(chan struct{})
			go func() { browser.MustClose(); close(done) }()
			select {
			case <-done:
			case <-time.After(chromeCleanupTimeout):
			}
		}()
	}

	return browser, cleanup, nil
}

// newStealthPage creates a page with comprehensive anti-detection patches
// (webdriver, plugins, languages, chrome runtime, permissions, WebGL, canvas, etc.)
func (a *Auth) newStealthPage(browser *rod.Browser) (*rod.Page, error) {
	return stealth.Page(browser)
}

// browserLogin uses Rod to log in via the Ubisoft web login page.
// It intercepts the session API response to extract auth tokens.
func (a *Auth) browserLogin(ctx context.Context, email, password string) error {
	a.logger.Printf("Starting browser login for %s...", email)

	browser, cleanup, err := a.newBrowser(ctx, true)
	if err != nil {
		return err
	}
	defer cleanup()

	page, err := a.newStealthPage(browser)
	if err != nil {
		return err
	}

	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		return fmt.Errorf("failed to enable network events: %w", err)
	}

	sessionCh := make(chan []byte, 4)
	errCh := make(chan error, 4)

	loginCtx, loginCancel := context.WithTimeout(ctx, browserLoginTimeout)
	defer loginCancel()

	// Scope the page to the login context so EachEvent stops when it expires.
	page = page.Context(loginCtx)

	go page.EachEvent(
		func(e *proto.NetworkResponseReceived) {
			// Document-level 403: DataDome blocking the page navigation itself.
			if e.Type == proto.NetworkResourceTypeDocument &&
				e.Response.Status == 403 &&
				strings.Contains(e.Response.URL, "connect.ubisoft.com") {
				a.logger.Printf("[network] Page blocked (403): %s", e.Response.URL)
				select {
				case errCh <- fmt.Errorf("page blocked by DataDome (403): %s", e.Response.URL):
				default:
				}
				return
			}

			// Session API response — accept both XHR and Fetch.
			if !strings.Contains(e.Response.URL, "/v3/profiles/sessions") ||
				(e.Type != proto.NetworkResourceTypeXHR && e.Type != proto.NetworkResourceTypeFetch) {
				return
			}

			a.logger.Printf("[network] Session response (%s): %d %s", e.Type, e.Response.Status, e.Response.URL)

			status := e.Response.Status
			requestID := e.RequestID

			go func() {
				time.Sleep(200 * time.Millisecond)
				result, err := (proto.NetworkGetResponseBody{RequestID: requestID}).Call(page)
				if status != 200 {
					// Read the body to check if DataDome is handling it.
					if err == nil && strings.Contains(result.Body, "captcha-delivery.com") {
						// DataDome returned an interstitial challenge. Its client-side JS
						// on the page will solve it automatically and retry the XHR.
						// Don't treat this as an error — wait for the retried 200.
						a.logger.Printf("[network] DataDome interstitial on session XHR (status %d), waiting for automatic retry...", status)
						return
					}
					msg := fmt.Sprintf("session endpoint returned %d", status)
					if err == nil {
						a.logger.Printf("[network] %d body: %s", status, result.Body)
						msg = fmt.Sprintf("session endpoint returned %d: %s", status, result.Body)
					}
					select {
					case errCh <- fmt.Errorf("%s", msg):
					case <-loginCtx.Done():
					}
					return
				}
				if err != nil {
					select {
					case errCh <- fmt.Errorf("failed to read session body: %w", err):
					case <-loginCtx.Done():
					}
					return
				}
				select {
				case sessionCh <- []byte(result.Body):
				case <-loginCtx.Done():
				}
			}()
		},
	)()

	// Navigate — rod's Navigate fires the CDP command and returns as soon as
	// the frame starts navigating, without waiting for full page load.
	if err := page.Navigate(loginPageUrl); err != nil {
		return fmt.Errorf("browser navigation failed: %w", err)
	}

	// Wait for email field to appear, then fill the form.
	if _, err := page.Timeout(10 * time.Second).Element("#AuthEmail"); err != nil {
		return fmt.Errorf("login form not visible: %w", err)
	}
	time.Sleep(time.Second)

	// Run form fill in a goroutine so we can simultaneously listen for the
	// session response. DataDome may auto-retry the XHR after solving its
	// interstitial, delivering a 200 while fillLoginForm is still working.
	go func() {
		if err := a.fillLoginForm(page, email, password); err != nil {
			select {
			case errCh <- fmt.Errorf("form fill: %w", err):
			default:
			}
		}
	}()

	select {
	case body := <-sessionCh:
		return a.parseSessionBody(ctx, body)
	case err := <-errCh:
		a.logger.Printf("[select] Received error: %v", err)
		return fmt.Errorf("browser login error: %w", err)
	case <-loginCtx.Done():
		return fmt.Errorf("browser login timed out (%v)", browserLoginTimeout)
	}
}

// fillLoginForm types credentials one field at a time, then clicks submit.
func (a *Auth) fillLoginForm(page *rod.Page, email, password string) error {
	// Email — find, click, type
	emailEl, err := page.Element("#AuthEmail")
	if err != nil {
		return fmt.Errorf("email field not found: %w", err)
	}
	if err := emailEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click email: %w", err)
	}
	time.Sleep(time.Second)
	if err := emailEl.Type(stringToKeys(email)...); err != nil {
		return fmt.Errorf("failed to type email: %w", err)
	}
	time.Sleep(time.Second)

	// Password — find, focus, type
	passEl, err := page.Element("#AuthPassword")
	if err != nil {
		return fmt.Errorf("password field not found: %w", err)
	}
	if err := passEl.Focus(); err != nil {
		return fmt.Errorf("failed to focus password: %w", err)
	}
	time.Sleep(time.Second)
	if err := passEl.Type(stringToKeys(password)...); err != nil {
		return fmt.Errorf("failed to type password: %w", err)
	}
	time.Sleep(time.Second)

	// Submit via JS. The first click triggers DataDome's interstitial challenge;
	// its client-side JS solves it and sets the datadome cookie. The second click
	// (after a delay) sends the request with the valid cookie → 200.
	submitJS := `() => {
		const btn = document.querySelector('button[type="submit"]')
			|| document.querySelector('input[type="submit"]')
			|| document.querySelector('form button');
		if (btn) { btn.click(); return true; }
		const form = document.querySelector('form');
		if (form) { form.requestSubmit(); return true; }
		return false;
	}`

	a.logger.Println("Submitting form (1st click — triggers DataDome)...")
	if _, err = page.Eval(submitJS); err != nil {
		return fmt.Errorf("failed to submit form: %w", err)
	}

	// Wait for DataDome to solve the interstitial and set the cookie.
	time.Sleep(5 * time.Second)

	a.logger.Println("Submitting form (2nd click — with datadome cookie)...")
	if _, err = page.Eval(submitJS); err != nil {
		return fmt.Errorf("failed to re-submit form: %w", err)
	}
	return nil
}

// stringToKeys converts a string into a slice of input.Key for use with element.Type.
func stringToKeys(s string) []input.Key {
	keys := make([]input.Key, 0, len(s))
	for _, r := range s {
		keys = append(keys, input.Key(r))
	}
	return keys
}

// solveDataDomeInterstitial opens the DataDome interstitial URL in a real browser,
// waits for the JS challenge to complete, and returns the resulting datadome cookie.
// Uses context.Background() so it is never cancelled by the caller's request context.
func (a *Auth) solveDataDomeInterstitial(interstitialURL string) (string, error) {
	a.logger.Println("Solving DataDome interstitial via browser...")

	// Deliberately NOT using the caller's ctx: the interstitial solve must run
	// to completion regardless of request timeouts or connection drops.
	solveCtx, solveCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer solveCancel()

	browser, cleanup, err := a.newBrowser(solveCtx, true)
	if err != nil {
		return "", err
	}
	defer cleanup()

	page, err := a.newStealthPage(browser)
	if err != nil {
		return "", err
	}

	// Navigate to the DataDome interstitial. Rod fires the CDP navigate command
	// and returns immediately; we then sleep to let DataDome's JS challenge run.
	if err := page.Navigate(interstitialURL); err != nil {
		// Navigation may return an error if DataDome redirects to the API endpoint
		// (which is not a normal HTML page). Ignore it — we just need the cookie.
		a.logger.Printf("Interstitial navigation returned: %v", err)
	}

	// Give DataDome's JS enough time to fingerprint the browser and set the cookie.
	time.Sleep(10 * time.Second)

	cookies, err := page.Cookies([]string{"https://public-ubiservices.ubi.com"})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	for _, c := range cookies {
		if c.Name == "datadome" {
			a.logger.Printf("DataDome cookie obtained (domain: %s)", c.Domain)
			return c.Value, nil
		}
	}

	return "", fmt.Errorf("datadome cookie not found after interstitial")
}

// parseSessionBody validates and saves the session JSON body returned by the network listener.
func (a *Auth) parseSessionBody(ctx context.Context, body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("empty session response body")
	}

	type sessionResponse struct {
		UserData
		URL string `json:"url"`
	}

	var resp sessionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse session response: %w", err)
	}
	if resp.URL != "" {
		return fmt.Errorf("captcha challenge detected (url: %s)", resp.URL)
	}
	if resp.Ticket == "" {
		return fmt.Errorf("session response missing ticket: %s", string(body))
	}

	a.UserData = resp.UserData
	if err := a.saveUserData(ctx, &a.UserData); err != nil {
		return fmt.Errorf("failed to save user data: %w", err)
	}

	a.logger.Println("Browser login successful. Session cached.")
	return nil
}

// saveDataDomeCookie persists the datadome cookie to Redis (if available) and file.
func (a *Auth) saveDataDomeCookie(ctx context.Context, value string) error {
	expires := time.Now().UTC().Add(dataDomeCookieTTL * time.Second)

	if a.cache != nil {
		if _, err := a.cache.Put(ctx, dataDomeCookieCacheKey, value, dataDomeCookieTTL); err != nil {
			a.logger.Println("Redis unavailable for datadome cookie; using file only")
		}
	}

	payload, err := json.Marshal(dataDomeCookieFile{Value: value, Expires: expires})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dataDomeCookiePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(dataDomeCookiePath, payload, 0644)
}

// loadDataDomeCookie retrieves a previously cached datadome cookie from Redis or file.
// Returns "" if no valid cached cookie exists.
func (a *Auth) loadDataDomeCookie(ctx context.Context) string {
	if a.cache != nil {
		if val, err := a.cache.Get(ctx, dataDomeCookieCacheKey); err == nil && val.Data != "" {
			return val.Data
		}
	}

	data, err := os.ReadFile(dataDomeCookiePath)
	if err != nil {
		return ""
	}
	var f dataDomeCookieFile
	if err := json.Unmarshal(data, &f); err != nil || time.Now().UTC().After(f.Expires) {
		return ""
	}
	return f.Value
}
