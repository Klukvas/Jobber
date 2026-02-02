package redis

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/redis/go-redis/v9"
)

// Client represents a Redis client
type Client struct {
	*redis.Client
}

// New creates a new Redis client
func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis: %w", err)
	}

	return &Client{Client: rdb}, nil
}

// Health checks the Redis health
func (c *Client) Health(ctx context.Context) error {
	return c.Ping(ctx).Err()
}
