# Competitive Provider

The Competitive provider ingests win/loss data and competitive capability gaps from CRM and competitive intelligence systems (Salesforce, HubSpot, Clari, Gong) and normalizes them to signals that reference [MarketSpec](https://github.com/ProductBuildersHQ/market-spec) entities.

## Installation

The Competitive provider is built into omnisignal:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/competitive"
)
```

## Configuration

```go
provider, err := omnisignal.New("competitive", omnisignal.Config{
    Options: map[string]any{
        "source": "salesforce", // or "hubspot", "clari", "gong", "custom"
    },
})
```

### Options

| Field | Required | Description |
|-------|----------|-------------|
| `Options["source"]` | No | CRM/CI system identifier: `salesforce`, `hubspot`, `clari`, `gong`, or `custom` (default) |
| `Options["adapter"]` | No | A custom `competitive.Adapter` implementation. Defaults to an empty `MemoryAdapter` |
| `Options["competitor_mappings"]` | No | `map[string]string` mapping CRM competitor names to `competitor:` refs |
| `Options[omnisignal.OptCustomerMappings]` | No | `map[string]string` mapping account names to `customer:` refs |
| `Options[omnisignal.OptMarketMappings]` | No | `map[string]string` mapping CRM categories to `market:` refs |

The default in-process adapter is `competitive.MemoryAdapter`. Production integrations should implement the `competitive.Adapter` interface against a CRM API or export and pass it via `Options["adapter"]`.

## Adapter Interface

```go
type Adapter interface {
    // FetchDeals retrieves win/loss records.
    FetchDeals(ctx context.Context, opts omnisignal.FetchOptions) ([]DealRecord, error)

    // FetchGaps retrieves competitive gap signals.
    FetchGaps(ctx context.Context, opts omnisignal.FetchOptions) ([]CompetitiveGap, error)

    // Source returns the adapter's source identifier.
    Source() string
}
```

`Provider.Fetch` calls both `FetchDeals` and `FetchGaps` and returns the combined signal set.

## Data Types

### DealRecord

Represents a win/loss record from CRM:

```go
type DealRecord struct {
    ID                string          // Unique identifier for this record
    OpportunityID     string          // CRM opportunity identifier
    AccountName       string          // Customer/prospect name
    AccountRef        string          // MarketSpec customer ref (e.g., "customer:acme-001")
    Outcome           Outcome         // "win" or "loss"
    Competitors       []string        // Competitors involved in the deal
    PrimaryCompetitor string          // Main competitor (for losses, who won)
    Reasons           []string        // Factors that led to the outcome
    DealValue         int64           // Opportunity amount in cents
    Market            string          // MarketSpec market slug
    ClosedAt          time.Time       // When the deal closed
    Source            string          // CRM system
    Metadata          map[string]any  // Additional source-specific fields
}
```

`Outcome` is one of `competitive.OutcomeWin` or `competitive.OutcomeLoss`.

### CompetitiveGap

Represents a capability gap identified through competitive analysis:

```go
type CompetitiveGap struct {
    ID           string          // Unique identifier
    Title        string          // Describes the gap
    Description  string          // Detail about the gap
    Competitor   string          // Competitor with the advantage
    Capability   string          // Specific feature or capability lacking
    Impact       string          // Business impact (e.g., "lost 3 deals worth $500K")
    Severity     common.Severity // How critical the gap is
    Market       string          // MarketSpec market slug
    DealIDs      []string        // Related deal records
    IdentifiedAt time.Time       // When the gap was identified
    Source       string          // Where this gap was identified
    Metadata     map[string]any  // Additional fields
}
```

## Usage

### Manual Ingestion via MemoryAdapter

```go
adapter := competitive.NewMemoryAdapter("salesforce")
adapter.AddDeal(competitive.DealRecord{
    ID:                "opp-1042",
    OpportunityID:      "006XXXXXXXXXXXX",
    AccountName:        "Acme Corp",
    AccountRef:         "customer:acme-001",
    Outcome:            competitive.OutcomeLoss,
    PrimaryCompetitor:  "okta",
    Reasons:            []string{"missing native FIDO2 support"},
    DealValue:          250000_00,
    Market:             "identity-governance",
    ClosedAt:           time.Now(),
})
adapter.AddGap(competitive.CompetitiveGap{
    ID:         "gap-passwordless",
    Title:      "No native FIDO2/WebAuthn support",
    Competitor: "okta",
    Capability: "passwordless-auth",
    Impact:     "lost 3 deals worth $500K in Q2",
    Severity:   common.SeverityHigh,
    Market:     "identity-governance",
    DealIDs:    []string{"opp-1042"},
})

provider, err := omnisignal.New("competitive", omnisignal.Config{
    Options: map[string]any{
        "source":  "salesforce",
        "adapter": adapter,
    },
})

signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{})
```

### Fetch with Time Filters

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-90 * 24 * time.Hour),
    Limit: 100,
})
```

`MemoryAdapter.FetchDeals` filters by `ClosedAt`; `MemoryAdapter.FetchGaps` filters by `IdentifiedAt`. Both honor `Since`/`Until`/`Limit`.

## Signal Mapping

### Deal Records

Win/loss deals are normalized based on `Outcome`:

| Outcome | Signal Type | Severity | Summary |
|---------|--------------|----------|---------|
| `win` | `signal.TypeCompetitorLaunch` (reused for competitive wins) | `info` | `Competitive win vs {PrimaryCompetitor}: {AccountName}` |
| `loss` | `signal.TypeCompetitiveGap` | `high` | `Lost to {PrimaryCompetitor}: {AccountName}` |

| Deal Field | Signal Field |
|------------|--------------|
| `competitive-{Source}-{ID}` | `ID` |
| `Source` | `Source` (`common.SourceSystem{Type: "crm", Name: Source, ExternalID: OpportunityID}`) |
| `Reasons` | `Description` (formatted as `"Reasons: r1; r2"`) |
| `"competitive"` / `Outcome` | `Domain.Name` / `Domain.Subdomain` |
| `ClosedAt` (or now if zero) | `ObservedAt` |

Deal metadata: `competitive_source`, `outcome`, `opportunity_id`, `deal_value`, `reasons`, plus `signal.MetaCustomerRef` (from `AccountRef` or resolved via `customer_mappings`), `signal.MetaMarketRef`, `signal.MetaCompetitorRef` (primary), and `competitor_refs` (all competitors).

### Competitive Gaps

Gaps are normalized to `signal.TypeCompetitiveGap`:

| Gap Field | Signal Field |
|-----------|--------------|
| `competitive-gap-{Source}-{ID}` | `ID` |
| `Source` | `Source` (`common.SourceSystem{Type: "competitive-intel", Name: Source}`) |
| `Title` | `Summary` |
| `Description` | `Description` |
| `Severity` | `Severity` |
| `"competitive"` / `"gap"` | `Domain.Name` / `Domain.Subdomain` |
| `IdentifiedAt` (or now if zero) | `ObservedAt` |

Gap metadata: `competitive_source`, `capability`, `impact`, `deal_ids`, plus `signal.MetaCompetitorRef`, `signal.MetaMarketRef`, and `signal.MetaCapabilityRef` as typed refs.

Any additional fields set on `DealRecord.Metadata` / `CompetitiveGap.Metadata` are copied through to `signal.Metadata` unchanged.

## Capabilities

```go
Capabilities{
    SupportsStreaming:  false,
    SupportsBatchFetch: true,
    SupportsFiltering:  true,
    SignalTypes: []signal.Type{
        signal.TypeCompetitiveGap,
        signal.TypeCompetitorLaunch,
    },
}
```

## Raw vs. Curated

Competitive deals and gaps are curated signals — they're already-aggregated outcomes from CRM systems rather than raw events to be clustered. See [Raw vs. Curated Signals](../metadata-conventions.md#derived-metrics-not-metadata) and the [PRISM-Roadmap handoff guide](../prism-roadmap-handoff.md) for how competitive signals contribute to SAM-SOM pillar scoring.

## Streaming

The Competitive provider doesn't support real-time streaming. `Subscribe()` returns `omnisignal.ErrNotSupported`.
