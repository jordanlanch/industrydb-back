package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailSequenceSend holds the schema definition for the EmailSequenceSend entity.
type EmailSequenceSend struct {
	ent.Schema
}

// Fields of the EmailSequenceSend.
func (EmailSequenceSend) Fields() []ent.Field {
	return []ent.Field{
		field.Int("enrollment_id").
			Positive().
			Comment("Enrollment this send belongs to"),

		field.Int("step_id").
			Positive().
			Comment("Sequence step this send is for"),

		field.Int("lead_id").
			Positive().
			Comment("Lead receiving this email"),

		field.Enum("status").
			Values("scheduled", "sent", "opened", "clicked", "bounced", "failed").
			Default("scheduled").
			Comment("Send status"),

		field.Time("scheduled_for").
			Comment("When this email is scheduled to be sent"),

		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("When the email was actually sent"),

		field.Time("opened_at").
			Optional().
			Nillable().
			Comment("When the email was opened (if tracked)"),

		field.Time("clicked_at").
			Optional().
			Nillable().
			Comment("When a link in the email was clicked (if tracked)"),

		field.Bool("bounced").
			Default(false).
			Comment("Whether the email bounced"),

		field.Text("error_message").
			Optional().
			Comment("Error message if send failed"),

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

// Edges of the EmailSequenceSend.
func (EmailSequenceSend) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("enrollment", EmailSequenceEnrollment.Type).
			Ref("sends").
			Field("enrollment_id").
			Unique().
			Required(),

		edge.From("step", EmailSequenceStep.Type).
			Ref("sends").
			Field("step_id").
			Unique().
			Required(),

		edge.From("lead", Lead.Type).
			Ref("email_sequence_sends").
			Field("lead_id").
			Unique().
			Required(),
	}
}

// Indexes of the EmailSequenceSend.
func (EmailSequenceSend) Indexes() []ent.Index {
	return []ent.Index{
		// Find sends for an enrollment
		index.Fields("enrollment_id", "status").
			StorageKey("idx_send_enrollment_status"),

		// Find scheduled sends
		index.Fields("status", "scheduled_for").
			StorageKey("idx_send_scheduled"),

		// Find sends for a lead
		index.Fields("lead_id", "status").
			StorageKey("idx_send_lead_status"),

		// Analytics queries
		index.Fields("sent_at").
			StorageKey("idx_send_time"),

		index.Fields("opened_at").
			StorageKey("idx_send_opened"),
	}
}
