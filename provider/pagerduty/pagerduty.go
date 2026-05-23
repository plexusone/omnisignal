// Package pagerduty provides a PagerDuty signal provider for omnisignal.
//
// This is a thick provider that uses the official PagerDuty Go SDK.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/plexusone/omnisignal/provider/pagerduty"
//	)
//
//	provider, err := omnisignal.New("pagerduty", omnisignal.Config{
//	    APIKey: os.Getenv("PAGERDUTY_API_KEY"),
//	})
package pagerduty

import (
	"context"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

const (
	// ProviderName is the identifier for this provider.
	ProviderName = "pagerduty"

	// DefaultTimeout for API requests.
	DefaultTimeout = 30 * time.Second
)

func init() {
	omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThick)
}

// Provider implements omnisignal.Provider for PagerDuty.
type Provider struct {
	client *pagerduty.Client
	config omnisignal.Config
}

// NewProvider creates a new PagerDuty provider.
func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("%w: APIKey is required", omnisignal.ErrInvalidConfig)
	}

	client := pagerduty.NewClient(cfg.APIKey)

	return &Provider{
		client: client,
		config: cfg,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// Fetch retrieves incidents from PagerDuty.
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	var signals []signal.Signal

	listOpts := pagerduty.ListIncidentsOptions{
		Since: opts.Since.Format(time.RFC3339),
	}

	if !opts.Until.IsZero() {
		listOpts.Until = opts.Until.Format(time.RFC3339)
	}

	// Map severity filters to PagerDuty urgencies
	if len(opts.Severities) > 0 {
		listOpts.Urgencies = mapSeveritiesToUrgencies(opts.Severities)
	}

	// Map status filters
	if len(opts.Statuses) > 0 {
		listOpts.Statuses = opts.Statuses
	}

	// Pagination
	listOpts.Limit = 100
	if opts.Limit > 0 && opts.Limit < 100 {
		listOpts.Limit = uint(opts.Limit)
	}

	offset := uint(0)
	for {
		listOpts.Offset = offset

		resp, err := p.client.ListIncidentsWithContext(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("fetching incidents: %w", err)
		}

		for _, incident := range resp.Incidents {
			sig := p.normalizeIncident(incident)
			signals = append(signals, sig)

			if opts.Limit > 0 && len(signals) >= opts.Limit {
				return signals, nil
			}
		}

		if !resp.More {
			break
		}
		offset += uint(len(resp.Incidents))
	}

	return signals, nil
}

// Subscribe is not yet implemented for PagerDuty.
// PagerDuty supports webhooks, but that requires a webhook receiver.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Capabilities returns what this provider supports.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:   false, // Requires webhook setup
		SupportsBatchFetch:  true,
		SupportsFiltering:   true,
		SupportsAcknowledge: true,
		MaxBatchSize:        100,
		RateLimitPerMinute:  900, // PagerDuty rate limit
		SignalTypes: []signal.Type{
			signal.TypeAlert,
			signal.TypeOutage,
		},
	}
}

// Close releases resources.
func (p *Provider) Close() error {
	return nil
}

// normalizeIncident converts a PagerDuty incident to a signal-spec Signal.
func (p *Provider) normalizeIncident(incident pagerduty.Incident) signal.Signal {
	// Parse timestamps
	observedAt, _ := time.Parse(time.RFC3339, incident.CreatedAt)
	if observedAt.IsZero() {
		observedAt = time.Now()
	}

	// Map urgency to severity
	severity := mapUrgencyToSeverity(incident.Urgency)

	// Map status
	status := mapIncidentStatus(incident.Status)

	// Extract service as entity
	var entities []common.Entity
	if incident.Service.ID != "" {
		entities = append(entities, common.Entity{
			Type: "service",
			Name: incident.Service.Summary,
			Attributes: map[string]string{
				"pagerduty_id": incident.Service.ID,
			},
		})
	}

	// Build domain from service
	domain := common.Domain{
		Name: "operations", // Default domain
	}
	if incident.Service.Summary != "" {
		domain.Subdomain = normalizeServiceName(incident.Service.Summary)
	}

	// Determine signal type
	signalType := signal.TypeAlert
	if incident.Status == "resolved" || incident.Urgency == "high" {
		signalType = signal.TypeOutage
	}

	return signal.Signal{
		ID:     fmt.Sprintf("pd-%s", incident.ID),
		Type:   signalType,
		Status: status,
		Source: common.SourceSystem{
			Type:       "alerting",
			Name:       "pagerduty",
			ExternalID: incident.ID,
			URL:        incident.HTMLURL,
		},
		Domain:      domain,
		Severity:    severity,
		Summary:     incident.Title,
		Description: incident.Description,
		Entities:    entities,
		ObservedAt:  observedAt,
		ReceivedAt:  time.Now(),
		Metadata: map[string]any{
			"pagerduty_incident_number": incident.IncidentNumber,
			"pagerduty_urgency":         incident.Urgency,
			"pagerduty_status":          incident.Status,
			"pagerduty_priority":        incident.Priority,
		},
	}
}

// mapUrgencyToSeverity converts PagerDuty urgency to signal-spec severity.
func mapUrgencyToSeverity(urgency string) common.Severity {
	switch urgency {
	case "high":
		return common.SeverityHigh
	case "low":
		return common.SeverityLow
	default:
		return common.SeverityMedium
	}
}

// mapIncidentStatus converts PagerDuty status to signal-spec status.
func mapIncidentStatus(pdStatus string) signal.Status {
	switch pdStatus {
	case "triggered":
		return signal.StatusNew
	case "acknowledged":
		return signal.StatusProcessing
	case "resolved":
		return signal.StatusArchived
	default:
		return signal.StatusNew
	}
}

// mapSeveritiesToUrgencies converts signal-spec severities to PagerDuty urgencies.
func mapSeveritiesToUrgencies(severities []string) []string {
	urgencies := make([]string, 0, len(severities))
	for _, s := range severities {
		switch s {
		case "critical", "high":
			urgencies = append(urgencies, "high")
		case "medium", "low", "info":
			urgencies = append(urgencies, "low")
		}
	}
	return urgencies
}

// normalizeServiceName converts a service name to a valid subdomain.
func normalizeServiceName(name string) string {
	// Simple normalization - in production, use proper kebab-case conversion
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32) // lowercase
		} else if c == ' ' || c == '_' {
			result += "-"
		}
	}
	return result
}
