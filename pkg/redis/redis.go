package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/redis"
)

const (
	StatusCacheTTL    = 24 * time.Hour
	StatusCachePrefix = "notification:status:"
)

func NewRedisClient(cfg *config.Config, ctx context.Context) (*redis.Client, error) {
	options := redis.Options{
		Address:  fmt.Sprintf("%s:%d", cfg.GetString("REDIS_HOST"), cfg.GetInt("REDIS_PORT")),
		Password: cfg.GetString("REDIS_PASSWORD"),
	}

	client, err := redis.Connect(options)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func CacheKey(id string) string {
	return StatusCachePrefix + id
}
