# PagerDuty Provider

The PagerDuty provider fetches incidents from PagerDuty using the official [go-pagerduty](https://github.com/PagerDuty/go-pagerduty) SDK.

## Installation

The PagerDuty provider is built into omnisignal:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty"
)
```

## Configuration

```go
provider, err := omnisignal.New("pagerduty", omnisignal.Config{
    APIKey: os.Getenv("PAGERDUTY_API_KEY"),
})
```

### Required Fields

| Field | Description |
|-------|-------------|
| `APIKey` | PagerDuty API key (v2 REST API key) |

### Getting an API Key

1. Log in to PagerDuty
2. Go to **Integrations** → **API Access Keys**
3. Create a new **REST API Key** (read-only is sufficient)

## Capabilities

```go
caps := provider.Capabilities()
```

| Capability | Value |
|------------|-------|
| `SupportsStreaming` | `false` |
| `SupportsBatchFetch` | `true` |
| `SupportsFiltering` | `true` |
| `SupportsAcknowledge` | `true` |
| `MaxBatchSize` | 100 |
| `RateLimitPerMinute` | 900 |
| `SignalTypes` | `alert`, `outage` |

## Usage

### Fetch Recent Incidents

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-24 * time.Hour),
})
```

### Filter by Status

PagerDuty statuses:

- `triggered` - New, unacknowledged
- `acknowledged` - Someone is working on it
- `resolved` - Incident closed

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:    time.Now().Add(-7 * 24 * time.Hour),
    Statuses: []string{"triggered", "acknowledged"},
})
```

### Filter by Severity

Severity mapping:

| OmniSignal | PagerDuty |
|------------|-----------|
| `critical`, `high` | High urgency |
| `medium`, `low`, `info` | Low urgency |

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:      time.Now().Add(-24 * time.Hour),
    Severities: []string{"critical", "high"},
})
```

## Signal Mapping

PagerDuty incidents are normalized to signals:

| Signal Field | PagerDuty Source |
|--------------|------------------|
| `ID` | `pd-{incident.ID}` |
| `Type` | `alert` or `outage` |
| `Status` | Mapped from incident status |
| `Severity` | Mapped from urgency |
| `Summary` | `incident.Title` |
| `Description` | `incident.Description` |
| `ObservedAt` | `incident.CreatedAt` |
| `Source.URL` | `incident.HTMLURL` |
| `Source.ExternalID` | `incident.ID` |

### Status Mapping

| PagerDuty Status | Signal Status |
|------------------|---------------|
| `triggered` | `new` |
| `acknowledged` | `processing` |
| `resolved` | `archived` |

### Metadata

Provider-specific data in `signal.Metadata`:

```go
metadata := signal.Metadata

incidentNumber := metadata["pagerduty_incident_number"].(int)
urgency := metadata["pagerduty_urgency"].(string)
status := metadata["pagerduty_status"].(string)
priority := metadata["pagerduty_priority"]
```

## Entities

Services are extracted as entities:

```go
for _, entity := range signal.Entities {
    if entity.Type == "service" {
        fmt.Printf("Service: %s (ID: %s)\n",
            entity.Name,
            entity.Attributes["pagerduty_id"],
        )
    }
}
```

## Streaming

PagerDuty doesn't support real-time streaming via API. The `Subscribe()` method returns `ErrNotSupported`.

For real-time updates, configure [PagerDuty Webhooks](https://developer.pagerduty.com/docs/webhooks/v3-overview/) to push events to your application.
