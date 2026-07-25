# OmniSignal Architecture

## Purpose

OmniSignal is the **signal ingestion runtime** in the ProductContext ecosystem. It fetches, normalizes, and streams operational observations from external systems into the canonical `signal.Signal` type defined by [signal-spec](https://github.com/plexusone/signal-spec).

OmniSignal answers: **What signals are we observing?**

## Ecosystem Position

OmniSignal sits between external systems and the decision layer.

```text
External Systems (Jira, PagerDuty, Aha, Zendesk, GitHub, ...)
        |
        v
OmniSignal (ingestion runtime)
        |
        v
signal-spec (canonical data model)
        |
        v
PRISM-Roadmap (prioritization & decisions)
```

### Four-Domain Architecture

| Domain | Owner | Responsibility |
|---|---|---|
| **OrganizationSpec** | ProductBuildersHQ | Customers, prospects, partners, accounts |
| **MarketSpec** | ProductBuildersHQ | Markets, competitors, analyst intelligence, trends, TAM/SAM/SOM |
| **OmniSignal + signal-spec** | PlexusOne | Normalized evidence and observations |
| **PRISM-Roadmap** | ProductBuildersHQ | Strategy, prioritization, investment decisions |

### Three-Layer Model

1. **Entity layer** -- Stable reference data defined by OrganizationSpec and MarketSpec:
   Customer, Market, Competitor, Analyst Report, Product, Capability.

2. **Signal layer** -- Evidence about those entities, defined by signal-spec and ingested by OmniSignal:
   Support ticket, enhancement request, competitive gap, analyst finding, market observation.

3. **Decision layer** -- Interpretation and prioritization in PRISM-Roadmap:
   CSAT contribution, SAM/SOM expansion, TAM expansion, strategic alignment.

## Relationship: OmniSignal vs signal-spec

| Repo | Role | Contains |
|---|---|---|
| **signal-spec** | Canonical data model | Go types (`signal.Signal`, `rootcause.RootCause`, `common.Severity`), JSON schemas, validation |
| **omnisignal** | Ingestion runtime | Provider interface, registry, adapters (Jira, PagerDuty, ...), fetch/subscribe lifecycle |

OmniSignal imports `signal-spec/pkg/signal` and returns `[]signal.Signal` from its `Provider.Fetch()` and `Provider.Subscribe()` methods. The data model is owned by signal-spec; OmniSignal owns the runtime contract for producing those signals.

## Provider Architecture

OmniSignal uses a registry-based provider pattern (same as OmniLLM, OmniVoice, OmniStorage).

```text
omnisignal/
    provider.go       Provider interface, Config, FetchOptions, Capabilities
    registry.go       Register(), New(), List() -- global provider registry

    provider/
        jira/         Jira adapter (thick provider, SDK-based)
        pagerduty/    PagerDuty adapter (thick provider, SDK-based)
```

### Provider Interface

```go
type Provider interface {
    Name() string
    Fetch(ctx context.Context, opts FetchOptions) ([]signal.Signal, error)
    Subscribe(ctx context.Context, opts SubscribeOptions) (<-chan signal.Signal, error)
    Capabilities() Capabilities
    Close() error
}
```

### Registration

Providers self-register via `init()` with a priority level:

- `PriorityThin = 0` -- Native HTTP implementations
- `PriorityThick = 10` -- SDK-based implementations (preferred)

Higher priority wins when multiple implementations exist for the same source.

### Import-Based Activation

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty"
)

provider, err := omnisignal.New("pagerduty", omnisignal.Config{
    APIKey: os.Getenv("PAGERDUTY_API_KEY"),
})
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-24 * time.Hour),
})
```

## Signal Flow

```text
External System
    |
    v
Provider.Fetch() / Provider.Subscribe()
    |
    v
signal.Signal (canonical IR from signal-spec)
    |
    v
Consolidation (LLM clustering, deduplication)
    |
    v
Canonical Signal / Root Cause
    |
    v
Derived Metrics (frustration, momentum, reach)
    |
    v
PRISM-Roadmap (CSAT / SAM-SOM / TAM pillars)
```

## Cross-Repo References

Signals reference entities from MarketSpec and OrganizationSpec via typed IDs in their metadata:

```json
{
  "id": "sig-001",
  "type": "support_ticket",
  "source": {"provider": "jira", "external_id": "SUPPORT-1234"},
  "metadata": {
    "customer_ref": "customer:acme",
    "market_ref": "market:identity-governance",
    "capability_ref": "capability:scim-group-push"
  }
}
```

The canonical entity definitions live in MarketSpec and OrganizationSpec. OmniSignal stores lightweight references, not full copies.

## Consolidation Pipeline

```text
Raw Signals (many)
    |
    v
Embedding (vector representation per signal)
    |
    v
Semantic Clustering (group similar signals)
    |
    v
Root Cause Candidates (cluster summaries)
    |
    v
LLM Summarization (generate canonical description)
    |
    v
Human Review (optional validation)
    |
    v
Canonical Root Cause (mapped in signal-spec)
```

New signals are automatically matched to existing root causes. This is analogous to observability platforms grouping stack traces into incidents.

## Derived Metrics

Computed from aggregated signals, not stored in the IR itself:

| Metric | Formula | Purpose |
|---|---|---|
| Frustration | Weighted signal count x age (days) | How long demand has been waiting |
| Momentum | Signals in last 30 days | Growing interest |
| Revenue | ARR x confidence | Financial impact |
| Reach | Distinct customer accounts | Breadth of demand |
| Urgency | Support cases x severity weight | Pain level |
| Strategic | Vision alignment x weight | Long-term fit |

## Adapter Pattern

Each adapter converts vendor-specific objects to `signal.Signal`:

```text
Jira Issue        -->  signal.Signal{Type: TypeSupportTicket}
PagerDuty Alert   -->  signal.Signal{Type: TypeAlert}
Aha Idea          -->  signal.Signal{Type: TypeFeedback}
Zendesk Ticket    -->  signal.Signal{Type: TypeSupportTicket}
GitHub Issue      -->  signal.Signal{Type: TypeFeedback}
```

Adapters are responsible for:

- Authentication with the external system
- Pagination and rate limiting
- Field mapping to canonical types
- Fingerprint generation for deduplication
