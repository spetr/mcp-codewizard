// Package shared defines shared interfaces and types for external plugins.
package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Handshake is a common handshake that is shared by plugin and host.
// Prevents plugins compiled with different versions from running.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "MCP_CODEINDEX_PLUGIN",
	MagicCookieValue: "mcp-codewizard-v1",
}

// PluginType identifies the type of plugin.
type PluginType string

const (
	PluginTypeEmbedding PluginType = "embedding"
	PluginTypeChunking  PluginType = "chunking"
	PluginTypeReranker  PluginType = "reranker"
)

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	string(PluginTypeEmbedding): &EmbeddingPlugin{},
	string(PluginTypeReranker):  &RerankerPlugin{},
}

// EmbeddingProvider is the interface that embedding plugins must implement.
// This mirrors pkg/provider.EmbeddingProvider but is self-contained for plugins.
type EmbeddingProvider interface {
	Name() string
	Embed(texts []string) ([][]float32, error)
	Dimensions() int
	MaxBatchSize() int
	Warmup() error
	Close() error
}

// RerankerProvider is the interface that reranker plugins must implement.
type RerankerProvider interface {
	Name() string
	Rerank(query string, documents []string) ([]RerankResult, error)
	MaxDocuments() int
	Warmup() error
	Close() error
}

// RerankResult represents a reranking result.
type RerankResult struct {
	Index int
	Score float32
}

// EmbeddingPlugin is the plugin.Plugin implementation for embedding providers.
type EmbeddingPlugin struct {
	Impl EmbeddingProvider
}

func (p *EmbeddingPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &EmbeddingRPCServer{Impl: p.Impl}, nil
}

func (p *EmbeddingPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &EmbeddingRPCClient{client: c}, nil
}

// RerankerPlugin is the plugin.Plugin implementation for reranker providers.
type RerankerPlugin struct {
	Impl RerankerProvider
}

func (p *RerankerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RerankerRPCServer{Impl: p.Impl}, nil
}

func (p *RerankerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RerankerRPCClient{client: c}, nil
}
