package auth

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"log"
	"net/http"
)

// this is what's supposed to be stored locally
// or in redis
type UserData struct {
	PlatformType   string `json:"platformType"`
	Ticket         string `json:"ticket"`
	SessionId      string `json:"sessionId"`
	ProfileId      string `json:"profileId"`
	UserId         string `json:"userId"`
	NameOnPlatform string `json:"nameOnPlatform"`
	Environment    string `json:"environment"`
	Expiration     string `json:"expiration"`
}

// ubisoft responses

type SuccesfulLoginResponse struct{}

type FailedLoginResponse struct{}

// auth related

type Auth struct {
	logger    *log.Logger
	cache     *cache.Cache
	config    *config.Config
	UserData  UserData
	client    *http.Client
	userAgent string
}
