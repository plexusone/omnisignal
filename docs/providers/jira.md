# Jira Provider

The Jira provider fetches issues from Jira using the [go-jira](https://github.com/andygrunwald/go-jira) community SDK.

## Installation

The Jira provider is built into omnisignal:

```go
import (
    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/jira"
)
```

## Configuration

```go
provider, err := omnisignal.New("jira", omnisignal.Config{
    BaseURL:   "https://company.atlassian.net",
    APIKey:    os.Getenv("JIRA_USER"),    // Username/email
    APISecret: os.Getenv("JIRA_TOKEN"),   // API token
    Options: map[string]any{
        "projects": []string{"INFRA", "SUPPORT"},
    },
})
```

### Required Fields

| Field | Description |
|-------|-------------|
| `BaseURL` | Jira instance URL (e.g., `https://company.atlassian.net`) |
| `APIKey` | Username or email address |
| `APISecret` | API token |

### Optional Fields

| Field | Description |
|-------|-------------|
| `Options["projects"]` | List of project keys to filter (e.g., `["INFRA", "SUPPORT"]`) |

### Getting an API Token

1. Log in to [Atlassian Account](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click **Create API token**
3. Use your email as `APIKey` and the token as `APISecret`

## Capabilities

```go
caps := provider.Capabilities()
```

| Capability | Value |
|------------|-------|
| `SupportsStreaming` | `false` |
| `SupportsBatchFetch` | `true` |
| `SupportsFiltering` | `true` |
| `SupportsAcknowledge` | `false` |
| `MaxBatchSize` | 100 |
| `RateLimitPerMinute` | varies by instance |
| `SignalTypes` | `support_ticket`, `feedback` |

## Usage

### Fetch Recent Issues

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-7 * 24 * time.Hour),
})
```

### Filter by Status

Use Jira status names:

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:    time.Now().Add(-30 * 24 * time.Hour),
    Statuses: []string{"Open", "In Progress", "In Review"},
})
```

### Filter by Severity

Severity maps to Jira priority:

| OmniSignal | Jira Priorities |
|------------|-----------------|
| `critical` | Highest, Blocker |
| `high` | High |
| `medium` | Medium, Normal |
| `low` | Low |
| `info` | Lowest, Trivial |

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since:      time.Now().Add(-7 * 24 * time.Hour),
    Severities: []string{"critical", "high"},
})
```

### Custom JQL Filter

Use the `jql` filter for advanced queries:

```go
signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
    Since: time.Now().Add(-30 * 24 * time.Hour),
    Filters: map[string]string{
        "jql": "labels = incident AND component = api",
    },
})
```

## Signal Mapping

Jira issues are normalized to signals:

| Signal Field | Jira Source |
|--------------|-------------|
| `ID` | `jira-{issue.Key}` |
| `Type` | `support_ticket` |
| `Status` | Mapped from issue status |
| `Severity` | Mapped from priority |
| `Summary` | `issue.Fields.Summary` |
| `Description` | `issue.Fields.Description` |
| `ObservedAt` | `issue.Fields.Created` |
| `Source.URL` | `{BaseURL}/browse/{issue.Key}` |
| `Source.ExternalID` | `issue.Key` |
| `Domain.Name` | Project key (lowercase) |
| `Domain.Subdomain` | Issue type (normalized) |
| `Tags` | Labels (kebab-case only) |

### Status Mapping

| Jira Status | Signal Status |
|-------------|---------------|
| Contains "done", "closed", "resolved" | `archived` |
| Contains "progress", "review" | `processing` |
| Other | `new` |

### Metadata

Provider-specific data in `signal.Metadata`:

```go
metadata := signal.Metadata

issueType := metadata["jira_issue_type"].(string)
project := metadata["jira_project"].(string)
status := metadata["jira_status"].(string)
priority := metadata["jira_priority"].(string)
reporter := metadata["jira_reporter"].(string)
```

## Entities

Components are extracted as entities:

```go
for _, entity := range signal.Entities {
    if entity.Type == "component" {
        fmt.Printf("Component: %s (ID: %s)\n",
            entity.Name,
            entity.Attributes["jira_id"],
        )
    }
}
```

## Tags

Jira labels are converted to tags if they are valid kebab-case:

- `incident` → included
- `high-priority` → included
- `HighPriority` → excluded (not kebab-case)
- `has spaces` → excluded (not kebab-case)

## Streaming

Jira doesn't support real-time streaming via API. The `Subscribe()` method returns `ErrNotSupported`.

For real-time updates, configure [Jira Webhooks](https://developer.atlassian.com/server/jira/platform/webhooks/) to push events to your application.
