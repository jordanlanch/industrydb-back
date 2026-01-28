package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Subscription holds the schema definition for the Subscription entity.
type Subscription struct {
	ent.Schema
}

// Fields of the Subscription.
func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Positive().
			Comment("User ID foreign key"),
		field.Enum("tier").
			Values("free", "starter", "pro", "business").
			Comment("Subscription tier"),
		field.Enum("status").
			Values("active", "canceled", "past_due", "unpaid", "trialing").
			Default("active").
			Comment("Subscription status"),
		field.String("stripe_subscription_id").
			Optional().
			Comment("Stripe subscription ID"),
		field.String("stripe_price_id").
			Optional().
			Comment("Stripe price ID"),
		field.Time("current_period_start").
			Optional().
			Comment("Current billing period start"),
		field.Time("current_period_end").
			Optional().
			Comment("Current billing period end"),
		field.Bool("cancel_at_period_end").
			Default(false).
			Comment("Whether to cancel at period end"),
		field.Time("canceled_at").
			Optional().
			Nillable().
			Comment("Cancellation timestamp"),
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

// Edges of the Subscription.
func (Subscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("subscriptions").
			Field("user_id").
			Unique().
			Required().
			Comment("Subscription owner"),
	}
}

// Indexes of the Subscription.
func (Subscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("stripe_subscription_id").Unique(),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
