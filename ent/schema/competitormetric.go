package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CompetitorMetric holds the schema definition for the CompetitorMetric entity.
type CompetitorMetric struct {
	ent.Schema
}

// Fields of the CompetitorMetric.
func (CompetitorMetric) Fields() []ent.Field {
	return []ent.Field{
		field.Int("competitor_id").
			Comment("Competitor this metric belongs to"),
		field.Enum("metric_type").
			Values("pricing", "features", "market_share", "traffic", "employees", "funding", "reviews", "social_media", "custom").
			Comment("Type of metric being tracked"),
		field.String("metric_name").
			NotEmpty().
			MaxLen(100).
			Comment("Name of the metric"),
		field.Text("metric_value").
			NotEmpty().
			Comment("Value of the metric (stored as text for flexibility)"),
		field.Float("numeric_value").
			Optional().
			Nillable().
			Comment("Numeric representation if applicable"),
		field.String("unit").
			Optional().
			Nillable().
			MaxLen(50).
			Comment("Unit of measurement (USD, users, etc.)"),
		field.Text("notes").
			Optional().
			Nillable().
			Comment("Additional notes about this metric"),
		field.String("source").
			Optional().
			Nillable().
			Comment("Source of the metric data"),
		field.Time("recorded_at").
			Default(time.Now).
			Comment("When this metric was recorded"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
	}
}

// Edges of the CompetitorMetric.
func (CompetitorMetric) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("competitor", CompetitorProfile.Type).
			Ref("metrics").
			Field("competitor_id").
			Unique().
			Required().
			Comment("Competitor this metric belongs to"),
	}
}

// Indexes of the CompetitorMetric.
func (CompetitorMetric) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("competitor_id"),
		index.Fields("metric_type"),
		index.Fields("metric_name"),
		index.Fields("recorded_at"),
		index.Fields("created_at"),
	}
}
