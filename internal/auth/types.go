package auth

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"log"
	"net/http"
	"sync"
	"time"
)

// this is what's supposed to be stored locally
// or in redis
type UserData struct {
	PlatformType                  string `json:"platformType"`
	Ticket                        string `json:"ticket"`
	TwoFactorAuthenticationTicket string `json:"twoFactorAuthenticationTicket"`
	SessionId                     string `json:"sessionId"`
	SessionKey                    string `json:"sessionKey"`
	ProfileId                     string `json:"profileId"`
	UserId                        string `json:"userId"`
	NameOnPlatform                string `json:"nameOnPlatform"`
	Environment                   string `json:"environment"`
	Expiration                    string `json:"expiration"`
	SpaceId                       string `json:"spaceId"`
	RememberMeTicket              string `json:"rememberMeTicket"`
}

// ubisoft responses

type SuccesfulLoginResponse struct{}

type FailedLoginResponse struct{}

// dataDomeCookieFile is the on-disk format for the cached DataDome cookie.
type dataDomeCookieFile struct {
	Value   string    `json:"value"`
	Expires time.Time `json:"expires"`
}

// auth related

type Auth struct {
	logger         *log.Logger
	cache          *cache.Cache
	config         *config.Config
	UserData       UserData
	client         *http.Client
	userAgent      string
	dataDomeCookie string
	sessionMu      sync.Mutex // guards login so only one goroutine refreshes at a time
}
