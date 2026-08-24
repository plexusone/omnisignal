# Persistent Store

The `store/sqlite` package implements [`consolidate.Store`](https://pkg.go.dev/github.com/plexusone/omnisignal/consolidate#Store) on a local SQLite file via [Ent](https://entgo.io/), so signals and root causes survive across runs instead of living only in an in-memory batch. It adds embedding similarity search over the persisted corpus via [sqlite-vec](https://github.com/asg017/sqlite-vec) (`modernc.org/sqlite/vec`) `vec0` virtual tables — both dependencies are pure Go, no cgo required.

## Opening a Store

```go
import "github.com/plexusone/omnisignal/store/sqlite"

st, err := sqlite.Open("omnisignal.db",
    sqlite.WithEmbeddingDimension(1536), // required — matches your embedder's output size
    sqlite.WithDistanceMetric(sqlite.DistanceCosine), // default; also sqlite.DistanceL2
)
if err != nil {
    log.Fatal(err)
}
defer st.Close()
```

`WithEmbeddingDimension` is required: the store has no way to infer the vector length your embedder produces, and `Open` returns `sqlite.ErrDimensionRequired` if it's left unset. `Open` is safe to call repeatedly against an existing, populated file — schema creation is append-only.

## Wiring into the Consolidation Pipeline

`*sqlite.Store` implements `consolidate.Store`, so it plugs directly into a `consolidate.Pipeline` in place of an in-memory store:

```go
pipeline := consolidate.NewPipeline(
    consolidate.WithStore(st),
    consolidate.WithEmbedder(embedder),
    consolidate.WithSummarizer(summarizer),
    consolidate.WithReviewer(reviewer),
)
```

Root causes, signals, and signal-to-root-cause links persist across process restarts, and `Pipeline.Attach` can incrementally attach newly ingested signals to root causes saved in a previous run.

## Signal and Root Cause Persistence

Beyond the `consolidate.Store` methods, the store exposes direct signal access:

```go
err := st.SaveSignal(ctx, sig)     // upsert by Signal.ID; links to sig.RootCauseID if set
got, err := st.GetSignal(ctx, id)  // RootCauseID populated from the link table, if linked
```

`SaveRootCause` / `GetRootCause` / `ListRootCauses` / `LinkSignal` / `GetLinkedSignals` implement the `consolidate.Store` interface directly — see its [godoc](https://pkg.go.dev/github.com/plexusone/omnisignal/consolidate#Store) for the full contract. `LinkSignal` upserts by signal ID, so re-attaching a signal to a different root cause (e.g. after a reject → re-review cycle) moves the link rather than accumulating duplicates.

Embeddings are stored separately from the Ent-managed rows, in `signal_vec` and `rootcause_vec` `vec0` tables — Ent has no concept of SQLite virtual tables. A save writes the Ent row first, then the embedding; a failure partway through is safe to repair with a retried, idempotent `Save*` call rather than requiring a shared transaction across the two layers.

## Similarity Search

```go
neighbors, err := st.NearestRootCauses(ctx, queryEmbedding, 10) // topK=10, nearest first
```

`NearestRootCauses` runs a `vec0` KNN query using the configured `DistanceMetric` (`DistanceCosine` by default, or `DistanceL2`), then hydrates the matching root causes via Ent. This is corpus-scale similarity search that `consolidate.Pipeline`'s in-memory brute-force cosine comparison doesn't provide on its own — useful for deduplicating a candidate root cause against everything already persisted before proposing it as new.

## Design Notes

- **Pure Go, no cgo**: both `modernc.org/sqlite` and `modernc.org/sqlite/vec` are cgo-free, so builds don't need a C toolchain.
- **Signal↔RootCause links use a join table**, not a `root_cause_id` column on `Signal` — `LinkSignal` can be called before `SaveSignal` is ever called for that signal, and a column would silently no-op the link in that ordering.
- **Fingerprint-based dedup** is independent of ID-based upsert: `Signal.Fingerprint` carries its own unique index (nullable, so multiple unset fingerprints are allowed) as a content-dedup guard separate from the provider-assigned `ID`.

See the [`store/sqlite` package](https://pkg.go.dev/github.com/plexusone/omnisignal/store/sqlite) for full API reference.
