package global

import (
	"alexanderthegreat96/ubi-go/internal/auth"
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
)

func NewService(logger *log.Logger, authService *auth.Auth, cache *cache.Cache, cfg *config.Config) *GlobalService {
	client, err := utils.NewBrowserClient("")
	if err != nil {
		logger.Printf("Warning: failed to create browser client: %v", err)
		client = &http.Client{}
	}
	return &GlobalService{
		auth:   authService,
		logger: logger,
		cache:  cache,
		client: client,
		config: cfg,
	}
}

// function to make requests to the ubisoft API
func (s *GlobalService) MakeRequest(ctx context.Context, method, url string, body any, customHeaders map[string]string) (map[string]any, error) {
	return s.doRequest(ctx, method, url, body, true, customHeaders)
}

func (s *GlobalService) getTicket() (string, error) {
	ticket, err := s.auth.GetTicket()
	if err != nil {
		return "", err
	}

	return ticket, nil
}

func (s *GlobalService) doRequest(ctx context.Context, method, url string, body any, retryOn401 bool, customHeaders map[string]string) (map[string]any, error) {
	if err := s.auth.EnsureValidSession(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure valid session: %w", err)
	}

	ticket, err := s.getTicket()
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	sessionId, err := s.auth.GetSessionId()
	if err != nil {
		return nil, fmt.Errorf("failed to get session ID: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	headers := map[string]string{
		"Authorization":  ticket,
		"Ubi-AppId":      ubisoftAppId,
		"Ubi-SessionId":  sessionId,
		"Ubi-LocaleCode": "en-US",
		"Content-Type":   "application/json; charset=utf-8",
		"User-Agent":     utils.GetRandomUserAgent(),
		"Origin":         "https://connect.ubisoft.com",
		"Referer":        "https://connect.ubisoft.com",
	}

	maps.Copy(headers, customHeaders)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == 401 && retryOn401 {
		s.logger.Println("Got 401 from Ubisoft API, re-authenticating and retrying...")
		if err := s.auth.EnsureSession(ctx); err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}
		return s.doRequest(ctx, method, url, body, false, customHeaders)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s — %s", resp.StatusCode, resp.Status, string(respBody))
	}

	var data map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, fmt.Errorf("failed to parse response JSON: %w", err)
		}
	}

	return data, nil
}
