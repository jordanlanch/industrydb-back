package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailSequenceStep holds the schema definition for the EmailSequenceStep entity.
type EmailSequenceStep struct {
	ent.Schema
}

// Fields of the EmailSequenceStep.
func (EmailSequenceStep) Fields() []ent.Field {
	return []ent.Field{
		field.Int("sequence_id").
			Positive().
			Comment("Sequence this step belongs to"),

		field.Int("step_order").
			Positive().
			Comment("Order of this step in the sequence (1, 2, 3...)"),

		field.Int("delay_days").
			NonNegative().
			Default(0).
			Comment("Days to wait before sending (0 = send immediately)"),

		field.String("subject").
			NotEmpty().
			MaxLen(500).
			Comment("Email subject line"),

		field.Text("body").
			NotEmpty().
			Comment("Email body (supports variables: {{lead_name}}, {{user_name}}, etc.)"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
	}
}

// Edges of the EmailSequenceStep.
func (EmailSequenceStep) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sequence", EmailSequence.Type).
			Ref("steps").
			Field("sequence_id").
			Unique().
			Required(),

		edge.To("sends", EmailSequenceSend.Type).
			Comment("Emails sent from this step"),
	}
}

// Indexes of the EmailSequenceStep.
func (EmailSequenceStep) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sequence_id", "step_order").
			Unique().
			StorageKey("idx_email_sequence_step_order"),
	}
}
