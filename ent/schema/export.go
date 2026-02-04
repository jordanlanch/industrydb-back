package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Export holds the schema definition for the Export entity.
type Export struct {
	ent.Schema
}

// Fields of the Export.
func (Export) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Positive().
			Comment("User ID foreign key"),
		field.Int("organization_id").
			Optional().
			Nillable().
			Comment("Organization ID if export belongs to organization"),
		field.Enum("format").
			Values("csv", "excel").
			Comment("Export format"),
		field.JSON("filters_applied", map[string]interface{}{}).
			Optional().
			Comment("Filters used for this export"),
		field.Int("lead_count").
			NonNegative().
			Comment("Number of leads in export"),
		field.String("file_url").
			Optional().
			Comment("URL to download file"),
		field.String("file_path").
			Optional().
			Comment("Local file path"),
		field.Enum("status").
			Values("pending", "processing", "ready", "failed", "expired").
			Default("pending").
			Comment("Export status"),
		field.String("error_message").
			Optional().
			Comment("Error message if failed"),
		field.Time("expires_at").
			Optional().
			Comment("Expiration timestamp (24h after creation)"),
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

// Edges of the Export.
func (Export) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("exports").
			Field("user_id").
			Unique().
			Required().
			Comment("Export owner"),
		edge.From("organization", Organization.Type).
			Ref("exports").
			Field("organization_id").
			Unique().
			Comment("Organization this export belongs to (optional)"),
	}
}

// Indexes of the Export.
func (Export) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("organization_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("expires_at"),
	}
}
