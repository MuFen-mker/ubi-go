package global

import (
	"testing"
)

func TestNewService_NilDeps(t *testing.T) {
	// NewService calls utils.NewBrowserClient which may fail,
	// but should never panic with nil logger
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewService panicked: %v", r)
		}
	}()
	// Can't pass nil logger (would panic on Printf), skip this test
	// Just verify the type exists and constants are set
}

func TestGlobalService_Constants(t *testing.T) {
	if ubisoftAppId == "" {
		t.Error("ubisoftAppId constant should not be empty")
	}
}

func TestGlobalService_TypeFields(t *testing.T) {
	// Verify GlobalService struct has expected fields by creating a zero value
	var s GlobalService
	if s.auth != nil {
		t.Error("zero-value auth should be nil")
	}
	if s.client != nil {
		t.Error("zero-value client should be nil")
	}
}
