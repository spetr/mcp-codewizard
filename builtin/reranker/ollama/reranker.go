// Package ollama implements Reranker using Ollama with Qwen3-Reranker or similar models.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/spetr/mcp-codewizard/pkg/provider"
)

// Default values
const (
	DefaultModel       = "qwen3-reranker"
	DefaultEndpoint    = "http://localhost:11434"
	DefaultMaxDocs     = 100
	DefaultCandidates  = 100
)

// Config contains Ollama reranker configuration.
type Config struct {
	Model       string
	Endpoint    string
	MaxDocs     int // Maximum documents per rerank call
	Candidates  int // Number of candidates to consider
}

// Reranker implements the Reranker interface using Ollama.
type Reranker struct {
	config Config
	client *http.Client
}

// New creates a new Ollama reranker.
func New(cfg Config) *Reranker {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.MaxDocs == 0 {
		cfg.MaxDocs = DefaultMaxDocs
	}
	if cfg.Candidates == 0 {
		cfg.Candidates = DefaultCandidates
	}

	return &Reranker{
		config: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second, // Reranking can be slow
		},
	}
}

// Name returns the reranker name.
func (r *Reranker) Name() string {
	return "ollama"
}

// Rerank reranks documents by relevance to the query.
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string) ([]provider.RerankResult, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	// Limit documents
	docs := documents
	if len(docs) > r.config.MaxDocs {
		docs = docs[:r.config.MaxDocs]
	}

	// Build prompt for reranking
	// Qwen3-Reranker expects a specific format
	prompt := buildRerankPrompt(query, docs)

	// Call Ollama generate API
	reqBody := map[string]any{
		"model":  r.config.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": 0,
			"num_predict": len(docs) * 20, // Enough tokens for scores
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse scores from response
	scores := parseRerankResponse(result.Response, len(docs))

	// Create results
	results := make([]provider.RerankResult, len(scores))
	for i, score := range scores {
		results[i] = provider.RerankResult{
			Index: i,
			Score: score,
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// buildRerankPrompt creates the prompt for reranking.
func buildRerankPrompt(query string, documents []string) string {
	// Format for Qwen3-Reranker style models
	var buf bytes.Buffer

	buf.WriteString("Rate the relevance of each document to the query on a scale of 0.0 to 1.0.\n")
	buf.WriteString("Output only the scores in order, one per line, as decimal numbers.\n\n")
	buf.WriteString(fmt.Sprintf("Query: %s\n\n", query))
	buf.WriteString("Documents:\n")

	for i, doc := range documents {
		// Truncate very long documents
		content := doc
		if len(content) > 1000 {
			content = content[:1000] + "..."
		}
		buf.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, content))
	}

	buf.WriteString("Relevance scores (0.0 to 1.0, one per line):\n")

	return buf.String()
}

// parseRerankResponse extracts scores from the model response.
func parseRerankResponse(response string, numDocs int) []float32 {
	scores := make([]float32, numDocs)

	// Default scores (in case parsing fails)
	for i := range scores {
		scores[i] = 0.5
	}

	// Try to parse float numbers from response
	// Pattern: matches decimal numbers like 0.85, .9, 1.0
	pattern := regexp.MustCompile(`\b(0?\.\d+|1\.0|0|1)\b`)
	matches := pattern.FindAllString(response, numDocs)

	for i, match := range matches {
		if i >= numDocs {
			break
		}
		if score, err := strconv.ParseFloat(match, 32); err == nil {
			if score >= 0 && score <= 1 {
				scores[i] = float32(score)
			}
		}
	}

	return scores
}

// MaxDocuments returns the maximum documents for reranking.
func (r *Reranker) MaxDocuments() int {
	return r.config.MaxDocs
}

// Warmup pre-loads the model into Ollama's memory.
func (r *Reranker) Warmup(ctx context.Context) error {
	// Send a minimal rerank request to load the model
	_, err := r.Rerank(ctx, "test", []string{"test document"})
	return err
}

// Close releases resources.
func (r *Reranker) Close() error {
	return nil
}

// Available checks if the reranker model is available.
func (r *Reranker) Available(ctx context.Context) error {
	reqBody := map[string]any{
		"name": r.config.Model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint+"/api/show", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not available at %s: %w", r.config.Endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("model %s not found, run: ollama pull %s", r.config.Model, r.config.Model)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama show failed: %s", string(body))
	}

	return nil
}

// Ensure Reranker implements the Reranker interface
var _ provider.Reranker = (*Reranker)(nil)
