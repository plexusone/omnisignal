# OmniSignal

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnisignal/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnisignal/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnisignal/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnisignal/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omnisignal/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omnisignal/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnisignal
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnisignal
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omnisignal
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomnisignal
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omnisignal
 [repo-url]: https://github.com/plexusone/omnisignal
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnisignal/blob/main/LICENSE

Unified signal ingestion abstraction for operational intelligence.

## Overview

`omnisignal` provides a unified interface for ingesting operational signals from various external systems (alerting, ticketing, security, monitoring). It follows the same architectural pattern as [omnillm](https://github.com/plexusone/omnillm):

• **Provider interface** defines the contract for signal sources
• **Registry** allows dynamic provider registration
• **Thick providers** use official SDKs (PagerDuty, Jira, New Relic)
• **Thin providers** use native HTTP for sources without Go SDKs

## Installation

```bash
go get github.com/plexusone/omnisignal
```

## Quick Start

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty" // Register PagerDuty
)

func main() {
    provider, err := omnisignal.New("pagerduty", omnisignal.Config{
        APIKey: os.Getenv("PAGERDUTY_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    // Fetch incidents from the last 24 hours
    signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
        Since: time.Now().Add(-24 * time.Hour),
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, sig := range signals {
        fmt.Printf("Signal: %s - %s (%s)\n", sig.ID, sig.Summary, sig.Severity)
    }
}
```

## Available Providers

### Built-in Providers

| Provider | Type | Import Path | SDK |
|----------|------|-------------|-----|
| PagerDuty | Alerting | `omnisignal/provider/pagerduty` | [go-pagerduty](https://github.com/PagerDuty/go-pagerduty) |
| Jira | Ticketing | `omnisignal/provider/jira` | [go-jira](https://github.com/andygrunwald/go-jira) |
| [Analyst](docs/providers/analyst.md) | Market Intelligence | `omnisignal/provider/analyst` | none (thin) |
| [Competitive](docs/providers/competitive.md) | Market Intelligence | `omnisignal/provider/competitive` | none (thin) |

### External Providers (Thick)

| Provider | Type | Import Path | SDK |
|----------|------|-------------|-----|
| New Relic | Monitoring | `omni-newrelic/omnisignal` | [newrelic-client-go](https://github.com/newrelic/newrelic-client-go) |
| [Aha](docs/providers/aha.md) | Product Feedback | `grokify/aha-studio/omnisignal` | [aha-go](https://github.com/grokify/aha-go) |

### Planned Providers

| Provider | Type | Priority |
|----------|------|----------|
| Zendesk | Ticketing | P1 |
| Datadog | Monitoring | P1 |
| Opsgenie | Alerting | P1 |
| ServiceNow | ITSM | P2 |
| Snyk | Security | P2 |

## Provider Interface

```go
type Provider interface {
    // Name returns the provider identifier
    Name() string

    // Fetch retrieves signals matching the given options
    Fetch(ctx context.Context, opts FetchOptions) ([]signal.Signal, error)

    // Subscribe opens a real-time stream of signals
    Subscribe(ctx context.Context, opts SubscribeOptions) (<-chan signal.Signal, error)

    // Capabilities returns what this provider supports
    Capabilities() Capabilities

    // Close releases any resources
    Close() error
}
```

## Configuration

```go
type Config struct {
    APIKey    string            // Primary authentication credential
    APISecret string            // Secondary credential (if required)
    BaseURL   string            // Override default API endpoint
    Timeout   time.Duration     // Request timeout
    RetryMax  int               // Max retry attempts
    Options   map[string]any    // Provider-specific options
}
```

### Provider-Specific Options

**PagerDuty:**
```go
omnisignal.Config{
    APIKey: "your-api-key",
}
```

**Jira:**
```go
omnisignal.Config{
    BaseURL:   "https://company.atlassian.net",
    APIKey:    "user@example.com",  // Username
    APISecret: "api-token",         // API token
    Options: map[string]any{
        "projects": []string{"INFRA", "SUPPORT"},
    },
}
```

**New Relic** (via omni-newrelic):
```go
omnisignal.Config{
    APIKey: "NRAK-xxx",
    Options: map[string]any{
        "account_id": 12345,
        "region":     "US",
    },
}
```

**Analyst** (Gartner, Forrester, IDC — see [docs](docs/providers/analyst.md)):
```go
omnisignal.Config{
    Options: map[string]any{
        "source": "gartner", // or "forrester", "idc", "custom"
    },
}
```

**Competitive** (win/loss and gaps from CRM — see [docs](docs/providers/competitive.md)):
```go
omnisignal.Config{
    Options: map[string]any{
        "source":              "salesforce", // or "hubspot", "clari", "gong", "custom"
        "competitor_mappings": map[string]string{"Okta Inc": "competitor:okta"},
        omnisignal.OptCustomerMappings: map[string]string{"Acme Corp": "customer:acme-001"},
        omnisignal.OptMarketMappings:   map[string]string{"IAM": "market:identity-governance"},
    },
}
```

### Config Helpers

`Config` provides typed accessors for reading provider `Options`:

```go
func (c Config) GetOption(key string, defaultVal any) any
func (c Config) GetStringOption(key, defaultVal string) string
func (c Config) GetStringMap(key string) map[string]string
```

`GetStringMap` accepts both `map[string]string` and `map[string]any` (with string values), which makes it safe to use with config loaded from JSON/YAML as well as Go literals.

Well-known option keys carry cross-repo reference mappings from source system values (e.g., organization names) to [MarketSpec](https://github.com/ProductBuildersHQ/market-spec) typed refs (e.g., `customer:acme-001`):

| Constant | Key | Maps |
|----------|-----|------|
| `OptCustomerMappings` | `customer_mappings` | Organization/account names → customer refs |
| `OptCapabilityMappings` | `capability_mappings` | Components/labels → capability refs |
| `OptMarketMappings` | `market_mappings` | Categories → market refs |

## Fetch Options

```go
type FetchOptions struct {
    Since      time.Time         // Filter signals after this time
    Until      time.Time         // Filter signals before this time
    Limit      int               // Maximum signals to return
    Statuses   []string          // Filter by status
    Severities []string          // Filter by severity
    Filters    map[string]string // Provider-specific filters
}
```

## Output Format

All providers normalize their data to [signal-spec](https://github.com/plexusone/signal-spec) types:

```go
type Signal struct {
    ID          string           // Unique signal identifier
    Type        Type             // support_ticket, alert, security_finding, etc.
    Status      Status           // new, processing, mapped, archived
    Source      SourceSystem     // Origin system details
    Domain      Domain           // Category/subcategory
    Severity    Severity         // critical, high, medium, low, info
    Summary     string           // Brief description
    Description string           // Full content
    Entities    []Entity         // Referenced system components
    ObservedAt  time.Time        // When signal was observed
    ReceivedAt  time.Time        // When signal was received
    Tags        []Tag            // User-defined labels
    Metadata    map[string]any   // Source-specific data
}
```

## Creating a Custom Provider

```go
package myprovider

import "github.com/plexusone/omnisignal"

func init() {
    omnisignal.Register("myprovider", NewProvider, omnisignal.PriorityThick)
}

func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
    // Initialize your provider
    return &Provider{config: cfg}, nil
}

type Provider struct {
    config omnisignal.Config
}

func (p *Provider) Name() string { return "myprovider" }

func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
    // Fetch from your source and normalize to signal.Signal
}

func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
    return nil, omnisignal.ErrNotSupported
}

func (p *Provider) Capabilities() omnisignal.Capabilities {
    return omnisignal.Capabilities{
        SupportsBatchFetch: true,
    }
}

func (p *Provider) Close() error { return nil }
```

## Curated Signals

Some sources (e.g., Aha Ideas, analyst findings, competitive deals) already represent a single aggregated data point rather than a raw event stream. Mark these signals as **curated** so the [consolidation pipeline](#consolidation-pipeline) skips clustering and maps them directly to root causes:

```go
sig.Metadata[omnisignal.MetaCurated] = true

// Or use the helper to check:
if omnisignal.IsCurated(sig.Metadata) {
    // Skip clustering, map directly to a canonical signal
}
```

See [Metadata Conventions](docs/metadata-conventions.md#raw-vs-curated-signals) for the full raw vs. curated model.

## Metrics Engine

The `metrics` package computes derived scores from signal sets via a pluggable formula registry:

```go
import "github.com/plexusone/omnisignal/metrics"

// Built-in formulas: frustration, momentum, reach, urgency
result, err := metrics.Compute(ctx, "frustration", signals, metrics.Options{
    Weights: map[string]float64{"support_ticket": 1.5},
})

// Or run every registered formula at once
results, errs := metrics.ComputeAll(ctx, signals, metrics.Options{})
```

| Formula | Description |
|---------|-------------|
| `frustration` | Weighted signal count multiplied by the age (in days) of the oldest signal |
| `momentum` | Count of signals observed within a trailing window (default 30 days) |
| `reach` | Count of distinct customer references across all signals |
| `urgency` | Sum of severity-weighted signal counts |

Weight overrides and window size can be loaded from JSON config and merged:

```go
cfg, err := metrics.LoadConfig("metrics.json")
merged := defaultCfg.Merge(cfg)
result, err := metrics.Compute(ctx, "urgency", signals, merged.ToOptions())
```

Custom formulas can be added via `metrics.Register(metrics.NewFormula(name, description, computeFn))`.

## Consolidation Pipeline

The `consolidate` package groups related raw signals into canonical root causes through a 5-stage pipeline: **embed → cluster → summarize → review → attach**.

```go
import "github.com/plexusone/omnisignal/consolidate"

pipeline := consolidate.NewPipeline(
    consolidate.WithEmbedder(consolidate.NewOmniLLMEmbedder(omnillmClient, consolidate.EmbedderConfig{
        Model: "text-embedding-3-small",
    })),
    consolidate.WithSummarizer(consolidate.NewLLMSummarizer(omnillmClient, consolidate.SummarizerConfig{
        Model: "gpt-4o-mini",
    })),
    consolidate.WithReviewer(consolidate.NewMemoryReviewer(consolidate.ReviewConfig{
        AutoApproveThreshold: 5,
    })),
    consolidate.WithSimilarityThreshold(0.85),
)

result, err := pipeline.Process(ctx, signals)
// result.RootCauses, result.Clusters, result.Attached, result.Stats
```

- **Embed** — `Embedder` generates vector embeddings via OmniLLM (`OmniLLMEmbedder`)
- **Cluster** — signals are grouped by cosine similarity (`Clusterer`, `IncrementalClusterer`)
- **Summarize** — an LLM generates a root cause title, description, and symptom patterns per cluster (`LLMSummarizer`)
- **Review** — optional human review queue with approve/reject and auto-approve threshold (`MemoryReviewer`)
- **Attach** — new signals are linked to existing root causes incrementally via `Pipeline.Attach`

[Curated signals](#curated-signals) skip embedding and clustering and map directly to root causes, preserving their existing aggregation.

## Related Packages

- [signal-spec](https://github.com/plexusone/signal-spec) - Canonical signal data model
- [signal](https://github.com/plexusone/signal) - Operational intelligence platform
- [omnillm](https://github.com/plexusone/omnillm) - Unified LLM abstraction

## License

MIT
