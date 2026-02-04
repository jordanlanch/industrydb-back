package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Territory holds the schema definition for the Territory entity.
type Territory struct {
	ent.Schema
}

// Fields of the Territory.
func (Territory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(200).
			Comment("Territory name (e.g., 'North America', 'EMEA', 'West Coast')"),

		field.Text("description").
			Optional().
			Comment("Description of the territory coverage"),

		field.JSON("countries", []string{}).
			Optional().
			Comment("List of country codes covered by this territory"),

		field.JSON("regions", []string{}).
			Optional().
			Comment("List of regions/states covered (e.g., ['CA', 'OR', 'WA'])"),

		field.JSON("cities", []string{}).
			Optional().
			Comment("Specific cities covered by this territory"),

		field.JSON("industries", []string{}).
			Optional().
			Comment("Industries this territory focuses on"),

		field.Int("created_by_user_id").
			Positive().
			Comment("User who created this territory"),

		field.Bool("active").
			Default(true).
			Comment("Whether this territory is currently active"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the territory was created"),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("When the territory was last updated"),
	}
}

// Edges of the Territory.
func (Territory) Edges() []ent.Edge {
	return []ent.Edge{
		// Creator
		edge.From("created_by", User.Type).
			Ref("territories_created").
			Field("created_by_user_id").
			Required().
			Unique(),

		// Members
		edge.To("members", TerritoryMember.Type),

		// Leads assigned to this territory
		edge.To("leads", Lead.Type),
	}
}

// Indexes of the Territory.
func (Territory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("active"),
		index.Fields("created_by_user_id"),
		index.Fields("created_at"),
	}
}
