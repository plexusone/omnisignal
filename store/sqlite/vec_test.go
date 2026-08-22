package sqlite_test

import (
	"context"
	"testing"

	"github.com/plexusone/signal-spec/pkg/rootcause"
)

func TestNearestRootCauses_KNNOrdering(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	// Three root causes at increasing cosine distance from the query vector
	// [1, 0, 0]: near is nearly identical, mid is orthogonal-ish, far points
	// the opposite direction.
	near := testRootCause("RC-NEAR")
	near.Embedding = []float32{0.99, 0.01, 0.01}

	mid := testRootCause("RC-MID")
	mid.Embedding = []float32{0.1, 0.9, 0.1}

	far := testRootCause("RC-FAR")
	far.Embedding = []float32{-1, 0, 0}

	if err := s.SaveRootCause(ctx, near); err != nil {
		t.Fatalf("SaveRootCause(near): %v", err)
	}
	if err := s.SaveRootCause(ctx, mid); err != nil {
		t.Fatalf("SaveRootCause(mid): %v", err)
	}
	if err := s.SaveRootCause(ctx, far); err != nil {
		t.Fatalf("SaveRootCause(far): %v", err)
	}

	got, err := s.NearestRootCauses(ctx, []float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatalf("NearestRootCauses: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3", len(got))
	}

	wantOrder := []string{"RC-NEAR", "RC-MID", "RC-FAR"}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("result[%d].ID = %q, want %q (full order: %v)", i, got[i].ID, id, idsOf(got))
		}
	}

	if len(got[0].SignalIDs) != 0 {
		t.Errorf("SignalIDs = %v, want empty (no signals linked)", got[0].SignalIDs)
	}
}

func TestNearestRootCauses_TopKLimits(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	for i, emb := range [][]float32{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
		rc := testRootCause(string(rune('A' + i)))
		rc.Embedding = emb
		if err := s.SaveRootCause(ctx, rc); err != nil {
			t.Fatalf("SaveRootCause: %v", err)
		}
	}

	got, err := s.NearestRootCauses(ctx, []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("NearestRootCauses: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2 (topK)", len(got))
	}
}

func idsOf(rcs []rootcause.RootCause) []string {
	ids := make([]string, len(rcs))
	for i, rc := range rcs {
		ids[i] = rc.ID
	}
	return ids
}
