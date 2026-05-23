# Providers Overview

OmniSignal uses a provider pattern to support multiple signal sources through a unified interface.

## Provider Types

### Thick Providers

Thick providers use official SDKs from the signal source vendor. They offer:

- Full API coverage
- Automatic authentication handling
- Built-in pagination
- Type-safe request/response handling

Examples: PagerDuty, Jira, New Relic

### Thin Providers

Thin providers use native HTTP without external SDK dependencies. They offer:

- Minimal dependencies
- Smaller binary size
- Direct control over API calls

## Built-in Providers

| Provider | Type | Import Path | Signal Types |
|----------|------|-------------|--------------|
| [PagerDuty](pagerduty.md) | Alerting | `omnisignal/provider/pagerduty` | alert, outage |
| [Jira](jira.md) | Ticketing | `omnisignal/provider/jira` | support_ticket, feedback |

## External Providers

| Provider | Type | Module | Signal Types |
|----------|------|--------|--------------|
| New Relic | Monitoring | `omni-newrelic/omnisignal` | alert, metric_anomaly |

## Provider Registration

Providers register themselves automatically via `init()`:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty" // Registers "pagerduty"
    _ "github.com/plexusone/omnisignal/provider/jira"      // Registers "jira"
)

func main() {
    // List registered providers
    providers := omnisignal.List()
    // ["jira", "pagerduty"]

    // Check if a provider is registered
    if omnisignal.IsRegistered("pagerduty") {
        provider, err := omnisignal.New("pagerduty", cfg)
    }
}
```

## Priority System

When multiple implementations exist for the same provider, the one with higher priority wins:

| Priority | Constant | Description |
|----------|----------|-------------|
| 10 | `PriorityThick` | SDK-based implementations |
| 0 | `PriorityThin` | HTTP-based implementations |

This allows thick providers to automatically override thin providers when both are imported.

## Creating Instances

Use `omnisignal.New()` to create provider instances:

```go
// Create with explicit config
provider, err := omnisignal.New("pagerduty", omnisignal.Config{
    APIKey: os.Getenv("PAGERDUTY_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Panic on error (use only in init)
provider := omnisignal.MustNew("pagerduty", cfg)
```

## Common Errors

| Error | Cause |
|-------|-------|
| `ErrProviderNotFound` | Provider name not registered; check import |
| `ErrInvalidConfig` | Missing required configuration field |
| `ErrAuthentication` | API key or credentials invalid |
| `ErrRateLimited` | Too many requests; implement backoff |
| `ErrNotSupported` | Operation not available for this provider |
