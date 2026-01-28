package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SavedSearch holds the schema definition for the SavedSearch entity.
type SavedSearch struct {
	ent.Schema
}

// Fields of the SavedSearch.
func (SavedSearch) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User who created this saved search"),
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("Name/title for this saved search"),
		field.JSON("filters", map[string]interface{}{}).
			Comment("Search filters (industry, country, city, etc.)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When this search was saved"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("Last update timestamp"),
	}
}

// Edges of the SavedSearch.
func (SavedSearch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("saved_searches").
			Field("user_id").
			Unique().
			Required().
			Comment("User who owns this saved search"),
	}
}

// Indexes of the SavedSearch.
func (SavedSearch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("created_at"),
	}
}
