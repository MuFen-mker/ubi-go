package utils

import (
	"encoding/json"
	"testing"
)

func TestGetRandomUserAgent(t *testing.T) {
	ua := GetRandomUserAgent()
	if ua == "" {
		t.Fatal("GetRandomUserAgent returned empty string")
	}
	// Should contain a browser identifier
	if len(ua) < 20 {
		t.Errorf("user agent seems too short: %q", ua)
	}
}

func TestGetRandomUserAgent_Varies(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seen[GetRandomUserAgent()] = true
	}
	if len(seen) < 2 {
		t.Error("GetRandomUserAgent returned the same value 100 times; expected variation")
	}
}

func TestToJson(t *testing.T) {
	input := map[string]any{"name": "test", "value": 42.0}
	result, err := ToJson(input)
	if err != nil {
		t.Fatalf("ToJson failed: %v", err)
	}

	// Should be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("ToJson output is not valid JSON: %v", err)
	}
	if parsed["name"] != "test" {
		t.Errorf("expected name=test, got %v", parsed["name"])
	}
	if parsed["value"] != 42.0 {
		t.Errorf("expected value=42, got %v", parsed["value"])
	}
}

func TestToJson_EmptyMap(t *testing.T) {
	result, err := ToJson(map[string]any{})
	if err != nil {
		t.Fatalf("ToJson failed on empty map: %v", err)
	}
	if result != "{}" {
		t.Errorf("expected {}, got %q", result)
	}
}

func TestFromJson(t *testing.T) {
	input := `{"name":"test","value":42}`
	result, err := FromJson(input)
	if err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result["name"])
	}
	if result["value"] != 42.0 {
		t.Errorf("expected value=42, got %v", result["value"])
	}
}

func TestFromJson_Invalid(t *testing.T) {
	_, err := FromJson("not json")
	if err == nil {
		t.Error("FromJson should fail on invalid JSON")
	}
}

func TestFromJson_EmptyObject(t *testing.T) {
	result, err := FromJson("{}")
	if err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestToJson_FromJson_Roundtrip(t *testing.T) {
	original := map[string]any{
		"key1": "value1",
		"key2": 123.0,
		"key3": true,
	}
	jsonStr, err := ToJson(original)
	if err != nil {
		t.Fatalf("ToJson failed: %v", err)
	}
	result, err := FromJson(jsonStr)
	if err != nil {
		t.Fatalf("FromJson failed: %v", err)
	}
	for k, v := range original {
		if result[k] != v {
			t.Errorf("roundtrip mismatch for %q: got %v, want %v", k, result[k], v)
		}
	}
}
