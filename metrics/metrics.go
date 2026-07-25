// Package metrics provides a formula registry for computing derived signal metrics.
//
// The registry allows pluggable Compute functions for metrics like frustration,
// momentum, reach, and urgency. Each formula is registered by name and can be
// retrieved and executed against a set of signals.
//
// Example:
//
//	import "github.com/plexusone/omnisignal/metrics"
//
//	// Compute frustration for a signal group
//	formula := metrics.Get("frustration")
//	result, err := formula.Compute(ctx, signals, metrics.Options{
//	    Weights: map[string]float64{"support_ticket": 1.5},
//	})
package metrics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/plexusone/signal-spec/pkg/signal"
)

// Common errors.
var (
	ErrFormulaNotFound = errors.New("formula not found")
	ErrNoSignals       = errors.New("no signals provided")
	ErrInvalidWeight   = errors.New("invalid weight value")
)

// Result holds the output of a metric computation.
type Result struct {
	// Name is the metric name (e.g., "frustration", "momentum").
	Name string

	// Value is the computed metric value.
	Value float64

	// Breakdown provides per-component contributions to the value.
	// Keys are component identifiers (signal types, sources, etc.).
	Breakdown map[string]float64

	// ComputedAt is when this result was generated.
	ComputedAt time.Time

	// SignalCount is the number of signals used in computation.
	SignalCount int

	// Metadata holds formula-specific additional data.
	Metadata map[string]any
}

// Options configures metric computation.
type Options struct {
	// Weights overrides default weights by signal type or source.
	// Keys can be signal types (e.g., "support_ticket") or source names
	// (e.g., "pagerduty"). More specific keys take precedence.
	Weights map[string]float64

	// Now overrides the current time for age calculations.
	// Zero value uses time.Now().
	Now time.Time

	// WindowDays limits signals to those observed within this many days.
	// Zero means no time window filter.
	WindowDays int

	// Extra holds formula-specific options.
	Extra map[string]any
}

// GetNow returns the effective "now" time for the options.
func (o Options) GetNow() time.Time {
	if o.Now.IsZero() {
		return time.Now()
	}
	return o.Now
}

// GetWeight returns the weight for a signal, checking type then source.
// Returns defaultWeight if no override is found.
func (o Options) GetWeight(typ signal.Type, sourceName string, defaultWeight float64) float64 {
	if o.Weights == nil {
		return defaultWeight
	}
	if w, ok := o.Weights[string(typ)]; ok {
		return w
	}
	if w, ok := o.Weights[sourceName]; ok {
		return w
	}
	return defaultWeight
}

// Formula defines a pluggable metric computation.
type Formula interface {
	// Name returns the metric identifier.
	Name() string

	// Description returns a brief explanation of what the metric measures.
	Description() string

	// Compute calculates the metric for the given signals.
	Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error)
}

// FormulaFunc adapts a function to the Formula interface.
type FormulaFunc struct {
	name        string
	description string
	compute     func(ctx context.Context, signals []signal.Signal, opts Options) (Result, error)
}

// Name returns the metric identifier.
func (f FormulaFunc) Name() string { return f.name }

// Description returns a brief explanation.
func (f FormulaFunc) Description() string { return f.description }

// Compute delegates to the wrapped function.
func (f FormulaFunc) Compute(ctx context.Context, signals []signal.Signal, opts Options) (Result, error) {
	return f.compute(ctx, signals, opts)
}

// NewFormula creates a Formula from a function.
func NewFormula(name, description string, fn func(ctx context.Context, signals []signal.Signal, opts Options) (Result, error)) Formula {
	return FormulaFunc{name: name, description: description, compute: fn}
}

var (
	registry = make(map[string]Formula)
	mu       sync.RWMutex
)

// Register adds a formula to the global registry.
// Later registrations with the same name override earlier ones.
func Register(f Formula) {
	mu.Lock()
	defer mu.Unlock()
	registry[f.Name()] = f
}

// Get retrieves a formula by name.
// Returns nil if not found.
func Get(name string) Formula {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// MustGet retrieves a formula, panicking if not found.
func MustGet(name string) Formula {
	f := Get(name)
	if f == nil {
		panic(fmt.Sprintf("metrics: formula %q not found", name))
	}
	return f
}

// List returns all registered formula names in alphabetical order.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsRegistered checks if a formula is registered.
func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[name]
	return ok
}

// Unregister removes a formula from the registry.
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(registry, name)
}

// ClearRegistry removes all formulas. Primarily for testing.
func ClearRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Formula)
}

// builtinFormulas holds formulas registered during init() so they can be restored.
var builtinFormulas []Formula

// saveBuiltin records a formula as a builtin during init().
// Called by each formula's init() function.
func saveBuiltin(f Formula) {
	builtinFormulas = append(builtinFormulas, f)
	Register(f)
}

// RegisterBuiltins re-registers all built-in formulas.
// Use after ClearRegistry() in tests that need the standard formulas.
func RegisterBuiltins() {
	for _, f := range builtinFormulas {
		Register(f)
	}
}

// Compute is a convenience function that gets a formula and computes it.
// Returns ErrFormulaNotFound if the formula doesn't exist.
func Compute(ctx context.Context, name string, signals []signal.Signal, opts Options) (Result, error) {
	f := Get(name)
	if f == nil {
		return Result{}, fmt.Errorf("%w: %s", ErrFormulaNotFound, name)
	}
	return f.Compute(ctx, signals, opts)
}

// ComputeAll runs all registered formulas and returns their results.
// Errors from individual formulas are collected in the returned error slice.
func ComputeAll(ctx context.Context, signals []signal.Signal, opts Options) ([]Result, []error) {
	names := List()
	results := make([]Result, 0, len(names))
	var errs []error

	for _, name := range names {
		result, err := Compute(ctx, name, signals, opts)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		results = append(results, result)
	}

	return results, errs
}
