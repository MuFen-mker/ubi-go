package stats

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"log"
)

type StatsService struct {
	logger *log.Logger
	cache  *cache.Cache
	global *global.GlobalService
	config *config.Config
}



type PlayerIdentifier struct {
	UserId       string
	PlatformType string
}

type SpaceIdMapping struct {
	Platform string
	SpaceId  string
}

type Game struct {
	Platform string
	SpaceId  string
}

type Stats struct {
}
