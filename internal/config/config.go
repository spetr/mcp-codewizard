// Package config handles configuration loading and validation.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete configuration.
type Config struct {
	Embedding   EmbeddingConfig   `mapstructure:"embedding" yaml:"embedding"`
	Chunking    ChunkingConfig    `mapstructure:"chunking" yaml:"chunking"`
	Reranker    RerankerConfig    `mapstructure:"reranker" yaml:"reranker"`
	Search      SearchConfig      `mapstructure:"search" yaml:"search"`
	VectorStore VectorStoreConfig `mapstructure:"vectorstore" yaml:"vectorstore"`
	Index       IndexConfig       `mapstructure:"index" yaml:"index"`
	Limits      LimitsConfig      `mapstructure:"limits" yaml:"limits"`
	Analysis    AnalysisConfig    `mapstructure:"analysis" yaml:"analysis"`
	Logging     LoggingConfig     `mapstructure:"logging" yaml:"logging"`
	MCP         MCPConfig         `mapstructure:"mcp" yaml:"mcp"`
}

// MCPConfig contains MCP server configuration.
type MCPConfig struct {
	// Mode controls which tools are exposed:
	// - "full": all tools registered individually (default, ~60 tools)
	// - "router": only route, list_tools, suggest_tool (3 tools)
	// - "hybrid": essential tools + router for others (~10 tools + router)
	Mode string `mapstructure:"mode" yaml:"mode"`
}

// EmbeddingConfig contains embedding provider configuration.
type EmbeddingConfig struct {
	Provider  string `mapstructure:"provider" yaml:"provider"`     // ollama, openai, voyage, jina
	Model     string `mapstructure:"model" yaml:"model"`           // model name
	Endpoint  string `mapstructure:"endpoint" yaml:"endpoint"`     // API endpoint
	APIKey    string `mapstructure:"api_key" yaml:"api_key"`       // API key
	BatchSize int    `mapstructure:"batch_size" yaml:"batch_size"` // documents per batch
}

// ChunkingConfig contains chunking strategy configuration.
type ChunkingConfig struct {
	Strategy     string `mapstructure:"strategy" yaml:"strategy"`             // treesitter, simple
	MaxChunkSize int    `mapstructure:"max_chunk_size" yaml:"max_chunk_size"` // max tokens per chunk
}

// RerankerConfig contains reranker configuration.
type RerankerConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Provider   string `mapstructure:"provider" yaml:"provider"`     // ollama, none
	Model      string `mapstructure:"model" yaml:"model"`           // model name
	Endpoint   string `mapstructure:"endpoint" yaml:"endpoint"`     // API endpoint
	Candidates int    `mapstructure:"candidates" yaml:"candidates"` // candidates for reranking
}

// SearchConfig contains search configuration.
type SearchConfig struct {
	Mode         string  `mapstructure:"mode" yaml:"mode"`                   // vector, bm25, hybrid
	VectorWeight float32 `mapstructure:"vector_weight" yaml:"vector_weight"` // weight for vector search
	BM25Weight   float32 `mapstructure:"bm25_weight" yaml:"bm25_weight"`     // weight for BM25
	DefaultLimit int     `mapstructure:"default_limit" yaml:"default_limit"` // default result limit
}

// VectorStoreConfig contains vector store configuration.
type VectorStoreConfig struct {
	Provider string `mapstructure:"provider" yaml:"provider"` // sqlitevec
}

// IndexConfig contains indexing configuration.
type IndexConfig struct {
	Include      []string `mapstructure:"include" yaml:"include"`             // glob patterns to include
	Exclude      []string `mapstructure:"exclude" yaml:"exclude"`             // glob patterns to exclude
	UseGitIgnore bool     `mapstructure:"use_gitignore" yaml:"use_gitignore"` // respect .gitignore
}

// LimitsConfig contains resource limits.
type LimitsConfig struct {
	MaxFileSize    string        `mapstructure:"max_file_size" yaml:"max_file_size"`       // e.g., "1MB"
	MaxFiles       int           `mapstructure:"max_files" yaml:"max_files"`               // max files to index
	MaxChunkTokens int           `mapstructure:"max_chunk_tokens" yaml:"max_chunk_tokens"` // max tokens per chunk
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout"`                   // indexing timeout
	MemoryLimit    string        `mapstructure:"memory_limit" yaml:"memory_limit"`         // memory limit
	Workers        int           `mapstructure:"workers" yaml:"workers"`                   // parallel workers
}

// AnalysisConfig contains analysis options.
type AnalysisConfig struct {
	ExtractSymbols    bool `mapstructure:"extract_symbols" yaml:"extract_symbols"`
	ExtractReferences bool `mapstructure:"extract_references" yaml:"extract_references"`
	DetectPatterns    bool `mapstructure:"detect_patterns" yaml:"detect_patterns"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level"`   // debug, info, warn, error
	Format string `mapstructure:"format" yaml:"format"` // text, json
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Embedding: EmbeddingConfig{
			Provider:  "ollama",
			Model:     "nomic-embed-code",
			Endpoint:  "http://localhost:11434",
			BatchSize: 32,
		},
		Chunking: ChunkingConfig{
			Strategy:     "treesitter",
			MaxChunkSize: 2000,
		},
		Reranker: RerankerConfig{
			Enabled:    true,
			Provider:   "ollama",
			Model:      "qwen3-reranker",
			Endpoint:   "http://localhost:11434",
			Candidates: 100,
		},
		Search: SearchConfig{
			Mode:         "hybrid",
			VectorWeight: 0.7,
			BM25Weight:   0.3,
			DefaultLimit: 10,
		},
		VectorStore: VectorStoreConfig{
			Provider: "sqlitevec",
		},
		Index: IndexConfig{
			Include: []string{
				// Core languages
				"**/*.go", "**/*.py", "**/*.js", "**/*.mjs", "**/*.cjs", "**/*.ts",
				"**/*.jsx", "**/*.tsx", "**/*.rs", "**/*.java",
				"**/*.c", "**/*.cpp", "**/*.cc", "**/*.cxx", "**/*.h", "**/*.hpp",
				// Additional languages
				"**/*.rb", "**/*.php", "**/*.cs", "**/*.kt", "**/*.kts",
				"**/*.swift", "**/*.scala", "**/*.sc",
				"**/*.lua", "**/*.sql", "**/*.proto",
				"**/*.sh", "**/*.bash",
				"**/*.ex", "**/*.exs", "**/*.elm",
				"**/*.groovy", "**/*.gradle",
				"**/*.ml", "**/*.mli",
				// New languages (v0.2.0)
				"**/*.dart", "**/*.vb",
				"**/*.pas", "**/*.pp", "**/*.dpr",
				"**/*.ps1", "**/*.psm1", "**/*.psd1",
				"**/*.r", "**/*.R",
				// Config/markup
				"**/*.html", "**/*.htm", "**/*.xhtml", "**/*.svelte", "**/*.vue",
				"**/*.css", "**/*.yaml", "**/*.yml",
				"**/*.toml", "**/*.json",
				"**/*.tf", "**/*.hcl", "**/*.cue",
				"**/*.md",
				"**/Dockerfile",
			},
			Exclude: []string{
				"**/vendor/**", "**/node_modules/**", "**/.git/**",
				"**/dist/**", "**/build/**", "**/target/**", "**/bin/**", "**/obj/**",
				"**/*.min.js", "**/*.min.css", "**/*.generated.*",
				"**/package-lock.json", "**/yarn.lock", "**/pnpm-lock.yaml",
				"**/go.sum", "**/Cargo.lock", "**/composer.lock",
			},
			UseGitIgnore: true,
		},
		Limits: LimitsConfig{
			MaxFileSize:    "1MB",
			MaxFiles:       50000,
			MaxChunkTokens: 2000,
			Timeout:        30 * time.Minute,
			Workers:        0, // 0 = use runtime.NumCPU()
		},
		Analysis: AnalysisConfig{
			ExtractSymbols:    true,
			ExtractReferences: true,
			DetectPatterns:    false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		MCP: MCPConfig{
			Mode: "hybrid",
		},
	}
}

// ConfigDir returns the path to .mcp-codewizard directory.
func ConfigDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".mcp-codewizard")
}

// ConfigPath returns the path to config.yaml.
func ConfigPath(projectRoot string) string {
	return filepath.Join(ConfigDir(projectRoot), "config.yaml")
}

// IndexDBPath returns the path to index.db.
func IndexDBPath(projectRoot string) string {
	return filepath.Join(ConfigDir(projectRoot), "index.db")
}

// Load loads configuration from file, falling back to defaults.
func Load(projectRoot string) (*Config, []string, error) {
	cfg := DefaultConfig()
	warnings := []string{}

	configPath := ConfigPath(projectRoot)

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		warnings = append(warnings, "No config file found, using defaults")
		return cfg, warnings, nil
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults for missing values
	if cfg.Embedding.Provider == "" {
		cfg.Embedding.Provider = "ollama"
		warnings = append(warnings, "Using default embedding provider: ollama")
	}
	if cfg.Embedding.Model == "" {
		cfg.Embedding.Model = "nomic-embed-code"
	}
	if cfg.Embedding.Endpoint == "" {
		cfg.Embedding.Endpoint = "http://localhost:11434"
	}
	if cfg.Embedding.BatchSize == 0 {
		cfg.Embedding.BatchSize = 32
	}

	if cfg.Chunking.Strategy == "" {
		cfg.Chunking.Strategy = "treesitter"
	}
	if cfg.Chunking.MaxChunkSize == 0 {
		cfg.Chunking.MaxChunkSize = 2000
	}

	if cfg.Reranker.Candidates == 0 {
		cfg.Reranker.Candidates = 100
	}

	if cfg.Search.DefaultLimit == 0 {
		cfg.Search.DefaultLimit = 10
	}
	if cfg.Search.VectorWeight == 0 && cfg.Search.BM25Weight == 0 {
		cfg.Search.VectorWeight = 0.7
		cfg.Search.BM25Weight = 0.3
	}

	return cfg, warnings, nil
}

// Save saves configuration to file.
func Save(projectRoot string, cfg *Config) error {
	configDir := ConfigDir(projectRoot)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(ConfigPath(projectRoot))
	v.SetConfigType("yaml")

	// Set all values
	v.Set("embedding", cfg.Embedding)
	v.Set("chunking", cfg.Chunking)
	v.Set("reranker", cfg.Reranker)
	v.Set("search", cfg.Search)
	v.Set("vectorstore", cfg.VectorStore)
	v.Set("index", cfg.Index)
	v.Set("limits", cfg.Limits)
	v.Set("analysis", cfg.Analysis)
	v.Set("logging", cfg.Logging)
	v.Set("mcp", cfg.MCP)

	return v.WriteConfig()
}

// Validate validates the configuration.
func Validate(cfg *Config) []error {
	var errs []error

	// Validate embedding
	validEmbeddingProviders := map[string]bool{
		"ollama": true, "openai": true, "voyage": true, "jina": true,
	}
	if !validEmbeddingProviders[cfg.Embedding.Provider] {
		errs = append(errs, fmt.Errorf("invalid embedding provider: %s", cfg.Embedding.Provider))
	}

	// Validate chunking
	validChunkingStrategies := map[string]bool{
		"treesitter": true, "simple": true,
	}
	if !validChunkingStrategies[cfg.Chunking.Strategy] {
		errs = append(errs, fmt.Errorf("invalid chunking strategy: %s", cfg.Chunking.Strategy))
	}

	// Validate reranker
	if cfg.Reranker.Enabled {
		validRerankerProviders := map[string]bool{
			"ollama": true, "none": true,
		}
		if !validRerankerProviders[cfg.Reranker.Provider] {
			errs = append(errs, fmt.Errorf("invalid reranker provider: %s", cfg.Reranker.Provider))
		}
	}

	// Validate search
	validSearchModes := map[string]bool{
		"vector": true, "bm25": true, "hybrid": true,
	}
	if cfg.Search.Mode != "" && !validSearchModes[cfg.Search.Mode] {
		errs = append(errs, fmt.Errorf("invalid search mode: %s", cfg.Search.Mode))
	}

	// Validate MCP mode
	validMCPModes := map[string]bool{
		"full": true, "router": true, "hybrid": true, "": true,
	}
	if !validMCPModes[cfg.MCP.Mode] {
		errs = append(errs, fmt.Errorf("invalid MCP mode: %s (valid: full, router, hybrid)", cfg.MCP.Mode))
	}

	return errs
}

// Hash returns a hash of configuration that affects indexing.
// Used for detecting when reindexing is needed.
func (c *Config) Hash() string {
	data := fmt.Sprintf("%s:%s:%s:%d",
		c.Embedding.Provider,
		c.Embedding.Model,
		c.Chunking.Strategy,
		c.Chunking.MaxChunkSize,
	)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// Copy creates a deep copy of the config.
// Used for runtime modifications without affecting the original.
func (c *Config) Copy() *Config {
	copy := *c

	// Deep copy slices
	if c.Index.Include != nil {
		copy.Index.Include = make([]string, len(c.Index.Include))
		for i, v := range c.Index.Include {
			copy.Index.Include[i] = v
		}
	}
	if c.Index.Exclude != nil {
		copy.Index.Exclude = make([]string, len(c.Index.Exclude))
		for i, v := range c.Index.Exclude {
			copy.Index.Exclude[i] = v
		}
	}

	return &copy
}
