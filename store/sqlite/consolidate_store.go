package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/rootcause"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/omnisignal/store/sqlite/ent"
	entrootcause "github.com/plexusone/omnisignal/store/sqlite/ent/rootcause"
	"github.com/plexusone/omnisignal/store/sqlite/ent/signalrootcauselink"
)

var _ consolidate.Store = (*Store)(nil)

// SaveRootCause persists a root cause, upserting by ID, plus its embedding
// (if set) into rootcause_vec. See the package doc on transaction handling:
// the Ent row is written first, so a failure upserting the embedding leaves
// the row intact and safe to repair by calling SaveRootCause again
// (idempotent), rather than risking a half-shared transaction across Ent's
// ORM layer and the hand-written vec0 SQL.
func (s *Store) SaveRootCause(ctx context.Context, rc rootcause.RootCause) error {
	err := s.ent.RootCause.Create().
		SetID(rc.ID).
		SetTitle(rc.Title).
		SetDescription(rc.Description).
		SetStatus(string(rc.Status)).
		SetDomainName(rc.Domain.Name).
		SetDomainSubdomain(rc.Domain.Subdomain).
		SetDomainTeam(rc.Domain.Team).
		SetSeverity(string(rc.Severity)).
		SetSymptomPatterns(rc.SymptomPatterns).
		SetImpactSignalCount(rc.Impact.SignalCount).
		SetImpactAffectedCustomers(rc.Impact.AffectedCustomers).
		SetImpactAffectedEntities(rc.Impact.AffectedEntities).
		SetImpactEscalationRate(rc.Impact.EscalationRate).
		SetImpactEstimatedRevenueLoss(rc.Impact.EstimatedRevenueLoss).
		SetTrendDirection(string(rc.Trend.Direction)).
		SetTrendVelocity(rc.Trend.Velocity).
		SetNillableTrendPeriodStart(zeroTimeToNil(rc.Trend.Period.Start)).
		SetNillableTrendPeriodEnd(zeroTimeToNil(rc.Trend.Period.End)).
		SetPriorityScore(rc.PriorityScore).
		SetFirstSeen(rc.FirstSeen).
		SetLastSeen(rc.LastSeen).
		SetOwnerTeam(rc.OwnerTeam).
		SetRemediationID(rc.RemediationID).
		SetRecurrenceCount(rc.RecurrenceCount).
		SetMetadata(rc.Metadata).
		SetTags(rc.Tags).
		OnConflictColumns(entrootcause.FieldID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("saving root cause %s: %w", rc.ID, err)
	}
	return s.upsertRootCauseVec(ctx, rc.ID, rc.Embedding)
}

// GetRootCause retrieves a root cause by ID, with SignalIDs populated from
// the link table.
func (s *Store) GetRootCause(ctx context.Context, id string) (rootcause.RootCause, error) {
	r, err := s.ent.RootCause.Get(ctx, id)
	if ent.IsNotFound(err) {
		return rootcause.RootCause{}, ErrNotFound
	}
	if err != nil {
		return rootcause.RootCause{}, fmt.Errorf("getting root cause %s: %w", id, err)
	}
	rc := rootCauseFromEnt(r)
	linked, err := s.GetLinkedSignals(ctx, id)
	if err != nil {
		return rootcause.RootCause{}, err
	}
	rc.SignalIDs = linked
	return rc, nil
}

// ListRootCauses returns root causes matching filter, ordered by FirstSeen
// descending (most recent first).
func (s *Store) ListRootCauses(ctx context.Context, filter consolidate.RootCauseFilter) ([]rootcause.RootCause, error) {
	q := s.ent.RootCause.Query()

	if len(filter.Status) > 0 {
		statuses := make([]string, len(filter.Status))
		for i, st := range filter.Status {
			statuses[i] = string(st)
		}
		q = q.Where(entrootcause.StatusIn(statuses...))
	}
	if !filter.Since.IsZero() {
		q = q.Where(entrootcause.FirstSeenGTE(filter.Since))
	}
	if filter.Domain != "" {
		// Accept both "name" and "name/subdomain" forms, matching the split
		// convention consolidate/summarize.go already uses for domain strings.
		name, subdomain, hasSub := splitDomain(filter.Domain)
		if hasSub {
			q = q.Where(entrootcause.DomainNameEQ(name), entrootcause.DomainSubdomainEQ(subdomain))
		} else {
			q = q.Where(entrootcause.DomainNameEQ(name))
		}
	}
	q = q.Order(ent.Desc(entrootcause.FieldFirstSeen))
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing root causes: %w", err)
	}

	out := make([]rootcause.RootCause, 0, len(rows))
	for _, r := range rows {
		rc := rootCauseFromEnt(r)
		linked, err := s.GetLinkedSignals(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		rc.SignalIDs = linked
		out = append(out, rc)
	}
	return out, nil
}

// LinkSignal associates signalID with rootCauseID, upserting by signalID so
// a signal reattached to a different root cause (e.g. after reject ->
// re-review) moves the pointer instead of accumulating duplicate links.
func (s *Store) LinkSignal(ctx context.Context, signalID, rootCauseID string) error {
	err := s.ent.SignalRootCauseLink.Create().
		SetSignalID(signalID).
		SetRootCauseID(rootCauseID).
		SetLinkedAt(time.Now()).
		OnConflictColumns(signalrootcauselink.FieldSignalID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("linking signal %s to root cause %s: %w", signalID, rootCauseID, err)
	}
	return nil
}

// GetLinkedSignals returns the signal IDs currently linked to rootCauseID.
func (s *Store) GetLinkedSignals(ctx context.Context, rootCauseID string) ([]string, error) {
	ids, err := s.ent.SignalRootCauseLink.Query().
		Where(signalrootcauselink.RootCauseID(rootCauseID)).
		Select(signalrootcauselink.FieldSignalID).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting linked signals for %s: %w", rootCauseID, err)
	}
	return ids, nil
}

func rootCauseFromEnt(r *ent.RootCause) rootcause.RootCause {
	return rootcause.RootCause{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Status:      rootcause.Status(r.Status),
		Domain: common.Domain{
			Name:      r.DomainName,
			Subdomain: r.DomainSubdomain,
			Team:      r.DomainTeam,
		},
		Severity:        common.Severity(r.Severity),
		SymptomPatterns: r.SymptomPatterns,
		Impact: rootcause.ImpactMetrics{
			SignalCount:          r.ImpactSignalCount,
			AffectedCustomers:    r.ImpactAffectedCustomers,
			AffectedEntities:     r.ImpactAffectedEntities,
			EscalationRate:       r.ImpactEscalationRate,
			EstimatedRevenueLoss: r.ImpactEstimatedRevenueLoss,
		},
		Trend: rootcause.Trend{
			Direction: rootcause.TrendDirection(r.TrendDirection),
			Velocity:  r.TrendVelocity,
			Period: common.TimeRange{
				Start: nilTimeToZero(r.TrendPeriodStart),
				End:   nilTimeToZero(r.TrendPeriodEnd),
			},
		},
		PriorityScore:   r.PriorityScore,
		FirstSeen:       r.FirstSeen,
		LastSeen:        r.LastSeen,
		OwnerTeam:       r.OwnerTeam,
		RemediationID:   r.RemediationID,
		RecurrenceCount: r.RecurrenceCount,
		Metadata:        r.Metadata,
		Tags:            r.Tags,
	}
}

func zeroTimeToNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func nilTimeToZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// splitDomain splits a "name" or "name/subdomain" filter string, matching
// the convention already used by consolidate/summarize.go for domain strings.
func splitDomain(s string) (name, subdomain string, hasSub bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
