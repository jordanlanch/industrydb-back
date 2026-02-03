package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jordanlanch/industrydb/ent"
	_ "github.com/lib/pq"
)

// Client holds the database client
type Client struct {
	Ent *ent.Client
	db  *sql.DB // Underlying database for pool stats
}

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	MaxOpenConns    int           // Maximum number of open connections
	MaxIdleConns    int           // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum amount of time a connection may be reused
	ConnMaxIdleTime time.Duration // Maximum amount of time a connection may be idle
}

// DefaultPoolConfig returns sensible defaults for connection pooling
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,                // PostgreSQL default is 100, we use 25% for this app
		MaxIdleConns:    5,                 // Keep some connections warm
		ConnMaxLifetime: 5 * time.Minute,   // Recycle connections every 5 minutes
		ConnMaxIdleTime: 10 * time.Minute,  // Close idle connections after 10 minutes
	}
}

// NewClient creates a new database client with connection pooling
func NewClient(databaseURL string) (*Client, error) {
	return NewClientWithPool(databaseURL, DefaultPoolConfig())
}

// NewClientWithPool creates a new database client with custom pool configuration
func NewClientWithPool(databaseURL string, poolCfg PoolConfig) (*Client, error) {
	// Open sql.DB first to configure connection pool
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(poolCfg.MaxOpenConns)
	db.SetMaxIdleConns(poolCfg.MaxIdleConns)
	db.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(poolCfg.ConnMaxIdleTime)

	log.Printf("✅ Database connection pool configured (max_open: %d, max_idle: %d, max_lifetime: %s, max_idle_time: %s)",
		poolCfg.MaxOpenConns, poolCfg.MaxIdleConns, poolCfg.ConnMaxLifetime, poolCfg.ConnMaxIdleTime)

	// Create Ent client from the configured sql.DB
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	// Run migrations
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed creating schema resources: %w", err)
	}

	log.Println("✅ Database connected and migrations applied")

	return &Client{
		Ent: client,
		db:  db,
	}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.Ent.Close()
}

// Ping checks if the database is reachable
func (c *Client) Ping(ctx context.Context) error {
	// Try a simple query to check connection
	_, err := c.Ent.User.Query().Limit(1).Count(ctx)
	return err
}

// Stats returns database connection pool statistics
func (c *Client) Stats() sql.DBStats {
	return c.db.Stats()
}
