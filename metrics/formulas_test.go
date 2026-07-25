package metrics_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/plexusone/omnisignal/metrics"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestMomentumRegistered(t *testing.T) {
	if !metrics.IsRegistered("momentum") {
		t.Fatal("momentum formula not registered")
	}
}

func TestMomentumBasic(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -5)},
		{ID: "2", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -10)},
		{ID: "3", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -25)},
		{ID: "4", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -35)}, // Outside default 30-day window
	}

	result, err := metrics.Compute(context.Background(), "momentum", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 3 {
		t.Errorf("Value = %f, want 3 (signals within 30-day window)", result.Value)
	}

	meta := result.Metadata
	if meta["window_days"].(int) != 30 {
		t.Errorf("window_days = %v, want 30", meta["window_days"])
	}
}

func TestMomentumCustomWindow(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -5)},
		{ID: "2", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -10)},
	}

	result, err := metrics.Compute(context.Background(), "momentum", signals, metrics.Options{
		Now:        now,
		WindowDays: 7,
	})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 1 {
		t.Errorf("Value = %f, want 1 (only signal from 5 days ago)", result.Value)
	}
}

func TestMomentumBreakdown(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -1)},
		{ID: "2", Type: signal.TypeCloudIncident, ObservedAt: now.AddDate(0, 0, -1)},
		{ID: "3", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -1)},
	}

	result, err := metrics.Compute(context.Background(), "momentum", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Breakdown["support_ticket"] != 2 {
		t.Errorf("Breakdown[support_ticket] = %f, want 2", result.Breakdown["support_ticket"])
	}
	if result.Breakdown["cloud_incident"] != 1 {
		t.Errorf("Breakdown[cloud_incident] = %f, want 1", result.Breakdown["cloud_incident"])
	}
}

func TestReachRegistered(t *testing.T) {
	if !metrics.IsRegistered("reach") {
		t.Fatal("reach formula not registered")
	}
}

func TestReachBasic(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now,
			Metadata: map[string]any{
				signal.MetaCustomerRef: "customer:acme-001",
			},
		},
		{
			ID:         "2",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now,
			Metadata: map[string]any{
				signal.MetaCustomerRef: "customer:globex-002",
			},
		},
		{
			ID:         "3",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now,
			Metadata: map[string]any{
				signal.MetaCustomerRef: "customer:acme-001", // Duplicate
			},
		},
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 2 {
		t.Errorf("Value = %f, want 2 (distinct customers)", result.Value)
	}
}

func TestReachFromCustomersArray(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeEnhancementRequest,
			ObservedAt: now,
			Metadata: map[string]any{
				signal.MetaCustomers: []string{"acme-001", "globex-002", "initech-003"},
			},
		},
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 3 {
		t.Errorf("Value = %f, want 3", result.Value)
	}
}

func TestReachFromEntities(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now,
			Entities: []common.Entity{
				{Type: "customer", Ref: "customer:wayne-enterprises"},
				{Type: "component", Ref: "capability:auth"},
			},
		},
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 1 {
		t.Errorf("Value = %f, want 1 (only customer entity)", result.Value)
	}
}

func TestReachDedupes(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now,
			Metadata:   map[string]any{signal.MetaCustomerRef: "customer:acme"},
			Entities:   []common.Entity{{Type: "customer", Ref: "customer:acme"}},
		},
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 1 {
		t.Errorf("Value = %f, want 1 (same ref in metadata and entity)", result.Value)
	}
}

func TestReachWindowDays(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now.AddDate(0, 0, -5),
			Metadata:   map[string]any{signal.MetaCustomerRef: "customer:recent"},
		},
		{
			ID:         "2",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now.AddDate(0, 0, -15),
			Metadata:   map[string]any{signal.MetaCustomerRef: "customer:old"},
		},
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{
		Now:        now,
		WindowDays: 7,
	})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 1 {
		t.Errorf("Value = %f, want 1 (only recent signal)", result.Value)
	}
}

func TestUrgencyRegistered(t *testing.T) {
	if !metrics.IsRegistered("urgency") {
		t.Fatal("urgency formula not registered")
	}
}

func TestUrgencyBasic(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, Severity: common.SeverityCritical, ObservedAt: now},
		{ID: "2", Type: signal.TypeSupportTicket, Severity: common.SeverityHigh, ObservedAt: now},
		{ID: "3", Type: signal.TypeSupportTicket, Severity: common.SeverityMedium, ObservedAt: now},
	}

	result, err := metrics.Compute(context.Background(), "urgency", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// 4.0 + 3.0 + 2.0 = 9.0
	expected := 9.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}
}

func TestUrgencyCustomWeights(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Severity: common.SeverityCritical, ObservedAt: now},
		{ID: "2", Severity: common.SeverityCritical, ObservedAt: now},
	}

	result, err := metrics.Compute(context.Background(), "urgency", signals, metrics.Options{
		Now: now,
		Weights: map[string]float64{
			"critical": 10.0,
		},
	})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 20.0 {
		t.Errorf("Value = %f, want 20.0", result.Value)
	}
}

func TestUrgencyBreakdown(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Severity: common.SeverityCritical, ObservedAt: now},
		{ID: "2", Severity: common.SeverityLow, ObservedAt: now},
		{ID: "3", Severity: common.SeverityLow, ObservedAt: now},
	}

	result, err := metrics.Compute(context.Background(), "urgency", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Breakdown["critical"] != 4.0 {
		t.Errorf("Breakdown[critical] = %f, want 4.0", result.Breakdown["critical"])
	}
	if result.Breakdown["low"] != 2.0 {
		t.Errorf("Breakdown[low] = %f, want 2.0 (2 signals x 1.0)", result.Breakdown["low"])
	}

	meta := result.Metadata
	counts := meta["count_by_severity"].(map[string]int)
	if counts["critical"] != 1 {
		t.Errorf("count_by_severity[critical] = %d, want 1", counts["critical"])
	}
	if counts["low"] != 2 {
		t.Errorf("count_by_severity[low] = %d, want 2", counts["low"])
	}
}

func TestUrgencyWindowDays(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Severity: common.SeverityCritical, ObservedAt: now.AddDate(0, 0, -1)},
		{ID: "2", Severity: common.SeverityCritical, ObservedAt: now.AddDate(0, 0, -10)},
	}

	result, err := metrics.Compute(context.Background(), "urgency", signals, metrics.Options{
		Now:        now,
		WindowDays: 7,
	})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.Value != 4.0 {
		t.Errorf("Value = %f, want 4.0 (only recent critical)", result.Value)
	}
}

func TestDefaultSeverityWeights(t *testing.T) {
	expected := map[common.Severity]float64{
		common.SeverityCritical: 4.0,
		common.SeverityHigh:     3.0,
		common.SeverityMedium:   2.0,
		common.SeverityLow:      1.0,
		common.SeverityInfo:     0.5,
	}

	for sev, weight := range expected {
		if got, ok := metrics.DefaultSeverityWeights[sev]; !ok {
			t.Errorf("missing default weight for %s", sev)
		} else if got != weight {
			t.Errorf("DefaultSeverityWeights[%s] = %f, want %f", sev, got, weight)
		}
	}
}

func TestAllFormulasRegistered(t *testing.T) {
	expected := []string{"frustration", "momentum", "reach", "urgency"}

	for _, name := range expected {
		if !metrics.IsRegistered(name) {
			t.Errorf("formula %q not registered", name)
		}
	}
}
