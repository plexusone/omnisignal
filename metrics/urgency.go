package metrics

import (
	"context"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

// DefaultSeverityWeights maps severity levels to weights.
var DefaultSeverityWeights = map[common.Severity]float64{
	common.SeverityCritical: 4.0,
	common.SeverityHigh:     3.0,
	common.SeverityMedium:   2.0,
	common.SeverityLow:      1.0,
	common.SeverityInfo:     0.5,
}

func init() {
	saveBuiltin(urgencyFormula{})
}

type urgencyFormula struct{}

func (urgencyFormula) Name() string { return "urgency" }

func (urgencyFormula) Description() string {
	return "Sum of severity-weighted signal counts"
}

func (f urgencyFormula) Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error) {
	now := opts.GetNow()
	var total float64
	breakdown := make(map[string]float64)
	countBySeverity := make(map[string]int)

	for _, sig := range signals {
		if opts.WindowDays > 0 {
			cutoff := now.AddDate(0, 0, -opts.WindowDays)
			if sig.ObservedAt.Before(cutoff) {
				continue
			}
		}

		weight := f.getSeverityWeight(sig.Severity, opts)
		total += weight
		breakdown[string(sig.Severity)] += weight
		countBySeverity[string(sig.Severity)]++
	}

	return Result{
		Name:        f.Name(),
		Value:       total,
		Breakdown:   breakdown,
		ComputedAt:  now,
		SignalCount: len(signals),
		Metadata: map[string]any{
			"count_by_severity": countBySeverity,
		},
	}, nil
}

func (urgencyFormula) getSeverityWeight(sev common.Severity, opts Options) float64 {
	if opts.Weights != nil {
		if w, ok := opts.Weights[string(sev)]; ok {
			return w
		}
	}
	if w, ok := DefaultSeverityWeights[sev]; ok {
		return w
	}
	return 1.0
}
