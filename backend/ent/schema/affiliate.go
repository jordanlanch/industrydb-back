package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Affiliate holds the schema definition for the Affiliate entity.
type Affiliate struct {
	ent.Schema
}

// Fields of the Affiliate.
func (Affiliate) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Unique().
			Comment("User associated with this affiliate account"),
		field.String("affiliate_code").
			Unique().
			NotEmpty().
			MaxLen(32).
			Comment("Unique affiliate tracking code"),
		field.Enum("status").
			Values("pending", "active", "suspended", "terminated").
			Default("pending").
			Comment("Affiliate status"),
		field.Float("commission_rate").
			Default(0.10).
			Comment("Commission rate (0.10 = 10%)"),
		field.Float("total_earnings").
			Default(0.0).
			Comment("Total earnings accumulated"),
		field.Float("pending_earnings").
			Default(0.0).
			Comment("Earnings pending payout"),
		field.Float("paid_earnings").
			Default(0.0).
			Comment("Total earnings paid out"),
		field.Int("total_clicks").
			Default(0).
			Comment("Total clicks on affiliate links"),
		field.Int("total_conversions").
			Default(0).
			Comment("Total conversions from affiliate traffic"),
		field.String("payment_method").
			Optional().
			Comment("Payment method (paypal, bank_transfer, etc.)"),
		field.String("payment_details").
			Optional().
			Sensitive().
			Comment("Payment account details (encrypted)"),
		field.Time("approved_at").
			Optional().
			Nillable().
			Comment("When affiliate was approved"),
		field.Time("last_payout_at").
			Optional().
			Nillable().
			Comment("Last payout date"),
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

// Edges of the Affiliate.
func (Affiliate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("affiliate").
			Field("user_id").
			Unique().
			Required().
			Comment("User who owns this affiliate account"),
		edge.To("clicks", AffiliateClick.Type).
			Comment("Clicks on affiliate links"),
		edge.To("conversions", AffiliateConversion.Type).
			Comment("Conversions from affiliate traffic"),
	}
}

// Indexes of the Affiliate.
func (Affiliate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("affiliate_code").Unique(),
		index.Fields("user_id").Unique(),
		index.Fields("status"),
		index.Fields("total_earnings"),
	}
}
