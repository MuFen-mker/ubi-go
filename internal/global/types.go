package global

import (
	"alexanderthegreat96/ubi-go/internal/auth"
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"log"
	"net/http"
)

type GlobalService struct {
	auth   *auth.Auth
	logger *log.Logger
	cache  *cache.Cache
	client *http.Client
	config *config.Config
}
