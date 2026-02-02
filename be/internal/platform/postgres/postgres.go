package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client represents a PostgreSQL client
type Client struct {
	Pool *pgxpool.Pool
}

// New creates a new PostgreSQL client
func New(ctx context.Context, cfg config.DatabaseConfig) (*Client, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Set connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &Client{Pool: pool}, nil
}

// Close closes the database connection pool
func (c *Client) Close() {
	c.Pool.Close()
}

// Health checks the database health
func (c *Client) Health(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}
