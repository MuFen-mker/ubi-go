package auth

import (
	"alexanderthegreat96/ubi-go/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	loginPageUrl = "https://connect.ubisoft.com/login?" +
		"appId=" + ubisoftAppId +
		"&lang=en-GB" +
		"&nextUrl=https%3A%2F%2Fconnect.ubisoft.com%2Flogged-in.html"
)

// browserLogin uses headless Chrome to log in via the Ubisoft web login page.
// this bypasses DataDome since a real browser engine handles JS challenges.
// it intercepts the session API response to extract auth tokens.
func (a *Auth) browserLogin(ctx context.Context, email, password string, config *config.Config) error {
	a.logger.Printf("Starting browser login for %s...", email)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.ExecPath(config.GetChromiumCommand()),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	browserCtx, timeoutCancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer timeoutCancel()

	sessionCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	chromedp.ListenTarget(browserCtx, func(ev any) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			url := e.Response.URL
			if strings.Contains(url, "/v3/profiles/sessions") && e.Type == network.ResourceTypeXHR {
				a.logger.Printf("[network] Session XHR response: %d %s", e.Response.Status, url)
				go func() {
					time.Sleep(500 * time.Millisecond)
					c := chromedp.FromContext(browserCtx)
					execCtx := cdp.WithExecutor(browserCtx, c.Target)
					body, err := network.GetResponseBody(e.RequestID).Do(execCtx)
					if err != nil {
						errCh <- fmt.Errorf("failed to get response body: %w", err)
						return
					}
					a.logger.Printf("[network] Session body length: %d", len(body))
					sessionCh <- body
				}()
			}
		}
	})

	err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.Navigate(loginPageUrl),
		chromedp.WaitVisible(`#AuthEmail`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		return fmt.Errorf("browser navigation failed: %w", err)
	}

	fillScript := fmt.Sprintf(`(() => {
		function setNativeValue(el, value) {
			const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
			nativeInputValueSetter.call(el, value);
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			el.dispatchEvent(new Event('blur', { bubbles: true }));
		}
		const emailEl = document.querySelector('#AuthEmail');
		const passEl = document.querySelector('#AuthPassword');
		if (!emailEl || !passEl) return 'fields not found';
		emailEl.focus();
		setNativeValue(emailEl, %q);
		passEl.focus();
		setNativeValue(passEl, %q);
		return 'ok';
	})()`, email, password)

	var result string
	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(fillScript, &result),
	); err != nil {
		return fmt.Errorf("failed to fill login form: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("login form fill returned: %s", result)
	}
	a.logger.Println("Filled email and password fields")

	if err := chromedp.Run(browserCtx,
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Click(`button.btn-primary`, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("failed to click login button: %w", err)
	}
	a.logger.Println("Clicked LOG IN button, waiting for session response...")

	select {
	case body := <-sessionCh:
		var userData UserData
		if err := json.Unmarshal(body, &userData); err != nil {
			return fmt.Errorf("failed to parse session response: %w", err)
		}

		userData.Expiration = time.Now().UTC().Format(time.RFC3339Nano)
		a.UserData = userData

		if err := a.saveUserData(ctx, &userData); err != nil {
			return fmt.Errorf("failed to save user data: %w", err)
		}

		a.logger.Println("Browser login successful. Session cached.")
		return nil

	case err := <-errCh:
		return fmt.Errorf("browser login error: %w", err)

	case <-browserCtx.Done():
		return fmt.Errorf("browser login timed out waiting for session response")
	}
}
