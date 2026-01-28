package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jordanlanch/industrydb/ent"
	_ "github.com/lib/pq"
)

// Client holds the database client
type Client struct {
	Ent *ent.Client
}

// NewClient creates a new database client
func NewClient(databaseURL string) (*Client, error) {
	client, err := ent.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	// Run migrations
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed creating schema resources: %w", err)
	}

	log.Println("✅ Database connected and migrations applied")

	return &Client{
		Ent: client,
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
