// Package simple implements a simple line-based chunking strategy.
// This is used as a fallback when TreeSitter is not available or
// doesn't support the language.
package simple

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Default values
const (
	DefaultMaxChunkSize = 2000 // tokens (approximated as chars/4)
	DefaultMinChunkSize = 100  // minimum chars to create a chunk
	CharsPerToken       = 4    // rough approximation
)

// Config contains configuration for simple chunking.
type Config struct {
	MaxChunkSize int // Maximum chunk size in tokens
	MinChunkSize int // Minimum chunk size in chars
}

// Chunker implements a simple line-based chunking strategy.
type Chunker struct {
	config Config
}

// New creates a new simple chunker.
func New(cfg Config) *Chunker {
	if cfg.MaxChunkSize == 0 {
		cfg.MaxChunkSize = DefaultMaxChunkSize
	}
	if cfg.MinChunkSize == 0 {
		cfg.MinChunkSize = DefaultMinChunkSize
	}
	return &Chunker{config: cfg}
}

// Name returns the strategy name.
func (c *Chunker) Name() string {
	return "simple"
}

// Chunk splits a file into chunks based on blank lines and size limits.
func (c *Chunker) Chunk(file *types.SourceFile) ([]*types.Chunk, error) {
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	maxChars := c.config.MaxChunkSize * CharsPerToken

	var chunks []*types.Chunk
	var currentLines []string
	var currentChars int
	startLine := 1

	flushChunk := func(endLine int) {
		if len(currentLines) == 0 || currentChars < c.config.MinChunkSize {
			return
		}

		chunkContent := strings.Join(currentLines, "\n")
		hash := sha256.Sum256([]byte(chunkContent))
		hashStr := hex.EncodeToString(hash[:8])

		chunks = append(chunks, &types.Chunk{
			ID:        fmt.Sprintf("%s:%d:%s", file.Path, startLine, hashStr),
			FilePath:  file.Path,
			Language:  file.Language,
			Content:   chunkContent,
			ChunkType: types.ChunkTypeBlock,
			StartLine: startLine,
			EndLine:   endLine,
			Hash:      hex.EncodeToString(hash[:]),
		})
	}

	for i, line := range lines {
		lineNum := i + 1
		lineLen := len(line)

		// Check if we should start a new chunk
		shouldSplit := false

		// Split on blank lines (paragraph boundaries)
		if strings.TrimSpace(line) == "" && currentChars > c.config.MinChunkSize {
			shouldSplit = true
		}

		// Split if chunk is getting too large
		if currentChars+lineLen > maxChars && currentChars > 0 {
			shouldSplit = true
		}

		// Split on function/class definitions (heuristic)
		if currentChars > c.config.MinChunkSize && looksLikeDefinition(line, file.Language) {
			shouldSplit = true
		}

		if shouldSplit {
			flushChunk(lineNum - 1)
			currentLines = nil
			currentChars = 0
			startLine = lineNum
		}

		currentLines = append(currentLines, line)
		currentChars += lineLen + 1 // +1 for newline
	}

	// Flush remaining content
	flushChunk(len(lines))

	// If no chunks were created (small file), create one for the whole file
	if len(chunks) == 0 && len(content) > 0 {
		hash := sha256.Sum256([]byte(content))
		chunks = append(chunks, &types.Chunk{
			ID:        fmt.Sprintf("%s:1:%s", file.Path, hex.EncodeToString(hash[:8])),
			FilePath:  file.Path,
			Language:  file.Language,
			Content:   content,
			ChunkType: types.ChunkTypeFile,
			StartLine: 1,
			EndLine:   len(lines),
			Hash:      hex.EncodeToString(hash[:]),
		})
	}

	return chunks, nil
}

// looksLikeDefinition checks if a line looks like a function/class definition.
func looksLikeDefinition(line, language string) bool {
	trimmed := strings.TrimSpace(line)

	switch language {
	case "go":
		return strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ")
	case "python":
		return strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "async def ")
	case "javascript", "typescript", "jsx", "tsx":
		return strings.HasPrefix(trimmed, "function ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "export function ") ||
			strings.HasPrefix(trimmed, "export class ") ||
			strings.HasPrefix(trimmed, "export default function ") ||
			strings.HasPrefix(trimmed, "const ") && strings.Contains(trimmed, "= function") ||
			strings.HasPrefix(trimmed, "const ") && strings.Contains(trimmed, "=>")
	case "rust":
		return strings.HasPrefix(trimmed, "fn ") ||
			strings.HasPrefix(trimmed, "pub fn ") ||
			strings.HasPrefix(trimmed, "impl ") ||
			strings.HasPrefix(trimmed, "struct ") ||
			strings.HasPrefix(trimmed, "pub struct ") ||
			strings.HasPrefix(trimmed, "enum ") ||
			strings.HasPrefix(trimmed, "pub enum ")
	case "java":
		return strings.Contains(trimmed, "class ") ||
			strings.Contains(trimmed, "interface ") ||
			(strings.Contains(trimmed, "(") && strings.Contains(trimmed, "{") &&
				!strings.HasPrefix(trimmed, "if") &&
				!strings.HasPrefix(trimmed, "for") &&
				!strings.HasPrefix(trimmed, "while"))
	case "c", "cpp", "h":
		// Very basic heuristic for C/C++
		return strings.Contains(trimmed, "(") &&
			strings.HasSuffix(strings.TrimSpace(trimmed), "{") &&
			!strings.HasPrefix(trimmed, "if") &&
			!strings.HasPrefix(trimmed, "for") &&
			!strings.HasPrefix(trimmed, "while")
	}

	return false
}

// SupportedLanguages returns all languages (simple chunking works with any text).
func (c *Chunker) SupportedLanguages() []string {
	return []string{
		"go", "python", "javascript", "typescript", "jsx", "tsx",
		"rust", "java", "c", "cpp", "h", "ruby", "php", "swift",
		"kotlin", "scala", "haskell", "ocaml", "elixir", "erlang",
		"lua", "perl", "r", "julia", "dart", "text",
	}
}

// SupportsLanguage returns true for any language.
func (c *Chunker) SupportsLanguage(lang string) bool {
	return true // Simple chunker works with any text
}

// ExtractSymbols returns empty slice (simple chunker doesn't extract symbols).
func (c *Chunker) ExtractSymbols(file *types.SourceFile) ([]*types.Symbol, error) {
	// Simple chunker doesn't do symbol extraction
	return nil, nil
}

// ExtractReferences returns empty slice (simple chunker doesn't extract references).
func (c *Chunker) ExtractReferences(file *types.SourceFile) ([]*types.Reference, error) {
	// Simple chunker doesn't do reference extraction
	return nil, nil
}

// Close releases resources.
func (c *Chunker) Close() error {
	return nil
}

// DetectLanguage detects language from file extension.
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	// Handle special filenames first
	if base == "dockerfile" {
		return "dockerfile"
	}

	switch ext {
	// Core programming languages
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "h"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala", ".sc":
		return "scala"
	case ".cs":
		return "csharp"
	case ".lua":
		return "lua"
	case ".sql":
		return "sql"
	case ".dart":
		return "dart"
	case ".r", ".R":
		return "r"
	case ".ex", ".exs":
		return "elixir"
	case ".elm":
		return "elm"
	case ".groovy", ".gradle":
		return "groovy"
	case ".ml", ".mli":
		return "ocaml"

	// Markup and data formats
	case ".html", ".htm", ".xhtml":
		return "html"
	case ".css":
		return "css"
	case ".svelte":
		return "svelte"
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".proto":
		return "proto"
	case ".cue":
		return "cue"

	// Shell and scripting
	case ".sh", ".bash":
		return "bash"
	case ".ps1", ".psm1", ".psd1":
		return "powershell"

	// Infrastructure
	case ".tf", ".hcl":
		return "hcl"

	// Pascal family
	case ".pas", ".pp", ".dpr":
		return "pascal"

	// Visual Basic
	case ".vb":
		return "vbnet"

	// Functional languages
	case ".hs":
		return "haskell"
	case ".erl":
		return "erlang"
	case ".pl", ".pm":
		return "perl"
	case ".jl":
		return "julia"

	default:
		return "text"
	}
}

// Ensure Chunker implements ChunkingStrategy interface
var _ provider.ChunkingStrategy = (*Chunker)(nil)
