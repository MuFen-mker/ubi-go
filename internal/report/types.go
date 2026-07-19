package report

import (
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"log"
)

type ReportService struct {
	logger *log.Logger
	config *config.Config
	global *global.GlobalService
}

// ReportPayload carries the semantic report data sent by the caller.
// The Ubisoft-specific case structure is assembled inside the service so
// that all Ubisoft API knowledge stays within this microservice.
type ReportPayload struct {
	Platform             string `json:"platform"`
	ProductInstallmentID string `json:"productInstallmentId"`
	ReportedID           string `json:"reportedId"`
	ReportedUsername     string `json:"reportedUsername"`
	Description          string `json:"description"`
	ReportedVideoLink    string `json:"reportedVideoLink"`
}
