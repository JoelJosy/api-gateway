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
	"github.com/JoelJosy/api-gateway/health"
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

	if v := os.Getenv("REDIS_ADDR"); v != "" {
    	cfg.Redis.Address = v
	}

	// Load public key for jwt verification
	pubKey, err := config.LoadPublicKey(*cfg)
	if err != nil {
		log.Fatalf("failed to load public key: %v", err)
	}

	// Load redis client
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{cfg.Redis.Address},
		Password: "", // no password
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
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
		log.Fatalf("RateLimiter not initialized: %v", err)
	}

	// init proxy and health handler
	r := router.NewRouter(cfg.Routes)
	p := proxy.NewProxy(r, cfg.Proxy)
	healthHandler := health.NewHandler(rdb)

	// channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	// notify if ctrl c / kill and propagate into channel
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// server request handler
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler)
	mux.Handle("/", middleware.Chain(
		p,
		middleware.LoggerMiddleware,
		middleware.AuthMiddleware(cfg, pubKey),
		rl.Middleware(),
	))

	// init server manually
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	// non blocking fn to listen and serve
	go func() {
		log.Printf("API Gateway starting on port %d", cfg.Port)
		err := srv.ListenAndServe()
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
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Gateway stopped. Goodbye!")
}
