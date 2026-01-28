package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UsageLog holds the schema definition for the UsageLog entity.
type UsageLog struct {
	ent.Schema
}

// Fields of the UsageLog.
func (UsageLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who performed the action"),
		field.Enum("action").
			Values(
				"search",
				"export",
				"api_call",
			).
			Comment("Type of action performed"),
		field.Int("count").
			Default(1).
			Comment("Number of leads accessed/exported"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional context (filters, format, etc.)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Timestamp of action"),
	}
}

// Edges of the UsageLog.
func (UsageLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("usage_logs").
			Field("user_id").
			Unique().
			Required().
			Comment("User who performed the action"),
	}
}

// Indexes of the UsageLog.
func (UsageLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("action"),
		index.Fields("created_at"),
	}
}
