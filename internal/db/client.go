package db

import (
	"context"
	"fmt"
	"net/url"

	"github.com/crymfox/nac/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
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

// GetPersonalProjectID finds the ID of the default personal project.
// It first tries to find a project linked to a user, then falls back to any personal project.
func (c *Client) GetPersonalProjectID(ctx context.Context) (string, error) {
	var id string

	// Check if tables exist first to avoid noisy errors during migrations
	var exists bool
	_ = c.pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'project_relation')").Scan(&exists)
	if exists {
		// Try to find the project of the first owner/admin
		query := `
			SELECT pr."projectId" 
			FROM project_relation pr
			JOIN project p ON p.id = pr."projectId"
			WHERE p.type = 'personal'
			LIMIT 1
		`
		err := c.pool.QueryRow(ctx, query).Scan(&id)
		if err == nil {
			return id, nil
		}
	}

	_ = c.pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'project')").Scan(&exists)
	if exists {
		// Fallback to any personal project
		query := `SELECT id FROM project WHERE type = 'personal' LIMIT 1`
		err := c.pool.QueryRow(ctx, query).Scan(&id)
		if err == nil {
			return id, nil
		}
	}

	return "", fmt.Errorf("no personal project found")
}
