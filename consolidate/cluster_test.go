package consolidate_test

import (
	"testing"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func TestDefaultClusterConfig(t *testing.T) {
	cfg := consolidate.DefaultClusterConfig()
	if cfg.SimilarityThreshold != 0.85 {
		t.Errorf("SimilarityThreshold = %f, want 0.85", cfg.SimilarityThreshold)
	}
	if cfg.MinClusterSize != 1 {
		t.Errorf("MinClusterSize = %d, want 1", cfg.MinClusterSize)
	}
	if cfg.MaxClusterSize != 0 {
		t.Errorf("MaxClusterSize = %d, want 0", cfg.MaxClusterSize)
	}
}

func TestNewClusterer(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.DefaultClusterConfig())
	if c == nil {
		t.Fatal("NewClusterer returned nil")
	}
}

func TestClustererEmptySignals(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.DefaultClusterConfig())
	clusters, outliers := c.Cluster(nil)
	if len(clusters) != 0 {
		t.Errorf("clusters = %d, want 0", len(clusters))
	}
	if len(outliers) != 0 {
		t.Errorf("outliers = %d, want 0", len(outliers))
	}
}

func TestClustererIdenticalEmbeddings(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.99,
		MinClusterSize:      1,
	})

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: []float32{1.0, 0.0}},
		{ID: "c", Embedding: []float32{1.0, 0.0}},
	}

	clusters, outliers := c.Cluster(signals)

	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1", len(clusters))
	}
	if len(clusters[0].Signals) != 3 {
		t.Errorf("cluster size = %d, want 3", len(clusters[0].Signals))
	}
	if len(outliers) != 0 {
		t.Errorf("outliers = %d, want 0", len(outliers))
	}
}

func TestClustererDifferentEmbeddings(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.9,
		MinClusterSize:      1,
	})

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: []float32{0.0, 1.0}},
		{ID: "c", Embedding: []float32{-1.0, 0.0}},
	}

	clusters, outliers := c.Cluster(signals)

	if len(clusters) != 3 {
		t.Fatalf("clusters = %d, want 3", len(clusters))
	}
	if len(outliers) != 0 {
		t.Errorf("outliers = %d, want 0", len(outliers))
	}
}

func TestClustererMinClusterSize(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.9,
		MinClusterSize:      2,
	})

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: []float32{0.0, 1.0}},
	}

	clusters, outliers := c.Cluster(signals)

	if len(clusters) != 0 {
		t.Errorf("clusters = %d, want 0 (all below min size)", len(clusters))
	}
	if len(outliers) != 2 {
		t.Errorf("outliers = %d, want 2", len(outliers))
	}
}

func TestClustererMaxClusterSize(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.99,
		MinClusterSize:      1,
		MaxClusterSize:      2,
	})

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: []float32{1.0, 0.0}},
		{ID: "c", Embedding: []float32{1.0, 0.0}},
	}

	clusters, _ := c.Cluster(signals)

	if len(clusters) != 2 {
		t.Fatalf("clusters = %d, want 2 (one capped, one new)", len(clusters))
	}
	if len(clusters[0].Signals) != 2 {
		t.Errorf("cluster 0 size = %d, want 2", len(clusters[0].Signals))
	}
}

func TestClustererNoEmbedding(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.DefaultClusterConfig())

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: nil},
		{ID: "c"},
	}

	clusters, outliers := c.Cluster(signals)

	if len(clusters) != 1 {
		t.Errorf("clusters = %d, want 1", len(clusters))
	}
	if len(outliers) != 2 {
		t.Errorf("outliers = %d, want 2 (no embeddings)", len(outliers))
	}
}

func TestFindBestCluster(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.8,
	})

	clusters := []consolidate.Cluster{
		{ID: "c1", Centroid: []float32{1.0, 0.0}},
		{ID: "c2", Centroid: []float32{0.0, 1.0}},
	}

	sig := signal.Signal{ID: "new", Embedding: []float32{0.95, 0.3}}

	best, sim := c.FindBestCluster(sig, clusters)
	if best == nil {
		t.Fatal("expected a match")
	}
	if best.ID != "c1" {
		t.Errorf("best cluster = %s, want c1", best.ID)
	}
	if sim < 0.8 {
		t.Errorf("similarity = %f, should be >= 0.8", sim)
	}
}

func TestFindBestClusterNoMatch(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.ClusterConfig{
		SimilarityThreshold: 0.99,
	})

	clusters := []consolidate.Cluster{
		{ID: "c1", Centroid: []float32{1.0, 0.0}},
	}

	sig := signal.Signal{ID: "new", Embedding: []float32{0.0, 1.0}}

	best, sim := c.FindBestCluster(sig, clusters)
	if best != nil {
		t.Error("expected no match")
	}
	if sim >= 0 {
		t.Errorf("similarity = %f, should be < 0", sim)
	}
}

func TestAttachToCluster(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.DefaultClusterConfig())

	cluster := consolidate.Cluster{
		ID:       "c1",
		Signals:  []signal.Signal{{ID: "a", Embedding: []float32{1.0, 0.0}}},
		Centroid: []float32{1.0, 0.0},
	}

	newSig := signal.Signal{ID: "b", Embedding: []float32{0.9, 0.1}}
	c.AttachToCluster(newSig, &cluster)

	if len(cluster.Signals) != 2 {
		t.Errorf("cluster size = %d, want 2", len(cluster.Signals))
	}
	if len(cluster.Centroid) != 2 {
		t.Error("centroid should be updated")
	}
}

func TestMergeClusters(t *testing.T) {
	c := consolidate.NewClusterer(consolidate.DefaultClusterConfig())

	a := consolidate.Cluster{
		ID:      "a",
		Signals: []signal.Signal{{ID: "s1"}, {ID: "s2"}},
	}
	b := consolidate.Cluster{
		ID:      "b",
		Signals: []signal.Signal{{ID: "s3"}},
	}

	merged := c.MergeClusters(a, b)

	if merged.ID != "a" {
		t.Errorf("merged ID = %s, want a", merged.ID)
	}
	if len(merged.Signals) != 3 {
		t.Errorf("merged signals = %d, want 3", len(merged.Signals))
	}
}

func TestClusterSimilarity(t *testing.T) {
	a := consolidate.Cluster{Centroid: []float32{1.0, 0.0}}
	b := consolidate.Cluster{Centroid: []float32{1.0, 0.0}}

	sim := consolidate.ClusterSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("similarity = %f, want 1.0", sim)
	}

	c := consolidate.Cluster{Centroid: []float32{0.0, 1.0}}
	sim = consolidate.ClusterSimilarity(a, c)
	if sim != 0.0 {
		t.Errorf("similarity = %f, want 0.0", sim)
	}
}

func TestIncrementalClusterer(t *testing.T) {
	ic := consolidate.NewIncrementalClusterer(
		consolidate.ClusterConfig{SimilarityThreshold: 0.9},
		nil,
	)

	if ic.ClusterCount() != 0 {
		t.Error("should start with 0 clusters")
	}

	// Add first signal - creates new cluster
	clusterID := ic.Add(signal.Signal{ID: "a", Embedding: []float32{1.0, 0.0}})
	if clusterID != "a" {
		t.Errorf("cluster ID = %s, want a", clusterID)
	}
	if ic.ClusterCount() != 1 {
		t.Errorf("cluster count = %d, want 1", ic.ClusterCount())
	}

	// Add similar signal - attaches to existing
	clusterID = ic.Add(signal.Signal{ID: "b", Embedding: []float32{0.99, 0.1}})
	if clusterID != "a" {
		t.Errorf("cluster ID = %s, want a (should attach)", clusterID)
	}
	if ic.ClusterCount() != 1 {
		t.Errorf("cluster count = %d, want 1", ic.ClusterCount())
	}

	// Add different signal - creates new cluster
	clusterID = ic.Add(signal.Signal{ID: "c", Embedding: []float32{0.0, 1.0}})
	if clusterID != "c" {
		t.Errorf("cluster ID = %s, want c", clusterID)
	}
	if ic.ClusterCount() != 2 {
		t.Errorf("cluster count = %d, want 2", ic.ClusterCount())
	}
}

func TestIncrementalClustererNoEmbedding(t *testing.T) {
	ic := consolidate.NewIncrementalClusterer(consolidate.DefaultClusterConfig(), nil)

	clusterID := ic.Add(signal.Signal{ID: "no-embedding"})
	if clusterID != "" {
		t.Errorf("expected empty cluster ID for signal without embedding")
	}
}

func TestIncrementalClustererAddBatch(t *testing.T) {
	ic := consolidate.NewIncrementalClusterer(
		consolidate.ClusterConfig{SimilarityThreshold: 0.99},
		nil,
	)

	signals := []signal.Signal{
		{ID: "a", Embedding: []float32{1.0, 0.0}},
		{ID: "b", Embedding: []float32{1.0, 0.0}},
		{ID: "c", Embedding: []float32{0.0, 1.0}},
	}

	result := ic.AddBatch(signals)

	if len(result) != 3 {
		t.Errorf("result size = %d, want 3", len(result))
	}
	if result["a"] != "a" {
		t.Errorf("a mapped to %s, want a", result["a"])
	}
	if result["b"] != "a" {
		t.Errorf("b mapped to %s, want a (same cluster)", result["b"])
	}
	if result["c"] != "c" {
		t.Errorf("c mapped to %s, want c (new cluster)", result["c"])
	}
}

func TestIncrementalClustererWithExisting(t *testing.T) {
	existing := []consolidate.Cluster{
		{ID: "existing", Centroid: []float32{1.0, 0.0}, Signals: []signal.Signal{{ID: "old"}}},
	}

	ic := consolidate.NewIncrementalClusterer(
		consolidate.ClusterConfig{SimilarityThreshold: 0.9},
		existing,
	)

	if ic.ClusterCount() != 1 {
		t.Errorf("should start with 1 existing cluster")
	}

	// Add similar signal
	clusterID := ic.Add(signal.Signal{ID: "new", Embedding: []float32{0.99, 0.1}})
	if clusterID != "existing" {
		t.Errorf("should attach to existing cluster, got %s", clusterID)
	}
}
