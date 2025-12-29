package types

import (
	"errors"
	"fmt"
)

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

// ChunkError represents an error related to chunk processing.
type ChunkError struct {
	FilePath string
	Line     int
	Err      error
}

func (e *ChunkError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("chunk error at %s:%d: %v", e.FilePath, e.Line, e.Err)
	}
	return fmt.Sprintf("chunk error in %s: %v", e.FilePath, e.Err)
}

func (e *ChunkError) Unwrap() error {
	return e.Err
}

// NewChunkError creates a new ChunkError.
func NewChunkError(filePath string, line int, err error) *ChunkError {
	return &ChunkError{FilePath: filePath, Line: line, Err: err}
}

// SymbolError represents an error related to symbol processing.
type SymbolError struct {
	SymbolID   string
	SymbolName string
	Err        error
}

func (e *SymbolError) Error() string {
	if e.SymbolName != "" {
		return fmt.Sprintf("symbol error for '%s': %v", e.SymbolName, e.Err)
	}
	return fmt.Sprintf("symbol error for %s: %v", e.SymbolID, e.Err)
}

func (e *SymbolError) Unwrap() error {
	return e.Err
}

// NewSymbolError creates a new SymbolError.
func NewSymbolError(id, name string, err error) *SymbolError {
	return &SymbolError{SymbolID: id, SymbolName: name, Err: err}
}

// ProviderError represents an error from a provider.
type ProviderError struct {
	Provider string
	Op       string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s provider: %s failed: %v", e.Provider, e.Op, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider, op string, err error) *ProviderError {
	return &ProviderError{Provider: provider, Op: op, Err: err}
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s (got: %v)", e.Field, e.Message, e.Value)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Message: message}
}

// MultiError collects multiple errors.
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors occurred: %v", len(e.Errors), e.Errors)
}

// Add adds an error to the collection.
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if any errors were collected.
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ErrorOrNil returns nil if no errors, otherwise returns the MultiError.
func (e *MultiError) ErrorOrNil() error {
	if !e.HasErrors() {
		return nil
	}
	return e
}

// NewMultiError creates a new MultiError.
func NewMultiError() *MultiError {
	return &MultiError{Errors: make([]error, 0)}
}

// IsNotFound returns true if the error is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsProviderNotAvailable returns true if the error is or wraps ErrProviderNotAvailable.
func IsProviderNotAvailable(err error) bool {
	return errors.Is(err, ErrProviderNotAvailable)
}

// IsInvalidConfig returns true if the error is or wraps ErrInvalidConfig.
func IsInvalidConfig(err error) bool {
	return errors.Is(err, ErrInvalidConfig)
}
