package config

import (
	"github.com/alexanderthegreat96/envparser/v3"
)

/**
* Shared types used within the project
* Contains both internal and external types
**/

type Config struct {
	appName                string
	isDevMode              bool
	redisHost              string
	redisPort              int
	redisPass              string
	redisGlobalCacheKey    string
	ubisoftAccounts        []map[string]string
	proxies                []string
	env                    *envparser.EnvData
	cacheProfileDuration   int
	cacheStatsDuration     int
	cacheStatsCardDuration int
	apiPort                int
	chromiumCommand        string
}
