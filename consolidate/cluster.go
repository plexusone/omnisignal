package consolidate

import (
	"github.com/plexusone/signal-spec/pkg/signal"
)

// ClusterConfig configures the clustering algorithm.
type ClusterConfig struct {
	// SimilarityThreshold is the minimum cosine similarity for grouping.
	// Default is 0.85.
	SimilarityThreshold float64

	// MinClusterSize is the minimum signals to form a valid cluster.
	// Signals in smaller clusters are treated as outliers.
	// Default is 1.
	MinClusterSize int

	// MaxClusterSize limits cluster growth.
	// Zero means no limit.
	MaxClusterSize int
}

// DefaultClusterConfig returns the default clustering configuration.
func DefaultClusterConfig() ClusterConfig {
	return ClusterConfig{
		SimilarityThreshold: 0.85,
		MinClusterSize:      1,
		MaxClusterSize:      0,
	}
}

// Clusterer groups signals by embedding similarity.
type Clusterer struct {
	config ClusterConfig
}

// NewClusterer creates a new signal clusterer.
func NewClusterer(cfg ClusterConfig) *Clusterer {
	if cfg.SimilarityThreshold <= 0 {
		cfg.SimilarityThreshold = 0.85
	}
	if cfg.MinClusterSize <= 0 {
		cfg.MinClusterSize = 1
	}
	return &Clusterer{config: cfg}
}

// Cluster groups signals by embedding similarity.
// Returns clusters and outlier signals that didn't fit any cluster.
func (c *Clusterer) Cluster(signals []signal.Signal) ([]Cluster, []signal.Signal) {
	if len(signals) == 0 {
		return nil, nil
	}

	var clusters []Cluster
	var outliers []signal.Signal
	assigned := make(map[int]bool)

	for i, sig := range signals {
		if assigned[i] {
			continue
		}
		if len(sig.Embedding) == 0 {
			outliers = append(outliers, sig)
			assigned[i] = true
			continue
		}

		cluster := Cluster{
			ID:       sig.ID,
			Signals:  []signal.Signal{sig},
			Centroid: sig.Embedding,
		}
		assigned[i] = true

		for j := i + 1; j < len(signals); j++ {
			if assigned[j] {
				continue
			}
			if len(signals[j].Embedding) == 0 {
				continue
			}

			if c.config.MaxClusterSize > 0 && len(cluster.Signals) >= c.config.MaxClusterSize {
				break
			}

			sim := cosineSimilarity(sig.Embedding, signals[j].Embedding)
			if sim >= c.config.SimilarityThreshold {
				cluster.Signals = append(cluster.Signals, signals[j])
				assigned[j] = true
			}
		}

		if len(cluster.Signals) >= c.config.MinClusterSize {
			if len(cluster.Signals) > 1 {
				cluster.Centroid = computeCentroid(cluster.Signals)
			}
			clusters = append(clusters, cluster)
		} else {
			outliers = append(outliers, cluster.Signals...)
		}
	}

	return clusters, outliers
}

// FindBestCluster returns the cluster most similar to the signal.
// Returns nil and -1 similarity if no cluster meets the threshold.
func (c *Clusterer) FindBestCluster(sig signal.Signal, clusters []Cluster) (*Cluster, float64) {
	if len(sig.Embedding) == 0 {
		return nil, -1
	}

	var best *Cluster
	var bestSim float64 = -1

	for i := range clusters {
		if len(clusters[i].Centroid) == 0 {
			continue
		}

		sim := cosineSimilarity(sig.Embedding, clusters[i].Centroid)
		if sim >= c.config.SimilarityThreshold && sim > bestSim {
			best = &clusters[i]
			bestSim = sim
		}
	}

	return best, bestSim
}

// AttachToCluster adds a signal to a cluster and updates the centroid.
func (c *Clusterer) AttachToCluster(sig signal.Signal, cluster *Cluster) {
	if c.config.MaxClusterSize > 0 && len(cluster.Signals) >= c.config.MaxClusterSize {
		return
	}

	cluster.Signals = append(cluster.Signals, sig)
	cluster.Centroid = computeCentroid(cluster.Signals)
}

// MergeClusters combines two clusters into one.
func (c *Clusterer) MergeClusters(a, b Cluster) Cluster {
	signals := make([]signal.Signal, 0, len(a.Signals)+len(b.Signals))
	signals = append(signals, a.Signals...)
	signals = append(signals, b.Signals...)

	return Cluster{
		ID:       a.ID,
		Signals:  signals,
		Centroid: computeCentroid(signals),
	}
}

// ClusterSimilarity computes the similarity between two clusters.
func ClusterSimilarity(a, b Cluster) float64 {
	if len(a.Centroid) == 0 || len(b.Centroid) == 0 {
		return 0
	}
	return cosineSimilarity(a.Centroid, b.Centroid)
}

// IncrementalClusterer maintains clusters and attaches new signals incrementally.
type IncrementalClusterer struct {
	clusterer *Clusterer
	clusters  []Cluster
}

// NewIncrementalClusterer creates an incremental clusterer with existing clusters.
func NewIncrementalClusterer(cfg ClusterConfig, existingClusters []Cluster) *IncrementalClusterer {
	return &IncrementalClusterer{
		clusterer: NewClusterer(cfg),
		clusters:  existingClusters,
	}
}

// Add processes a new signal, either attaching to an existing cluster or creating a new one.
// Returns the cluster ID the signal was assigned to, or empty if it became an outlier.
func (ic *IncrementalClusterer) Add(sig signal.Signal) string {
	if len(sig.Embedding) == 0 {
		return ""
	}

	best, _ := ic.clusterer.FindBestCluster(sig, ic.clusters)
	if best != nil {
		ic.clusterer.AttachToCluster(sig, best)
		return best.ID
	}

	// Create new cluster
	newCluster := Cluster{
		ID:       sig.ID,
		Signals:  []signal.Signal{sig},
		Centroid: sig.Embedding,
	}
	ic.clusters = append(ic.clusters, newCluster)
	return newCluster.ID
}

// AddBatch processes multiple signals.
// Returns a map of signal ID to cluster ID.
func (ic *IncrementalClusterer) AddBatch(signals []signal.Signal) map[string]string {
	result := make(map[string]string)
	for _, sig := range signals {
		if clusterID := ic.Add(sig); clusterID != "" {
			result[sig.ID] = clusterID
		}
	}
	return result
}

// Clusters returns all current clusters.
func (ic *IncrementalClusterer) Clusters() []Cluster {
	return ic.clusters
}

// ClusterCount returns the number of clusters.
func (ic *IncrementalClusterer) ClusterCount() int {
	return len(ic.clusters)
}
