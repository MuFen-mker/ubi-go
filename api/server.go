package api

import (
	"alexanderthegreat96/ubi-go/internal/auth"
	"alexanderthegreat96/ubi-go/internal/profile"
	"alexanderthegreat96/ubi-go/internal/report"
	"alexanderthegreat96/ubi-go/internal/stats"
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	mux      *http.ServeMux
	logger   *log.Logger
	handlers *Handlers
	port     int
}

func NewServer(logger *log.Logger, statsService *stats.StatsService, reportService *report.ReportService, profileService *profile.ProfileService, authService *auth.Auth, port int) *Server {
	mux := http.NewServeMux()
	handlers := NewHandlers(logger, statsService, reportService, profileService, authService)

	s := &Server{
		mux:      mux,
		logger:   logger,
		handlers: handlers,
		port:     port,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/report", s.handlers.ReportHandler)
	s.mux.HandleFunc("/health", s.handlers.HealthCheck)
	s.mux.HandleFunc("/profile", s.handlers.GetProfile)
	s.mux.HandleFunc("/stats", s.handlers.GetStats)
	s.mux.HandleFunc("/statscard", s.handlers.GetStatsCard)
	s.mux.HandleFunc("/games", s.handlers.GetGames)
	s.mux.HandleFunc("/username/check", s.handlers.CheckUsernameAvailability)

	s.logger.Println("Routes configured")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Printf("Starting API server on %s", addr)

	var handler http.Handler = s.mux
	handler = CORSMiddleware(handler)
	handler = LoggingMiddleware(s.logger)(handler)

	return http.ListenAndServe(addr, handler)
}
