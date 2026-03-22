package profile

import (
	"alexanderthegreat96/ubi-go/internal/cache"
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/global"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

func NewService(logger *log.Logger, cache *cache.Cache, cfg *config.Config, global *global.GlobalService) *ProfileService {
	return &ProfileService{
		logger: logger,
		cache:  cache,
		config: cfg,
		global: global,
	}
}

func (s *ProfileService) GetProfileByUid(identifier string) (UserProfile, error) {
	s.logger.Printf("Attempting to retrieve username by UUID for: %s", identifier)
	ctx := context.Background()
	cacheKey := fmt.Sprintf("profile:uid:%s", identifier)
	if s.cache != nil {
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData.Data != "" {
			var cachedProfile UserProfile
			if err := json.Unmarshal([]byte(cachedData.Data), &cachedProfile); err == nil {
				return cachedProfile, nil
			}
		}
	}

	response, err := s.global.MakeRequest(ctx, "GET", fmt.Sprintf("%s/%s/%s", ubiServicesUrl, ubiServicesProfileUrlV1, identifier), nil, nil)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to retrieve profile: %w", err)
	}
	// Robust checks for response
	if len(response) == 0 {
		return UserProfile{}, fmt.Errorf("no profile found for uid: %s", identifier)
	}
	// Defensive: check for required fields
	profile := UserProfile{
		IdOnPlatform:   fmt.Sprint(response["idOnPlatform"]),
		NameOnPlatform: fmt.Sprint(response["nameOnPlatform"]),
		PlatformType:   fmt.Sprint(response["platformType"]),
		ProfileId:      fmt.Sprint(response["profileId"]),
		UserId:         fmt.Sprint(response["userId"]),
	}
	// Cache the result
	if s.cache != nil {
		if profileJson, err := json.Marshal(profile); err == nil {
			s.cache.Put(ctx, cacheKey, string(profileJson), s.config.GetCacheProfileDuration())
		}
	}
	return profile, nil
}

func (s *ProfileService) GetProfileByUsername(username, platform string) (UserProfile, error) {
	s.logger.Printf("Searching for Division2 Profile Data: %s (%s)", username, platform)
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s:username:%s:platform:%s", ubisoftProfilesCacheKey, username, platform)
	if s.cache != nil {
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData.Data != "" {
			var cachedProfile UserProfile
			if err := json.Unmarshal([]byte(cachedData.Data), &cachedProfile); err == nil {
				return cachedProfile, nil
			}
		}
	}
	params := url.Values{}
	params.Set("nameOnPlatform", username)
	params.Set("platformType", platform)
	url := fmt.Sprintf("https://public-ubiservices.ubi.com/v2/profiles/?%s", params.Encode())
	response, err := s.global.MakeRequest(ctx, "GET", url, nil, nil)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to retrieve profile: %w", err)
	}
	profiles, ok := response["profiles"].([]any)
	if !ok || len(profiles) == 0 {
		return UserProfile{}, fmt.Errorf("no profile found, retry with a different name")
	}
	profile, ok := profiles[0].(map[string]any)
	if !ok {
		return UserProfile{}, fmt.Errorf("unexpected profile format")
	}
	result := UserProfile{
		IdOnPlatform:   fmt.Sprint(profile["idOnPlatform"]),
		NameOnPlatform: fmt.Sprint(profile["nameOnPlatform"]),
		PlatformType:   fmt.Sprint(profile["platformType"]),
		ProfileId:      fmt.Sprint(profile["profileId"]),
		UserId:         fmt.Sprint(profile["userId"]),
	}
	// Cache the result
	if s.cache != nil {
		if profileJson, err := json.Marshal(result); err == nil {
			s.cache.Put(ctx, cacheKey, string(profileJson), s.config.GetCacheProfileDuration())
		}
	}
	return result, nil
}
