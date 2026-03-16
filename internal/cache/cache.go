package cache

import (
	"alexanderthegreat96/ubi-go/internal/config"
	"alexanderthegreat96/ubi-go/internal/constants"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alexanderthegreat96/envparser/v3"
	"github.com/redis/go-redis/v9"
)

/*
Package cache provides a wrapper around go-redis/v9 with convenient methods for:

  - Adding, retrieving, and deleting elements from Redis.
  - Managing FIFO queues with optional uniqueness constraints.

Simple cache usage:

	ctx := context.Background()

	cache, err := NewCache()
	if err != nil {
	    log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Store an item for 60 seconds
	if err, _ := cache.Put(ctx, "greeting", "Hello, world!", 60); err != nil {
	    log.Fatalf("Failed to put item: %v", err)
	}

	// Retrieve an item
	value, err := cache.Get(ctx, "greeting")
	if err != nil {
	    log.Fatalf("Failed to get item: %v", err)
	}
	log.Printf("Cached value: %s", value)

	// Check if it exists
	exists, _ := cache.Exists(ctx, "greeting")
	log.Printf("Key exists? %v", exists)

	// Delete the item
	if err := cache.Delete(ctx, "greeting"); err != nil {
	    log.Fatalf("Failed to delete item: %v", err)
	}

Queue processing usage:

	// Enqueue some work (unique items)
	_, _ = cache.EnqueueItemUnique(ctx, "emailQueue", "user1@example.com", "user1")
	_, _ = cache.EnqueueItemUnique(ctx, "emailQueue", "user2@example.com", "user2")

	// Define how to process each queue item
	processEmail := func(email string) error {
	    log.Printf("Sending email to %s", email)
	    time.Sleep(500 * time.Millisecond) // Simulate work
	    return nil
	}

	// Process the queue until empty or context is canceled
	err = cache.ProcessQueue(
	    ctx,
	    "emailQueue", // queue name
	    processEmail, // processFunc
	    true,         // isUnique: clear unique set when done
	    5,            // timeout in seconds for BRPop
	)

	if err != nil {
	    log.Fatalf("Queue processing stopped: %v", err)
	}
*/

// ErrCacheMiss is returned when a cache key is not found in Redis.
var ErrCacheMiss = redis.Nil

func NewCache(env *envparser.EnvData, logger *log.Logger) (*Cache, error) {
	cache := &Cache{
		logger: logger,
		config: config.AppConfig(env),
	}

	client, err := cache.connect(constants.DEFAULT_REDIS_TIMEOUT * time.Second)
	if err != nil {
		logger.Printf("Failed to initialize Redis: %v", err)
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}
	cache.client = client
	return cache, nil
}

func (c *Cache) Put(ctx context.Context, key string, value string, ttl int) (CacheResult, error) {
	if key == "" {
		return CacheResult{}, fmt.Errorf("cache key cannot be empty")
	}
	return c.SetData(ctx, key, value, ttl)
}

func (c *Cache) Get(ctx context.Context, key string) (CacheResult, error) {
	if key == "" {
		return CacheResult{}, fmt.Errorf("no cache key provided")
	}

	fullKey := c.fullKey(key)
	fn := func(ctx context.Context) (CacheResult, error) {
		data, err := c.client.Get(ctx, fullKey).Result()
		if err == ErrCacheMiss {
			return CacheResult{Key: fullKey}, ErrCacheMiss
		}
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to get data: %w", err)
		}
		return CacheResult{Key: fullKey, Data: data}, nil
	}

	result, err := fn(ctx)
	if err == ErrCacheMiss {
		return result, err // cache miss, do not retry
	}
	result, err = c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to get data: %w", err)
	}
	return result, nil
}

func (c *Cache) Delete(ctx context.Context, key string) (CacheResult, error) {
	if key == "" {
		return CacheResult{}, fmt.Errorf("cache key cannot be empty")
	}
	return c.ResetData(ctx, key)
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("cache key cannot be empty")
	}
	fullKey := c.fullKey(key)
	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *Cache) SetData(ctx context.Context, key, data string, expirationInSeconds int) (CacheResult, error) {
	if key == "" {
		return CacheResult{}, fmt.Errorf("no cache key provided")
	}

	fullKey := c.fullKey(key)
	fn := func(ctx context.Context) (CacheResult, error) {
		err := c.client.Set(ctx, fullKey, data, time.Duration(expirationInSeconds)*time.Second).Err()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to set data: %w", err)
		}
		return CacheResult{
			Key:                 fullKey,
			Data:                data,
			ExpirationInSeconds: expirationInSeconds,
		}, nil
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to set data: %w", err)
	}
	return result, nil
}

func (c *Cache) ResetData(ctx context.Context, key string) (CacheResult, error) {
	if key == "" {
		return CacheResult{}, fmt.Errorf("no cache key provided")
	}

	fullKey := c.fullKey(key)
	result, err := c.Get(ctx, key)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to check existing data: %w", err)
	}
	if result.Data == "" && result.Message != "" {
		return CacheResult{}, fmt.Errorf("no data to reset: %s", result.Message)
	}

	fn := func(ctx context.Context) (CacheResult, error) {
		err := c.client.Del(ctx, fullKey).Err()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to delete data: %w", err)
		}
		return CacheResult{
			Key:     fullKey,
			Message: "Data deleted successfully",
		}, nil
	}

	result, err = c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to reset data: %w", err)
	}
	return result, nil
}

func (c *Cache) EnqueueItem(ctx context.Context, queueName, item string) (CacheResult, error) {
	if queueName == "" || item == "" {
		return CacheResult{}, fmt.Errorf("queue name or item cannot be empty")
	}

	fullQueueName := c.fullKey(queueName)
	fn := func(ctx context.Context) (CacheResult, error) {
		err := c.client.LPush(ctx, fullQueueName, item).Err()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to enqueue item: %w", err)
		}
		return CacheResult{
			Queue: fullQueueName,
			Item:  item,
		}, nil
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to enqueue item: %w", err)
	}
	return result, nil
}

func (c *Cache) EnqueueItemUnique(ctx context.Context, queueName, item, itemID string) (CacheResult, error) {
	if queueName == "" || item == "" || itemID == "" {
		return CacheResult{}, fmt.Errorf("queue name, item, or itemID cannot be empty")
	}

	fullQueueName := c.fullKey(queueName)
	uniqueSetName := c.fullKey(queueName + "_set")
	fn := func(ctx context.Context) (CacheResult, error) {
		isMember, err := c.client.SIsMember(ctx, uniqueSetName, itemID).Result()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to check unique set: %w", err)
		}
		if isMember {
			return CacheResult{}, fmt.Errorf("duplicate item not added: %s", itemID)
		}
		err = c.client.LPush(ctx, fullQueueName, item).Err()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to enqueue item: %w", err)
		}
		err = c.client.SAdd(ctx, uniqueSetName, itemID).Err()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to add to unique set: %w", err)
		}
		return CacheResult{
			Queue: fullQueueName,
			Item:  item,
		}, nil
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to enqueue unique item: %w", err)
	}
	return result, nil
}

func (c *Cache) DequeueItem(ctx context.Context, queueName string, timeout int) (CacheResult, error) {
	if queueName == "" {
		return CacheResult{}, fmt.Errorf("queue name cannot be empty")
	}

	fullQueueName := c.fullKey(queueName)
	fn := func(ctx context.Context) (CacheResult, error) {
		item, err := c.client.BRPop(ctx, time.Duration(timeout)*time.Second, fullQueueName).Result()
		if err == redis.Nil {
			return CacheResult{
				Queue:   fullQueueName,
				Message: "Queue is empty or timeout reached",
			}, fmt.Errorf("queue empty or timeout: %s", fullQueueName)
		}
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to dequeue item: %w", err)
		}
		return CacheResult{
			Queue: fullQueueName,
			Item:  item[1], // BRPop returns [key, value]
		}, nil
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to dequeue item: %w", err)
	}
	return result, nil
}

func (c *Cache) ProcessQueue(
	ctx context.Context,
	queueName string,
	processFunc func(string) error,
	isUnique bool,
	timeout int,
) error {
	if queueName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	fullQueueName := c.fullKey(queueName)
	uniqueSetName := c.fullKey(queueName + "_set")

	defer func() {
		if err := c.clearUniqueIfNeeded(ctx, isUnique, uniqueSetName); err != nil {
			c.logger.Printf("Error during unique set cleanup: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			item, err := c.DequeueItem(ctx, queueName, timeout)
			if err != nil {
				c.logger.Printf("Queue %s is empty or error occurred: %v", fullQueueName, err)
				return nil // cleanup happens via defer
			}
			if item.Item == "" {
				return nil // cleanup happens via defer
			}

			if err := func() error {
				defer func() {
					if r := recover(); r != nil {
						c.logger.Printf("Panic recovered while processing item %s: %v", item.Item, r)
					}
				}()
				return processFunc(item.Item)
			}(); err != nil {
				c.logger.Printf("Error processing item %s: %v", item.Item, err)
			} else {
				c.logger.Printf("Processed item: %s", item.Item)
			}
		}
	}
}

func (c *Cache) ClearCache(ctx context.Context, prefix string) (CacheResult, error) {
	if prefix == "" {
		prefix = c.config.GetRedisGlobalCacheKey()
	}
	if !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}

	fn := func(ctx context.Context) (CacheResult, error) {
		var cursor uint64
		deletedKeysCount := 0

		for {
			keys, nextCursor, err := c.client.Scan(ctx, cursor, prefix+"*", 0).Result()
			if err != nil {
				return CacheResult{}, fmt.Errorf("failed to scan keys: %w", err)
			}
			if len(keys) > 0 {
				err = c.client.Del(ctx, keys...).Err()
				if err != nil {
					return CacheResult{}, fmt.Errorf("failed to delete keys: %w", err)
				}
				deletedKeysCount += len(keys)
			}
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		return CacheResult{
			Message: fmt.Sprintf("Cleared %d keys with prefix '%s'", deletedKeysCount, prefix),
		}, nil
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		return CacheResult{}, fmt.Errorf("failed to clear cache: %w", err)
	}
	return result, nil
}

func (c *Cache) connect(timeout time.Duration) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%d", c.config.GetRedisHost(), c.config.GetRedisPort()),
		Password:    c.config.GetRedisPass(),
		DB:          0,
		DialTimeout: timeout,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		c.logger.Printf("Failed to connect to Redis: %v", err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return client, nil
}

func (c *Cache) reconnect(ctx context.Context, retries int, delay time.Duration) (*redis.Client, error) {
	for attempt := 1; attempt <= retries; attempt++ {
		c.logger.Printf("Attempting to reconnect to Redis (attempt %d/%d)...", attempt, retries)
		client, err := c.connect(1 * time.Second)
		if err == nil {
			return client, nil
		}
		c.logger.Printf("Reconnect attempt %d failed: %v", attempt, err)
		if attempt < retries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	err := fmt.Errorf("failed to reconnect to Redis after %d attempts", retries)
	c.logger.Printf("%v", err)
	return nil, err
}

func (c *Cache) executeWithReconnect(ctx context.Context, fn func(context.Context) (CacheResult, error), retries int) (CacheResult, error) {
	for attempt := 1; attempt <= retries; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		c.logger.Printf("Redis operation failed: %v", err)
		if attempt < retries {
			c.logger.Printf("Attempting to reconnect and retry operation (attempt %d/%d)...", attempt, retries)
			client, reconErr := c.reconnect(ctx, retries, 2*time.Second)
			if reconErr != nil {
				return CacheResult{}, fmt.Errorf("reconnect failed: %w", reconErr)
			}
			c.client = client
		} else {
			return CacheResult{}, fmt.Errorf("max retries reached: %w", err)
		}
	}
	return CacheResult{}, fmt.Errorf("unexpected error in executeWithReconnect")
}

func (c *Cache) fullKey(key string) string {
	return fmt.Sprintf("%s:%s", c.config.GetRedisGlobalCacheKey(), key)
}

// use for clearing elements if they are unique - applies to queues
func (c *Cache) clearUniqueIfNeeded(ctx context.Context, isUnique bool, uniqueSetName string) error {
	if !isUnique {
		return nil
	}

	fn := func(ctx context.Context) (CacheResult, error) {
		typ, err := c.client.Type(ctx, uniqueSetName).Result()
		if err != nil {
			return CacheResult{}, fmt.Errorf("failed to check set type: %w", err)
		}
		if typ == "set" {
			err = c.client.Del(ctx, uniqueSetName).Err()
			if err != nil {
				return CacheResult{}, fmt.Errorf("failed to clear unique set: %w", err)
			}
			return CacheResult{
				Message: fmt.Sprintf("Unique set '%s' cleared", uniqueSetName),
			}, nil
		}
		return CacheResult{}, fmt.Errorf("set not found: %s", uniqueSetName)
	}

	result, err := c.executeWithReconnect(ctx, fn, constants.DEFAULT_REDIS_RETRIES)
	if err != nil {
		c.logger.Printf("Failed to clear unique set '%s': %v", uniqueSetName, err)
		return fmt.Errorf("failed to clear unique set: %w", err)
	}
	if result.Message != "" {
		c.logger.Printf("%s", result.Message)
	}
	return nil
}
