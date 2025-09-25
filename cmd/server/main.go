package main

import (
	"log"

	"eeo/backend/internal/config"
	"eeo/backend/internal/server"
	gameservice "eeo/backend/internal/service/game"
)

func main() {
	cfg := config.Load()

	gameSvc, err := gameservice.New()
	if err != nil {
		log.Fatalf("load scene failed: %v", err)
	}

	srv := server.New(cfg, gameSvc)

	if err := srv.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
