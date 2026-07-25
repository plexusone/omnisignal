package competitive_test

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnisignal"
	"github.com/plexusone/omnisignal/provider/competitive"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestNewProvider(t *testing.T) {
	p, err := competitive.NewProvider(omnisignal.Config{})
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
	if p.Name() != competitive.ProviderName {
		t.Errorf("Name = %s, want %s", p.Name(), competitive.ProviderName)
	}
}

func TestProviderCapabilities(t *testing.T) {
	p, _ := competitive.NewProvider(omnisignal.Config{})
	caps := p.Capabilities()

	if caps.SupportsStreaming {
		t.Error("competitive provider should not support streaming")
	}
	if !caps.SupportsBatchFetch {
		t.Error("competitive provider should support batch fetch")
	}
	if len(caps.SignalTypes) != 2 {
		t.Errorf("SignalTypes = %v, want 2 types", caps.SignalTypes)
	}
}

func TestProviderFetchDeals(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)
	adapter.AddDeal(competitive.DealRecord{
		ID:                "d1",
		OpportunityID:     "OPP-001",
		AccountName:       "Acme Corp",
		Outcome:           competitive.OutcomeLoss,
		PrimaryCompetitor: "okta",
		Competitors:       []string{"okta", "ping"},
		Reasons:           []string{"Missing SSO", "Price too high"},
		DealValue:         500000_00,
		Market:            "identity-governance",
		ClosedAt:          time.Now().Add(-24 * time.Hour),
	})

	p, err := competitive.NewProvider(omnisignal.Config{
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
	if sig.Type != signal.TypeCompetitiveGap {
		t.Errorf("Type = %s, want competitive_gap (for loss)", sig.Type)
	}
	if sig.Severity != common.SeverityHigh {
		t.Errorf("Severity = %s, want high", sig.Severity)
	}

	// Check metadata
	if sig.Metadata[signal.MetaCompetitorRef] != "competitor:okta" {
		t.Errorf("competitor_ref = %v", sig.Metadata[signal.MetaCompetitorRef])
	}
	if sig.Metadata[signal.MetaMarketRef] != "market:identity-governance" {
		t.Errorf("market_ref = %v", sig.Metadata[signal.MetaMarketRef])
	}
	if sig.Metadata["deal_value"] != int64(500000_00) {
		t.Errorf("deal_value = %v", sig.Metadata["deal_value"])
	}

	refs, ok := sig.Metadata["competitor_refs"].([]string)
	if !ok || len(refs) != 2 {
		t.Errorf("competitor_refs = %v", sig.Metadata["competitor_refs"])
	}
}

func TestProviderFetchWin(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)
	adapter.AddDeal(competitive.DealRecord{
		ID:                "d2",
		AccountName:       "Globex",
		Outcome:           competitive.OutcomeWin,
		PrimaryCompetitor: "auth0",
		ClosedAt:          time.Now(),
	})

	p, _ := competitive.NewProvider(omnisignal.Config{
		Options: map[string]any{"adapter": adapter},
	})

	signals, _ := p.Fetch(context.Background(), omnisignal.FetchOptions{})

	if len(signals) != 1 {
		t.Fatal("expected 1 signal")
	}

	sig := signals[0]
	if sig.Type != signal.TypeCompetitorLaunch {
		t.Errorf("Type = %s, want competitor_launch (used for wins)", sig.Type)
	}
	if sig.Severity != common.SeverityInfo {
		t.Errorf("Severity = %s, want info for wins", sig.Severity)
	}
	if sig.Domain.Subdomain != "win" {
		t.Errorf("Domain.Subdomain = %s, want win", sig.Domain.Subdomain)
	}
}

func TestProviderFetchGaps(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceClari)
	adapter.AddGap(competitive.CompetitiveGap{
		ID:           "g1",
		Title:        "Missing FIDO2 support",
		Description:  "Competitor has native FIDO2, we rely on third-party",
		Competitor:   "okta",
		Capability:   "passwordless-auth",
		Impact:       "Lost 3 deals worth $1.5M",
		Severity:     common.SeverityCritical,
		Market:       "identity-governance",
		DealIDs:      []string{"d1", "d2", "d3"},
		IdentifiedAt: time.Now(),
	})

	p, _ := competitive.NewProvider(omnisignal.Config{
		Options: map[string]any{"adapter": adapter},
	})

	signals, err := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(signals) != 1 {
		t.Fatal("expected 1 signal")
	}

	sig := signals[0]
	if sig.Type != signal.TypeCompetitiveGap {
		t.Errorf("Type = %s, want competitive_gap", sig.Type)
	}
	if sig.Severity != common.SeverityCritical {
		t.Errorf("Severity = %s, want critical", sig.Severity)
	}
	if sig.Metadata[signal.MetaCompetitorRef] != "competitor:okta" {
		t.Errorf("competitor_ref = %v", sig.Metadata[signal.MetaCompetitorRef])
	}
	if sig.Metadata[signal.MetaCapabilityRef] != "capability:passwordless-auth" {
		t.Errorf("capability_ref = %v", sig.Metadata[signal.MetaCapabilityRef])
	}
	if sig.Metadata["impact"] != "Lost 3 deals worth $1.5M" {
		t.Errorf("impact = %v", sig.Metadata["impact"])
	}
}

func TestMemoryAdapterTimeFilters(t *testing.T) {
	now := time.Now()
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)

	adapter.AddDeal(competitive.DealRecord{ID: "old", ClosedAt: now.Add(-48 * time.Hour)})
	adapter.AddDeal(competitive.DealRecord{ID: "recent", ClosedAt: now.Add(-1 * time.Hour)})
	adapter.AddGap(competitive.CompetitiveGap{ID: "g-old", IdentifiedAt: now.Add(-48 * time.Hour)})
	adapter.AddGap(competitive.CompetitiveGap{ID: "g-recent", IdentifiedAt: now.Add(-1 * time.Hour)})

	deals, _ := adapter.FetchDeals(context.Background(), omnisignal.FetchOptions{
		Since: now.Add(-24 * time.Hour),
	})
	if len(deals) != 1 || deals[0].ID != "recent" {
		t.Errorf("expected only recent deal, got %d", len(deals))
	}

	gaps, _ := adapter.FetchGaps(context.Background(), omnisignal.FetchOptions{
		Since: now.Add(-24 * time.Hour),
	})
	if len(gaps) != 1 || gaps[0].ID != "g-recent" {
		t.Errorf("expected only recent gap, got %d", len(gaps))
	}
}

func TestMemoryAdapterLimit(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)
	for i := 0; i < 10; i++ {
		adapter.AddDeal(competitive.DealRecord{ID: string(rune('a' + i))})
	}

	deals, _ := adapter.FetchDeals(context.Background(), omnisignal.FetchOptions{Limit: 3})
	if len(deals) != 3 {
		t.Errorf("Limit 3 returned %d deals", len(deals))
	}
}

func TestMemoryAdapterClear(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)
	adapter.AddDeal(competitive.DealRecord{ID: "test"})
	adapter.AddGap(competitive.CompetitiveGap{ID: "gap"})
	adapter.Clear()

	deals, _ := adapter.FetchDeals(context.Background(), omnisignal.FetchOptions{})
	gaps, _ := adapter.FetchGaps(context.Background(), omnisignal.FetchOptions{})

	if len(deals) != 0 || len(gaps) != 0 {
		t.Error("Clear should remove all records")
	}
}

func TestProviderSubscribeNotSupported(t *testing.T) {
	p, _ := competitive.NewProvider(omnisignal.Config{})
	_, err := p.Subscribe(context.Background(), omnisignal.SubscribeOptions{})
	if err != omnisignal.ErrNotSupported {
		t.Errorf("Subscribe error = %v, want ErrNotSupported", err)
	}
}

func TestProviderClose(t *testing.T) {
	p, _ := competitive.NewProvider(omnisignal.Config{})
	if err := p.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestOutcomeConstants(t *testing.T) {
	if competitive.OutcomeWin != "win" {
		t.Error("OutcomeWin mismatch")
	}
	if competitive.OutcomeLoss != "loss" {
		t.Error("OutcomeLoss mismatch")
	}
}

func TestSourceConstants(t *testing.T) {
	if competitive.SourceSalesforce != "salesforce" {
		t.Error("SourceSalesforce mismatch")
	}
	if competitive.SourceHubSpot != "hubspot" {
		t.Error("SourceHubSpot mismatch")
	}
	if competitive.SourceClari != "clari" {
		t.Error("SourceClari mismatch")
	}
	if competitive.SourceGong != "gong" {
		t.Error("SourceGong mismatch")
	}
}

func TestDealRecordFields(t *testing.T) {
	now := time.Now()
	d := competitive.DealRecord{
		ID:                "d-123",
		OpportunityID:     "OPP-456",
		AccountName:       "Test Corp",
		AccountRef:        "customer:test-001",
		Outcome:           competitive.OutcomeLoss,
		Competitors:       []string{"comp1", "comp2"},
		PrimaryCompetitor: "comp1",
		Reasons:           []string{"reason1", "reason2"},
		DealValue:         100000_00,
		Market:            "test-market",
		ClosedAt:          now,
		Source:            competitive.SourceSalesforce,
		Metadata:          map[string]any{"key": "value"},
	}

	if d.ID != "d-123" {
		t.Error("ID mismatch")
	}
	if d.Outcome != competitive.OutcomeLoss {
		t.Error("Outcome mismatch")
	}
	if len(d.Competitors) != 2 {
		t.Error("Competitors mismatch")
	}
	if d.DealValue != 100000_00 {
		t.Error("DealValue mismatch")
	}
}

func TestCompetitiveGapFields(t *testing.T) {
	now := time.Now()
	g := competitive.CompetitiveGap{
		ID:           "g-123",
		Title:        "Test Gap",
		Description:  "Description",
		Competitor:   "competitor1",
		Capability:   "feature1",
		Impact:       "High impact",
		Severity:     common.SeverityHigh,
		Market:       "test-market",
		DealIDs:      []string{"d1", "d2"},
		IdentifiedAt: now,
		Source:       competitive.SourceClari,
		Metadata:     map[string]any{"key": "value"},
	}

	if g.ID != "g-123" {
		t.Error("ID mismatch")
	}
	if g.Severity != common.SeverityHigh {
		t.Error("Severity mismatch")
	}
	if len(g.DealIDs) != 2 {
		t.Error("DealIDs mismatch")
	}
}

func TestCustomerRefFromMappings(t *testing.T) {
	adapter := competitive.NewMemoryAdapter(competitive.SourceSalesforce)
	adapter.AddDeal(competitive.DealRecord{
		ID:          "d1",
		AccountName: "Acme Corp",
		Outcome:     competitive.OutcomeWin,
	})

	p, _ := competitive.NewProvider(omnisignal.Config{
		Options: map[string]any{
			"adapter":           adapter,
			"customer_mappings": map[string]string{"Acme Corp": "customer:acme-001"},
		},
	})

	signals, _ := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if len(signals) != 1 {
		t.Fatal("expected 1 signal")
	}

	if signals[0].Metadata[signal.MetaCustomerRef] != "customer:acme-001" {
		t.Errorf("customer_ref = %v, want customer:acme-001", signals[0].Metadata[signal.MetaCustomerRef])
	}
}
