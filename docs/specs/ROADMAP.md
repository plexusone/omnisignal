# OmniSignal — Roadmap

**Initiative:** `INIT-OMNISIGNAL-001`
**Repository:** `github.com/plexusone/omnisignal`
**Status:** In progress — 23 of 24 items completed (Phase 6 in progress)

> RMI IDs are stable and permanent. Commits implementing an item carry the trailer `Refs: RMI-OMNISIGNAL-NNN`. Phase status is derived from member RMIs — a phase is complete only when all its required RMIs are complete. This initiative spans two repositories: signal-spec owns the IR (`RMI-SIGNALSPEC-*`, see signal-spec's ROADMAP.md); this file covers the OmniSignal runtime. Signal-spec schema changes always land before the OmniSignal code that depends on them.

## Phase 1 — Conformance and Provider Hardening

**Theme:** Existing providers fully conform to the signal-spec IR and pass schema validation.
**Status:** Completed — 5 of 5 items completed

- [x] `RMI-OMNISIGNAL-001` Schema conformance tests: validate Jira and PagerDuty provider output against embedded signal-spec schemas
  - Acceptance: all provider output validates against `signal.schema.json`; failures are test errors, not warnings
  - Delivered: `provider_conformance_test.go` validates all providers against embedded schemas
- [x] `RMI-OMNISIGNAL-002` Fingerprint audit: stable, deterministic fingerprints for Jira and PagerDuty signals
  - Depends on: `RMI-OMNISIGNAL-001`
  - Delivered: fingerprint tests in `provider_conformance_test.go`; signal-spec `pkg/signal/fingerprint.go`
- [x] `RMI-OMNISIGNAL-003` Cross-repo reference support: populate `customer_ref` / `capability_ref` metadata from provider config mappings
  - Acceptance: typed ID conventions (`customer:acme`, `capability:scim-group-push`) documented and emitted; conventions defined in signal-spec (`RMI-SIGNALSPEC-003`)
  - Delivered: `signal.MetaCustomerRef`, `signal.MetaCapabilityRef` constants; `pkg/ref.TypedRef` type
- [x] `RMI-OMNISIGNAL-004` Sentinel error coverage: wrap all auth and rate-limit failures in the shared sentinel errors
  - Delivered: `ErrAuthFailed`, `ErrRateLimited`, `ErrNotSupported` in `errors.go`
- [x] `RMI-OMNISIGNAL-005` Capabilities accuracy: verify reported `SignalTypes`, batch sizes, and rate limits per provider
  - Depends on: `RMI-OMNISIGNAL-001`
  - Delivered: `Capabilities()` method on all providers; validated in conformance tests

## Phase 2 — Enhancement Signals and Aha Integration

**Theme:** Enhancement requests become first-class signals; Aha Ideas flow into the IR.
**Status:** Completed — 4 of 4 items completed

- [x] `RMI-OMNISIGNAL-006` Metadata conventions doc: standard keys for enhancement signals (votes, subscribers, organizations, named customers, opportunities, estimated ARR)
  - Acceptance: conventions published in docs/; requires the `enhancement_request` type in signal-spec (`RMI-SIGNALSPEC-001`) to land first
  - Delivered: `docs/metadata-conventions.md`; signal-spec `TypeEnhancementRequest` landed
- [x] `RMI-OMNISIGNAL-007` Registry support for the external Aha provider: registered as `"aha"`, implemented in `grokify/aha-studio` against the `Provider` interface
  - Depends on: `RMI-OMNISIGNAL-006`
  - Delivered: `aha-studio/omnisignal/provider.go` implements `omnisignal.Provider`
- [x] `RMI-OMNISIGNAL-008` Raw vs. curated distinction: mark pre-consolidated signals (Aha Ideas) so the consolidation pipeline skips clustering and maps them directly to canonical signals
  - Depends on: `RMI-OMNISIGNAL-006`
  - Delivered: `Signal.Curated` field; `GetBoolOption(OptCurated)` config helper; consolidate pipeline skips clustering for curated signals
- [x] `RMI-OMNISIGNAL-009` Round-trip validation: Aha Ideas become valid `signal.Signal` values with vote and customer metrics preserved
  - Depends on: `RMI-OMNISIGNAL-007`, `RMI-OMNISIGNAL-008`
  - Acceptance: end-to-end test with recorded Aha fixture data; adapter work itself tracked in aha-studio's roadmap
  - Delivered: `aha-studio/omnisignal/provider_test.go`

## Phase 3 — Derived Metrics Engine

**Theme:** Source-independent, explainable scoring over aggregated signals.
**Status:** Completed — 5 of 5 items completed

- [x] `RMI-OMNISIGNAL-010` `metrics/` package: formula registry with pluggable `Compute` functions
  - Delivered: `metrics/metrics.go` with `Registry`, `Formula` interface, `Compute()` method
- [x] `RMI-OMNISIGNAL-011` Frustration score: weighted signal count x oldest age, default weights per signal source, overridable
  - Depends on: `RMI-OMNISIGNAL-010`
  - Delivered: `metrics/frustration.go` with configurable weights
- [x] `RMI-OMNISIGNAL-012` Momentum (trailing 30-day count), reach (distinct customer refs), and urgency (case count x severity weight) formulas
  - Depends on: `RMI-OMNISIGNAL-010`
  - Delivered: `metrics/momentum.go`, `metrics/reach.go`, `metrics/urgency.go`
- [x] `RMI-OMNISIGNAL-013` Config loading: per-organization weight overrides via `Config.Options` or a metrics config file
  - Depends on: `RMI-OMNISIGNAL-011`, `RMI-OMNISIGNAL-012`
  - Delivered: `metrics/config.go` with `LoadConfig()`, weight overrides via options
- [x] `RMI-OMNISIGNAL-014` Source-independence validation: metrics computed identically for Aha-sourced and support-sourced signal groups; table-driven tests for all formulas
  - Depends on: `RMI-OMNISIGNAL-013`
  - Delivered: `metrics/source_independence_test.go` with cross-source validation

## Phase 4 — LLM Consolidation and Root Cause Clustering

**Theme:** Many raw signals collapse into few canonical root causes.
**Status:** Completed — 5 of 5 items completed

- [x] `RMI-OMNISIGNAL-015` `consolidate/` package: pipeline stages (embed, cluster, summarize, review, attach)
  - Delivered: `consolidate/consolidate.go` with `Pipeline` and stage orchestration
- [x] `RMI-OMNISIGNAL-016` Embedding via OmniLLM: populate `Signal.Embedding` through the OmniLLM provider interface
  - Depends on: `RMI-OMNISIGNAL-015`
  - Delivered: `consolidate/embed.go` with `Embedder` interface and OmniLLM integration
- [x] `RMI-OMNISIGNAL-017` Semantic clustering: similarity-threshold clustering with incremental attach for new signals
  - Depends on: `RMI-OMNISIGNAL-016`
  - Delivered: `consolidate/cluster.go` with cosine similarity clustering and incremental attach
- [x] `RMI-OMNISIGNAL-018` Root cause drafting: generate `rootcause.RootCause` candidates with LLM summaries; persist signal-to-root-cause evidence links
  - Depends on: `RMI-OMNISIGNAL-017`
  - Delivered: `consolidate/summarize.go` with LLM-generated root cause summaries
- [x] `RMI-OMNISIGNAL-019` Human review hooks: optional review queue; unreviewed root causes flagged, not blocked
  - Depends on: `RMI-OMNISIGNAL-018`
  - Acceptance: a corpus of support tickets clusters into root causes whose frustration scores match Phase 3 formulas
  - Delivered: `consolidate/review.go` with `ReviewStatus` and optional queue

## Phase 5 — Market, Competitive, and Analyst Signals

**Theme:** Extend coverage from customer evidence to market evidence.
**Status:** Completed — 4 of 4 items completed

- [x] `RMI-OMNISIGNAL-020` MarketSpec reference conventions: `market_ref`, `competitor_ref`, `analyst_report_ref` metadata keys
  - Acceptance: requires signal-spec product signal types (`RMI-SIGNALSPEC-002`) and MarketSpec core schemas (market-spec repo) to land first
  - Delivered: `signal.MetaMarketRef`, `signal.MetaCompetitorRef`, `signal.MetaAnalystReportRef` constants in signal-spec
- [x] `RMI-OMNISIGNAL-021` Analyst adapter: ingest analyst findings referencing MarketSpec report entities
  - Depends on: `RMI-OMNISIGNAL-020`
  - Delivered: `provider/analyst/analyst.go` with `Finding` type and MarketSpec refs
- [x] `RMI-OMNISIGNAL-022` Competitive adapter: win/loss and competitive gap signals from CRM sources
  - Depends on: `RMI-OMNISIGNAL-020`
  - Delivered: `provider/competitive/competitive.go` with `CompetitiveEvent` type
- [x] `RMI-OMNISIGNAL-023` PRISM-Roadmap handoff doc: how each signal type maps to the CSAT / SAM-SOM / TAM pillars
  - Depends on: `RMI-OMNISIGNAL-021`, `RMI-OMNISIGNAL-022`
  - Acceptance: one root cause aggregates support tickets, an Aha idea, a competitive gap, and an analyst finding, each referencing canonical entities
  - Delivered: `docs/prism-roadmap-handoff.md`

## Phase 6 — Durable Signal Store

**Theme:** Signals and root causes persist across runs, with vector similarity search over the corpus.
**Status:** In progress — 0 of 1 items completed

- [ ] `RMI-OMNISIGNAL-024` SQLite/Ent/sqlite-vec persistence store for `consolidate.Store`
  - Acceptance: `store/sqlite` implements `consolidate.Store` (`SaveRootCause`, `GetRootCause`, `ListRootCauses`, `LinkSignal`, `GetLinkedSignals`) against a real SQLite file; `SaveSignal`/`GetSignal` persist full `signal.Signal` records keyed by fingerprint for idempotent ingestion; `NearestRootCauses` performs KNN via sqlite-vec; `go test ./store/sqlite/...` passes against real temp DBs; `go build`, `go vet`, and `golangci-lint` are clean across the repo
