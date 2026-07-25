package consolidate_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/rootcause"
	"github.com/plexusone/signal-spec/pkg/signal"
)

type mockLLMClient struct {
	completeFunc func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	calls        []llmCall
}

type llmCall struct {
	model        string
	systemPrompt string
	userPrompt   string
}

func (m *mockLLMClient) Complete(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	m.calls = append(m.calls, llmCall{model: model, systemPrompt: systemPrompt, userPrompt: userPrompt})
	if m.completeFunc != nil {
		return m.completeFunc(ctx, model, systemPrompt, userPrompt)
	}
	return "Title: Test Root Cause\nDescription: This is a test description.\nSymptom patterns:\n- Pattern 1\n- Pattern 2", nil
}

func TestNewLLMSummarizer(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{
		Model: "test-model",
	})
	if s == nil {
		t.Fatal("NewLLMSummarizer returned nil")
	}
}

func TestLLMSummarizerEmptyCluster(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{})

	_, err := s.Summarize(context.Background(), consolidate.Cluster{})
	if err != consolidate.ErrEmptyCluster {
		t.Errorf("expected ErrEmptyCluster, got %v", err)
	}
}

func TestLLMSummarizerBasic(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{
		Model: "gpt-4",
	})

	cluster := consolidate.Cluster{
		ID: "cluster-1",
		Signals: []signal.Signal{
			{ID: "s1", Summary: "Error in payment processing"},
			{ID: "s2", Summary: "Payment gateway timeout"},
		},
	}

	rc, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}

	if rc.ID != "rc-cluster-1" {
		t.Errorf("ID = %s, want rc-cluster-1", rc.ID)
	}
	if rc.Title != "Test Root Cause" {
		t.Errorf("Title = %s", rc.Title)
	}
	if rc.Status != rootcause.StatusNew {
		t.Errorf("Status = %s, want new", rc.Status)
	}
	if len(rc.SignalIDs) != 2 {
		t.Errorf("SignalIDs = %d, want 2", len(rc.SignalIDs))
	}
	if len(client.calls) != 1 {
		t.Errorf("LLM calls = %d, want 1", len(client.calls))
	}
}

func TestLLMSummarizerError(t *testing.T) {
	expectedErr := errors.New("LLM failed")
	client := &mockLLMClient{
		completeFunc: func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
			return "", expectedErr
		},
	}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{})

	cluster := consolidate.Cluster{
		ID:      "c1",
		Signals: []signal.Signal{{ID: "s1"}},
	}

	_, err := s.Summarize(context.Background(), cluster)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestLLMSummarizerPromptContent(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{
		Model:           "test",
		IncludeMetadata: true,
	})

	cluster := consolidate.Cluster{
		ID: "c1",
		Signals: []signal.Signal{
			{
				ID:          "s1",
				Type:        signal.TypeSupportTicket,
				Severity:    common.SeverityHigh,
				Summary:     "Test summary",
				Description: "Test description",
				Domain:      common.Domain{Name: "product", Subdomain: "auth"},
				Entities:    []common.Entity{{Type: "service", Name: "api"}},
				Metadata:    map[string]any{"key": "value"},
			},
		},
	}

	_, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	prompt := client.calls[0].userPrompt

	if !strings.Contains(prompt, "Cluster ID: c1") {
		t.Error("prompt should contain cluster ID")
	}
	if !strings.Contains(prompt, "Type: support_ticket") {
		t.Error("prompt should contain signal type")
	}
	if !strings.Contains(prompt, "Severity: high") {
		t.Error("prompt should contain severity")
	}
	if !strings.Contains(prompt, "Test summary") {
		t.Error("prompt should contain summary")
	}
	if !strings.Contains(prompt, "Test description") {
		t.Error("prompt should contain description")
	}
	if !strings.Contains(prompt, "product/auth") {
		t.Error("prompt should contain domain")
	}
	if !strings.Contains(prompt, "service:api") {
		t.Error("prompt should contain entities")
	}
	if !strings.Contains(prompt, "key=value") {
		t.Error("prompt should contain metadata when enabled")
	}
}

func TestLLMSummarizerMaxSignals(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{
		MaxSignals: 2,
	})

	signals := make([]signal.Signal, 5)
	for i := range signals {
		signals[i] = signal.Signal{ID: string(rune('a' + i))}
	}

	cluster := consolidate.Cluster{ID: "c1", Signals: signals}

	_, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	prompt := client.calls[0].userPrompt
	if !strings.Contains(prompt, "Showing first 2 of 5") {
		t.Error("prompt should indicate truncation")
	}
}

func TestLLMSummarizerInfersDomain(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{})

	cluster := consolidate.Cluster{
		ID: "c1",
		Signals: []signal.Signal{
			{ID: "s1", Domain: common.Domain{Name: "product"}},
			{ID: "s2", Domain: common.Domain{Name: "product"}},
			{ID: "s3", Domain: common.Domain{Name: "infra"}},
		},
	}

	rc, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	if rc.Domain.Name != "product" {
		t.Errorf("Domain = %s, want product (most common)", rc.Domain.Name)
	}
}

func TestLLMSummarizerInfersSeverity(t *testing.T) {
	client := &mockLLMClient{}
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{})

	cluster := consolidate.Cluster{
		ID: "c1",
		Signals: []signal.Signal{
			{ID: "s1", Severity: common.SeverityLow},
			{ID: "s2", Severity: common.SeverityCritical},
			{ID: "s3", Severity: common.SeverityMedium},
		},
	}

	rc, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	if rc.Severity != common.SeverityCritical {
		t.Errorf("Severity = %s, want critical (highest)", rc.Severity)
	}
}

func TestLLMSummarizerCustomPrompt(t *testing.T) {
	client := &mockLLMClient{}
	customPrompt := "Custom summarization prompt"
	s := consolidate.NewLLMSummarizer(client, consolidate.SummarizerConfig{
		SystemPrompt: customPrompt,
	})

	cluster := consolidate.Cluster{
		ID:      "c1",
		Signals: []signal.Signal{{ID: "s1"}},
	}

	_, err := s.Summarize(context.Background(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	if client.calls[0].systemPrompt != customPrompt {
		t.Errorf("system prompt = %q, want custom", client.calls[0].systemPrompt)
	}
}

func TestLLMSummarizerImplementsInterface(t *testing.T) {
	var _ consolidate.Summarizer = (*consolidate.LLMSummarizer)(nil)
}

func TestMemoryEvidenceStore(t *testing.T) {
	store := consolidate.NewMemoryEvidenceStore()
	ctx := context.Background()

	link := consolidate.EvidenceLink{
		SignalID:    "s1",
		RootCauseID: "rc1",
		Similarity:  0.95,
		LinkedAt:    time.Now(),
	}

	err := store.Link(ctx, link)
	if err != nil {
		t.Fatalf("Link error: %v", err)
	}

	links, err := store.GetLinks(ctx, "rc1")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Fatalf("GetLinks returned %d, want 1", len(links))
	}
	if links[0].SignalID != "s1" {
		t.Error("wrong signal ID")
	}

	found, err := store.GetSignalLink(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("GetSignalLink returned nil")
	}
	if found.RootCauseID != "rc1" {
		t.Error("wrong root cause ID")
	}
}

func TestMemoryEvidenceStoreNotFound(t *testing.T) {
	store := consolidate.NewMemoryEvidenceStore()
	ctx := context.Background()

	found, err := store.GetSignalLink(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Error("expected nil for nonexistent signal")
	}

	links, err := store.GetLinks(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 {
		t.Error("expected empty list for nonexistent root cause")
	}
}

func TestEvidenceStoreInterface(t *testing.T) {
	var _ consolidate.EvidenceStore = (*consolidate.MemoryEvidenceStore)(nil)
}

func TestEvidenceLinkFields(t *testing.T) {
	now := time.Now()
	link := consolidate.EvidenceLink{
		SignalID:    "sig-123",
		RootCauseID: "rc-456",
		Similarity:  0.92,
		LinkedAt:    now,
	}

	if link.SignalID != "sig-123" {
		t.Error("SignalID mismatch")
	}
	if link.RootCauseID != "rc-456" {
		t.Error("RootCauseID mismatch")
	}
	if link.Similarity != 0.92 {
		t.Error("Similarity mismatch")
	}
	if link.LinkedAt != now {
		t.Error("LinkedAt mismatch")
	}
}

func TestDefaultSummarizerPrompt(t *testing.T) {
	prompt := consolidate.DefaultSummarizerPrompt
	if !strings.Contains(prompt, "root cause") {
		t.Error("prompt should mention root cause")
	}
	if !strings.Contains(prompt, "Title") {
		t.Error("prompt should mention Title format")
	}
}
