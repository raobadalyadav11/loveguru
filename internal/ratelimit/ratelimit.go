package ratelimit

import (
	"context"
	"time"

	"loveguru/internal/cache"
)

type RateLimiter struct {
	cache    *cache.Cache
	requests map[string]*RequestCounter
}

type RequestCounter struct {
	Count     int
	ResetTime time.Time
}

type Config struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

func NewRateLimiter(cacheClient *cache.Cache) *RateLimiter {
	return &RateLimiter{
		cache:    cacheClient,
		requests: make(map[string]*RequestCounter),
	}
}

func (r *RateLimiter) Allow(key string, config Config) (bool, error) {
	ctx := context.Background()

	// Check minute limit
	minuteKey := "ratelimit:minute:" + key
	if err := r.checkLimit(ctx, minuteKey, config.RequestsPerMinute, time.Minute); err != nil {
		return false, err
	}

	// Check hour limit
	hourKey := "ratelimit:hour:" + key
	if err := r.checkLimit(ctx, hourKey, config.RequestsPerHour, time.Hour); err != nil {
		return false, err
	}

	// Check day limit
	dayKey := "ratelimit:day:" + key
	if err := r.checkLimit(ctx, dayKey, config.RequestsPerDay, 24*time.Hour); err != nil {
		return false, err
	}

	// Increment counters
	if err := r.increment(ctx, minuteKey, time.Minute); err != nil {
		return false, err
	}
	if err := r.increment(ctx, hourKey, time.Hour); err != nil {
		return false, err
	}
	if err := r.increment(ctx, dayKey, 24*time.Hour); err != nil {
		return false, err
	}

	return true, nil
}

func (r *RateLimiter) checkLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	if limit <= 0 {
		return nil // No limit set
	}

	var count int
	err := r.cache.Get(ctx, key, &count)
	if err == nil {
		if count >= limit {
			return ErrRateLimitExceeded
		}
	}

	return nil
}

func (r *RateLimiter) increment(ctx context.Context, key string, window time.Duration) error {
	count, err := r.cache.Increment(ctx, key)
	if err != nil {
		return err
	}

	// Set expiration on first increment
	if count == 1 {
		return r.cache.Expire(ctx, key, window)
	}

	return nil
}

func (r *RateLimiter) Reset(key string) error {
	ctx := context.Background()

	keys := []string{
		"ratelimit:minute:" + key,
		"ratelimit:hour:" + key,
		"ratelimit:day:" + key,
	}

	for _, k := range keys {
		if err := r.cache.Delete(ctx, k); err != nil {
			return err
		}
	}

	return nil
}

func (r *RateLimiter) GetRemaining(key string, config Config) (map[string]int, error) {
	ctx := context.Background()

	result := make(map[string]int)

	// Get remaining for each window
	minuteKey := "ratelimit:minute:" + key
	hourKey := "ratelimit:hour:" + key
	dayKey := "ratelimit:day:" + key

	var count int
	if err := r.cache.Get(ctx, minuteKey, &count); err == nil {
		result["minute"] = config.RequestsPerMinute - count
		if result["minute"] < 0 {
			result["minute"] = 0
		}
	} else {
		result["minute"] = config.RequestsPerMinute
	}

	if err := r.cache.Get(ctx, hourKey, &count); err == nil {
		result["hour"] = config.RequestsPerHour - count
		if result["hour"] < 0 {
			result["hour"] = 0
		}
	} else {
		result["hour"] = config.RequestsPerHour
	}

	if err := r.cache.Get(ctx, dayKey, &count); err == nil {
		result["day"] = config.RequestsPerDay - count
		if result["day"] < 0 {
			result["day"] = 0
		}
	} else {
		result["day"] = config.RequestsPerDay
	}

	return result, nil
}

var ErrRateLimitExceeded = RateLimitError{
	message: "rate limit exceeded",
}

type RateLimitError struct {
	message string
}

func (e RateLimitError) Error() string {
	return e.message
}

// Common configurations
var (
	AuthConfig = Config{
		RequestsPerMinute: 5,
		RequestsPerHour:   20,
		RequestsPerDay:    100,
	}

	ChatConfig = Config{
		RequestsPerMinute: 30,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
	}

	CallConfig = Config{
		RequestsPerMinute: 10,
		RequestsPerHour:   50,
		RequestsPerDay:    200,
	}

	AIConfig = Config{
		RequestsPerMinute: 20,
		RequestsPerHour:   500,
		RequestsPerDay:    2000,
	}
)
