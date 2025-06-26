package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"

	"github.com/ASRafalsky/internal/config"
	"github.com/ASRafalsky/internal/server/middleware"
)

func Run(ctx context.Context, repo dataRepository, cfg config.Config, logger *log.Logger) {
	logger.Info("Starting server", cfg.Addr)

	srv := http.Server{
		Addr:    cfg.Addr,
		Handler: middleware.WithLogging(newRouter(repo, logger), logger),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server:", err.Error())
		}
	}()

	<-ctx.Done()
	if ctxErr := ctx.Err(); ctxErr != nil {
		logger.Info("Stopping server by cause:", ctxErr.Error())
	}

	srvCtx, srvCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer srvCancel()
	if err := srv.Shutdown(srvCtx); err != nil {
		logger.Fatal("Failed to shutdown server:", err.Error())
	}
	logger.Info("Server shutdown completed")
}
