# Aha Provider

The Aha provider ingests Ideas from [Aha!](https://www.aha.io/) as enhancement request signals.

## Location

The Aha provider is implemented in `grokify/aha-studio`, not in omnisignal core. This keeps heavy Aha SDK dependencies out of the core module.

```
github.com/grokify/aha-studio/
└── omnisignal/
    └── provider.go  // Implements omnisignal.Provider
```

## Registration

Import the provider to auto-register:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/grokify/aha-studio/omnisignal" // Register Aha provider
)

provider, err := omnisignal.New("aha", omnisignal.Config{
    BaseURL:   "https://company.aha.io",
    APIKey:    os.Getenv("AHA_API_KEY"),
    Options: map[string]any{
        "products": []string{"PROD-1"},
    },
})
```

## Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `BaseURL` | Yes | Aha instance URL (e.g., `https://company.aha.io`) |
| `APIKey` | Yes | Aha API key |
| `Options["products"]` | No | Filter by product IDs |
| `Options["statuses"]` | No | Filter by idea status |
| `Options["customer_mappings"]` | No | Map organization names to typed refs |
| `Options["capability_mappings"]` | No | Map categories to capability refs |

## Signal Mapping

Aha Ideas are normalized to `signal.TypeEnhancementRequest`:

| Aha Field | Signal Field |
|-----------|--------------|
| `id` | `ID` (prefixed with `aha-`) |
| `name` | `Summary` |
| `description` | `Description` |
| `created_at` | `ObservedAt` |
| `votes_count` | `Metadata["votes"]` |
| `watchers_count` | `Metadata["subscribers"]` |
| `idea_organizations` | `Metadata["organizations"]` |
| `idea_contacts` | `Metadata["customers"]` |
| `product.id` | `Domain.Name` |
| `category` | `Domain.Subdomain` |
| (derived from votes/watchers) | `Severity` |

## Capabilities

```go
Capabilities{
    SupportsStreaming:   false,
    SupportsBatchFetch:  true,
    SupportsFiltering:   true,
    SupportsAcknowledge: false,
    MaxBatchSize:        200,
    RateLimitPerMinute:  300,
    SignalTypes: []signal.Type{
        signal.TypeEnhancementRequest,
    },
}
```

## Raw vs. Curated

Aha Ideas are considered "curated" signals — they've already been aggregated and voted on by users. Set the `curated` metadata key to `true` so the consolidation pipeline can skip clustering:

```go
Metadata: map[string]any{
    "curated": true,
    // ... other fields
}
```

See [Raw vs. Curated Signals](../metadata-conventions.md#derived-metrics-not-metadata) for details.

## Implementation Status

The Aha provider is tracked in [INIT-AHASTUDIO-001](https://github.com/grokify/aha-studio/blob/main/docs/specs/ROADMAP.md). The omnisignal adapter work is covered by the "OmniSignal adapter" RMIs in that initiative.
