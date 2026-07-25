package consolidate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
)

type mockOmniLLMClient struct {
	embedFunc func(ctx context.Context, model string, texts []string) ([][]float32, error)
	calls     []embedCall
}

type embedCall struct {
	model string
	texts []string
}

func (m *mockOmniLLMClient) Embed(ctx context.Context, model string, texts []string) ([][]float32, error) {
	m.calls = append(m.calls, embedCall{model: model, texts: texts})
	if m.embedFunc != nil {
		return m.embedFunc(ctx, model, texts)
	}
	// Default: return unit vectors
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1.0, 0.0, 0.0}
	}
	return result, nil
}

func TestNewOmniLLMEmbedder(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		Model: "text-embedding-3-small",
	})
	if embedder == nil {
		t.Fatal("NewOmniLLMEmbedder returned nil")
	}
}

func TestOmniLLMEmbedderEmptySignals(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{})

	result, err := embedder.Embed(context.Background(), nil)
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
	if len(client.calls) != 0 {
		t.Error("should not call client for empty signals")
	}
}

func TestOmniLLMEmbedderBasic(t *testing.T) {
	client := &mockOmniLLMClient{
		embedFunc: func(ctx context.Context, model string, texts []string) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{float32(i), 0.5, 0.5}
			}
			return result, nil
		},
	}

	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		Model: "test-model",
	})

	signals := []signal.Signal{
		{ID: "1", Summary: "First signal"},
		{ID: "2", Summary: "Second signal"},
	}

	result, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if len(result[0].Embedding) != 3 {
		t.Errorf("Embedding length = %d, want 3", len(result[0].Embedding))
	}
	if result[0].Embedding[0] != 0.0 {
		t.Errorf("First embedding[0] = %f, want 0", result[0].Embedding[0])
	}
	if result[1].Embedding[0] != 1.0 {
		t.Errorf("Second embedding[0] = %f, want 1", result[1].Embedding[0])
	}

	if len(client.calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(client.calls))
	}
	if client.calls[0].model != "test-model" {
		t.Errorf("model = %s, want test-model", client.calls[0].model)
	}
}

func TestOmniLLMEmbedderBatching(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		BatchSize: 2,
	})

	signals := make([]signal.Signal, 5)
	for i := range signals {
		signals[i] = signal.Signal{ID: string(rune('a' + i)), Summary: "Signal"}
	}

	_, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}

	// 5 signals with batch size 2 = 3 batches (2 + 2 + 1)
	if len(client.calls) != 3 {
		t.Errorf("expected 3 batches, got %d", len(client.calls))
	}
	if len(client.calls[0].texts) != 2 {
		t.Errorf("batch 0 size = %d, want 2", len(client.calls[0].texts))
	}
	if len(client.calls[2].texts) != 1 {
		t.Errorf("batch 2 size = %d, want 1", len(client.calls[2].texts))
	}
}

func TestOmniLLMEmbedderError(t *testing.T) {
	expectedErr := errors.New("embedding failed")
	client := &mockOmniLLMClient{
		embedFunc: func(ctx context.Context, model string, texts []string) ([][]float32, error) {
			return nil, expectedErr
		},
	}

	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{})

	signals := []signal.Signal{{ID: "1", Summary: "Test"}}
	_, err := embedder.Embed(context.Background(), signals)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestOmniLLMEmbedderCountMismatch(t *testing.T) {
	client := &mockOmniLLMClient{
		embedFunc: func(ctx context.Context, model string, texts []string) ([][]float32, error) {
			// Return wrong number of embeddings
			return [][]float32{{1.0}}, nil
		},
	}

	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{})

	signals := []signal.Signal{
		{ID: "1", Summary: "First"},
		{ID: "2", Summary: "Second"},
	}
	_, err := embedder.Embed(context.Background(), signals)
	if err == nil {
		t.Fatal("expected count mismatch error")
	}
}

func TestDefaultTextFunc(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{})

	signals := []signal.Signal{
		{ID: "1", Summary: "Summary only"},
		{ID: "2", Summary: "Has both", Description: "With description"},
	}

	_, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatal(err)
	}

	if client.calls[0].texts[0] != "Summary only" {
		t.Errorf("text 0 = %q", client.calls[0].texts[0])
	}
	if client.calls[0].texts[1] != "Has both\n\nWith description" {
		t.Errorf("text 1 = %q", client.calls[0].texts[1])
	}
}

func TestTextFromSummary(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		TextFunc: consolidate.TextFromSummary(),
	})

	signals := []signal.Signal{
		{ID: "1", Summary: "Just summary", Description: "Ignored description"},
	}

	_, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatal(err)
	}

	if client.calls[0].texts[0] != "Just summary" {
		t.Errorf("text = %q, want %q", client.calls[0].texts[0], "Just summary")
	}
}

func TestTextFromMetadata(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		TextFunc: consolidate.TextFromMetadata("full_text"),
	})

	signals := []signal.Signal{
		{
			ID:       "1",
			Summary:  "Fallback",
			Metadata: map[string]any{"full_text": "Custom text from metadata"},
		},
		{
			ID:      "2",
			Summary: "No metadata fallback",
		},
	}

	_, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatal(err)
	}

	if client.calls[0].texts[0] != "Custom text from metadata" {
		t.Errorf("text 0 = %q", client.calls[0].texts[0])
	}
	if client.calls[0].texts[1] != "No metadata fallback" {
		t.Errorf("text 1 = %q", client.calls[0].texts[1])
	}
}

func TestTextWithEntities(t *testing.T) {
	client := &mockOmniLLMClient{}
	embedder := consolidate.NewOmniLLMEmbedder(client, consolidate.EmbedderConfig{
		TextFunc: consolidate.TextWithEntities(),
	})

	signals := []signal.Signal{
		{
			ID:          "1",
			Summary:     "Signal summary",
			Description: "Signal description",
			Entities: []common.Entity{
				{Type: "service", Name: "api-gateway"},
				{Type: "database", Name: "users-db"},
			},
		},
	}

	_, err := embedder.Embed(context.Background(), signals)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Signal summary\n\nSignal description\n\nEntities:\n- service: api-gateway\n- database: users-db"
	if client.calls[0].texts[0] != expected {
		t.Errorf("text = %q, want %q", client.calls[0].texts[0], expected)
	}
}

func TestOmniLLMEmbedderImplementsInterface(t *testing.T) {
	var _ consolidate.Embedder = (*consolidate.OmniLLMEmbedder)(nil)
}

func TestOmniLLMClientInterface(t *testing.T) {
	var _ consolidate.OmniLLMClient = (*mockOmniLLMClient)(nil)
}
