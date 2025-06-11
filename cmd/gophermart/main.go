package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/ASRafalsky/telemetry/pkg/log"

	"github.com/ASRafalsky/internal/config"
	"github.com/ASRafalsky/internal/db/postgres"
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

	server.Run(ctx, db, cfg, Log)

	Log.Info("Server stopped.")
}
