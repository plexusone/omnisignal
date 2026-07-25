package consolidate

import (
	"context"
	"fmt"

	"github.com/plexusone/signal-spec/pkg/signal"
)

// EmbedderConfig configures an OmniLLM-based embedder.
type EmbedderConfig struct {
	// Model is the embedding model to use (e.g., "text-embedding-3-small").
	Model string

	// BatchSize is the maximum signals per embedding request.
	// Default is 100.
	BatchSize int

	// TextFunc extracts the text to embed from a signal.
	// Default concatenates Summary and Description.
	TextFunc func(signal.Signal) string
}

// OmniLLMClient is the interface for embedding requests.
// This matches the omnillm.Client embedding API.
type OmniLLMClient interface {
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, model string, texts []string) ([][]float32, error)
}

// OmniLLMEmbedder implements Embedder using OmniLLM.
type OmniLLMEmbedder struct {
	client OmniLLMClient
	config EmbedderConfig
}

// NewOmniLLMEmbedder creates an embedder backed by OmniLLM.
func NewOmniLLMEmbedder(client OmniLLMClient, cfg EmbedderConfig) *OmniLLMEmbedder {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.TextFunc == nil {
		cfg.TextFunc = defaultTextFunc
	}
	return &OmniLLMEmbedder{
		client: client,
		config: cfg,
	}
}

// Embed generates embeddings for signals via OmniLLM.
func (e *OmniLLMEmbedder) Embed(ctx context.Context, signals []signal.Signal) ([]signal.Signal, error) {
	if len(signals) == 0 {
		return signals, nil
	}

	result := make([]signal.Signal, len(signals))
	copy(result, signals)

	// Process in batches
	for i := 0; i < len(result); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(result) {
			end = len(result)
		}

		batch := result[i:end]
		texts := make([]string, len(batch))
		for j, sig := range batch {
			texts[j] = e.config.TextFunc(sig)
		}

		embeddings, err := e.client.Embed(ctx, e.config.Model, texts)
		if err != nil {
			return nil, fmt.Errorf("embedding batch %d: %w", i/e.config.BatchSize, err)
		}

		if len(embeddings) != len(batch) {
			return nil, fmt.Errorf("embedding count mismatch: got %d, want %d", len(embeddings), len(batch))
		}

		for j := range batch {
			result[i+j].Embedding = embeddings[j]
		}
	}

	return result, nil
}

// defaultTextFunc concatenates Summary and Description.
func defaultTextFunc(sig signal.Signal) string {
	if sig.Description == "" {
		return sig.Summary
	}
	return sig.Summary + "\n\n" + sig.Description
}

// TextFromSummary returns a TextFunc that only uses Summary.
func TextFromSummary() func(signal.Signal) string {
	return func(sig signal.Signal) string {
		return sig.Summary
	}
}

// TextFromMetadata returns a TextFunc that extracts text from a metadata key.
func TextFromMetadata(key string) func(signal.Signal) string {
	return func(sig signal.Signal) string {
		if sig.Metadata == nil {
			return sig.Summary
		}
		if v, ok := sig.Metadata[key].(string); ok && v != "" {
			return v
		}
		return sig.Summary
	}
}

// TextWithEntities returns a TextFunc that includes entity names.
func TextWithEntities() func(signal.Signal) string {
	return func(sig signal.Signal) string {
		text := sig.Summary
		if sig.Description != "" {
			text += "\n\n" + sig.Description
		}
		if len(sig.Entities) > 0 {
			text += "\n\nEntities:"
			for _, e := range sig.Entities {
				text += "\n- " + e.Type + ": " + e.Name
			}
		}
		return text
	}
}
