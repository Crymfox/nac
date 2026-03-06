package db

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/crymfox/nac/internal/config"
)

// Client wraps a pgx connection pool.
type Client struct {
	pool *pgxpool.Pool
}

// NewClient creates a new Postgres connection pool from a DBConfig.
func NewClient(ctx context.Context, dbCfg config.DBConfig) (*Client, error) {
	host, port, dbName, user, pass, err := config.ResolveDBConfig(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("resolving DB config: %w", err)
	}

	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		host,
		port,
		url.QueryEscape(dbName),
	)

	// Append SSL options
	sslMode := "disable"
	if dbCfg.SSL {
		if dbCfg.SSLRejectUnauthorized {
			sslMode = "verify-full"
		} else {
			sslMode = "require"
		}
	}
	dsn += fmt.Sprintf("?sslmode=%s", sslMode)

	// Configure pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	// Connect
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &Client{pool: pool}, nil
}

// Close closes the database connection pool.
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// Pool returns the underlying pgxpool.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}
