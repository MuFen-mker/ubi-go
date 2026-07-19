package report

import (
	"context"
	"testing"
)

func TestHandleReport_MissingPlatform(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{
		ProductInstallmentID: "4969",
		ReportedUsername:     "cheater",
		Description:          "cheating",
	})
	if err == nil {
		t.Error("expected error when platform is missing")
	}
}

func TestHandleReport_MissingProductInstallmentID(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{
		Platform:         "uplay",
		ReportedUsername: "cheater",
		Description:      "cheating",
	})
	if err == nil {
		t.Error("expected error when productInstallmentId is missing")
	}
}

func TestHandleReport_MissingReportedUsername(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{
		Platform:             "uplay",
		ProductInstallmentID: "4969",
		Description:          "cheating",
	})
	if err == nil {
		t.Error("expected error when reportedUsername is missing")
	}
}

func TestHandleReport_MissingDescription(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{
		Platform:             "uplay",
		ProductInstallmentID: "4969",
		ReportedUsername:     "cheater",
	})
	if err == nil {
		t.Error("expected error when description is missing")
	}
}

func TestHandleReport_UnsupportedPlatform(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{
		Platform:             "switch",
		ProductInstallmentID: "4969",
		ReportedUsername:     "cheater",
		Description:          "cheating",
	})
	if err == nil {
		t.Error("expected error when platform is unsupported")
	}
}

func TestHandleReport_AllFieldsMissing(t *testing.T) {
	s := &ReportService{}
	_, err := s.HandleReport(context.Background(), ReportPayload{})
	if err == nil {
		t.Error("expected error when all fields are missing")
	}
}
