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

	// Personal Care & Beauty - Additional
	"hair_salon": {
		Prefixes: []string{"Salon", "Style", "Chic", "Elite", "Modern", "Classic", "Glamour", "Trend", "Fashion", "Beautiful"},
		Suffixes: []string{"Hair Salon", "Hair Studio", "Hair Design", "Hairstyling", "Hair Care", "Hair & Beauty"},
	},
	"tanning_salon": {
		Prefixes: []string{"Sun", "Bronze", "Golden", "Tropical", "Paradise", "Sunshine", "Island", "Beach", "Glow", "Tan"},
		Suffixes: []string{"Tanning Salon", "Tanning Studio", "Tan", "Sunbeds", "Tanning Center", "Tanning Lounge"},
	},
	"cosmetics": {
		Prefixes: []string{"Beauty", "Glamour", "Luxury", "Pure", "Divine", "Radiant", "Elegant", "Chic", "Premium", "Elite"},
		Suffixes: []string{"Cosmetics", "Beauty Products", "Makeup", "Beauty Store", "Cosmetics Shop", "Beauty Boutique"},
	},
	"perfumery": {
		Prefixes: []string{"Essence", "Aroma", "Scent", "Fragrance", "Parfum", "Luxury", "Elite", "Divine", "Pure", "Classic"},
		Suffixes: []string{"Perfumery", "Fragrances", "Scents", "Perfume Shop", "Perfume Boutique", "Aromas"},
	},
	"waxing_salon": {
		Prefixes: []string{"Smooth", "Silk", "Pure", "Perfect", "Bare", "Soft", "Flawless", "Clean", "Natural", "Beauty"},
		Suffixes: []string{"Waxing", "Wax Studio", "Hair Removal", "Waxing Salon", "Wax Bar", "Waxing Center"},
	},

	// Health & Wellness - Additional
	"chiropractor": {
		Prefixes: []string{"Align", "Wellness", "Spine", "Health", "Active", "Complete", "Total", "Advanced", "Modern", "Family"},
		Suffixes: []string{"Chiropractic", "Chiropractor", "Spine Care", "Wellness Center", "Chiropractic Clinic", "Health"},
	},
	"optician": {
		Prefixes: []string{"Vision", "Clear", "Perfect", "Eye", "Sight", "Premier", "Elite", "Modern", "Professional", "Quality"},
		Suffixes: []string{"Opticians", "Eyewear", "Vision Center", "Eye Care", "Optical", "Eyeglasses"},
	},
	"clinic": {
		Prefixes: []string{"Family", "Community", "Health", "Medical", "Care", "Wellness", "Advanced", "Complete", "Premier", "Quality"},
		Suffixes: []string{"Clinic", "Medical Clinic", "Health Center", "Healthcare", "Medical Care", "Family Practice"},
	},
	"hospital": {
		Prefixes: []string{"General", "Regional", "Community", "Medical", "Health", "University", "Memorial", "St.", "City", "County"},
		Suffixes: []string{"Hospital", "Medical Center", "Health System", "Healthcare", "Medical Campus"},
	},
	"veterinary": {
		Prefixes: []string{"Pet", "Animal", "Caring", "Companion", "Family", "Advanced", "Complete", "Quality", "Premier", "Modern"},
		Suffixes: []string{"Veterinary", "Animal Hospital", "Vet Clinic", "Pet Care", "Animal Care", "Veterinary Services"},
	},
	"yoga_studio": {
		Prefixes: []string{"Zen", "Peaceful", "Harmony", "Balance", "Flow", "Pure", "Mindful", "Serenity", "Bliss", "Om"},
		Suffixes: []string{"Yoga", "Yoga Studio", "Yoga Center", "Yoga & Wellness", "Yoga Practice", "Yoga Space"},
	},
	"pilates_studio": {
		Prefixes: []string{"Core", "Balance", "Strength", "Pure", "Elite", "Modern", "Classical", "Zen", "Flow", "Peak"},
		Suffixes: []string{"Pilates", "Pilates Studio", "Pilates Center", "Core Fitness", "Pilates & Wellness"},
	},
	"physical_therapy": {
		Prefixes: []string{"Active", "Motion", "Recovery", "Restore", "Advanced", "Complete", "Premier", "Quality", "Total", "Modern"},
		Suffixes: []string{"Physical Therapy", "PT", "Therapy", "Rehab", "Rehabilitation", "Therapy Center"},
	},

	// Food & Beverage - Additional
	"fast_food": {
		Prefixes: []string{"Quick", "Express", "Fast", "Speed", "Rapid", "Instant", "Super", "Mega", "Ultimate", "Best"},
		Suffixes: []string{"Fast Food", "Quick Eats", "Express", "Burgers", "Food", "Grill"},
	},
	"ice_cream": {
		Prefixes: []string{"Sweet", "Frozen", "Creamy", "Cold", "Chill", "Frosty", "Cool", "Tasty", "Yummy", "Delicious"},
		Suffixes: []string{"Ice Cream", "Creamery", "Ice Cream Shop", "Frozen Treats", "Gelato", "Ice Cream Parlor"},
	},
	"juice_bar": {
		Prefixes: []string{"Fresh", "Pure", "Healthy", "Green", "Organic", "Natural", "Vital", "Energy", "Juice", "Raw"},
		Suffixes: []string{"Juice Bar", "Juice", "Smoothies", "Juicery", "Juice & Smoothie", "Fresh Juice"},
	},
	"pizza": {
		Prefixes: []string{"Pizza", "Slice", "Tony's", "Luigi's", "Mario's", "New York", "Chicago", "Italian", "Gourmet", "Authentic"},
		Suffixes: []string{"Pizza", "Pizzeria", "Pizza House", "Pizza Kitchen", "Pizza Place", "Pizza Shop"},
	},
	"sushi": {
		Prefixes: []string{"Sushi", "Tokyo", "Osaka", "Zen", "Sakura", "Koi", "Dragon", "Samurai", "Shogun", "Imperial"},
		Suffixes: []string{"Sushi", "Sushi Bar", "Japanese Restaurant", "Sushi House", "Sushi Kitchen", "Japanese Cuisine"},
	},
	"brewery": {
		Prefixes: []string{"Craft", "Artisan", "Local", "Barrel", "Hop", "Malt", "Brew", "Iron", "Stone", "Mountain"},
		Suffixes: []string{"Brewery", "Brewing Company", "Craft Beer", "Brewhouse", "Beer Co", "Ale House"},
	},
	"winery": {
		Prefixes: []string{"Valley", "Estate", "Vineyard", "Hillside", "Heritage", "Noble", "Reserve", "Private", "Grand", "Royal"},
		Suffixes: []string{"Winery", "Vineyards", "Estate Wines", "Wine Cellars", "Vintners", "Wine Company"},
	},
	"food_truck": {
		Prefixes: []string{"Street", "Mobile", "Gourmet", "Urban", "Rolling", "Tasty", "Flavor", "Hungry", "Food", "Wheels"},
		Suffixes: []string{"Food Truck", "Kitchen", "Eats", "Street Food", "Mobile Kitchen", "Food Wheels"},
	},
	"catering": {
		Prefixes: []string{"Elegant", "Premier", "Gourmet", "Deluxe", "Classic", "Elite", "Quality", "Perfect", "Royal", "Grand"},
		Suffixes: []string{"Catering", "Catering Services", "Event Catering", "Catering Company", "Events", "Catering Co"},
	},
	"deli": {
		Prefixes: []string{"Corner", "Local", "Artisan", "Gourmet", "Classic", "European", "Fresh", "Quality", "Premium", "Fine"},
		Suffixes: []string{"Deli", "Delicatessen", "Market", "Food Market", "Specialty Foods", "Gourmet Market"},
	},

	// Automotive - Additional
	"tire_shop": {
		Prefixes: []string{"Tire", "Wheel", "Quick", "Express", "Premier", "Quality", "Pro", "Expert", "Complete", "Total"},
		Suffixes: []string{"Tire Shop", "Tire Center", "Tires", "Tire Service", "Wheels", "Tire & Auto"},
	},
	"auto_parts": {
		Prefixes: []string{"Auto", "Quality", "Premier", "Advanced", "Complete", "Total", "Express", "Quick", "Pro", "Expert"},
		Suffixes: []string{"Auto Parts", "Parts", "Auto Supplies", "Car Parts", "Automotive Parts", "Parts Center"},
	},
	"gas_station": {
		Prefixes: []string{"Quick", "Express", "Fast", "Speed", "24/7", "Corner", "Main", "Central", "City", "Highway"},
		Suffixes: []string{"Gas Station", "Fuel", "Gas & Go", "Service Station", "Fuel Stop", "Gas Mart"},
	},
	"motorcycle_dealer": {
		Prefixes: []string{"Speed", "Thunder", "Iron", "Chrome", "Eagle", "Harley", "Custom", "Sport", "Power", "Freedom"},
		Suffixes: []string{"Motorcycles", "Cycles", "Motorcycle Sales", "Bike Shop", "Motorsports", "Powersports"},
	},
	"rv_dealer": {
		Prefixes: []string{"Adventure", "Freedom", "Camping", "Travel", "Journey", "Road", "Happy", "Premier", "Quality", "Best"},
		Suffixes: []string{"RV Sales", "RV Center", "RV Dealer", "RVs", "Motorhomes", "Campers"},
	},

	// Retail - Additional
	"florist": {
		Prefixes: []string{"Bloom", "Petal", "Blossom", "Garden", "Fresh", "Floral", "Rose", "Violet", "Daisy", "Lily"},
		Suffixes: []string{"Florist", "Flowers", "Floral Design", "Flower Shop", "Bouquets", "Arrangements"},
	},
	"bookstore": {
		Prefixes: []string{"Page", "Novel", "Chapter", "Story", "Book", "Reader", "Literary", "Classic", "Modern", "Corner"},
		Suffixes: []string{"Bookstore", "Books", "Bookshop", "Book House", "Reading Room", "Book Center"},
	},
	"electronics": {
		Prefixes: []string{"Tech", "Digital", "Electronic", "Gadget", "Modern", "Advanced", "Smart", "Cyber", "High-Tech", "Future"},
		Suffixes: []string{"Electronics", "Tech Store", "Technology", "Electronics Store", "Tech Shop", "Gadgets"},
	},
	"furniture": {
		Prefixes: []string{"Home", "Modern", "Classic", "Contemporary", "Elegant", "Quality", "Fine", "Custom", "Designer", "Urban"},
		Suffixes: []string{"Furniture", "Furnishings", "Home Store", "Furniture Gallery", "Home Decor", "Furniture Outlet"},
	},
	"hardware": {
		Prefixes: []string{"Home", "Builder", "Ace", "True", "City", "Country", "Pro", "Quality", "Complete", "Total"},
		Suffixes: []string{"Hardware", "Hardware Store", "Home Improvement", "Building Supply", "Hardware & Tools"},
	},
	"jewelry": {
		Prefixes: []string{"Diamond", "Gold", "Silver", "Precious", "Fine", "Elegant", "Luxury", "Royal", "Estate", "Classic"},
		Suffixes: []string{"Jewelry", "Jewelers", "Jewelry Store", "Fine Jewelry", "Gems", "Jewelry Gallery"},
	},
	"gift_shop": {
		Prefixes: []string{"Gift", "Treasure", "Special", "Unique", "Perfect", "Sweet", "Charming", "Lovely", "Delightful", "Amazing"},
		Suffixes: []string{"Gift Shop", "Gifts", "Gift Gallery", "Gift Boutique", "Gifts & More", "Gift Store"},
	},
	"toy_store": {
		Prefixes: []string{"Toy", "Fun", "Kids", "Play", "Tiny", "Little", "Wonder", "Magic", "Happy", "Joyful"},
		Suffixes: []string{"Toy Store", "Toys", "Toy Shop", "Toy Box", "Toy Kingdom", "Toy World"},
	},
	"pet_store": {
		Prefixes: []string{"Pet", "Paws", "Furry", "Happy", "Healthy", "Pampered", "Critter", "Animal", "Pet", "Companion"},
		Suffixes: []string{"Pet Store", "Pet Shop", "Pet Supplies", "Pets", "Pet Care", "Pet Market"},
	},
	"bicycle_shop": {
		Prefixes: []string{"Cycle", "Bike", "Pedal", "Wheel", "Speed", "Pro", "Mountain", "Road", "Sport", "Gear"},
		Suffixes: []string{"Bicycle Shop", "Bikes", "Cycle Center", "Bicycle Store", "Cycling", "Bike Shop"},
	},
	"sporting_goods": {
		Prefixes: []string{"Sport", "Athletic", "Pro", "Champion", "Victory", "Team", "Performance", "Active", "Fitness", "Game"},
		Suffixes: []string{"Sporting Goods", "Sports", "Sports Store", "Athletic Store", "Sports Equipment", "Sportswear"},
	},
	"music_store": {
		Prefixes: []string{"Music", "Sound", "Melody", "Harmony", "Tune", "Note", "Rock", "Guitar", "Piano", "Instrument"},
		Suffixes: []string{"Music Store", "Music Shop", "Music Center", "Instruments", "Music House", "Music Gallery"},
	},
	"art_supply": {
		Prefixes: []string{"Art", "Creative", "Artist", "Canvas", "Palette", "Studio", "Craft", "Color", "Design", "Artistic"},
		Suffixes: []string{"Art Supply", "Art Store", "Art Shop", "Art Materials", "Art & Craft", "Art Studio"},
	},
	"stationery": {
		Prefixes: []string{"Paper", "Write", "Pen", "Office", "Note", "Quality", "Fine", "Classic", "Modern", "Professional"},
		Suffixes: []string{"Stationery", "Office Supply", "Paper Store", "Stationery Shop", "Office Supplies", "Paper Goods"},
	},
	"garden_center": {
		Prefixes: []string{"Green", "Garden", "Plant", "Nature", "Bloom", "Grow", "Fresh", "Botanical", "Nursery", "Greenhouse"},
		Suffixes: []string{"Garden Center", "Nursery", "Garden Shop", "Plants", "Garden Supply", "Garden Store"},
	},

	// Professional Services - Additional
	"real_estate": {
		Prefixes: []string{"Premier", "Elite", "Quality", "Best", "Top", "First", "Prime", "Century", "Realty", "Home"},
		Suffixes: []string{"Real Estate", "Realty", "Properties", "Real Estate Agency", "Realtors", "Property Services"},
	},
	"insurance": {
		Prefixes: []string{"Safe", "Secure", "Trust", "Shield", "Guardian", "Reliable", "Premier", "Quality", "Complete", "Total"},
		Suffixes: []string{"Insurance", "Insurance Agency", "Insurance Services", "Insurance Group", "Assurance"},
	},
	"financial_advisor": {
		Prefixes: []string{"Wealth", "Premier", "Elite", "Quality", "Trust", "Capital", "Strategic", "Prosperity", "Vision", "Summit"},
		Suffixes: []string{"Financial Advisors", "Wealth Management", "Financial Services", "Financial Planning", "Investment Advisors"},
	},
	"notary": {
		Prefixes: []string{"Professional", "Reliable", "Certified", "Quick", "Mobile", "Express", "Quality", "Premier", "Complete", "Total"},
		Suffixes: []string{"Notary Public", "Notary Services", "Notary", "Notarization", "Notary Office"},
	},
	"tax_service": {
		Prefixes: []string{"Professional", "Expert", "Premier", "Quality", "Complete", "Total", "Express", "Quick", "Reliable", "Accurate"},
		Suffixes: []string{"Tax Services", "Tax Preparation", "Tax Advisors", "Tax Professionals", "Tax Center", "Tax Solutions"},
	},
	"marketing_agency": {
		Prefixes: []string{"Creative", "Digital", "Brand", "Strategic", "Modern", "Innovative", "Premier", "Elite", "Pro", "Dynamic"},
		Suffixes: []string{"Marketing", "Marketing Agency", "Advertising", "Marketing Solutions", "Marketing Group", "Media"},
	},
	"photography": {
		Prefixes: []string{"Picture", "Portrait", "Photo", "Image", "Capture", "Lens", "Focus", "Frame", "Vision", "Creative"},
		Suffixes: []string{"Photography", "Photo Studio", "Photography Services", "Photos", "Photographers", "Photo Lab"},
	},
	"printing": {
		Prefixes: []string{"Quick", "Express", "Fast", "Print", "Copy", "Digital", "Quality", "Premier", "Professional", "Complete"},
		Suffixes: []string{"Printing", "Print Shop", "Copy Center", "Print Services", "Printers", "Printing Services"},
	},
	"event_planning": {
		Prefixes: []string{"Perfect", "Elegant", "Premier", "Elite", "Classic", "Dream", "Special", "Unique", "Creative", "Exceptional"},
		Suffixes: []string{"Event Planning", "Events", "Event Services", "Event Planners", "Event Coordination", "Event Design"},
	},
	"travel_agency": {
		Prefixes: []string{"Travel", "Journey", "Adventure", "Explore", "Discover", "Vacation", "Getaway", "Dream", "World", "Global"},
		Suffixes: []string{"Travel Agency", "Travel", "Travel Services", "Tours", "Travel Consultants", "Travel Planners"},
	},

	// Hospitality
	"hotel": {
		Prefixes: []string{"Grand", "Royal", "Plaza", "Marriott", "Hilton", "Sheraton", "Hyatt", "Embassy", "Holiday", "Comfort"},
		Suffixes: []string{"Hotel", "Inn", "Suites", "Resort", "Lodge", "Palace"},
	},
	"motel": {
		Prefixes: []string{"Roadside", "Highway", "Rest", "Travelers", "Budget", "Comfort", "Quality", "Sleep", "Stay", "Wayside"},
		Suffixes: []string{"Motel", "Motor Inn", "Motor Lodge", "Inn", "Lodge"},
	},
	"hostel": {
		Prefixes: []string{"Backpackers", "Travelers", "Budget", "Youth", "Friendly", "Cozy", "Happy", "Social", "Urban", "Downtown"},
		Suffixes: []string{"Hostel", "Backpackers", "Accommodation", "Lodge", "Guest House"},
	},
	"bed_breakfast": {
		Prefixes: []string{"Cozy", "Charming", "Historic", "Victorian", "Country", "Manor", "Inn", "Garden", "Cottage", "Homestead"},
		Suffixes: []string{"B&B", "Bed & Breakfast", "Inn", "Guest House", "Lodge"},
	},
	"vacation_rental": {
		Prefixes: []string{"Beach", "Mountain", "Lake", "Vacation", "Holiday", "Getaway", "Paradise", "Dream", "Luxury", "Private"},
		Suffixes: []string{"Vacation Rental", "Holiday Home", "Rental", "Vacation Home", "Retreat"},
	},
	"campground": {
		Prefixes: []string{"Pine", "Forest", "Lake", "River", "Mountain", "Sunset", "Wilderness", "Nature", "Scenic", "Happy"},
		Suffixes: []string{"Campground", "Camping", "Camp", "Campsite", "Camping Resort"},
	},
	"rv_park": {
		Prefixes: []string{"Sunset", "Mountain", "Lake", "Happy", "Paradise", "Riverside", "Valley", "Desert", "Forest", "Country"},
		Suffixes: []string{"RV Park", "RV Resort", "RV Campground", "Motor Home Park", "RV Camp"},
	},
	"resort": {
		Prefixes: []string{"Paradise", "Grand", "Luxury", "Royal", "Tropical", "Beach", "Mountain", "Spa", "Premier", "Executive"},
		Suffixes: []string{"Resort", "Resort & Spa", "Beach Resort", "Golf Resort", "Vacation Resort"},
	},

	// Home Services
	"plumber": {
		Prefixes: []string{"Rapid", "Expert", "Pro", "Quality", "Reliable", "24/7", "Emergency", "Master", "Licensed", "Professional"},
		Suffixes: []string{"Plumbing", "Plumbers", "Plumbing Services", "Pipe Repair", "Drain Cleaning", "Plumbing Solutions"},
	},
	"electrician": {
		Prefixes: []string{"Expert", "Professional", "Licensed", "Master", "Quality", "Reliable", "24/7", "Emergency", "Complete", "Total"},
		Suffixes: []string{"Electrical", "Electrician", "Electrical Services", "Electric", "Electrical Contractors", "Electrical Solutions"},
	},
	"hvac": {
		Prefixes: []string{"Comfort", "Climate", "Air", "Heating", "Cooling", "Quality", "Premier", "Expert", "Professional", "Reliable"},
		Suffixes: []string{"HVAC", "Heating & Cooling", "Climate Control", "HVAC Services", "Air Conditioning", "Heating Services"},
	},
	"locksmith": {
		Prefixes: []string{"Quick", "Emergency", "24/7", "Mobile", "Express", "Professional", "Master", "Reliable", "Secure", "Safe"},
		Suffixes: []string{"Locksmith", "Lock Services", "Locksmith Services", "Lock & Key", "Security", "Locks"},
	},
	"roofing": {
		Prefixes: []string{"Quality", "Premier", "Professional", "Expert", "Reliable", "Complete", "Total", "Master", "Elite", "Advanced"},
		Suffixes: []string{"Roofing", "Roofers", "Roofing Services", "Roof Repair", "Roofing Contractors", "Roofing Solutions"},
	},
	"painting": {
		Prefixes: []string{"Professional", "Quality", "Expert", "Premier", "Complete", "Total", "Custom", "Precision", "Master", "Elite"},
		Suffixes: []string{"Painting", "Painters", "Painting Services", "Paint Contractors", "Painting Solutions", "House Painters"},
	},
	"cleaning": {
		Prefixes: []string{"Sparkle", "Clean", "Fresh", "Pure", "Professional", "Quality", "Expert", "Premier", "Complete", "Total"},
		Suffixes: []string{"Cleaning", "Cleaning Services", "Cleaners", "Janitorial", "Maid Service", "Housekeeping"},
	},
	"landscaping": {
		Prefixes: []string{"Green", "Lawn", "Garden", "Landscape", "Nature", "Quality", "Premier", "Professional", "Expert", "Complete"},
		Suffixes: []string{"Landscaping", "Lawn Care", "Landscape Services", "Landscapers", "Landscape Design", "Yard Services"},
	},
	"pest_control": {
		Prefixes: []string{"Pest", "Bug", "Termite", "Critter", "Safe", "Eco", "Green", "Professional", "Quality", "Reliable"},
		Suffixes: []string{"Pest Control", "Exterminators", "Pest Services", "Pest Management", "Exterminating", "Pest Solutions"},
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
