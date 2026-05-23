# Configuration

## Provider Configuration

All providers are configured using the `Config` struct:

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

### Fields

| Field | Description | Required |
|-------|-------------|----------|
| `APIKey` | Primary authentication credential (API key, username, etc.) | Provider-dependent |
| `APISecret` | Secondary credential (API secret, password, token) | Provider-dependent |
| `BaseURL` | Override the provider's default API endpoint | No |
| `Timeout` | HTTP request timeout; zero uses provider default | No |
| `RetryMax` | Maximum retry attempts; zero uses provider default | No |
| `Options` | Provider-specific configuration map | No |

### Helper Methods

```go
// GetOption retrieves a typed option value with a default fallback
val := config.GetOption("key", defaultValue)

// GetStringOption retrieves a string option with a default fallback
str := config.GetStringOption("key", "default")
```

## Fetch Options

Control what signals are retrieved:

```go
type FetchOptions struct {
    Since      time.Time         // Filter signals after this time (inclusive)
    Until      time.Time         // Filter signals before this time (exclusive)
    Limit      int               // Maximum signals to return (0 = unlimited)
    Statuses   []string          // Filter by status
    Severities []string          // Filter by severity
    Filters    map[string]string // Provider-specific filters
}
```

### Time Filtering

```go
// Last 24 hours
opts := omnisignal.FetchOptions{
    Since: time.Now().Add(-24 * time.Hour),
}

// Specific time range
opts := omnisignal.FetchOptions{
    Since: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    Until: time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
}
```

### Status Filtering

Status values are provider-specific:

```go
// PagerDuty statuses
opts := omnisignal.FetchOptions{
    Statuses: []string{"triggered", "acknowledged"},
}

// Jira statuses
opts := omnisignal.FetchOptions{
    Statuses: []string{"Open", "In Progress"},
}
```

### Severity Filtering

Uses normalized severity levels:

```go
opts := omnisignal.FetchOptions{
    Severities: []string{"critical", "high", "medium", "low", "info"},
}
```

Providers map these to their native concepts:

| Normalized | PagerDuty | Jira |
|------------|-----------|------|
| critical | high urgency | Highest, Blocker |
| high | high urgency | High |
| medium | low urgency | Medium, Normal |
| low | low urgency | Low |
| info | low urgency | Lowest, Trivial |

## Subscribe Options

For providers that support real-time streaming:

```go
type SubscribeOptions struct {
    BufferSize int               // Channel buffer size (default: 100)
    Filters    map[string]string // Provider-specific filters
}
```

!!! note
    Most providers don't support streaming via `Subscribe()`. Check `Capabilities().SupportsStreaming` before using.

## Provider Capabilities

Query what a provider supports:

```go
type Capabilities struct {
    SupportsStreaming   bool         // Real-time signal streaming
    SupportsBatchFetch  bool         // Efficient batch fetching
    SupportsFiltering   bool         // Server-side filtering
    SupportsAcknowledge bool         // Acknowledge signals
    MaxBatchSize        int          // Max signals per request
    RateLimitPerMinute  int          // Provider rate limit
    SignalTypes         []signal.Type // Signal types this provider emits
}
```

### Example

```go
caps := provider.Capabilities()

if caps.SupportsStreaming {
    ch, err := provider.Subscribe(ctx, omnisignal.SubscribeOptions{})
    // ...
}

if caps.RateLimitPerMinute > 0 {
    fmt.Printf("Rate limit: %d requests/minute\n", caps.RateLimitPerMinute)
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

The `Metadata` field contains provider-specific data not captured by the standard fields.
