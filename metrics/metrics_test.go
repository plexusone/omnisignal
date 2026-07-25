package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnisignal/metrics"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestRegisterAndGet(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	called := false
	f := metrics.NewFormula("test_metric", "A test metric", func(ctx context.Context, signals []signal.Signal, opts metrics.Options) (metrics.Result, error) {
		called = true
		return metrics.Result{
			Name:        "test_metric",
			Value:       42.0,
			ComputedAt:  time.Now(),
			SignalCount: len(signals),
		}, nil
	})

	metrics.Register(f)

	if !metrics.IsRegistered("test_metric") {
		t.Error("formula should be registered")
	}

	got := metrics.Get("test_metric")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name() != "test_metric" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test_metric")
	}
	if got.Description() != "A test metric" {
		t.Errorf("Description() = %q, want %q", got.Description(), "A test metric")
	}

	result, err := got.Compute(context.Background(), nil, metrics.Options{})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}
	if !called {
		t.Error("compute function was not called")
	}
	if result.Value != 42.0 {
		t.Errorf("Value = %f, want 42.0", result.Value)
	}
}

func TestList(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	metrics.Register(metrics.NewFormula("zebra", "", nil))
	metrics.Register(metrics.NewFormula("alpha", "", nil))
	metrics.Register(metrics.NewFormula("beta", "", nil))

	names := metrics.List()
	if len(names) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" || names[2] != "zebra" {
		t.Errorf("List() = %v, want [alpha beta zebra]", names)
	}
}

func TestUnregister(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	metrics.Register(metrics.NewFormula("temp", "", nil))
	if !metrics.IsRegistered("temp") {
		t.Error("should be registered")
	}

	metrics.Unregister("temp")
	if metrics.IsRegistered("temp") {
		t.Error("should be unregistered")
	}
}

func TestGetNotFound(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	if metrics.Get("nonexistent") != nil {
		t.Error("Get should return nil for nonexistent formula")
	}
}

func TestMustGetPanics(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for nonexistent formula")
		}
	}()

	metrics.MustGet("nonexistent")
}

func TestCompute(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	metrics.Register(metrics.NewFormula("simple", "Simple metric", func(ctx context.Context, signals []signal.Signal, opts metrics.Options) (metrics.Result, error) {
		return metrics.Result{
			Name:        "simple",
			Value:       float64(len(signals)),
			ComputedAt:  opts.GetNow(),
			SignalCount: len(signals),
		}, nil
	}))

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket},
		{ID: "2", Type: signal.TypeSupportTicket},
	}

	result, err := metrics.Compute(context.Background(), "simple", signals, metrics.Options{})
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}
	if result.Value != 2.0 {
		t.Errorf("Value = %f, want 2.0", result.Value)
	}
}

func TestComputeNotFound(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	_, err := metrics.Compute(context.Background(), "missing", nil, metrics.Options{})
	if err == nil {
		t.Error("Compute should error for missing formula")
	}
}

func TestComputeAll(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	metrics.Register(metrics.NewFormula("a", "", func(ctx context.Context, signals []signal.Signal, opts metrics.Options) (metrics.Result, error) {
		return metrics.Result{Name: "a", Value: 1.0}, nil
	}))
	metrics.Register(metrics.NewFormula("b", "", func(ctx context.Context, signals []signal.Signal, opts metrics.Options) (metrics.Result, error) {
		return metrics.Result{Name: "b", Value: 2.0}, nil
	}))

	results, errs := metrics.ComputeAll(context.Background(), nil, metrics.Options{})
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestOptionsGetWeight(t *testing.T) {
	opts := metrics.Options{
		Weights: map[string]float64{
			"support_ticket": 1.5,
			"jira":           2.0,
		},
	}

	tests := []struct {
		typ        signal.Type
		source     string
		defaultW   float64
		wantWeight float64
	}{
		{signal.TypeSupportTicket, "zendesk", 1.0, 1.5},
		{signal.TypeCloudIncident, "jira", 1.0, 2.0},
		{signal.TypeCloudIncident, "pagerduty", 1.0, 1.0},
		{signal.TypeAlert, "datadog", 3.0, 3.0},
	}

	for _, tt := range tests {
		got := opts.GetWeight(tt.typ, tt.source, tt.defaultW)
		if got != tt.wantWeight {
			t.Errorf("GetWeight(%s, %s, %f) = %f, want %f", tt.typ, tt.source, tt.defaultW, got, tt.wantWeight)
		}
	}
}

func TestOptionsGetNow(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	opts := metrics.Options{Now: now}
	if got := opts.GetNow(); !got.Equal(now) {
		t.Errorf("GetNow() = %v, want %v", got, now)
	}

	opts = metrics.Options{}
	before := time.Now()
	got := opts.GetNow()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("GetNow() = %v, not in range [%v, %v]", got, before, after)
	}
}

func TestFormulaFuncImplementsInterface(t *testing.T) {
	var _ metrics.Formula = metrics.FormulaFunc{}
}

func TestResultFields(t *testing.T) {
	now := time.Now()
	r := metrics.Result{
		Name:        "test",
		Value:       99.5,
		Breakdown:   map[string]float64{"a": 50, "b": 49.5},
		ComputedAt:  now,
		SignalCount: 10,
		Metadata:    map[string]any{"extra": true},
	}

	if r.Name != "test" {
		t.Errorf("Name = %s", r.Name)
	}
	if r.Value != 99.5 {
		t.Errorf("Value = %f", r.Value)
	}
	if len(r.Breakdown) != 2 {
		t.Errorf("Breakdown has %d entries", len(r.Breakdown))
	}
	if !r.ComputedAt.Equal(now) {
		t.Errorf("ComputedAt mismatch")
	}
	if r.SignalCount != 10 {
		t.Errorf("SignalCount = %d", r.SignalCount)
	}
	if r.Metadata["extra"] != true {
		t.Errorf("Metadata mismatch")
	}
}

func TestOptionsWithSignals(t *testing.T) {
	metrics.ClearRegistry()
	defer metrics.RegisterBuiltins()

	metrics.Register(metrics.NewFormula("weighted", "Weighted sum", func(ctx context.Context, signals []signal.Signal, opts metrics.Options) (metrics.Result, error) {
		var total float64
		breakdown := make(map[string]float64)

		for _, sig := range signals {
			w := opts.GetWeight(sig.Type, sig.Source.Name, 1.0)
			total += w
			breakdown[string(sig.Type)] += w
		}

		return metrics.Result{
			Name:        "weighted",
			Value:       total,
			Breakdown:   breakdown,
			ComputedAt:  opts.GetNow(),
			SignalCount: len(signals),
		}, nil
	}))

	signals := []signal.Signal{
		{ID: "1", Type: signal.TypeSupportTicket, Source: common.SourceSystem{Name: "zendesk"}},
		{ID: "2", Type: signal.TypeSupportTicket, Source: common.SourceSystem{Name: "zendesk"}},
		{ID: "3", Type: signal.TypeCloudIncident, Source: common.SourceSystem{Name: "pagerduty"}},
	}

	opts := metrics.Options{
		Weights: map[string]float64{
			"support_ticket": 2.0,
			"cloud_incident": 3.0,
		},
	}

	result, err := metrics.Compute(context.Background(), "weighted", signals, opts)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	expectedValue := 2.0 + 2.0 + 3.0
	if result.Value != expectedValue {
		t.Errorf("Value = %f, want %f", result.Value, expectedValue)
	}
	if result.Breakdown["support_ticket"] != 4.0 {
		t.Errorf("Breakdown[support_ticket] = %f, want 4.0", result.Breakdown["support_ticket"])
	}
	if result.Breakdown["cloud_incident"] != 3.0 {
		t.Errorf("Breakdown[cloud_incident] = %f, want 3.0", result.Breakdown["cloud_incident"])
	}
}
