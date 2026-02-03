package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SMSCampaign holds the schema definition for the SMSCampaign entity.
type SMSCampaign struct {
	ent.Schema
}

// Fields of the SMSCampaign.
func (SMSCampaign) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who created the campaign"),
		field.String("name").
			NotEmpty().
			MaxLen(200).
			Comment("Campaign name"),
		field.Text("message_template").
			NotEmpty().
			Comment("SMS message template with placeholders"),
		field.JSON("target_filters", map[string]interface{}{}).
			Optional().
			Comment("Filters for targeting leads (industry, country, etc.)"),
		field.Enum("status").
			Values("draft", "scheduled", "sending", "sent", "failed").
			Default("draft").
			Comment("Campaign status"),
		field.Time("scheduled_at").
			Optional().
			Nillable().
			Comment("When campaign is scheduled to send"),
		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("When campaign sending completed"),
		field.Int("total_recipients").
			Default(0).
			NonNegative().
			Comment("Total number of recipients"),
		field.Int("sent_count").
			Default(0).
			NonNegative().
			Comment("Number of messages sent"),
		field.Int("delivered_count").
			Default(0).
			NonNegative().
			Comment("Number of messages delivered"),
		field.Int("failed_count").
			Default(0).
			NonNegative().
			Comment("Number of messages failed"),
		field.Float("estimated_cost").
			Default(0.0).
			Min(0).
			Comment("Estimated cost in USD"),
		field.Float("actual_cost").
			Default(0.0).
			Min(0).
			Comment("Actual cost in USD"),
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

// Edges of the SMSCampaign.
func (SMSCampaign) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("sms_campaigns").
			Field("user_id").
			Unique().
			Required().
			Comment("User who created this campaign"),
		edge.To("messages", SMSMessage.Type).
			Comment("SMS messages in this campaign"),
	}
}

// Indexes of the SMSCampaign.
func (SMSCampaign) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("scheduled_at"),
	}
}
