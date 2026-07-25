# PRISM-Roadmap Handoff

This document describes how OmniSignal signals map to the three prioritization pillars in PRISM-Roadmap:

- **CSAT (Customer Satisfaction)** — signals from existing customers indicating pain, friction, or enhancement requests
- **SAM-SOM (Serviceable Market)** — signals about competitive positioning and expansion within current market segments
- **TAM (Total Addressable Market)** — signals about market trends, new segments, and strategic opportunities

## Signal Type Mapping

| Signal Type | Pillar | Rationale |
|-------------|--------|-----------|
| `support_ticket` | CSAT | Direct customer pain points |
| `cloud_incident` | CSAT | Service reliability affecting existing customers |
| `enhancement_request` | CSAT / SAM-SOM | Customer requests may signal churn risk (CSAT) or competitive gaps (SAM-SOM) |
| `competitive_gap` | SAM-SOM | Capability gaps losing deals to competitors |
| `competitor_launch` | SAM-SOM / TAM | Competitive moves affecting current and future positioning |
| `analyst_finding` | TAM / SAM-SOM | Market intelligence and strategic guidance |
| `market_observation` | TAM | Broad market trends and emerging opportunities |

## Cross-Repo References

Signals carry typed refs that link to canonical entities in MarketSpec and other ecosystem repositories:

| Metadata Key | Entity Type | Example | Used By |
|--------------|-------------|---------|---------|
| `customer_ref` | Customer | `customer:acme-001` | CSAT scoring |
| `capability_ref` | Capability | `capability:passwordless-auth` | Gap analysis |
| `market_ref` | Market | `market:identity-governance` | TAM/SAM sizing |
| `competitor_ref` | Competitor | `competitor:okta` | Competitive positioning |
| `analyst_report_ref` | Report | `analyst-report:gartner-mq-iam-2026` | Strategic context |

## Root Cause Aggregation

The consolidation pipeline clusters related signals into root causes. A single root cause may aggregate:

- **Support tickets** — raw customer pain signals (CSAT)
- **Enhancement requests** — curated demand signals with votes/watchers (CSAT/SAM-SOM)
- **Competitive gaps** — win/loss data from CRM (SAM-SOM)
- **Analyst findings** — strategic context from Gartner/Forrester (TAM/SAM-SOM)

### Example Root Cause

```
Root Cause: "Passwordless Authentication Gap"
├── Support tickets (12): Login friction, MFA complaints
├── Aha Idea: "Add FIDO2/WebAuthn support" (votes: 47, ARR: $2.1M)
├── Competitive gaps (3): Lost to Okta citing native FIDO2
└── Analyst finding: Gartner MQ 2026 calls out missing passwordless
```

This root cause touches all three pillars:

- **CSAT**: 12 support tickets = customer pain
- **SAM-SOM**: 3 competitive losses + Aha idea with $2.1M ARR at stake
- **TAM**: Analyst finding indicates strategic importance

## Pillar Scoring

PRISM-Roadmap computes pillar scores from signal-level metrics:

| Pillar | Input Signals | Metrics Used |
|--------|---------------|--------------|
| CSAT | support_ticket, cloud_incident, enhancement_request | Frustration, Urgency |
| SAM-SOM | enhancement_request, competitive_gap, competitor_launch | Reach, Momentum |
| TAM | analyst_finding, market_observation, competitor_launch | Market size, Strategic alignment |

### Metric Definitions

- **Frustration** — weighted count × age: how long customers have been waiting
- **Urgency** — severity-weighted count: how critical the signals are
- **Reach** — distinct customer count: how widespread the impact
- **Momentum** — trailing 30-day count: how fast signals are accumulating

## Integration Points

### Signal → Root Cause → Roadmap Item

```
┌─────────────────┐     ┌────────────────┐     ┌─────────────────┐
│  Raw Signals    │────▶│  Root Causes   │────▶│  Roadmap Items  │
│  (OmniSignal)   │     │  (OmniSignal)  │     │  (PRISM)        │
└─────────────────┘     └────────────────┘     └─────────────────┘
         │                      │                      │
         ▼                      ▼                      ▼
   Fingerprint &          LLM Summary &          Pillar scores
   Clustering             Severity rollup        & prioritization
```

### API Contract

PRISM-Roadmap consumes root causes via the `consolidate.Pipeline`:

```go
import "github.com/plexusone/omnisignal/consolidate"

pipeline := consolidate.New(consolidate.Config{
    Embedder:   llmEmbedder,
    Summarizer: llmSummarizer,
    Reviewer:   memoryReviewer,
})

// Process new signals
result, err := pipeline.Process(ctx, signals)

// Attach late signals to existing root causes
attached, err := pipeline.Attach(ctx, lateSignals, existingRootCauses)
```

Each `RootCause` includes:

- `ID`, `Title`, `Description` — LLM-generated summary
- `SignalIDs` — contributing signals for evidence
- `Domain`, `Severity` — inferred from signals
- `SymptomPatterns` — common manifestations
- `Embedding` — cluster centroid for semantic search

## Curated vs Raw Signals

| Signal Source | Classification | Consolidation Behavior |
|---------------|----------------|------------------------|
| Support tickets | Raw | Clustered by embedding similarity |
| Incidents | Raw | Clustered by embedding similarity |
| Aha Ideas | Curated | Skip clustering, map 1:1 to canonical signal |
| Analyst findings | Curated | Skip clustering, preserve report structure |
| Competitive gaps | Curated | Skip clustering, preserve deal linkage |

Set `metadata["curated"] = true` to bypass clustering for pre-aggregated sources.

## Acceptance Criteria

Per RMI-OMNISIGNAL-023, a valid integration demonstrates:

1. **One root cause** aggregating multiple signal types
2. **Each signal** referencing at least one canonical MarketSpec entity
3. **Pillar coverage** — signals contributing to CSAT, SAM-SOM, and TAM

Example test case:

```go
signals := []signal.Signal{
    // CSAT: support tickets
    {Type: signal.TypeSupportTicket, Metadata: map[string]any{
        signal.MetaCustomerRef: "customer:acme-001",
    }},
    // SAM-SOM: competitive gap
    {Type: signal.TypeCompetitiveGap, Metadata: map[string]any{
        signal.MetaCompetitorRef: "competitor:okta",
        signal.MetaMarketRef:     "market:identity-governance",
    }},
    // TAM: analyst finding
    {Type: signal.TypeAnalystFinding, Metadata: map[string]any{
        signal.MetaAnalystReportRef: "analyst-report:gartner-mq-iam-2026",
        signal.MetaMarketRef:        "market:identity-governance",
    }},
}

result, _ := pipeline.Process(ctx, signals)
// result.RootCauses[0] aggregates all three signal types
```
