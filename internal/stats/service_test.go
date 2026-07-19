package stats

import (
	"encoding/json"
	"log"
	"os"
	"testing"
)

func TestGetStats_TooManyIdentifiers(t *testing.T) {
	s := &StatsService{
		logger: log.New(os.Stderr, "[test] ", 0),
	}
	ids := make([]string, 21)
	for i := range ids {
		ids[i] = "id"
	}
	_, err := s.GetStats("space-id", ids)
	if err == nil {
		t.Error("expected error when more than 20 identifiers")
	}
}

func TestGetStats_TwentyOneIdentifiers(t *testing.T) {
	s := &StatsService{
		logger: log.New(os.Stderr, "[test] ", 0),
	}
	ids := make([]string, 21)
	for i := range ids {
		ids[i] = "id"
	}
	_, err := s.GetStats("space-id", ids)
	if err == nil || err.Error() != "Max of 20 identifiers per request are allowed." {
		t.Error("21 identifiers should be rejected with max-identifiers error")
	}
}

func TestPlayerIdentifier_Fields(t *testing.T) {
	p := PlayerIdentifier{UserId: "uid-1", PlatformType: "uplay"}
	if p.UserId != "uid-1" {
		t.Errorf("expected UserId=uid-1, got %q", p.UserId)
	}
	if p.PlatformType != "uplay" {
		t.Errorf("expected PlatformType=uplay, got %q", p.PlatformType)
	}
}

func TestSpaceIdMapping_Fields(t *testing.T) {
	m := SpaceIdMapping{Platform: "pc", SpaceId: "abc-123"}
	if m.Platform != "pc" || m.SpaceId != "abc-123" {
		t.Errorf("unexpected values: %+v", m)
	}
}

func TestGame_Fields(t *testing.T) {
	g := Game{Platform: "uplay", SpaceId: "space-xyz"}
	if g.Platform != "uplay" || g.SpaceId != "space-xyz" {
		t.Errorf("unexpected values: %+v", g)
	}
}

func TestGame_JSONRoundtrip(t *testing.T) {
	original := Game{Platform: "pc", SpaceId: "abc"}
	data, _ := json.Marshal(original)
	var restored Game
	json.Unmarshal(data, &restored)
	if restored != original {
		t.Errorf("roundtrip mismatch: %+v vs %+v", restored, original)
	}
}
