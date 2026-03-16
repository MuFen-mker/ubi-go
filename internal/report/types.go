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

type ReportPayload struct {
	ReporterID  string `json:"reporterId"`
	ReportedID  string `json:"reportedId"`
	Reason      string `json:"reason"`
	Description string `json:"description"`
}
