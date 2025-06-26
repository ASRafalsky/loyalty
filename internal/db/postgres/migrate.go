package postgres

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (d DB) MigrateUp(path string) error {
	driver, err := postgres.WithInstance(d.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+path,
		"postgres", driver)
	if err != nil {
		return err
	}

	return m.Up()
}

func (d DB) MigrateRollback(path string, stepCnt int) error {
	driver, err := postgres.WithInstance(d.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+path,
		"postgres", driver)
	if err != nil {
		return err
	}

	return m.Steps(-stepCnt)
}
