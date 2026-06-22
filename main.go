package main

import (
	"fmt"
	"log"

	"github.com/JoelJosy/api-gateway/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("API Gateway starting on port %d\n", cfg.Port)
	fmt.Println("Loaded routes:")
	for _, route := range cfg.Routes {
		fmt.Printf("- %s -> %s\n", route.Path, route.Upstream)
	}
}