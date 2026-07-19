package cache

import (
	"encoding/json"
	"testing"
)

func TestCacheResult_Fields(t *testing.T) {
	r := CacheResult{
		Key:                 "test:key",
		Data:                `{"value":1}`,
		Queue:               "test:queue",
		Item:                "item-1",
		ExpirationInSeconds: 300,
		Message:             "ok",
	}
	if r.Key != "test:key" {
		t.Errorf("expected Key=test:key, got %q", r.Key)
	}
	if r.ExpirationInSeconds != 300 {
		t.Errorf("expected ExpirationInSeconds=300, got %d", r.ExpirationInSeconds)
	}
}

func TestCacheResult_JSONRoundtrip(t *testing.T) {
	original := CacheResult{
		Key:  "k",
		Data: "v",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var restored CacheResult
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestCacheErrorResult(t *testing.T) {
	e := CacheErrorResult{Error: "something failed"}
	if e.Error != "something failed" {
		t.Errorf("expected error message, got %q", e.Error)
	}
}

func TestErrCacheMiss(t *testing.T) {
	if ErrCacheMiss.Error() == "" {
		t.Error("ErrCacheMiss should have an error message")
	}
}
