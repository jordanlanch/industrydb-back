package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TerritoryMember holds the schema definition for the TerritoryMember entity.
type TerritoryMember struct {
	ent.Schema
}

// Fields of the TerritoryMember.
func (TerritoryMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int("territory_id").
			Positive().
			Comment("ID of the territory"),

		field.Int("user_id").
			Positive().
			Comment("ID of the user who is a member"),

		field.Enum("role").
			Values("manager", "member").
			Default("member").
			Comment("Role in the territory (manager can manage territory, member can only view/work leads)"),

		field.Time("joined_at").
			Default(time.Now).
			Immutable().
			Comment("When the user joined this territory"),

		field.Int("added_by_user_id").
			Positive().
			Comment("User who added this member to the territory"),
	}
}

// Edges of the TerritoryMember.
func (TerritoryMember) Edges() []ent.Edge {
	return []ent.Edge{
		// Territory
		edge.From("territory", Territory.Type).
			Ref("members").
			Field("territory_id").
			Required().
			Unique(),

		// User (member)
		edge.From("user", User.Type).
			Ref("territory_memberships").
			Field("user_id").
			Required().
			Unique(),

		// Added by
		edge.From("added_by", User.Type).
			Ref("territory_members_added").
			Field("added_by_user_id").
			Required().
			Unique(),
	}
}

// Indexes of the TerritoryMember.
func (TerritoryMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("territory_id", "user_id").Unique(),
		index.Fields("user_id"),
		index.Fields("role"),
	}
}
