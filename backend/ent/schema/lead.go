package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Lead holds the schema definition for the Lead entity.
type Lead struct {
	ent.Schema
}

// Fields of the Lead.
func (Lead) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Comment("Business name"),
		field.Enum("industry").
			Values(
				// Personal Care & Beauty (10)
				"tattoo", "beauty", "barber", "spa", "nail_salon",
				"hair_salon", "tanning_salon", "cosmetics", "perfumery", "waxing_salon",

				// Health & Wellness (12)
				"gym", "dentist", "pharmacy", "massage",
				"chiropractor", "optician", "clinic", "hospital", "veterinary",
				"yoga_studio", "pilates_studio", "physical_therapy",

				// Food & Beverage (14)
				"restaurant", "cafe", "bar", "bakery",
				"fast_food", "ice_cream", "juice_bar", "pizza", "sushi",
				"brewery", "winery", "food_truck", "catering", "deli",

				// Automotive (8)
				"car_repair", "car_wash", "car_dealer",
				"tire_shop", "auto_parts", "gas_station", "motorcycle_dealer", "rv_dealer",

				// Retail (17)
				"clothing", "convenience",
				"florist", "bookstore", "electronics", "furniture", "hardware",
				"jewelry", "gift_shop", "toy_store", "pet_store", "bicycle_shop",
				"sporting_goods", "music_store", "art_supply", "stationery", "garden_center",

				// Professional Services (12)
				"lawyer", "accountant",
				"real_estate", "insurance", "financial_advisor", "notary", "tax_service",
				"marketing_agency", "photography", "printing", "event_planning", "travel_agency",

				// Hospitality (8)
				"hotel", "motel", "hostel", "bed_breakfast", "vacation_rental",
				"campground", "rv_park", "resort",

				// Home Services (9)
				"plumber", "electrician", "hvac", "locksmith", "roofing",
				"painting", "cleaning", "landscaping", "pest_control",
			).
			Comment("Industry type"),
		field.String("country").
			NotEmpty().
			MaxLen(2).
			Comment("ISO 3166-1 alpha-2 country code"),
		field.String("city").
			NotEmpty().
			Comment("City name"),
		field.String("address").
			Optional().
			Comment("Full street address"),
		field.String("postal_code").
			Optional().
			Comment("Postal/ZIP code"),
		field.String("phone").
			Optional().
			Comment("Phone number"),
		field.String("email").
			Optional().
			Comment("Email address"),
		field.String("website").
			Optional().
			Comment("Website URL"),
		field.JSON("social_media", map[string]string{}).
			Optional().
			Comment("Social media links (facebook, instagram, twitter, etc.)"),
		field.Float("latitude").
			Optional().
			Comment("GPS latitude"),
		field.Float("longitude").
			Optional().
			Comment("GPS longitude"),
		field.Bool("verified").
			Default(false).
			Comment("Whether the lead has been verified"),
		field.Int("quality_score").
			Default(50).
			Min(0).
			Max(100).
			Comment("Data quality score (0-100)"),

		// CRM Lifecycle fields
		field.Enum("status").
			Values("new", "contacted", "qualified", "negotiating", "won", "lost", "archived").
			Default("new").
			Comment("Lead lifecycle status"),
		field.Time("status_changed_at").
			Default(time.Now).
			Comment("When the status was last changed"),
		field.JSON("custom_fields", map[string]interface{}{}).
			Optional().
			Comment("User-defined custom fields (flexible metadata storage)"),
		field.String("osm_id").
			Optional().
			Comment("OpenStreetMap ID"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional metadata from OSM"),

		// Sub-niche categorization fields
		field.String("sub_niche").
			Optional().
			Comment("Sub-category within industry (e.g., italian, crossfit, watercolor)"),
		field.JSON("specialties", []string{}).
			Optional().
			Comment("Additional specialty tags (e.g., [pasta, seafood, fine_dining])"),

		// Industry-specific fields
		field.String("cuisine_type").
			Optional().
			Comment("For restaurants: cuisine type from OSM cuisine= tag"),
		field.String("sport_type").
			Optional().
			Comment("For gyms: sport/fitness type from OSM sport= tag"),
		field.String("tattoo_style").
			Optional().
			Comment("For tattoos: style type (traditional, japanese, watercolor)"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("Last update timestamp"),
	}
}

// Edges of the Lead.
func (Lead) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("notes", LeadNote.Type).
			Comment("Notes and comments on this lead"),

		edge.To("status_history", LeadStatusHistory.Type).
			Comment("History of status changes for this lead"),
	}
}

// Indexes of the Lead.
func (Lead) Indexes() []ent.Index {
	return []ent.Index{
		// Primary search indexes
		index.Fields("industry", "country"),
		index.Fields("industry", "country", "city"),
		index.Fields("country", "city"),

		// Filter indexes
		index.Fields("email"),
		index.Fields("phone"),
		index.Fields("verified"),

		// Geographic indexes
		index.Fields("latitude", "longitude"),

		// Quality and uniqueness
		index.Fields("quality_score"),
		index.Fields("osm_id").Unique(),

		// Sub-niche indexes
		index.Fields("industry", "sub_niche"),
		index.Fields("industry", "country", "sub_niche"),
		index.Fields("sub_niche"),
		index.Fields("cuisine_type"),
		index.Fields("sport_type"),
		index.Fields("tattoo_style"),

		// Temporal
		index.Fields("created_at"),
	}
}
