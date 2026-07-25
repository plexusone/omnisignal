package omnisignal_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
	"github.com/plexusone/signal-spec/schema"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func compileSignalSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	var raw any
	if err := json.Unmarshal(schema.SignalSchema, &raw); err != nil {
		t.Fatalf("unmarshal embedded schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("signal.schema.json", raw); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	sch, err := c.Compile("signal.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func validateSignal(t *testing.T, sch *jsonschema.Schema, sig signal.Signal) {
	t.Helper()
	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal signal: %v", err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal for validation: %v", err)
	}
	if err := sch.Validate(v); err != nil {
		t.Errorf("signal %s failed schema validation: %v", sig.ID, err)
	}
}

func TestJiraSignalConformance(t *testing.T) {
	sch := compileSignalSchema(t)
	now := time.Now().UTC().Truncate(time.Second)

	signals := []signal.Signal{
		{
			ID:     "jira-INFRA-123",
			Type:   signal.TypeSupportTicket,
			Status: signal.StatusNew,
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
			Severity:    common.SeverityHigh,
			Summary:     "Database connection pool exhaustion",
			Description: "Production DB pool hits max connections during peak hours",
			Entities: []common.Entity{
				{
					Type: "component",
					Name: "PostgreSQL",
					Attributes: map[string]string{
						"jira_id": "10001",
					},
				},
			},
			ObservedAt: now.Add(-24 * time.Hour),
			ReceivedAt: now,
			Tags:       []common.Tag{"database", "production"},
			Metadata: map[string]any{
				"jira_issue_type": "Bug",
				"jira_project":    "INFRA",
				"jira_status":     "Open",
				"jira_priority":   "High",
				"jira_reporter":   "Jane Doe",
			},
		},
		{
			ID:     "jira-SUPPORT-456",
			Type:   signal.TypeSupportTicket,
			Status: signal.StatusProcessing,
			Source: common.SourceSystem{
				Type:       "ticketing",
				Name:       "jira",
				ExternalID: "SUPPORT-456",
				URL:        "https://company.atlassian.net/browse/SUPPORT-456",
			},
			Domain: common.Domain{
				Name:      "support",
				Subdomain: "story",
			},
			Severity:   common.SeverityMedium,
			Summary:    "OAuth token refresh failures for enterprise SSO",
			ObservedAt: now.Add(-48 * time.Hour),
			ReceivedAt: now,
			Metadata: map[string]any{
				"jira_issue_type": "Story",
				"jira_project":    "SUPPORT",
				"jira_status":     "In Progress",
				"jira_priority":   "Medium",
				"jira_reporter":   "John Smith",
			},
		},
		{
			ID:     "jira-INFRA-789",
			Type:   signal.TypeSupportTicket,
			Status: signal.StatusArchived,
			Source: common.SourceSystem{
				Type:       "ticketing",
				Name:       "jira",
				ExternalID: "INFRA-789",
			},
			Domain: common.Domain{
				Name: "infra",
			},
			Severity:   common.SeverityInfo,
			Summary:    "Minor log format inconsistency",
			ObservedAt: now.Add(-72 * time.Hour),
			ReceivedAt: now,
			Metadata: map[string]any{
				"jira_issue_type": "Task",
				"jira_project":    "INFRA",
				"jira_status":     "Done",
				"jira_priority":   "Lowest",
				"jira_reporter":   "Bot",
			},
		},
	}

	for _, sig := range signals {
		t.Run(sig.ID, func(t *testing.T) {
			validateSignal(t, sch, sig)
		})
	}
}

func TestPagerDutySignalConformance(t *testing.T) {
	sch := compileSignalSchema(t)
	now := time.Now().UTC().Truncate(time.Second)

	signals := []signal.Signal{
		{
			ID:     "pd-P1234ABC",
			Type:   signal.TypeAlert,
			Status: signal.StatusNew,
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
			Severity: common.SeverityHigh,
			Summary:  "API Gateway p99 latency > 2s",
			Entities: []common.Entity{
				{
					Type: "service",
					Name: "API Gateway",
					Attributes: map[string]string{
						"pagerduty_id": "PSERVICE01",
					},
				},
			},
			ObservedAt: now.Add(-1 * time.Hour),
			ReceivedAt: now,
			Metadata: map[string]any{
				"pagerduty_incident_number": 42,
				"pagerduty_urgency":         "high",
				"pagerduty_status":          "triggered",
			},
		},
		{
			ID:     "pd-P5678DEF",
			Type:   signal.TypeOutage,
			Status: signal.StatusArchived,
			Source: common.SourceSystem{
				Type:       "alerting",
				Name:       "pagerduty",
				ExternalID: "P5678DEF",
				URL:        "https://company.pagerduty.com/incidents/P5678DEF",
			},
			Domain: common.Domain{
				Name:      "operations",
				Subdomain: "payment-service",
			},
			Severity:    common.SeverityCritical,
			Summary:     "Payment service unavailable",
			Description: "All payment processing halted",
			Entities: []common.Entity{
				{
					Type: "service",
					Name: "Payment Service",
					Attributes: map[string]string{
						"pagerduty_id": "PSERVICE02",
					},
				},
			},
			ObservedAt: now.Add(-6 * time.Hour),
			ReceivedAt: now,
			Metadata: map[string]any{
				"pagerduty_incident_number": 43,
				"pagerduty_urgency":         "high",
				"pagerduty_status":          "resolved",
			},
		},
		{
			ID:     "pd-P9012GHI",
			Type:   signal.TypeAlert,
			Status: signal.StatusProcessing,
			Source: common.SourceSystem{
				Type:       "alerting",
				Name:       "pagerduty",
				ExternalID: "P9012GHI",
			},
			Domain: common.Domain{
				Name: "operations",
			},
			Severity:   common.SeverityLow,
			Summary:    "Disk usage approaching threshold",
			ObservedAt: now.Add(-30 * time.Minute),
			ReceivedAt: now,
			Metadata: map[string]any{
				"pagerduty_incident_number": 44,
				"pagerduty_urgency":         "low",
				"pagerduty_status":          "acknowledged",
			},
		},
	}

	for _, sig := range signals {
		t.Run(sig.ID, func(t *testing.T) {
			validateSignal(t, sch, sig)
		})
	}
}

func TestEnhancementRequestSignalConformance(t *testing.T) {
	sch := compileSignalSchema(t)
	now := time.Now().UTC().Truncate(time.Second)

	sig := signal.Signal{
		ID:     "aha-FEAT-100",
		Type:   signal.TypeEnhancementRequest,
		Status: signal.StatusNew,
		Source: common.SourceSystem{
			Type:       "product_management",
			Name:       "aha",
			ExternalID: "FEAT-100",
			URL:        "https://company.aha.io/features/FEAT-100",
		},
		Domain: common.Domain{
			Name:      "identity",
			Subdomain: "scim",
		},
		Severity: common.SeverityMedium,
		Summary:  "Add SCIM provisioning for enterprise SSO",
		Entities: []common.Entity{
			{
				Type: "capability",
				Name: "SCIM Provisioning",
				Ref:  "capability:scim-provisioning",
			},
		},
		ObservedAt: now.Add(-7 * 24 * time.Hour),
		ReceivedAt: now,
		Metadata: map[string]any{
			signal.MetaVotes:         42,
			signal.MetaSubscribers:   15,
			signal.MetaOrganizations: []string{"Acme Corp", "Globex"},
			signal.MetaCustomers:     []string{"acme-001", "globex-002"},
			signal.MetaEstimatedARR:  150000_00,
			signal.MetaCustomerRef:   "customer:acme-001",
			signal.MetaCapabilityRef: "capability:scim-provisioning",
		},
	}

	validateSignal(t, sch, sig)
}

func TestProductSignalTypesConformance(t *testing.T) {
	sch := compileSignalSchema(t)
	now := time.Now().UTC().Truncate(time.Second)

	productTypes := []struct {
		typ     signal.Type
		summary string
	}{
		{signal.TypeCompetitiveGap, "Competitor X launched SSO feature we lack"},
		{signal.TypeCompetitorLaunch, "Competitor Y released v3.0 with AI features"},
		{signal.TypeAnalystFinding, "Gartner MQ places us in Leaders quadrant"},
		{signal.TypeMarketObservation, "IAM market growing 15% CAGR per Forrester"},
	}

	for _, pt := range productTypes {
		t.Run(string(pt.typ), func(t *testing.T) {
			sig := signal.Signal{
				ID:     "test-" + string(pt.typ),
				Type:   pt.typ,
				Status: signal.StatusNew,
				Source: common.SourceSystem{
					Type: "market_intelligence",
					Name: "manual",
				},
				Domain:     common.Domain{Name: "market"},
				Severity:   common.SeverityMedium,
				Summary:    pt.summary,
				ObservedAt: now,
				ReceivedAt: now,
			}
			validateSignal(t, sch, sig)
		})
	}
}

func TestSignalWithDerivedMetricsConformance(t *testing.T) {
	sch := compileSignalSchema(t)
	now := time.Now().UTC().Truncate(time.Second)

	frustration := 85.5
	momentum := 12.0
	reach := 8.0
	urgency := 45.0

	sig := signal.Signal{
		ID:     "test-derived",
		Type:   signal.TypeSupportTicket,
		Status: signal.StatusMapped,
		Source: common.SourceSystem{
			Type: "ticketing",
			Name: "zendesk",
		},
		Domain:     common.Domain{Name: "authentication"},
		Severity:   common.SeverityHigh,
		Summary:    "Repeated OAuth failures",
		ObservedAt: now,
		ReceivedAt: now,
		Derived: &signal.DerivedMetrics{
			Frustration: &frustration,
			Momentum:    &momentum,
			Reach:       &reach,
			Urgency:     &urgency,
			ComputedAt:  &now,
			Extra: map[string]float64{
				"weighted_impact": 92.3,
			},
		},
	}

	validateSignal(t, sch, sig)
}
