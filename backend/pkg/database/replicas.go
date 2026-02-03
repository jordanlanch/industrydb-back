package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jordanlanch/industrydb/ent"
)

// ReplicaConfig holds configuration for read replicas
type ReplicaConfig struct {
	// ReadReplicaURLs is a list of read replica connection strings
	ReadReplicaURLs []string

	// LoadBalanceStrategy determines how to distribute read queries
	// Options: "random", "round-robin", "least-connections"
	LoadBalanceStrategy string

	// FallbackToPrimary enables falling back to primary if all replicas fail
	FallbackToPrimary bool

	// HealthCheckInterval is how often to check replica health
	HealthCheckInterval time.Duration
}

// DefaultReplicaConfig returns default configuration for read replicas
func DefaultReplicaConfig() ReplicaConfig {
	return ReplicaConfig{
		ReadReplicaURLs:     []string{},
		LoadBalanceStrategy: "round-robin",
		FallbackToPrimary:   true,
		HealthCheckInterval: 30 * time.Second,
	}
}

// ClientWithReplicas extends Client to support read replicas
type ClientWithReplicas struct {
	*Client // Embeds regular client (primary connection)

	// Read replicas
	readReplicas []*replicaConnection
	replicaMu    sync.RWMutex

	// Load balancing
	rrIndex uint64 // Round-robin index (atomic)
	config  ReplicaConfig

	// Health checking
	healthCheckStop chan struct{}
	healthCheckWg   sync.WaitGroup
}

type replicaConnection struct {
	db      *sql.DB
	entCli  *ent.Client
	url     string
	healthy bool
	mu      sync.RWMutex
}

// NewClientWithReplicas creates a database client with read replica support
func NewClientWithReplicas(primaryURL string, poolCfg PoolConfig, sslCfg *SSLConfig, replicaCfg ReplicaConfig) (*ClientWithReplicas, error) {
	// Create primary connection
	primaryClient, err := NewClientWithPoolAndSSL(primaryURL, poolCfg, sslCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary client: %w", err)
	}

	client := &ClientWithReplicas{
		Client:          primaryClient,
		readReplicas:    make([]*replicaConnection, 0, len(replicaCfg.ReadReplicaURLs)),
		config:          replicaCfg,
		healthCheckStop: make(chan struct{}),
	}

	// Connect to read replicas
	for _, replicaURL := range replicaCfg.ReadReplicaURLs {
		replica, err := client.connectReplica(replicaURL, poolCfg, sslCfg)
		if err != nil {
			log.Printf("⚠️  Failed to connect to read replica %s: %v", replicaURL, err)
			continue // Skip failed replicas, don't fail entire initialization
		}
		client.readReplicas = append(client.readReplicas, replica)
	}

	if len(client.readReplicas) > 0 {
		log.Printf("✅ Connected to %d read replica(s)", len(client.readReplicas))

		// Start health checking if replicas exist
		if replicaCfg.HealthCheckInterval > 0 {
			client.startHealthChecking()
		}
	} else {
		log.Printf("ℹ️  No read replicas configured, all queries will use primary")
	}

	return client, nil
}

// connectReplica creates a connection to a read replica
func (c *ClientWithReplicas) connectReplica(replicaURL string, poolCfg PoolConfig, sslCfg *SSLConfig) (*replicaConnection, error) {
	// Build connection string with SSL
	connStr, err := BuildConnectionString(replicaURL, sslCfg)
	if err != nil {
		return nil, fmt.Errorf("failed building connection string: %w", err)
	}

	// Open connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection: %w", err)
	}

	// Configure pool (read replicas typically need fewer connections)
	readPoolCfg := poolCfg
	readPoolCfg.MaxOpenConns = poolCfg.MaxOpenConns / 2 // Half the connections of primary
	if readPoolCfg.MaxOpenConns < 5 {
		readPoolCfg.MaxOpenConns = 5
	}

	db.SetMaxOpenConns(readPoolCfg.MaxOpenConns)
	db.SetMaxIdleConns(readPoolCfg.MaxIdleConns)
	db.SetConnMaxLifetime(readPoolCfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(readPoolCfg.ConnMaxIdleTime)

	// Create Ent client
	drv := entsql.OpenDB(dialect.Postgres, db)
	entClient := ent.NewClient(ent.Driver(drv))

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping replica: %w", err)
	}

	log.Printf("✅ Read replica connected: %s", replicaURL)

	return &replicaConnection{
		db:      db,
		entCli:  entClient,
		url:     replicaURL,
		healthy: true,
	}, nil
}

// GetReadClient returns an Ent client for read operations (may be replica or primary)
func (c *ClientWithReplicas) GetReadClient() *ent.Client {
	// If no replicas, use primary
	if len(c.readReplicas) == 0 {
		return c.Ent
	}

	// Select replica based on load balancing strategy
	replica := c.selectReplica()
	if replica != nil {
		replica.mu.RLock()
		healthy := replica.healthy
		replica.mu.RUnlock()

		if healthy {
			return replica.entCli
		}
	}

	// Fallback to primary if configured
	if c.config.FallbackToPrimary {
		return c.Ent
	}

	// If no fallback, try to find any healthy replica
	for _, r := range c.readReplicas {
		r.mu.RLock()
		healthy := r.healthy
		r.mu.RUnlock()

		if healthy {
			return r.entCli
		}
	}

	// Last resort: use primary
	return c.Ent
}

// GetWriteClient returns the primary Ent client for write operations
func (c *ClientWithReplicas) GetWriteClient() *ent.Client {
	return c.Ent
}

// selectReplica selects a replica based on configured load balancing strategy
func (c *ClientWithReplicas) selectReplica() *replicaConnection {
	c.replicaMu.RLock()
	defer c.replicaMu.RUnlock()

	if len(c.readReplicas) == 0 {
		return nil
	}

	switch c.config.LoadBalanceStrategy {
	case "random":
		return c.readReplicas[rand.Intn(len(c.readReplicas))]

	case "round-robin":
		index := atomic.AddUint64(&c.rrIndex, 1)
		return c.readReplicas[index%uint64(len(c.readReplicas))]

	case "least-connections":
		// Simple implementation: just use round-robin
		// Full implementation would track active connections per replica
		index := atomic.AddUint64(&c.rrIndex, 1)
		return c.readReplicas[index%uint64(len(c.readReplicas))]

	default:
		// Default to round-robin
		index := atomic.AddUint64(&c.rrIndex, 1)
		return c.readReplicas[index%uint64(len(c.readReplicas))]
	}
}

// startHealthChecking starts a goroutine that periodically checks replica health
func (c *ClientWithReplicas) startHealthChecking() {
	c.healthCheckWg.Add(1)

	go func() {
		defer c.healthCheckWg.Done()

		ticker := time.NewTicker(c.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.checkReplicaHealth()
			case <-c.healthCheckStop:
				return
			}
		}
	}()

	log.Printf("✅ Replica health checking started (interval: %s)", c.config.HealthCheckInterval)
}

// checkReplicaHealth checks the health of all read replicas
func (c *ClientWithReplicas) checkReplicaHealth() {
	c.replicaMu.RLock()
	defer c.replicaMu.RUnlock()

	for _, replica := range c.readReplicas {
		go func(r *replicaConnection) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := r.db.PingContext(ctx)

			r.mu.Lock()
			wasHealthy := r.healthy
			r.healthy = (err == nil)
			r.mu.Unlock()

			// Log status changes
			if wasHealthy && !r.healthy {
				log.Printf("⚠️  Read replica became unhealthy: %s (error: %v)", r.url, err)
			} else if !wasHealthy && r.healthy {
				log.Printf("✅ Read replica recovered: %s", r.url)
			}
		}(replica)
	}
}

// GetReplicaStats returns statistics about read replicas
func (c *ClientWithReplicas) GetReplicaStats() map[string]interface{} {
	c.replicaMu.RLock()
	defer c.replicaMu.RUnlock()

	stats := map[string]interface{}{
		"total_replicas":   len(c.readReplicas),
		"healthy_replicas": 0,
		"replicas":         []map[string]interface{}{},
	}

	healthyCount := 0
	for _, replica := range c.readReplicas {
		replica.mu.RLock()
		healthy := replica.healthy
		replica.mu.RUnlock()

		if healthy {
			healthyCount++
		}

		stats["replicas"] = append(stats["replicas"].([]map[string]interface{}), map[string]interface{}{
			"url":     replica.url,
			"healthy": healthy,
			"stats":   replica.db.Stats(),
		})
	}

	stats["healthy_replicas"] = healthyCount

	return stats
}

// Close closes all database connections (primary and replicas)
func (c *ClientWithReplicas) Close() error {
	// Stop health checking
	close(c.healthCheckStop)
	c.healthCheckWg.Wait()

	// Close replicas
	c.replicaMu.Lock()
	for _, replica := range c.readReplicas {
		if err := replica.entCli.Close(); err != nil {
			log.Printf("⚠️  Error closing replica connection: %v", err)
		}
		if err := replica.db.Close(); err != nil {
			log.Printf("⚠️  Error closing replica database: %v", err)
		}
	}
	c.replicaMu.Unlock()

	// Close primary
	return c.Client.Close()
}
