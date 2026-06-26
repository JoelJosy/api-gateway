package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/JoelJosy/api-gateway/middleware"
	"github.com/JoelJosy/api-gateway/proxy"
	"github.com/JoelJosy/api-gateway/router"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	// Load public key for jwt verification
	pubKey, err := config.ParsePublicKeyPEM(cfg.PubKeyPath)
	if err != nil {
		log.Fatalf("failed to load public key: %v", err)
	}

	// Load redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: "", // no password
		DB:       0,  // use default DB
		Protocol: 2,
	})
	// redis needs context to handle timeouts/cancellation
	ctx := context.Background()
	// test redis connection
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		// fail fast
		log.Fatalf("Redis not available: %v", err)
	}
	log.Printf("Redis connected: %s", pong)

	fmt.Printf("API Gateway starting on port %d\n", cfg.Port)

	// init ratelimiter
	rl, err := middleware.NewRateLimiter(rdb, cfg)
	if err != nil {
		fmt.Printf("RateLimiter not initialized: %v", err)
	}

	// init proxy
	r := router.NewRouter(cfg.Routes)
	p := proxy.NewProxy(r, cfg.Proxy)

	handler := middleware.Chain(
		p,
		middleware.LoggerMiddleware,
		middleware.AuthMiddleware(cfg, pubKey),
		rl.Middleware())

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), handler))
}
