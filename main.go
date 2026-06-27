package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	err = rdb.Ping(ctx).Err()
	if err != nil {
		// fail fast
		log.Fatalf("Redis not available: %v", err)
	}

	// init ratelimiter
	rl, err := middleware.NewRateLimiter(rdb, cfg)
	if err != nil {
		fmt.Printf("RateLimiter not initialized: %v", err)
	}

	// init proxy
	r := router.NewRouter(cfg.Routes)
	p := proxy.NewProxy(r, cfg.Proxy)

	// channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	// notify if ctrl c / kill and propagate into channel
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	
	// server request handler
	handler := middleware.Chain(
		p,
		middleware.LoggerMiddleware,
		middleware.AuthMiddleware(cfg, pubKey),
		rl.Middleware())

	// init server manually
	srv := &http.Server{
    	Addr:    fmt.Sprintf(":%d", cfg.Port),
    	Handler: handler,
	}

	// non blocking fn to listen and serve
	go func() {
    	log.Printf("API Gateway starting on port %d", cfg.Port)
    	err := srv.ListenAndServe(); 
		if err != nil && err != http.ErrServerClosed {
        	log.Fatalf("Server failed to start: %v", err)
    	}
	}()
	
	// blocks main, until ctrl c / kill
	<-stop
	log.Println("Shutting down gateway...")
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// stops server from accepting new connections
	// waits for ongoing reqs to finish (within timeout)
	err = srv.Shutdown(shutdownCtx); 
	if err != nil {
    	log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Gateway stopped. Goodbye!")
}
