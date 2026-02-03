package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	_ "github.com/lib/pq"
)

func seedAnalytics() {
	// Use hardcoded database URL for Docker environment
	dbURL := "postgres://industrydb:localdev@db:5432/industrydb?sslmode=disable"
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Open database connection
	client, err := ent.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get all users
	users, err := client.User.Query().All(ctx)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}

	if len(users) == 0 {
		log.Fatal("No users found. Please create at least one user first.")
	}

	fmt.Printf("Found %d users\n", len(users))

	// Seed random data for analytics testing
	rand.Seed(time.Now().UnixNano())

	totalLogs := 0
	now := time.Now()

	for _, user := range users {
		// Generate logs for the last 90 days
		daysBack := 90
		logsForUser := 0

		for i := 0; i < daysBack; i++ {
			date := now.AddDate(0, 0, -i)

			// Random number of searches per day (0-10)
			searchCount := rand.Intn(11)
			for j := 0; j < searchCount; j++ {
				// Random time within the day
				hour := rand.Intn(24)
				minute := rand.Intn(60)
				timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())

				industries := []string{"tattoo", "beauty", "barber", "gym", "restaurant"}
				countries := []string{"US", "GB", "ES", "DE", "FR"}

				metadata := map[string]interface{}{
					"industry": industries[rand.Intn(len(industries))],
					"country":  countries[rand.Intn(len(countries))],
				}

				_, err := client.UsageLog.Create().
					SetUserID(user.ID).
					SetAction("search").
					SetCount(1).
					SetMetadata(metadata).
					SetCreatedAt(timestamp).
					Save(ctx)

				if err != nil {
					log.Printf("Failed to create search log: %v", err)
					continue
				}

				logsForUser++
				totalLogs++
			}

			// Random exports (less frequent - 0-3 per day)
			exportCount := rand.Intn(4)
			for j := 0; j < exportCount; j++ {
				hour := rand.Intn(24)
				minute := rand.Intn(60)
				timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())

				formats := []string{"csv", "excel"}
				leadCount := rand.Intn(50) + 10 // 10-60 leads per export

				metadata := map[string]interface{}{
					"format": formats[rand.Intn(len(formats))],
					"leads":  leadCount,
				}

				_, err := client.UsageLog.Create().
					SetUserID(user.ID).
					SetAction("export").
					SetCount(leadCount).
					SetMetadata(metadata).
					SetCreatedAt(timestamp).
					Save(ctx)

				if err != nil {
					log.Printf("Failed to create export log: %v", err)
					continue
				}

				logsForUser++
				totalLogs++
			}

			// Random API calls (occasional - 0-2 per day)
			if rand.Float32() < 0.3 { // 30% chance
				apiCallCount := rand.Intn(3)
				for j := 0; j < apiCallCount; j++ {
					hour := rand.Intn(24)
					minute := rand.Intn(60)
					timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())

					endpoints := []string{"/api/v1/leads", "/api/v1/leads/search", "/api/v1/exports"}

					metadata := map[string]interface{}{
						"endpoint": endpoints[rand.Intn(len(endpoints))],
						"method":   "GET",
					}

					_, err := client.UsageLog.Create().
						SetUserID(user.ID).
						SetAction("api_call").
						SetCount(1).
						SetMetadata(metadata).
						SetCreatedAt(timestamp).
						Save(ctx)

					if err != nil {
						log.Printf("Failed to create API call log: %v", err)
						continue
					}

					logsForUser++
					totalLogs++
				}
			}
		}

		fmt.Printf("Created %d logs for user %s (ID: %d)\n", logsForUser, user.Email, user.ID)
	}

	fmt.Printf("\n✅ Successfully created %d total usage logs\n", totalLogs)
	fmt.Printf("📊 Data covers the last 90 days\n")
	fmt.Printf("🔍 Actions: search, export, api_call\n")
}

func main() {
	seedAnalytics()
}
