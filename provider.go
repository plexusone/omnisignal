// Package omnisignal provides a unified interface for ingesting operational signals
// from various external systems (alerting, ticketing, security, monitoring).
//
// omnisignal follows the same architectural pattern as omnillm:
//   - Provider interface defines the contract for signal sources
//   - Registry allows dynamic provider registration
//   - Thick providers use official SDKs (PagerDuty, Datadog)
//   - Thin providers use native HTTP for sources without Go SDKs
//
// Example usage:
//
//	import (
//	    "github.com/plexusone/omnisignal"
//	    _ "github.com/plexusone/omnisignal/provider/pagerduty" // Register PagerDuty
//	)
//
//	provider, err := omnisignal.New("pagerduty", omnisignal.Config{
//	    APIKey: os.Getenv("PAGERDUTY_API_KEY"),
//	})
//	signals, err := provider.Fetch(ctx, omnisignal.FetchOptions{
//	    Since: time.Now().Add(-24 * time.Hour),
//	})
package omnisignal

import (
	"context"
	"errors"
	"time"

	"github.com/plexusone/signal-spec/pkg/signal"
)

// Common errors returned by providers.
var (
	ErrNotSupported     = errors.New("operation not supported by this provider")
	ErrInvalidConfig    = errors.New("invalid provider configuration")
	ErrAuthentication   = errors.New("authentication failed")
	ErrRateLimited      = errors.New("rate limited by provider")
	ErrProviderNotFound = errors.New("provider not found in registry")
)

// Provider defines the interface for signal ingestion providers.
//
// Implementations should be safe for concurrent use.
type Provider interface {
	// Name returns the provider identifier (e.g., "pagerduty", "jira", "newrelic").
	Name() string

	// Fetch retrieves signals matching the given options.
	// Returns signals in chronological order (oldest first).
	// Implementations should handle pagination internally.
	Fetch(ctx context.Context, opts FetchOptions) ([]signal.Signal, error)

	// Subscribe opens a real-time stream of signals.
	// Returns ErrNotSupported if the provider doesn't support streaming.
	// The returned channel is closed when the context is canceled.
	Subscribe(ctx context.Context, opts SubscribeOptions) (<-chan signal.Signal, error)

	// Capabilities returns what this provider supports.
	Capabilities() Capabilities

	// Close releases any resources held by the provider.
	Close() error
}

// FetchOptions configures a fetch operation.
type FetchOptions struct {
	// Since filters signals observed after this time (inclusive).
	Since time.Time

	// Until filters signals observed before this time (exclusive).
	// Zero value means no upper bound.
	Until time.Time

	// Limit is the maximum number of signals to return.
	// Zero means no limit (fetch all matching signals).
	Limit int

	// Statuses filters by signal status in the source system.
	// Empty means all statuses.
	Statuses []string

	// Severities filters by severity level.
	// Empty means all severities.
	Severities []string

	// Filters contains provider-specific filter parameters.
	// Keys and values are provider-dependent.
	Filters map[string]string
}

// SubscribeOptions configures a subscription.
type SubscribeOptions struct {
	// BufferSize is the channel buffer size for incoming signals.
	// Default is 100 if not specified.
	BufferSize int

	// Filters contains provider-specific filter parameters.
	Filters map[string]string
}

// Capabilities describes what features a provider supports.
type Capabilities struct {
	// SupportsStreaming indicates real-time signal streaming via Subscribe.
	SupportsStreaming bool

	// SupportsBatchFetch indicates efficient batch fetching.
	SupportsBatchFetch bool

	// SupportsFiltering indicates server-side filtering support.
	SupportsFiltering bool

	// SupportsAcknowledge indicates the ability to acknowledge signals.
	SupportsAcknowledge bool

	// MaxBatchSize is the maximum signals per fetch request.
	// Zero means no limit or unknown.
	MaxBatchSize int

	// RateLimitPerMinute is the provider's rate limit.
	// Zero means no limit or unknown.
	RateLimitPerMinute int

	// SignalTypes lists the signal types this provider can emit.
	SignalTypes []signal.Type
}

// Config holds provider configuration.
type Config struct {
	// APIKey is the primary authentication credential.
	APIKey string

	// APISecret is a secondary credential (if required).
	APISecret string

	// BaseURL overrides the default API endpoint.
	// Empty uses the provider's default.
	BaseURL string

	// Timeout for API requests. Zero uses provider default.
	Timeout time.Duration

	// RetryMax is the maximum number of retry attempts.
	// Zero means use provider default.
	RetryMax int

	// Options contains provider-specific configuration.
	Options map[string]any
}

// GetOption retrieves a typed option value with a default fallback.
func (c Config) GetOption(key string, defaultVal any) any {
	if c.Options == nil {
		return defaultVal
	}
	if val, ok := c.Options[key]; ok {
		return val
	}
	return defaultVal
}

// GetStringOption retrieves a string option with a default fallback.
func (c Config) GetStringOption(key, defaultVal string) string {
	val := c.GetOption(key, defaultVal)
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}
