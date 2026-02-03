package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ExperimentAssignment holds the schema definition for the ExperimentAssignment entity.
type ExperimentAssignment struct {
	ent.Schema
}

// Fields of the ExperimentAssignment.
func (ExperimentAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("User assigned to variant"),
		field.Int("experiment_id").
			Comment("Experiment this assignment belongs to"),
		field.String("variant").
			NotEmpty().
			Comment("Variant assigned (e.g., control, variant_a)"),
		field.Bool("exposed").
			Default(false).
			Comment("Whether user has been exposed to variant"),
		field.Time("exposed_at").
			Optional().
			Nillable().
			Comment("When user was first exposed"),
		field.Bool("converted").
			Default(false).
			Comment("Whether user converted (for conversion tracking)"),
		field.Time("converted_at").
			Optional().
			Nillable().
			Comment("When user converted"),
		field.Float("metric_value").
			Optional().
			Nillable().
			Comment("Numeric value for target metric (e.g., revenue amount)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Assignment timestamp"),
	}
}

// Edges of the ExperimentAssignment.
func (ExperimentAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("experiment", Experiment.Type).
			Ref("assignments").
			Field("experiment_id").
			Unique().
			Required().
			Comment("Experiment this assignment belongs to"),
		edge.From("user", User.Type).
			Ref("experiment_assignments").
			Field("user_id").
			Unique().
			Required().
			Comment("User assigned to variant"),
	}
}

// Indexes of the ExperimentAssignment.
func (ExperimentAssignment) Indexes() []ent.Index {
	return []ent.Index{
		// Unique constraint: one assignment per user per experiment
		index.Fields("user_id", "experiment_id").Unique(),
		index.Fields("experiment_id", "variant"),
		index.Fields("exposed"),
		index.Fields("converted"),
	}
}
