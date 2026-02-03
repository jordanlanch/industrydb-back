package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AffiliateClick holds the schema definition for the AffiliateClick entity.
type AffiliateClick struct {
	ent.Schema
}

// Fields of the AffiliateClick.
func (AffiliateClick) Fields() []ent.Field {
	return []ent.Field{
		field.Int("affiliate_id").
			Comment("Affiliate who received the click"),
		field.String("ip_address").
			Optional().
			Nillable().
			Comment("IP address of visitor"),
		field.String("user_agent").
			Optional().
			Nillable().
			Comment("Browser user agent"),
		field.String("referrer").
			Optional().
			Nillable().
			Comment("Referrer URL"),
		field.String("landing_page").
			Optional().
			Nillable().
			Comment("Landing page URL"),
		field.String("utm_source").
			Optional().
			Nillable().
			Comment("UTM source parameter"),
		field.String("utm_medium").
			Optional().
			Nillable().
			Comment("UTM medium parameter"),
		field.String("utm_campaign").
			Optional().
			Nillable().
			Comment("UTM campaign parameter"),
		field.Bool("converted").
			Default(false).
			Comment("Whether this click resulted in a conversion"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Click timestamp"),
	}
}

// Edges of the AffiliateClick.
func (AffiliateClick) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("affiliate", Affiliate.Type).
			Ref("clicks").
			Field("affiliate_id").
			Unique().
			Required().
			Comment("Affiliate who received this click"),
	}
}

// Indexes of the AffiliateClick.
func (AffiliateClick) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("affiliate_id"),
		index.Fields("created_at"),
		index.Fields("converted"),
		index.Fields("ip_address"),
	}
}
