package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.ClusterClient
	cfg *config.Config
	sha string // redis returned sha hash of script (fingerprint)
}

func NewRateLimiter(rdb *redis.ClusterClient, cfg *config.Config) (*RateLimiter, error) {
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

func (rl *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string
			if (rl.cfg.RateLimit.KeyBy == "ip") {
				key = strings.Split(r.RemoteAddr, ":")[0]
			} else {
				// Safely read the public claims key from your auth file
				if userId, ok := r.Context().Value(claimsKey).(string); ok {
					key = userId
				} else {
					// Fallback to IP if the route was public and has no user context
					key = strings.Split(r.RemoteAddr, ":")[0]
				}
			}

			allow, remaining, err := rl.Allow(r.Context(), key)
			
			// Set the rate limit response headers 
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.cfg.RateLimit.MaxTokens))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			// Fail closed: intercept if redis outage
			if err != nil {
				// Log the internal database error so operations can see it
				slog.Error("rate limiter redis error", "err", err, "key", key)
				
				// Block the user with a 500 or 503 since it's a gateway/infrastructure issue
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
				return
			}

			// Rate limit exceeded
			if !allow {
				// Calculate wait time: 1 token divided by tokens-per-second refill rate
				refill := float64(rl.cfg.RateLimit.RefillRate)
    			waitTime := 1.0 / refill
				retryAfter := int(math.Ceil(waitTime))
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				
				// Block request
				// to return json error response instead of plain text
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Too many requests"}`))
				return
			} 
			next.ServeHTTP(w, r)
		})
	}
}