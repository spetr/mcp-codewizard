package provider

// VectorStore stores and searches vector embeddings.
// It composes all the smaller store interfaces for backwards compatibility.
// New code should depend on the smaller interfaces (ChunkStore, SymbolStore, etc.)
// rather than VectorStore when possible.
type VectorStore interface {
	Store
	ChunkStore
	Searcher
	SymbolStore
	ReferenceStore
	MetadataStore
	FileCache
}

// VectorStoreWithHistory extends VectorStore with git history capabilities.
// Use this interface when you need temporal search across git history.
type VectorStoreWithHistory interface {
	VectorStore
	GitHistoryStore
}

// VectorStoreConfig contains configuration for vector stores.
type VectorStoreConfig struct {
	Provider string // "sqlitevec"
	Path     string // Path to database file
}
