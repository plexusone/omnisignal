# Quick Start

This guide walks you through fetching your first signals from PagerDuty.

## Prerequisites

- OmniSignal installed (see [Installation](installation.md))
- A PagerDuty API key

## Create a Provider

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
    // Create a PagerDuty provider
    provider, err := omnisignal.New("pagerduty", omnisignal.Config{
        APIKey: os.Getenv("PAGERDUTY_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    fmt.Printf("Provider: %s\n", provider.Name())
}
```

## Fetch Signals

Retrieve incidents from the last 24 hours:

```go
ctx := context.Background()

signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-24 * time.Hour),
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d signals\n", len(signals))

for _, sig := range signals {
    fmt.Printf("[%s] %s - %s\n", sig.Severity, sig.ID, sig.Summary)
}
```

## Filter by Status

Fetch only triggered (active) incidents:

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:    time.Now().Add(-7 * 24 * time.Hour),
    Statuses: []string{"triggered", "acknowledged"},
})
```

## Filter by Severity

Fetch only high-urgency incidents:

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:      time.Now().Add(-24 * time.Hour),
    Severities: []string{"critical", "high"},
})
```

## Limit Results

Fetch at most 10 signals:

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-24 * time.Hour),
    Limit: 10,
})
```

## Check Provider Capabilities

```go
caps := provider.Capabilities()

fmt.Printf("Supports streaming: %v\n", caps.SupportsStreaming)
fmt.Printf("Supports batch fetch: %v\n", caps.SupportsBatchFetch)
fmt.Printf("Max batch size: %d\n", caps.MaxBatchSize)
fmt.Printf("Signal types: %v\n", caps.SignalTypes)
```

## Next Steps

- [Configuration](../configuration.md) - Learn about all configuration options
- [PagerDuty Provider](../providers/pagerduty.md) - PagerDuty-specific features
- [Jira Provider](../providers/jira.md) - Fetch tickets from Jira
