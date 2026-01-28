package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuditLog holds the schema definition for the AuditLog entity.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Optional().
			Nillable().
			Comment("User ID (null for system actions)"),
		field.Enum("action").
			Values(
				"user_login",
				"user_logout",
				"user_register",
				"user_profile_update",
				"user_password_change",
				"user_email_verify",
				"user_account_delete",
				"user_update",
				"user_suspension",
				"data_export",
				"lead_search",
				"lead_view",
				"export_create",
				"export_download",
				"subscription_create",
				"subscription_update",
				"subscription_cancel",
				"payment_success",
				"payment_failed",
				"api_key_create",
				"api_key_delete",
			).
			Comment("Action performed"),
		field.String("resource_type").
			Optional().
			Comment("Type of resource affected (user, lead, export, etc.)"),
		field.String("resource_id").
			Optional().
			Comment("ID of affected resource"),
		field.String("ip_address").
			Optional().
			Comment("IP address of user"),
		field.String("user_agent").
			Optional().
			Comment("User agent string"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Additional context data"),
		field.Enum("severity").
			Values("info", "warning", "error", "critical").
			Default("info").
			Comment("Event severity level"),
		field.String("description").
			Optional().
			Comment("Human-readable description"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Timestamp of event"),
	}
}

// Edges of the AuditLog.
func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("audit_logs").
			Field("user_id").
			Unique().
			Comment("User who performed the action"),
	}
}

// Indexes of the AuditLog.
func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("action"),
		index.Fields("resource_type", "resource_id"),
		index.Fields("created_at"),
		index.Fields("severity"),
	}
}
