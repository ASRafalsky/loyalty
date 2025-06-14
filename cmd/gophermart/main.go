package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/ASRafalsky/telemetry/pkg/log"

	"github.com/ASRafalsky/internal/client"
	"github.com/ASRafalsky/internal/config"
	"github.com/ASRafalsky/internal/db/postgres"
	"github.com/ASRafalsky/internal/repository"
	"github.com/ASRafalsky/internal/server"
)

func main() {
	cfg, err := config.UpdateCfg()
	if err != nil {
		panic(err)
	}

	Log, err := log.AddLoggerWith(cfg.LogLevel, cfg.LogPath)
	if err != nil {
		panic(err)
	}
	defer Log.Sync()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	db, err := postgres.InitDB(ctx, cfg.DB, Log)
	if err != nil {
		Log.Fatal("Database initialization failed", err.Error())
	}

	repo := repository.NewExtendedRepository(db)
	enricher := client.NewEnricher(cfg, repo)
	if err = enricher.Run(ctx, Log); err != nil {
		Log.Fatal("Enricher failed", err.Error())
	}
	server.Run(ctx, repo, cfg, Log)

	Log.Info("Server stopped.")
}
