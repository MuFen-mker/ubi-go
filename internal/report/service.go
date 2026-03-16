package report

import (
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"context"
	"fmt"
	"log"
)

func NewService(logger *log.Logger, cfg *config.Config, global *global.GlobalService) *ReportService {
	return &ReportService{
		logger: logger,
		config: cfg,
		global: global,
	}
}

// HandleReport sends a player report to Ubisoft and returns the parsed response or error
func (s *ReportService) HandleReport(ctx context.Context, payload ReportPayload) (map[string]any, error) {
	if payload.ReporterID == "" || payload.ReportedID == "" || payload.Reason == "" {
		return nil, fmt.Errorf("missing required fields: reporterId, reportedId, reason")
	}

	url := fmt.Sprintf("%s/%s", ubiServicesUrl, reportUrl)

	headers := map[string]string{
		"Accept-Encoding":             "gzip, deflate, br",
		"Accept-Language":             "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7",
		"Access-Control-Allow-Origin": "*",
		"Accept":                      "application/json, text/plain, */*",
		"Dnt":                         "1",
		"Origin":                      "https://www.ubisoft.com",
		"Referer":                     "https://www.ubisoft.com/",
		"Ubi-AppId":                   "4391c956-8943-48eb-8859-07b0778f47b9",
		"Ubi-LocaleCode":              "en-US",
		"Ubi-Genomeid":                "1a6f2698-1350-416e-b8e8-29d77fb86437",
	}

	data, err := s.global.MakeRequest(ctx, "POST", url, payload, headers)
	if err != nil {
		return nil, err
	}

	if errMsg, ok := data["error"]; ok {
		return data, fmt.Errorf("ubisoft error: %v", errMsg)
	}
	if code, ok := data["errorCode"]; ok {
		return map[string]any{"error": data["message"]}, fmt.Errorf("ubisoft errorCode: %v", code)
	}

	return map[string]any{"data": data}, nil
}
