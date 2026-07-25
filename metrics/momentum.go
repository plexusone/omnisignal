package metrics

import (
	"context"

	"github.com/plexusone/signal-spec/pkg/signal"
)

const DefaultMomentumWindowDays = 30

func init() {
	saveBuiltin(momentumFormula{})
}

type momentumFormula struct{}

func (momentumFormula) Name() string { return "momentum" }

func (momentumFormula) Description() string {
	return "Count of signals observed within the trailing window (default 30 days)"
}

func (f momentumFormula) Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error) {
	now := opts.GetNow()
	windowDays := opts.WindowDays
	if windowDays <= 0 {
		windowDays = DefaultMomentumWindowDays
	}
	cutoff := now.AddDate(0, 0, -windowDays)

	var count int
	breakdown := make(map[string]float64)

	for _, sig := range signals {
		if sig.ObservedAt.After(cutoff) || sig.ObservedAt.Equal(cutoff) {
			count++
			breakdown[string(sig.Type)]++
		}
	}

	return Result{
		Name:        f.Name(),
		Value:       float64(count),
		Breakdown:   breakdown,
		ComputedAt:  now,
		SignalCount: len(signals),
		Metadata: map[string]any{
			"window_days":     windowDays,
			"count_in_window": count,
		},
	}, nil
}
