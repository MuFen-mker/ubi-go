package stats

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"alexanderthegreat96/ubi-go/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func NewService(logger *log.Logger, cache *cache.Cache, global *global.GlobalService, config *config.Config) *StatsService {
	return &StatsService{
		logger: logger,
		cache:  cache,
		global: global,
		config: config,
	}
}

func (s *StatsService) GetStats(spaceId string, identifiers []string) ([]map[string]any, error) {
	if len(identifiers) > 20 {
		return nil, fmt.Errorf("Max of 20 identifiers per request are allowed.")
	}

	url := fmt.Sprintf("%s/v1/profiles/stats?spaceId=%s&profileIds=%s",
		ubiServicesUrl, spaceId, strings.Join(identifiers, ","))

	ctx := context.Background()
	res, err := s.global.MakeRequest(ctx, "GET", url, nil, nil)
	if err != nil {
		return nil, err
	}

	profilesRaw, ok := res["profiles"]
	if !ok {
		return nil, fmt.Errorf("no profile data found")
	}

	profiles, ok := profilesRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("profiles is not a list")
	}

	output := []map[string]any{}

	for _, itemRaw := range profiles {
		item, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}

		profileId, profileOk := item["profileId"].(string)
		stats, statsOk := item["stats"]
		if !profileOk || !statsOk {
			continue
		}

		result := map[string]any{"profileId": profileId}
		result["cached"] = false

		cacheKey := fmt.Sprintf("%s:%s:%s", ubisoftStatsCacheKey, profileId, spaceId)
		if s.cache != nil {
			cachedData, err := s.cache.Get(ctx, cacheKey)
			if err != nil {
				fmt.Println(err)
			}

			if cachedData.Data != "" {
				cachedDataJson, err := utils.FromJson(cachedData.Data)
				if err != nil {
					result["stats"] = stats
				} else {
					result["stats"] = cachedDataJson
					result["cached"] = true
				}

			} else {
				statsMap, ok := stats.(map[string]any)
				if !ok {
					continue
				}

				result["stats"] = statsMap
				statsJson, err := utils.ToJson(statsMap)
				if err == nil {
					s.cache.Put(ctx, cacheKey, statsJson, s.config.GetCacheStatsDuration())
				}
			}
		} else {
			statsMap, ok := stats.(map[string]any)
			if !ok {
				continue
			}

			result["stats"] = statsMap
			statsJson, err := utils.ToJson(statsMap)
			if err == nil && s.cache != nil {
				s.cache.Put(ctx, cacheKey, statsJson, s.config.GetCacheStatsDuration())
			}
		}

		output = append(output, result)
	}

	return output, nil
}

func (s *StatsService) GetStatsCard(spaceId, identifier string) (map[string]any, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("ubi-stats-card:%s:%s", identifier, spaceId)
	if s.cache != nil {
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData.Data != "" {
			var cachedResult map[string]any
			if err := json.Unmarshal([]byte(cachedData.Data), &cachedResult); err == nil {
				return cachedResult, nil
			}
		}
	}

	url := fmt.Sprintf("https://public-ubiservices.ubi.com/v1/profiles/%s/statscard?spaceId=%s", identifier, spaceId)

	resp, err := s.global.MakeRequest(ctx, "GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	cards, ok := resp["Statscards"].([]any)
	if !ok || len(cards) == 0 {
		return nil, fmt.Errorf("No statscard data found for this UID")
	}
	result := make(map[string]any)
	for _, item := range cards {
		if m, ok := item.(map[string]any); ok {
			if stat, ok := m["statName"].(string); ok {
				result[stat] = m["value"]
			}
		}
	}
	if s.cache != nil {
		if resultJson, err := json.Marshal(result); err == nil {
			s.cache.Put(ctx, cacheKey, string(resultJson), s.config.GetCacheStatsCardDuration())
		}
	}
	return result, nil
}
