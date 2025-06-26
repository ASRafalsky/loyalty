package config

import (
	"flag"
	"path/filepath"
	"runtime"

	"github.com/caarlos0/env/v10"
)

const (
	defaultAddr     = ":8080"
	defaultDBAddr   = ""
	defaultLogLevel = "info"
)

type CommonFields struct {
	Addr     string `env:"RUN_ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
	LogPath  string `env:"LOG_PATH"`
	Key      string `env:"KEY"`
}

type DB struct {
	DSN             string `env:"DATABASE_URI"`
	MigrationsPath  string
	MigrationsTable string
}

type Config struct {
	CommonFields
	DB          DB
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func UpdateCfg() (Config, error) {
	var migrationPath string
	_, file, _, ok := runtime.Caller(0)
	if ok {
		migrationPath = filepath.Dir(file) + "/db/migrations"
	}
	cfg := Config{}
	flag.StringVar(&cfg.Addr, "a", defaultAddr, "address and port to run server")
	flag.StringVar(&cfg.AccrualAddr, "r", defaultAddr, "accrual system address")
	flag.StringVar(&cfg.LogLevel, "l", defaultLogLevel, "log level")
	flag.StringVar(&cfg.LogPath, "p", "", "log file path")
	flag.StringVar(&cfg.DB.DSN, "d", defaultDBAddr, "database address")
	flag.StringVar(&cfg.DB.MigrationsPath, "m", migrationPath, "path to migrations")
	flag.StringVar(&cfg.DB.MigrationsTable, "t", "", "name of migration table, where migrator writes own data")
	flag.Parse()

	err := env.Parse(&cfg)
	return cfg, err
}
