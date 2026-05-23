// Package jira provides a Jira signal provider for omnisignal.
//
// This is a thick provider that uses the go-jira community SDK.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/plexusone/omnisignal/provider/jira"
//	)
//
//	provider, err := omnisignal.New("jira", omnisignal.Config{
//	    BaseURL:   "https://company.atlassian.net",
//	    APIKey:    os.Getenv("JIRA_USER"),    // Username/email
//	    APISecret: os.Getenv("JIRA_TOKEN"),   // API token
//	    Options: map[string]any{
//	        "projects": []string{"INFRA", "SUPPORT"},
//	    },
//	})
package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

const (
	// ProviderName is the identifier for this provider.
	ProviderName = "jira"

	// DefaultTimeout for API requests.
	DefaultTimeout = 30 * time.Second
)

func init() {
	omnisignal.Register(ProviderName, NewProvider, omnisignal.PriorityThick)
}

// Provider implements omnisignal.Provider for Jira.
type Provider struct {
	client   *jira.Client
	config   omnisignal.Config
	projects []string
}

// NewProvider creates a new Jira provider.
func NewProvider(cfg omnisignal.Config) (omnisignal.Provider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("%w: BaseURL is required", omnisignal.ErrInvalidConfig)
	}
	if cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, fmt.Errorf("%w: APIKey (user) and APISecret (token) are required", omnisignal.ErrInvalidConfig)
	}

	tp := jira.BasicAuthTransport{
		Username: cfg.APIKey,
		Password: cfg.APISecret,
	}

	client, err := jira.NewClient(tp.Client(), cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating Jira client: %w", err)
	}

	// Get projects from options
	var projects []string
	if p, ok := cfg.Options["projects"].([]string); ok {
		projects = p
	}
	if p, ok := cfg.Options["projects"].([]any); ok {
		for _, v := range p {
			if s, ok := v.(string); ok {
				projects = append(projects, s)
			}
		}
	}

	return &Provider{
		client:   client,
		config:   cfg,
		projects: projects,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderName
}

// Fetch retrieves issues from Jira.
func (p *Provider) Fetch(ctx context.Context, opts omnisignal.FetchOptions) ([]signal.Signal, error) {
	var signals []signal.Signal

	// Build JQL query
	jql := p.buildJQL(opts)

	searchOpts := &jira.SearchOptions{
		StartAt:    0,
		MaxResults: 100,
		Expand:     "changelog",
	}

	if opts.Limit > 0 && opts.Limit < 100 {
		searchOpts.MaxResults = opts.Limit
	}

	for {
		issues, resp, err := p.client.Issue.Search(jql, searchOpts)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}

		for _, issue := range issues {
			sig := p.normalizeIssue(issue)
			signals = append(signals, sig)

			if opts.Limit > 0 && len(signals) >= opts.Limit {
				return signals, nil
			}
		}

		if resp.StartAt+len(issues) >= resp.Total {
			break
		}
		searchOpts.StartAt = resp.StartAt + len(issues)
	}

	return signals, nil
}

// Subscribe is not supported for Jira.
// Jira supports webhooks, but that requires webhook configuration.
func (p *Provider) Subscribe(ctx context.Context, opts omnisignal.SubscribeOptions) (<-chan signal.Signal, error) {
	return nil, omnisignal.ErrNotSupported
}

// Capabilities returns what this provider supports.
func (p *Provider) Capabilities() omnisignal.Capabilities {
	return omnisignal.Capabilities{
		SupportsStreaming:   false,
		SupportsBatchFetch:  true,
		SupportsFiltering:   true,
		SupportsAcknowledge: false,
		MaxBatchSize:        100,
		RateLimitPerMinute:  0, // Varies by instance
		SignalTypes: []signal.Type{
			signal.TypeSupportTicket,
			signal.TypeFeedback,
		},
	}
}

// Close releases resources.
func (p *Provider) Close() error {
	return nil
}

// buildJQL constructs a JQL query from fetch options.
func (p *Provider) buildJQL(opts omnisignal.FetchOptions) string {
	var conditions []string

	// Project filter
	if len(p.projects) > 0 {
		projectList := strings.Join(p.projects, ", ")
		conditions = append(conditions, fmt.Sprintf("project IN (%s)", projectList))
	}

	// Time filter
	if !opts.Since.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created >= '%s'", opts.Since.Format("2006-01-02")))
	}
	if !opts.Until.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created < '%s'", opts.Until.Format("2006-01-02")))
	}

	// Status filter
	if len(opts.Statuses) > 0 {
		statusList := "\"" + strings.Join(opts.Statuses, "\", \"") + "\""
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", statusList))
	}

	// Priority/severity filter
	if len(opts.Severities) > 0 {
		priorities := mapSeveritiesToPriorities(opts.Severities)
		if len(priorities) > 0 {
			priorityList := "\"" + strings.Join(priorities, "\", \"") + "\""
			conditions = append(conditions, fmt.Sprintf("priority IN (%s)", priorityList))
		}
	}

	// Custom filters from options
	if jqlFilter, ok := opts.Filters["jql"]; ok && jqlFilter != "" {
		conditions = append(conditions, jqlFilter)
	}

	if len(conditions) == 0 {
		return "ORDER BY created DESC"
	}

	return strings.Join(conditions, " AND ") + " ORDER BY created DESC"
}

// normalizeIssue converts a Jira issue to a signal-spec Signal.
func (p *Provider) normalizeIssue(issue jira.Issue) signal.Signal {
	// Parse timestamps
	observedAt := time.Time(issue.Fields.Created)
	if observedAt.IsZero() {
		observedAt = time.Now()
	}

	// Map priority to severity
	severity := mapPriorityToSeverity(issue.Fields.Priority)

	// Map status
	status := mapIssueStatus(issue.Fields.Status)

	// Build domain from project
	domain := common.Domain{
		Name: strings.ToLower(issue.Fields.Project.Key),
	}
	if issue.Fields.Type.Name != "" {
		domain.Subdomain = normalizeTypeName(issue.Fields.Type.Name)
	}

	// Extract components as entities
	var entities []common.Entity
	for _, comp := range issue.Fields.Components {
		entities = append(entities, common.Entity{
			Type: "component",
			Name: comp.Name,
			Attributes: map[string]string{
				"jira_id": comp.ID,
			},
		})
	}

	// Build description
	description := ""
	if issue.Fields.Description != "" {
		description = issue.Fields.Description
	}

	// Extract labels as tags
	var tags []common.Tag
	for _, label := range issue.Fields.Labels {
		if isValidTag(label) {
			tags = append(tags, common.Tag(label))
		}
	}

	return signal.Signal{
		ID:     fmt.Sprintf("jira-%s", issue.Key),
		Type:   signal.TypeSupportTicket,
		Status: status,
		Source: common.SourceSystem{
			Type:       "ticketing",
			Name:       "jira",
			ExternalID: issue.Key,
			URL:        fmt.Sprintf("%s/browse/%s", p.config.BaseURL, issue.Key),
		},
		Domain:      domain,
		Severity:    severity,
		Summary:     issue.Fields.Summary,
		Description: description,
		Entities:    entities,
		ObservedAt:  observedAt,
		ReceivedAt:  time.Now(),
		Tags:        tags,
		Metadata: map[string]any{
			"jira_issue_type": issue.Fields.Type.Name,
			"jira_project":    issue.Fields.Project.Key,
			"jira_status":     issue.Fields.Status.Name,
			"jira_priority":   issue.Fields.Priority.Name,
			"jira_reporter":   issue.Fields.Reporter.DisplayName,
		},
	}
}

// mapPriorityToSeverity converts Jira priority to signal-spec severity.
func mapPriorityToSeverity(priority *jira.Priority) common.Severity {
	if priority == nil {
		return common.SeverityMedium
	}

	name := strings.ToLower(priority.Name)
	switch {
	case strings.Contains(name, "highest") || strings.Contains(name, "blocker"):
		return common.SeverityCritical
	case strings.Contains(name, "high"):
		return common.SeverityHigh
	case strings.Contains(name, "medium") || strings.Contains(name, "normal"):
		return common.SeverityMedium
	case strings.Contains(name, "low"):
		return common.SeverityLow
	case strings.Contains(name, "lowest") || strings.Contains(name, "trivial"):
		return common.SeverityInfo
	default:
		return common.SeverityMedium
	}
}

// mapIssueStatus converts Jira status to signal-spec status.
func mapIssueStatus(status *jira.Status) signal.Status {
	if status == nil {
		return signal.StatusNew
	}

	name := strings.ToLower(status.Name)
	switch {
	case strings.Contains(name, "done") || strings.Contains(name, "closed") || strings.Contains(name, "resolved"):
		return signal.StatusArchived
	case strings.Contains(name, "progress") || strings.Contains(name, "review"):
		return signal.StatusProcessing
	default:
		return signal.StatusNew
	}
}

// mapSeveritiesToPriorities converts signal-spec severities to Jira priorities.
func mapSeveritiesToPriorities(severities []string) []string {
	var priorities []string
	for _, s := range severities {
		switch s {
		case "critical":
			priorities = append(priorities, "Highest", "Blocker")
		case "high":
			priorities = append(priorities, "High")
		case "medium":
			priorities = append(priorities, "Medium", "Normal")
		case "low":
			priorities = append(priorities, "Low")
		case "info":
			priorities = append(priorities, "Lowest", "Trivial")
		}
	}
	return priorities
}

// normalizeTypeName converts an issue type to a valid subdomain.
func normalizeTypeName(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else if c == ' ' || c == '_' {
			result += "-"
		}
	}
	return result
}

// isValidTag checks if a string is a valid kebab-case tag.
func isValidTag(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if c < 'a' || c > 'z' {
				return false
			}
		} else {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}
