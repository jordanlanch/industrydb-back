package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LeadRecommendation holds the schema definition for the LeadRecommendation entity.
type LeadRecommendation struct {
	ent.Schema
}

// Fields of the LeadRecommendation.
func (LeadRecommendation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User receiving this recommendation"),
		field.Int("lead_id").
			Comment("Recommended lead"),
		field.Float("score").
			Min(0).
			Max(100).
			Comment("Recommendation score (0-100)"),
		field.String("reason").
			NotEmpty().
			MaxLen(500).
			Comment("Why this lead is recommended"),
		field.Enum("status").
			Values("pending", "accepted", "rejected", "expired").
			Default("pending").
			Comment("Recommendation status"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional recommendation metadata"),
		field.Time("expires_at").
			Comment("When this recommendation expires"),
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

// Edges of the LeadRecommendation.
func (LeadRecommendation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("lead_recommendations").
			Field("user_id").
			Unique().
			Required().
			Comment("User receiving this recommendation"),
		edge.From("lead", Lead.Type).
			Ref("recommendations").
			Field("lead_id").
			Unique().
			Required().
			Comment("Recommended lead"),
	}
}

// Indexes of the LeadRecommendation.
func (LeadRecommendation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("lead_id"),
		index.Fields("status"),
		index.Fields("score"),
		index.Fields("created_at"),
		index.Fields("expires_at"),
	}
}
