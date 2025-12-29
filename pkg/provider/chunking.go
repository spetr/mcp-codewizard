package provider

import (
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// ChunkingStrategy splits source files into chunks.
type ChunkingStrategy interface {
	// Name returns the strategy name (e.g., "treesitter", "simple").
	Name() string

	// Chunk splits a source file into chunks.
	Chunk(file *types.SourceFile) ([]*types.Chunk, error)

	// SupportedLanguages returns languages this strategy supports.
	// Empty slice means all languages (for simple chunking).
	SupportedLanguages() []string

	// SupportsLanguage checks if a language is supported.
	SupportsLanguage(lang string) bool

	// ExtractSymbols extracts symbols from a source file.
	// May return nil if not supported (e.g., simple chunking).
	ExtractSymbols(file *types.SourceFile) ([]*types.Symbol, error)

	// ExtractReferences extracts references from a source file.
	// May return nil if not supported.
	ExtractReferences(file *types.SourceFile) ([]*types.Reference, error)

	// Close releases any resources.
	Close() error
}

// ChunkingConfig contains configuration for chunking strategies.
type ChunkingConfig struct {
	Strategy     string // "treesitter", "simple"
	MaxChunkSize int    // Max tokens per chunk
}

// LanguageDetector detects the programming language of a file.
type LanguageDetector interface {
	// DetectLanguage returns the language for a file path.
	// Returns empty string if unknown.
	DetectLanguage(filePath string) string

	// DetectFromContent tries to detect language from content.
	DetectFromContent(content []byte) string
}
