package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/testdata"
)

// Industry configuration: how many leads to generate per industry
var industryConfig = map[string]int{
	// Personal Care & Beauty (5 industries): 200 leads each = 1,000 total
	"tattoo": 200,
	"beauty": 200,
	"barber": 200,
	"spa":    200,
	"nail":   200,

	// Health & Wellness (4 industries): 150 leads each = 600 total
	"dentist":  150,
	"pharmacy": 150,
	"massage":  150,
	"gym":      150,

	// Food & Beverage (4 industries): 150 leads each = 600 total
	"restaurant": 150,
	"cafe":       150,
	"bar":        150,
	"bakery":     150,

	// Automotive (3 industries): 100 leads each = 300 total
	"car_repair": 100,
	"car_wash":   100,
	"car_dealer": 100,

	// Retail (2 industries): 100 leads each = 200 total
	"clothing":    100,
	"convenience": 100,

	// Professional Services (2 industries): 100 leads each = 200 total
	"lawyer":     100,
	"accountant": 100,
}

type progressBar struct {
	total   int
	current int
	width   int
	start   time.Time
}

func newProgressBar(total int) *progressBar {
	return &progressBar{
		total:   total,
		current: 0,
		width:   50,
		start:   time.Now(),
	}
}

func (p *progressBar) update(current int) {
	p.current = current
	percent := float64(current) / float64(p.total) * 100
	filled := int(float64(p.width) * float64(current) / float64(p.total))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	// Calculate ETA
	elapsed := time.Since(p.start)
	var eta time.Duration
	if current > 0 {
		eta = time.Duration(float64(elapsed) / float64(current) * float64(p.total-current))
	}

	fmt.Printf("\r[%s] %d/%d (%.1f%%) | Elapsed: %s | ETA: %s",
		bar, current, p.total, percent,
		elapsed.Round(time.Second),
		eta.Round(time.Second))
}

func (p *progressBar) finish() {
	p.update(p.total)
	fmt.Println() // New line after completion
}

func main() {
	// Command line flags
	industries := flag.String("industries", "", "Comma-separated list of industries to seed (e.g., tattoo,beauty,gym). Empty = all industries")
	reset := flag.Bool("reset", false, "Delete all existing leads before seeding")
	batchSize := flag.Int("batch-size", 100, "Number of leads to insert per batch")
	flag.Parse()

	// Parse industries filter
	var industriesToSeed []string
	if *industries == "" {
		// Seed all industries
		for industry := range industryConfig {
			industriesToSeed = append(industriesToSeed, industry)
		}
	} else {
		industriesToSeed = strings.Split(*industries, ",")
	}

	// Database connection
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable"
	}

	client, err := ent.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Reset database if requested
	if *reset {
		fmt.Println("⚠️  Resetting database (deleting all leads)...")
		deleted, err := client.Lead.Delete().Exec(ctx)
		if err != nil {
			log.Fatalf("Failed to reset database: %v", err)
		}
		fmt.Printf("✅ Deleted %d existing leads\n\n", deleted)
	}

	// Calculate total leads to generate
	totalLeads := 0
	for _, industry := range industriesToSeed {
		if count, ok := industryConfig[industry]; ok {
			totalLeads += count
		}
	}

	fmt.Printf("🌱 Seeding %d leads across %d industries...\n\n", totalLeads, len(industriesToSeed))

	pb := newProgressBar(totalLeads)
	processedLeads := 0

	// Seed each industry
	for _, industry := range industriesToSeed {
		count, ok := industryConfig[industry]
		if !ok {
			fmt.Printf("⚠️  Unknown industry: %s (skipping)\n", industry)
			continue
		}

		fmt.Printf("\n📊 Seeding %s: %d leads\n", industry, count)

		// Generate leads with quality/completeness distribution
		leads := testdata.GenerateLeadsWithDistribution(industry, count)

		// Check for duplicates
		existingLeads, err := client.Lead.Query().
			Where(lead.IndustryEQ(lead.Industry(industry))).
			Count(ctx)

		if err != nil {
			log.Printf("Warning: Failed to check existing leads for %s: %v", industry, err)
		} else if existingLeads > 0 {
			fmt.Printf("ℹ️  Found %d existing %s leads (skipping duplicates)\n", existingLeads, industry)
		}

		// Insert in batches
		startTime := time.Now()
		if err := testdata.BulkInsertLeads(ctx, client, leads, *batchSize); err != nil {
			log.Printf("❌ Failed to seed %s: %v", industry, err)
			continue
		}

		processedLeads += count
		pb.update(processedLeads)

		duration := time.Since(startTime)
		fmt.Printf(" ✅ Completed in %s (%.0f leads/sec)\n",
			duration.Round(time.Millisecond),
			float64(count)/duration.Seconds())
	}

	pb.finish()

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📈 SEEDING SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	// Count leads by industry
	for _, industry := range industriesToSeed {
		count, err := client.Lead.Query().
			Where(lead.IndustryEQ(lead.Industry(industry))).
			Count(ctx)
		if err != nil {
			log.Printf("Failed to count %s leads: %v", industry, err)
			continue
		}
		fmt.Printf("%-15s: %4d leads\n", industry, count)
	}

	// Overall stats
	totalCount, _ := client.Lead.Query().Count(ctx)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("TOTAL: %d leads\n", totalCount)

	// Country distribution
	fmt.Println("\n📍 Geographic Distribution:")
	countries := []string{"US", "GB", "DE", "ES", "FR", "CA", "AU"}
	for _, country := range countries {
		count, err := client.Lead.Query().
			Where(lead.CountryEQ(country)).
			Count(ctx)
		if err != nil {
			continue
		}
		if count > 0 {
			fmt.Printf("%-3s: %4d leads\n", country, count)
		}
	}

	// Quality distribution
	fmt.Println("\n⭐ Quality Distribution:")
	highQuality, _ := client.Lead.Query().Where(lead.QualityScoreGTE(80)).Count(ctx)
	mediumQuality, _ := client.Lead.Query().Where(lead.QualityScoreGTE(50), lead.QualityScoreLT(80)).Count(ctx)
	lowQuality, _ := client.Lead.Query().Where(lead.QualityScoreLT(50)).Count(ctx)

	fmt.Printf("High (80-100):   %4d leads (%.1f%%)\n", highQuality, float64(highQuality)/float64(totalCount)*100)
	fmt.Printf("Medium (50-79):  %4d leads (%.1f%%)\n", mediumQuality, float64(mediumQuality)/float64(totalCount)*100)
	fmt.Printf("Low (0-49):      %4d leads (%.1f%%)\n", lowQuality, float64(lowQuality)/float64(totalCount)*100)

	// Data completeness
	fmt.Println("\n📧 Data Completeness:")
	withEmail, _ := client.Lead.Query().Where(lead.EmailNotNil()).Count(ctx)
	withPhone, _ := client.Lead.Query().Where(lead.PhoneNotNil()).Count(ctx)
	withWebsite, _ := client.Lead.Query().Where(lead.WebsiteNotNil()).Count(ctx)
	withAddress, _ := client.Lead.Query().Where(lead.AddressNotNil()).Count(ctx)

	fmt.Printf("Email:   %4d leads (%.1f%%)\n", withEmail, float64(withEmail)/float64(totalCount)*100)
	fmt.Printf("Phone:   %4d leads (%.1f%%)\n", withPhone, float64(withPhone)/float64(totalCount)*100)
	fmt.Printf("Website: %4d leads (%.1f%%)\n", withWebsite, float64(withWebsite)/float64(totalCount)*100)
	fmt.Printf("Address: %4d leads (%.1f%%)\n", withAddress, float64(withAddress)/float64(totalCount)*100)

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("✅ Seeding completed successfully!")
}
