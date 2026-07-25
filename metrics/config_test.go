package metrics_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/plexusone/omnisignal/metrics"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")

	content := `{
		"weights": {
			"support_ticket": 2.0,
			"pagerduty": 3.0,
			"critical": 5.0
		},
		"window_days": 14,
		"extra": {
			"custom_key": "custom_value"
		}
	}`

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	cfg, err := metrics.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Weights["support_ticket"] != 2.0 {
		t.Errorf("Weights[support_ticket] = %f, want 2.0", cfg.Weights["support_ticket"])
	}
	if cfg.Weights["pagerduty"] != 3.0 {
		t.Errorf("Weights[pagerduty] = %f, want 3.0", cfg.Weights["pagerduty"])
	}
	if cfg.Weights["critical"] != 5.0 {
		t.Errorf("Weights[critical] = %f, want 5.0", cfg.Weights["critical"])
	}
	if cfg.WindowDays != 14 {
		t.Errorf("WindowDays = %d, want 14", cfg.WindowDays)
	}
	if cfg.Extra["custom_key"] != "custom_value" {
		t.Errorf("Extra[custom_key] = %v", cfg.Extra["custom_key"])
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := metrics.LoadConfig("/nonexistent/path/metrics.json")
	if err == nil {
		t.Error("LoadConfig should error for nonexistent file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	_, err := metrics.LoadConfig(path)
	if err == nil {
		t.Error("LoadConfig should error for invalid JSON")
	}
}

func TestLoadConfigOrDefault(t *testing.T) {
	cfg := metrics.LoadConfigOrDefault("/nonexistent/path/metrics.json")

	if cfg.Weights != nil {
		t.Errorf("Weights should be nil, got %v", cfg.Weights)
	}
	if cfg.WindowDays != 0 {
		t.Errorf("WindowDays should be 0, got %d", cfg.WindowDays)
	}
}

func TestConfigToOptions(t *testing.T) {
	cfg := metrics.Config{
		Weights: map[string]float64{
			"support_ticket": 2.0,
		},
		WindowDays: 7,
		Extra: map[string]any{
			"key": "value",
		},
	}

	opts := cfg.ToOptions()

	if opts.Weights["support_ticket"] != 2.0 {
		t.Errorf("Weights[support_ticket] = %f, want 2.0", opts.Weights["support_ticket"])
	}
	if opts.WindowDays != 7 {
		t.Errorf("WindowDays = %d, want 7", opts.WindowDays)
	}
	if opts.Extra["key"] != "value" {
		t.Errorf("Extra[key] = %v", opts.Extra["key"])
	}
}

func TestConfigMerge(t *testing.T) {
	base := metrics.Config{
		Weights: map[string]float64{
			"support_ticket": 1.0,
			"cloud_incident": 2.0,
		},
		WindowDays: 30,
		Extra: map[string]any{
			"base_key": "base_value",
		},
	}

	override := metrics.Config{
		Weights: map[string]float64{
			"support_ticket": 5.0,  // Override
			"outage":         10.0, // Add new
		},
		WindowDays: 7,
		Extra: map[string]any{
			"override_key": "override_value",
		},
	}

	merged := base.Merge(override)

	if merged.Weights["support_ticket"] != 5.0 {
		t.Errorf("Weights[support_ticket] = %f, want 5.0 (overridden)", merged.Weights["support_ticket"])
	}
	if merged.Weights["cloud_incident"] != 2.0 {
		t.Errorf("Weights[cloud_incident] = %f, want 2.0 (from base)", merged.Weights["cloud_incident"])
	}
	if merged.Weights["outage"] != 10.0 {
		t.Errorf("Weights[outage] = %f, want 10.0 (added)", merged.Weights["outage"])
	}
	if merged.WindowDays != 7 {
		t.Errorf("WindowDays = %d, want 7 (overridden)", merged.WindowDays)
	}
	if merged.Extra["base_key"] != "base_value" {
		t.Errorf("Extra[base_key] should be preserved")
	}
	if merged.Extra["override_key"] != "override_value" {
		t.Errorf("Extra[override_key] should be added")
	}
}

func TestConfigMergePreservesWindowDays(t *testing.T) {
	base := metrics.Config{WindowDays: 30}
	override := metrics.Config{} // Zero WindowDays

	merged := base.Merge(override)

	if merged.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30 (preserved from base)", merged.WindowDays)
	}
}

func TestConfigFromProviderOptions(t *testing.T) {
	opts := map[string]any{
		"metrics": map[string]any{
			"weights": map[string]any{
				"support_ticket": 3.0,
			},
			"window_days": 14,
		},
	}

	cfg := metrics.ConfigFromProviderOptions(opts)

	if cfg.Weights["support_ticket"] != 3.0 {
		t.Errorf("Weights[support_ticket] = %f, want 3.0", cfg.Weights["support_ticket"])
	}
	if cfg.WindowDays != 14 {
		t.Errorf("WindowDays = %d, want 14", cfg.WindowDays)
	}
}

func TestConfigFromProviderOptionsNil(t *testing.T) {
	cfg := metrics.ConfigFromProviderOptions(nil)

	if cfg.Weights != nil {
		t.Errorf("Weights should be nil")
	}
}

func TestConfigFromProviderOptionsMissingMetrics(t *testing.T) {
	opts := map[string]any{
		"other": "value",
	}

	cfg := metrics.ConfigFromProviderOptions(opts)

	if cfg.Weights != nil {
		t.Errorf("Weights should be nil")
	}
}

func TestConfigFromProviderOptionsInvalid(t *testing.T) {
	opts := map[string]any{
		"metrics": "not a map",
	}

	cfg := metrics.ConfigFromProviderOptions(opts)

	if cfg.Weights != nil {
		t.Errorf("Weights should be nil for invalid metrics value")
	}
}

func TestFullWorkflow(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.json")
	orgPath := filepath.Join(dir, "org.json")

	globalContent := `{
		"weights": {
			"support_ticket": 1.0,
			"cloud_incident": 2.0,
			"outage": 3.0
		},
		"window_days": 30
	}`

	orgContent := `{
		"weights": {
			"support_ticket": 10.0
		},
		"window_days": 7
	}`

	if err := os.WriteFile(globalPath, []byte(globalContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orgPath, []byte(orgContent), 0600); err != nil {
		t.Fatal(err)
	}

	global, err := metrics.LoadConfig(globalPath)
	if err != nil {
		t.Fatal(err)
	}

	org, err := metrics.LoadConfig(orgPath)
	if err != nil {
		t.Fatal(err)
	}

	effective := global.Merge(org)
	opts := effective.ToOptions()

	if opts.Weights["support_ticket"] != 10.0 {
		t.Errorf("support_ticket should be 10.0 (org override)")
	}
	if opts.Weights["cloud_incident"] != 2.0 {
		t.Errorf("cloud_incident should be 2.0 (from global)")
	}
	if opts.Weights["outage"] != 3.0 {
		t.Errorf("outage should be 3.0 (from global)")
	}
	if opts.WindowDays != 7 {
		t.Errorf("WindowDays should be 7 (org override)")
	}
}
