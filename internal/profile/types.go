package profile

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"log"
)

type ProfileService struct {
	logger *log.Logger
	cache  *cache.Cache
	config *config.Config
	global *global.GlobalService
}

type UserProfile struct {
	IdOnPlatform   string `json:"IdOnPlatform"`
	NameOnPlatform string `json:"NameOnPlatform"`
	PlatformType   string `json:"PlatformType"`
	ProfileId      string `json:"ProfileId"`
	UserId         string `json:"UserId"`
}
