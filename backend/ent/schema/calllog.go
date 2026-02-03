package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CallLog holds the schema definition for the CallLog entity.
type CallLog struct {
	ent.Schema
}

// Fields of the CallLog.
func (CallLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who made/received the call"),
		field.Int("lead_id").
			Optional().
			Nillable().
			Comment("Lead associated with the call"),
		field.String("phone_number").
			NotEmpty().
			MaxLen(20).
			Comment("Phone number called (E.164 format)"),
		field.Enum("direction").
			Values("inbound", "outbound").
			Comment("Call direction"),
		field.Enum("status").
			Values("initiated", "ringing", "in_progress", "completed", "failed", "busy", "no_answer", "canceled").
			Default("initiated").
			Comment("Call status"),
		field.Int("duration").
			Default(0).
			NonNegative().
			Comment("Call duration in seconds"),
		field.String("provider_call_id").
			Optional().
			Nillable().
			MaxLen(100).
			Comment("Provider's call ID (Twilio SID, etc.)"),
		field.String("from_number").
			Optional().
			Nillable().
			MaxLen(20).
			Comment("Caller's phone number"),
		field.String("to_number").
			Optional().
			Nillable().
			MaxLen(20).
			Comment("Recipient's phone number"),
		field.Float("cost").
			Default(0.0).
			Min(0).
			Comment("Call cost in USD"),
		field.String("recording_url").
			Optional().
			Nillable().
			Comment("URL to call recording"),
		field.Int("recording_duration").
			Optional().
			Nillable().
			NonNegative().
			Comment("Recording duration in seconds"),
		field.Text("notes").
			Optional().
			Nillable().
			Comment("Call notes added by user"),
		field.String("disposition").
			Optional().
			Nillable().
			MaxLen(50).
			Comment("Call outcome (interested, not_interested, callback, etc.)"),
		field.Bool("is_recorded").
			Default(false).
			Comment("Whether call was recorded"),
		field.Time("started_at").
			Optional().
			Nillable().
			Comment("When call started"),
		field.Time("ended_at").
			Optional().
			Nillable().
			Comment("When call ended"),
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

// Edges of the CallLog.
func (CallLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("call_logs").
			Field("user_id").
			Unique().
			Required().
			Comment("User who made/received this call"),
		edge.From("lead", Lead.Type).
			Ref("call_logs").
			Field("lead_id").
			Unique().
			Comment("Lead associated with this call"),
	}
}

// Indexes of the CallLog.
func (CallLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("lead_id"),
		index.Fields("status"),
		index.Fields("direction"),
		index.Fields("provider_call_id"),
		index.Fields("created_at"),
		index.Fields("started_at"),
	}
}
