package sqlite

import "errors"

var (
	// ErrNotFound is returned when a requested Signal or RootCause doesn't exist.
	ErrNotFound = errors.New("sqlite: not found")

	// ErrDimensionRequired is returned by Open when no embedding dimension
	// was configured via WithEmbeddingDimension.
	ErrDimensionRequired = errors.New("sqlite: embedding dimension is required (WithEmbeddingDimension)")

	// ErrDimensionMismatch is returned when a saved embedding's length
	// doesn't match the store's configured dimension.
	ErrDimensionMismatch = errors.New("sqlite: embedding length does not match configured dimension")
)
