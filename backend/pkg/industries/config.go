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

		// Personal Care & Beauty - Additional (5 new)
		{
			ID:            "hair_salon",
			Name:          "Hair Salons",
			Category:      "personal_care",
			Icon:          "💇",
			OSMPrimaryTag: "shop=hairdresser",
			Description:   "Hair salons and hairdressing services",
			Active:        true,
			SortOrder:     21,
		},
		{
			ID:            "tanning_salon",
			Name:          "Tanning Salons",
			Category:      "personal_care",
			Icon:          "☀️",
			OSMPrimaryTag: "shop=tanning_salon",
			Description:   "Tanning salons and sunbathing services",
			Active:        true,
			SortOrder:     22,
		},
		{
			ID:            "cosmetics",
			Name:          "Cosmetics Stores",
			Category:      "personal_care",
			Icon:          "💄",
			OSMPrimaryTag: "shop=cosmetics",
			Description:   "Cosmetics and beauty product stores",
			Active:        true,
			SortOrder:     23,
		},
		{
			ID:            "perfumery",
			Name:          "Perfumeries",
			Category:      "personal_care",
			Icon:          "🌸",
			OSMPrimaryTag: "shop=perfumery",
			Description:   "Perfume and fragrance stores",
			Active:        true,
			SortOrder:     24,
		},
		{
			ID:            "waxing_salon",
			Name:          "Waxing Salons",
			Category:      "personal_care",
			Icon:          "🧴",
			OSMPrimaryTag: "shop=beauty",
			OSMAdditionalTags: []string{
				"beauty=waxing",
			},
			Description: "Waxing and hair removal services",
			Active:      true,
			SortOrder:   25,
		},

		// Health & Wellness - Additional (8 new)
		{
			ID:            "chiropractor",
			Name:          "Chiropractors",
			Category:      "health_wellness",
			Icon:          "🦴",
			OSMPrimaryTag: "amenity=chiropractor",
			Description:   "Chiropractic care and spinal adjustment services",
			Active:        true,
			SortOrder:     26,
		},
		{
			ID:            "optician",
			Name:          "Opticians",
			Category:      "health_wellness",
			Icon:          "👓",
			OSMPrimaryTag: "shop=optician",
			Description:   "Opticians and eyewear stores",
			Active:        true,
			SortOrder:     27,
		},
		{
			ID:            "clinic",
			Name:          "Medical Clinics",
			Category:      "health_wellness",
			Icon:          "🏥",
			OSMPrimaryTag: "amenity=clinic",
			Description:   "Medical clinics and healthcare centers",
			Active:        true,
			SortOrder:     28,
		},
		{
			ID:            "hospital",
			Name:          "Hospitals",
			Category:      "health_wellness",
			Icon:          "🏥",
			OSMPrimaryTag: "amenity=hospital",
			Description:   "Hospitals and emergency care facilities",
			Active:        true,
			SortOrder:     29,
		},
		{
			ID:            "veterinary",
			Name:          "Veterinarians",
			Category:      "health_wellness",
			Icon:          "🐾",
			OSMPrimaryTag: "amenity=veterinary",
			Description:   "Veterinary clinics and animal hospitals",
			Active:        true,
			SortOrder:     30,
		},
		{
			ID:            "yoga_studio",
			Name:          "Yoga Studios",
			Category:      "health_wellness",
			Icon:          "🧘",
			OSMPrimaryTag: "leisure=yoga",
			OSMAdditionalTags: []string{
				"sport=yoga",
			},
			Description: "Yoga studios and meditation centers",
			Active:      true,
			SortOrder:   31,
		},
		{
			ID:            "pilates_studio",
			Name:          "Pilates Studios",
			Category:      "health_wellness",
			Icon:          "🤸",
			OSMPrimaryTag: "sport=pilates",
			Description:   "Pilates studios and core fitness centers",
			Active:        true,
			SortOrder:     32,
		},
		{
			ID:            "physical_therapy",
			Name:          "Physical Therapy",
			Category:      "health_wellness",
			Icon:          "💪",
			OSMPrimaryTag: "healthcare=physiotherapist",
			OSMAdditionalTags: []string{
				"amenity=physiotherapist",
			},
			Description: "Physical therapy and rehabilitation services",
			Active:      true,
			SortOrder:   33,
		},

		// Food & Beverage - Additional (10 new)
		{
			ID:            "fast_food",
			Name:          "Fast Food",
			Category:      "food_beverage",
			Icon:          "🍔",
			OSMPrimaryTag: "amenity=fast_food",
			Description:   "Fast food restaurants and quick service",
			Active:        true,
			SortOrder:     34,
		},
		{
			ID:            "ice_cream",
			Name:          "Ice Cream Shops",
			Category:      "food_beverage",
			Icon:          "🍦",
			OSMPrimaryTag: "amenity=ice_cream",
			Description:   "Ice cream parlors and frozen dessert shops",
			Active:        true,
			SortOrder:     35,
		},
		{
			ID:            "juice_bar",
			Name:          "Juice Bars",
			Category:      "food_beverage",
			Icon:          "🥤",
			OSMPrimaryTag: "amenity=cafe",
			OSMAdditionalTags: []string{
				"cuisine=juice",
			},
			Description: "Juice bars and smoothie shops",
			Active:      true,
			SortOrder:   36,
		},
		{
			ID:            "pizza",
			Name:          "Pizzerias",
			Category:      "food_beverage",
			Icon:          "🍕",
			OSMPrimaryTag: "amenity=restaurant",
			OSMAdditionalTags: []string{
				"cuisine=pizza",
			},
			Description: "Pizza restaurants and pizzerias",
			Active:      true,
			SortOrder:   37,
		},
		{
			ID:            "sushi",
			Name:          "Sushi Restaurants",
			Category:      "food_beverage",
			Icon:          "🍣",
			OSMPrimaryTag: "amenity=restaurant",
			OSMAdditionalTags: []string{
				"cuisine=sushi",
			},
			Description: "Sushi restaurants and Japanese cuisine",
			Active:      true,
			SortOrder:   38,
		},
		{
			ID:            "brewery",
			Name:          "Breweries",
			Category:      "food_beverage",
			Icon:          "🍺",
			OSMPrimaryTag: "craft=brewery",
			OSMAdditionalTags: []string{
				"amenity=brewery",
			},
			Description: "Craft breweries and beer production",
			Active:      true,
			SortOrder:   39,
		},
		{
			ID:            "winery",
			Name:          "Wineries",
			Category:      "food_beverage",
			Icon:          "🍷",
			OSMPrimaryTag: "craft=winery",
			OSMAdditionalTags: []string{
				"amenity=winery",
			},
			Description: "Wineries and wine production",
			Active:      true,
			SortOrder:   40,
		},
		{
			ID:            "food_truck",
			Name:          "Food Trucks",
			Category:      "food_beverage",
			Icon:          "🚚",
			OSMPrimaryTag: "amenity=fast_food",
			Description:   "Mobile food trucks and street food",
			Active:        true,
			SortOrder:     41,
		},
		{
			ID:            "catering",
			Name:          "Catering Services",
			Category:      "food_beverage",
			Icon:          "🍽️",
			OSMPrimaryTag: "office=catering",
			Description:   "Catering and event food services",
			Active:        true,
			SortOrder:     42,
		},
		{
			ID:            "deli",
			Name:          "Delicatessens",
			Category:      "food_beverage",
			Icon:          "🥪",
			OSMPrimaryTag: "shop=deli",
			Description:   "Delicatessens and specialty food shops",
			Active:        true,
			SortOrder:     43,
		},

		// Automotive - Additional (5 new)
		{
			ID:            "tire_shop",
			Name:          "Tire Shops",
			Category:      "automotive",
			Icon:          "🛞",
			OSMPrimaryTag: "shop=tyres",
			Description:   "Tire shops and wheel services",
			Active:        true,
			SortOrder:     44,
		},
		{
			ID:            "auto_parts",
			Name:          "Auto Parts Stores",
			Category:      "automotive",
			Icon:          "🔧",
			OSMPrimaryTag: "shop=car_parts",
			Description:   "Auto parts and accessories stores",
			Active:        true,
			SortOrder:     45,
		},
		{
			ID:            "gas_station",
			Name:          "Gas Stations",
			Category:      "automotive",
			Icon:          "⛽",
			OSMPrimaryTag: "amenity=fuel",
			Description:   "Gas stations and fuel services",
			Active:        true,
			SortOrder:     46,
		},
		{
			ID:            "motorcycle_dealer",
			Name:          "Motorcycle Dealers",
			Category:      "automotive",
			Icon:          "🏍️",
			OSMPrimaryTag: "shop=motorcycle",
			Description:   "Motorcycle dealerships and sales",
			Active:        true,
			SortOrder:     47,
		},
		{
			ID:            "rv_dealer",
			Name:          "RV Dealers",
			Category:      "automotive",
			Icon:          "🚐",
			OSMPrimaryTag: "shop=caravan",
			Description:   "RV and camper dealerships",
			Active:        true,
			SortOrder:     48,
		},

		// Retail - Additional (15 new)
		{
			ID:            "florist",
			Name:          "Florists",
			Category:      "retail",
			Icon:          "💐",
			OSMPrimaryTag: "shop=florist",
			Description:   "Flower shops and florists",
			Active:        true,
			SortOrder:     49,
		},
		{
			ID:            "bookstore",
			Name:          "Bookstores",
			Category:      "retail",
			Icon:          "📚",
			OSMPrimaryTag: "shop=books",
			Description:   "Bookstores and book shops",
			Active:        true,
			SortOrder:     50,
		},
		{
			ID:            "electronics",
			Name:          "Electronics Stores",
			Category:      "retail",
			Icon:          "📱",
			OSMPrimaryTag: "shop=electronics",
			Description:   "Electronics and technology stores",
			Active:        true,
			SortOrder:     51,
		},
		{
			ID:            "furniture",
			Name:          "Furniture Stores",
			Category:      "retail",
			Icon:          "🛋️",
			OSMPrimaryTag: "shop=furniture",
			Description:   "Furniture and home furnishing stores",
			Active:        true,
			SortOrder:     52,
		},
		{
			ID:            "hardware",
			Name:          "Hardware Stores",
			Category:      "retail",
			Icon:          "🔨",
			OSMPrimaryTag: "shop=hardware",
			Description:   "Hardware and home improvement stores",
			Active:        true,
			SortOrder:     53,
		},
		{
			ID:            "jewelry",
			Name:          "Jewelry Stores",
			Category:      "retail",
			Icon:          "💍",
			OSMPrimaryTag: "shop=jewelry",
			Description:   "Jewelry and accessory stores",
			Active:        true,
			SortOrder:     54,
		},
		{
			ID:            "gift_shop",
			Name:          "Gift Shops",
			Category:      "retail",
			Icon:          "🎁",
			OSMPrimaryTag: "shop=gift",
			Description:   "Gift shops and novelty stores",
			Active:        true,
			SortOrder:     55,
		},
		{
			ID:            "toy_store",
			Name:          "Toy Stores",
			Category:      "retail",
			Icon:          "🧸",
			OSMPrimaryTag: "shop=toys",
			Description:   "Toy stores and children's shops",
			Active:        true,
			SortOrder:     56,
		},
		{
			ID:            "pet_store",
			Name:          "Pet Stores",
			Category:      "retail",
			Icon:          "🐕",
			OSMPrimaryTag: "shop=pet",
			Description:   "Pet stores and pet supplies",
			Active:        true,
			SortOrder:     57,
		},
		{
			ID:            "bicycle_shop",
			Name:          "Bicycle Shops",
			Category:      "retail",
			Icon:          "🚴",
			OSMPrimaryTag: "shop=bicycle",
			Description:   "Bicycle shops and cycling stores",
			Active:        true,
			SortOrder:     58,
		},
		{
			ID:            "sporting_goods",
			Name:          "Sporting Goods",
			Category:      "retail",
			Icon:          "⚽",
			OSMPrimaryTag: "shop=sports",
			Description:   "Sporting goods and athletic equipment stores",
			Active:        true,
			SortOrder:     59,
		},
		{
			ID:            "music_store",
			Name:          "Music Stores",
			Category:      "retail",
			Icon:          "🎸",
			OSMPrimaryTag: "shop=music",
			Description:   "Music stores and instrument shops",
			Active:        true,
			SortOrder:     60,
		},
		{
			ID:            "art_supply",
			Name:          "Art Supply Stores",
			Category:      "retail",
			Icon:          "🎨",
			OSMPrimaryTag: "shop=art",
			Description:   "Art supply and craft stores",
			Active:        true,
			SortOrder:     61,
		},
		{
			ID:            "stationery",
			Name:          "Stationery Stores",
			Category:      "retail",
			Icon:          "✏️",
			OSMPrimaryTag: "shop=stationery",
			Description:   "Stationery and office supply stores",
			Active:        true,
			SortOrder:     62,
		},
		{
			ID:            "garden_center",
			Name:          "Garden Centers",
			Category:      "retail",
			Icon:          "🌱",
			OSMPrimaryTag: "shop=garden_centre",
			Description:   "Garden centers and plant nurseries",
			Active:        true,
			SortOrder:     63,
		},

		// Professional Services - Additional (10 new)
		{
			ID:            "real_estate",
			Name:          "Real Estate Agencies",
			Category:      "professional",
			Icon:          "🏡",
			OSMPrimaryTag: "office=estate_agent",
			Description:   "Real estate agencies and property services",
			Active:        true,
			SortOrder:     64,
		},
		{
			ID:            "insurance",
			Name:          "Insurance Agencies",
			Category:      "professional",
			Icon:          "🛡️",
			OSMPrimaryTag: "office=insurance",
			Description:   "Insurance agencies and brokers",
			Active:        true,
			SortOrder:     65,
		},
		{
			ID:            "financial_advisor",
			Name:          "Financial Advisors",
			Category:      "professional",
			Icon:          "💼",
			OSMPrimaryTag: "office=financial_advisor",
			Description:   "Financial advisors and wealth management",
			Active:        true,
			SortOrder:     66,
		},
		{
			ID:            "notary",
			Name:          "Notaries",
			Category:      "professional",
			Icon:          "📋",
			OSMPrimaryTag: "office=notary",
			Description:   "Notary public and document services",
			Active:        true,
			SortOrder:     67,
		},
		{
			ID:            "tax_service",
			Name:          "Tax Services",
			Category:      "professional",
			Icon:          "💰",
			OSMPrimaryTag: "office=tax_advisor",
			Description:   "Tax preparation and advisory services",
			Active:        true,
			SortOrder:     68,
		},
		{
			ID:            "marketing_agency",
			Name:          "Marketing Agencies",
			Category:      "professional",
			Icon:          "📢",
			OSMPrimaryTag: "office=advertising_agency",
			Description:   "Marketing and advertising agencies",
			Active:        true,
			SortOrder:     69,
		},
		{
			ID:            "photography",
			Name:          "Photography Studios",
			Category:      "professional",
			Icon:          "📷",
			OSMPrimaryTag: "shop=photo",
			Description:   "Photography studios and services",
			Active:        true,
			SortOrder:     70,
		},
		{
			ID:            "printing",
			Name:          "Print Shops",
			Category:      "professional",
			Icon:          "🖨️",
			OSMPrimaryTag: "shop=copyshop",
			Description:   "Print shops and copy centers",
			Active:        true,
			SortOrder:     71,
		},
		{
			ID:            "event_planning",
			Name:          "Event Planners",
			Category:      "professional",
			Icon:          "🎉",
			OSMPrimaryTag: "office=event_planning",
			Description:   "Event planning and coordination services",
			Active:        true,
			SortOrder:     72,
		},
		{
			ID:            "travel_agency",
			Name:          "Travel Agencies",
			Category:      "professional",
			Icon:          "✈️",
			OSMPrimaryTag: "shop=travel_agency",
			Description:   "Travel agencies and booking services",
			Active:        true,
			SortOrder:     73,
		},

		// Hospitality (NEW CATEGORY - 8 industries)
		{
			ID:            "hotel",
			Name:          "Hotels",
			Category:      "hospitality",
			Icon:          "🏨",
			OSMPrimaryTag: "tourism=hotel",
			Description:   "Hotels and accommodations",
			Active:        true,
			SortOrder:     74,
		},
		{
			ID:            "motel",
			Name:          "Motels",
			Category:      "hospitality",
			Icon:          "🛏️",
			OSMPrimaryTag: "tourism=motel",
			Description:   "Motels and roadside lodging",
			Active:        true,
			SortOrder:     75,
		},
		{
			ID:            "hostel",
			Name:          "Hostels",
			Category:      "hospitality",
			Icon:          "🏠",
			OSMPrimaryTag: "tourism=hostel",
			Description:   "Hostels and budget accommodations",
			Active:        true,
			SortOrder:     76,
		},
		{
			ID:            "bed_breakfast",
			Name:          "Bed & Breakfasts",
			Category:      "hospitality",
			Icon:          "🏡",
			OSMPrimaryTag: "tourism=guest_house",
			Description:   "Bed and breakfast establishments",
			Active:        true,
			SortOrder:     77,
		},
		{
			ID:            "vacation_rental",
			Name:          "Vacation Rentals",
			Category:      "hospitality",
			Icon:          "🏖️",
			OSMPrimaryTag: "tourism=apartment",
			Description:   "Vacation rentals and holiday homes",
			Active:        true,
			SortOrder:     78,
		},
		{
			ID:            "campground",
			Name:          "Campgrounds",
			Category:      "hospitality",
			Icon:          "⛺",
			OSMPrimaryTag: "tourism=camp_site",
			Description:   "Campgrounds and camping facilities",
			Active:        true,
			SortOrder:     79,
		},
		{
			ID:            "rv_park",
			Name:          "RV Parks",
			Category:      "hospitality",
			Icon:          "🚐",
			OSMPrimaryTag: "tourism=caravan_site",
			Description:   "RV parks and caravan sites",
			Active:        true,
			SortOrder:     80,
		},
		{
			ID:            "resort",
			Name:          "Resorts",
			Category:      "hospitality",
			Icon:          "🌴",
			OSMPrimaryTag: "tourism=resort",
			Description:   "Resorts and vacation complexes",
			Active:        true,
			SortOrder:     81,
		},

		// Home Services (NEW CATEGORY - 9 industries)
		{
			ID:            "plumber",
			Name:          "Plumbers",
			Category:      "home_services",
			Icon:          "🚰",
			OSMPrimaryTag: "craft=plumber",
			Description:   "Plumbing services and repairs",
			Active:        true,
			SortOrder:     82,
		},
		{
			ID:            "electrician",
			Name:          "Electricians",
			Category:      "home_services",
			Icon:          "⚡",
			OSMPrimaryTag: "craft=electrician",
			Description:   "Electrical services and repairs",
			Active:        true,
			SortOrder:     83,
		},
		{
			ID:            "hvac",
			Name:          "HVAC Services",
			Category:      "home_services",
			Icon:          "🌡️",
			OSMPrimaryTag: "craft=hvac",
			Description:   "Heating, ventilation, and air conditioning services",
			Active:        true,
			SortOrder:     84,
		},
		{
			ID:            "locksmith",
			Name:          "Locksmiths",
			Category:      "home_services",
			Icon:          "🔑",
			OSMPrimaryTag: "craft=locksmith",
			Description:   "Locksmith and security services",
			Active:        true,
			SortOrder:     85,
		},
		{
			ID:            "roofing",
			Name:          "Roofing Services",
			Category:      "home_services",
			Icon:          "🏠",
			OSMPrimaryTag: "craft=roofer",
			Description:   "Roofing and roof repair services",
			Active:        true,
			SortOrder:     86,
		},
		{
			ID:            "painting",
			Name:          "Painting Services",
			Category:      "home_services",
			Icon:          "🎨",
			OSMPrimaryTag: "craft=painter",
			Description:   "House painting and decorating services",
			Active:        true,
			SortOrder:     87,
		},
		{
			ID:            "cleaning",
			Name:          "Cleaning Services",
			Category:      "home_services",
			Icon:          "🧹",
			OSMPrimaryTag: "office=cleaning",
			Description:   "Professional cleaning and janitorial services",
			Active:        true,
			SortOrder:     88,
		},
		{
			ID:            "landscaping",
			Name:          "Landscaping Services",
			Category:      "home_services",
			Icon:          "🌳",
			OSMPrimaryTag: "craft=gardener",
			Description:   "Landscaping and lawn care services",
			Active:        true,
			SortOrder:     89,
		},
		{
			ID:            "pest_control",
			Name:          "Pest Control",
			Category:      "home_services",
			Icon:          "🐜",
			OSMPrimaryTag: "craft=pest_control",
			Description:   "Pest control and extermination services",
			Active:        true,
			SortOrder:     90,
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
		{
			ID:          "hospitality",
			Name:        "Hospitality",
			Icon:        "🏨",
			Description: "Hotels, motels, resorts, and accommodations",
			SortOrder:   7,
		},
		{
			ID:          "home_services",
			Name:        "Home Services",
			Icon:        "🏠",
			Description: "Plumbing, electrical, HVAC, and home repair services",
			SortOrder:   8,
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
