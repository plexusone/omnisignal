package omnisignal

import (
	"fmt"
	"sort"
	"sync"
)

// Priority levels for provider registration.
// Higher priority providers override lower priority ones.
const (
	PriorityThin  = 0  // Native HTTP implementations
	PriorityThick = 10 // SDK-based implementations
)

// ProviderFactory creates a new provider instance from configuration.
type ProviderFactory func(cfg Config) (Provider, error)

type registryEntry struct {
	factory  ProviderFactory
	priority int
}

var (
	registry = make(map[string]registryEntry)
	mu       sync.RWMutex
)

// Register adds a provider factory to the global registry.
//
// If a provider with the same name exists, the one with higher priority wins.
// For equal priorities, the later registration wins.
//
// Example:
//
//	func init() {
//	    omnisignal.Register("pagerduty", NewProvider, omnisignal.PriorityThick)
//	}
func Register(name string, factory ProviderFactory, priority int) {
	mu.Lock()
	defer mu.Unlock()

	existing, exists := registry[name]
	if exists && existing.priority > priority {
		// Existing registration has higher priority, keep it
		return
	}

	registry[name] = registryEntry{
		factory:  factory,
		priority: priority,
	}
}

// New creates a provider instance by name using the registered factory.
//
// Returns ErrProviderNotFound if no provider is registered with that name.
func New(name string, cfg Config) (Provider, error) {
	mu.RLock()
	entry, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	return entry.factory(cfg)
}

// MustNew creates a provider instance, panicking on error.
// Use only in initialization code where failure is unrecoverable.
func MustNew(name string, cfg Config) Provider {
	p, err := New(name, cfg)
	if err != nil {
		panic(fmt.Sprintf("omnisignal: failed to create provider %s: %v", name, err))
	}
	return p
}

// List returns all registered provider names in alphabetical order.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsRegistered checks if a provider is registered.
func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[name]
	return ok
}

// GetPriority returns the priority of a registered provider.
// Returns -1 if the provider is not registered.
func GetPriority(name string) int {
	mu.RLock()
	defer mu.RUnlock()

	if entry, ok := registry[name]; ok {
		return entry.priority
	}
	return -1
}

// Unregister removes a provider from the registry.
// Primarily useful for testing.
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(registry, name)
}

// ClearRegistry removes all providers from the registry.
// Primarily useful for testing.
func ClearRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]registryEntry)
}
