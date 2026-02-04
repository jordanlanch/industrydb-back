package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserBehavior holds the schema definition for the UserBehavior entity.
type UserBehavior struct {
	ent.Schema
}

// Fields of the UserBehavior.
func (UserBehavior) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User performing this action"),
		field.Enum("action_type").
			Values("search", "view", "export", "contact", "save", "filter", "sort").
			Comment("Type of action performed"),
		field.Int("lead_id").
			Optional().
			Nillable().
			Comment("Lead associated with this action (if applicable)"),
		field.String("industry").
			Optional().
			Nillable().
			MaxLen(50).
			Comment("Industry filter used (if search/filter)"),
		field.String("country").
			Optional().
			Nillable().
			MaxLen(2).
			Comment("Country filter used (if search/filter)"),
		field.String("city").
			Optional().
			Nillable().
			MaxLen(100).
			Comment("City filter used (if search/filter)"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional action metadata (filters, sort options, etc.)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When this action occurred"),
	}
}

// Edges of the UserBehavior.
func (UserBehavior) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("behaviors").
			Field("user_id").
			Unique().
			Required().
			Comment("User who performed this action"),
	}
}

// Indexes of the UserBehavior.
func (UserBehavior) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("action_type"),
		index.Fields("lead_id"),
		index.Fields("industry"),
		index.Fields("country"),
		index.Fields("created_at"),
	}
}
