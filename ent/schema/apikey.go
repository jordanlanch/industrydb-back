package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIKey holds the schema definition for the APIKey entity.
type APIKey struct {
	ent.Schema
}

// Fields of the APIKey.
func (APIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Positive().
			Comment("User ID foreign key"),
		field.String("key_hash").
			Sensitive().
			Unique().
			NotEmpty().
			Comment("SHA256 hash of the API key"),
		field.String("name").
			NotEmpty().
			Comment("Friendly name for the key"),
		field.String("prefix").
			MaxLen(10).
			Comment("First few characters of key (for display)"),
		field.Time("last_used_at").
			Optional().
			Nillable().
			Comment("Last usage timestamp"),
		field.Int("usage_count").
			Default(0).
			NonNegative().
			Comment("Total number of API calls"),
		field.Bool("revoked").
			Default(false).
			Comment("Whether the key has been revoked"),
		field.Time("revoked_at").
			Optional().
			Nillable().
			Comment("Revocation timestamp"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("Optional expiration timestamp"),
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

// Edges of the APIKey.
func (APIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("api_keys").
			Field("user_id").
			Unique().
			Required().
			Comment("API key owner"),
	}
}

// Indexes of the APIKey.
func (APIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("key_hash").Unique(),
		index.Fields("revoked"),
		index.Fields("created_at"),
	}
}
