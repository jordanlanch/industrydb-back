package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LeadAssignment holds the schema definition for the LeadAssignment entity.
type LeadAssignment struct {
	ent.Schema
}

// Fields of the LeadAssignment.
func (LeadAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("lead_id").
			Positive().
			Comment("ID of the assigned lead"),

		field.Int("user_id").
			Positive().
			Comment("ID of the user who owns this lead"),

		field.Int("assigned_by_user_id").
			Optional().
			Nillable().
			Comment("ID of the user who made the assignment (null for automatic assignments)"),

		field.Enum("assignment_type").
			Values("auto", "manual").
			Default("manual").
			Comment("Whether the assignment was automatic or manual"),

		field.String("assignment_reason").
			Optional().
			MaxLen(200).
			Comment("Reason for assignment (e.g., 'round-robin', 'location match', 'manual')"),

		field.Time("assigned_at").
			Default(time.Now).
			Immutable().
			Comment("When the lead was assigned"),

		field.Bool("is_active").
			Default(true).
			Comment("Whether this is the current assignment (false if reassigned)"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
	}
}

// Edges of the LeadAssignment.
func (LeadAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("lead", Lead.Type).
			Ref("assignments").
			Field("lead_id").
			Unique().
			Required(),

		edge.From("user", User.Type).
			Ref("assigned_leads").
			Field("user_id").
			Unique().
			Required(),

		edge.From("assigned_by", User.Type).
			Ref("lead_assignments_made").
			Field("assigned_by_user_id").
			Unique(),
	}
}

// Indexes of the LeadAssignment.
func (LeadAssignment) Indexes() []ent.Index {
	return []ent.Index{
		// Find all leads assigned to a user
		index.Fields("user_id", "is_active").
			StorageKey("idx_lead_assignment_user_active"),

		// Find current assignment for a lead
		index.Fields("lead_id", "is_active").
			StorageKey("idx_lead_assignment_lead_active"),

		// Assignment history
		index.Fields("assigned_at").
			StorageKey("idx_lead_assignment_time"),

		// Assignment type queries
		index.Fields("assignment_type", "assigned_at").
			StorageKey("idx_lead_assignment_type_time"),
	}
}
