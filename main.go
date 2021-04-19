package main

import (
	"log"

	"github.com/b-harvest/gravity-dex-backend/config"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := cfg.Log.Build()
	if err != nil {
		log.Fatalf("failed to build logger: %v", err)
	}
	defer logger.Sync()
}
