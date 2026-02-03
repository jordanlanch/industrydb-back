package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
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

// SSLConfig holds SSL/TLS configuration for database connections
type SSLConfig struct {
	Mode         string // disable, require, verify-ca, verify-full
	CertPath     string // Path to client certificate
	KeyPath      string // Path to client key
	RootCertPath string // Path to root CA certificate
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

// BuildConnectionString builds a PostgreSQL connection string with SSL parameters
func BuildConnectionString(baseURL string, sslCfg *SSLConfig) (string, error) {
	// If no SSL config provided, return base URL as-is
	if sslCfg == nil {
		return baseURL, nil
	}

	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Get existing query parameters
	query := parsedURL.Query()

	// Set SSL mode (overrides any existing sslmode in URL)
	if sslCfg.Mode != "" {
		query.Set("sslmode", sslCfg.Mode)
	}

	// Add SSL certificate paths if provided
	if sslCfg.CertPath != "" {
		query.Set("sslcert", sslCfg.CertPath)
	}
	if sslCfg.KeyPath != "" {
		query.Set("sslkey", sslCfg.KeyPath)
	}
	if sslCfg.RootCertPath != "" {
		query.Set("sslrootcert", sslCfg.RootCertPath)
	}

	// Rebuild URL with updated query parameters
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// NewClient creates a new database client with connection pooling
func NewClient(databaseURL string) (*Client, error) {
	return NewClientWithPool(databaseURL, DefaultPoolConfig())
}

// NewClientWithSSL creates a new database client with SSL configuration
func NewClientWithSSL(databaseURL string, sslCfg *SSLConfig) (*Client, error) {
	return NewClientWithPoolAndSSL(databaseURL, DefaultPoolConfig(), sslCfg)
}

// NewClientWithPool creates a new database client with custom pool configuration
func NewClientWithPool(databaseURL string, poolCfg PoolConfig) (*Client, error) {
	return NewClientWithPoolAndSSL(databaseURL, poolCfg, nil)
}

// NewClientWithPoolAndSSL creates a new database client with custom pool and SSL configuration
func NewClientWithPoolAndSSL(databaseURL string, poolCfg PoolConfig, sslCfg *SSLConfig) (*Client, error) {
	// Build connection string with SSL parameters
	connStr, err := BuildConnectionString(databaseURL, sslCfg)
	if err != nil {
		return nil, fmt.Errorf("failed building connection string: %w", err)
	}

	// Log SSL mode if configured
	if sslCfg != nil && sslCfg.Mode != "" && sslCfg.Mode != "disable" {
		log.Printf("🔒 Database SSL enabled (mode: %s)", sslCfg.Mode)
		if sslCfg.CertPath != "" {
			log.Printf("   Client certificate: %s", sslCfg.CertPath)
		}
		if sslCfg.RootCertPath != "" {
			log.Printf("   Root CA certificate: %s", sslCfg.RootCertPath)
		}
	}

	// Open sql.DB first to configure connection pool
	db, err := sql.Open("postgres", connStr)
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
