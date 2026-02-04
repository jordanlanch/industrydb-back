package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Experiment holds the schema definition for the Experiment entity.
type Experiment struct {
	ent.Schema
}

// Fields of the Experiment.
func (Experiment) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(255).
			Comment("Experiment name"),
		field.String("key").
			Unique().
			NotEmpty().
			MaxLen(100).
			Comment("Unique key for referencing experiment in code"),
		field.Text("description").
			Optional().
			Comment("Experiment description"),
		field.Enum("status").
			Values("draft", "running", "paused", "completed").
			Default("draft").
			Comment("Experiment status"),
		field.JSON("variants", []string{}).
			Comment("List of variant names (e.g., [control, variant_a, variant_b])"),
		field.JSON("traffic_split", map[string]int{}).
			Comment("Traffic allocation per variant (e.g., {control: 50, variant_a: 50})"),
		field.Time("start_date").
			Optional().
			Nillable().
			Comment("When experiment starts"),
		field.Time("end_date").
			Optional().
			Nillable().
			Comment("When experiment ends"),
		field.String("target_metric").
			Optional().
			Comment("Primary metric to measure (e.g., conversion_rate, revenue)"),
		field.Float("confidence_level").
			Default(0.95).
			Comment("Statistical confidence level (default 95%)"),
		field.Int("min_sample_size").
			Default(100).
			Comment("Minimum users per variant before analysis"),
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

// Edges of the Experiment.
func (Experiment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("assignments", ExperimentAssignment.Type).
			Comment("User assignments to this experiment"),
	}
}

// Indexes of the Experiment.
func (Experiment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key").Unique(),
		index.Fields("status"),
		index.Fields("start_date"),
		index.Fields("end_date"),
	}
}
