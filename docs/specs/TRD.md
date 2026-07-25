# OmniSignal Technical Requirements

## Overview

OmniSignal is a Go library implementing the signal ingestion runtime for the ProductContext ecosystem. This document specifies the technical design: package structure, interfaces, the signal IR, adapter implementation, the derived metrics engine, and the LLM consolidation pipeline.

## Module

```text
github.com/plexusone/omnisignal
```

Core dependency: `github.com/plexusone/signal-spec` (canonical data model). The core interface has no other external dependencies; individual providers may depend on vendor SDKs.

## Package Structure

```text
omnisignal/
    provider.go       Provider interface, Config, FetchOptions,
                      SubscribeOptions, Capabilities, sentinel errors
    registry.go       Register(), New(), MustNew(), List(),
                      IsRegistered(), priority handling

    provider/
        jira/         Jira adapter (thick, SDK-based)
        pagerduty/    PagerDuty adapter (thick, SDK-based)
        ...           One package per source system

    metrics/          Derived metrics engine (planned)
    consolidate/      LLM consolidation pipeline (planned)

    internal/         Shared helpers not part of the public API
```

## Provider Interface

The runtime contract every adapter implements:

```go
type Provider interface {
    // Name returns the provider identifier (e.g., "pagerduty", "jira").
    Name() string

    // Fetch retrieves signals matching the given options.
    // Returns signals in chronological order (oldest first).
    // Implementations handle pagination internally.
    Fetch(ctx context.Context, opts FetchOptions) ([]signal.Signal, error)

    // Subscribe opens a real-time stream of signals.
    // Returns ErrNotSupported if streaming is unavailable.
    // The channel is closed when the context is canceled.
    Subscribe(ctx context.Context, opts SubscribeOptions) (<-chan signal.Signal, error)

    // Capabilities returns what this provider supports.
    Capabilities() Capabilities

    // Close releases held resources.
    Close() error
}
```

### Requirements

| ID | Requirement |
|---|---|
| TR-1 | Providers must be safe for concurrent use |
| TR-2 | `Fetch` returns signals oldest-first and handles pagination internally |
| TR-3 | `Subscribe` returns `ErrNotSupported` when streaming is unavailable, never panics |
| TR-4 | Errors wrap the sentinel errors (`ErrAuthentication`, `ErrRateLimited`, ...) for `errors.Is` checks |
| TR-5 | `Capabilities()` accurately reports streaming, filtering, batch limits, and emitted `signal.Type` values |
| TR-6 | Providers never silently discard errors; unreturnable errors are logged via `slog.Logger` from context |

## Registry Pattern

Providers self-register in `init()` with a priority level:

```go
func init() {
    omnisignal.Register("pagerduty", NewProvider, omnisignal.PriorityThick)
}
```

| Priority | Value | Meaning |
|---|---|---|
| `PriorityThin` | 0 | Native HTTP implementation |
| `PriorityThick` | 10 | Official SDK implementation (wins on conflict) |

Callers activate providers by blank import and construct them with `omnisignal.New(name, cfg)`. The registry is a `sync.RWMutex`-guarded map; `Unregister` and `ClearRegistry` exist for tests.

## Signal IR

The IR is owned by signal-spec (`pkg/signal.Signal`). OmniSignal produces it and never redefines it. Key fields adapters must populate:

| Field | Requirement |
|---|---|
| `ID` | Stable, unique per signal (typically derived from source system ID) |
| `Type` | One of `signal.TypeValues()`: `support_ticket`, `cloud_incident`, `security_finding`, `posture_drift`, `alert`, `outage`, `vulnerability`, `feedback` |
| `Source` | `common.SourceSystem` identifying provider and external ID |
| `Severity` | Mapped from source severity to `common.Severity` |
| `Summary` / `Description` | Brief and full content |
| `ObservedAt` / `ReceivedAt` | Source timestamp vs. ingestion timestamp |
| `Fingerprint` | Deterministic hash for deduplication |
| `Metadata` | Source-specific fields plus cross-repo entity references |

New product-signal types (`enhancement_request`, `competitive_gap`, `analyst_finding`, `market_observation`) are added in signal-spec first, then consumed here. Until they exist, enhancement requests map to `TypeFeedback`.

### Cross-Repo References

Entity references use typed IDs in `Metadata`, formatted `<entity-type>:<slug>`:

```json
{
  "metadata": {
    "customer_ref": "customer:acme",
    "market_ref": "market:identity-governance",
    "capability_ref": "capability:scim-group-push"
  }
}
```

| Prefix | Defined by |
|---|---|
| `customer:`, `prospect:`, `partner:` | OrganizationSpec |
| `market:`, `competitor:`, `analyst-report:` | MarketSpec |
| `capability:`, `product:` | ProductContext |

OmniSignal stores references only, never full entity copies. Reference validation against the owning spec is the consumer's responsibility.

## Adapter Implementation Guide

Each adapter package must provide:

1. `NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error)` factory.
2. An `init()` that registers the factory with the appropriate priority.
3. Field mapping from vendor objects to `signal.Signal`, including severity and status normalization.
4. Fingerprint generation (stable hash of source system + external ID + discriminating fields).
5. Internal pagination and rate-limit handling honoring `Capabilities().RateLimitPerMinute`.
6. Table-driven tests using recorded fixtures; no live API calls in unit tests.

Vendor SDK versions must be verified against the latest release before being added to `go.mod`.

## Derived Metrics Engine (planned: `metrics/`)

Metrics are computed over aggregated signals, never stored in the IR. Formulas are configuration, not schema:

```go
type Formula struct {
    Name    string             // "frustration", "momentum", ...
    Weights map[signal.Type]float64
    Compute func(signals []signal.Signal, now time.Time) float64
}
```

| Metric | Formula | Inputs |
|---|---|---|
| Frustration | weighted signal count x oldest age (days) | All signals for a root cause |
| Momentum | signals observed in trailing 30 days | `ObservedAt` |
| Reach | distinct `customer_ref` values | `Metadata` |
| Urgency | support cases x severity weight | `Type`, `Severity` |

Default weights (overridable per organization):

| Signal source | Weight |
|---|---|
| Enterprise support ticket | 4.0 |
| Named customer request | 5.0 |
| Support ticket | 1.0 |
| Prospect request | 2.0 |
| GitHub issue | 0.5 |
| Aha vote | 0.2 |

## Consolidation Pipeline (planned: `consolidate/`)

```text
Raw signals
    -> Embedding (per-signal vector, stored in Signal.Embedding)
    -> Semantic clustering (similarity threshold, configurable)
    -> Root cause candidates (cluster -> rootcause.RootCause draft)
    -> LLM summarization (canonical title + summary)
    -> Optional human review
    -> Canonical root cause; Signal.RootCauseID set, Status -> mapped
```

Technical requirements:

| ID | Requirement |
|---|---|
| TR-10 | Embedding and LLM calls go through OmniLLM provider interfaces, not vendor SDKs directly |
| TR-11 | New signals are matched to existing root causes before creating new clusters |
| TR-12 | Cluster membership and merge decisions are persisted as `signal-spec` evidence links for auditability |
| TR-13 | Pipeline stages are independently runnable (batch embedding, batch clustering, incremental attach) |
| TR-14 | Human review is optional and non-blocking; unreviewed root causes are flagged, not hidden |

## Relationship to signal-spec

| Concern | signal-spec | omnisignal |
|---|---|---|
| `Signal`, `RootCause`, `Remediation` types | Owns | Imports |
| JSON schemas (generated, embedded) | Owns | Conforms to |
| Enum values (`Type`, `Status`, `Severity`) | Owns | Maps vendor values onto |
| Provider interface, registry, adapters | -- | Owns |
| Metrics and consolidation runtime | -- | Owns |

Schema changes follow the Go-first workflow in signal-spec: update Go structs, regenerate schemas with invopop/jsonschema, lint with schemago, embed via `go:embed`.

## Testing and Quality

- `go test ./...` and `golangci-lint run` must pass before push.
- Providers use table-driven tests with fixture data; `t.Fatal` on setup errors.
- Registry tests use `Unregister`/`ClearRegistry` to isolate state.
- Conformance: every provider's output must validate against the embedded signal-spec schema.
