# omnisignal

Unified signal ingestion abstraction for operational intelligence.

[![Go Reference](https://pkg.go.dev/badge/github.com/plexusone/omnisignal.svg)](https://pkg.go.dev/github.com/plexusone/omnisignal)

## Overview

`omnisignal` provides a unified interface for ingesting operational signals from various external systems (alerting, ticketing, security, monitoring). It follows the same architectural pattern as [omnillm](https://github.com/plexusone/omnillm):

- **Provider interface** defines the contract for signal sources
- **Registry** allows dynamic provider registration
- **Thick providers** use official SDKs (PagerDuty, Jira, New Relic)
- **Thin providers** use native HTTP for sources without Go SDKs

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

### External Providers (Thick)

| Provider | Type | Import Path | SDK |
|----------|------|-------------|-----|
| New Relic | Monitoring | `omni-newrelic/omnisignal` | [newrelic-client-go](https://github.com/newrelic/newrelic-client-go) |

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

## Related Packages

- [signal-spec](https://github.com/plexusone/signal-spec) - Canonical signal data model
- [signal](https://github.com/plexusone/signal) - Operational intelligence platform
- [omnillm](https://github.com/plexusone/omnillm) - Unified LLM abstraction

## License

MIT
