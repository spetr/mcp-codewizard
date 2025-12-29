package provider

import (
	"fmt"
	"sync"
)

// EmbeddingFactory creates an EmbeddingProvider from configuration.
type EmbeddingFactory func(config EmbeddingConfig) (EmbeddingProvider, error)

// RerankerFactory creates a Reranker from configuration.
type RerankerFactory func(config RerankerConfig) (Reranker, error)

// ChunkingFactory creates a ChunkingStrategy from configuration.
type ChunkingFactory func(config ChunkingConfig) (ChunkingStrategy, error)

// VectorStoreFactory creates a VectorStore.
type VectorStoreFactory func() (VectorStore, error)

// Registry holds factories for all provider types.
type Registry struct {
	mu sync.RWMutex

	embeddingFactories   map[string]EmbeddingFactory
	rerankerFactories    map[string]RerankerFactory
	chunkingFactories    map[string]ChunkingFactory
	vectorStoreFactories map[string]VectorStoreFactory
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		embeddingFactories:   make(map[string]EmbeddingFactory),
		rerankerFactories:    make(map[string]RerankerFactory),
		chunkingFactories:    make(map[string]ChunkingFactory),
		vectorStoreFactories: make(map[string]VectorStoreFactory),
	}
}

// RegisterEmbedding registers an embedding provider factory.
func (r *Registry) RegisterEmbedding(name string, factory EmbeddingFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.embeddingFactories[name] = factory
}

// RegisterReranker registers a reranker factory.
func (r *Registry) RegisterReranker(name string, factory RerankerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rerankerFactories[name] = factory
}

// RegisterChunking registers a chunking strategy factory.
func (r *Registry) RegisterChunking(name string, factory ChunkingFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunkingFactories[name] = factory
}

// RegisterVectorStore registers a vector store factory.
func (r *Registry) RegisterVectorStore(name string, factory VectorStoreFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vectorStoreFactories[name] = factory
}

// CreateEmbedding creates an embedding provider by name.
func (r *Registry) CreateEmbedding(name string, config EmbeddingConfig) (EmbeddingProvider, error) {
	r.mu.RLock()
	factory, ok := r.embeddingFactories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown embedding provider: %s (available: %v)", name, r.ListEmbeddings())
	}
	return factory(config)
}

// CreateReranker creates a reranker by name.
func (r *Registry) CreateReranker(name string, config RerankerConfig) (Reranker, error) {
	r.mu.RLock()
	factory, ok := r.rerankerFactories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown reranker provider: %s (available: %v)", name, r.ListRerankers())
	}
	return factory(config)
}

// CreateChunking creates a chunking strategy by name.
func (r *Registry) CreateChunking(name string, config ChunkingConfig) (ChunkingStrategy, error) {
	r.mu.RLock()
	factory, ok := r.chunkingFactories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown chunking strategy: %s (available: %v)", name, r.ListChunkings())
	}
	return factory(config)
}

// CreateVectorStore creates a vector store by name.
func (r *Registry) CreateVectorStore(name string) (VectorStore, error) {
	r.mu.RLock()
	factory, ok := r.vectorStoreFactories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown vector store: %s (available: %v)", name, r.ListVectorStores())
	}
	return factory()
}

// ListEmbeddings returns all registered embedding provider names.
func (r *Registry) ListEmbeddings() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.embeddingFactories))
	for name := range r.embeddingFactories {
		names = append(names, name)
	}
	return names
}

// ListRerankers returns all registered reranker names.
func (r *Registry) ListRerankers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.rerankerFactories))
	for name := range r.rerankerFactories {
		names = append(names, name)
	}
	return names
}

// ListChunkings returns all registered chunking strategy names.
func (r *Registry) ListChunkings() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.chunkingFactories))
	for name := range r.chunkingFactories {
		names = append(names, name)
	}
	return names
}

// ListVectorStores returns all registered vector store names.
func (r *Registry) ListVectorStores() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.vectorStoreFactories))
	for name := range r.vectorStoreFactories {
		names = append(names, name)
	}
	return names
}

// HasEmbedding checks if an embedding provider is registered.
func (r *Registry) HasEmbedding(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.embeddingFactories[name]
	return ok
}

// HasReranker checks if a reranker is registered.
func (r *Registry) HasReranker(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.rerankerFactories[name]
	return ok
}

// HasChunking checks if a chunking strategy is registered.
func (r *Registry) HasChunking(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.chunkingFactories[name]
	return ok
}

// HasVectorStore checks if a vector store is registered.
func (r *Registry) HasVectorStore(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.vectorStoreFactories[name]
	return ok
}

// DefaultRegistry is the global default registry.
var DefaultRegistry = NewRegistry()

// Register functions for the default registry.

// RegisterEmbedding registers an embedding provider in the default registry.
func RegisterEmbedding(name string, factory EmbeddingFactory) {
	DefaultRegistry.RegisterEmbedding(name, factory)
}

// RegisterReranker registers a reranker in the default registry.
func RegisterReranker(name string, factory RerankerFactory) {
	DefaultRegistry.RegisterReranker(name, factory)
}

// RegisterChunking registers a chunking strategy in the default registry.
func RegisterChunking(name string, factory ChunkingFactory) {
	DefaultRegistry.RegisterChunking(name, factory)
}

// RegisterVectorStore registers a vector store in the default registry.
func RegisterVectorStore(name string, factory VectorStoreFactory) {
	DefaultRegistry.RegisterVectorStore(name, factory)
}

