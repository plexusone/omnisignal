# OmniSignal — Roadmap

**Initiative:** `INIT-OMNISIGNAL-001`
**Repository:** `github.com/plexusone/omnisignal`
**Status:** Proposed — 0 of 23 items completed

> RMI IDs are stable and permanent. Commits implementing an item carry the trailer `Refs: RMI-OMNISIGNAL-NNN`. Phase status is derived from member RMIs — a phase is complete only when all its required RMIs are complete. This initiative spans two repositories: signal-spec owns the IR (`RMI-SIGNALSPEC-*`, see signal-spec's ROADMAP.md); this file covers the OmniSignal runtime. Signal-spec schema changes always land before the OmniSignal code that depends on them.

## Phase 1 — Conformance and Provider Hardening

**Theme:** Existing providers fully conform to the signal-spec IR and pass schema validation.
**Status:** Proposed — 0 of 5 items completed

- [ ] `RMI-OMNISIGNAL-001` Schema conformance tests: validate Jira and PagerDuty provider output against embedded signal-spec schemas
  - Acceptance: all provider output validates against `signal.schema.json`; failures are test errors, not warnings
- [ ] `RMI-OMNISIGNAL-002` Fingerprint audit: stable, deterministic fingerprints for Jira and PagerDuty signals
  - Depends on: `RMI-OMNISIGNAL-001`
- [ ] `RMI-OMNISIGNAL-003` Cross-repo reference support: populate `customer_ref` / `capability_ref` metadata from provider config mappings
  - Acceptance: typed ID conventions (`customer:acme`, `capability:scim-group-push`) documented and emitted; conventions defined in signal-spec (`RMI-SIGNALSPEC-003`)
- [ ] `RMI-OMNISIGNAL-004` Sentinel error coverage: wrap all auth and rate-limit failures in the shared sentinel errors
- [ ] `RMI-OMNISIGNAL-005` Capabilities accuracy: verify reported `SignalTypes`, batch sizes, and rate limits per provider
  - Depends on: `RMI-OMNISIGNAL-001`

## Phase 2 — Enhancement Signals and Aha Integration

**Theme:** Enhancement requests become first-class signals; Aha Ideas flow into the IR.
**Status:** Proposed — 0 of 4 items completed

- [ ] `RMI-OMNISIGNAL-006` Metadata conventions doc: standard keys for enhancement signals (votes, subscribers, organizations, named customers, opportunities, estimated ARR)
  - Acceptance: conventions published in docs/; requires the `enhancement_request` type in signal-spec (`RMI-SIGNALSPEC-001`) to land first
- [ ] `RMI-OMNISIGNAL-007` Registry support for the external Aha provider: registered as `"aha"`, implemented in `grokify/aha-studio` against the `Provider` interface
  - Depends on: `RMI-OMNISIGNAL-006`
- [ ] `RMI-OMNISIGNAL-008` Raw vs. curated distinction: mark pre-consolidated signals (Aha Ideas) so the consolidation pipeline skips clustering and maps them directly to canonical signals
  - Depends on: `RMI-OMNISIGNAL-006`
- [ ] `RMI-OMNISIGNAL-009` Round-trip validation: Aha Ideas become valid `signal.Signal` values with vote and customer metrics preserved
  - Depends on: `RMI-OMNISIGNAL-007`, `RMI-OMNISIGNAL-008`
  - Acceptance: end-to-end test with recorded Aha fixture data; adapter work itself tracked in aha-studio's roadmap

## Phase 3 — Derived Metrics Engine

**Theme:** Source-independent, explainable scoring over aggregated signals.
**Status:** Proposed — 0 of 5 items completed

- [ ] `RMI-OMNISIGNAL-010` `metrics/` package: formula registry with pluggable `Compute` functions
- [ ] `RMI-OMNISIGNAL-011` Frustration score: weighted signal count x oldest age, default weights per signal source, overridable
  - Depends on: `RMI-OMNISIGNAL-010`
- [ ] `RMI-OMNISIGNAL-012` Momentum (trailing 30-day count), reach (distinct customer refs), and urgency (case count x severity weight) formulas
  - Depends on: `RMI-OMNISIGNAL-010`
- [ ] `RMI-OMNISIGNAL-013` Config loading: per-organization weight overrides via `Config.Options` or a metrics config file
  - Depends on: `RMI-OMNISIGNAL-011`, `RMI-OMNISIGNAL-012`
- [ ] `RMI-OMNISIGNAL-014` Source-independence validation: metrics computed identically for Aha-sourced and support-sourced signal groups; table-driven tests for all formulas
  - Depends on: `RMI-OMNISIGNAL-013`

## Phase 4 — LLM Consolidation and Root Cause Clustering

**Theme:** Many raw signals collapse into few canonical root causes.
**Status:** Proposed — 0 of 5 items completed

- [ ] `RMI-OMNISIGNAL-015` `consolidate/` package: pipeline stages (embed, cluster, summarize, review, attach)
- [ ] `RMI-OMNISIGNAL-016` Embedding via OmniLLM: populate `Signal.Embedding` through the OmniLLM provider interface
  - Depends on: `RMI-OMNISIGNAL-015`
- [ ] `RMI-OMNISIGNAL-017` Semantic clustering: similarity-threshold clustering with incremental attach for new signals
  - Depends on: `RMI-OMNISIGNAL-016`
- [ ] `RMI-OMNISIGNAL-018` Root cause drafting: generate `rootcause.RootCause` candidates with LLM summaries; persist signal-to-root-cause evidence links
  - Depends on: `RMI-OMNISIGNAL-017`
- [ ] `RMI-OMNISIGNAL-019` Human review hooks: optional review queue; unreviewed root causes flagged, not blocked
  - Depends on: `RMI-OMNISIGNAL-018`
  - Acceptance: a corpus of support tickets clusters into root causes whose frustration scores match Phase 3 formulas

## Phase 5 — Market, Competitive, and Analyst Signals

**Theme:** Extend coverage from customer evidence to market evidence.
**Status:** Proposed — 0 of 4 items completed

- [ ] `RMI-OMNISIGNAL-020` MarketSpec reference conventions: `market_ref`, `competitor_ref`, `analyst_report_ref` metadata keys
  - Acceptance: requires signal-spec product signal types (`RMI-SIGNALSPEC-002`) and MarketSpec core schemas (market-spec repo) to land first
- [ ] `RMI-OMNISIGNAL-021` Analyst adapter: ingest analyst findings referencing MarketSpec report entities
  - Depends on: `RMI-OMNISIGNAL-020`
- [ ] `RMI-OMNISIGNAL-022` Competitive adapter: win/loss and competitive gap signals from CRM sources
  - Depends on: `RMI-OMNISIGNAL-020`
- [ ] `RMI-OMNISIGNAL-023` PRISM-Roadmap handoff doc: how each signal type maps to the CSAT / SAM-SOM / TAM pillars
  - Depends on: `RMI-OMNISIGNAL-021`, `RMI-OMNISIGNAL-022`
  - Acceptance: one root cause aggregates support tickets, an Aha idea, a competitive gap, and an analyst finding, each referencing canonical entities
