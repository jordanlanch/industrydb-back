package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AffiliateConversion holds the schema definition for the AffiliateConversion entity.
type AffiliateConversion struct {
	ent.Schema
}

// Fields of the AffiliateConversion.
func (AffiliateConversion) Fields() []ent.Field {
	return []ent.Field{
		field.Int("affiliate_id").
			Comment("Affiliate who earned this conversion"),
		field.Int("user_id").
			Comment("User who converted"),
		field.String("conversion_type").
			Comment("Type of conversion (registration, subscription, purchase)"),
		field.Float("order_value").
			Default(0.0).
			Comment("Value of the order/transaction"),
		field.Float("commission_amount").
			Comment("Commission earned from this conversion"),
		field.Float("commission_rate").
			Comment("Commission rate at time of conversion"),
		field.Enum("status").
			Values("pending", "approved", "paid", "rejected").
			Default("pending").
			Comment("Payout status"),
		field.String("rejection_reason").
			Optional().
			Nillable().
			Comment("Reason if conversion was rejected"),
		field.Time("approved_at").
			Optional().
			Nillable().
			Comment("When conversion was approved"),
		field.Time("paid_at").
			Optional().
			Nillable().
			Comment("When commission was paid"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Conversion timestamp"),
	}
}

// Edges of the AffiliateConversion.
func (AffiliateConversion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("affiliate", Affiliate.Type).
			Ref("conversions").
			Field("affiliate_id").
			Unique().
			Required().
			Comment("Affiliate who earned this conversion"),
		edge.From("user", User.Type).
			Ref("affiliate_conversions").
			Field("user_id").
			Unique().
			Required().
			Comment("User who converted"),
	}
}

// Indexes of the AffiliateConversion.
func (AffiliateConversion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("affiliate_id"),
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("conversion_type"),
	}
}
