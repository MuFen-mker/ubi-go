package profile

import (
	"encoding/json"
	"testing"
)

func TestUserProfile_JSONRoundtrip(t *testing.T) {
	original := UserProfile{
		IdOnPlatform:   "id-123",
		NameOnPlatform: "TestPlayer",
		PlatformType:   "uplay",
		ProfileId:      "profile-abc",
		UserId:         "user-xyz",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored UserProfile
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored != original {
		t.Errorf("roundtrip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}

func TestUserProfile_JSONFields(t *testing.T) {
	p := UserProfile{
		IdOnPlatform:   "id",
		NameOnPlatform: "name",
		PlatformType:   "uplay",
		ProfileId:      "pid",
		UserId:         "uid",
	}
	data, _ := json.Marshal(p)
	var raw map[string]string
	json.Unmarshal(data, &raw)

	expectedKeys := []string{"IdOnPlatform", "NameOnPlatform", "PlatformType", "ProfileId", "UserId"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q not found", key)
		}
	}
}

func TestUserProfile_EmptyFields(t *testing.T) {
	p := UserProfile{}
	if p.IdOnPlatform != "" || p.NameOnPlatform != "" || p.PlatformType != "" {
		t.Error("zero-value UserProfile should have empty string fields")
	}
}

func TestNewService(t *testing.T) {
	// Can create a service with nil dependencies (used without cache/global in tests)
	s := NewService(nil, nil, nil, nil)
	if s == nil {
		t.Fatal("NewService returned nil")
	}
}
