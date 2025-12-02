package main

import (
	"log"
	"order-service/internal/app"
	"order-service/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and run application
	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
