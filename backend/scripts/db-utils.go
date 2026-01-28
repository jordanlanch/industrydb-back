package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

type DBStats struct {
	TotalLeads       int            `json:"total_leads"`
	IndustriesCount  int            `json:"industries_count"`
	CountriesCount   int            `json:"countries_count"`
	Industries       map[string]int `json:"industries"`
	Countries        map[string]int `json:"countries"`
	QualityDistribution map[string]int `json:"quality_distribution"`
	VerificationStatus  map[string]int `json:"verification_status"`
}

type ExportData struct {
	ExportedAt time.Time    `json:"exported_at"`
	TotalLeads int          `json:"total_leads"`
	Leads      []ExportLead `json:"leads"`
}

type ExportLead struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Industry     string    `json:"industry"`
	Country      string    `json:"country"`
	City         string    `json:"city"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Website      string    `json:"website,omitempty"`
	Address      string    `json:"address,omitempty"`
	PostalCode   string    `json:"postal_code,omitempty"`
	QualityScore int       `json:"quality_score"`
	Verified     bool      `json:"verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func main() {
	action := flag.String("action", "stats", "Action to perform: stats, reset, clean-test, export, import")
	output := flag.String("output", "backup.json", "Output file for export action")
	input := flag.String("input", "backup.json", "Input file for import action")
	flag.Parse()

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

	switch *action {
	case "stats":
		showStats(ctx, client)
	case "reset":
		resetDatabase(ctx, client)
	case "clean-test":
		cleanTestData(ctx, client)
	case "export":
		exportDatabase(ctx, client, *output)
	case "import":
		importDatabase(ctx, client, *input)
	default:
		log.Fatalf("Unknown action: %s. Available actions: stats, reset, clean-test, export, import", *action)
	}
}

func showStats(ctx context.Context, client *ent.Client) {
	fmt.Println("📊 DATABASE STATISTICS")
	fmt.Println("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	// Total leads
	totalLeads, err := client.Lead.Query().Count(ctx)
	if err != nil {
		log.Fatalf("Failed to count leads: %v", err)
	}
	fmt.Printf("\n📈 Total Leads: %d\n", totalLeads)

	// Industries
	fmt.Println("\n🏭 Leads by Industry:")
	industries := []string{"tattoo", "beauty", "barber", "spa", "nail", "dentist", "pharmacy", "massage", "gym",
		"restaurant", "cafe", "bar", "bakery", "car_repair", "car_wash", "car_dealer", "clothing", "convenience", "lawyer", "accountant"}

	industriesWithData := 0
	for _, industry := range industries {
		count, err := client.Lead.Query().Where(lead.IndustryEQ(lead.Industry(industry))).Count(ctx)
		if err != nil {
			log.Printf("Failed to count %s leads: %v", industry, err)
			continue
		}
		if count > 0 {
			fmt.Printf("  %-15s: %4d leads\n", industry, count)
			industriesWithData++
		}
	}
	fmt.Printf("\nIndustries with data: %d/%d\n", industriesWithData, len(industries))

	// Countries
	fmt.Println("\n🌍 Leads by Country:")
	countries := []string{"US", "GB", "DE", "ES", "FR", "CA", "AU"}
	countriesWithData := 0
	for _, country := range countries {
		count, err := client.Lead.Query().Where(lead.CountryEQ(country)).Count(ctx)
		if err != nil {
			log.Printf("Failed to count %s leads: %v", country, err)
			continue
		}
		if count > 0 {
			fmt.Printf("  %s: %4d leads\n", country, count)
			countriesWithData++
		}
	}
	fmt.Printf("\nCountries with data: %d\n", countriesWithData)

	// Quality distribution
	fmt.Println("\n⭐ Quality Distribution:")
	highQuality, _ := client.Lead.Query().Where(lead.QualityScoreGTE(80)).Count(ctx)
	mediumQuality, _ := client.Lead.Query().Where(lead.QualityScoreGTE(50), lead.QualityScoreLT(80)).Count(ctx)
	lowQuality, _ := client.Lead.Query().Where(lead.QualityScoreLT(50)).Count(ctx)

	if totalLeads > 0 {
		fmt.Printf("  High (80-100):   %4d leads (%.1f%%)\n", highQuality, float64(highQuality)/float64(totalLeads)*100)
		fmt.Printf("  Medium (50-79):  %4d leads (%.1f%%)\n", mediumQuality, float64(mediumQuality)/float64(totalLeads)*100)
		fmt.Printf("  Low (0-49):      %4d leads (%.1f%%)\n", lowQuality, float64(lowQuality)/float64(totalLeads)*100)
	}

	// Verification status
	fmt.Println("\n✓ Verification Status:")
	verified, _ := client.Lead.Query().Where(lead.VerifiedEQ(true)).Count(ctx)
	unverified, _ := client.Lead.Query().Where(lead.VerifiedEQ(false)).Count(ctx)

	if totalLeads > 0 {
		fmt.Printf("  Verified:   %4d leads (%.1f%%)\n", verified, float64(verified)/float64(totalLeads)*100)
		fmt.Printf("  Unverified: %4d leads (%.1f%%)\n", unverified, float64(unverified)/float64(totalLeads)*100)
	}

	// Data completeness
	fmt.Println("\n📧 Data Completeness:")
	withEmail, _ := client.Lead.Query().Where(lead.EmailNotNil(), lead.EmailNEQ("")).Count(ctx)
	withPhone, _ := client.Lead.Query().Where(lead.PhoneNotNil(), lead.PhoneNEQ("")).Count(ctx)
	withWebsite, _ := client.Lead.Query().Where(lead.WebsiteNotNil(), lead.WebsiteNEQ("")).Count(ctx)
	withAddress, _ := client.Lead.Query().Where(lead.AddressNotNil(), lead.AddressNEQ("")).Count(ctx)

	if totalLeads > 0 {
		fmt.Printf("  Email:   %4d leads (%.1f%%)\n", withEmail, float64(withEmail)/float64(totalLeads)*100)
		fmt.Printf("  Phone:   %4d leads (%.1f%%)\n", withPhone, float64(withPhone)/float64(totalLeads)*100)
		fmt.Printf("  Website: %4d leads (%.1f%%)\n", withWebsite, float64(withWebsite)/float64(totalLeads)*100)
		fmt.Printf("  Address: %4d leads (%.1f%%)\n", withAddress, float64(withAddress)/float64(totalLeads)*100)
	}

	fmt.Println("\n" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")
}

func resetDatabase(ctx context.Context, client *ent.Client) {
	fmt.Println("⚠️  WARNING: This will delete ALL data from the database!")
	fmt.Print("Type 'yes' to confirm: ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("❌ Reset cancelled")
		return
	}

	fmt.Println("\n🗑️  Deleting all leads...")
	deleted, err := client.Lead.Delete().Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to delete leads: %v", err)
	}

	fmt.Printf("✅ Deleted %d leads\n", deleted)
	fmt.Println("✅ Database reset completed")
}

func cleanTestData(ctx context.Context, client *ent.Client) {
	fmt.Println("🧹 Cleaning test data...")

	// Delete leads with "test" in their name
	deleted, err := client.Lead.Delete().Where(lead.NameContains("test")).Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to clean test data: %v", err)
	}

	fmt.Printf("✅ Deleted %d test leads\n", deleted)
}

func exportDatabase(ctx context.Context, client *ent.Client, outputFile string) {
	fmt.Printf("📤 Exporting database to %s...\n", outputFile)

	// Query all leads
	leads, err := client.Lead.Query().All(ctx)
	if err != nil {
		log.Fatalf("Failed to query leads: %v", err)
	}

	// Convert to export format
	exportLeads := make([]ExportLead, len(leads))
	for i, l := range leads {
		exportLeads[i] = ExportLead{
			ID:           l.ID,
			Name:         l.Name,
			Industry:     string(l.Industry),
			Country:      l.Country,
			City:         l.City,
			Email:        l.Email,
			Phone:        l.Phone,
			Website:      l.Website,
			Address:      l.Address,
			PostalCode:   l.PostalCode,
			QualityScore: l.QualityScore,
			Verified:     l.Verified,
			CreatedAt:    l.CreatedAt,
			UpdatedAt:    l.UpdatedAt,
		}
	}

	exportData := ExportData{
		ExportedAt: time.Now(),
		TotalLeads: len(leads),
		Leads:      exportLeads,
	}

	// Write to file
	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		log.Fatalf("Failed to encode data: %v", err)
	}

	fmt.Printf("✅ Exported %d leads to %s\n", len(leads), outputFile)
}

func importDatabase(ctx context.Context, client *ent.Client, inputFile string) {
	fmt.Printf("📥 Importing database from %s...\n", inputFile)

	// Read file
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	var exportData ExportData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&exportData); err != nil {
		log.Fatalf("Failed to decode data: %v", err)
	}

	fmt.Printf("Found %d leads in backup (exported at %s)\n", exportData.TotalLeads, exportData.ExportedAt.Format(time.RFC3339))

	// Import leads
	imported := 0
	for _, l := range exportData.Leads {
		leadCreate := client.Lead.Create().
			SetName(l.Name).
			SetIndustry(lead.Industry(l.Industry)).
			SetCountry(l.Country).
			SetCity(l.City).
			SetQualityScore(l.QualityScore).
			SetVerified(l.Verified).
			SetCreatedAt(l.CreatedAt).
			SetUpdatedAt(l.UpdatedAt)

		if l.Email != "" {
			leadCreate.SetEmail(l.Email)
		}
		if l.Phone != "" {
			leadCreate.SetPhone(l.Phone)
		}
		if l.Website != "" {
			leadCreate.SetWebsite(l.Website)
		}
		if l.Address != "" {
			leadCreate.SetAddress(l.Address)
		}
		if l.PostalCode != "" {
			leadCreate.SetPostalCode(l.PostalCode)
		}

		if _, err := leadCreate.Save(ctx); err != nil {
			log.Printf("Warning: Failed to import lead %d: %v", l.ID, err)
			continue
		}
		imported++
	}

	fmt.Printf("✅ Imported %d/%d leads\n", imported, len(exportData.Leads))
}
