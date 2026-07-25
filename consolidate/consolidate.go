// Package consolidate provides a pipeline for consolidating raw signals into
// canonical root causes through embedding, clustering, summarization, and review.
//
// The pipeline consists of five stages:
//   - Embed: Generate vector embeddings for signals via OmniLLM
//   - Cluster: Group similar signals by embedding similarity
//   - Summarize: Generate root cause summaries from clusters via LLM
//   - Review: Optional human review queue for generated root causes
//   - Attach: Link new signals to existing root causes incrementally
//
// Curated signals (marked with metadata["curated"]=true) skip the clustering
// stage and map directly to root causes, preserving their existing aggregation.
//
// Example:
//
//	pipeline := consolidate.NewPipeline(
//	    consolidate.WithEmbedder(embedder),
//	    consolidate.WithSummarizer(summarizer),
//	    consolidate.WithSimilarityThreshold(0.85),
//	)
//	rootCauses, err := pipeline.Process(ctx, signals)
package consolidate

import (
	"context"
	"errors"
	"time"

	"github.com/plexusone/signal-spec/pkg/rootcause"
	"github.com/plexusone/signal-spec/pkg/signal"
)

// Common errors.
var (
	ErrNoEmbedder    = errors.New("embedder not configured")
	ErrNoSummarizer  = errors.New("summarizer not configured")
	ErrEmptyCluster  = errors.New("cluster contains no signals")
	ErrInvalidConfig = errors.New("invalid pipeline configuration")
)

// Stage represents a pipeline stage.
type Stage string

const (
	StageEmbed     Stage = "embed"
	StageCluster   Stage = "cluster"
	StageSummarize Stage = "summarize"
	StageReview    Stage = "review"
	StageAttach    Stage = "attach"
)

// Embedder generates vector embeddings for signals.
type Embedder interface {
	// Embed generates embeddings for the given signals.
	// Returns signals with their Embedding field populated.
	Embed(ctx context.Context, signals []signal.Signal) ([]signal.Signal, error)
}

// Summarizer generates root cause summaries from signal clusters.
type Summarizer interface {
	// Summarize generates a root cause from a cluster of signals.
	Summarize(ctx context.Context, cluster Cluster) (rootcause.RootCause, error)
}

// Reviewer handles human review of generated root causes.
type Reviewer interface {
	// Submit queues a root cause for review.
	Submit(ctx context.Context, rc rootcause.RootCause) error

	// Pending returns root causes awaiting review.
	Pending(ctx context.Context) ([]rootcause.RootCause, error)

	// Approve marks a root cause as reviewed and approved.
	Approve(ctx context.Context, id string) error

	// Reject marks a root cause as reviewed and rejected.
	Reject(ctx context.Context, id string, reason string) error
}

// Store persists root causes and signal-to-root-cause links.
type Store interface {
	// SaveRootCause persists a root cause.
	SaveRootCause(ctx context.Context, rc rootcause.RootCause) error

	// GetRootCause retrieves a root cause by ID.
	GetRootCause(ctx context.Context, id string) (rootcause.RootCause, error)

	// ListRootCauses returns all root causes matching the filter.
	ListRootCauses(ctx context.Context, filter RootCauseFilter) ([]rootcause.RootCause, error)

	// LinkSignal associates a signal with a root cause.
	LinkSignal(ctx context.Context, signalID, rootCauseID string) error

	// GetLinkedSignals returns signal IDs linked to a root cause.
	GetLinkedSignals(ctx context.Context, rootCauseID string) ([]string, error)
}

// RootCauseFilter specifies criteria for listing root causes.
type RootCauseFilter struct {
	// Status filters by root cause status.
	Status []rootcause.Status

	// Since filters root causes created after this time.
	Since time.Time

	// Domain filters by domain name.
	Domain string

	// Limit is the maximum number of results.
	Limit int
}

// Cluster represents a group of similar signals.
type Cluster struct {
	// ID is the cluster identifier.
	ID string

	// Signals are the signals in this cluster.
	Signals []signal.Signal

	// Centroid is the average embedding of the cluster.
	Centroid []float32

	// CreatedAt is when the cluster was created.
	CreatedAt time.Time
}

// Result holds the output of pipeline processing.
type Result struct {
	// RootCauses are the generated or updated root causes.
	RootCauses []rootcause.RootCause

	// Clusters are the signal clusters formed.
	Clusters []Cluster

	// Attached are signals linked to existing root causes.
	Attached map[string]string // signal ID -> root cause ID

	// Skipped are signals that couldn't be processed.
	Skipped []SkippedSignal

	// Stats holds processing statistics.
	Stats Stats
}

// SkippedSignal records a signal that was skipped during processing.
type SkippedSignal struct {
	SignalID string
	Reason   string
	Stage    Stage
}

// Stats holds pipeline processing statistics.
type Stats struct {
	// SignalsProcessed is the total signals processed.
	SignalsProcessed int

	// SignalsEmbedded is the count embedded successfully.
	SignalsEmbedded int

	// SignalsClustered is the count assigned to clusters.
	SignalsClustered int

	// SignalsAttached is the count attached to existing root causes.
	SignalsAttached int

	// ClustersFormed is the number of new clusters created.
	ClustersFormed int

	// RootCausesGenerated is the number of new root causes.
	RootCausesGenerated int

	// Duration is the total processing time.
	Duration time.Duration
}

// Pipeline orchestrates signal consolidation.
type Pipeline struct {
	embedder            Embedder
	summarizer          Summarizer
	reviewer            Reviewer
	store               Store
	similarityThreshold float64
	minClusterSize      int
	requireReview       bool
}

// Option configures a Pipeline.
type Option func(*Pipeline)

// WithEmbedder sets the embedding provider.
func WithEmbedder(e Embedder) Option {
	return func(p *Pipeline) { p.embedder = e }
}

// WithSummarizer sets the summarization provider.
func WithSummarizer(s Summarizer) Option {
	return func(p *Pipeline) { p.summarizer = s }
}

// WithReviewer sets the review queue handler.
func WithReviewer(r Reviewer) Option {
	return func(p *Pipeline) { p.reviewer = r }
}

// WithStore sets the persistence store.
func WithStore(s Store) Option {
	return func(p *Pipeline) { p.store = s }
}

// WithSimilarityThreshold sets the clustering similarity threshold.
// Signals with embedding similarity >= threshold are grouped together.
// Default is 0.85.
func WithSimilarityThreshold(t float64) Option {
	return func(p *Pipeline) { p.similarityThreshold = t }
}

// WithMinClusterSize sets the minimum signals required to form a cluster.
// Default is 1.
func WithMinClusterSize(n int) Option {
	return func(p *Pipeline) { p.minClusterSize = n }
}

// WithRequireReview enables mandatory human review for generated root causes.
func WithRequireReview(required bool) Option {
	return func(p *Pipeline) { p.requireReview = required }
}

// NewPipeline creates a new consolidation pipeline.
func NewPipeline(opts ...Option) *Pipeline {
	p := &Pipeline{
		similarityThreshold: 0.85,
		minClusterSize:      1,
		requireReview:       false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Process runs the consolidation pipeline on the given signals.
func (p *Pipeline) Process(ctx context.Context, signals []signal.Signal) (Result, error) {
	start := time.Now()
	result := Result{
		Attached: make(map[string]string),
		Stats:    Stats{SignalsProcessed: len(signals)},
	}

	if len(signals) == 0 {
		return result, nil
	}

	// Separate curated and raw signals
	var curated, raw []signal.Signal
	for _, sig := range signals {
		if isCurated(sig) {
			curated = append(curated, sig)
		} else {
			raw = append(raw, sig)
		}
	}

	// Process curated signals directly (skip clustering)
	for _, sig := range curated {
		rc, err := p.curatedToRootCause(ctx, sig)
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedSignal{
				SignalID: sig.ID,
				Reason:   err.Error(),
				Stage:    StageSummarize,
			})
			continue
		}
		result.RootCauses = append(result.RootCauses, rc)
		result.Stats.RootCausesGenerated++
	}

	// Process raw signals through full pipeline
	if len(raw) > 0 {
		rawResult, err := p.processRaw(ctx, raw)
		if err != nil {
			return result, err
		}
		result.Clusters = rawResult.Clusters
		result.RootCauses = append(result.RootCauses, rawResult.RootCauses...)
		result.Skipped = append(result.Skipped, rawResult.Skipped...)
		for k, v := range rawResult.Attached {
			result.Attached[k] = v
		}
		result.Stats.SignalsEmbedded = rawResult.Stats.SignalsEmbedded
		result.Stats.SignalsClustered = rawResult.Stats.SignalsClustered
		result.Stats.SignalsAttached = rawResult.Stats.SignalsAttached
		result.Stats.ClustersFormed = rawResult.Stats.ClustersFormed
		result.Stats.RootCausesGenerated += rawResult.Stats.RootCausesGenerated
	}

	result.Stats.Duration = time.Since(start)
	return result, nil
}

// processRaw handles the full pipeline for raw (non-curated) signals.
func (p *Pipeline) processRaw(ctx context.Context, signals []signal.Signal) (Result, error) {
	result := Result{
		Attached: make(map[string]string),
	}

	// Stage 1: Embed
	if p.embedder == nil {
		return result, ErrNoEmbedder
	}
	embedded, err := p.embedder.Embed(ctx, signals)
	if err != nil {
		return result, err
	}
	result.Stats.SignalsEmbedded = len(embedded)

	// Stage 2: Cluster
	clusters := p.cluster(embedded)
	result.Clusters = clusters
	result.Stats.ClustersFormed = len(clusters)
	for _, c := range clusters {
		result.Stats.SignalsClustered += len(c.Signals)
	}

	// Stage 3: Summarize
	if p.summarizer == nil {
		return result, ErrNoSummarizer
	}
	for _, cluster := range clusters {
		if len(cluster.Signals) < p.minClusterSize {
			for _, sig := range cluster.Signals {
				result.Skipped = append(result.Skipped, SkippedSignal{
					SignalID: sig.ID,
					Reason:   "cluster below minimum size",
					Stage:    StageSummarize,
				})
			}
			continue
		}

		rc, err := p.summarizer.Summarize(ctx, cluster)
		if err != nil {
			for _, sig := range cluster.Signals {
				result.Skipped = append(result.Skipped, SkippedSignal{
					SignalID: sig.ID,
					Reason:   err.Error(),
					Stage:    StageSummarize,
				})
			}
			continue
		}

		// Stage 4: Review (optional)
		if p.requireReview && p.reviewer != nil {
			rc.Status = rootcause.StatusNew
			if err := p.reviewer.Submit(ctx, rc); err != nil {
				result.Skipped = append(result.Skipped, SkippedSignal{
					SignalID: cluster.ID,
					Reason:   err.Error(),
					Stage:    StageReview,
				})
				continue
			}
		}

		result.RootCauses = append(result.RootCauses, rc)
		result.Stats.RootCausesGenerated++

		// Link signals to root cause
		if p.store != nil {
			for _, sig := range cluster.Signals {
				if err := p.store.LinkSignal(ctx, sig.ID, rc.ID); err == nil {
					result.Attached[sig.ID] = rc.ID
					result.Stats.SignalsAttached++
				}
			}
		}
	}

	return result, nil
}

// cluster groups signals by embedding similarity.
func (p *Pipeline) cluster(signals []signal.Signal) []Cluster {
	if len(signals) == 0 {
		return nil
	}

	var clusters []Cluster
	assigned := make(map[int]bool)

	for i, sig := range signals {
		if assigned[i] {
			continue
		}

		cluster := Cluster{
			ID:        sig.ID,
			Signals:   []signal.Signal{sig},
			Centroid:  sig.Embedding,
			CreatedAt: time.Now(),
		}
		assigned[i] = true

		// Find similar signals
		for j := i + 1; j < len(signals); j++ {
			if assigned[j] {
				continue
			}
			sim := cosineSimilarity(sig.Embedding, signals[j].Embedding)
			if sim >= p.similarityThreshold {
				cluster.Signals = append(cluster.Signals, signals[j])
				assigned[j] = true
			}
		}

		// Update centroid
		if len(cluster.Signals) > 1 {
			cluster.Centroid = computeCentroid(cluster.Signals)
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// curatedToRootCause converts a curated signal directly to a root cause.
func (p *Pipeline) curatedToRootCause(ctx context.Context, sig signal.Signal) (rootcause.RootCause, error) {
	if p.summarizer != nil {
		// Use summarizer for consistency
		return p.summarizer.Summarize(ctx, Cluster{
			ID:        sig.ID,
			Signals:   []signal.Signal{sig},
			CreatedAt: time.Now(),
		})
	}

	// Fallback: create root cause directly from signal
	now := time.Now()
	return rootcause.RootCause{
		ID:          "rc-" + sig.ID,
		Title:       sig.Summary,
		Description: sig.Description,
		Domain:      sig.Domain,
		Status:      rootcause.StatusNew,
		FirstSeen:   now,
		LastSeen:    now,
	}, nil
}

// Attach links a new signal to an existing root cause if similar enough.
// Returns the root cause ID if attached, empty string otherwise.
func (p *Pipeline) Attach(ctx context.Context, sig signal.Signal, rootCauses []rootcause.RootCause) (string, error) {
	if len(sig.Embedding) == 0 {
		if p.embedder == nil {
			return "", ErrNoEmbedder
		}
		embedded, err := p.embedder.Embed(ctx, []signal.Signal{sig})
		if err != nil {
			return "", err
		}
		sig = embedded[0]
	}

	// Find most similar root cause
	var bestID string
	var bestSim float64

	for _, rc := range rootCauses {
		if len(rc.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(sig.Embedding, rc.Embedding)
		if sim >= p.similarityThreshold && sim > bestSim {
			bestSim = sim
			bestID = rc.ID
		}
	}

	if bestID != "" && p.store != nil {
		if err := p.store.LinkSignal(ctx, sig.ID, bestID); err != nil {
			return "", err
		}
	}

	return bestID, nil
}

// isCurated checks if a signal is marked as curated.
func isCurated(sig signal.Signal) bool {
	if sig.Metadata == nil {
		return false
	}
	v, ok := sig.Metadata["curated"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (sqrt(normA) * sqrt(normB))
}

// computeCentroid calculates the average embedding of signals.
func computeCentroid(signals []signal.Signal) []float32 {
	if len(signals) == 0 {
		return nil
	}

	dim := len(signals[0].Embedding)
	if dim == 0 {
		return nil
	}

	centroid := make([]float32, dim)
	for _, sig := range signals {
		for i, v := range sig.Embedding {
			if i < dim {
				centroid[i] += v
			}
		}
	}

	n := float32(len(signals))
	for i := range centroid {
		centroid[i] /= n
	}

	return centroid
}

// sqrt is a simple square root approximation.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
