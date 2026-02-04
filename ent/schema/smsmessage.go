package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SMSMessage holds the schema definition for the SMSMessage entity.
type SMSMessage struct {
	ent.Schema
}

// Fields of the SMSMessage.
func (SMSMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int("campaign_id").
			Optional().
			Nillable().
			Comment("Campaign this message belongs to"),
		field.Int("lead_id").
			Optional().
			Nillable().
			Comment("Lead this message was sent to"),
		field.String("phone_number").
			NotEmpty().
			MaxLen(20).
			Comment("Recipient phone number (E.164 format)"),
		field.Text("message_body").
			NotEmpty().
			Comment("Actual message content sent"),
		field.String("twilio_sid").
			Optional().
			Nillable().
			MaxLen(100).
			Comment("Twilio message SID"),
		field.Enum("status").
			Values("queued", "sending", "sent", "delivered", "failed", "undelivered").
			Default("queued").
			Comment("Message delivery status"),
		field.String("error_message").
			Optional().
			Nillable().
			Comment("Error message if failed"),
		field.Int("error_code").
			Optional().
			Nillable().
			Comment("Twilio error code if failed"),
		field.Float("cost").
			Default(0.0).
			Min(0).
			Comment("Cost in USD"),
		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("When message was sent"),
		field.Time("delivered_at").
			Optional().
			Nillable().
			Comment("When message was delivered"),
		field.Time("failed_at").
			Optional().
			Nillable().
			Comment("When message failed"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
	}
}

// Edges of the SMSMessage.
func (SMSMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("campaign", SMSCampaign.Type).
			Ref("messages").
			Field("campaign_id").
			Unique().
			Comment("Campaign this message belongs to"),
		edge.From("lead", Lead.Type).
			Ref("sms_messages").
			Field("lead_id").
			Unique().
			Comment("Lead this message was sent to"),
	}
}

// Indexes of the SMSMessage.
func (SMSMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("campaign_id"),
		index.Fields("lead_id"),
		index.Fields("status"),
		index.Fields("twilio_sid"),
		index.Fields("created_at"),
	}
}
