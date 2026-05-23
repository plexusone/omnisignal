# OmniSignal

**Unified signal ingestion abstraction for operational intelligence**

OmniSignal provides a unified interface for ingesting operational signals from various external systems (alerting, ticketing, security, monitoring). It follows the same architectural pattern as [OmniLLM](https://github.com/plexusone/omnillm):

• **Provider interface** defines the contract for signal sources
• **Registry** allows dynamic provider registration
• **Thick providers** use official SDKs (PagerDuty, Jira, New Relic)
• **Thin providers** use native HTTP for sources without Go SDKs

## Features

- **Multi-Source Support**: PagerDuty, Jira, New Relic, with more planned
- **Unified API**: Same interface across all signal sources
- **Normalized Output**: All signals conform to [signal-spec](https://github.com/plexusone/signal-spec)
- **Batch Fetching**: Efficient pagination handled internally
- **Filtering**: Time-based, status, severity, and provider-specific filters
- **Streaming Ready**: Subscribe interface for real-time signals (provider-dependent)
- **Type Safe**: Full Go type safety with comprehensive error handling

## Quick Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty"
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
    signals, err := provider.Fetch(context.Background(), omnisignal.FetchOptions{
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

## Supported Providers

| Provider | Type | Status | SDK |
|----------|------|--------|-----|
| [PagerDuty](providers/pagerduty.md) | Alerting | Built-in | [go-pagerduty](https://github.com/PagerDuty/go-pagerduty) |
| [Jira](providers/jira.md) | Ticketing | Built-in | [go-jira](https://github.com/andygrunwald/go-jira) |
| New Relic | Monitoring | External | [newrelic-client-go](https://github.com/newrelic/newrelic-client-go) |

### Planned Providers

| Provider | Type | Priority |
|----------|------|----------|
| Zendesk | Ticketing | P1 |
| Datadog | Monitoring | P1 |
| Opsgenie | Alerting | P1 |
| ServiceNow | ITSM | P2 |
| Snyk | Security | P2 |

## Next Steps

- [Installation](getting-started/installation.md) - Get OmniSignal set up
- [Quick Start](getting-started/quickstart.md) - Your first signal fetch
- [Providers](providers/index.md) - Configure specific providers
- [Configuration](configuration.md) - Understand configuration options

## Related Packages

- [signal-spec](https://github.com/plexusone/signal-spec) - Canonical signal data model
- [signal](https://github.com/plexusone/signal) - Operational intelligence platform
- [omnillm](https://github.com/plexusone/omnillm) - Unified LLM abstraction
