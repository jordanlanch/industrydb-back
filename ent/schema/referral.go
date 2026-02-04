package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Referral holds the schema definition for the Referral entity.
type Referral struct {
	ent.Schema
}

// Fields of the Referral.
func (Referral) Fields() []ent.Field {
	return []ent.Field{
		field.Int("referrer_user_id").
			Comment("User who sent the referral"),

		field.Int("referred_user_id").
			Optional().
			Nillable().
			Comment("User who was referred (null until signup)"),

		field.String("referral_code").
			Unique().
			MaxLen(32).
			Comment("Unique referral code for sharing"),

		field.Enum("status").
			Values("pending", "completed", "rewarded", "expired").
			Default("pending").
			Comment("Current status of the referral"),

		field.Enum("reward_type").
			Values("credit", "discount", "upgrade", "none").
			Default("credit").
			Comment("Type of reward to grant"),

		field.Float("reward_amount").
			Default(0).
			Comment("Monetary reward amount (e.g., $10 credit)"),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the referral code was created"),

		field.Time("completed_at").
			Optional().
			Nillable().
			Comment("When the referred user signed up"),

		field.Time("rewarded_at").
			Optional().
			Nillable().
			Comment("When the reward was granted to referrer"),

		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("When the referral code expires"),
	}
}

// Edges of the Referral.
func (Referral) Edges() []ent.Edge {
	return []ent.Edge{
		// Referrer (user who created the referral)
		edge.From("referrer", User.Type).
			Ref("sent_referrals").
			Field("referrer_user_id").
			Required().
			Unique(),

		// Referred user (who signed up using the code)
		edge.From("referred", User.Type).
			Ref("received_referrals").
			Field("referred_user_id").
			Unique(),
	}
}

// Indexes of the Referral.
func (Referral) Indexes() []ent.Index {
	return []ent.Index{
		// Index on referral_code for fast lookup
		index.Fields("referral_code").
			Unique(),

		// Index on referrer_user_id for listing referrals
		index.Fields("referrer_user_id"),

		// Index on status for filtering
		index.Fields("status"),

		// Composite index for referrer and status
		index.Fields("referrer_user_id", "status"),
	}
}
