package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Webhook holds the schema definition for the Webhook entity.
type Webhook struct {
	ent.Schema
}

// Fields of the Webhook.
func (Webhook) Fields() []ent.Field {
	return []ent.Field{
		field.String("url").
			NotEmpty().
			Comment("Webhook endpoint URL"),
		field.JSON("events", []string{}).
			Comment("List of events to subscribe to (lead.created, export.completed, etc.)"),
		field.String("secret").
			Sensitive().
			Comment("Secret for HMAC signature verification"),
		field.Bool("active").
			Default(true).
			Comment("Whether webhook is active"),
		field.String("description").
			Optional().
			Comment("User-provided description of webhook"),
		field.Int("retry_count").
			Default(3).
			Comment("Number of retries for failed deliveries"),
		field.Time("last_triggered_at").
			Optional().
			Nillable().
			Comment("Last time webhook was triggered"),
		field.Int("success_count").
			Default(0).
			Comment("Number of successful deliveries"),
		field.Int("failure_count").
			Default(0).
			Comment("Number of failed deliveries"),
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

// Edges of the Webhook.
func (Webhook) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("webhooks").
			Unique().
			Required(),
	}
}

// Indexes of the Webhook.
func (Webhook) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("active"),
		index.Fields("created_at"),
	}
}
