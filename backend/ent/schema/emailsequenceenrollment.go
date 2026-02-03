package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailSequenceEnrollment holds the schema definition for the EmailSequenceEnrollment entity.
type EmailSequenceEnrollment struct {
	ent.Schema
}

// Fields of the EmailSequenceEnrollment.
func (EmailSequenceEnrollment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("sequence_id").
			Positive().
			Comment("Sequence the lead is enrolled in"),

		field.Int("lead_id").
			Positive().
			Comment("Lead enrolled in this sequence"),

		field.Int("enrolled_by_user_id").
			Positive().
			Comment("User who enrolled this lead"),

		field.Enum("status").
			Values("active", "paused", "completed", "stopped").
			Default("active").
			Comment("Enrollment status"),

		field.Int("current_step").
			NonNegative().
			Default(0).
			Comment("Current step number in the sequence (0 = not started)"),

		field.Time("enrolled_at").
			Default(time.Now).
			Immutable().
			Comment("When the lead was enrolled"),

		field.Time("completed_at").
			Optional().
			Nillable().
			Comment("When the sequence was completed"),

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

// Edges of the EmailSequenceEnrollment.
func (EmailSequenceEnrollment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sequence", EmailSequence.Type).
			Ref("enrollments").
			Field("sequence_id").
			Unique().
			Required(),

		edge.From("lead", Lead.Type).
			Ref("email_sequence_enrollments").
			Field("lead_id").
			Unique().
			Required(),

		edge.From("enrolled_by", User.Type).
			Ref("email_sequence_enrollments_made").
			Field("enrolled_by_user_id").
			Unique().
			Required(),

		edge.To("sends", EmailSequenceSend.Type).
			Comment("Emails sent for this enrollment"),
	}
}

// Indexes of the EmailSequenceEnrollment.
func (EmailSequenceEnrollment) Indexes() []ent.Index {
	return []ent.Index{
		// Find active enrollments for a lead
		index.Fields("lead_id", "status").
			StorageKey("idx_enrollment_lead_status"),

		// Find enrollments for a sequence
		index.Fields("sequence_id", "status").
			StorageKey("idx_enrollment_sequence_status"),

		// Prevent duplicate enrollments (one lead per sequence at a time)
		index.Fields("sequence_id", "lead_id").
			Unique().
			StorageKey("idx_enrollment_unique"),

		index.Fields("enrolled_at").
			StorageKey("idx_enrollment_time"),
	}
}
