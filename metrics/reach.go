package metrics

import (
	"context"

	"github.com/plexusone/signal-spec/pkg/signal"
)

func init() {
	saveBuiltin(reachFormula{})
}

type reachFormula struct{}

func (reachFormula) Name() string { return "reach" }

func (reachFormula) Description() string {
	return "Count of distinct customer references across all signals"
}

func (f reachFormula) Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error) {
	now := opts.GetNow()
	seen := make(map[string]struct{})
	breakdown := make(map[string]float64)

	for _, sig := range signals {
		if opts.WindowDays > 0 {
			cutoff := now.AddDate(0, 0, -opts.WindowDays)
			if sig.ObservedAt.Before(cutoff) {
				continue
			}
		}

		refs := extractCustomerRefs(sig)
		for _, ref := range refs {
			if _, ok := seen[ref]; !ok {
				seen[ref] = struct{}{}
				breakdown[string(sig.Type)]++
			}
		}
	}

	return Result{
		Name:        f.Name(),
		Value:       float64(len(seen)),
		Breakdown:   breakdown,
		ComputedAt:  now,
		SignalCount: len(signals),
		Metadata: map[string]any{
			"distinct_customers": len(seen),
		},
	}, nil
}

func extractCustomerRefs(sig signal.Signal) []string {
	var refs []string

	if sig.Metadata != nil {
		if ref, ok := sig.Metadata[signal.MetaCustomerRef].(string); ok && ref != "" {
			refs = append(refs, ref)
		}

		if customers, ok := sig.Metadata[signal.MetaCustomers].([]string); ok {
			refs = append(refs, customers...)
		}

		if customers, ok := sig.Metadata[signal.MetaCustomers].([]any); ok {
			for _, c := range customers {
				if s, ok := c.(string); ok && s != "" {
					refs = append(refs, s)
				}
			}
		}
	}

	for _, entity := range sig.Entities {
		if entity.Type == "customer" && entity.Ref != "" {
			refs = append(refs, entity.Ref)
		}
	}

	return refs
}
