package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

// Signal holds the schema definition for the Signal entity, mirroring
// signal.Signal (github.com/plexusone/signal-spec/pkg/signal) minus its
// Embedding field, which lives in the signal_vec vec0 table instead — Ent
// has no concept of SQLite virtual tables, so the two are kept separate and
// joined on id, the same way aha-studio keeps FTS5 tables alongside Ent.
type Signal struct {
	ent.Schema
}

// Fields of the Signal.
func (Signal) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("type"),
		field.String("status"),

		// Flattened from common.SourceSystem.
		field.String("source_type"),
		field.String("source_name"),
		field.String("source_external_id").Optional(),
		field.String("source_url").Optional(),

		// Flattened from common.Domain.
		field.String("domain_name"),
		field.String("domain_subdomain").Optional(),
		field.String("domain_team").Optional(),

		field.String("severity"),
		field.String("summary"),
		field.String("description").Optional(),

		field.JSON("entities", []common.Entity{}).Optional(),

		field.Time("observed_at"),
		field.Time("received_at"),

		// Dedup key for idempotent ingestion from event sources. Nillable so
		// the unique index below allows multiple signals with no fingerprint
		// (SQLite treats NULL, not "", as distinct across a unique index).
		field.String("fingerprint").Optional().Nillable(),

		field.JSON("metadata", map[string]any{}).Optional(),
		field.JSON("derived", &signal.DerivedMetrics{}).Optional(),
		field.JSON("tags", []common.Tag{}).Optional(),
	}
}

// Indexes of the Signal.
func (Signal) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("fingerprint").Unique(),
		index.Fields("status"),
		index.Fields("domain_name"),
		index.Fields("received_at"),
		index.Fields("status", "domain_name"),
	}
}
