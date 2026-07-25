package consolidate_test

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/signal-spec/pkg/rootcause"
	"github.com/plexusone/signal-spec/pkg/signal"
)

type mockEmbedder struct {
	embeddings map[string][]float32
}

func (m *mockEmbedder) Embed(ctx context.Context, signals []signal.Signal) ([]signal.Signal, error) {
	result := make([]signal.Signal, len(signals))
	for i, sig := range signals {
		result[i] = sig
		if emb, ok := m.embeddings[sig.ID]; ok {
			result[i].Embedding = emb
		} else {
			result[i].Embedding = []float32{0.5, 0.5, 0.5}
		}
	}
	return result, nil
}

type mockSummarizer struct{}

func (m *mockSummarizer) Summarize(ctx context.Context, cluster consolidate.Cluster) (rootcause.RootCause, error) {
	now := time.Now()
	return rootcause.RootCause{
		ID:          "rc-" + cluster.ID,
		Title:       "Summary of " + cluster.ID,
		Description: "Generated from cluster",
		Status:      rootcause.StatusNew,
		FirstSeen:   now,
		LastSeen:    now,
	}, nil
}

func TestNewPipeline(t *testing.T) {
	p := consolidate.NewPipeline()
	if p == nil {
		t.Fatal("NewPipeline returned nil")
	}
}

func TestPipelineWithOptions(t *testing.T) {
	embedder := &mockEmbedder{}
	summarizer := &mockSummarizer{}

	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(embedder),
		consolidate.WithSummarizer(summarizer),
		consolidate.WithSimilarityThreshold(0.9),
		consolidate.WithMinClusterSize(2),
		consolidate.WithRequireReview(true),
	)

	if p == nil {
		t.Fatal("NewPipeline with options returned nil")
	}
}

func TestProcessEmptySignals(t *testing.T) {
	p := consolidate.NewPipeline()
	result, err := p.Process(context.Background(), nil)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if result.Stats.SignalsProcessed != 0 {
		t.Errorf("SignalsProcessed = %d, want 0", result.Stats.SignalsProcessed)
	}
}

func TestProcessRequiresEmbedder(t *testing.T) {
	p := consolidate.NewPipeline(
		consolidate.WithSummarizer(&mockSummarizer{}),
	)

	signals := []signal.Signal{
		{ID: "1", Summary: "Test signal"},
	}

	_, err := p.Process(context.Background(), signals)
	if err != consolidate.ErrNoEmbedder {
		t.Errorf("expected ErrNoEmbedder, got %v", err)
	}
}

func TestProcessRequiresSummarizer(t *testing.T) {
	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(&mockEmbedder{}),
	)

	signals := []signal.Signal{
		{ID: "1", Summary: "Test signal"},
	}

	_, err := p.Process(context.Background(), signals)
	if err != consolidate.ErrNoSummarizer {
		t.Errorf("expected ErrNoSummarizer, got %v", err)
	}
}

func TestProcessCuratedSignals(t *testing.T) {
	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(&mockEmbedder{}),
		consolidate.WithSummarizer(&mockSummarizer{}),
	)

	signals := []signal.Signal{
		{
			ID:       "curated-1",
			Summary:  "Curated idea",
			Metadata: map[string]any{"curated": true},
		},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if len(result.RootCauses) != 1 {
		t.Errorf("RootCauses = %d, want 1", len(result.RootCauses))
	}
	if result.Stats.RootCausesGenerated != 1 {
		t.Errorf("RootCausesGenerated = %d, want 1", result.Stats.RootCausesGenerated)
	}
}

func TestProcessRawSignals(t *testing.T) {
	embedder := &mockEmbedder{
		embeddings: map[string][]float32{
			"1": {1.0, 0.0, 0.0},
			"2": {0.99, 0.1, 0.0}, // Similar to 1
			"3": {0.0, 1.0, 0.0},  // Different
		},
	}

	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(embedder),
		consolidate.WithSummarizer(&mockSummarizer{}),
		consolidate.WithSimilarityThreshold(0.9),
	)

	signals := []signal.Signal{
		{ID: "1", Summary: "Signal 1"},
		{ID: "2", Summary: "Signal 2"},
		{ID: "3", Summary: "Signal 3"},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.SignalsEmbedded != 3 {
		t.Errorf("SignalsEmbedded = %d, want 3", result.Stats.SignalsEmbedded)
	}
	if result.Stats.ClustersFormed < 2 {
		t.Errorf("ClustersFormed = %d, want at least 2", result.Stats.ClustersFormed)
	}
}

func TestProcessMixedSignals(t *testing.T) {
	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(&mockEmbedder{}),
		consolidate.WithSummarizer(&mockSummarizer{}),
	)

	signals := []signal.Signal{
		{
			ID:       "curated-1",
			Summary:  "Curated",
			Metadata: map[string]any{"curated": true},
		},
		{
			ID:      "raw-1",
			Summary: "Raw signal",
		},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.SignalsProcessed != 2 {
		t.Errorf("SignalsProcessed = %d, want 2", result.Stats.SignalsProcessed)
	}
	if result.Stats.RootCausesGenerated != 2 {
		t.Errorf("RootCausesGenerated = %d, want 2", result.Stats.RootCausesGenerated)
	}
}

func TestClusteringSimilarSignals(t *testing.T) {
	embedder := &mockEmbedder{
		embeddings: map[string][]float32{
			"a": {1.0, 0.0},
			"b": {1.0, 0.0}, // Identical to a
			"c": {1.0, 0.0}, // Identical to a
		},
	}

	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(embedder),
		consolidate.WithSummarizer(&mockSummarizer{}),
		consolidate.WithSimilarityThreshold(0.99),
	)

	signals := []signal.Signal{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.ClustersFormed != 1 {
		t.Errorf("ClustersFormed = %d, want 1 (all similar)", result.Stats.ClustersFormed)
	}
	if len(result.Clusters) != 1 {
		t.Errorf("Clusters = %d, want 1", len(result.Clusters))
	} else if len(result.Clusters[0].Signals) != 3 {
		t.Errorf("Cluster[0].Signals = %d, want 3", len(result.Clusters[0].Signals))
	}
}

func TestClusteringDissimilarSignals(t *testing.T) {
	embedder := &mockEmbedder{
		embeddings: map[string][]float32{
			"a": {1.0, 0.0},
			"b": {0.0, 1.0},
			"c": {-1.0, 0.0},
		},
	}

	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(embedder),
		consolidate.WithSummarizer(&mockSummarizer{}),
		consolidate.WithSimilarityThreshold(0.9),
	)

	signals := []signal.Signal{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.ClustersFormed != 3 {
		t.Errorf("ClustersFormed = %d, want 3 (all different)", result.Stats.ClustersFormed)
	}
}

func TestMinClusterSize(t *testing.T) {
	embedder := &mockEmbedder{
		embeddings: map[string][]float32{
			"a": {1.0, 0.0},
			"b": {0.0, 1.0},
		},
	}

	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(embedder),
		consolidate.WithSummarizer(&mockSummarizer{}),
		consolidate.WithSimilarityThreshold(0.9),
		consolidate.WithMinClusterSize(2),
	)

	signals := []signal.Signal{
		{ID: "a"},
		{ID: "b"},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.RootCausesGenerated != 0 {
		t.Errorf("RootCausesGenerated = %d, want 0 (clusters too small)", result.Stats.RootCausesGenerated)
	}
	if len(result.Skipped) != 2 {
		t.Errorf("Skipped = %d, want 2", len(result.Skipped))
	}
}

func TestResultStats(t *testing.T) {
	p := consolidate.NewPipeline(
		consolidate.WithEmbedder(&mockEmbedder{}),
		consolidate.WithSummarizer(&mockSummarizer{}),
	)

	signals := []signal.Signal{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	result, err := p.Process(context.Background(), signals)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Stats.Duration == 0 {
		t.Error("Duration should be non-zero")
	}
	if result.Stats.SignalsProcessed != 3 {
		t.Errorf("SignalsProcessed = %d, want 3", result.Stats.SignalsProcessed)
	}
}

func TestStageConstants(t *testing.T) {
	stages := []consolidate.Stage{
		consolidate.StageEmbed,
		consolidate.StageCluster,
		consolidate.StageSummarize,
		consolidate.StageReview,
		consolidate.StageAttach,
	}

	expected := []string{"embed", "cluster", "summarize", "review", "attach"}
	for i, stage := range stages {
		if string(stage) != expected[i] {
			t.Errorf("Stage %d = %q, want %q", i, stage, expected[i])
		}
	}
}

func TestSkippedSignalFields(t *testing.T) {
	s := consolidate.SkippedSignal{
		SignalID: "test-123",
		Reason:   "test reason",
		Stage:    consolidate.StageSummarize,
	}

	if s.SignalID != "test-123" {
		t.Error("SignalID mismatch")
	}
	if s.Reason != "test reason" {
		t.Error("Reason mismatch")
	}
	if s.Stage != consolidate.StageSummarize {
		t.Error("Stage mismatch")
	}
}

func TestClusterFields(t *testing.T) {
	now := time.Now()
	c := consolidate.Cluster{
		ID: "cluster-1",
		Signals: []signal.Signal{
			{ID: "sig-1"},
			{ID: "sig-2"},
		},
		Centroid:  []float32{0.5, 0.5},
		CreatedAt: now,
	}

	if c.ID != "cluster-1" {
		t.Error("ID mismatch")
	}
	if len(c.Signals) != 2 {
		t.Error("Signals length mismatch")
	}
	if len(c.Centroid) != 2 {
		t.Error("Centroid length mismatch")
	}
}

func TestRootCauseFilterFields(t *testing.T) {
	now := time.Now()
	f := consolidate.RootCauseFilter{
		Status: []rootcause.Status{rootcause.StatusNew},
		Since:  now,
		Domain: "product",
		Limit:  100,
	}

	if len(f.Status) != 1 {
		t.Error("Status length mismatch")
	}
	if f.Since != now {
		t.Error("Since mismatch")
	}
	if f.Domain != "product" {
		t.Error("Domain mismatch")
	}
	if f.Limit != 100 {
		t.Error("Limit mismatch")
	}
}

func TestInterfaceTypes(t *testing.T) {
	var _ consolidate.Embedder = (*mockEmbedder)(nil)
	var _ consolidate.Summarizer = (*mockSummarizer)(nil)
}
