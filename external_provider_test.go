package omnisignal_test

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

type mockAhaProvider struct {
	config omnisignal.Config
}

func newMockAhaProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	return &mockAhaProvider{config: cfg}, nil
}

func (p *mockAhaProvider) Name() string { return "aha" }

func (p *mockAhaProvider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	return []signal.Signal{
		{
			ID:     "aha-IDEA-100",
			Type:   signal.TypeEnhancementRequest,
			Status: signal.StatusNew,
			Source: common.SourceSystem{
				Type:       "product_management",
				Name:       "aha",
				ExternalID: "IDEA-100",
			},
			Domain:   common.Domain{Name: "product"},
			Severity: common.SeverityMedium,
			Summary:  "Test idea",
			Metadata: map[string]any{
				signal.MetaVotes:       10,
				signal.MetaSubscribers: 5,
				"curated":              true,
			},
			ObservedAt: time.Now(),
			ReceivedAt: time.Now(),
		},
	}, nil
}

func (p *mockAhaProvider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

func (p *mockAhaProvider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:  false,
		SupportsBatchFetch: true,
		SupportsFiltering:  true,
		MaxBatchSize:       200,
		RateLimitPerMinute: 300,
		SignalTypes:        []signal.Type{signal.TypeEnhancementRequest},
	}
}

func (p *mockAhaProvider) Close() error { return nil }

func TestExternalProviderRegistration(t *testing.T) {
	omnisignal.ClearRegistry()
	defer omnisignal.ClearRegistry()

	omnisignal.Register("aha", newMockAhaProvider, omnisignal.PriorityThick)

	if !omnisignal.IsRegistered("aha") {
		t.Fatal("aha provider not registered")
	}

	provider, err := omnisignal.New("aha", omnisignal.Config{
		BaseURL: "https://test.aha.io",
		APIKey:  "test-key",
	})
	if err != nil {
		t.Fatalf("creating aha provider: %v", err)
	}
	defer provider.Close()

	if provider.Name() != "aha" {
		t.Errorf("Name() = %s, want aha", provider.Name())
	}
}

func TestExternalProviderFetch(t *testing.T) {
	omnisignal.ClearRegistry()
	defer omnisignal.ClearRegistry()

	omnisignal.Register("aha", newMockAhaProvider, omnisignal.PriorityThick)

	provider, err := omnisignal.New("aha", omnisignal.Config{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}
	defer provider.Close()

	signals, err := provider.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}

	sig := signals[0]
	if sig.Type != signal.TypeEnhancementRequest {
		t.Errorf("Type = %s, want enhancement_request", sig.Type)
	}
	if sig.Metadata["curated"] != true {
		t.Error("expected curated=true in metadata")
	}
	if sig.Metadata[signal.MetaVotes] != 10 {
		t.Errorf("votes = %v, want 10", sig.Metadata[signal.MetaVotes])
	}
}

func TestExternalProviderCapabilities(t *testing.T) {
	omnisignal.ClearRegistry()
	defer omnisignal.ClearRegistry()

	omnisignal.Register("aha", newMockAhaProvider, omnisignal.PriorityThick)

	provider, err := omnisignal.New("aha", omnisignal.Config{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}
	defer provider.Close()

	caps := provider.Capabilities()
	if len(caps.SignalTypes) != 1 || caps.SignalTypes[0] != signal.TypeEnhancementRequest {
		t.Errorf("SignalTypes = %v, want [enhancement_request]", caps.SignalTypes)
	}
	if caps.MaxBatchSize != 200 {
		t.Errorf("MaxBatchSize = %d, want 200", caps.MaxBatchSize)
	}
}
