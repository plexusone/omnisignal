package omnisignal_test

import (
	"testing"

	"github.com/plexusone/omnisignal"
)

func TestConfigGetStringMap(t *testing.T) {
	tests := []struct {
		name     string
		options  map[string]any
		key      string
		expected map[string]string
	}{
		{
			name:     "nil options",
			options:  nil,
			key:      "mappings",
			expected: nil,
		},
		{
			name: "map[string]string",
			options: map[string]any{
				"mappings": map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			key: "mappings",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "map[string]any with strings",
			options: map[string]any{
				"mappings": map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
			},
			key: "mappings",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "map[string]any with mixed types",
			options: map[string]any{
				"mappings": map[string]any{
					"key1": "value1",
					"key2": 42,
				},
			},
			key: "mappings",
			expected: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "missing key",
			options: map[string]any{
				"other": map[string]string{"a": "b"},
			},
			key:      "mappings",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := omnisignal.Config{Options: tt.options}
			result := cfg.GetStringMap(tt.key)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestIsCurated(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		want     bool
	}{
		{"nil metadata", nil, false},
		{"empty metadata", map[string]any{}, false},
		{"curated true", map[string]any{"curated": true}, true},
		{"curated false", map[string]any{"curated": false}, false},
		{"curated string", map[string]any{"curated": "true"}, false},
		{"curated int", map[string]any{"curated": 1}, false},
		{"other keys only", map[string]any{"votes": 10}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := omnisignal.IsCurated(tt.metadata)
			if got != tt.want {
				t.Errorf("IsCurated(%v) = %v, want %v", tt.metadata, got, tt.want)
			}
		})
	}
}

func TestRefMappingOptionKeys(t *testing.T) {
	if omnisignal.OptCustomerMappings != "customer_mappings" {
		t.Errorf("OptCustomerMappings = %s", omnisignal.OptCustomerMappings)
	}
	if omnisignal.OptCapabilityMappings != "capability_mappings" {
		t.Errorf("OptCapabilityMappings = %s", omnisignal.OptCapabilityMappings)
	}
	if omnisignal.OptMarketMappings != "market_mappings" {
		t.Errorf("OptMarketMappings = %s", omnisignal.OptMarketMappings)
	}
}
