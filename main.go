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
	pubKey, err := config.ParsePublicKeyPEM("./public.pem")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Load redis client
	rdb := redis.NewClient(&redis.Options{
    	Addr: cfg.Redis.Address,
		Password: "", // no password
		DB:       0,  // use default DB
		Protocol: 2,
	})
	// redis needs context to handle timeouts/cancellation
	ctx := context.Background()
	// test redis connection
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis unreachable: %v. Rate limiting will fail open.", err)
		return
	}
	log.Printf("Redis response: %s", pong)


	fmt.Printf("API Gateway starting on port %d\n", cfg.Port)

	// init proxy
	r := router.NewRouter(cfg.Routes)
	p := proxy.NewProxy(r)

	handler := middleware.Chain(p, middleware.LoggerMiddleware, middleware.AuthMiddleware(cfg, pubKey))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), handler))
}