package constants

const (
	ENV_FILE                          = ".env"
	DEFAULT_APP_NAME                  = "Ubi-Go"
	IS_DEV_MODE_DEFAULT               = false
	DEFAULT_REDIS_TIMEOUT             = 1
	DEFAULT_REDIS_RETRIES             = 3
	DEFAULT_REDIS_HOST                = "localhost"
	DEFAULT_REDIS_PORT                = 6379
	DEFAULT_REDIS_PASS                = "test"
	DEFAULT_REDIS_GLOBAL_CACHE_KEY    = "ubi-go"
	DEFAULT_CACHE_PROFILE_DURATION    = 300
	DEFAULT_CACHE_STATS_DURATION      = 300
	DEFAULT_CACHE_STATS_CARD_DURATION = 300
	DEFAULT_API_PORT                  = 8080
	DEFAULT_CHROMIUM_COMMAND          = "/usr/bin/chromium"
)

const (
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorYellow    = "\033[33m"
	ColorGreen     = "\033[32m"
	ColorBlue      = "\033[34m"
	ColorOpenGreen = "\033[38;5;48m"
)
