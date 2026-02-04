package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CRMLeadSync holds the schema definition for the CRMLeadSync entity.
type CRMLeadSync struct {
	ent.Schema
}

// Fields of the CRMLeadSync.
func (CRMLeadSync) Fields() []ent.Field {
	return []ent.Field{
		field.Int("integration_id").
			Comment("CRM integration ID"),
		field.Int("lead_id").
			Comment("Lead ID in IndustryDB"),
		field.String("crm_lead_id").
			Comment("Lead ID in external CRM"),
		field.Enum("sync_status").
			Values("pending", "synced", "failed", "deleted").
			Default("pending").
			Comment("Sync status"),
		field.Enum("sync_direction").
			Values("to_crm", "from_crm").
			Comment("Direction of last sync"),
		field.Time("synced_at").
			Optional().
			Nillable().
			Comment("Last successful sync timestamp"),
		field.String("sync_error").
			Optional().
			Comment("Last sync error message"),
		field.JSON("crm_data", map[string]interface{}{}).
			Optional().
			Comment("CRM-specific data (custom fields, etc.)"),
		field.Bool("auto_update").
			Default(true).
			Comment("Automatically update when lead changes"),
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

// Edges of the CRMLeadSync.
func (CRMLeadSync) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("integration", CRMIntegration.Type).
			Ref("synced_leads").
			Unique().
			Required().
			Field("integration_id").
			Comment("Parent CRM integration"),
		// Note: Lead edge would be added once Lead schema is defined
		// For now, using lead_id as field only
	}
}

// Indexes of the CRMLeadSync.
func (CRMLeadSync) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("integration_id"),
		index.Fields("lead_id"),
		index.Fields("crm_lead_id"),
		index.Fields("sync_status"),
		index.Fields("synced_at"),
		// Unique: one sync record per integration per lead
		index.Fields("integration_id", "lead_id").Unique(),
	}
}
