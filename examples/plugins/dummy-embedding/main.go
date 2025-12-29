// Package main implements a dummy embedding plugin for testing.
// This plugin generates deterministic embeddings based on text hashing.
package main

import (
	"crypto/sha256"
	"math"

	"github.com/hashicorp/go-plugin"

	"github.com/spetr/mcp-codewizard/pkg/plugin/shared"
)

const (
	dimensions = 384
	batchSize  = 32
)

// DummyEmbedding is a dummy embedding provider for testing.
type DummyEmbedding struct{}

// Name returns the provider name.
func (d *DummyEmbedding) Name() string {
	return "dummy-embedding"
}

// Embed generates deterministic embeddings based on text hashing.
// This is useful for testing the plugin system without requiring an actual model.
func (d *DummyEmbedding) Embed(texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		embeddings[i] = d.hashToEmbedding(text)
	}

	return embeddings, nil
}

// hashToEmbedding converts text to a deterministic embedding using SHA256.
func (d *DummyEmbedding) hashToEmbedding(text string) []float32 {
	// Create multiple hashes to fill the embedding dimensions
	embedding := make([]float32, dimensions)

	hash := sha256.Sum256([]byte(text))

	for i := 0; i < dimensions; i++ {
		// Use hash bytes cyclically
		byteIdx := i % 32

		// If we need more variation, rehash with the index
		if i >= 32 {
			combined := append(hash[:], byte(i/32))
			hash = sha256.Sum256(combined)
		}

		// Convert byte to float32 in range [-1, 1]
		embedding[i] = (float32(hash[byteIdx]) / 127.5) - 1.0
	}

	// Normalize the embedding
	return normalize(embedding)
}

// normalize normalizes a vector to unit length.
func normalize(vec []float32) []float32 {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}

	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return vec
	}

	for i := range vec {
		vec[i] /= norm
	}

	return vec
}

// Dimensions returns the embedding dimensions.
func (d *DummyEmbedding) Dimensions() int {
	return dimensions
}

// MaxBatchSize returns the maximum batch size.
func (d *DummyEmbedding) MaxBatchSize() int {
	return batchSize
}

// Warmup warms up the provider (no-op for dummy).
func (d *DummyEmbedding) Warmup() error {
	return nil
}

// Close closes the provider (no-op for dummy).
func (d *DummyEmbedding) Close() error {
	return nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			string(shared.PluginTypeEmbedding): &shared.EmbeddingPlugin{
				Impl: &DummyEmbedding{},
			},
		},
	})
}

// Compile with: go build -o dummy-embedding ./examples/plugins/dummy-embedding
