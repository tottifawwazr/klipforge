package dependency

import (
	"context"
	"fmt"

	redisclient "github.com/redis/go-redis/v9"

	"github.com/klipforge/klipforge/services/api/internal/config"
)

// Redis owns the API's Redis client and implements health.Checker.
type Redis struct {
	client *redisclient.Client
}

func NewRedis(cfg config.RedisConfig) (*Redis, error) {
	options, err := redisclient.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis configuration: %w", err)
	}

	options.PoolSize = cfg.PoolSize
	options.MinIdleConns = cfg.MinIdleConnections
	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	return &Redis{client: redisclient.NewClient(options)}, nil
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Close() error {
	return r.client.Close()
}
