package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CRMIntegration holds the schema definition for the CRMIntegration entity.
type CRMIntegration struct {
	ent.Schema
}

// Fields of the CRMIntegration.
func (CRMIntegration) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who owns this integration"),
		field.Enum("provider").
			Values("salesforce", "hubspot", "pipedrive", "zoho").
			Comment("CRM provider"),
		field.Bool("enabled").
			Default(true).
			Comment("Whether integration is active"),
		field.String("access_token").
			Sensitive().
			Comment("OAuth access token (encrypted)"),
		field.String("refresh_token").
			Optional().
			Sensitive().
			Comment("OAuth refresh token (encrypted)"),
		field.Time("token_expires_at").
			Optional().
			Nillable().
			Comment("When access token expires"),
		field.String("instance_url").
			Optional().
			Comment("CRM instance URL (for Salesforce)"),
		field.String("api_key").
			Optional().
			Sensitive().
			Comment("API key (for providers using API keys)"),
		field.JSON("settings", map[string]interface{}{}).
			Optional().
			Comment("Provider-specific settings"),
		field.Enum("sync_direction").
			Values("bidirectional", "to_crm", "from_crm").
			Default("bidirectional").
			Comment("Data sync direction"),
		field.Bool("auto_sync").
			Default(false).
			Comment("Automatically sync leads to CRM"),
		field.Int("sync_interval_minutes").
			Default(15).
			Positive().
			Comment("Sync interval in minutes (for auto-sync)"),
		field.Time("last_sync_at").
			Optional().
			Nillable().
			Comment("Last successful sync timestamp"),
		field.String("last_sync_error").
			Optional().
			Comment("Last sync error message"),
		field.Int("synced_leads_count").
			Default(0).
			NonNegative().
			Comment("Total leads synced to CRM"),
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

// Edges of the CRMIntegration.
func (CRMIntegration) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("crm_integrations").
			Unique().
			Required().
			Field("user_id").
			Comment("Integration owner"),
		edge.To("synced_leads", CRMLeadSync.Type).
			Comment("Leads synced through this integration"),
	}
}

// Indexes of the CRMIntegration.
func (CRMIntegration) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("provider"),
		index.Fields("enabled"),
		index.Fields("last_sync_at"),
		// Unique: one integration per user per provider
		index.Fields("user_id", "provider").Unique(),
	}
}
