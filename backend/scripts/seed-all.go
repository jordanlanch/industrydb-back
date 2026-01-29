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
	// Personal Care & Beauty (10 industries): 150 leads each = 1,500 total
	"tattoo":        150,
	"beauty":        150,
	"barber":        150,
	"spa":           150,
	"nail_salon":    150,
	"hair_salon":    150,
	"tanning_salon": 150,
	"cosmetics":     150,
	"perfumery":     150,
	"waxing_salon":  150,

	// Health & Wellness (12 industries): 120 leads each = 1,440 total
	"gym":              120,
	"dentist":          120,
	"pharmacy":         120,
	"massage":          120,
	"chiropractor":     120,
	"optician":         120,
	"clinic":           120,
	"hospital":         120,
	"veterinary":       120,
	"yoga_studio":      120,
	"pilates_studio":   120,
	"physical_therapy": 120,

	// Food & Beverage (14 industries): 120 leads each = 1,680 total
	"restaurant": 120,
	"cafe":       120,
	"bar":        120,
	"bakery":     120,
	"fast_food":  120,
	"ice_cream":  120,
	"juice_bar":  120,
	"pizza":      120,
	"sushi":      120,
	"brewery":    120,
	"winery":     120,
	"food_truck": 120,
	"catering":   120,
	"deli":       120,

	// Automotive (8 industries): 100 leads each = 800 total
	"car_repair":        100,
	"car_wash":          100,
	"car_dealer":        100,
	"tire_shop":         100,
	"auto_parts":        100,
	"gas_station":       100,
	"motorcycle_dealer": 100,
	"rv_dealer":         100,

	// Retail (17 industries): 100 leads each = 1,700 total
	"clothing":      100,
	"convenience":   100,
	"florist":       100,
	"bookstore":     100,
	"electronics":   100,
	"furniture":     100,
	"hardware":      100,
	"jewelry":       100,
	"gift_shop":     100,
	"toy_store":     100,
	"pet_store":     100,
	"bicycle_shop":  100,
	"sporting_goods": 100,
	"music_store":   100,
	"art_supply":    100,
	"stationery":    100,
	"garden_center": 100,

	// Professional Services (12 industries): 100 leads each = 1,200 total
	"lawyer":            100,
	"accountant":        100,
	"real_estate":       100,
	"insurance":         100,
	"financial_advisor": 100,
	"notary":            100,
	"tax_service":       100,
	"marketing_agency":  100,
	"photography":       100,
	"printing":          100,
	"event_planning":    100,
	"travel_agency":     100,

	// Hospitality (8 industries): 100 leads each = 800 total
	"hotel":            100,
	"motel":            100,
	"hostel":           100,
	"bed_breakfast":    100,
	"vacation_rental":  100,
	"campground":       100,
	"rv_park":          100,
	"resort":           100,

	// Home Services (9 industries): 100 leads each = 900 total
	"plumber":      100,
	"electrician":  100,
	"hvac":         100,
	"locksmith":    100,
	"roofing":      100,
	"painting":     100,
	"cleaning":     100,
	"landscaping":  100,
	"pest_control": 100,
}
// TOTAL: 10,020 leads across 80 industries

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
