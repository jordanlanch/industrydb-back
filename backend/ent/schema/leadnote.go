package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LeadNote holds the schema definition for the LeadNote entity.
type LeadNote struct {
	ent.Schema
}

// Fields of the LeadNote.
func (LeadNote) Fields() []ent.Field {
	return []ent.Field{
		field.Int("lead_id").
			Positive().
			Comment("ID of the lead this note belongs to"),
		field.Int("user_id").
			Positive().
			Comment("ID of the user who created this note"),
		field.Text("content").
			NotEmpty().
			MaxLen(10000).
			Comment("Note content (max 10,000 characters)"),
		field.Bool("is_pinned").
			Default(false).
			Comment("Whether this note is pinned to the top"),
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

// Edges of the LeadNote.
func (LeadNote) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("lead", Lead.Type).
			Ref("notes").
			Field("lead_id").
			Unique().
			Required().
			Comment("Lead this note belongs to"),
		edge.From("user", User.Type).
			Ref("lead_notes").
			Field("user_id").
			Unique().
			Required().
			Comment("User who created this note"),
	}
}

// Indexes of the LeadNote.
func (LeadNote) Indexes() []ent.Index {
	return []ent.Index{
		// Most common query: get all notes for a lead
		index.Fields("lead_id", "created_at"),
		// User activity tracking
		index.Fields("user_id", "created_at"),
		// Pinned notes first
		index.Fields("lead_id", "is_pinned", "created_at"),
	}
}
