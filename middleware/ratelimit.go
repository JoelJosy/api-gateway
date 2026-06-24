package middleware

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
	cfg *config.Config
	sha string // redis returned sha hash of script (fingerprint)
}

func NewRateLimiter(rdb *redis.Client, cfg *config.Config) (*RateLimiter, error) {
	// read lua script from file
	script, err := os.ReadFile("scripts/rate_limit.lua")
	if err != nil {
		return nil, err
	}

	// load script into redis
	sha, err := rdb.ScriptLoad(context.Background(), string(script)).Result()
	if err != nil {
		return nil, err
	}

	return &RateLimiter{
		rdb: rdb,
		cfg: cfg,
		sha: sha,
	}, nil
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int, error) {
	// KEYS and ARGV for Lua
	keys := []string{"rate_limit:" + key}
	args := []interface{}{
		rl.cfg.RateLimit.MaxTokens,
		rl.cfg.RateLimit.RefillRate,
		time.Now().Unix(), // current time in seconds
	}

	// run cached lua script
	res, err := rl.rdb.EvalSha(ctx, rl.sha, keys, args...).Result()
	if err != nil {
		return false, 0, err
	}

	// Redis returns []interface{}, type assertion
	results, ok := res.([]interface{})
	if !ok || len(results) < 2 {
		return false, 0, fmt.Errorf("invalid lua response")
	}

	// Lua returns numbers as int64
	allowed := results[0].(int64) == 1
	remaining := int(results[1].(int64))
	return allowed, remaining, nil
}