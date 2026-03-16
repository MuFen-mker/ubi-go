package api

import (
	"alexanderthegreat96/ubi-go/internal/auth"
	"alexanderthegreat96/ubi-go/internal/profile"
	"alexanderthegreat96/ubi-go/internal/report"
	"alexanderthegreat96/ubi-go/internal/stats"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Handlers struct {
	reportService  *report.ReportService
	logger         *log.Logger
	statsService   *stats.StatsService
	profileService *profile.ProfileService
	authService    *auth.Auth
}

func NewHandlers(logger *log.Logger, statsService *stats.StatsService, reportService *report.ReportService, profileService *profile.ProfileService, authService *auth.Auth) *Handlers {
	return &Handlers{
		logger:         logger,
		statsService:   statsService,
		reportService:  reportService,
		profileService: profileService,
		authService:    authService,
	}
}

func (h *Handlers) ReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed; use POST")
		return
	}
	var payload report.ReportPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Printf("/report: invalid JSON: %v", err)
		WriteError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	result, err := h.reportService.HandleReport(context.Background(), payload)
	if err != nil {
		h.logger.Printf("/report: failed to process report: %v", err)
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, result, "Report sent successfully")
}

func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, http.StatusOK, map[string]string{"status": "healthy"}, "Server is running")
}

func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	username := r.URL.Query().Get("username")
	platform := r.URL.Query().Get("platform")
	if uid != "" {
		profile, err := h.profileService.GetProfileByUid(uid)
		if err != nil {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("Failed to retrieve profile: %v", err))
			return
		}
		WriteSuccess(w, http.StatusOK, profile, "Profile retrieved successfully")
		return
	}

	if username != "" && platform != "" {
		profile, err := h.profileService.GetProfileByUsername(username, platform)
		if err != nil {
			WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		WriteSuccess(w, http.StatusOK, profile, "Profile retrieved successfully")
		return
	}

	WriteError(w, http.StatusBadRequest, "Missing required parameters: uid or (username and platform)")
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	gameId := r.URL.Query().Get("gameId")
	platform := r.URL.Query().Get("platform")
	uidsParam := r.URL.Query().Get("uids")

	if gameId == "" || platform == "" || uidsParam == "" {
		WriteError(w, http.StatusBadRequest, "Missing required parameters: gameId, platform, uids")
		return
	}

	uids := strings.Split(uidsParam, ",")
	for i, uid := range uids {
		uids[i] = strings.TrimSpace(uid)
	}

	if len(uids) > 20 {
		h.handleStatsBatch(w, gameId, platform, uids)
		return
	}

	statsData, err := h.statsService.GetStats(gameId, uids)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Failed to retrieve stats: %v", err))
		return
	}

	WriteSuccess(w, http.StatusOK, statsData, "Stats retrieved successfully")
}

func (h *Handlers) handleStatsBatch(w http.ResponseWriter, gameId string, platform string, uids []string) {
	const batchSize = 20
	var allStats []map[string]any

	for i := 0; i < len(uids); i += batchSize {
		end := min(i+batchSize, len(uids))

		batch := uids[i:end]
		statsData, err := h.statsService.GetStats(gameId, batch)
		if err != nil {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("Failed to retrieve batch stats: %v", err))
			return
		}

		allStats = append(allStats, statsData...)
	}

	WriteSuccess(w, http.StatusOK, allStats, fmt.Sprintf("Stats retrieved successfully for %d players", len(uids)))
}

func (h *Handlers) GetStatsCard(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	spaceId := r.URL.Query().Get("spaceId")
	if uid == "" || spaceId == "" {
		WriteError(w, http.StatusBadRequest, "Missing required parameters: uid, spaceId")
		return
	}

	stats, err := h.statsService.GetStatsCard(spaceId, uid)
	if err != nil {
		WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, stats, "Statscard retrieved successfully")
}

func (h *Handlers) CheckUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Username availability check not yet implemented")
}

func (h *Handlers) GetGames(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Games endpoint not yet implemented")
}

func (h *Handlers) RootAPIInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
	  "endpoints": [
	    {
	      "path": "/health",
	      "method": "GET",
	      "description": "Health check endpoint",
	      "response": {"status": "ok"}
	    },
	    {
	      "path": "/profile",
	      "method": "GET",
	      "params": ["uid OR (username, platform)"],
	      "description": "Get user profile by UID or username+platform",
	      "response": {"success": true, "data": {"IdOnPlatform": "...", "NameOnPlatform": "...", "PlatformType": "...", "ProfileId": "...", "UserId": "..."}}
	    },
	    {
	      "path": "/stats",
	      "method": "GET",
	      "params": ["profileIds (comma-separated)", "spaceId"],
	      "description": "Get stats for one or more profiles",
	      "response": {"success": true, "data": [{"profileId": "...", "stats": {"stat1": 123, "stat2": 456}}]}
	    },
	    {
	      "path": "/statscard",
	      "method": "GET",
	      "params": ["uid", "spaceId"],
	      "description": "Get statscard for a profile",
	      "response": {"success": true, "data": {"statName": "value", "statName2": "value2"}}
	    }
	  ]
	}`))
}
