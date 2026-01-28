package industries

// SubNicheConfig holds metadata for a sub-niche category within an industry
type SubNicheConfig struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	OSMTag      string   `json:"osm_tag"`       // OSM tag to match (e.g., "cuisine=italian")
	OSMValues   []string `json:"osm_values"`    // Possible OSM values to match
	Description string   `json:"description"`
	SortOrder   int      `json:"sort_order"`
	Popular     bool     `json:"popular"`       // Mark trending/popular sub-niches
}

// IndustryConfig holds metadata for an industry
type IndustryConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Category          string            `json:"category"`
	Icon              string            `json:"icon"`
	OSMPrimaryTag     string            `json:"osm_primary_tag"`
	OSMAdditionalTags []string          `json:"osm_additional_tags,omitempty"`
	Description       string            `json:"description"`
	Active            bool              `json:"active"`
	SortOrder         int               `json:"sort_order"`
	HasSubNiches      bool              `json:"has_sub_niches"`       // Whether this industry has sub-niches
	SubNicheLabel     string            `json:"sub_niche_label"`      // Display label (e.g., "Cuisine Type", "Gym Type")
	SubNiches         []SubNicheConfig  `json:"sub_niches,omitempty"` // List of sub-niches
}

// CategoryInfo holds category metadata
type CategoryInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// AllIndustries returns all industry configurations
func AllIndustries() []IndustryConfig {
	return []IndustryConfig{
		// Personal Care & Beauty (Category 1)
		{
			ID:            "tattoo",
			Name:          "Tattoo Studios",
			Category:      "personal_care",
			Icon:          "🎨",
			OSMPrimaryTag: "shop=tattoo",
			Description:   "Tattoo and body art studios",
			Active:        true,
			SortOrder:     1,
			HasSubNiches:  true,
			SubNicheLabel: "Tattoo Style",
			SubNiches:     TattooSubNiches(),
		},
		{
			ID:            "beauty",
			Name:          "Beauty Salons",
			Category:      "personal_care",
			Icon:          "💅",
			OSMPrimaryTag: "shop=beauty",
			Description:   "Beauty salons and cosmetic services",
			Active:        true,
			SortOrder:     2,
			HasSubNiches:  true,
			SubNicheLabel: "Service Type",
			SubNiches:     BeautySubNiches(),
		},
		{
			ID:            "barber",
			Name:          "Barber Shops",
			Category:      "personal_care",
			Icon:          "💈",
			OSMPrimaryTag: "shop=hairdresser",
			OSMAdditionalTags: []string{
				"shop=barber",
			},
			Description: "Barbershops and hair salons",
			Active:      true,
			SortOrder:   3,
		},
		{
			ID:            "spa",
			Name:          "Spas & Wellness",
			Category:      "personal_care",
			Icon:          "🧖",
			OSMPrimaryTag: "leisure=spa",
			OSMAdditionalTags: []string{
				"amenity=spa",
			},
			Description: "Spas and wellness centers",
			Active:      true,
			SortOrder:   4,
		},
		{
			ID:            "nail_salon",
			Name:          "Nail Salons",
			Category:      "personal_care",
			Icon:          "💅",
			OSMPrimaryTag: "shop=beauty",
			OSMAdditionalTags: []string{
				"beauty=nails",
			},
			Description: "Nail salons and manicure services",
			Active:      true,
			SortOrder:   5,
		},

		// Health & Wellness (Category 2)
		{
			ID:            "gym",
			Name:          "Gyms & Fitness",
			Category:      "health_wellness",
			Icon:          "💪",
			OSMPrimaryTag: "leisure=fitness_centre",
			OSMAdditionalTags: []string{
				"leisure=sports_centre",
				"amenity=gym",
			},
			Description:   "Gyms and fitness centers",
			Active:        true,
			SortOrder:     6,
			HasSubNiches:  true,
			SubNicheLabel: "Gym Type",
			SubNiches:     GymSubNiches(),
		},
		{
			ID:            "dentist",
			Name:          "Dentists",
			Category:      "health_wellness",
			Icon:          "🦷",
			OSMPrimaryTag: "amenity=dentist",
			Description:   "Dental clinics and dentists",
			Active:        true,
			SortOrder:     7,
		},
		{
			ID:            "pharmacy",
			Name:          "Pharmacies",
			Category:      "health_wellness",
			Icon:          "💊",
			OSMPrimaryTag: "amenity=pharmacy",
			Description:   "Pharmacies and drugstores",
			Active:        true,
			SortOrder:     8,
		},
		{
			ID:            "massage",
			Name:          "Massage Therapy",
			Category:      "health_wellness",
			Icon:          "💆",
			OSMPrimaryTag: "shop=massage",
			OSMAdditionalTags: []string{
				"amenity=massage",
			},
			Description: "Massage therapy and wellness centers",
			Active:      true,
			SortOrder:   9,
		},

		// Food & Beverage (Category 3)
		{
			ID:            "restaurant",
			Name:          "Restaurants",
			Category:      "food_beverage",
			Icon:          "🍽️",
			OSMPrimaryTag: "amenity=restaurant",
			Description:   "Restaurants and dining establishments",
			Active:        true,
			SortOrder:     10,
			HasSubNiches:  true,
			SubNicheLabel: "Cuisine Type",
			SubNiches:     RestaurantSubNiches(),
		},
		{
			ID:            "cafe",
			Name:          "Cafes & Coffee Shops",
			Category:      "food_beverage",
			Icon:          "☕",
			OSMPrimaryTag: "amenity=cafe",
			Description:   "Cafes and coffee shops",
			Active:        true,
			SortOrder:     11,
		},
		{
			ID:            "bar",
			Name:          "Bars & Pubs",
			Category:      "food_beverage",
			Icon:          "🍺",
			OSMPrimaryTag: "amenity=bar",
			OSMAdditionalTags: []string{
				"amenity=pub",
			},
			Description: "Bars, pubs, and nightlife venues",
			Active:      true,
			SortOrder:   12,
		},
		{
			ID:            "bakery",
			Name:          "Bakeries",
			Category:      "food_beverage",
			Icon:          "🥖",
			OSMPrimaryTag: "shop=bakery",
			Description:   "Bakeries and pastry shops",
			Active:        true,
			SortOrder:     13,
		},

		// Automotive (Category 4)
		{
			ID:            "car_repair",
			Name:          "Car Repair",
			Category:      "automotive",
			Icon:          "🔧",
			OSMPrimaryTag: "shop=car_repair",
			Description:   "Auto repair and maintenance shops",
			Active:        true,
			SortOrder:     14,
		},
		{
			ID:            "car_wash",
			Name:          "Car Wash",
			Category:      "automotive",
			Icon:          "🚗",
			OSMPrimaryTag: "amenity=car_wash",
			Description:   "Car wash and detailing services",
			Active:        true,
			SortOrder:     15,
		},
		{
			ID:            "car_dealer",
			Name:          "Car Dealers",
			Category:      "automotive",
			Icon:          "🚙",
			OSMPrimaryTag: "shop=car",
			Description:   "Car dealerships and sales",
			Active:        true,
			SortOrder:     16,
		},

		// Retail (Category 5)
		{
			ID:            "clothing",
			Name:          "Clothing Stores",
			Category:      "retail",
			Icon:          "👕",
			OSMPrimaryTag: "shop=clothes",
			Description:   "Clothing and fashion stores",
			Active:        true,
			SortOrder:     17,
		},
		{
			ID:            "convenience",
			Name:          "Convenience Stores",
			Category:      "retail",
			Icon:          "🏪",
			OSMPrimaryTag: "shop=convenience",
			Description:   "Convenience stores and mini markets",
			Active:        true,
			SortOrder:     18,
		},

		// Professional Services (Category 6)
		{
			ID:            "lawyer",
			Name:          "Lawyers",
			Category:      "professional",
			Icon:          "⚖️",
			OSMPrimaryTag: "office=lawyer",
			Description:   "Law offices and legal services",
			Active:        true,
			SortOrder:     19,
		},
		{
			ID:            "accountant",
			Name:          "Accountants",
			Category:      "professional",
			Icon:          "📊",
			OSMPrimaryTag: "office=accountant",
			Description:   "Accounting and financial services",
			Active:        true,
			SortOrder:     20,
		},
	}
}

// AllCategories returns all category configurations
func AllCategories() []CategoryInfo {
	return []CategoryInfo{
		{
			ID:          "personal_care",
			Name:        "Personal Care & Beauty",
			Icon:        "💅",
			Description: "Beauty salons, barbershops, spas, and personal care services",
			SortOrder:   1,
		},
		{
			ID:          "health_wellness",
			Name:        "Health & Wellness",
			Icon:        "💪",
			Description: "Gyms, dentists, pharmacies, and wellness centers",
			SortOrder:   2,
		},
		{
			ID:          "food_beverage",
			Name:        "Food & Beverage",
			Icon:        "🍽️",
			Description: "Restaurants, cafes, bars, and food establishments",
			SortOrder:   3,
		},
		{
			ID:          "automotive",
			Name:        "Automotive",
			Icon:        "🔧",
			Description: "Car repair, car wash, and automotive services",
			SortOrder:   4,
		},
		{
			ID:          "retail",
			Name:        "Retail",
			Icon:        "🏪",
			Description: "Clothing stores, convenience stores, and retail shops",
			SortOrder:   5,
		},
		{
			ID:          "professional",
			Name:        "Professional Services",
			Icon:        "⚖️",
			Description: "Lawyers, accountants, and professional services",
			SortOrder:   6,
		},
	}
}

// GetIndustryByID returns an industry config by ID
func GetIndustryByID(id string) *IndustryConfig {
	for _, industry := range AllIndustries() {
		if industry.ID == id {
			return &industry
		}
	}
	return nil
}

// GetIndustriesByCategory returns all industries in a category
func GetIndustriesByCategory(category string) []IndustryConfig {
	var industries []IndustryConfig
	for _, industry := range AllIndustries() {
		if industry.Category == category {
			industries = append(industries, industry)
		}
	}
	return industries
}

// GetSubNichesByIndustry returns all sub-niches for an industry
func GetSubNichesByIndustry(industryID string) []SubNicheConfig {
	industry := GetIndustryByID(industryID)
	if industry == nil || !industry.HasSubNiches {
		return []SubNicheConfig{}
	}
	return industry.SubNiches
}

// GetSubNicheByID returns a specific sub-niche by industry and sub-niche ID
func GetSubNicheByID(industryID, subNicheID string) *SubNicheConfig {
	subNiches := GetSubNichesByIndustry(industryID)
	for _, subNiche := range subNiches {
		if subNiche.ID == subNicheID {
			return &subNiche
		}
	}
	return nil
}
