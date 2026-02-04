package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EmailCampaign holds the schema definition for the EmailCampaign entity.
type EmailCampaign struct {
	ent.Schema
}

// Fields of the EmailCampaign.
func (EmailCampaign) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Comment("Campaign name"),
		field.String("subject").
			NotEmpty().
			Comment("Email subject line"),
		field.Text("content_html").
			Comment("HTML email content"),
		field.Text("content_text").
			Optional().
			Comment("Plain text email content (fallback)"),
		field.Enum("status").
			Values("draft", "scheduled", "sending", "sent", "paused", "failed").
			Default("draft").
			Comment("Campaign status"),
		field.Int("user_id").
			Comment("User who created the campaign"),
		field.String("from_email").
			NotEmpty().
			Comment("Sender email address"),
		field.String("from_name").
			NotEmpty().
			Comment("Sender name"),
		field.String("reply_to").
			Optional().
			Comment("Reply-to email address"),
		field.Time("scheduled_at").
			Optional().
			Nillable().
			Comment("When to send campaign (for scheduled campaigns)"),
		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("When campaign was sent"),
		field.Int("recipients_count").
			Default(0).
			NonNegative().
			Comment("Total number of recipients"),
		field.Int("sent_count").
			Default(0).
			NonNegative().
			Comment("Number of emails successfully sent"),
		field.Int("failed_count").
			Default(0).
			NonNegative().
			Comment("Number of emails that failed"),
		field.Int("opened_count").
			Default(0).
			NonNegative().
			Comment("Number of emails opened"),
		field.Int("clicked_count").
			Default(0).
			NonNegative().
			Comment("Number of links clicked"),
		field.Int("unsubscribed_count").
			Default(0).
			NonNegative().
			Comment("Number of unsubscribes from this campaign"),
		field.JSON("tags", []string{}).
			Optional().
			Comment("Campaign tags for organization"),
		field.String("sendgrid_batch_id").
			Optional().
			Nillable().
			Comment("SendGrid batch ID for tracking"),
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

// Edges of the EmailCampaign.
func (EmailCampaign) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("email_campaigns").
			Unique().
			Required().
			Field("user_id").
			Comment("Campaign creator"),
		edge.To("recipients", EmailCampaignRecipient.Type).
			Comment("Campaign recipients"),
	}
}

// Indexes of the EmailCampaign.
func (EmailCampaign) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("scheduled_at"),
		index.Fields("sent_at"),
		index.Fields("created_at"),
	}
}
