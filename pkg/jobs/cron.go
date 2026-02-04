package jobs

import (
	"context"
	"log"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/robfig/cron/v3"
)

// CronManager manages scheduled jobs
type CronManager struct {
	cron    *cron.Cron
	monitor *DataMonitor
	logger  *log.Logger
}

// NewCronManager creates a new cron manager
func NewCronManager(db *ent.Client, cache *cache.Client, logger *log.Logger) *CronManager {
	if logger == nil {
		logger = log.Default()
	}

	return &CronManager{
		cron:    cron.New(),
		monitor: NewDataMonitor(db, cache, logger),
		logger:  logger,
	}
}

// SetupJobs configures all scheduled jobs
func (cm *CronManager) SetupJobs() error {
	cm.logger.Println("Setting up cron jobs...")

	// Daily at 2 AM: Populate industries with low data (< 100 leads)
	_, err := cm.cron.AddFunc("0 2 * * *", func() {
		cm.logger.Println("🕐 Running daily data population job...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Detect industries with < 100 leads
		pairs, err := cm.monitor.DetectLowDataIndustries(ctx, 100)
		if err != nil {
			cm.logger.Printf("❌ Failed to detect low data industries: %v", err)
			return
		}

		if len(pairs) == 0 {
			cm.logger.Println("✅ No industries with low data found")
			return
		}

		cm.logger.Printf("Found %d industry/country pairs with < 100 leads", len(pairs))

		// Populate top 10 with lowest data (highest priority)
		count := 10
		if len(pairs) < count {
			count = len(pairs)
		}

		topPairs := pairs[:count]
		cm.logger.Printf("Populating top %d pairs...", count)

		// Trigger fetches (max 3 concurrent)
		if err := cm.monitor.TriggerDataFetchBatch(ctx, topPairs, 1000, 3); err != nil {
			cm.logger.Printf("⚠️ Batch fetch completed with errors: %v", err)
			return
		}

		cm.logger.Println("✅ Daily data population job completed")
	})

	if err != nil {
		return err
	}

	// Weekly on Sunday at 3 AM: Detect and populate missing combinations
	_, err = cm.cron.AddFunc("0 3 * * 0", func() {
		cm.logger.Println("🕐 Running weekly missing data detection job...")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()

		// Detect missing combinations
		pairs, err := cm.monitor.DetectMissingCombinations(ctx)
		if err != nil {
			cm.logger.Printf("❌ Failed to detect missing combinations: %v", err)
			return
		}

		if len(pairs) == 0 {
			cm.logger.Println("✅ No missing combinations found")
			return
		}

		cm.logger.Printf("Found %d missing combinations", len(pairs))

		// Populate top 20 missing combinations
		count := 20
		if len(pairs) < count {
			count = len(pairs)
		}

		topPairs := pairs[:count]
		cm.logger.Printf("Populating top %d missing combinations...", count)

		// Trigger fetches (max 5 concurrent)
		if err := cm.monitor.TriggerDataFetchBatch(ctx, topPairs, 500, 5); err != nil {
			cm.logger.Printf("⚠️ Batch fetch completed with errors: %v", err)
			return
		}

		cm.logger.Println("✅ Weekly missing data detection job completed")
	})

	if err != nil {
		return err
	}

	// Daily at 4 AM: Log population statistics
	_, err = cm.cron.AddFunc("0 4 * * *", func() {
		cm.logger.Println("🕐 Logging population statistics...")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		stats, err := cm.monitor.GetPopulationStats(ctx)
		if err != nil {
			cm.logger.Printf("❌ Failed to get population stats: %v", err)
			return
		}

		cm.logger.Printf("📊 Population Statistics:")
		cm.logger.Printf("  Total leads: %v", stats["total_leads"])
		cm.logger.Printf("  Total combinations: %v", stats["total_combinations"])
		cm.logger.Printf("  Top industries: %v", stats["top_industries"])
		cm.logger.Printf("  Top countries: %v", stats["top_countries"])
	})

	if err != nil {
		return err
	}

	cm.logger.Println("✅ Cron jobs configured successfully")
	cm.logger.Println("  - Daily at 2 AM: Populate low-data industries")
	cm.logger.Println("  - Weekly on Sunday at 3 AM: Populate missing combinations")
	cm.logger.Println("  - Daily at 4 AM: Log statistics")

	return nil
}

// Start starts the cron scheduler
func (cm *CronManager) Start() {
	cm.logger.Println("🚀 Starting cron scheduler...")
	cm.cron.Start()
}

// Stop stops the cron scheduler
func (cm *CronManager) Stop() {
	cm.logger.Println("🛑 Stopping cron scheduler...")
	cm.cron.Stop()
}

// GetMonitor returns the data monitor (for manual triggers)
func (cm *CronManager) GetMonitor() *DataMonitor {
	return cm.monitor
}
