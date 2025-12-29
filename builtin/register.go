// Package builtin registers all built-in providers with the default registry.
package builtin

import (
	ollamaEmbed "github.com/spetr/mcp-codewizard/builtin/embedding/ollama"
	openaiEmbed "github.com/spetr/mcp-codewizard/builtin/embedding/openai"
	ollamaRerank "github.com/spetr/mcp-codewizard/builtin/reranker/ollama"
	"github.com/spetr/mcp-codewizard/builtin/reranker/none"
	simpleChunker "github.com/spetr/mcp-codewizard/builtin/chunking/simple"
	tsChunker "github.com/spetr/mcp-codewizard/builtin/chunking/treesitter"
	"github.com/spetr/mcp-codewizard/builtin/vectorstore/sqlitevec"
	"github.com/spetr/mcp-codewizard/pkg/provider"
)

func init() {
	// Register embedding providers
	provider.RegisterEmbedding("ollama", func(cfg provider.EmbeddingConfig) (provider.EmbeddingProvider, error) {
		return ollamaEmbed.New(ollamaEmbed.Config{
			Endpoint:  cfg.Endpoint,
			Model:     cfg.Model,
			BatchSize: cfg.BatchSize,
		}), nil
	})

	provider.RegisterEmbedding("openai", func(cfg provider.EmbeddingConfig) (provider.EmbeddingProvider, error) {
		return openaiEmbed.New(openaiEmbed.Config{
			APIKey:    cfg.APIKey,
			Model:     cfg.Model,
			BatchSize: cfg.BatchSize,
		}), nil
	})

	// Register rerankers
	provider.RegisterReranker("ollama", func(cfg provider.RerankerConfig) (provider.Reranker, error) {
		return ollamaRerank.New(ollamaRerank.Config{
			Endpoint:   cfg.Endpoint,
			Model:      cfg.Model,
			MaxDocs:    cfg.Candidates,
			Candidates: cfg.Candidates,
		}), nil
	})

	provider.RegisterReranker("none", func(cfg provider.RerankerConfig) (provider.Reranker, error) {
		return none.New(), nil
	})

	// Register chunking strategies
	provider.RegisterChunking("treesitter", func(cfg provider.ChunkingConfig) (provider.ChunkingStrategy, error) {
		return tsChunker.New(tsChunker.Config{
			MaxChunkSize: cfg.MaxChunkSize,
		}), nil
	})

	provider.RegisterChunking("simple", func(cfg provider.ChunkingConfig) (provider.ChunkingStrategy, error) {
		return simpleChunker.New(simpleChunker.Config{
			MaxChunkSize: cfg.MaxChunkSize,
		}), nil
	})

	// Register vector stores
	provider.RegisterVectorStore("sqlitevec", func() (provider.VectorStore, error) {
		return sqlitevec.New(), nil
	})
}
