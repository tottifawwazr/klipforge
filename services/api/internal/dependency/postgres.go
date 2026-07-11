package dependency

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/config"
)

// Postgres owns the API's connection pool and implements health.Checker.
type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, cfg config.DatabaseConfig) (*Postgres, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnectionLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnectionIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) Close() {
	p.pool.Close()
}
