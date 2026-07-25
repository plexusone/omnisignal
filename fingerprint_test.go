package omnisignal_test

import (
	"testing"
	"time"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestJiraFingerprintStability(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)

	sig := signal.Signal{
		ID:   "jira-INFRA-123",
		Type: signal.TypeSupportTicket,
		Source: common.SourceSystem{
			Type:       "ticketing",
			Name:       "jira",
			ExternalID: "INFRA-123",
			URL:        "https://company.atlassian.net/browse/INFRA-123",
		},
		Domain: common.Domain{
			Name:      "infra",
			Subdomain: "bug",
		},
		Severity:   common.SeverityHigh,
		Summary:    "DB pool exhaustion",
		ObservedAt: now,
		ReceivedAt: now,
		Metadata: map[string]any{
			"jira_issue_type": "Bug",
			"jira_project":    "INFRA",
			"jira_status":     "Open",
			"jira_priority":   "High",
			"jira_reporter":   "Jane Doe",
		},
	}

	fp1, err := signal.ComputeFingerprint(sig)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	sig.Status = signal.StatusMapped
	sig.ReceivedAt = now.Add(10 * time.Minute)
	sig.RootCauseID = "rc-001"
	frustration := 99.9
	sig.Derived = &signal.DerivedMetrics{Frustration: &frustration}
	sig.Embedding = []float32{0.1, 0.2, 0.3}

	fp2, err := signal.ComputeFingerprint(sig)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if fp1 != fp2 {
		t.Errorf("mutable fields changed fingerprint: %s != %s", fp1, fp2)
	}
}

func TestPagerDutyFingerprintStability(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)

	sig := signal.Signal{
		ID:   "pd-P1234ABC",
		Type: signal.TypeAlert,
		Source: common.SourceSystem{
			Type:       "alerting",
			Name:       "pagerduty",
			ExternalID: "P1234ABC",
			URL:        "https://company.pagerduty.com/incidents/P1234ABC",
		},
		Domain: common.Domain{
			Name:      "operations",
			Subdomain: "api-gateway",
		},
		Severity:   common.SeverityHigh,
		Summary:    "API Gateway p99 latency > 2s",
		ObservedAt: now,
		ReceivedAt: now,
		Metadata: map[string]any{
			"pagerduty_incident_number": 42,
			"pagerduty_urgency":         "high",
			"pagerduty_status":          "triggered",
		},
	}

	fp1, err := signal.ComputeFingerprint(sig)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	sig.Status = signal.StatusArchived
	sig.ReceivedAt = now.Add(1 * time.Hour)

	fp2, err := signal.ComputeFingerprint(sig)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if fp1 != fp2 {
		t.Errorf("mutable fields changed fingerprint: %s != %s", fp1, fp2)
	}
}

func TestDifferentSignalsDifferentFingerprints(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)

	jira := signal.Signal{
		ID:         "jira-INFRA-123",
		Type:       signal.TypeSupportTicket,
		Source:     common.SourceSystem{Type: "ticketing", Name: "jira", ExternalID: "INFRA-123"},
		Domain:     common.Domain{Name: "infra"},
		Severity:   common.SeverityHigh,
		Summary:    "DB pool exhaustion",
		ObservedAt: now,
	}

	pd := signal.Signal{
		ID:         "pd-P1234ABC",
		Type:       signal.TypeAlert,
		Source:     common.SourceSystem{Type: "alerting", Name: "pagerduty", ExternalID: "P1234ABC"},
		Domain:     common.Domain{Name: "operations"},
		Severity:   common.SeverityHigh,
		Summary:    "API Gateway p99 latency > 2s",
		ObservedAt: now,
	}

	fpJira, err := signal.ComputeFingerprint(jira)
	if err != nil {
		t.Fatalf("jira: %v", err)
	}
	fpPD, err := signal.ComputeFingerprint(pd)
	if err != nil {
		t.Fatalf("pagerduty: %v", err)
	}

	if fpJira == fpPD {
		t.Error("different signals produced same fingerprint")
	}
}
