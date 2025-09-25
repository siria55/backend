package main

import (
	"log"

	"eeo/backend/internal/config"
	"eeo/backend/internal/server"
	agentservice "eeo/backend/internal/service/agent"
	gameservice "eeo/backend/internal/service/game"
)

func main() {
	cfg := config.Load()

	agentSvc := agentservice.New()
	gameSvc, err := gameservice.New()
	if err != nil {
		log.Fatalf("load scene failed: %v", err)
	}

	srv := server.New(cfg, agentSvc, gameSvc)

	if err := srv.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
