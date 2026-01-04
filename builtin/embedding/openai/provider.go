// Package openai implements EmbeddingProvider using OpenAI's API.
package openai

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sashabaranov/go-openai"

	"github.com/spetr/mcp-codewizard/pkg/provider"
)

// Default values
const (
	DefaultModel      = openai.AdaEmbeddingV2
	DefaultBatchSize  = 100 // OpenAI supports up to 2048 inputs per request
	DefaultDimensions = 1536 // text-embedding-ada-002 default
)

// Model dimensions for known models
var modelDimensions = map[string]int{
	"text-embedding-ada-002": 1536,
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
}

// Model context windows (max tokens) for known models
var modelMaxTokens = map[string]int{
	// OpenAI models
	"text-embedding-ada-002": 8191,
	"text-embedding-3-small": 8191,
	"text-embedding-3-large": 8191,
	// Nomic models (often used via OpenAI-compatible APIs)
	"nomic-embed-text":       8192,
	"nomic-embed-text-v1":    8192,
	"nomic-embed-text-v1.5":  8192,
	"nomic-embed-code":       2048, // Conservative default for code model
	"nomic-embed-code-7b":    2048, // LiteLLM reports 2048 for this
	// Voyage models
	"voyage-2":      4096,
	"voyage-code-2": 16000,
	"voyage-large-2": 16000,
	// Jina models
	"jina-embeddings-v2-base-en": 8192,
	"jina-embeddings-v2-base-code": 8192,
}

// Config contains OpenAI provider configuration.
type Config struct {
	Model      string
	APIKey     string // If empty, uses OPENAI_API_KEY env var
	BaseURL    string // Optional: custom API endpoint (for Azure, etc.)
	BatchSize  int
	Dimensions int // Set to 0 to use default for model
}

// Provider implements the EmbeddingProvider interface for OpenAI.
type Provider struct {
	config     Config
	client     *openai.Client
	dimensions int
	mu         sync.RWMutex
}

// New creates a new OpenAI embedding provider.
func New(cfg Config) *Provider {
	if cfg.Model == "" {
		cfg.Model = string(DefaultModel)
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = DefaultBatchSize
	}

	// Get API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Create OpenAI client
	clientConfig := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	// Get dimensions for known models
	dimensions := cfg.Dimensions
	if dimensions == 0 {
		if d, ok := modelDimensions[cfg.Model]; ok {
			dimensions = d
		} else {
			dimensions = DefaultDimensions
		}
	}

	return &Provider{
		config:     cfg,
		client:     client,
		dimensions: dimensions,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai"
}

// Embed generates embeddings for the given texts.
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, len(texts))

	// Process in batches
	for i := 0; i < len(texts); i += p.config.BatchSize {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		end := i + p.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		// Create embedding request
		req := openai.EmbeddingRequest{
			Input: batch,
			Model: openai.EmbeddingModel(p.config.Model),
		}

		resp, err := p.client.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("openai embedding failed: %w", err)
		}

		// Extract embeddings from response
		for j, data := range resp.Data {
			results[i+j] = data.Embedding
		}

		// Update dimensions from response if needed
		if len(resp.Data) > 0 && p.dimensions == 0 {
			p.mu.Lock()
			p.dimensions = len(resp.Data[0].Embedding)
			p.mu.Unlock()
		}
	}

	return results, nil
}

// Dimensions returns the embedding dimensions.
func (p *Provider) Dimensions() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dimensions
}

// MaxTokens returns the maximum context window size in tokens.
func (p *Provider) MaxTokens() int {
	if maxTokens, ok := modelMaxTokens[p.config.Model]; ok {
		return maxTokens
	}
	// Default for unknown models - conservative estimate
	return 2048
}

// MaxBatchSize returns the maximum batch size.
func (p *Provider) MaxBatchSize() int {
	return p.config.BatchSize
}

// Warmup tests the API connection.
func (p *Provider) Warmup(ctx context.Context) error {
	// Send a test embedding request
	_, err := p.Embed(ctx, []string{"test"})
	return err
}

// Close releases resources.
func (p *Provider) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// Available checks if OpenAI API is accessible.
func (p *Provider) Available(ctx context.Context) error {
	// Check if API key is set
	if p.config.APIKey == "" && os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Try a minimal embedding request to verify API access
	req := openai.EmbeddingRequest{
		Input: []string{"test"},
		Model: openai.EmbeddingModel(p.config.Model),
	}

	_, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return fmt.Errorf("openai API not accessible: %w", err)
	}

	return nil
}

// Ensure Provider implements EmbeddingProvider interface
var _ provider.EmbeddingProvider = (*Provider)(nil)
