package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MarketReport holds the schema definition for the MarketReport entity.
type MarketReport struct {
	ent.Schema
}

// Fields of the MarketReport.
func (MarketReport) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who requested this report"),
		field.String("title").
			NotEmpty().
			MaxLen(200).
			Comment("Report title"),
		field.String("industry").
			NotEmpty().
			MaxLen(50).
			Comment("Industry covered by this report"),
		field.String("country").
			Optional().
			Nillable().
			MaxLen(2).
			Comment("Country filter (ISO 3166-1 alpha-2)"),
		field.Enum("report_type").
			Values("competitive_analysis", "market_trends", "industry_snapshot", "growth_analysis").
			Comment("Type of market intelligence report"),
		field.JSON("data", map[string]interface{}{}).
			Comment("Report data and statistics"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional report metadata (filters used, generation parameters)"),
		field.Time("period_start").
			Comment("Start of reporting period"),
		field.Time("period_end").
			Comment("End of reporting period"),
		field.Time("generated_at").
			Default(time.Now).
			Immutable().
			Comment("When the report was generated"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("When the report expires (optional)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
	}
}

// Edges of the MarketReport.
func (MarketReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("market_reports").
			Field("user_id").
			Unique().
			Required().
			Comment("User who requested this report"),
	}
}

// Indexes of the MarketReport.
func (MarketReport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("industry"),
		index.Fields("country"),
		index.Fields("report_type"),
		index.Fields("generated_at"),
		index.Fields("created_at"),
	}
}
