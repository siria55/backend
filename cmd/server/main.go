package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"eeo/backend/internal/config"
	"eeo/backend/internal/server"
	actionservice "eeo/backend/internal/service/action"
	gameservice "eeo/backend/internal/service/game"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("open database failed: %v", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database failed: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	const defaultSceneID = "mars_outpost_min"

	gameSvc, err := gameservice.New(db, defaultSceneID)
	if err != nil {
		log.Fatalf("load scene failed: %v", err)
	}

	actionSvc := actionservice.New(db)

	srv := server.New(cfg, gameSvc, actionSvc)

	if err := srv.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
