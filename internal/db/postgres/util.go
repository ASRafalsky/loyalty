package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"github.com/golang-migrate/migrate/v4"

	"github.com/ASRafalsky/internal/config"
)

func (d DB) WaitDBIsReady(ctx context.Context, tryCnt int, timeout time.Duration) error {
	var err error
	for range tryCnt {
		ctxPing, cancel := context.WithTimeout(ctx, timeout)
		err = d.Ping(ctxPing)
		cancel()
	}
	return err
}

func InitDB(ctx context.Context, cfg config.DB, l *log.Logger) (DB, error) {
	if cfg.DSN == "" {
		return DB{}, errors.New("database not configured")
	}
	db, err := Open(cfg.DSN)
	if err == nil {
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)
		l.Info("Opened database with dsn", "dsn", cfg.DSN)
		if err = db.WaitDBIsReady(ctx, 5, time.Second); err != nil {
			l.Error("failed to ping to database: ", err.Error())
		}
		if err = db.MigrateUp(cfg.MigrationsPath); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				l.Info("Nothing to migrate")
			} else {
				l.Error("failed to migrate database: ", err.Error())
				if err = db.MigrateRollback(cfg.MigrationsPath, 1); err != nil {
					l.Error("failed to rollback database: ", err.Error())
				}
			}
		}
	} else {
		return DB{}, errors.New("failed to connect to database")
	}
	return db, nil
}
