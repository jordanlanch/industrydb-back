package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CompetitorProfile holds the schema definition for the CompetitorProfile entity.
type CompetitorProfile struct {
	ent.Schema
}

// Fields of the CompetitorProfile.
func (CompetitorProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who added this competitor"),
		field.String("name").
			NotEmpty().
			MaxLen(200).
			Comment("Competitor company name"),
		field.String("website").
			Optional().
			Nillable().
			Comment("Competitor website URL"),
		field.String("industry").
			NotEmpty().
			MaxLen(50).
			Comment("Competitor's primary industry"),
		field.String("country").
			Optional().
			Nillable().
			MaxLen(2).
			Comment("Competitor's primary country (ISO 3166-1 alpha-2)"),
		field.Text("description").
			Optional().
			Nillable().
			Comment("Description of competitor and their offerings"),
		field.Enum("market_position").
			Values("leader", "challenger", "follower", "nicher").
			Optional().
			Nillable().
			Comment("Competitor's market position"),
		field.Int("estimated_employees").
			Optional().
			Nillable().
			NonNegative().
			Comment("Estimated number of employees"),
		field.String("estimated_revenue").
			Optional().
			Nillable().
			MaxLen(50).
			Comment("Estimated annual revenue range"),
		field.JSON("strengths", []string{}).
			Optional().
			Comment("List of competitor strengths"),
		field.JSON("weaknesses", []string{}).
			Optional().
			Comment("List of competitor weaknesses"),
		field.JSON("products", []string{}).
			Optional().
			Comment("List of competitor products/services"),
		field.JSON("pricing_tiers", map[string]interface{}{}).
			Optional().
			Comment("Competitor pricing tiers"),
		field.JSON("target_markets", []string{}).
			Optional().
			Comment("Competitor's target markets"),
		field.String("linkedin_url").
			Optional().
			Nillable().
			Comment("LinkedIn company page URL"),
		field.String("twitter_handle").
			Optional().
			Nillable().
			MaxLen(100).
			Comment("Twitter handle"),
		field.Bool("is_active").
			Default(true).
			Comment("Whether competitor is actively tracked"),
		field.Time("last_analyzed_at").
			Optional().
			Nillable().
			Comment("Last time competitor was analyzed"),
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

// Edges of the CompetitorProfile.
func (CompetitorProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("competitor_profiles").
			Field("user_id").
			Unique().
			Required().
			Comment("User who tracks this competitor"),
		edge.To("metrics", CompetitorMetric.Type).
			Comment("Metrics tracked for this competitor"),
	}
}

// Indexes of the CompetitorProfile.
func (CompetitorProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("industry"),
		index.Fields("country"),
		index.Fields("is_active"),
		index.Fields("created_at"),
	}
}
