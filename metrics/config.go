package metrics

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds metrics configuration including weight overrides.
type Config struct {
	// Weights overrides default weights for signal types, sources, or severities.
	Weights map[string]float64 `json:"weights,omitempty"`

	// WindowDays is the default trailing window for time-bounded metrics.
	WindowDays int `json:"window_days,omitempty"`

	// Extra holds formula-specific configuration.
	Extra map[string]any `json:"extra,omitempty"`
}

// LoadConfig reads metrics configuration from a JSON file.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// LoadConfigOrDefault reads metrics configuration from a JSON file.
// If the file doesn't exist, returns an empty Config (no error).
func LoadConfigOrDefault(path string) Config {
	cfg, err := LoadConfig(path)
	if err != nil {
		return Config{}
	}
	return cfg
}

// ToOptions converts Config to Options for use with Compute.
func (c Config) ToOptions() Options {
	return Options{
		Weights:    c.Weights,
		WindowDays: c.WindowDays,
		Extra:      c.Extra,
	}
}

// Merge combines two Configs. Values from other override values from c.
func (c Config) Merge(other Config) Config {
	result := Config{
		Weights:    make(map[string]float64),
		WindowDays: c.WindowDays,
		Extra:      make(map[string]any),
	}

	for k, v := range c.Weights {
		result.Weights[k] = v
	}
	for k, v := range other.Weights {
		result.Weights[k] = v
	}

	if other.WindowDays > 0 {
		result.WindowDays = other.WindowDays
	}

	for k, v := range c.Extra {
		result.Extra[k] = v
	}
	for k, v := range other.Extra {
		result.Extra[k] = v
	}

	return result
}

// ConfigFromProviderOptions extracts metrics Config from omnisignal.Config.Options.
// Looks for "metrics" key containing a Config-shaped object.
func ConfigFromProviderOptions(opts map[string]any) Config {
	if opts == nil {
		return Config{}
	}

	metricsOpts, ok := opts["metrics"]
	if !ok {
		return Config{}
	}

	data, err := json.Marshal(metricsOpts)
	if err != nil {
		return Config{}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}

	return cfg
}
