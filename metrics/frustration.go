package metrics

import (
	"context"
	"time"

	"github.com/plexusone/signal-spec/pkg/signal"
)

// Default weights by signal type for frustration score.
var DefaultFrustrationWeights = map[string]float64{
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

func init() {
	saveBuiltin(frustrationFormula{})
}

type frustrationFormula struct{}

func (frustrationFormula) Name() string { return "frustration" }

func (frustrationFormula) Description() string {
	return "Weighted signal count multiplied by the age of the oldest signal in days"
}

func (f frustrationFormula) Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error) {
	if len(signals) == 0 {
		return Result{
			Name:        f.Name(),
			Value:       0,
			ComputedAt:  opts.GetNow(),
			SignalCount: 0,
		}, nil
	}

	now := opts.GetNow()
	var weightedCount float64
	var oldest time.Time
	breakdown := make(map[string]float64)

	for _, sig := range signals {
		if opts.WindowDays > 0 {
			cutoff := now.AddDate(0, 0, -opts.WindowDays)
			if sig.ObservedAt.Before(cutoff) {
				continue
			}
		}

		weight := f.getWeight(sig, opts)
		weightedCount += weight
		breakdown[string(sig.Type)] += weight

		if oldest.IsZero() || sig.ObservedAt.Before(oldest) {
			oldest = sig.ObservedAt
		}
	}

	var ageDays float64
	if !oldest.IsZero() {
		ageDays = now.Sub(oldest).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
	}

	frustration := weightedCount * ageDays

	return Result{
		Name:        f.Name(),
		Value:       frustration,
		Breakdown:   breakdown,
		ComputedAt:  now,
		SignalCount: len(signals),
		Metadata: map[string]any{
			"weighted_count":  weightedCount,
			"oldest_age_days": ageDays,
		},
	}, nil
}

func (frustrationFormula) getWeight(sig signal.Signal, opts Options) float64 {
	if opts.Weights != nil {
		if w, ok := opts.Weights[string(sig.Type)]; ok {
			return w
		}
		if w, ok := opts.Weights[sig.Source.Name]; ok {
			return w
		}
	}
	if w, ok := DefaultFrustrationWeights[string(sig.Type)]; ok {
		return w
	}
	return 1.0
}
