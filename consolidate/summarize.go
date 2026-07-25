package consolidate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/rootcause"
	"github.com/plexusone/signal-spec/pkg/signal"
)

// SummarizerConfig configures the LLM-based summarizer.
type SummarizerConfig struct {
	// Model is the LLM model to use.
	Model string

	// SystemPrompt customizes the summarization behavior.
	// If empty, uses DefaultSummarizerPrompt.
	SystemPrompt string

	// MaxSignals limits how many signals to include in context.
	// Default is 20.
	MaxSignals int

	// IncludeMetadata includes signal metadata in the prompt.
	IncludeMetadata bool
}

// DefaultSummarizerPrompt is the default system prompt for summarization.
const DefaultSummarizerPrompt = `You are a root cause analyst. Given a cluster of related signals (support tickets, incidents, feedback), generate a concise root cause summary.

Output format:
- Title: A brief, actionable title (max 100 chars)
- Description: 2-3 sentences explaining the underlying issue
- Symptom patterns: 3-5 common manifestations

Focus on the underlying cause, not individual symptoms.`

// LLMClient is the interface for LLM completion requests.
type LLMClient interface {
	// Complete generates a completion for the given prompt.
	Complete(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

// LLMSummarizer implements Summarizer using an LLM.
type LLMSummarizer struct {
	client LLMClient
	config SummarizerConfig
}

// NewLLMSummarizer creates a summarizer backed by an LLM.
func NewLLMSummarizer(client LLMClient, cfg SummarizerConfig) *LLMSummarizer {
	if cfg.MaxSignals <= 0 {
		cfg.MaxSignals = 20
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = DefaultSummarizerPrompt
	}
	return &LLMSummarizer{
		client: client,
		config: cfg,
	}
}

// Summarize generates a root cause from a signal cluster.
func (s *LLMSummarizer) Summarize(ctx context.Context, cluster Cluster) (rootcause.RootCause, error) {
	if len(cluster.Signals) == 0 {
		return rootcause.RootCause{}, ErrEmptyCluster
	}

	prompt := s.buildPrompt(cluster)

	response, err := s.client.Complete(ctx, s.config.Model, s.config.SystemPrompt, prompt)
	if err != nil {
		return rootcause.RootCause{}, fmt.Errorf("LLM completion: %w", err)
	}

	rc := s.parseResponse(response, cluster)
	return rc, nil
}

// buildPrompt creates the user prompt from the cluster.
func (s *LLMSummarizer) buildPrompt(cluster Cluster) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Cluster ID: %s\n", cluster.ID))
	b.WriteString(fmt.Sprintf("Signal count: %d\n\n", len(cluster.Signals)))

	signals := cluster.Signals
	if len(signals) > s.config.MaxSignals {
		signals = signals[:s.config.MaxSignals]
		b.WriteString(fmt.Sprintf("(Showing first %d of %d signals)\n\n", s.config.MaxSignals, len(cluster.Signals)))
	}

	b.WriteString("Signals:\n")
	for i, sig := range signals {
		b.WriteString(fmt.Sprintf("\n--- Signal %d ---\n", i+1))
		b.WriteString(fmt.Sprintf("Type: %s\n", sig.Type))
		b.WriteString(fmt.Sprintf("Severity: %s\n", sig.Severity))
		b.WriteString(fmt.Sprintf("Summary: %s\n", sig.Summary))
		if sig.Description != "" {
			b.WriteString(fmt.Sprintf("Description: %s\n", sig.Description))
		}
		if sig.Domain.Name != "" {
			b.WriteString(fmt.Sprintf("Domain: %s", sig.Domain.Name))
			if sig.Domain.Subdomain != "" {
				b.WriteString(fmt.Sprintf("/%s", sig.Domain.Subdomain))
			}
			b.WriteString("\n")
		}
		if len(sig.Entities) > 0 {
			b.WriteString("Entities: ")
			for j, e := range sig.Entities {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(fmt.Sprintf("%s:%s", e.Type, e.Name))
			}
			b.WriteString("\n")
		}
		if s.config.IncludeMetadata && sig.Metadata != nil {
			b.WriteString("Metadata: ")
			for k, v := range sig.Metadata {
				b.WriteString(fmt.Sprintf("%s=%v ", k, v))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// parseResponse extracts a root cause from the LLM response.
func (s *LLMSummarizer) parseResponse(response string, cluster Cluster) rootcause.RootCause {
	now := time.Now()

	// Extract title and description from response
	title, description, patterns := extractSections(response)

	// Determine domain from signals
	domain := inferDomain(cluster.Signals)

	// Determine severity from signals
	severity := inferSeverity(cluster.Signals)

	// Collect signal IDs
	signalIDs := make([]string, len(cluster.Signals))
	for i, sig := range cluster.Signals {
		signalIDs[i] = sig.ID
	}

	return rootcause.RootCause{
		ID:              "rc-" + cluster.ID,
		Title:           title,
		Description:     description,
		Status:          rootcause.StatusNew,
		Domain:          domain,
		Severity:        severity,
		SymptomPatterns: patterns,
		SignalIDs:       signalIDs,
		FirstSeen:       now,
		LastSeen:        now,
		Embedding:       cluster.Centroid,
	}
}

// extractSections parses the LLM response into title, description, and patterns.
func extractSections(response string) (title, description string, patterns []string) {
	lines := strings.Split(response, "\n")

	var currentSection string
	var descLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "title:") {
			title = strings.TrimSpace(line[6:])
			currentSection = "title"
		} else if strings.HasPrefix(lower, "description:") {
			desc := strings.TrimSpace(line[12:])
			if desc != "" {
				descLines = append(descLines, desc)
			}
			currentSection = "description"
		} else if strings.HasPrefix(lower, "symptom") || strings.HasPrefix(lower, "pattern") {
			currentSection = "patterns"
		} else if currentSection == "description" && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
			descLines = append(descLines, line)
		} else if currentSection == "patterns" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			pattern := strings.TrimLeft(line, "-* ")
			if pattern != "" && !strings.HasPrefix(strings.ToLower(pattern), "symptom") {
				patterns = append(patterns, pattern)
			}
		}
	}

	description = strings.Join(descLines, " ")

	// Fallback if parsing failed
	if title == "" && len(lines) > 0 {
		title = lines[0]
		if len(title) > 100 {
			title = title[:97] + "..."
		}
	}
	if description == "" {
		description = response
		if len(description) > 500 {
			description = description[:497] + "..."
		}
	}

	return title, description, patterns
}

// inferDomain determines the most common domain from signals.
func inferDomain(signals []signal.Signal) common.Domain {
	if len(signals) == 0 {
		return common.Domain{}
	}

	domainCounts := make(map[string]int)
	for _, sig := range signals {
		if sig.Domain.Name != "" {
			key := sig.Domain.Name
			if sig.Domain.Subdomain != "" {
				key += "/" + sig.Domain.Subdomain
			}
			domainCounts[key]++
		}
	}

	var bestDomain string
	var bestCount int
	for d, count := range domainCounts {
		if count > bestCount {
			bestDomain = d
			bestCount = count
		}
	}

	if bestDomain == "" {
		return common.Domain{}
	}

	parts := strings.SplitN(bestDomain, "/", 2)
	domain := common.Domain{Name: parts[0]}
	if len(parts) > 1 {
		domain.Subdomain = parts[1]
	}
	return domain
}

// inferSeverity determines severity from the highest signal severity.
func inferSeverity(signals []signal.Signal) common.Severity {
	if len(signals) == 0 {
		return common.SeverityMedium
	}

	severityRank := map[common.Severity]int{
		common.SeverityCritical: 5,
		common.SeverityHigh:     4,
		common.SeverityMedium:   3,
		common.SeverityLow:      2,
		common.SeverityInfo:     1,
	}

	var maxSev common.Severity
	var maxRank int

	for _, sig := range signals {
		rank := severityRank[sig.Severity]
		if rank > maxRank {
			maxRank = rank
			maxSev = sig.Severity
		}
	}

	if maxRank == 0 {
		return common.SeverityMedium
	}
	return maxSev
}

// EvidenceLink represents the association between a signal and a root cause.
type EvidenceLink struct {
	SignalID    string
	RootCauseID string
	Similarity  float64
	LinkedAt    time.Time
}

// EvidenceStore persists signal-to-root-cause evidence links.
type EvidenceStore interface {
	// Link creates an evidence link.
	Link(ctx context.Context, link EvidenceLink) error

	// GetLinks returns all links for a root cause.
	GetLinks(ctx context.Context, rootCauseID string) ([]EvidenceLink, error)

	// GetSignalLink returns the link for a specific signal.
	GetSignalLink(ctx context.Context, signalID string) (*EvidenceLink, error)
}

// MemoryEvidenceStore is an in-memory implementation of EvidenceStore.
type MemoryEvidenceStore struct {
	links []EvidenceLink
}

// NewMemoryEvidenceStore creates an in-memory evidence store.
func NewMemoryEvidenceStore() *MemoryEvidenceStore {
	return &MemoryEvidenceStore{}
}

// Link creates an evidence link.
func (s *MemoryEvidenceStore) Link(ctx context.Context, link EvidenceLink) error {
	s.links = append(s.links, link)
	return nil
}

// GetLinks returns all links for a root cause.
func (s *MemoryEvidenceStore) GetLinks(ctx context.Context, rootCauseID string) ([]EvidenceLink, error) {
	var result []EvidenceLink
	for _, link := range s.links {
		if link.RootCauseID == rootCauseID {
			result = append(result, link)
		}
	}
	return result, nil
}

// GetSignalLink returns the link for a specific signal.
func (s *MemoryEvidenceStore) GetSignalLink(ctx context.Context, signalID string) (*EvidenceLink, error) {
	for _, link := range s.links {
		if link.SignalID == signalID {
			return &link, nil
		}
	}
	return nil, nil
}
