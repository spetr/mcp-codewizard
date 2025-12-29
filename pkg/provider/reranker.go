package provider

import (
	"context"
)

// RerankResult represents a single reranking result.
type RerankResult struct {
	Index int     // Original position in input slice
	Score float32 // Relevance score (higher = more relevant)
}

// Reranker re-ranks search results for better precision.
type Reranker interface {
	// Name returns the reranker name (e.g., "ollama", "none").
	Name() string

	// Rerank re-orders documents by relevance to the query.
	// Returns results sorted by score (highest first).
	Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)

	// MaxDocuments returns the maximum number of documents for reranking.
	MaxDocuments() int

	// Warmup pre-loads the model (optional, for Ollama).
	Warmup(ctx context.Context) error

	// Close releases any resources.
	Close() error
}

// RerankerConfig contains configuration for reranker providers.
type RerankerConfig struct {
	Enabled    bool   // Whether reranking is enabled
	Provider   string // "ollama", "none"
	Model      string // Model name (e.g., "qwen3-reranker")
	Endpoint   string // API endpoint
	Candidates int    // How many candidates to rerank
}
