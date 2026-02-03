package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			Unique().
			NotEmpty().
			Comment("User email address"),
		field.String("password_hash").
			Sensitive().
			NotEmpty().
			Comment("Bcrypt hashed password"),
		field.String("name").
			NotEmpty().
			Comment("User full name"),
		field.Enum("subscription_tier").
			Values("free", "starter", "pro", "business").
			Default("free").
			Comment("Current subscription tier"),
		field.Enum("role").
			Values("user", "admin", "superadmin").
			Default("user").
			Comment("User role for access control"),
		field.Int("usage_count").
			Default(0).
			NonNegative().
			Comment("Number of leads accessed this month"),
		field.Int("usage_limit").
			Default(50).
			Positive().
			Comment("Monthly usage limit based on tier"),
		field.Time("last_reset_at").
			Default(time.Now).
			Comment("Last time usage was reset"),
		field.Time("last_login_at").
			Optional().
			Nillable().
			Comment("Last login timestamp"),
		field.Bool("email_verified").
			Default(false).
			Comment("Whether email is verified"),
		field.String("email_verification_token").
			Optional().
			Nillable().
			Sensitive().
			Comment("Token for email verification"),
		field.Time("email_verification_token_expires_at").
			Optional().
			Nillable().
			Comment("Expiration time for verification token"),
		field.Time("email_verified_at").
			Optional().
			Nillable().
			Comment("When email was verified"),
		field.Time("accepted_terms_at").
			Optional().
			Nillable().
			Comment("When user accepted Terms of Service and Privacy Policy"),
		field.Bool("onboarding_completed").
			Default(false).
			Comment("Whether user has completed onboarding wizard"),
		field.Bool("totp_enabled").
			Default(false).
			Comment("Whether TOTP two-factor authentication is enabled"),
		field.String("totp_secret").
			Optional().
			Nillable().
			Sensitive().
			Comment("TOTP secret key for 2FA"),
		field.String("oauth_provider").
			Optional().
			Nillable().
			Comment("OAuth provider (google, github, etc.)"),
		field.String("oauth_id").
			Optional().
			Nillable().
			Comment("OAuth provider user ID"),
		field.String("stripe_customer_id").
			Optional().
			Nillable().
			Comment("Stripe customer ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Creation timestamp"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("Last update timestamp"),
		field.Time("deleted_at").
			Optional().
			Nillable().
			Comment("Soft delete timestamp for GDPR compliance"),
		field.Int("onboarding_step").
			Default(0).
			NonNegative().
			Comment("Current onboarding wizard step (0-5, 0=not started)"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subscriptions", Subscription.Type).
			Comment("User's subscription history"),
		edge.To("exports", Export.Type).
			Comment("User's export history"),
		edge.To("api_keys", APIKey.Type).
			Comment("User's API keys"),
		edge.To("audit_logs", AuditLog.Type).
			Comment("User's audit log entries"),
		edge.To("usage_logs", UsageLog.Type).
			Comment("User's usage log entries"),
		edge.To("owned_organizations", Organization.Type).
			Comment("Organizations owned by this user"),
		edge.To("organization_memberships", OrganizationMember.Type).
			Comment("Organization memberships"),
		edge.To("saved_searches", SavedSearch.Type).
			Comment("User's saved searches"),
		edge.To("webhooks", Webhook.Type).
			Comment("User's configured webhooks"),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
		index.Fields("stripe_customer_id"),
		index.Fields("subscription_tier"),
		index.Fields("created_at"),
	}
}
