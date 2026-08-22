package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/plexusone/signal-spec/pkg/rootcause"

	"github.com/plexusone/omnisignal/store/sqlite/ent"
	entrootcause "github.com/plexusone/omnisignal/store/sqlite/ent/rootcause"
)

// createVecSchema creates the sqlite-vec vec0 virtual tables. These live
// outside Ent's schema model entirely — Ent has no concept of SQLite virtual
// tables, the same reason aha-studio hand-writes its FTS5 tables alongside
// the Ent-managed ones. Safe to call repeatedly (CREATE ... IF NOT EXISTS).
func (s *Store) createVecSchema(ctx context.Context) error {
	stmts := []string{
		fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS signal_vec USING vec0(embedding float[%d], +signal_id TEXT)`, s.dim),
		fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS rootcause_vec USING vec0(embedding float[%d], +root_cause_id TEXT)`, s.dim),
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// vecLiteral formats an embedding as the JSON array literal vec_f32() expects.
func vecLiteral(embedding []float32) (string, error) {
	b, err := json.Marshal(embedding)
	if err != nil {
		return "", fmt.Errorf("marshaling embedding: %w", err)
	}
	return string(b), nil
}

// upsertRootCauseVec replaces the rootcause_vec row for id, if embedding is
// non-empty. vec0 has no in-place vector UPDATE, so this is delete-then-insert.
func (s *Store) upsertRootCauseVec(ctx context.Context, id string, embedding []float32) error {
	if len(embedding) == 0 {
		return nil
	}
	if len(embedding) != s.dim {
		return fmt.Errorf("%w: got %d, want %d", ErrDimensionMismatch, len(embedding), s.dim)
	}
	lit, err := vecLiteral(embedding)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM rootcause_vec WHERE root_cause_id = ?`, id); err != nil {
		return fmt.Errorf("clearing existing rootcause_vec row: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO rootcause_vec (root_cause_id, embedding) VALUES (?, vec_f32(?))`, id, lit); err != nil {
		return fmt.Errorf("inserting rootcause_vec row: %w", err)
	}
	return nil
}

// upsertSignalVec replaces the signal_vec row for id, if embedding is non-empty.
func (s *Store) upsertSignalVec(ctx context.Context, id string, embedding []float32) error {
	if len(embedding) == 0 {
		return nil
	}
	if len(embedding) != s.dim {
		return fmt.Errorf("%w: got %d, want %d", ErrDimensionMismatch, len(embedding), s.dim)
	}
	lit, err := vecLiteral(embedding)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM signal_vec WHERE signal_id = ?`, id); err != nil {
		return fmt.Errorf("clearing existing signal_vec row: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO signal_vec (signal_id, embedding) VALUES (?, vec_f32(?))`, id, lit); err != nil {
		return fmt.Errorf("inserting signal_vec row: %w", err)
	}
	return nil
}

// NearestRootCauses returns the topK root causes whose embeddings are
// nearest to embedding, ordered nearest-first, via a sqlite-vec KNN query.
// This is the corpus-scale similarity search lever that
// consolidate.Pipeline's in-memory brute-force cosine comparison doesn't
// have; wiring it into Pipeline.Attach's caller is follow-up work, not part
// of this store.
func (s *Store) NearestRootCauses(ctx context.Context, embedding []float32, topK int) ([]rootcause.RootCause, error) {
	if len(embedding) != s.dim {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrDimensionMismatch, len(embedding), s.dim)
	}
	if topK <= 0 {
		return nil, nil
	}
	lit, err := vecLiteral(embedding)
	if err != nil {
		return nil, err
	}

	// Explicit ORDER BY <distance fn> rather than vec0's MATCH ... AND k = ?
	// shorthand: the shorthand's implicit "distance" column reflects
	// whatever metric the vec0 table was created with (L2 by default), not
	// s.metric — this way the configured DistanceMetric is actually honored.
	//nolint:gosec // G201: sqlFunc() returns one of two fixed constants, never user input
	query := fmt.Sprintf(`
		SELECT root_cause_id
		FROM rootcause_vec
		ORDER BY %s(embedding, vec_f32(?))
		LIMIT ?
	`, s.metric.sqlFunc())
	rows, err := s.db.QueryContext(ctx, query, lit, topK)
	if err != nil {
		return nil, fmt.Errorf("knn query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning knn row: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hydrate full rows via Ent, then reorder to match the KNN distance
	// ordering (Ent's IDIn doesn't guarantee result order).
	entRows, err := s.ent.RootCause.Query().Where(entrootcause.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("hydrating root causes: %w", err)
	}
	byID := make(map[string]*ent.RootCause, len(entRows))
	for _, r := range entRows {
		byID[r.ID] = r
	}

	out := make([]rootcause.RootCause, 0, len(ids))
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			continue
		}
		rc := rootCauseFromEnt(r)
		linked, err := s.GetLinkedSignals(ctx, id)
		if err != nil {
			return nil, err
		}
		rc.SignalIDs = linked
		out = append(out, rc)
	}
	return out, nil
}
