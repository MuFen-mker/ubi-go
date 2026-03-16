package main

import (
	"alexanderthegreat96/ubi-go/api"
	"alexanderthegreat96/ubi-go/internal/auth"
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/constants"
	"alexanderthegreat96/ubi-go/internal/global"
	"alexanderthegreat96/ubi-go/internal/profile"
	"alexanderthegreat96/ubi-go/internal/report"
	"alexanderthegreat96/ubi-go/internal/stats"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexanderthegreat96/envparser/v3"
)

var (
	logger *log.Logger
	cfg    *config.Config
	env    *envparser.EnvData
)

func init() {
	env = envparser.NewEnvParser(
		envparser.WithFilename(constants.ENV_FILE),
		envparser.WithRootPath(true),
	)

	if env.EnvError != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to read .env file: %v\n", env.EnvError)
	}

	cfg = config.AppConfig(env)
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", cfg.GetAppName()), log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	ctx := context.Background()
	cache, err := cache.NewCache(env, logger)
	if err != nil {
		logger.Printf("Failed to initialize cache: %v", err)
	}

	authService := auth.NewAuth(logger, cache, cfg)

	err = authService.EnsureSession(ctx)
	if err != nil {
		logger.Fatalf("Login failed: %s", err.Error())
		return
	}

	globalService := global.NewService(logger, authService, cache, cfg)
	profileService := profile.NewService(logger, cache, cfg, globalService)
	statsService := stats.NewService(logger, cache, globalService, cfg)
	reportService := report.NewService(logger, cfg, globalService)
	server := api.NewServer(logger, statsService, reportService, profileService, authService, cfg.GetApiPort())

	// ticket refresher for high availability
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			err := authService.EnsureValidSession(ctx)
			if err != nil {
				logger.Printf("[Session Refresher] Failed to refresh Ubisoft session: %v", err)
			} else {
				logger.Printf("[Session Refresher] Ubisoft session checked/refreshed successfully.")
			}
			<-ticker.C
		}
	}()

	logger.Printf("API server starting on port %d", cfg.GetApiPort())
	if err := server.Start(); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
