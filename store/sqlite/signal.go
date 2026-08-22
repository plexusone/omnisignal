package sqlite

import (
	"context"
	"fmt"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"

	"github.com/plexusone/omnisignal/store/sqlite/ent"
	entsignal "github.com/plexusone/omnisignal/store/sqlite/ent/signal"
	"github.com/plexusone/omnisignal/store/sqlite/ent/signalrootcauselink"
)

// SaveSignal persists a signal, upserting by ID (the identity a provider
// assigns to the same source event across re-ingestion runs; Fingerprint
// carries its own separate unique index as a content-dedup guard), plus its
// embedding (if set) into signal_vec. Same non-transactional ordering
// rationale as SaveRootCause: the Ent row lands first, so a failed embedding
// write is safe to repair with a retried, idempotent SaveSignal call.
func (s *Store) SaveSignal(ctx context.Context, sig signal.Signal) error {
	err := s.ent.Signal.Create().
		SetID(sig.ID).
		SetType(string(sig.Type)).
		SetStatus(string(sig.Status)).
		SetSourceType(sig.Source.Type).
		SetSourceName(sig.Source.Name).
		SetSourceExternalID(sig.Source.ExternalID).
		SetSourceURL(sig.Source.URL).
		SetDomainName(sig.Domain.Name).
		SetDomainSubdomain(sig.Domain.Subdomain).
		SetDomainTeam(sig.Domain.Team).
		SetSeverity(string(sig.Severity)).
		SetSummary(sig.Summary).
		SetDescription(sig.Description).
		SetEntities(sig.Entities).
		SetObservedAt(sig.ObservedAt).
		SetReceivedAt(sig.ReceivedAt).
		SetNillableFingerprint(nonEmptyPtr(sig.Fingerprint)).
		SetMetadata(sig.Metadata).
		SetDerived(sig.Derived).
		SetTags(sig.Tags).
		OnConflictColumns(entsignal.FieldID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("saving signal %s: %w", sig.ID, err)
	}

	if sig.RootCauseID != "" {
		if err := s.LinkSignal(ctx, sig.ID, sig.RootCauseID); err != nil {
			return err
		}
	}

	return s.upsertSignalVec(ctx, sig.ID, sig.Embedding)
}

// GetSignal retrieves a signal by ID, with RootCauseID populated from the
// link table if one exists.
func (s *Store) GetSignal(ctx context.Context, id string) (signal.Signal, error) {
	r, err := s.ent.Signal.Get(ctx, id)
	if ent.IsNotFound(err) {
		return signal.Signal{}, ErrNotFound
	}
	if err != nil {
		return signal.Signal{}, fmt.Errorf("getting signal %s: %w", id, err)
	}

	sig := signalFromEnt(r)

	link, err := s.ent.SignalRootCauseLink.Query().
		Where(signalrootcauselink.SignalID(id)).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		// no root cause linked yet
	case err != nil:
		return signal.Signal{}, fmt.Errorf("getting root cause link for signal %s: %w", id, err)
	default:
		sig.RootCauseID = link.RootCauseID
	}

	return sig, nil
}

func signalFromEnt(r *ent.Signal) signal.Signal {
	return signal.Signal{
		ID:     r.ID,
		Type:   signal.Type(r.Type),
		Status: signal.Status(r.Status),
		Source: common.SourceSystem{
			Type:       r.SourceType,
			Name:       r.SourceName,
			ExternalID: r.SourceExternalID,
			URL:        r.SourceURL,
		},
		Domain: common.Domain{
			Name:      r.DomainName,
			Subdomain: r.DomainSubdomain,
			Team:      r.DomainTeam,
		},
		Severity:    common.Severity(r.Severity),
		Summary:     r.Summary,
		Description: r.Description,
		Entities:    r.Entities,
		ObservedAt:  r.ObservedAt,
		ReceivedAt:  r.ReceivedAt,
		Fingerprint: ptrToString(r.Fingerprint),
		Metadata:    r.Metadata,
		Derived:     r.Derived,
		Tags:        r.Tags,
	}
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
