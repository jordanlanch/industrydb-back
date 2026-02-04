package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://industrydb:localdev@localhost:5432/industrydb?sslmode=disable"
		log.Printf("DATABASE_URL not set, using default: %s", dbURL)
	}

	// Connect to database
	client, err := ent.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get admin password from environment or use default
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "changeme123"
		log.Println("⚠️  ADMIN_PASSWORD not set, using default password")
		log.Println("⚠️  Please change this password after first login!")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed hashing password: %v", err)
	}

	// Check if admin already exists
	existingAdmin, err := client.User.Query().
		Where(user.EmailEQ("admin@industrydb.io")).
		Only(ctx)

	if err == nil {
		// Admin exists, update it
		log.Printf("Admin user already exists (ID: %d), updating...", existingAdmin.ID)

		_, err = existingAdmin.Update().
			SetPasswordHash(string(hashedPassword)).
			SetRole(user.RoleSuperadmin).
			SetUpdatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			log.Fatalf("failed updating admin user: %v", err)
		}

		log.Println("✅ Admin user updated successfully")
		printAdminCredentials("admin@industrydb.io", adminPassword)
		return
	}

	// Admin doesn't exist, create it
	if !ent.IsNotFound(err) {
		log.Fatalf("error checking existing admin: %v", err)
	}

	log.Println("Creating new admin user...")

	admin, err := client.User.Create().
		SetEmail("admin@industrydb.io").
		SetName("Admin User").
		SetPasswordHash(string(hashedPassword)).
		SetRole(user.RoleSuperadmin).
		SetSubscriptionTier(user.SubscriptionTierBusiness). // Give admin business tier
		SetUsageCount(0).
		SetUsageLimit(100000). // High limit for admin
		SetEmailVerifiedAt(time.Now()). // Auto-verify admin email
		SetAcceptedTermsAt(time.Now()).
		Save(ctx)

	if err != nil {
		log.Fatalf("failed creating admin user: %v", err)
	}

	log.Printf("✅ Admin user created successfully (ID: %d)", admin.ID)
	printAdminCredentials("admin@industrydb.io", adminPassword)

	// Create additional test admin if in development
	if os.Getenv("ENVIRONMENT") == "development" {
		log.Println("\n📝 Creating additional test admin for development...")

		testAdmin, err := client.User.Create().
			SetEmail("test-admin@industrydb.io").
			SetName("Test Admin").
			SetPasswordHash(string(hashedPassword)).
			SetRole(user.RoleAdmin). // Regular admin (not superadmin)
			SetSubscriptionTier(user.SubscriptionTierPro).
			SetUsageCount(0).
			SetUsageLimit(10000).
			SetEmailVerifiedAt(time.Now()).
			SetAcceptedTermsAt(time.Now()).
			Save(ctx)

		if err != nil {
			log.Printf("❌ Failed creating test admin: %v", err)
		} else {
			log.Printf("✅ Test admin created successfully (ID: %d)", testAdmin.ID)
			printAdminCredentials("test-admin@industrydb.io", adminPassword)
		}
	}
}

func printAdminCredentials(email, password string) {
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔐  ADMIN CREDENTIALS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Email:    %s\n", email)
	fmt.Printf("Password: %s\n", password)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  IMPORTANT: Change this password after first login!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
