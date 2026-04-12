package database

import (
	"context"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func NewDatabase(dbURL string, migraURL string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = 32

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	if migraURL != "" {
		if err := runMigrations(migraURL, dbURL); err != nil {
			pool.Close()
			return nil, err
		}
	}
	return &Postgres{Pool: pool}, nil
}

func runMigrations(migraURL, dbURL string) error {
	m, err := migrate.New(migraURL, dbURLForMigrate(dbURL))
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	log.Println("Миграции применены или изменений не было")
	return nil
}

func dbURLForMigrate(dbURL string) string {
	return strings.Replace(dbURL, "postgresql://", "postgres://", 1)
}

func (p *Postgres) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
