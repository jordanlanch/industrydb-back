package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Industry holds the schema definition for the Industry entity.
type Industry struct {
	ent.Schema
}

// Fields of the Industry.
func (Industry) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			NotEmpty().
			Unique().
			Immutable().
			Comment("Unique industry identifier (e.g., 'tattoo', 'beauty')"),
		field.String("name").
			NotEmpty().
			Comment("Display name (e.g., 'Tattoo Studios')"),
		field.String("category").
			NotEmpty().
			Comment("Category group (e.g., 'personal_care', 'health_wellness')"),
		field.String("icon").
			Optional().
			Comment("Icon emoji or identifier"),
		field.String("osm_primary_tag").
			NotEmpty().
			Comment("Primary OpenStreetMap tag (e.g., 'shop=tattoo')"),
		field.JSON("osm_additional_tags", []string{}).
			Optional().
			Comment("Additional OSM tags for this industry"),
		field.String("description").
			Optional().
			Comment("Brief description of the industry"),
		field.Bool("active").
			Default(true).
			Comment("Whether this industry is active/available"),
		field.Int("sort_order").
			Default(0).
			Comment("Display order in UI"),
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

// Edges of the Industry.
func (Industry) Edges() []ent.Edge {
	return nil
}

// Indexes of the Industry.
func (Industry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("category"),
		index.Fields("active"),
		index.Fields("sort_order"),
	}
}
