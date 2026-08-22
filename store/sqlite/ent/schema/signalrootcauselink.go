package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SignalRootCauseLink holds the schema definition for the link between a
// Signal and the RootCause it's attached to.
//
// This is a dedicated table rather than a root_cause_id column on Signal
// because consolidate.Pipeline calls Store.LinkSignal(sig.ID, rc.ID)
// (consolidate/consolidate.go processRaw/Attach) without ever calling
// SaveSignal first — a column on Signal would silently no-op the link for
// signals that were never explicitly persisted. The unique index on
// signal_id makes LinkSignal an upsert (insert or re-point on conflict),
// which also models reattachment cleanly: reject -> re-review -> re-link
// moves the pointer instead of accumulating duplicate links.
type SignalRootCauseLink struct {
	ent.Schema
}

// Fields of the SignalRootCauseLink.
func (SignalRootCauseLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("signal_id"),
		field.String("root_cause_id"),
		field.Time("linked_at").Default(time.Now),
	}
}

// Indexes of the SignalRootCauseLink.
func (SignalRootCauseLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("signal_id").Unique(),
		index.Fields("root_cause_id"),
	}
}
