package types

import "errors"

// Sentinel errors for common error conditions.
var (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrInvalidConfig is returned when configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrProviderNotAvailable is returned when a provider is not available.
	ErrProviderNotAvailable = errors.New("provider not available")

	// ErrIndexNotFound is returned when the index doesn't exist.
	ErrIndexNotFound = errors.New("index not found")

	// ErrParseError is returned when parsing fails.
	ErrParseError = errors.New("parse error")

	// ErrEmbeddingFailed is returned when embedding generation fails.
	ErrEmbeddingFailed = errors.New("embedding failed")

	// ErrSearchFailed is returned when search fails.
	ErrSearchFailed = errors.New("search failed")

	// ErrStoreFailed is returned when store operation fails.
	ErrStoreFailed = errors.New("store operation failed")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")

	// ErrCancelled is returned when an operation is cancelled.
	ErrCancelled = errors.New("operation cancelled")
)

