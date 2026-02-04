package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailCampaignRecipient holds the schema definition for the EmailCampaignRecipient entity.
type EmailCampaignRecipient struct {
	ent.Schema
}

// Fields of the EmailCampaignRecipient.
func (EmailCampaignRecipient) Fields() []ent.Field {
	return []ent.Field{
		field.Int("campaign_id").
			Comment("Email campaign ID"),
		field.String("email").
			NotEmpty().
			Comment("Recipient email address"),
		field.String("name").
			Optional().
			Comment("Recipient name"),
		field.Enum("status").
			Values("pending", "sent", "failed", "opened", "clicked", "unsubscribed").
			Default("pending").
			Comment("Recipient status"),
		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("When email was sent to this recipient"),
		field.Time("opened_at").
			Optional().
			Nillable().
			Comment("When recipient opened email"),
		field.Time("clicked_at").
			Optional().
			Nillable().
			Comment("When recipient clicked link"),
		field.Time("unsubscribed_at").
			Optional().
			Nillable().
			Comment("When recipient unsubscribed"),
		field.String("failure_reason").
			Optional().
			Comment("Reason for send failure"),
		field.String("sendgrid_message_id").
			Optional().
			Comment("SendGrid message ID for tracking"),
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

// Edges of the EmailCampaignRecipient.
func (EmailCampaignRecipient) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("campaign", EmailCampaign.Type).
			Ref("recipients").
			Unique().
			Required().
			Field("campaign_id").
			Comment("Parent campaign"),
	}
}

// Indexes of the EmailCampaignRecipient.
func (EmailCampaignRecipient) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("campaign_id"),
		index.Fields("email"),
		index.Fields("status"),
		index.Fields("sent_at"),
		index.Fields("created_at"),
		// Composite index for campaign + email lookup
		index.Fields("campaign_id", "email").Unique(),
	}
}
