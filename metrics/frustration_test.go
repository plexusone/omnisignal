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

func TestFrustrationRegistered(t *testing.T) {
	if !metrics.IsRegistered("frustration") {
		t.Fatal("frustration formula not registered")
	}
}

func TestFrustrationEmptySignals(t *testing.T) {
	result, err := metrics.Compute(context.Background(), "frustration", nil, metrics.Options{})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}
	if result.Value != 0 {
		t.Errorf("Value = %f, want 0", result.Value)
	}
	if result.SignalCount != 0 {
		t.Errorf("SignalCount = %d, want 0", result.SignalCount)
	}
}

func TestFrustrationBasicCalculation(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	oldest := now.AddDate(0, 0, -10) // 10 days ago

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			ObservedAt: oldest,
			Source:     common.SourceSystem{Name: "zendesk"},
		},
		{
			ID:         "2",
			Type:       signal.TypeSupportTicket,
			ObservedAt: now.AddDate(0, 0, -5), // 5 days ago
			Source:     common.SourceSystem{Name: "zendesk"},
		},
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// 2 tickets x 1.0 weight = 2.0 weighted count
	// 10 days age
	// 2.0 * 10 = 20.0
	expected := 20.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}

	meta := result.Metadata
	if meta["weighted_count"].(float64) != 2.0 {
		t.Errorf("weighted_count = %v, want 2.0", meta["weighted_count"])
	}
	if math.Abs(meta["oldest_age_days"].(float64)-10.0) > 0.001 {
		t.Errorf("oldest_age_days = %v, want 10.0", meta["oldest_age_days"])
	}
}

func TestFrustrationDefaultWeights(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	oneDay := now.AddDate(0, 0, -1)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: oneDay}, // weight 1.0
		{ID: "2", Type: signal.TypeCloudIncident, ObservedAt: oneDay}, // weight 2.0
		{ID: "3", Type: signal.TypeOutage, ObservedAt: oneDay},        // weight 3.0
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// (1.0 + 2.0 + 3.0) * 1 day = 6.0
	expected := 6.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}

	if result.Breakdown["support_ticket"] != 1.0 {
		t.Errorf("Breakdown[support_ticket] = %f, want 1.0", result.Breakdown["support_ticket"])
	}
	if result.Breakdown["cloud_incident"] != 2.0 {
		t.Errorf("Breakdown[cloud_incident] = %f, want 2.0", result.Breakdown["cloud_incident"])
	}
	if result.Breakdown["outage"] != 3.0 {
		t.Errorf("Breakdown[outage] = %f, want 3.0", result.Breakdown["outage"])
	}
}

func TestFrustrationCustomWeights(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	oneDay := now.AddDate(0, 0, -1)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: oneDay},
		{ID: "2", Type: signal.TypeSupportTicket, ObservedAt: oneDay},
	}

	opts := metrics.Options{
		Now: now,
		Weights: map[string]float64{
			"support_ticket": 5.0, // Override default 1.0
		},
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, opts)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// 2 tickets x 5.0 weight * 1 day = 10.0
	expected := 10.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}
}

func TestFrustrationSourceWeight(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	oneDay := now.AddDate(0, 0, -1)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeFeedback, ObservedAt: oneDay, Source: common.SourceSystem{Name: "aha"}},
	}

	opts := metrics.Options{
		Now: now,
		Weights: map[string]float64{
			"aha": 10.0, // Source-specific weight
		},
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, opts)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// 1 signal x 10.0 (source weight) * 1 day = 10.0
	expected := 10.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}
}

func TestFrustrationTypeWeightTakesPrecedence(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	oneDay := now.AddDate(0, 0, -1)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: oneDay, Source: common.SourceSystem{Name: "zendesk"}},
	}

	opts := metrics.Options{
		Now: now,
		Weights: map[string]float64{
			"support_ticket": 3.0, // Type weight
			"zendesk":        7.0, // Source weight (should not be used)
		},
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, opts)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// Type weight takes precedence: 3.0 * 1 day = 3.0
	expected := 3.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}
}

func TestFrustrationWindowDays(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -5)},  // 5 days ago - included
		{ID: "2", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -15)}, // 15 days ago - excluded
		{ID: "3", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -30)}, // 30 days ago - excluded
	}

	opts := metrics.Options{
		Now:        now,
		WindowDays: 7, // Only include last 7 days
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, opts)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// Only 1 signal included, weight 1.0, age 5 days
	expected := 5.0
	if math.Abs(result.Value-expected) > 0.001 {
		t.Errorf("Value = %f, want %f", result.Value, expected)
	}

	meta := result.Metadata
	if meta["weighted_count"].(float64) != 1.0 {
		t.Errorf("weighted_count = %v, want 1.0 (only 1 signal in window)", meta["weighted_count"])
	}
}

func TestFrustrationZeroAge(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now},
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// 1 ticket x 1.0 weight x 0 days = 0
	if result.Value != 0 {
		t.Errorf("Value = %f, want 0 (signal observed at 'now')", result.Value)
	}
}

func TestFrustrationFutureSignal(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, 1)}, // Future
	}

	result, err := metrics.Compute(context.Background(), "frustration", signals, metrics.Options{Now: now})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	// Future signals should have 0 age
	if result.Value != 0 {
		t.Errorf("Value = %f, want 0 (future signal)", result.Value)
	}
}

func TestDefaultFrustrationWeights(t *testing.T) {
	expected := map[string]float64{
		"support_ticket":      1.0,
		"cloud_incident":      2.0,
		"security_finding":    1.5,
		"posture_drift":       0.5,
		"alert":               0.5,
		"outage":              3.0,
		"vulnerability":       1.0,
		"feedback":            0.8,
		"enhancement_request": 0.5,
		"competitive_gap":     0.3,
		"competitor_launch":   0.2,
		"analyst_finding":     0.2,
		"market_observation":  0.1,
	}

	for typ, weight := range expected {
		if got, ok := metrics.DefaultFrustrationWeights[typ]; !ok {
			t.Errorf("missing default weight for %s", typ)
		} else if got != weight {
			t.Errorf("DefaultFrustrationWeights[%s] = %f, want %f", typ, got, weight)
		}
	}
}
