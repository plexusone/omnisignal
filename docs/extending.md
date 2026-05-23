# Creating Custom Providers

This guide explains how to create your own OmniSignal provider.

## Provider Interface

All providers must implement the `Provider` interface:

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

## Basic Implementation

```go
package myprovider

import (
    "context"

    "github.com/plexusone/omnisignal"
    "github.com/plexusone/signal-spec/pkg/signal"
)

const ProviderName = "myprovider"

func init() {
    omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThick)
}

type Provider struct {
    config omnisignal.Config
    client *MyAPIClient
}

func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
    if cfg.APIKey == "" {
        return nil, fmt.Errorf("%w: APIKey is required", omnisignal.ErrInvalidConfig)
    }

    client := NewMyAPIClient(cfg.APIKey)

    return &Provider{
        config: cfg,
        client: client,
    }, nil
}

func (p *Provider) Name() string {
    return ProviderName
}

func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
    // Fetch from your source
    items, err := p.client.List(ctx, opts.Since, opts.Until)
    if err != nil {
        return nil, err
    }

    // Normalize to signals
    signals := make([]signal.Signal, 0, len(items))
    for _, item := range items {
        sig := p.normalize(item)
        signals = append(signals, sig)

        if opts.Limit > 0 && len(signals) >= opts.Limit {
            break
        }
    }

    return signals, nil
}

func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
    // Return ErrNotSupported if streaming isn't available
    return nil, omnisignal.ErrNotSupported
}

func (p *Provider) Capabilities() omnisignal.Capabilities {
    return omnisignal.Capabilities{
        SupportsStreaming:  false,
        SupportsBatchFetch: true,
        SupportsFiltering:  true,
        MaxBatchSize:       100,
        SignalTypes: []signal.Type{
            signal.TypeAlert,
        },
    }
}

func (p *Provider) Close() error {
    return p.client.Close()
}
```

## Signal Normalization

Convert your source data to the standard signal format:

```go
func (p *Provider) normalize(item MyItem) signal.Signal {
    return signal.Signal{
        ID:     fmt.Sprintf("myprovider-%s", item.ID),
        Type:   signal.TypeAlert,
        Status: mapStatus(item.Status),
        Source: common.SourceSystem{
            Type:       "monitoring",
            Name:       ProviderName,
            ExternalID: item.ID,
            URL:        item.URL,
        },
        Domain: common.Domain{
            Name:      "operations",
            Subdomain: item.Category,
        },
        Severity:    mapSeverity(item.Priority),
        Summary:     item.Title,
        Description: item.Description,
        ObservedAt:  item.CreatedAt,
        ReceivedAt:  time.Now(),
        Metadata: map[string]any{
            "myprovider_custom_field": item.CustomField,
        },
    }
}
```

## Registration

Register your provider in `init()`:

```go
func init() {
    omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThick)
}
```

### Priority Levels

| Priority | Constant | Use Case |
|----------|----------|----------|
| 10 | `PriorityThick` | SDK-based providers |
| 0 | `PriorityThin` | HTTP-only providers |

Higher priority providers override lower priority ones with the same name.

## Pagination

Handle pagination internally so callers don't need to worry about it:

```go
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
    var signals []signal.Signal
    var cursor string

    for {
        page, nextCursor, err := p.client.ListPage(ctx, cursor, 100)
        if err != nil {
            return nil, err
        }

        for _, item := range page {
            sig := p.normalize(item)
            signals = append(signals, sig)

            if opts.Limit > 0 && len(signals) >= opts.Limit {
                return signals, nil
            }
        }

        if nextCursor == "" {
            break
        }
        cursor = nextCursor
    }

    return signals, nil
}
```

## Error Handling

Use standard errors when appropriate:

```go
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
    resp, err := p.client.Do(ctx, req)
    if err != nil {
        return nil, err
    }

    switch resp.StatusCode {
    case 401, 403:
        return nil, omnisignal.ErrAuthentication
    case 429:
        return nil, omnisignal.ErrRateLimited
    }

    // ...
}
```

## Testing

Test with the registry helpers:

```go
func TestProvider(t *testing.T) {
    // Clear registry for isolated tests
    omnisignal.ClearRegistry()
    defer omnisignal.ClearRegistry()

    // Register provider
    omnisignal.Register("test", NewProvider, omnisignal.PriorityThick)

    // Verify registration
    if !omnisignal.IsRegistered("test") {
        t.Fatal("provider not registered")
    }

    // Create instance
    provider, err := omnisignal.New("test", omnisignal.Config{
        APIKey: "test-key",
    })
    if err != nil {
        t.Fatal(err)
    }
    defer provider.Close()

    // Test fetch
    signals, err := provider.Fetch(context.Background(), omnisignal.FetchOptions{})
    // ...
}
```

## External Providers

For providers with heavy dependencies, create a separate module:

```
github.com/plexusone/omni-newrelic/
├── go.mod
├── omnisignal/
│   └── newrelic.go  // Implements omnisignal.Provider
└── omnillm/
    └── newrelic.go  // Implements omnillm.Provider (if applicable)
```

This keeps the core omnisignal module lightweight.
