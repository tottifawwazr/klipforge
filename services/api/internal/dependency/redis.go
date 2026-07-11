package dependency

import (
	"context"
	"fmt"
	"time"

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

func (r *Redis) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	pipe := r.client.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return count.Val() <= limit, nil
}
