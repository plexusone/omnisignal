package omnisignal_test

import (
	"testing"

	"github.com/plexusone/omnisignal"
	_ "github.com/plexusone/omnisignal/provider/jira"
	_ "github.com/plexusone/omnisignal/provider/pagerduty"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestJiraCapabilities(t *testing.T) {
	if !omnisignal.IsRegistered("jira") {
		t.Skip("jira provider not registered")
	}

	provider, err := omnisignal.New("jira", omnisignal.Config{
		BaseURL:   "https://test.atlassian.net",
		APIKey:    "user@example.com",
		APISecret: "test-token",
	})
	if err != nil {
		t.Fatalf("creating jira provider: %v", err)
	}
	defer provider.Close()

	caps := provider.Capabilities()

	t.Run("SignalTypes", func(t *testing.T) {
		expected := map[signal.Type]bool{
			signal.TypeSupportTicket: true,
			signal.TypeFeedback:      true,
		}

		if len(caps.SignalTypes) != len(expected) {
			t.Errorf("expected %d signal types, got %d", len(expected), len(caps.SignalTypes))
		}

		for _, st := range caps.SignalTypes {
			if !expected[st] {
				t.Errorf("unexpected signal type: %s", st)
			}
		}

		if !expected[signal.TypeSupportTicket] {
			t.Error("missing TypeSupportTicket - normalizeIssue produces this type")
		}
	})

	t.Run("BatchSize", func(t *testing.T) {
		if caps.MaxBatchSize != 100 {
			t.Errorf("MaxBatchSize = %d, want 100 (Jira default page size)", caps.MaxBatchSize)
		}
	})

	t.Run("RateLimit", func(t *testing.T) {
		if caps.RateLimitPerMinute != 0 {
			t.Logf("RateLimitPerMinute = %d (varies by Jira instance)", caps.RateLimitPerMinute)
		}
	})

	t.Run("Features", func(t *testing.T) {
		if !caps.SupportsBatchFetch {
			t.Error("SupportsBatchFetch should be true")
		}
		if !caps.SupportsFiltering {
			t.Error("SupportsFiltering should be true (JQL)")
		}
		if caps.SupportsStreaming {
			t.Error("SupportsStreaming should be false (no webhook receiver)")
		}
		if caps.SupportsAcknowledge {
			t.Error("SupportsAcknowledge should be false")
		}
	})
}

func TestPagerDutyCapabilities(t *testing.T) {
	if !omnisignal.IsRegistered("pagerduty") {
		t.Skip("pagerduty provider not registered")
	}

	provider, err := omnisignal.New("pagerduty", omnisignal.Config{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("creating pagerduty provider: %v", err)
	}
	defer provider.Close()

	caps := provider.Capabilities()

	t.Run("SignalTypes", func(t *testing.T) {
		expected := map[signal.Type]bool{
			signal.TypeAlert:  true,
			signal.TypeOutage: true,
		}

		if len(caps.SignalTypes) != len(expected) {
			t.Errorf("expected %d signal types, got %d", len(expected), len(caps.SignalTypes))
		}

		for _, st := range caps.SignalTypes {
			if !expected[st] {
				t.Errorf("unexpected signal type: %s", st)
			}
		}

		if !expected[signal.TypeAlert] {
			t.Error("missing TypeAlert - normalizeIncident produces this type")
		}
		if !expected[signal.TypeOutage] {
			t.Error("missing TypeOutage - normalizeIncident produces this for resolved high-urgency")
		}
	})

	t.Run("BatchSize", func(t *testing.T) {
		if caps.MaxBatchSize != 100 {
			t.Errorf("MaxBatchSize = %d, want 100 (PagerDuty default page size)", caps.MaxBatchSize)
		}
	})

	t.Run("RateLimit", func(t *testing.T) {
		if caps.RateLimitPerMinute != 900 {
			t.Errorf("RateLimitPerMinute = %d, want 900 (PagerDuty API limit)", caps.RateLimitPerMinute)
		}
	})

	t.Run("Features", func(t *testing.T) {
		if !caps.SupportsBatchFetch {
			t.Error("SupportsBatchFetch should be true")
		}
		if !caps.SupportsFiltering {
			t.Error("SupportsFiltering should be true (urgency, status filters)")
		}
		if caps.SupportsStreaming {
			t.Error("SupportsStreaming should be false (no webhook receiver)")
		}
		if !caps.SupportsAcknowledge {
			t.Error("SupportsAcknowledge should be true (PagerDuty supports ack)")
		}
	})
}

func TestCapabilitiesSignalTypesMatchProducedTypes(t *testing.T) {
	t.Run("jira", func(t *testing.T) {
		if !omnisignal.IsRegistered("jira") {
			t.Skip("jira provider not registered")
		}
		provider, err := omnisignal.New("jira", omnisignal.Config{
			BaseURL:   "https://test.atlassian.net",
			APIKey:    "user@example.com",
			APISecret: "test-token",
		})
		if err != nil {
			t.Fatalf("creating provider: %v", err)
		}
		defer provider.Close()

		caps := provider.Capabilities()
		typeSet := make(map[signal.Type]bool)
		for _, st := range caps.SignalTypes {
			typeSet[st] = true
		}

		if !typeSet[signal.TypeSupportTicket] {
			t.Error("Capabilities.SignalTypes missing TypeSupportTicket but normalizeIssue always produces it")
		}
	})

	t.Run("pagerduty", func(t *testing.T) {
		if !omnisignal.IsRegistered("pagerduty") {
			t.Skip("pagerduty provider not registered")
		}
		provider, err := omnisignal.New("pagerduty", omnisignal.Config{
			APIKey: "test-api-key",
		})
		if err != nil {
			t.Fatalf("creating provider: %v", err)
		}
		defer provider.Close()

		caps := provider.Capabilities()
		typeSet := make(map[signal.Type]bool)
		for _, st := range caps.SignalTypes {
			typeSet[st] = true
		}

		if !typeSet[signal.TypeAlert] {
			t.Error("Capabilities.SignalTypes missing TypeAlert but normalizeIncident produces it")
		}
		if !typeSet[signal.TypeOutage] {
			t.Error("Capabilities.SignalTypes missing TypeOutage but normalizeIncident produces it for resolved/high-urgency")
		}
	})
}
