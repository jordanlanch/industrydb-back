package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailSequence holds the schema definition for the EmailSequence entity.
type EmailSequence struct {
	ent.Schema
}

// Fields of the EmailSequence.
func (EmailSequence) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(200).
			Comment("Sequence name (e.g., 'New Lead Follow-up')"),

		field.Text("description").
			Optional().
			Comment("Description of what this sequence does"),

		field.Enum("status").
			Values("draft", "active", "paused", "archived").
			Default("draft").
			Comment("Sequence status"),

		field.Enum("trigger").
			Values("lead_created", "lead_assigned", "lead_status_changed", "manual").
			Default("manual").
			Comment("What triggers enrollment in this sequence"),

		field.Int("created_by_user_id").
			Positive().
			Comment("User who created this sequence"),

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

// Edges of the EmailSequence.
func (EmailSequence) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("created_by", User.Type).
			Ref("email_sequences_created").
			Field("created_by_user_id").
			Unique().
			Required(),

		edge.To("steps", EmailSequenceStep.Type).
			Comment("Steps in this sequence"),

		edge.To("enrollments", EmailSequenceEnrollment.Type).
			Comment("Leads enrolled in this sequence"),
	}
}

// Indexes of the EmailSequence.
func (EmailSequence) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("trigger"),
		index.Fields("created_by_user_id"),
		index.Fields("created_at"),
	}
}
