package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Organization holds the schema definition for the Organization entity.
type Organization struct {
	ent.Schema
}

// Fields of the Organization.
func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Comment("Organization name"),
		field.String("slug").
			Unique().
			NotEmpty().
			Comment("URL-friendly organization identifier"),
		field.Int("owner_id").
			Comment("User ID of organization owner"),
		field.Enum("subscription_tier").
			Values("free", "starter", "pro", "business").
			Default("free").
			Comment("Organization subscription tier"),
		field.Int("usage_limit").
			Default(50).
			Positive().
			Comment("Monthly usage limit for organization"),
		field.Int("usage_count").
			Default(0).
			NonNegative().
			Comment("Current month usage count"),
		field.Time("last_reset_at").
			Default(time.Now).
			Comment("Last time usage was reset"),
		field.String("stripe_customer_id").
			Optional().
			Nillable().
			Comment("Stripe customer ID for organization billing"),
		field.String("billing_email").
			Optional().
			Nillable().
			Comment("Email for billing notifications"),
		field.Bool("active").
			Default(true).
			Comment("Whether organization is active"),
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

// Edges of the Organization.
func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("owned_organizations").
			Unique().
			Required().
			Field("owner_id").
			Comment("Organization owner"),
		edge.To("members", OrganizationMember.Type).
			Comment("Organization members"),
		edge.To("exports", Export.Type).
			Comment("Organization exports"),
	}
}

// Indexes of the Organization.
func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
		index.Fields("owner_id"),
		index.Fields("subscription_tier"),
		index.Fields("active"),
		index.Fields("created_at"),
	}
}
