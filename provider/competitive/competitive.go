// Package competitive provides a competitive intelligence provider for omnisignal.
//
// This provider ingests win/loss data and competitive gap signals from CRM systems
// (Salesforce, HubSpot) and competitive intelligence platforms.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/plexusone/omnisignal/provider/competitive"
//	)
//
//	provider, err := omnisignal.New("competitive", omnisignal.Config{
//	    Options: map[string]any{
//	        "source": "salesforce",
//	    },
//	})
package competitive

import (
	"context"
	"fmt"
	"time"

	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/ref"
	"github.com/plexusone/signal-spec/pkg/signal"
)

const (
	// ProviderName is the identifier for this provider.
	ProviderName = "competitive"
)

// Source identifiers for CRM and competitive intelligence systems.
const (
	SourceSalesforce = "salesforce"
	SourceHubSpot    = "hubspot"
	SourceClari      = "clari"
	SourceGong       = "gong"
	SourceCustom     = "custom"
)

// Outcome represents the result of a competitive deal.
type Outcome string

const (
	OutcomeWin  Outcome = "win"
	OutcomeLoss Outcome = "loss"
)

func init() {
	omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThin)
}

// DealRecord represents a win/loss record from CRM.
type DealRecord struct {
	// ID is the unique identifier for this record.
	ID string

	// OpportunityID is the CRM opportunity identifier.
	OpportunityID string

	// AccountName is the customer/prospect name.
	AccountName string

	// AccountRef is the MarketSpec customer ref (e.g., "customer:acme-001").
	AccountRef string

	// Outcome is win or loss.
	Outcome Outcome

	// Competitors are the competitors involved in the deal.
	Competitors []string

	// PrimaryCompetitor is the main competitor (for losses, who won).
	PrimaryCompetitor string

	// Reasons are the factors that led to the outcome.
	Reasons []string

	// DealValue is the opportunity amount in cents.
	DealValue int64

	// Market is the MarketSpec market slug.
	Market string

	// ClosedAt is when the deal closed.
	ClosedAt time.Time

	// Source is the CRM system.
	Source string

	// Metadata contains additional source-specific fields.
	Metadata map[string]any
}

// CompetitiveGap represents a capability gap identified through competitive analysis.
type CompetitiveGap struct {
	// ID is the unique identifier.
	ID string

	// Title describes the gap.
	Title string

	// Description provides detail about the gap.
	Description string

	// Competitor is the competitor with the advantage.
	Competitor string

	// Capability is the specific feature or capability lacking.
	Capability string

	// Impact describes business impact (e.g., "lost 3 deals worth $500K").
	Impact string

	// Severity indicates how critical the gap is.
	Severity common.Severity

	// Market is the MarketSpec market slug.
	Market string

	// DealIDs links to related deal records.
	DealIDs []string

	// IdentifiedAt is when the gap was identified.
	IdentifiedAt time.Time

	// Source is where this gap was identified.
	Source string

	// Metadata contains additional fields.
	Metadata map[string]any
}

// Adapter converts competitive data to signals.
type Adapter interface {
	// FetchDeals retrieves win/loss records.
	FetchDeals(ctx context.Context, opts omnisignal.FetchOptions) ([]DealRecord, error)

	// FetchGaps retrieves competitive gap signals.
	FetchGaps(ctx context.Context, opts omnisignal.FetchOptions) ([]CompetitiveGap, error)

	// Source returns the adapter's source identifier.
	Source() string
}

// Provider implements omnisignal.Provider for competitive intelligence.
type Provider struct {
	config             omnisignal.Config
	adapter            Adapter
	competitorMappings map[string]string
	customerMappings   map[string]string
	marketMappings     map[string]string
}

// NewProvider creates a new competitive intelligence provider.
func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	source := cfg.GetStringOption("source", SourceCustom)

	var adapter Adapter
	if a, ok := cfg.Options["adapter"].(Adapter); ok {
		adapter = a
	} else {
		adapter = &MemoryAdapter{source: source}
	}

	return &Provider{
		config:             cfg,
		adapter:            adapter,
		competitorMappings: cfg.GetStringMap("competitor_mappings"),
		customerMappings:   cfg.GetStringMap(omnisignal.OptCustomerMappings),
		marketMappings:     cfg.GetStringMap(omnisignal.OptMarketMappings),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// Fetch retrieves competitive signals (both deals and gaps).
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	var signals []signal.Signal

	// Fetch deal records
	deals, err := p.adapter.FetchDeals(ctx, opts)
	if err != nil {
		return nil, omnisignal.WrapErrorByMessage(err, "fetching deal records")
	}
	for _, d := range deals {
		signals = append(signals, p.dealToSignal(d))
	}

	// Fetch competitive gaps
	gaps, err := p.adapter.FetchGaps(ctx, opts)
	if err != nil {
		return nil, omnisignal.WrapErrorByMessage(err, "fetching competitive gaps")
	}
	for _, g := range gaps {
		signals = append(signals, p.gapToSignal(g))
	}

	return signals, nil
}

// dealToSignal converts a DealRecord to a Signal.
func (p *Provider) dealToSignal(d DealRecord) signal.Signal {
	metadata := make(map[string]any)

	// Copy source metadata
	for k, v := range d.Metadata {
		metadata[k] = v
	}

	// Add deal-specific metadata
	metadata["competitive_source"] = d.Source
	metadata["outcome"] = string(d.Outcome)
	metadata["opportunity_id"] = d.OpportunityID
	metadata["deal_value"] = d.DealValue
	metadata["reasons"] = d.Reasons

	// Add typed refs
	if d.AccountRef != "" {
		metadata[signal.MetaCustomerRef] = d.AccountRef
	} else if d.AccountName != "" {
		if mapped, ok := p.customerMappings[d.AccountName]; ok {
			metadata[signal.MetaCustomerRef] = mapped
		}
	}

	if d.Market != "" {
		metadata[signal.MetaMarketRef] = string(ref.TypedRef("market:" + d.Market))
	}

	// Add competitor refs
	if d.PrimaryCompetitor != "" {
		metadata[signal.MetaCompetitorRef] = string(ref.TypedRef("competitor:" + d.PrimaryCompetitor))
	}
	if len(d.Competitors) > 0 {
		refs := make([]string, 0, len(d.Competitors))
		for _, c := range d.Competitors {
			refs = append(refs, string(ref.TypedRef("competitor:"+c)))
		}
		metadata["competitor_refs"] = refs
	}

	// Determine signal type based on outcome
	var signalType signal.Type
	var severity common.Severity
	var summary string

	if d.Outcome == OutcomeWin {
		signalType = signal.TypeCompetitorLaunch // Reusing for competitive win signal
		severity = common.SeverityInfo
		summary = fmt.Sprintf("Competitive win: %s", d.AccountName)
		if d.PrimaryCompetitor != "" {
			summary = fmt.Sprintf("Competitive win vs %s: %s", d.PrimaryCompetitor, d.AccountName)
		}
	} else {
		signalType = signal.TypeCompetitiveGap
		severity = common.SeverityHigh
		summary = fmt.Sprintf("Competitive loss: %s", d.AccountName)
		if d.PrimaryCompetitor != "" {
			summary = fmt.Sprintf("Lost to %s: %s", d.PrimaryCompetitor, d.AccountName)
		}
	}

	now := time.Now()
	closedAt := d.ClosedAt
	if closedAt.IsZero() {
		closedAt = now
	}

	return signal.Signal{
		ID:   fmt.Sprintf("competitive-%s-%s", d.Source, d.ID),
		Type: signalType,
		Source: common.SourceSystem{
			Type:       "crm",
			Name:       d.Source,
			ExternalID: d.OpportunityID,
		},
		Summary:     summary,
		Description: formatReasons(d.Reasons),
		Severity:    severity,
		Status:      signal.StatusNew,
		Domain: common.Domain{
			Name:      "competitive",
			Subdomain: string(d.Outcome),
		},
		ObservedAt: closedAt,
		ReceivedAt: now,
		Metadata:   metadata,
	}
}

// gapToSignal converts a CompetitiveGap to a Signal.
func (p *Provider) gapToSignal(g CompetitiveGap) signal.Signal {
	metadata := make(map[string]any)

	// Copy source metadata
	for k, v := range g.Metadata {
		metadata[k] = v
	}

	metadata["competitive_source"] = g.Source
	metadata["capability"] = g.Capability
	metadata["impact"] = g.Impact
	metadata["deal_ids"] = g.DealIDs

	// Add typed refs
	if g.Competitor != "" {
		metadata[signal.MetaCompetitorRef] = string(ref.TypedRef("competitor:" + g.Competitor))
	}
	if g.Market != "" {
		metadata[signal.MetaMarketRef] = string(ref.TypedRef("market:" + g.Market))
	}
	if g.Capability != "" {
		metadata[signal.MetaCapabilityRef] = string(ref.TypedRef("capability:" + g.Capability))
	}

	now := time.Now()
	identifiedAt := g.IdentifiedAt
	if identifiedAt.IsZero() {
		identifiedAt = now
	}

	return signal.Signal{
		ID:   fmt.Sprintf("competitive-gap-%s-%s", g.Source, g.ID),
		Type: signal.TypeCompetitiveGap,
		Source: common.SourceSystem{
			Type: "competitive-intel",
			Name: g.Source,
		},
		Summary:     g.Title,
		Description: g.Description,
		Severity:    g.Severity,
		Status:      signal.StatusNew,
		Domain: common.Domain{
			Name:      "competitive",
			Subdomain: "gap",
		},
		ObservedAt: identifiedAt,
		ReceivedAt: now,
		Metadata:   metadata,
	}
}

func formatReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	desc := "Reasons: "
	for i, r := range reasons {
		if i > 0 {
			desc += "; "
		}
		desc += r
	}
	return desc
}

// Subscribe is not supported for competitive intelligence.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Capabilities returns what this provider supports.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:  false,
		SupportsBatchFetch: true,
		SupportsFiltering:  true,
		SignalTypes: []signal.Type{
			signal.TypeCompetitiveGap,
			signal.TypeCompetitorLaunch,
		},
	}
}

// Close releases any resources held by the provider.
func (p *Provider) Close() error {
	return nil
}

// MemoryAdapter is an in-memory adapter for testing.
type MemoryAdapter struct {
	source string
	deals  []DealRecord
	gaps   []CompetitiveGap
}

// NewMemoryAdapter creates an in-memory adapter.
func NewMemoryAdapter(source string) *MemoryAdapter {
	return &MemoryAdapter{source: source}
}

// Source returns the adapter's source identifier.
func (m *MemoryAdapter) Source() string {
	return m.source
}

// FetchDeals retrieves deal records from memory.
func (m *MemoryAdapter) FetchDeals(ctx context.Context, opts omnisignal.FetchOptions) ([]DealRecord, error) {
	var result []DealRecord
	for _, d := range m.deals {
		if !opts.Since.IsZero() && d.ClosedAt.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && !d.ClosedAt.Before(opts.Until) {
			continue
		}
		result = append(result, d)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}

// FetchGaps retrieves competitive gaps from memory.
func (m *MemoryAdapter) FetchGaps(ctx context.Context, opts omnisignal.FetchOptions) ([]CompetitiveGap, error) {
	var result []CompetitiveGap
	for _, g := range m.gaps {
		if !opts.Since.IsZero() && g.IdentifiedAt.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && !g.IdentifiedAt.Before(opts.Until) {
			continue
		}
		result = append(result, g)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}

// AddDeal adds a deal record.
func (m *MemoryAdapter) AddDeal(d DealRecord) {
	if d.Source == "" {
		d.Source = m.source
	}
	m.deals = append(m.deals, d)
}

// AddGap adds a competitive gap.
func (m *MemoryAdapter) AddGap(g CompetitiveGap) {
	if g.Source == "" {
		g.Source = m.source
	}
	m.gaps = append(m.gaps, g)
}

// Clear removes all records.
func (m *MemoryAdapter) Clear() {
	m.deals = nil
	m.gaps = nil
}
