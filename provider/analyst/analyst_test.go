package analyst_test

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnisignal"
	"github.com/plexusone/omnisignal/provider/analyst"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestNewProvider(t *testing.T) {
	p, err := analyst.NewProvider(omnisignal.Config{})
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
	if p.Name() != analyst.ProviderName {
		t.Errorf("Name = %s, want %s", p.Name(), analyst.ProviderName)
	}
}

func TestProviderCapabilities(t *testing.T) {
	p, _ := analyst.NewProvider(omnisignal.Config{})
	caps := p.Capabilities()

	if caps.SupportsStreaming {
		t.Error("analyst provider should not support streaming")
	}
	if !caps.SupportsBatchFetch {
		t.Error("analyst provider should support batch fetch")
	}
	if len(caps.SignalTypes) != 1 || caps.SignalTypes[0] != signal.TypeAnalystFinding {
		t.Errorf("SignalTypes = %v, want [analyst_finding]", caps.SignalTypes)
	}
}

func TestProviderSubscribeNotSupported(t *testing.T) {
	p, _ := analyst.NewProvider(omnisignal.Config{})

	_, err := p.Subscribe(context.Background(), omnisignal.SubscribeOptions{})
	if err != omnisignal.ErrNotSupported {
		t.Errorf("Subscribe error = %v, want ErrNotSupported", err)
	}
}

func TestProviderFetchWithMemoryAdapter(t *testing.T) {
	findings := []analyst.Finding{
		{
			ID:          "f1",
			ReportID:    "gartner-mq-iam-2026",
			Title:       "Gap in passwordless authentication",
			Description: "Vendor lacks FIDO2 support",
			Category:    "capability_gap",
			Source:      analyst.SourceGartner,
			Severity:    common.SeverityHigh,
			Markets:     []string{"identity-governance"},
			Competitors: []string{"okta", "ping"},
			PublishedAt: time.Now().Add(-24 * time.Hour),
		},
	}

	adapter := analyst.NewMemoryAdapter(analyst.SourceGartner, findings)

	p, err := analyst.NewProvider(omnisignal.Config{
		Options: map[string]any{
			"adapter": adapter,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	signals, err := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	if len(signals) != 1 {
		t.Fatalf("Fetch returned %d signals, want 1", len(signals))
	}

	sig := signals[0]
	if sig.Type != signal.TypeAnalystFinding {
		t.Errorf("Type = %s, want analyst_finding", sig.Type)
	}
	if sig.Summary != "Gap in passwordless authentication" {
		t.Errorf("Summary = %s", sig.Summary)
	}
	if sig.Severity != common.SeverityHigh {
		t.Errorf("Severity = %s, want high", sig.Severity)
	}

	// Check metadata refs
	if sig.Metadata[signal.MetaAnalystReportRef] != "analyst-report:gartner-mq-iam-2026" {
		t.Errorf("analyst_report_ref = %v", sig.Metadata[signal.MetaAnalystReportRef])
	}
	if sig.Metadata[signal.MetaMarketRef] != "market:identity-governance" {
		t.Errorf("market_ref = %v", sig.Metadata[signal.MetaMarketRef])
	}

	// Multiple competitors should be in competitor_refs array
	refs, ok := sig.Metadata["competitor_refs"].([]string)
	if !ok || len(refs) != 2 {
		t.Errorf("competitor_refs = %v, want 2 refs", sig.Metadata["competitor_refs"])
	}
}

func TestMemoryAdapterTimeFilters(t *testing.T) {
	now := time.Now()
	adapter := analyst.NewMemoryAdapter(analyst.SourceGartner, nil)

	adapter.Add(analyst.Finding{ID: "old", PublishedAt: now.Add(-48 * time.Hour)})
	adapter.Add(analyst.Finding{ID: "recent", PublishedAt: now.Add(-1 * time.Hour)})
	adapter.Add(analyst.Finding{ID: "future", PublishedAt: now.Add(24 * time.Hour)})

	findings, err := adapter.Fetch(context.Background(), omnisignal.FetchOptions{
		Since: now.Add(-24 * time.Hour),
		Until: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) != 1 || findings[0].ID != "recent" {
		t.Errorf("expected only 'recent' finding, got %d findings", len(findings))
	}
}

func TestMemoryAdapterLimit(t *testing.T) {
	adapter := analyst.NewMemoryAdapter(analyst.SourceGartner, nil)
	for i := 0; i < 10; i++ {
		adapter.Add(analyst.Finding{ID: string(rune('a' + i))})
	}

	findings, _ := adapter.Fetch(context.Background(), omnisignal.FetchOptions{Limit: 3})
	if len(findings) != 3 {
		t.Errorf("Limit 3 returned %d findings", len(findings))
	}
}

func TestMemoryAdapterClear(t *testing.T) {
	adapter := analyst.NewMemoryAdapter(analyst.SourceGartner, nil)
	adapter.Add(analyst.Finding{ID: "test"})
	adapter.Clear()

	findings, _ := adapter.Fetch(context.Background(), omnisignal.FetchOptions{})
	if len(findings) != 0 {
		t.Error("Clear should remove all findings")
	}
}

func TestFindingFields(t *testing.T) {
	now := time.Now()
	f := analyst.Finding{
		ID:          "f-123",
		ReportID:    "report-456",
		Title:       "Test Finding",
		Description: "Description",
		Category:    "market_trend",
		Source:      analyst.SourceForrester,
		Severity:    common.SeverityMedium,
		Markets:     []string{"cloud-security"},
		Competitors: []string{"crowdstrike"},
		PublishedAt: now,
		Metadata:    map[string]any{"wave_position": "leader"},
	}

	if f.ID != "f-123" {
		t.Error("ID mismatch")
	}
	if f.ReportID != "report-456" {
		t.Error("ReportID mismatch")
	}
	if f.Source != analyst.SourceForrester {
		t.Error("Source mismatch")
	}
	if len(f.Markets) != 1 || f.Markets[0] != "cloud-security" {
		t.Error("Markets mismatch")
	}
	if f.Metadata["wave_position"] != "leader" {
		t.Error("Metadata mismatch")
	}
}

func TestProviderClose(t *testing.T) {
	p, _ := analyst.NewProvider(omnisignal.Config{})
	if err := p.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestSignalDomain(t *testing.T) {
	adapter := analyst.NewMemoryAdapter(analyst.SourceIDC, []analyst.Finding{
		{
			ID:       "test",
			Category: "competitive_position",
		},
	})

	p, _ := analyst.NewProvider(omnisignal.Config{
		Options: map[string]any{"adapter": adapter},
	})

	signals, _ := p.Fetch(context.Background(), omnisignal.FetchOptions{})

	if len(signals) != 1 {
		t.Fatal("expected 1 signal")
	}
	if signals[0].Domain.Name != "market" {
		t.Errorf("Domain.Name = %s, want market", signals[0].Domain.Name)
	}
	if signals[0].Domain.Subdomain != "competitive_position" {
		t.Errorf("Domain.Subdomain = %s, want competitive_position", signals[0].Domain.Subdomain)
	}
}

func TestSourceConstants(t *testing.T) {
	if analyst.SourceGartner != "gartner" {
		t.Error("SourceGartner mismatch")
	}
	if analyst.SourceForrester != "forrester" {
		t.Error("SourceForrester mismatch")
	}
	if analyst.SourceIDC != "idc" {
		t.Error("SourceIDC mismatch")
	}
	if analyst.SourceCustom != "custom" {
		t.Error("SourceCustom mismatch")
	}
}
