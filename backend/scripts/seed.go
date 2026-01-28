package main

import (
	"context"
	"log"
	"os"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://industrydb:localdev@localhost:5433/industrydb?sslmode=disable"
	}

	// Connect to database
	client, err := ent.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	log.Println("🌱 Seeding database with sample leads...")

	// Sample tattoo studios
	tattooStudios := []struct {
		name    string
		city    string
		country string
		phone   string
		email   string
	}{
		{"Ink Masters Studio", "New York", "US", "+1-212-555-0101", "info@inkmasters.com"},
		{"Dragon Tattoo", "Los Angeles", "US", "+1-323-555-0102", "contact@dragontattoo.com"},
		{"Sacred Ink", "Chicago", "US", "+1-312-555-0103", "hello@sacredink.com"},
		{"The Tattoo Parlor", "Miami", "US", "+1-305-555-0104", "info@tattooparlor.com"},
		{"Black Rose Tattoos", "Seattle", "US", "+1-206-555-0105", "bookings@blackrose.com"},
	}

	for _, studio := range tattooStudios {
		_, err := client.Lead.Create().
			SetName(studio.name).
			SetIndustry(lead.IndustryTattoo).
			SetCountry(studio.country).
			SetCity(studio.city).
			SetPhone(studio.phone).
			SetEmail(studio.email).
			SetVerified(true).
			SetQualityScore(85).
			Save(ctx)

		if err != nil {
			log.Printf("Failed to create %s: %v", studio.name, err)
		} else {
			log.Printf("✅ Created: %s", studio.name)
		}
	}

	// Sample beauty salons
	beautySalons := []struct {
		name    string
		city    string
		country string
		phone   string
		email   string
	}{
		{"Glamour Beauty", "New York", "US", "+1-212-555-0201", "info@glamourbeauty.com"},
		{"Bella Salon", "Los Angeles", "US", "+1-323-555-0202", "contact@bellasalon.com"},
		{"Elegance Spa", "Chicago", "US", "+1-312-555-0203", "hello@elegancespa.com"},
		{"The Beauty Bar", "Miami", "US", "+1-305-555-0204", "bookings@beautybar.com"},
		{"Radiance Salon", "Seattle", "US", "+1-206-555-0205", "info@radiancesalon.com"},
	}

	for _, salon := range beautySalons {
		_, err := client.Lead.Create().
			SetName(salon.name).
			SetIndustry(lead.IndustryBeauty).
			SetCountry(salon.country).
			SetCity(salon.city).
			SetPhone(salon.phone).
			SetEmail(salon.email).
			SetVerified(true).
			SetQualityScore(90).
			Save(ctx)

		if err != nil {
			log.Printf("Failed to create %s: %v", salon.name, err)
		} else {
			log.Printf("✅ Created: %s", salon.name)
		}
	}

	// Sample gyms
	gyms := []struct {
		name    string
		city    string
		country string
		phone   string
	}{
		{"Iron Fitness", "New York", "US", "+1-212-555-0301"},
		{"Power Gym", "Los Angeles", "US", "+1-323-555-0302"},
		{"Fit Zone", "Chicago", "US", "+1-312-555-0303"},
		{"Strong Bodies Gym", "Miami", "US", "+1-305-555-0304"},
		{"Peak Performance", "Seattle", "US", "+1-206-555-0305"},
	}

	for _, gym := range gyms {
		_, err := client.Lead.Create().
			SetName(gym.name).
			SetIndustry(lead.IndustryGym).
			SetCountry(gym.country).
			SetCity(gym.city).
			SetPhone(gym.phone).
			SetVerified(false).
			SetQualityScore(70).
			Save(ctx)

		if err != nil {
			log.Printf("Failed to create %s: %v", gym.name, err)
		} else {
			log.Printf("✅ Created: %s", gym.name)
		}
	}

	// Count total leads
	total, err := client.Lead.Query().Count(ctx)
	if err != nil {
		log.Fatalf("Failed to count leads: %v", err)
	}

	log.Printf("\n🎉 Seeding complete! Total leads in database: %d\n", total)
}
