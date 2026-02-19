package db

import (
	"context"
	"fmt"
	"time"

	"imagepp/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Open creates the connection pool and verifies connectivity
func Open(cfg *config.Config) error {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 15
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	return nil
}

// Get returns the shared pool so other packages can run queries
func Get() *pgxpool.Pool {
	return pool
}

// Close shuts down the pool when the app exits
func Close() {
	if pool != nil {
		pool.Close()
		pool = nil
	}
}
