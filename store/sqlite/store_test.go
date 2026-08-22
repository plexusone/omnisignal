package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/rootcause"
	"github.com/plexusone/signal-spec/pkg/signal"

	"github.com/plexusone/omnisignal/consolidate"
	sqlitestore "github.com/plexusone/omnisignal/store/sqlite"
)

func openTestStore(t *testing.T) *sqlitestore.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "omnisignal.db")
	s, err := sqlitestore.Open(path, sqlitestore.WithEmbeddingDimension(3))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func testSignal(id, fingerprint string) signal.Signal {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	return signal.Signal{
		ID:     id,
		Type:   "support_ticket",
		Status: "new",
		Source: common.SourceSystem{
			Type:       "ticketing",
			Name:       "zendesk",
			ExternalID: "ZD-100",
			URL:        "https://example.zendesk.com/tickets/100",
		},
		Domain: common.Domain{
			Name:      "authentication",
			Subdomain: "oauth",
			Team:      "identity",
		},
		Severity:    "high",
		Summary:     "Login fails for SSO users",
		Description: "Users report SSO login failures after the last deploy.",
		Entities: []common.Entity{
			{Type: "service", Name: "auth-service", Ref: "capability:sso"},
		},
		ObservedAt:  now,
		ReceivedAt:  now.Add(time.Minute),
		Fingerprint: fingerprint,
		Metadata:    map[string]any{"priority": "urgent"},
		Tags:        []common.Tag{"enterprise", "auth-failure"},
	}
}

func testRootCause(id string) rootcause.RootCause {
	firstSeen := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	return rootcause.RootCause{
		ID:          id,
		Title:       "SSO login failures",
		Description: "OAuth token refresh regressed after the auth-service deploy.",
		Status:      "open",
		Domain: common.Domain{
			Name:      "authentication",
			Subdomain: "oauth",
			Team:      "identity",
		},
		Severity:        "high",
		SymptomPatterns: []string{"login timeout", "invalid token"},
		Impact: rootcause.ImpactMetrics{
			SignalCount:          12,
			AffectedCustomers:    4,
			AffectedEntities:     []common.Entity{{Type: "service", Name: "auth-service"}},
			EscalationRate:       0.25,
			EstimatedRevenueLoss: 1500,
		},
		Trend: rootcause.Trend{
			Direction: "increasing",
			Velocity:  1.5,
			Period: common.TimeRange{
				Start: firstSeen,
				End:   lastSeen,
			},
		},
		PriorityScore:   80,
		FirstSeen:       firstSeen,
		LastSeen:        lastSeen,
		OwnerTeam:       "identity",
		RemediationID:   "REM-1",
		RecurrenceCount: 2,
		Metadata:        map[string]any{"jira": "OPS-42"},
		Tags:            []common.Tag{"enterprise"},
	}
}

func TestOpen_CreateAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "omnisignal.db")

	s1, err := sqlitestore.Open(path, sqlitestore.WithEmbeddingDimension(3))
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	ctx := context.Background()
	if err := s1.SaveRootCause(ctx, testRootCause("RC-1")); err != nil {
		t.Fatalf("SaveRootCause: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := sqlitestore.Open(path, sqlitestore.WithEmbeddingDimension(3))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()

	rc, err := s2.GetRootCause(ctx, "RC-1")
	if err != nil {
		t.Fatalf("GetRootCause after reopen: %v", err)
	}
	if rc.Title != "SSO login failures" {
		t.Errorf("Title = %q, want survived reopen", rc.Title)
	}
}

func TestOpen_RequiresEmbeddingDimension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "omnisignal.db")
	if _, err := sqlitestore.Open(path); err == nil {
		t.Fatal("expected error when embedding dimension is unset")
	}
}

func TestSignal_RoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sig := testSignal("SIG-1", "fp-abc")
	sig.Embedding = []float32{0.1, 0.2, 0.3}
	if err := s.SaveSignal(ctx, sig); err != nil {
		t.Fatalf("SaveSignal: %v", err)
	}

	got, err := s.GetSignal(ctx, "SIG-1")
	if err != nil {
		t.Fatalf("GetSignal: %v", err)
	}

	if got.Summary != sig.Summary {
		t.Errorf("Summary = %q, want %q", got.Summary, sig.Summary)
	}
	if got.Source != sig.Source {
		t.Errorf("Source = %+v, want %+v", got.Source, sig.Source)
	}
	if got.Domain != sig.Domain {
		t.Errorf("Domain = %+v, want %+v", got.Domain, sig.Domain)
	}
	if len(got.Entities) != 1 || got.Entities[0].Type != sig.Entities[0].Type ||
		got.Entities[0].Name != sig.Entities[0].Name || got.Entities[0].Ref != sig.Entities[0].Ref {
		t.Errorf("Entities = %+v, want %+v", got.Entities, sig.Entities)
	}
	if got.Metadata["priority"] != "urgent" {
		t.Errorf("Metadata = %+v, want priority=urgent", got.Metadata)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %+v, want 2 tags", got.Tags)
	}
	if got.Fingerprint != "fp-abc" {
		t.Errorf("Fingerprint = %q, want fp-abc", got.Fingerprint)
	}
}

func TestSignal_GetNotFound(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.GetSignal(context.Background(), "missing"); err != sqlitestore.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSignal_SaveIdempotentByID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sig := testSignal("SIG-2", "fp-dup")
	if err := s.SaveSignal(ctx, sig); err != nil {
		t.Fatalf("first SaveSignal: %v", err)
	}
	sig.Summary = "Updated summary"
	if err := s.SaveSignal(ctx, sig); err != nil {
		t.Fatalf("second SaveSignal: %v", err)
	}

	got, err := s.GetSignal(ctx, "SIG-2")
	if err != nil {
		t.Fatalf("GetSignal: %v", err)
	}
	if got.Summary != "Updated summary" {
		t.Errorf("Summary = %q, want updated in place, not duplicated", got.Summary)
	}
}

func TestRootCause_RoundTrip(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	rc := testRootCause("RC-2")
	rc.Embedding = []float32{0.4, 0.5, 0.6}
	if err := s.SaveRootCause(ctx, rc); err != nil {
		t.Fatalf("SaveRootCause: %v", err)
	}

	got, err := s.GetRootCause(ctx, "RC-2")
	if err != nil {
		t.Fatalf("GetRootCause: %v", err)
	}
	if got.Title != rc.Title || got.Description != rc.Description {
		t.Errorf("got = %+v, want title/description to match %+v", got, rc)
	}
	if got.Impact.SignalCount != rc.Impact.SignalCount ||
		got.Impact.AffectedCustomers != rc.Impact.AffectedCustomers ||
		got.Impact.EscalationRate != rc.Impact.EscalationRate ||
		got.Impact.EstimatedRevenueLoss != rc.Impact.EstimatedRevenueLoss ||
		len(got.Impact.AffectedEntities) != len(rc.Impact.AffectedEntities) {
		t.Errorf("Impact = %+v, want %+v", got.Impact, rc.Impact)
	}
	if got.Trend.Direction != rc.Trend.Direction || got.Trend.Velocity != rc.Trend.Velocity {
		t.Errorf("Trend = %+v, want %+v", got.Trend, rc.Trend)
	}
	if !got.Trend.Period.Start.Equal(rc.Trend.Period.Start) || !got.Trend.Period.End.Equal(rc.Trend.Period.End) {
		t.Errorf("Trend.Period = %+v, want %+v", got.Trend.Period, rc.Trend.Period)
	}
	if got.Metadata["jira"] != "OPS-42" {
		t.Errorf("Metadata = %+v, want jira=OPS-42", got.Metadata)
	}
}

func TestRootCause_GetNotFound(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.GetRootCause(context.Background(), "missing"); err != sqlitestore.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListRootCauses_Filters(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	open1 := testRootCause("RC-OPEN-1")
	open1.Status = "open"
	open1.FirstSeen = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	open2 := testRootCause("RC-OPEN-2")
	open2.Status = "open"
	open2.Domain = common.Domain{Name: "billing", Subdomain: "invoicing"}
	open2.FirstSeen = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	resolved := testRootCause("RC-RESOLVED-1")
	resolved.Status = "resolved"
	resolved.FirstSeen = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	for _, rc := range []rootcause.RootCause{open1, open2, resolved} {
		if err := s.SaveRootCause(ctx, rc); err != nil {
			t.Fatalf("SaveRootCause(%s): %v", rc.ID, err)
		}
	}

	t.Run("status", func(t *testing.T) {
		got, err := s.ListRootCauses(ctx, consolidate.RootCauseFilter{Status: []rootcause.Status{"open"}})
		if err != nil {
			t.Fatalf("ListRootCauses: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d root causes, want 2", len(got))
		}
	})

	t.Run("since", func(t *testing.T) {
		got, err := s.ListRootCauses(ctx, consolidate.RootCauseFilter{Since: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatalf("ListRootCauses: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d root causes since May, want 2 (open2, resolved)", len(got))
		}
	})

	t.Run("domain name only", func(t *testing.T) {
		got, err := s.ListRootCauses(ctx, consolidate.RootCauseFilter{Domain: "authentication"})
		if err != nil {
			t.Fatalf("ListRootCauses: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d root causes for authentication, want 2", len(got))
		}
	})

	t.Run("domain name/subdomain", func(t *testing.T) {
		got, err := s.ListRootCauses(ctx, consolidate.RootCauseFilter{Domain: "billing/invoicing"})
		if err != nil {
			t.Fatalf("ListRootCauses: %v", err)
		}
		if len(got) != 1 || got[0].ID != "RC-OPEN-2" {
			t.Fatalf("got %+v, want only RC-OPEN-2", got)
		}
	})

	t.Run("limit", func(t *testing.T) {
		got, err := s.ListRootCauses(ctx, consolidate.RootCauseFilter{Limit: 1})
		if err != nil {
			t.Fatalf("ListRootCauses: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d root causes, want 1 (limited)", len(got))
		}
	})
}

func TestLinkSignal_ReattachMovesPointer(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	for _, rc := range []string{"RC-A", "RC-B"} {
		if err := s.SaveRootCause(ctx, testRootCause(rc)); err != nil {
			t.Fatalf("SaveRootCause(%s): %v", rc, err)
		}
	}

	if err := s.LinkSignal(ctx, "SIG-X", "RC-A"); err != nil {
		t.Fatalf("LinkSignal(RC-A): %v", err)
	}
	linkedA, err := s.GetLinkedSignals(ctx, "RC-A")
	if err != nil {
		t.Fatalf("GetLinkedSignals(RC-A): %v", err)
	}
	if len(linkedA) != 1 || linkedA[0] != "SIG-X" {
		t.Fatalf("linkedA = %v, want [SIG-X]", linkedA)
	}

	// Reattach to a different root cause (e.g. reject -> re-review).
	if err := s.LinkSignal(ctx, "SIG-X", "RC-B"); err != nil {
		t.Fatalf("LinkSignal(RC-B): %v", err)
	}

	linkedA, err = s.GetLinkedSignals(ctx, "RC-A")
	if err != nil {
		t.Fatalf("GetLinkedSignals(RC-A) after reattach: %v", err)
	}
	if len(linkedA) != 0 {
		t.Errorf("linkedA after reattach = %v, want empty (pointer moved)", linkedA)
	}

	linkedB, err := s.GetLinkedSignals(ctx, "RC-B")
	if err != nil {
		t.Fatalf("GetLinkedSignals(RC-B): %v", err)
	}
	if len(linkedB) != 1 || linkedB[0] != "SIG-X" {
		t.Fatalf("linkedB = %v, want [SIG-X]", linkedB)
	}
}

func TestConsolidateStore_Interface(t *testing.T) {
	var _ consolidate.Store = (*sqlitestore.Store)(nil)
}
