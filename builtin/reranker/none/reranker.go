// Package none implements a no-op Reranker that passes through results unchanged.
package none

import (
	"context"

	"github.com/spetr/mcp-codewizard/pkg/provider"
)

// Reranker is a passthrough reranker that doesn't modify scores.
type Reranker struct{}

// New creates a new no-op reranker.
func New() *Reranker {
	return &Reranker{}
}

// Name returns the reranker name.
func (r *Reranker) Name() string {
	return "none"
}

// Rerank returns documents with original ordering.
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string) ([]provider.RerankResult, error) {
	results := make([]provider.RerankResult, len(documents))
	for i := range documents {
		// Assign decreasing scores to maintain original order
		results[i] = provider.RerankResult{
			Index: i,
			Score: float32(len(documents)-i) / float32(len(documents)),
		}
	}
	return results, nil
}

// MaxDocuments returns unlimited (large number).
func (r *Reranker) MaxDocuments() int {
	return 10000
}

// Warmup does nothing.
func (r *Reranker) Warmup(ctx context.Context) error {
	return nil
}

// Close does nothing.
func (r *Reranker) Close() error {
	return nil
}

// Ensure Reranker implements the Reranker interface
var _ provider.Reranker = (*Reranker)(nil)
