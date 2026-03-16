package config

import (
	"alexanderthegreat96/ubi-go/internal/constants"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alexanderthegreat96/envparser/v3"
)

func loadProxyFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var proxies []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			proxies = append(proxies, line)
		}
	}
	return proxies
}

func AppConfig(env *envparser.EnvData) *Config {
	cfg := &Config{
		env: env,
	}

	cfg.appName = cfg.GetStr("APP_NAME", constants.DEFAULT_APP_NAME)
	cfg.isDevMode = cfg.GetBool("IS_DEV_MODE", constants.IS_DEV_MODE_DEFAULT)
	cfg.redisHost = cfg.GetStr("REDIS_HOST", constants.DEFAULT_REDIS_HOST)
	cfg.redisPort = cfg.GetInt("REDIS_PORT", constants.DEFAULT_REDIS_PORT)
	cfg.redisPass = cfg.GetStr("REDIS_PASS", constants.DEFAULT_REDIS_PASS)
	cfg.redisGlobalCacheKey = cfg.GetStr("REDIS_GLOBAL_CACHE_KEY", constants.DEFAULT_REDIS_GLOBAL_CACHE_KEY)

	rawAccounts, _ := cfg.env.GetValue("UBISOFT_ACCOUNTS", "list", []any{})

	accounts := []map[string]string{}
	if rawList, ok := rawAccounts.([]any); ok {
		for _, item := range rawList {
			if m, ok := item.(map[string]any); ok {
				typedMap := map[string]string{}
				for k, v := range m {
					typedMap[k] = fmt.Sprintf("%v", v)
				}
				accounts = append(accounts, typedMap)
			}
		}
	}

	cfg.ubisoftAccounts = accounts

	cfg.proxies = loadProxyFile("proxy.txt")
	cfg.cacheProfileDuration = cfg.GetInt("CACHE_PROFILE_DURATION", constants.DEFAULT_CACHE_PROFILE_DURATION)
	cfg.cacheStatsDuration = cfg.GetInt("CACHE_STATS_DURATION", constants.DEFAULT_CACHE_STATS_DURATION)
	cfg.cacheStatsCardDuration = cfg.GetInt("CACHE_STATS_DURATION", constants.DEFAULT_CACHE_STATS_CARD_DURATION)
	cfg.apiPort = cfg.GetInt("API_PORT", constants.DEFAULT_API_PORT)
	cfg.chromiumCommand = cfg.GetStr("CHROMIUM_CMD", constants.DEFAULT_CHROMIUM_COMMAND)

	return cfg
}

func (c *Config) GetAppName() string {
	return c.appName
}

func (c *Config) IsDevMode() bool {
	return c.isDevMode
}

func (c *Config) GetRedisHost() string {
	return c.redisHost
}

func (c *Config) GetRedisPort() int {
	return c.redisPort
}

func (c *Config) GetRedisPass() string {
	return c.redisPass
}

func (c *Config) GetRedisGlobalCacheKey() string {
	return c.redisGlobalCacheKey
}

func (c *Config) GetUbisoftAccounts() []map[string]string {
	return c.ubisoftAccounts
}

func (c *Config) GetProxies() []string {
	return c.proxies
}

func (c *Config) GetCacheProfileDuration() int {
	return c.cacheProfileDuration
}

func (c *Config) GetCacheStatsDuration() int {
	return c.cacheStatsDuration
}

func (c *Config) GetCacheStatsCardDuration() int {
	return c.cacheStatsCardDuration
}

func (c *Config) GetApiPort() int {
	return c.apiPort
}

func (c *Config) GetChromiumCommand() string {
	return c.chromiumCommand
}

func (c *Config) GetEnv() *envparser.EnvData {
	return c.env
}

func (c *Config) GetBool(key string, defaultVal bool) bool {
	val, err := c.env.GetValue(key, "bool", defaultVal)
	if err != nil {
		return defaultVal
	}

	boolVal, ok := val.(bool)
	if !ok {
		return defaultVal
	}

	return boolVal
}

func (c *Config) GetInt(key string, defaultVal int) int {
	val, err := c.env.GetValue(key, "int", defaultVal)

	if err != nil {
		return defaultVal
	}

	intVal, ok := val.(int)
	if !ok {
		return defaultVal
	}

	return intVal
}

func (c *Config) GetStr(key, defaultVal string) string {
	val, err := c.env.GetValue(key, "string", defaultVal)
	if err != nil {
		return defaultVal
	}

	strVal, ok := val.(string)
	if !ok {
		return defaultVal
	}

	return strings.Trim(strVal, `"`)
}

func (c *Config) GetList(key string, defaultVal []any) []any {
	val, err := c.env.GetValue(key, "list", []any{})
	if err != nil {
		return defaultVal
	}

	listVal, ok := val.([]any)
	if !ok {
		return defaultVal
	}

	return listVal
}

type Account struct {
	Email    string
	Password string
}
