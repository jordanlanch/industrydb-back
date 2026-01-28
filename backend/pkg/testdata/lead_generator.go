package testdata

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

// LeadGeneratorConfig configures lead generation parameters
type LeadGeneratorConfig struct {
	Industry      string
	Count         int
	Country       string
	City          string
	MinQuality    int     // 0-100
	MaxQuality    int     // 0-100
	EmailChance   float64 // 0.0-1.0 (probability of having email)
	PhoneChance   float64
	WebsiteChance float64
	AddressChance float64
}

// LocationData maps countries to their major cities
var LocationData = map[string][]string{
	"US": {"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
		"Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose"},
	"GB": {"London", "Manchester", "Birmingham", "Leeds", "Glasgow",
		"Liverpool", "Newcastle", "Sheffield", "Bristol", "Edinburgh"},
	"DE": {"Berlin", "Munich", "Hamburg", "Cologne", "Frankfurt",
		"Stuttgart", "Düsseldorf", "Dortmund", "Essen", "Leipzig"},
	"ES": {"Madrid", "Barcelona", "Valencia", "Seville", "Zaragoza",
		"Málaga", "Murcia", "Palma", "Las Palmas", "Bilbao"},
	"FR": {"Paris", "Marseille", "Lyon", "Toulouse", "Nice",
		"Nantes", "Strasbourg", "Montpellier", "Bordeaux", "Lille"},
	"CA": {"Toronto", "Montreal", "Vancouver", "Calgary", "Edmonton",
		"Ottawa", "Winnipeg", "Quebec City", "Hamilton", "Kitchener"},
	"AU": {"Sydney", "Melbourne", "Brisbane", "Perth", "Adelaide",
		"Gold Coast", "Canberra", "Newcastle", "Wollongong", "Logan"},
}

// Industry-specific business name prefixes and suffixes
var businessNameParts = map[string]struct {
	Prefixes []string
	Suffixes []string
}{
	"tattoo": {
		Prefixes: []string{"Ink", "Sacred", "Dragon", "Iron", "Royal", "Black", "Electric", "Eternal", "Wild", "Golden"},
		Suffixes: []string{"Tattoo Studio", "Ink", "Body Art", "Tattoo Parlor", "Tattoo Shop", "Tattoo Co", "Tattoo Gallery"},
	},
	"beauty": {
		Prefixes: []string{"Bella", "Glamour", "Elite", "Luxe", "Divine", "Radiant", "Pure", "Elegant", "Chic", "Prima"},
		Suffixes: []string{"Beauty Salon", "Beauty Bar", "Beauty Studio", "Salon", "Beauty Lounge", "Beauty House"},
	},
	"barber": {
		Prefixes: []string{"Classic", "Gentleman's", "Royal", "Premium", "Master", "Modern", "Traditional", "Elite", "Old School", "Sharp"},
		Suffixes: []string{"Barber Shop", "Barbers", "Barbershop", "Barber Co", "Grooming", "Cuts"},
	},
	"gym": {
		Prefixes: []string{"Iron", "Peak", "Elite", "Power", "Alpha", "Titan", "Prime", "Force", "Ultimate", "Victory"},
		Suffixes: []string{"Fitness", "Gym", "Performance", "Training Center", "Athletic Club", "Fitness Studio"},
	},
	"spa": {
		Prefixes: []string{"Serenity", "Tranquil", "Zen", "Harmony", "Bliss", "Paradise", "Oasis", "Haven", "Pure", "Luxury"},
		Suffixes: []string{"Spa", "Day Spa", "Wellness Spa", "Spa & Wellness", "Relaxation Spa"},
	},
	"nail": {
		Prefixes: []string{"Perfect", "Polished", "Glamour", "Luxe", "Elegant", "Chic", "Modern", "Beauty", "Style", "Royal"},
		Suffixes: []string{"Nails", "Nail Salon", "Nail Studio", "Nail Bar", "Nail Lounge", "Nail Art"},
	},
	"dentist": {
		Prefixes: []string{"Bright", "Perfect", "Family", "Advanced", "Modern", "Premier", "Complete", "Gentle", "Elite", "Professional"},
		Suffixes: []string{"Dental", "Dentistry", "Dental Care", "Dental Clinic", "Dental Practice", "Dental Studio"},
	},
	"pharmacy": {
		Prefixes: []string{"City", "Community", "Family", "Health", "Care", "Express", "Quick", "Local", "Neighborhood", "Central"},
		Suffixes: []string{"Pharmacy", "Drugstore", "Chemist", "Apothecary", "Medications", "Health Store"},
	},
	"massage": {
		Prefixes: []string{"Healing", "Therapeutic", "Relaxation", "Wellness", "Natural", "Holistic", "Deep", "Gentle", "Professional", "Expert"},
		Suffixes: []string{"Massage", "Massage Therapy", "Massage Studio", "Bodywork", "Massage Center", "Therapy"},
	},
	"restaurant": {
		Prefixes: []string{"The", "Golden", "Silver", "Blue", "Red", "Green", "Royal", "Grand", "Casa", "Villa"},
		Suffixes: []string{"Restaurant", "Bistro", "Dining", "Kitchen", "Grill", "Eatery", "Table"},
	},
	"cafe": {
		Prefixes: []string{"Cozy", "Corner", "Daily", "Morning", "Central", "Urban", "Local", "Artisan", "Vintage", "Modern"},
		Suffixes: []string{"Cafe", "Coffee Shop", "Coffee House", "Coffeehouse", "Coffee Bar", "Coffee & Tea"},
	},
	"bar": {
		Prefixes: []string{"The", "Old", "New", "Red", "Blue", "Black", "White", "Golden", "Silver", "Royal"},
		Suffixes: []string{"Bar", "Tavern", "Pub", "Lounge", "Sports Bar", "Wine Bar", "Cocktail Bar"},
	},
	"bakery": {
		Prefixes: []string{"Fresh", "Golden", "Artisan", "Daily", "Classic", "French", "European", "Local", "Sweet", "Sunrise"},
		Suffixes: []string{"Bakery", "Bakers", "Bread", "Pastry Shop", "Patisserie", "Bake House"},
	},
	"car_repair": {
		Prefixes: []string{"Expert", "Professional", "Quality", "Fast", "Complete", "Master", "Precision", "Advanced", "Total", "Premier"},
		Suffixes: []string{"Auto Repair", "Auto Service", "Car Repair", "Automotive", "Auto Care", "Garage"},
	},
	"car_wash": {
		Prefixes: []string{"Express", "Quick", "Clean", "Shine", "Sparkle", "Ultimate", "Premium", "Professional", "Super", "Speedy"},
		Suffixes: []string{"Car Wash", "Auto Wash", "Car Care", "Wash & Detail", "Detailing", "Auto Spa"},
	},
	"car_dealer": {
		Prefixes: []string{"Premier", "Elite", "Quality", "Prestige", "Luxury", "Metro", "City", "Central", "Best", "Top"},
		Suffixes: []string{"Auto Sales", "Motors", "Automotive", "Car Sales", "Auto Group", "Dealership"},
	},
	"clothing": {
		Prefixes: []string{"Style", "Fashion", "Trend", "Chic", "Modern", "Urban", "Classic", "Elite", "Boutique", "Designer"},
		Suffixes: []string{"Boutique", "Fashion", "Clothing", "Apparel", "Style Shop", "Wardrobe"},
	},
	"convenience": {
		Prefixes: []string{"Quick", "Express", "24/7", "Corner", "Neighborhood", "Local", "City", "Central", "Fast", "Easy"},
		Suffixes: []string{"Convenience Store", "Market", "Shop", "Mini Mart", "Quick Stop", "Store"},
	},
	"lawyer": {
		Prefixes: []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Miller", "Davis", "Wilson", "Moore", "Taylor"},
		Suffixes: []string{"Law Firm", "Legal Services", "Attorneys", "Law Office", "Legal Group", "Associates"},
	},
	"accountant": {
		Prefixes: []string{"Professional", "Expert", "Premier", "Elite", "Quality", "Precision", "Complete", "Total", "Advanced", "Modern"},
		Suffixes: []string{"Accounting", "CPA", "Tax Services", "Accounting Firm", "Financial Services", "Bookkeeping"},
	},
}

// GenerateBusinessName creates industry-specific realistic business names
func GenerateBusinessName(industry string) string {
	parts, ok := businessNameParts[industry]
	if !ok {
		// Fallback for unknown industries
		return fmt.Sprintf("%s %s", gofakeit.Company(), gofakeit.BuzzWord())
	}

	prefix := parts.Prefixes[rand.Intn(len(parts.Prefixes))]
	suffix := parts.Suffixes[rand.Intn(len(parts.Suffixes))]

	return fmt.Sprintf("%s %s", prefix, suffix)
}

// GenerateLead creates a single lead with realistic data
func GenerateLead(config LeadGeneratorConfig) *ent.LeadCreate {
	// Generate quality score with normal distribution around config range
	quality := config.MinQuality + rand.Intn(config.MaxQuality-config.MinQuality+1)

	// Business name
	businessName := GenerateBusinessName(config.Industry)

	// Optional fields based on chances
	var email, phone, website, address, postalCode *string

	if rand.Float64() < config.EmailChance {
		// Generate email based on business name
		emailDomain := strings.ToLower(strings.ReplaceAll(businessName, " ", ""))
		emailDomain = strings.ReplaceAll(emailDomain, "'", "")
		if len(emailDomain) > 20 {
			emailDomain = emailDomain[:20]
		}
		emailVal := fmt.Sprintf("contact@%s.com", emailDomain)
		email = &emailVal
	}

	if rand.Float64() < config.PhoneChance {
		phoneVal := gofakeit.Phone()
		phone = &phoneVal
	}

	if rand.Float64() < config.WebsiteChance {
		websiteDomain := strings.ToLower(strings.ReplaceAll(businessName, " ", ""))
		websiteDomain = strings.ReplaceAll(websiteDomain, "'", "")
		if len(websiteDomain) > 20 {
			websiteDomain = websiteDomain[:20]
		}
		websiteVal := fmt.Sprintf("https://www.%s.com", websiteDomain)
		website = &websiteVal
	}

	if rand.Float64() < config.AddressChance {
		addressVal := gofakeit.Street()
		address = &addressVal
		postalVal := gofakeit.Zip()
		postalCode = &postalVal
	}

	// Calculate verification status based on quality
	verified := quality >= 80

	leadCreate := &ent.LeadCreate{}
	leadCreate.
		SetName(businessName).
		SetIndustry(lead.Industry(config.Industry)).
		SetCountry(config.Country).
		SetCity(config.City).
		SetQualityScore(quality).
		SetVerified(verified).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	// Set optional fields
	if email != nil {
		leadCreate.SetEmail(*email)
	}
	if phone != nil {
		leadCreate.SetPhone(*phone)
	}
	if website != nil {
		leadCreate.SetWebsite(*website)
	}
	if address != nil {
		leadCreate.SetAddress(*address)
	}
	if postalCode != nil {
		leadCreate.SetPostalCode(*postalCode)
	}

	return leadCreate
}

// GenerateLeads creates multiple leads with the given config
func GenerateLeads(config LeadGeneratorConfig) []*ent.LeadCreate {
	leads := make([]*ent.LeadCreate, config.Count)
	for i := 0; i < config.Count; i++ {
		leads[i] = GenerateLead(config)
	}
	return leads
}

// GenerateLeadsForIndustry generates leads for a specific industry with default settings
func GenerateLeadsForIndustry(industry string, count int) []*ent.LeadCreate {
	// Default quality distribution: normal distribution around 70
	minQuality := 40
	maxQuality := 95

	// Default completeness: 60% of leads have most fields
	emailChance := 0.6
	phoneChance := 0.7
	websiteChance := 0.5
	addressChance := 0.8

	// Pick random country and city
	countries := []string{"US", "GB", "DE", "ES", "FR", "CA", "AU"}
	country := countries[rand.Intn(len(countries))]
	cities := LocationData[country]
	city := cities[rand.Intn(len(cities))]

	config := LeadGeneratorConfig{
		Industry:      industry,
		Count:         count,
		Country:       country,
		City:          city,
		MinQuality:    minQuality,
		MaxQuality:    maxQuality,
		EmailChance:   emailChance,
		PhoneChance:   phoneChance,
		WebsiteChance: websiteChance,
		AddressChance: addressChance,
	}

	return GenerateLeads(config)
}

// GenerateLeadsWithDistribution generates leads with quality/completeness distribution
func GenerateLeadsWithDistribution(industry string, count int) []*ent.LeadCreate {
	leads := make([]*ent.LeadCreate, 0, count)

	// Quality distribution: 30% high (80-100), 50% medium (50-79), 20% low (0-49)
	highCount := int(float64(count) * 0.3)
	mediumCount := int(float64(count) * 0.5)
	_ = count - highCount - mediumCount // lowCount

	// Completeness distribution: 40% complete, 35% mostly complete, 25% incomplete
	completeCount := int(float64(count) * 0.4)
	mostlyCompleteCount := int(float64(count) * 0.35)
	_ = count - completeCount - mostlyCompleteCount // incompleteCount

	// Generate high quality leads (mostly complete)
	for i := 0; i < highCount && i < completeCount; i++ {
		country := pickRandomCountry()
		city := pickRandomCity(country)
		config := LeadGeneratorConfig{
			Industry:      industry,
			Count:         1,
			Country:       country,
			City:          city,
			MinQuality:    80,
			MaxQuality:    100,
			EmailChance:   0.9,
			PhoneChance:   0.95,
			WebsiteChance: 0.85,
			AddressChance: 0.95,
		}
		leads = append(leads, GenerateLead(config))
	}

	// Generate medium quality leads (mostly complete)
	for i := 0; i < mediumCount && len(leads) < count; i++ {
		country := pickRandomCountry()
		city := pickRandomCity(country)
		completeness := "complete"
		if i >= mostlyCompleteCount {
			completeness = "incomplete"
		}

		var emailChance, phoneChance, websiteChance, addressChance float64
		if completeness == "complete" {
			emailChance, phoneChance, websiteChance, addressChance = 0.8, 0.85, 0.7, 0.9
		} else if completeness == "mostly" {
			emailChance, phoneChance, websiteChance, addressChance = 0.6, 0.7, 0.5, 0.75
		} else {
			emailChance, phoneChance, websiteChance, addressChance = 0.3, 0.4, 0.2, 0.5
		}

		config := LeadGeneratorConfig{
			Industry:      industry,
			Count:         1,
			Country:       country,
			City:          city,
			MinQuality:    50,
			MaxQuality:    79,
			EmailChance:   emailChance,
			PhoneChance:   phoneChance,
			WebsiteChance: websiteChance,
			AddressChance: addressChance,
		}
		leads = append(leads, GenerateLead(config))
	}

	// Generate low quality leads (incomplete)
	for len(leads) < count {
		country := pickRandomCountry()
		city := pickRandomCity(country)
		config := LeadGeneratorConfig{
			Industry:      industry,
			Count:         1,
			Country:       country,
			City:          city,
			MinQuality:    0,
			MaxQuality:    49,
			EmailChance:   0.2,
			PhoneChance:   0.3,
			WebsiteChance: 0.1,
			AddressChance: 0.4,
		}
		leads = append(leads, GenerateLead(config))
	}

	return leads
}

func pickRandomCountry() string {
	countries := []string{"US", "GB", "DE", "ES", "FR", "CA", "AU"}
	return countries[rand.Intn(len(countries))]
}

func pickRandomCity(country string) string {
	cities := LocationData[country]
	return cities[rand.Intn(len(cities))]
}

// BulkInsertLeads inserts leads in batches for performance
func BulkInsertLeads(ctx context.Context, client *ent.Client, leads []*ent.LeadCreate, batchSize int) error {
	for i := 0; i < len(leads); i += batchSize {
		end := i + batchSize
		if end > len(leads) {
			end = len(leads)
		}

		batch := leads[i:end]
		if err := client.Lead.CreateBulk(batch...).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end, err)
		}
	}
	return nil
}
