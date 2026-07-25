// Package analyst provides an analyst findings provider for omnisignal.
//
// This provider ingests analyst findings from various sources (Gartner, Forrester,
// IDC, etc.) and converts them to signals that reference MarketSpec entities.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/plexusone/omnisignal/provider/analyst"
//	)
//
//	provider, err := omnisignal.New("analyst", omnisignal.Config{
//	    Options: map[string]any{
//	        "source": "gartner", // or "forrester", "idc", etc.
//	    },
//	})
package analyst

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
	ProviderName = "analyst"

	// DefaultTimeout for API requests.
	DefaultTimeout = 30 * time.Second
)

// Source identifiers for analyst firms.
const (
	SourceGartner   = "gartner"
	SourceForrester = "forrester"
	SourceIDC       = "idc"
	SourceCustom    = "custom"
)

func init() {
	omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThin)
}

// Finding represents an analyst finding from reports.
type Finding struct {
	// ID is the unique identifier for this finding.
	ID string

	// ReportID is the MarketSpec report identifier (e.g., "gartner-mq-iam-2026").
	ReportID string

	// Title is a brief summary of the finding.
	Title string

	// Description is the full finding text.
	Description string

	// Category classifies the finding (e.g., "capability_gap", "market_trend", "competitive_position").
	Category string

	// Source is the analyst firm (e.g., "gartner", "forrester").
	Source string

	// Severity indicates the impact level.
	Severity common.Severity

	// Markets are the MarketSpec market slugs this finding applies to.
	Markets []string

	// Competitors are the MarketSpec competitor slugs mentioned.
	Competitors []string

	// PublishedAt is when the report was published.
	PublishedAt time.Time

	// Metadata contains additional source-specific fields.
	Metadata map[string]any
}

// Adapter converts analyst findings to signals. Implement this interface to
// support different analyst data sources.
type Adapter interface {
	// Fetch retrieves findings from the source.
	Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]Finding, error)

	// Source returns the adapter's source identifier.
	Source() string
}

// Provider implements omnisignal.Provider for analyst findings.
type Provider struct {
	config         omnisignal.Config
	adapter        Adapter
	marketMappings map[string]string
}

// NewProvider creates a new analyst provider.
func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	source := cfg.GetStringOption("source", SourceCustom)

	var adapter Adapter
	if a, ok := cfg.Options["adapter"].(Adapter); ok {
		adapter = a
	} else {
		adapter = &MemoryAdapter{source: source}
	}

	return &Provider{
		config:         cfg,
		adapter:        adapter,
		marketMappings: cfg.GetStringMap(omnisignal.OptMarketMappings),
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// Fetch retrieves analyst findings as signals.
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	findings, err := p.adapter.Fetch(ctx, opts)
	if err != nil {
		return nil, omnisignal.WrapErrorByMessage(err, "fetching analyst findings")
	}

	signals := make([]signal.Signal, 0, len(findings))
	for _, f := range findings {
		sig := p.toSignal(f)
		signals = append(signals, sig)
	}

	return signals, nil
}

// toSignal converts an analyst Finding to a Signal.
func (p *Provider) toSignal(f Finding) signal.Signal {
	metadata := make(map[string]any)

	// Copy source metadata
	for k, v := range f.Metadata {
		metadata[k] = v
	}

	// Add analyst-specific metadata
	metadata["analyst_source"] = f.Source
	metadata["analyst_category"] = f.Category
	metadata["analyst_report_id"] = f.ReportID

	// Add MarketSpec typed refs
	if f.ReportID != "" {
		metadata[signal.MetaAnalystReportRef] = string(ref.TypedRef("analyst-report:" + f.ReportID))
	}

	// Add market refs
	if len(f.Markets) > 0 {
		refs := make([]string, 0, len(f.Markets))
		for _, m := range f.Markets {
			refs = append(refs, string(ref.TypedRef("market:"+m)))
		}
		if len(refs) == 1 {
			metadata[signal.MetaMarketRef] = refs[0]
		} else {
			metadata["market_refs"] = refs
		}
	}

	// Add competitor refs
	if len(f.Competitors) > 0 {
		refs := make([]string, 0, len(f.Competitors))
		for _, c := range f.Competitors {
			refs = append(refs, string(ref.TypedRef("competitor:"+c)))
		}
		if len(refs) == 1 {
			metadata[signal.MetaCompetitorRef] = refs[0]
		} else {
			metadata["competitor_refs"] = refs
		}
	}

	now := time.Now()
	observedAt := f.PublishedAt
	if observedAt.IsZero() {
		observedAt = now
	}

	return signal.Signal{
		ID:   fmt.Sprintf("analyst-%s-%s", f.Source, f.ID),
		Type: signal.TypeAnalystFinding,
		Source: common.SourceSystem{
			Type: "analyst",
			Name: f.Source,
		},
		Summary:     f.Title,
		Description: f.Description,
		Severity:    f.Severity,
		Status:      signal.StatusNew,
		Domain: common.Domain{
			Name:      "market",
			Subdomain: f.Category,
		},
		ObservedAt: observedAt,
		ReceivedAt: now,
		Metadata:   metadata,
	}
}

// Subscribe is not supported for analyst findings.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Capabilities returns what this provider supports.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:  false,
		SupportsBatchFetch: true,
		SupportsFiltering:  true,
		SignalTypes:        []signal.Type{signal.TypeAnalystFinding},
	}
}

// Close releases any resources held by the provider.
func (p *Provider) Close() error {
	return nil
}

// MemoryAdapter is an in-memory adapter for testing and manual finding ingestion.
type MemoryAdapter struct {
	source   string
	findings []Finding
}

// NewMemoryAdapter creates an in-memory adapter with pre-loaded findings.
func NewMemoryAdapter(source string, findings []Finding) *MemoryAdapter {
	return &MemoryAdapter{
		source:   source,
		findings: findings,
	}
}

// Source returns the adapter's source identifier.
func (m *MemoryAdapter) Source() string {
	return m.source
}

// Fetch retrieves findings from memory.
func (m *MemoryAdapter) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]Finding, error) {
	var result []Finding

	for _, f := range m.findings {
		// Apply time filters
		if !opts.Since.IsZero() && f.PublishedAt.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && !f.PublishedAt.Before(opts.Until) {
			continue
		}

		result = append(result, f)

		// Apply limit
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	return result, nil
}

// Add adds a finding to the in-memory store.
func (m *MemoryAdapter) Add(f Finding) {
	if f.Source == "" {
		f.Source = m.source
	}
	m.findings = append(m.findings, f)
}

// Clear removes all findings.
func (m *MemoryAdapter) Clear() {
	m.findings = nil
}
