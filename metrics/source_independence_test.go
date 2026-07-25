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

// skipIfFormulasNotRegistered skips the test if standard formulas aren't registered.
// This handles test ordering issues when other tests call ClearRegistry().
func skipIfFormulasNotRegistered(t *testing.T) {
	t.Helper()
	for _, name := range []string{"frustration", "momentum", "reach", "urgency"} {
		if !metrics.IsRegistered(name) {
			t.Skipf("formula %q not registered (run with -count=1 or check test ordering)", name)
		}
	}
}

func TestSourceIndependence(t *testing.T) {
	skipIfFormulasNotRegistered(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	ahaSignals := []signal.Signal{
		{
			ID:         "aha-IDEA-100",
			Type:       signal.TypeEnhancementRequest,
			Severity:   common.SeverityMedium,
			Source:     common.SourceSystem{Name: "aha", Type: "product_management"},
			ObservedAt: now.AddDate(0, 0, -10),
			Metadata: map[string]any{
				"curated":      true,
				"votes":        42,
				"customer_ref": "customer:acme-001",
			},
		},
		{
			ID:         "aha-IDEA-101",
			Type:       signal.TypeEnhancementRequest,
			Severity:   common.SeverityHigh,
			Source:     common.SourceSystem{Name: "aha", Type: "product_management"},
			ObservedAt: now.AddDate(0, 0, -5),
			Metadata: map[string]any{
				"curated":      true,
				"votes":        15,
				"customer_ref": "customer:globex-002",
			},
		},
	}

	supportSignals := []signal.Signal{
		{
			ID:         "zendesk-12345",
			Type:       signal.TypeEnhancementRequest,
			Severity:   common.SeverityMedium,
			Source:     common.SourceSystem{Name: "zendesk", Type: "support"},
			ObservedAt: now.AddDate(0, 0, -10),
			Metadata: map[string]any{
				"customer_ref": "customer:acme-001",
			},
		},
		{
			ID:         "zendesk-12346",
			Type:       signal.TypeEnhancementRequest,
			Severity:   common.SeverityHigh,
			Source:     common.SourceSystem{Name: "zendesk", Type: "support"},
			ObservedAt: now.AddDate(0, 0, -5),
			Metadata: map[string]any{
				"customer_ref": "customer:globex-002",
			},
		},
	}

	opts := metrics.Options{Now: now}

	t.Run("frustration", func(t *testing.T) {
		ahaResult, err := metrics.Compute(context.Background(), "frustration", ahaSignals, opts)
		if err != nil {
			t.Fatal(err)
		}
		supportResult, err := metrics.Compute(context.Background(), "frustration", supportSignals, opts)
		if err != nil {
			t.Fatal(err)
		}

		if math.Abs(ahaResult.Value-supportResult.Value) > 0.001 {
			t.Errorf("frustration: aha=%f, support=%f - should be equal for same signal shapes", ahaResult.Value, supportResult.Value)
		}
	})

	t.Run("momentum", func(t *testing.T) {
		ahaResult, err := metrics.Compute(context.Background(), "momentum", ahaSignals, opts)
		if err != nil {
			t.Fatal(err)
		}
		supportResult, err := metrics.Compute(context.Background(), "momentum", supportSignals, opts)
		if err != nil {
			t.Fatal(err)
		}

		if ahaResult.Value != supportResult.Value {
			t.Errorf("momentum: aha=%f, support=%f - should be equal", ahaResult.Value, supportResult.Value)
		}
	})

	t.Run("reach", func(t *testing.T) {
		ahaResult, err := metrics.Compute(context.Background(), "reach", ahaSignals, opts)
		if err != nil {
			t.Fatal(err)
		}
		supportResult, err := metrics.Compute(context.Background(), "reach", supportSignals, opts)
		if err != nil {
			t.Fatal(err)
		}

		if ahaResult.Value != supportResult.Value {
			t.Errorf("reach: aha=%f, support=%f - should be equal", ahaResult.Value, supportResult.Value)
		}
	})

	t.Run("urgency", func(t *testing.T) {
		ahaResult, err := metrics.Compute(context.Background(), "urgency", ahaSignals, opts)
		if err != nil {
			t.Fatal(err)
		}
		supportResult, err := metrics.Compute(context.Background(), "urgency", supportSignals, opts)
		if err != nil {
			t.Fatal(err)
		}

		if math.Abs(ahaResult.Value-supportResult.Value) > 0.001 {
			t.Errorf("urgency: aha=%f, support=%f - should be equal", ahaResult.Value, supportResult.Value)
		}
	})
}

func TestTableDrivenFormulas(t *testing.T) {
	skipIfFormulasNotRegistered(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		formula   string
		signals   []signal.Signal
		opts      metrics.Options
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "frustration: empty signals",
			formula:   "frustration",
			signals:   nil,
			opts:      metrics.Options{Now: now},
			wantValue: 0,
		},
		{
			name:    "frustration: single signal 10 days old",
			formula: "frustration",
			signals: []signal.Signal{
				{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -10)},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 10.0, // 1.0 weight * 10 days
		},
		{
			name:    "frustration: multiple signals",
			formula: "frustration",
			signals: []signal.Signal{
				{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -10)},
				{ID: "2", Type: signal.TypeOutage, ObservedAt: now.AddDate(0, 0, -5)},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 40.0, // (1.0 + 3.0) * 10 days oldest
		},
		{
			name:    "frustration: with custom weight",
			formula: "frustration",
			signals: []signal.Signal{
				{ID: "1", Type: signal.TypeSupportTicket, ObservedAt: now.AddDate(0, 0, -1)},
			},
			opts: metrics.Options{
				Now:     now,
				Weights: map[string]float64{"support_ticket": 5.0},
			},
			wantValue: 5.0, // 5.0 weight * 1 day
		},
		{
			name:      "momentum: empty signals",
			formula:   "momentum",
			signals:   nil,
			opts:      metrics.Options{Now: now},
			wantValue: 0,
		},
		{
			name:    "momentum: all within window",
			formula: "momentum",
			signals: []signal.Signal{
				{ID: "1", ObservedAt: now.AddDate(0, 0, -5)},
				{ID: "2", ObservedAt: now.AddDate(0, 0, -10)},
				{ID: "3", ObservedAt: now.AddDate(0, 0, -25)},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 3,
		},
		{
			name:    "momentum: some outside window",
			formula: "momentum",
			signals: []signal.Signal{
				{ID: "1", ObservedAt: now.AddDate(0, 0, -5)},
				{ID: "2", ObservedAt: now.AddDate(0, 0, -35)},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 1,
		},
		{
			name:    "momentum: custom window",
			formula: "momentum",
			signals: []signal.Signal{
				{ID: "1", ObservedAt: now.AddDate(0, 0, -3)},
				{ID: "2", ObservedAt: now.AddDate(0, 0, -10)},
			},
			opts:      metrics.Options{Now: now, WindowDays: 7},
			wantValue: 1,
		},
		{
			name:      "reach: empty signals",
			formula:   "reach",
			signals:   nil,
			opts:      metrics.Options{Now: now},
			wantValue: 0,
		},
		{
			name:    "reach: distinct customers",
			formula: "reach",
			signals: []signal.Signal{
				{ID: "1", ObservedAt: now, Metadata: map[string]any{"customer_ref": "customer:a"}},
				{ID: "2", ObservedAt: now, Metadata: map[string]any{"customer_ref": "customer:b"}},
				{ID: "3", ObservedAt: now, Metadata: map[string]any{"customer_ref": "customer:a"}},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 2,
		},
		{
			name:    "reach: no customers",
			formula: "reach",
			signals: []signal.Signal{
				{ID: "1", ObservedAt: now},
				{ID: "2", ObservedAt: now},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 0,
		},
		{
			name:      "urgency: empty signals",
			formula:   "urgency",
			signals:   nil,
			opts:      metrics.Options{Now: now},
			wantValue: 0,
		},
		{
			name:    "urgency: mixed severities",
			formula: "urgency",
			signals: []signal.Signal{
				{ID: "1", Severity: common.SeverityCritical, ObservedAt: now},
				{ID: "2", Severity: common.SeverityMedium, ObservedAt: now},
				{ID: "3", Severity: common.SeverityLow, ObservedAt: now},
			},
			opts:      metrics.Options{Now: now},
			wantValue: 7.0, // 4.0 + 2.0 + 1.0
		},
		{
			name:    "urgency: custom severity weight",
			formula: "urgency",
			signals: []signal.Signal{
				{ID: "1", Severity: common.SeverityCritical, ObservedAt: now},
			},
			opts: metrics.Options{
				Now:     now,
				Weights: map[string]float64{"critical": 100.0},
			},
			wantValue: 100.0,
		},
		{
			name:    "unknown formula",
			formula: "nonexistent",
			signals: nil,
			opts:    metrics.Options{Now: now},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := metrics.Compute(context.Background(), tt.formula, tt.signals, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if math.Abs(result.Value-tt.wantValue) > 0.001 {
				t.Errorf("Value = %f, want %f", result.Value, tt.wantValue)
			}
		})
	}
}

func TestComputeAllFormulas(t *testing.T) {
	skipIfFormulasNotRegistered(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{
			ID:         "1",
			Type:       signal.TypeSupportTicket,
			Severity:   common.SeverityCritical,
			ObservedAt: now.AddDate(0, 0, -7),
			Metadata: map[string]any{
				"customer_ref": "customer:acme",
			},
		},
		{
			ID:         "2",
			Type:       signal.TypeCloudIncident,
			Severity:   common.SeverityHigh,
			ObservedAt: now.AddDate(0, 0, -3),
			Metadata: map[string]any{
				"customer_ref": "customer:globex",
			},
		},
	}

	results, errs := metrics.ComputeAll(context.Background(), signals, metrics.Options{Now: now})

	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if len(results) < 4 {
		t.Fatalf("expected at least 4 results, got %d", len(results))
	}

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
		if r.SignalCount != 2 {
			t.Errorf("%s: SignalCount = %d, want 2", r.Name, r.SignalCount)
		}
	}

	for _, expected := range []string{"frustration", "momentum", "reach", "urgency"} {
		if !names[expected] {
			t.Errorf("missing result for %s", expected)
		}
	}
}

func TestBreakdownConsistency(t *testing.T) {
	skipIfFormulasNotRegistered(t)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, Severity: common.SeverityCritical, ObservedAt: now},
		{ID: "2", Type: signal.TypeSupportTicket, Severity: common.SeverityMedium, ObservedAt: now},
		{ID: "3", Type: signal.TypeCloudIncident, Severity: common.SeverityCritical, ObservedAt: now},
	}

	opts := metrics.Options{Now: now}

	t.Run("frustration breakdown sums to weighted count", func(t *testing.T) {
		result, err := metrics.Compute(context.Background(), "frustration", signals, opts)
		if err != nil {
			t.Fatal(err)
		}

		var sum float64
		for _, v := range result.Breakdown {
			sum += v
		}

		weightedCount := result.Metadata["weighted_count"].(float64)
		if math.Abs(sum-weightedCount) > 0.001 {
			t.Errorf("breakdown sum (%f) != weighted_count (%f)", sum, weightedCount)
		}
	})

	t.Run("momentum breakdown sums to total", func(t *testing.T) {
		result, err := metrics.Compute(context.Background(), "momentum", signals, opts)
		if err != nil {
			t.Fatal(err)
		}

		var sum float64
		for _, v := range result.Breakdown {
			sum += v
		}

		if math.Abs(sum-result.Value) > 0.001 {
			t.Errorf("breakdown sum (%f) != Value (%f)", sum, result.Value)
		}
	})

	t.Run("urgency breakdown sums to total", func(t *testing.T) {
		result, err := metrics.Compute(context.Background(), "urgency", signals, opts)
		if err != nil {
			t.Fatal(err)
		}

		var sum float64
		for _, v := range result.Breakdown {
			sum += v
		}

		if math.Abs(sum-result.Value) > 0.001 {
			t.Errorf("breakdown sum (%f) != Value (%f)", sum, result.Value)
		}
	})
}
