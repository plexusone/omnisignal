# Analyst Provider

The Analyst provider ingests analyst findings from research firms (Gartner, Forrester, IDC) and normalizes them to signals that reference [MarketSpec](https://github.com/ProductBuildersHQ/market-spec) entities.

## Installation

The Analyst provider is built into omnisignal:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/analyst"
)
```

## Configuration

```go
provider, err := omnisignal.New("analyst", omnisignal.Config{
    Options: map[string]any{
        "source": "gartner", // or "forrester", "idc", "custom"
    },
})
```

### Options

| Field | Required | Description |
|-------|----------|-------------|
| `Options["source"]` | No | Analyst firm identifier: `gartner`, `forrester`, `idc`, or `custom` (default) |
| `Options["adapter"]` | No | A custom `analyst.Adapter` implementation. Defaults to an empty `MemoryAdapter` |
| `Options[omnisignal.OptMarketMappings]` | No | `map[string]string` mapping source categories to `market:` refs |

The default in-process adapter is `analyst.MemoryAdapter`, which is useful for testing and for manually-curated finding ingestion. Production integrations should implement the `analyst.Adapter` interface against a firm's API or export format and pass it via `Options["adapter"]`.

## Adapter Interface

```go
type Adapter interface {
    // Fetch retrieves findings from the source.
    Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]Finding, error)

    // Source returns the adapter's source identifier.
    Source() string
}
```

## Finding Type

```go
type Finding struct {
    ID          string            // Unique identifier for this finding
    ReportID    string            // MarketSpec report identifier (e.g., "gartner-mq-iam-2026")
    Title       string            // Brief summary of the finding
    Description string            // Full finding text
    Category    string            // "capability_gap", "market_trend", "competitive_position", etc.
    Source      string            // Analyst firm (e.g., "gartner", "forrester")
    Severity    common.Severity   // Impact level
    Markets     []string          // MarketSpec market slugs this finding applies to
    Competitors []string          // MarketSpec competitor slugs mentioned
    PublishedAt time.Time         // When the report was published
    Metadata    map[string]any    // Additional source-specific fields
}
```

## Usage

### Manual Ingestion via MemoryAdapter

```go
adapter := analyst.NewMemoryAdapter("gartner", nil)
adapter.Add(analyst.Finding{
    ID:          "mq-iam-2026-passwordless",
    ReportID:    "gartner-mq-iam-2026",
    Title:       "Passwordless authentication now table stakes",
    Description: "Vendors lacking native FIDO2/WebAuthn support are losing evaluations.",
    Category:    "capability_gap",
    Severity:    common.SeverityHigh,
    Markets:     []string{"identity-governance"},
    Competitors: []string{"okta"},
    PublishedAt: time.Now(),
})

provider, err := omnisignal.New("analyst", omnisignal.Config{
    Options: map[string]any{
        "source":  "gartner",
        "adapter": adapter,
    },
})

signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{})
```

### Fetch with Time Filters

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-90 * 24 * time.Hour),
    Limit: 50,
})
```

`MemoryAdapter.Fetch` filters findings by `PublishedAt` against `Since`/`Until` and honors `Limit`.

## Signal Mapping

Findings are normalized to `signal.TypeAnalystFinding`:

| Finding Field | Signal Field |
|----------------|--------------|
| `analyst-{Source}-{ID}` | `ID` |
| `Source`, `ReportID`, `Category` | `Source` (`common.SourceSystem{Type: "analyst", Name: Source}`) |
| `Title` | `Summary` |
| `Description` | `Description` |
| `Severity` | `Severity` |
| `"market"` / `Category` | `Domain.Name` / `Domain.Subdomain` |
| `PublishedAt` (or now if zero) | `ObservedAt` |

### Metadata

| Key | Description |
|-----|-------------|
| `analyst_source` | Analyst firm identifier |
| `analyst_category` | Finding category |
| `analyst_report_id` | Raw report identifier |
| `signal.MetaAnalystReportRef` | Typed ref `analyst-report:{ReportID}` |
| `signal.MetaMarketRef` | Typed ref `market:{slug}` (single market) |
| `market_refs` | `[]string` of typed market refs (multiple markets) |
| `signal.MetaCompetitorRef` | Typed ref `competitor:{slug}` (single competitor) |
| `competitor_refs` | `[]string` of typed competitor refs (multiple competitors) |

Any additional fields set on `Finding.Metadata` are copied through to `signal.Metadata` unchanged.

## Capabilities

```go
Capabilities{
    SupportsStreaming:  false,
    SupportsBatchFetch: true,
    SupportsFiltering:  true,
    SignalTypes: []signal.Type{
        signal.TypeAnalystFinding,
    },
}
```

## Raw vs. Curated

Analyst findings are treated as curated signals — they represent a single, already-vetted finding rather than raw events to be clustered. See [Raw vs. Curated Signals](../metadata-conventions.md#derived-metrics-not-metadata) and the [PRISM-Roadmap handoff guide](../prism-roadmap-handoff.md) for how analyst findings contribute to TAM/SAM-SOM pillar scoring.

## Streaming

The Analyst provider doesn't support real-time streaming. `Subscribe()` returns `omnisignal.ErrNotSupported`.
