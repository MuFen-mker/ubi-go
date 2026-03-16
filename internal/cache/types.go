package cache

import (
	"alexanderthegreat96/ubi-go/internal/config"
	"log"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	logger *log.Logger
	client *redis.Client
	config *config.Config
}

type CacheResult struct {
	Key                 string
	Queue               string
	Item                string
	Data                string
	ExpirationInSeconds int
	Message             string
}

type CacheErrorResult struct {
	Error string
}
