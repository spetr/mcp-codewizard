package host

import (
	"context"

	"github.com/spetr/mcp-codewizard/pkg/plugin/shared"
	"github.com/spetr/mcp-codewizard/pkg/provider"
)

// EmbeddingAdapter adapts a plugin EmbeddingProvider to the provider.EmbeddingProvider interface.
type EmbeddingAdapter struct {
	plugin shared.EmbeddingProvider
}

// NewEmbeddingAdapter creates a new embedding adapter.
func NewEmbeddingAdapter(p shared.EmbeddingProvider) *EmbeddingAdapter {
	return &EmbeddingAdapter{plugin: p}
}

// Name returns the provider name.
func (a *EmbeddingAdapter) Name() string {
	return a.plugin.Name()
}

// Embed generates embeddings for the given texts.
func (a *EmbeddingAdapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Check context before calling plugin
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return a.plugin.Embed(texts)
}

// Dimensions returns the embedding dimensions.
func (a *EmbeddingAdapter) Dimensions() int {
	return a.plugin.Dimensions()
}

// MaxBatchSize returns the maximum batch size.
func (a *EmbeddingAdapter) MaxBatchSize() int {
	return a.plugin.MaxBatchSize()
}

// MaxTokens returns the maximum context window size in tokens.
// For plugins, we return a conservative default since we can't query the plugin.
func (a *EmbeddingAdapter) MaxTokens() int {
	return 2048 // Conservative default for plugins
}

// Warmup warms up the provider.
func (a *EmbeddingAdapter) Warmup(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return a.plugin.Warmup()
}

// Close closes the provider.
func (a *EmbeddingAdapter) Close() error {
	return a.plugin.Close()
}

// Available checks if the plugin is available.
func (a *EmbeddingAdapter) Available(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return a.plugin.Warmup()
}

// Ensure EmbeddingAdapter implements provider.EmbeddingProvider
var _ provider.EmbeddingProvider = (*EmbeddingAdapter)(nil)

// RerankerAdapter adapts a plugin RerankerProvider to the provider.Reranker interface.
type RerankerAdapter struct {
	plugin shared.RerankerProvider
}

// NewRerankerAdapter creates a new reranker adapter.
func NewRerankerAdapter(p shared.RerankerProvider) *RerankerAdapter {
	return &RerankerAdapter{plugin: p}
}

// Name returns the provider name.
func (a *RerankerAdapter) Name() string {
	return a.plugin.Name()
}

// Rerank reranks the documents for the given query.
func (a *RerankerAdapter) Rerank(ctx context.Context, query string, documents []string) ([]provider.RerankResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	results, err := a.plugin.Rerank(query, documents)
	if err != nil {
		return nil, err
	}

	// Convert shared.RerankResult to provider.RerankResult
	converted := make([]provider.RerankResult, len(results))
	for i, r := range results {
		converted[i] = provider.RerankResult{
			Index: r.Index,
			Score: r.Score,
		}
	}
	return converted, nil
}

// MaxDocuments returns the maximum number of documents.
func (a *RerankerAdapter) MaxDocuments() int {
	return a.plugin.MaxDocuments()
}

// Warmup warms up the provider.
func (a *RerankerAdapter) Warmup(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return a.plugin.Warmup()
}

// Close closes the provider.
func (a *RerankerAdapter) Close() error {
	return a.plugin.Close()
}

// Ensure RerankerAdapter implements provider.Reranker
var _ provider.Reranker = (*RerankerAdapter)(nil)
