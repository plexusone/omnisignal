# OmniSignal Product Requirements

## Overview

OmniSignal is a Go library that ingests operational and product signals from external systems (ticketing, alerting, CRM, product management) and normalizes them into the canonical `signal.Signal` format defined by [signal-spec](https://github.com/plexusone/signal-spec).

It is the evidence-collection layer in the ProductContext ecosystem, sitting between source systems and the PRISM-Roadmap prioritization engine.

## Problem Statement

Product and engineering teams collect feedback and operational data across many disconnected systems -- Jira, PagerDuty, Aha, Zendesk, Salesforce, GitHub. Each system has its own data model, API, and terminology. Without normalization, teams cannot:

- See aggregated demand for a capability across all sources
- Compute cross-source metrics like frustration score or momentum
- Feed evidence-based data into roadmap prioritization
- Identify root causes that span multiple support tickets

## Personas

| Persona | Needs |
|---|---|
| **Product Manager** | Aggregated signal views, frustration scores, demand evidence for roadmap decisions |
| **Engineering Lead** | Root cause mapping across incidents and tickets, trend visibility |
| **Data Analyst** | Normalized signal data for reporting and dashboards |
| **Platform Engineer** | Simple provider integration, consistent API across sources |

## Use Cases

### UC-1: Normalize Support Tickets

Ingest support tickets from Jira and Zendesk into a common signal format. Enable cross-system aggregation by customer, product, and capability.

### UC-2: Track Enhancement Demand

Ingest enhancement requests from Aha Ideas and similar systems. Compute vote-based and customer-based demand metrics.

### UC-3: Compute Frustration Scores

Calculate `weighted_signal_count x age_in_days` for root causes aggregated from multiple signals. Surface long-waiting, high-demand items.

### UC-4: Monitor Operational Health

Stream alerts and incidents from PagerDuty, Datadog, and cloud providers. Map to root causes and track recurrence.

### UC-5: Feed Roadmap Prioritization

Provide normalized, scored signals to PRISM-Roadmap for the three strategic pillars:

- **CSAT** -- Customer satisfaction from support and enhancement signals
- **SAM/SOM** -- Competitive gap and analyst signals
- **TAM** -- Market growth and trend signals

## Requirements

### Functional

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | Provider interface with Fetch and Subscribe methods | High |
| FR-2 | Registry-based provider discovery with priority levels | High |
| FR-3 | Jira provider (support tickets, bugs) | High |
| FR-4 | PagerDuty provider (alerts, incidents) | High |
| FR-5 | Aha provider (ideas, enhancement requests) | High |
| FR-6 | Zendesk provider (tickets) | Medium |
| FR-7 | GitHub provider (issues, discussions) | Medium |
| FR-8 | Salesforce provider (cases, opportunities) | Medium |
| FR-9 | Time-range and status filtering on fetch | High |
| FR-10 | Real-time streaming via Subscribe for supported providers | Medium |
| FR-11 | Fingerprint-based deduplication | High |
| FR-12 | Derived metrics engine (frustration, momentum, reach) | Medium |
| FR-13 | LLM-based signal consolidation into root causes | Medium |
| FR-14 | Cross-repo entity references via typed IDs | High |

### Non-Functional

| ID | Requirement | Priority |
|---|---|---|
| NF-1 | Thread-safe providers for concurrent use | High |
| NF-2 | Internal pagination handling (callers see a single result) | High |
| NF-3 | Rate limit awareness per provider | High |
| NF-4 | Configurable retry with backoff | Medium |
| NF-5 | Output conforms to signal-spec schemas | High |
| NF-6 | Zero external dependencies beyond signal-spec for core interface | High |

## Success Metrics

- Number of active provider integrations
- Signal volume ingested per day
- Root cause coverage (% of signals mapped to a root cause)
- Adoption by downstream consumers (PRISM-Roadmap, ProductContext)

## Out of Scope

- Full entity definitions for Customer, Market, Competitor (owned by OrganizationSpec, MarketSpec)
- Prioritization and scoring logic (owned by PRISM-Roadmap)
- UI or dashboard (consumers build their own views)
- Signal storage backend (callers decide persistence)
