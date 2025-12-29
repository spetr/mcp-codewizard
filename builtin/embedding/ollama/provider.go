// Package ollama implements EmbeddingProvider using Ollama's API.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/provider"
)

// Default values
const (
	DefaultModel       = "nomic-embed-code"
	DefaultEndpoint    = "http://localhost:11434"
	DefaultBatchSize   = 32
	DefaultDimensions  = 768  // nomic-embed-code default
	DefaultMaxTokens   = 8000 // Safe limit for most embedding models (chars, roughly ~2000 tokens)
)

// Config contains Ollama provider configuration.
type Config struct {
	Model      string
	Endpoint   string
	BatchSize  int
	Dimensions int // Set to 0 to auto-detect from first embedding
}

// Provider implements the EmbeddingProvider interface for Ollama.
type Provider struct {
	config     Config
	client     *http.Client
	dimensions int
	mu         sync.RWMutex
}

// New creates a new Ollama embedding provider.
func New(cfg Config) *Provider {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = DefaultBatchSize
	}

	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		dimensions: cfg.Dimensions,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Embed generates embeddings for the given texts.
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, len(texts))

	// Process in batches
	for i := 0; i < len(texts); i += p.config.BatchSize {
		end := i + p.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		// Embed each text in batch (Ollama doesn't support batch embedding in a single call)
		for j, text := range batch {
			embedding, err := p.embedSingle(ctx, text)
			if err != nil {
				return nil, fmt.Errorf("failed to embed text %d: %w", i+j, err)
			}
			results[i+j] = embedding

			// Auto-detect dimensions from first embedding
			if p.dimensions == 0 && len(embedding) > 0 {
				p.mu.Lock()
				p.dimensions = len(embedding)
				p.mu.Unlock()
			}
		}
	}

	return results, nil
}

// embedSingle embeds a single text.
func (p *Provider) embedSingle(ctx context.Context, text string) ([]float32, error) {
	// Truncate text if too long to avoid context length errors
	if len(text) > DefaultMaxTokens {
		text = text[:DefaultMaxTokens]
	}

	// Ollama embed API request
	reqBody := map[string]any{
		"model":  p.config.Model,
		"prompt": text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// Dimensions returns the embedding dimensions.
func (p *Provider) Dimensions() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.dimensions > 0 {
		return p.dimensions
	}
	return DefaultDimensions
}

// MaxBatchSize returns the maximum batch size.
func (p *Provider) MaxBatchSize() int {
	return p.config.BatchSize
}

// Warmup pre-loads the model into Ollama's memory.
func (p *Provider) Warmup(ctx context.Context) error {
	// Send a dummy embedding request to load the model
	_, err := p.embedSingle(ctx, "warmup")
	return err
}

// Close releases resources.
func (p *Provider) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// Available checks if Ollama is running and the model is available.
func (p *Provider) Available(ctx context.Context) error {
	// Check Ollama is running
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoint+"/api/version", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not available at %s: %w", p.config.Endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	// Check model is available
	return p.checkModel(ctx)
}

// checkModel verifies the model exists.
func (p *Provider) checkModel(ctx context.Context) error {
	reqBody := map[string]any{
		"name": p.config.Model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/show", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("model %s not found, run: ollama pull %s", p.config.Model, p.config.Model)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama show failed: %s", string(body))
	}

	return nil
}

// ListModels returns available embedding models.
func (p *Provider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			Size       int64  `json:"size"`
			ModifiedAt string `json:"modified_at"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []ModelInfo
	for _, m := range result.Models {
		modelType := detectModelType(m.Name)
		models = append(models, ModelInfo{
			Name: m.Name,
			Size: formatSize(m.Size),
			Type: modelType,
		})
	}

	return models, nil
}

// ModelInfo contains information about an Ollama model.
type ModelInfo struct {
	Name        string `json:"name"`
	Size        string `json:"size"`
	Type        string `json:"type"` // "embedding", "reranker", "llm"
	Recommended bool   `json:"recommended"`
}

// detectModelType guesses the model type from its name.
func detectModelType(name string) string {
	// Known embedding models
	embeddingModels := []string{
		"nomic-embed", "mxbai-embed", "bge-", "e5-", "gte-",
		"all-minilm", "snowflake-arctic-embed",
	}
	for _, em := range embeddingModels {
		if containsIgnoreCase(name, em) {
			return "embedding"
		}
	}

	// Known reranker models
	rerankerModels := []string{
		"rerank", "ranker",
	}
	for _, rm := range rerankerModels {
		if containsIgnoreCase(name, rm) {
			return "reranker"
		}
	}

	return "llm"
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && containsIgnoreCase(s[1:], substr) ||
			(len(s) >= len(substr) && equalFoldPrefix(s, substr)))
}

func equalFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if toLower(s[i]) != toLower(prefix[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// Ensure Provider implements EmbeddingProvider interface
var _ provider.EmbeddingProvider = (*Provider)(nil)
