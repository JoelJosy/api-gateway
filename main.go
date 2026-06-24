package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/JoelJosy/api-gateway/middleware"
	"github.com/JoelJosy/api-gateway/proxy"
	"github.com/JoelJosy/api-gateway/router"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pubKey, err := config.ParsePublicKeyPEM("./public.pem")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("API Gateway starting on port %d\n", cfg.Port)

	// init proxy
	r := router.NewRouter(cfg.Routes)
	p := proxy.NewProxy(r)

	handler := middleware.Chain(p, middleware.LoggerMiddleware, middleware.AuthMiddleware(cfg, pubKey))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), handler))
}