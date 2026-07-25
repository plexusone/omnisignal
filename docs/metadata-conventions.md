# Signal Metadata Conventions

This document defines the standard metadata keys for signals ingested by OmniSignal. These conventions ensure consistent data across providers and enable downstream systems (metrics engines, prioritization tools) to consume signals uniformly.

## Enhancement Request Metadata

Enhancement requests (`signal.TypeEnhancementRequest`) carry product demand signals. The following metadata keys are defined in `signal-spec/pkg/signal`:

| Key | Type | Description |
|-----|------|-------------|
| `votes` | `int` | Total vote/upvote count |
| `subscribers` | `int` | Number of watchers/subscribers |
| `organizations` | `[]string` | Requesting organization names |
| `customers` | `[]string` | Named customer identifiers |
| `opportunities` | `[]string` | Linked sales opportunity IDs |
| `estimated_arr` | `int64` | Estimated ARR at stake, in cents |

### Usage Example

```go
import "github.com/plexusone/signal-spec/pkg/signal"

sig := signal.Signal{
    Type: signal.TypeEnhancementRequest,
    Metadata: map[string]any{
        signal.MetaVotes:         42,
        signal.MetaSubscribers:   15,
        signal.MetaOrganizations: []string{"Acme Corp", "Globex"},
        signal.MetaCustomers:     []string{"acme-001", "globex-002"},
        signal.MetaOpportunities: []string{"OPP-2026-100"},
        signal.MetaEstimatedARR:  150000_00, // $150,000 in cents
    },
}
```

### Provider Mapping

Aha Ideas map to these keys as follows:

| Aha Field | Metadata Key |
|-----------|--------------|
| `votes_count` | `votes` |
| `watchers_count` | `subscribers` |
| `idea_organizations` | `organizations` |
| `idea_contacts` | `customers` |
| `linked_opportunities` | `opportunities` |
| (computed from opportunity values) | `estimated_arr` |

## Cross-Repo Reference Metadata

Signals can reference entities from other repositories using typed refs in `{type}:{slug}` format. The following keys carry these references:

| Key | Type | Description |
|-----|------|-------------|
| `customer_ref` | `string` | Reference to customer entity (e.g., `customer:acme-001`) |
| `capability_ref` | `string` or `[]string` | Reference to product capability |
| `market_ref` | `string` | Reference to market (e.g., `market:identity-governance`) |
| `competitor_ref` | `string` | Reference to competitor |
| `analyst_report_ref` | `string` | Reference to analyst report |

### Validation

Use `ref.Validate()` or `ref.ValidateStrict()` from `signal-spec/pkg/ref` to validate typed references:

```go
import "github.com/plexusone/signal-spec/pkg/ref"

r := ref.TypedRef("customer:acme-001")
if err := ref.Validate(r); err != nil {
    // invalid reference
}
```

## Provider-Specific Metadata

Providers may include source-specific metadata prefixed with the provider name:

### Jira

| Key | Type | Description |
|-----|------|-------------|
| `jira_issue_type` | `string` | Issue type (Bug, Story, Task, etc.) |
| `jira_project` | `string` | Project key |
| `jira_status` | `string` | Issue status |
| `jira_priority` | `string` | Priority name |
| `jira_reporter` | `string` | Reporter display name |

### PagerDuty

| Key | Type | Description |
|-----|------|-------------|
| `pagerduty_incident_number` | `int` | Incident number |
| `pagerduty_urgency` | `string` | Urgency level (high, low) |
| `pagerduty_status` | `string` | Incident status |
| `pagerduty_priority` | `any` | Priority object |

## Raw vs. Curated Signals

Signals are classified as **raw** or **curated** based on whether they need consolidation:

| Classification | Description | Example Sources |
|----------------|-------------|-----------------|
| **Raw** | Individual events that should be clustered | Support tickets, alerts, incidents |
| **Curated** | Pre-aggregated signals that skip clustering | Aha Ideas (already have votes/watchers) |

Set the `curated` metadata key to `true` for pre-consolidated signals:

```go
import "github.com/plexusone/omnisignal"

sig.Metadata["curated"] = true

// Or use the helper:
if omnisignal.IsCurated(sig.Metadata) {
    // Skip clustering, map directly to canonical signal
}
```

Curated signals bypass the embed→cluster→summarize stages and map directly to root causes, preserving their existing aggregation (votes, watchers, customer lists).

## Derived Metrics (Not Metadata)

Computed scores like frustration, momentum, reach, and urgency are NOT stored in metadata. They live in `Signal.Derived` and are excluded from fingerprinting — identical raw input produces identical fingerprints regardless of derived values.

```go
sig.Derived = &signal.DerivedMetrics{
    Frustration: &frustrationScore,
    Momentum:    &momentumScore,
    Reach:       &reachScore,
    Urgency:     &urgencyScore,
    ComputedAt:  &computedTime,
}
```
