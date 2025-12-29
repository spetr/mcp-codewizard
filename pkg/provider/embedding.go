// Package provider defines interfaces for pluggable components.
package provider

import (
	"context"
)

// EmbeddingProvider generates vector embeddings from text.
type EmbeddingProvider interface {
	// Name returns the provider name (e.g., "ollama", "openai").
	Name() string

	// Embed generates embeddings for the given texts.
	// Returns a slice of embeddings, one for each input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding dimension size.
	Dimensions() int

	// MaxBatchSize returns the maximum number of texts per batch.
	MaxBatchSize() int

	// Warmup pre-loads the model (optional, for Ollama).
	Warmup(ctx context.Context) error

	// Close releases any resources.
	Close() error
}

// EmbeddingConfig contains configuration for embedding providers.
type EmbeddingConfig struct {
	Provider  string // "ollama", "openai", "voyage", "jina"
	Model     string // Model name
	Endpoint  string // API endpoint (for Ollama)
	APIKey    string // API key (for OpenAI, Voyage, Jina)
	BatchSize int    // Documents per batch
}
