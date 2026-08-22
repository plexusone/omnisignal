// Package sqlite implements consolidate.Store on SQLite via Ent
// (github.com/plexusone/omnisignal/consolidate), plus embedding similarity
// search over the persisted corpus via sqlite-vec
// (modernc.org/sqlite/vec)'s vec0 virtual tables.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"

	"github.com/plexusone/omnisignal/store/sqlite/ent"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

// DistanceMetric selects the sqlite-vec distance function used by
// NearestRootCauses and NearestSignals.
type DistanceMetric string

const (
	// DistanceCosine ranks by cosine distance (the default).
	DistanceCosine DistanceMetric = "cosine"
	// DistanceL2 ranks by Euclidean (L2) distance.
	DistanceL2 DistanceMetric = "l2"
)

func (m DistanceMetric) sqlFunc() string {
	if m == DistanceL2 {
		return "vec_distance_L2"
	}
	return "vec_distance_cosine"
}

// Store persists signals and root causes to a local SQLite file, with
// sqlite-vec-backed similarity search over their embeddings. It implements
// consolidate.Store.
type Store struct {
	db     *sql.DB
	ent    *ent.Client
	dbPath string
	dim    int
	metric DistanceMetric
}

// Option configures a Store.
type Option func(*options)

type options struct {
	dim    int
	metric DistanceMetric
}

// WithEmbeddingDimension sets the fixed vector length stored in the vec0
// tables. Required — Open returns ErrDimensionRequired if unset, since the
// store has no way to infer the dimension of whichever embedder callers use.
func WithEmbeddingDimension(n int) Option {
	return func(o *options) { o.dim = n }
}

// WithDistanceMetric sets the similarity metric used by nearest-neighbor
// queries. Defaults to DistanceCosine.
func WithDistanceMetric(m DistanceMetric) Option {
	return func(o *options) { o.metric = m }
}

// Open opens or creates a SQLite database at dbPath, creating the Ent-managed
// tables and the sqlite-vec vec0 tables if they don't already exist. Safe to
// call repeatedly against an existing, populated file (schema creation is
// append-only, mirroring aha-studio's sync.Open).
func Open(dbPath string, opts ...Option) (*Store, error) {
	o := options{metric: DistanceCosine}
	for _, opt := range opts {
		opt(&o)
	}
	if o.dim <= 0 {
		return nil, ErrDimensionRequired
	}

	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}

	// _pragma=foreign_keys(1): required for Ent's SQLite schema inspector,
	// independent of WithForeignKeys(false) below. modernc.org/sqlite's DSN
	// convention, not the cgo mattn/go-sqlite3 driver's `_fk=1`.
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Wire Ent over the same *sql.DB modernc.org/sqlite opened above — not
	// ent.Open(), which hardcodes the cgo mattn/go-sqlite3 driver name.
	drv := entsql.OpenDB(dialect.SQLite, db)
	entClient := ent.NewClient(ent.Driver(drv))

	s := &Store{db: db, ent: entClient, dbPath: dbPath, dim: o.dim, metric: o.metric}

	ctx := context.Background()
	if err := entClient.Schema.Create(ctx, entschema.WithForeignKeys(false)); err != nil {
		_ = entClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("creating ent schema: %w", err)
	}

	if err := s.createVecSchema(ctx); err != nil {
		_ = entClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("creating vec0 schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection (and the Ent client sharing it).
func (s *Store) Close() error {
	if err := s.ent.Close(); err != nil {
		return err
	}
	return s.db.Close()
}
