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

// HandleReport assembles a Ubisoft player-report case from the supplied
// payload, submits it to Ubisoft, and returns the parsed response or an error.
func (s *ReportService) HandleReport(ctx context.Context, payload ReportPayload) (map[string]any, error) {
	if payload.Platform == "" || payload.ProductInstallmentID == "" || payload.ReportedUsername == "" || payload.Description == "" {
		return nil, fmt.Errorf("missing required fields: platform, productInstallmentId, reportedUsername, description")
	}

	platformID, ok := platformIDMapping[payload.Platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %q (allowed: uplay, psn, xbl)", payload.Platform)
	}

	url := fmt.Sprintf("%s/%s", ubiServicesUrl, reportUrl)

	body := map[string]any{
		"case": map[string]any{
			"ubiCategoryId":        ubiCategoryId,
			"requestType":          requestType,
			"platformId":           platformID,
			"productInstallmentId": payload.ProductInstallmentID,
			"locale":               caseLocale,
			"contactChannel":       contactChannel,
			"origin":               caseOrigin,
			"description":          payload.Description,
			"reportedUsername":     payload.ReportedUsername,
			"reportedVideoLink":    payload.ReportedVideoLink,
		},
		"attachments": []any{},
		"token":       "",
	}

	// The CSHelp player-report endpoint is a Ubisoft website surface, so it
	// expects the website app id (and genome id) rather than the global one.
	// These override the defaults set by global.doRequest.
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

	data, err := s.global.MakeRequest(ctx, "POST", url, body, headers)
	if err != nil {
		return nil, err
	}

	if errMsg, ok := data["error"]; ok {
		return data, fmt.Errorf("ubisoft error: %v", errMsg)
	}
	if code, ok := data["errorCode"]; ok {
		return map[string]any{"error": data["message"]}, fmt.Errorf("ubisoft errorCode: %v", code)
	}

	return data, nil
}
