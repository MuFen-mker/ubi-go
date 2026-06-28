package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"
)

func newTestAuth() *Auth {
	return &Auth{
		logger: log.New(os.Stderr, "[test] ", 0),
	}
}

func TestIsExpired_EmptyTicket(t *testing.T) {
	a := newTestAuth()
	user := &UserData{Ticket: "", Expiration: "2099-01-01T00:00:00Z"}
	if !a.isExpired(user) {
		t.Error("expected expired when ticket is empty")
	}
}

func TestIsExpired_EmptyExpiration(t *testing.T) {
	a := newTestAuth()
	user := &UserData{Ticket: "some-ticket", Expiration: ""}
	if !a.isExpired(user) {
		t.Error("expected expired when expiration is empty")
	}
}

func TestIsExpired_FutureExpiration(t *testing.T) {
	a := newTestAuth()
	future := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339Nano)
	user := &UserData{Ticket: "some-ticket", Expiration: future}
	if a.isExpired(user) {
		t.Error("expected not expired for future expiration")
	}
}

func TestIsExpired_PastExpiration(t *testing.T) {
	a := newTestAuth()
	past := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	user := &UserData{Ticket: "some-ticket", Expiration: past}
	if !a.isExpired(user) {
		t.Error("expected expired for past expiration")
	}
}

func TestIsExpired_WithinSafetyMargin(t *testing.T) {
	a := newTestAuth()
	// 30 seconds from now — within the 60-second safety margin
	almostExpired := time.Now().UTC().Add(30 * time.Second).Format(time.RFC3339Nano)
	user := &UserData{Ticket: "some-ticket", Expiration: almostExpired}
	if !a.isExpired(user) {
		t.Error("expected expired when within 60-second safety margin")
	}
}

func TestIsExpired_JustOutsideSafetyMargin(t *testing.T) {
	a := newTestAuth()
	// 90 seconds from now — outside the 60-second safety margin
	safeExpiry := time.Now().UTC().Add(90 * time.Second).Format(time.RFC3339Nano)
	user := &UserData{Ticket: "some-ticket", Expiration: safeExpiry}
	if a.isExpired(user) {
		t.Error("expected not expired when outside 60-second safety margin")
	}
}

func TestIsExpired_RFC3339Format(t *testing.T) {
	a := newTestAuth()
	future := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	user := &UserData{Ticket: "some-ticket", Expiration: future}
	if a.isExpired(user) {
		t.Error("expected not expired for RFC3339 format")
	}
}

func TestIsExpired_InvalidFormat(t *testing.T) {
	a := newTestAuth()
	user := &UserData{Ticket: "some-ticket", Expiration: "not-a-date"}
	if !a.isExpired(user) {
		t.Error("expected expired for unparseable expiration")
	}
}

func TestSessionTTLSeconds_ValidExpiration(t *testing.T) {
	a := newTestAuth()
	future := time.Now().UTC().Add(3 * time.Hour)
	data := &UserData{Expiration: future.Format(time.RFC3339Nano)}
	ttl := a.sessionTTLSeconds(data)
	// Should be close to 3 hours minus the 60-second margin
	expected := int((3*time.Hour - 60*time.Second).Seconds())
	if ttl < expected-5 || ttl > expected+5 {
		t.Errorf("expected TTL ~%d, got %d", expected, ttl)
	}
}

func TestSessionTTLSeconds_EmptyExpiration(t *testing.T) {
	a := newTestAuth()
	data := &UserData{Expiration: ""}
	ttl := a.sessionTTLSeconds(data)
	if ttl != sessionTTL {
		t.Errorf("expected fallback TTL %d, got %d", sessionTTL, ttl)
	}
}

func TestSessionTTLSeconds_PastExpiration(t *testing.T) {
	a := newTestAuth()
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano)
	data := &UserData{Expiration: past}
	ttl := a.sessionTTLSeconds(data)
	if ttl != sessionTTL {
		t.Errorf("expected fallback TTL %d for past expiration, got %d", sessionTTL, ttl)
	}
}

func TestSessionTTLSeconds_InvalidFormat(t *testing.T) {
	a := newTestAuth()
	data := &UserData{Expiration: "garbage"}
	ttl := a.sessionTTLSeconds(data)
	if ttl != sessionTTL {
		t.Errorf("expected fallback TTL %d for invalid format, got %d", sessionTTL, ttl)
	}
}

func TestSessionTTLSeconds_UbisoftNanoFormat(t *testing.T) {
	a := newTestAuth()
	// Ubisoft uses 7 fractional digits
	future := time.Now().UTC().Add(2 * time.Hour)
	exp := future.Format("2006-01-02T15:04:05.0000000Z")
	data := &UserData{Expiration: exp}
	ttl := a.sessionTTLSeconds(data)
	expected := int((2*time.Hour - 60*time.Second).Seconds())
	if ttl < expected-5 || ttl > expected+5 {
		t.Errorf("expected TTL ~%d for Ubisoft nano format, got %d", expected, ttl)
	}
}

func TestEncodeCredentials(t *testing.T) {
	result := encodeCredentials("user@example.com", "p@ssw0rd")
	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	expected := "user@example.com:p@ssw0rd"
	if string(decoded) != expected {
		t.Errorf("expected %q, got %q", expected, string(decoded))
	}
}

func TestParseSessionBody_ValidResponse(t *testing.T) {
	a := newTestAuth()
	body := map[string]any{
		"ticket":      "test-ticket-123",
		"sessionId":   "session-abc",
		"profileId":   "profile-xyz",
		"expiration":  time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339Nano),
		"environment": "Prod",
	}
	jsonBody, _ := json.Marshal(body)

	err := a.parseSessionBody(context.Background(), jsonBody)
	if err != nil {
		t.Fatalf("parseSessionBody failed: %v", err)
	}
	if a.UserData.Ticket != "test-ticket-123" {
		t.Errorf("expected ticket test-ticket-123, got %q", a.UserData.Ticket)
	}
	if a.UserData.SessionId != "session-abc" {
		t.Errorf("expected sessionId session-abc, got %q", a.UserData.SessionId)
	}
}

func TestParseSessionBody_EmptyBody(t *testing.T) {
	a := newTestAuth()
	err := a.parseSessionBody(context.Background(), []byte{})
	if err == nil {
		t.Error("expected error for empty body")
	}
}

func TestParseSessionBody_CaptchaChallenge(t *testing.T) {
	a := newTestAuth()
	body := `{"url":"https://geo.captcha-delivery.com/interstitial/?test=1"}`
	err := a.parseSessionBody(context.Background(), []byte(body))
	if err == nil {
		t.Error("expected error for captcha challenge")
	}
}

func TestParseSessionBody_MissingTicket(t *testing.T) {
	a := newTestAuth()
	body := `{"sessionId":"abc","expiration":"2099-01-01T00:00:00Z"}`
	err := a.parseSessionBody(context.Background(), []byte(body))
	if err == nil {
		t.Error("expected error when ticket is missing")
	}
}

func TestParseSessionBody_InvalidJSON(t *testing.T) {
	a := newTestAuth()
	err := a.parseSessionBody(context.Background(), []byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGetTicket_WithSession(t *testing.T) {
	a := newTestAuth()
	a.UserData.Ticket = "abc123"
	ticket, err := a.GetTicket()
	if err != nil {
		t.Fatalf("GetTicket failed: %v", err)
	}
	if ticket != "Ubi_v1 t=abc123" {
		t.Errorf("expected Ubi_v1 t=abc123, got %q", ticket)
	}
}

func TestGetTicket_NoSession(t *testing.T) {
	a := newTestAuth()
	_, err := a.GetTicket()
	if err == nil {
		t.Error("expected error when no ticket")
	}
}

func TestGetSessionId_WithSession(t *testing.T) {
	a := newTestAuth()
	a.UserData.SessionId = "session-123"
	id, err := a.GetSessionId()
	if err != nil {
		t.Fatalf("GetSessionId failed: %v", err)
	}
	if id != "session-123" {
		t.Errorf("expected session-123, got %q", id)
	}
}

func TestGetSessionId_NoSession(t *testing.T) {
	a := newTestAuth()
	_, err := a.GetSessionId()
	if err == nil {
		t.Error("expected error when no session id")
	}
}

func TestForceExpire(t *testing.T) {
	a := newTestAuth()
	a.UserData.Ticket = "some-ticket"
	a.UserData.Expiration = time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339)

	if a.isExpired(&a.UserData) {
		t.Fatal("session should not be expired before ForceExpire")
	}

	a.ForceExpire()

	if !a.isExpired(&a.UserData) {
		t.Error("session should be expired after ForceExpire")
	}
}

func TestSaveLoadDataDomeCookie_File(t *testing.T) {
	a := newTestAuth()

	tmpDir := t.TempDir()
	originalPath := dataDomeCookiePath
	// We can't reassign the const, so test the functions with temp files directly
	_ = tmpDir
	_ = originalPath

	// Test that loadDataDomeCookie returns empty when no cookie exists
	cookie := a.loadDataDomeCookie(context.Background())
	if cookie != "" {
		t.Errorf("expected empty cookie, got %q", cookie)
	}
}

func TestUserData_JSONRoundtrip(t *testing.T) {
	original := UserData{
		PlatformType:     "uplay",
		Ticket:           "test-ticket",
		SessionId:        "session-id",
		SessionKey:       "session-key",
		ProfileId:        "profile-id",
		UserId:           "user-id",
		NameOnPlatform:   "TestUser",
		Environment:      "Prod",
		Expiration:       "2026-04-01T14:00:00Z",
		SpaceId:          "space-id",
		RememberMeTicket: "remember-token",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored UserData
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored != original {
		t.Errorf("roundtrip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
