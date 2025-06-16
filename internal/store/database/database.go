package database

import (
	"database/sql"
	"embed"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"time"
)

func Open(addr string, maxOpenConns, maxIdleConns int, maxLifetime string) (*sql.DB, error) {
	db, err := sql.Open("pgx", addr)

	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	lifetime, err := time.ParseDuration(maxLifetime)

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(lifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func MigrateFS(db *sql.DB, fs embed.FS, dir string) error {
	goose.SetBaseFS(fs)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, dir); err != nil {
		return err
	}

	return nil
}
