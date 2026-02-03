package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LeadStatusHistory holds the schema definition for the LeadStatusHistory entity.
type LeadStatusHistory struct {
	ent.Schema
}

// Fields of the LeadStatusHistory.
func (LeadStatusHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int("lead_id").
			Positive().
			Comment("ID of the lead whose status changed"),

		field.Int("user_id").
			Positive().
			Comment("ID of the user who changed the status"),

		field.Enum("old_status").
			Values("new", "contacted", "qualified", "negotiating", "won", "lost", "archived").
			Optional().
			Nillable().
			Comment("Previous status (null for initial status)"),

		field.Enum("new_status").
			Values("new", "contacted", "qualified", "negotiating", "won", "lost", "archived").
			Comment("New status after the change"),

		field.Text("reason").
			Optional().
			MaxLen(1000).
			Comment("Optional reason for status change (e.g., 'Client not interested')"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the status change occurred"),
	}
}

// Edges of the LeadStatusHistory.
func (LeadStatusHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("lead", Lead.Type).
			Ref("status_history").
			Field("lead_id").
			Unique().
			Required(),

		edge.From("user", User.Type).
			Ref("lead_status_changes").
			Field("user_id").
			Unique().
			Required(),
	}
}

// Indexes of the LeadStatusHistory.
func (LeadStatusHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lead_id", "created_at").
			StorageKey("idx_lead_status_history_lead_time"),

		index.Fields("new_status", "created_at").
			StorageKey("idx_lead_status_history_status_time"),

		index.Fields("user_id").
			StorageKey("idx_lead_status_history_user"),
	}
}
