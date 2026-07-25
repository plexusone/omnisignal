# OmniSignal Implementation Plan

## Overview

This plan sequences the work to evolve OmniSignal from its current state (provider interface, registry, Jira and PagerDuty adapters) into the full product signal engine described in [ARCHITECTURE.md](ARCHITECTURE.md) and [PRD.md](PRD.md).

Each phase builds and tests independently. Signal-spec schema changes always land before the OmniSignal code that depends on them.

## Current State

| Component | Status |
|---|---|
| Provider interface (`Fetch`, `Subscribe`, `Capabilities`) | Complete |
| Registry with thin/thick priority | Complete |
| Jira provider | Complete |
| PagerDuty provider | Complete |
| Product signal types (enhancement, competitive, analyst) | Not started |
| Derived metrics engine | Not started |
| LLM consolidation | Not started |

## Phase 1: Align with signal-spec IR and Harden Existing Providers

Goal: existing providers fully conform to the canonical IR and pass schema validation.

| Task | Detail |
|---|---|
| Schema conformance tests | Validate Jira and PagerDuty output against embedded signal-spec schemas |
| Fingerprint audit | Ensure both providers generate stable, deterministic fingerprints |
| Cross-repo reference support | Populate `customer_ref` / `capability_ref` metadata from provider config mappings |
| Sentinel error coverage | Wrap all auth/rate-limit failures in the shared sentinel errors |
| Capabilities accuracy | Verify reported `SignalTypes`, batch sizes, and rate limits per provider |

Exit criteria: all provider output validates against signal-spec; `go test ./...` and `golangci-lint run` clean.

## Phase 2: Enhancement Signals and Aha Integration

Goal: ingest enhancement requests as first-class signals.

| Task | Detail |
|---|---|
| signal-spec: add `enhancement_request` type | New `signal.Type` plus metrics fields (votes, organizations, named customers, estimated ARR) in metadata conventions |
| Aha adapter | Lives in `grokify/aha-studio` (owns the Aha SDK and sync); implements the OmniSignal `Provider` interface and registers as `"aha"` |
| Metadata conventions doc | Define standard metadata keys for enhancement signals (votes, subscribers, organizations, opportunities) |
| Raw vs. curated distinction | Aha Ideas arrive pre-consolidated; mark them so the consolidation pipeline skips clustering and maps them directly to canonical signals |

Exit criteria: Aha Ideas round-trip into valid `signal.Signal` values with vote and customer metrics preserved.

## Phase 3: Derived Metrics Engine

Goal: source-independent scoring over aggregated signals.

| Task | Detail |
|---|---|
| `metrics/` package | Formula registry with pluggable `Compute` functions |
| Frustration score | Weighted signal count x oldest age; default weights per signal source, overridable |
| Momentum | Trailing 30-day signal count |
| Reach | Distinct customer references |
| Urgency | Support case count x severity weight |
| Config loading | Per-organization weight overrides via `Config.Options` or a metrics config file |

Exit criteria: metrics computed identically for Aha-sourced and support-sourced signal groups; formulas covered by table-driven tests.

## Phase 4: LLM Consolidation and Root Cause Clustering

Goal: many raw signals collapse into few canonical root causes.

| Task | Detail |
|---|---|
| `consolidate/` package | Pipeline stages: embed, cluster, summarize, review, attach |
| Embedding via OmniLLM | Populate `Signal.Embedding` through the OmniLLM provider interface |
| Semantic clustering | Similarity-threshold clustering with incremental attach for new signals |
| Root cause drafting | Generate `rootcause.RootCause` candidates with LLM summaries |
| Evidence links | Persist signal-to-root-cause membership for auditability |
| Human review hooks | Optional review queue; unreviewed root causes flagged, not blocked |

Exit criteria: a corpus of support tickets clusters into root causes whose frustration scores match Phase 3 formulas.

## Phase 5: Market, Competitive, and Analyst Signals

Goal: extend coverage from customer evidence to market evidence.

| Task | Detail |
|---|---|
| signal-spec: new types | `competitive_gap`, `competitor_launch`, `analyst_finding`, `market_observation` |
| MarketSpec references | `market_ref`, `competitor_ref`, `analyst_report_ref` metadata conventions |
| Analyst adapter | Ingest analyst findings referencing MarketSpec report entities |
| Competitive adapter | Win/loss and competitive gap signals from CRM sources |
| PRISM-Roadmap handoff | Document how each signal type maps to the CSAT / SAM-SOM / TAM pillars |

Exit criteria: one root cause can aggregate support tickets, an Aha idea, a competitive gap, and an analyst finding, each referencing canonical entities.

## Dependencies

| Dependency | Needed by |
|---|---|
| signal-spec type additions | Phases 2, 5 |
| aha-studio adapter work | Phase 2 |
| OmniLLM embedding/completion providers | Phase 4 |
| MarketSpec core schemas | Phase 5 |

## Working Agreements

- Conventional Commits; one logical change per commit.
- Tests and lint pass before push; schema changes ship with regenerated `.schema.json` files.
- Vendor SDK versions verified against latest releases before adding to `go.mod`.
