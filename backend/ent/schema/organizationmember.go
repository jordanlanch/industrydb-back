package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OrganizationMember holds the schema definition for the OrganizationMember entity.
type OrganizationMember struct {
	ent.Schema
}

// Fields of the OrganizationMember.
func (OrganizationMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int("organization_id").
			Comment("Organization ID"),
		field.Int("user_id").
			Comment("User ID"),
		field.Enum("role").
			Values("owner", "admin", "member", "viewer").
			Default("member").
			Comment("Member role in organization"),
		field.String("invited_by_email").
			Optional().
			Nillable().
			Comment("Email used for invitation"),
		field.Enum("status").
			Values("pending", "active", "suspended").
			Default("active").
			Comment("Member status"),
		field.Time("invited_at").
			Optional().
			Nillable().
			Comment("When invitation was sent"),
		field.Time("joined_at").
			Default(time.Now).
			Comment("When member joined organization"),
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

// Edges of the OrganizationMember.
func (OrganizationMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("members").
			Unique().
			Required().
			Field("organization_id").
			Comment("Organization this member belongs to"),
		edge.From("user", User.Type).
			Ref("organization_memberships").
			Unique().
			Required().
			Field("user_id").
			Comment("User who is a member"),
	}
}

// Indexes of the OrganizationMember.
func (OrganizationMember) Indexes() []ent.Index {
	return []ent.Index{
		// Composite unique index: user can only be in org once
		index.Fields("organization_id", "user_id").Unique(),
		index.Fields("organization_id"),
		index.Fields("user_id"),
		index.Fields("role"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
