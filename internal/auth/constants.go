package auth

const (
	loginPlatFormUrl       = "https://public-ubiservices.ubi.com/v3/profiles/sessions"
	defaultPlatformType    = "uplay"
	defaultUserAgent       = "Mozilla/5.0 (compatible; Go-Client/1.0)"
	configPath             = "config/ubisoft.json"
	userCachePath          = "caching/ubisoft-user.json"
	dataDomeCookiePath     = "caching/datadome.json"
	ubisoftAppId           = "f35adcb5-1911-440c-b1c9-48fdc1701c68"
	ubisoftAccountCacheKey = "ubi-account"
	dataDomeCookieCacheKey = "datadome-cookie"
	sessionTTL             = 300
	dataDomeCookieTTL      = 1800 // 30 minutes
)
