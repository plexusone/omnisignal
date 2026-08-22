package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/plexusone/signal-spec/pkg/common"
)

// RootCause holds the schema definition for the RootCause entity, mirroring
// rootcause.RootCause (github.com/plexusone/signal-spec/pkg/rootcause) minus
// its Embedding field, which lives in the rootcause_vec vec0 table instead
// (see the Signal schema's doc comment for why). SignalIDs is also not
// stored here — it's derived at read time from SignalRootCauseLink so it
// can't drift out of sync with the actual links.
type RootCause struct {
	ent.Schema
}

// Fields of the RootCause.
func (RootCause) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("title"),
		field.String("description").Optional(),
		field.String("status"),

		// Flattened from common.Domain.
		field.String("domain_name"),
		field.String("domain_subdomain").Optional(),
		field.String("domain_team").Optional(),

		field.String("severity"),

		field.JSON("symptom_patterns", []string{}).Optional(),

		// Flattened from rootcause.ImpactMetrics.
		field.Int("impact_signal_count").Default(0),
		field.Int("impact_affected_customers").Default(0),
		field.JSON("impact_affected_entities", []common.Entity{}).Optional(),
		field.Float("impact_escalation_rate").Default(0),
		field.Float("impact_estimated_revenue_loss").Default(0),

		// Flattened from rootcause.Trend.
		field.String("trend_direction").Optional(),
		field.Float("trend_velocity").Default(0),
		field.Time("trend_period_start").Optional().Nillable(),
		field.Time("trend_period_end").Optional().Nillable(),

		field.Int("priority_score").Default(0),

		field.Time("first_seen"),
		field.Time("last_seen"),

		field.String("owner_team").Optional(),
		field.String("remediation_id").Optional(),
		field.Int("recurrence_count").Default(0),

		field.JSON("metadata", map[string]any{}).Optional(),
		field.JSON("tags", []common.Tag{}).Optional(),
	}
}

// Indexes of the RootCause.
func (RootCause) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("domain_name"),
		index.Fields("first_seen"),
		index.Fields("status", "first_seen"),
	}
}
