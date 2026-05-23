# Installation

## Requirements

- Go 1.21 or later

## Install the Package

```bash
go get github.com/plexusone/omnisignal
```

## Import Providers

OmniSignal uses a provider registration pattern. Import the providers you need using blank imports:

```go
import (
    "github.com/plexusone/omnisignal"

    // Import providers you want to use
    _ "github.com/plexusone/omnisignal/provider/pagerduty"
    _ "github.com/plexusone/omnisignal/provider/jira"
)
```

Each provider registers itself automatically via `init()`.

## Available Providers

### Built-in Providers

These providers are included in the main omnisignal module:

| Provider | Import Path |
|----------|-------------|
| PagerDuty | `github.com/plexusone/omnisignal/provider/pagerduty` |
| Jira | `github.com/plexusone/omnisignal/provider/jira` |

### External Providers

These providers are available as separate modules:

| Provider | Module |
|----------|--------|
| New Relic | `github.com/plexusone/omni-newrelic/omnisignal` |

Install external providers separately:

```bash
go get github.com/plexusone/omni-newrelic
```

## Verify Installation

```go
package main

import (
    "fmt"

    "github.com/plexusone/omnisignal"
    _ "github.com/plexusone/omnisignal/provider/pagerduty"
)

func main() {
    providers := omnisignal.List()
    fmt.Printf("Registered providers: %v\n", providers)
    // Output: Registered providers: [pagerduty]
}
```
