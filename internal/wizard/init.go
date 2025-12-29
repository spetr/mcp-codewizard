package wizard

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spetr/mcp-codewizard/internal/config"
)

// InitOption represents a configuration option for the wizard.
type InitOption struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Choices     []Choice `json:"choices"`
	Default     int      `json:"default"` // index of default choice
}

// Choice represents a single choice for an option.
type Choice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
	Available   bool   `json:"available"`
	Warning     string `json:"warning,omitempty"`
}

// InitWizardState represents the current state of the initialization wizard.
type InitWizardState struct {
	Environment *DetectEnvironmentResult `json:"environment"`
	Options     []InitOption             `json:"options"`
	Selections  map[string]string        `json:"selections,omitempty"`
	Config      *config.Config           `json:"config,omitempty"`
	Messages    []string                 `json:"messages,omitempty"`
	Ready       bool                     `json:"ready"`
}

// GetInitOptions returns available initialization options based on detected environment.
func (w *Wizard) GetInitOptions(env *DetectEnvironmentResult) []InitOption {
	options := []InitOption{}

	// 1. Embedding Provider
	embeddingOption := InitOption{
		ID:          "embedding",
		Name:        "Embedding Provider",
		Description: "Choose how to generate code embeddings for semantic search",
		Choices:     []Choice{},
		Default:     0,
	}

	// Ollama option
	ollamaAvailable := env.Ollama.Available
	ollamaWarning := ""
	if !ollamaAvailable {
		ollamaWarning = "Ollama not running. Start with: ollama serve"
	} else {
		// Check if embedding model is available
		hasModel := false
		for _, m := range env.Ollama.Models {
			if strings.Contains(strings.ToLower(m.Name), "nomic-embed") ||
				strings.Contains(strings.ToLower(m.Name), "embed") {
				hasModel = true
				break
			}
		}
		if !hasModel {
			ollamaWarning = "Model not found. Run: ollama pull nomic-embed-code"
		}
	}

	embeddingOption.Choices = append(embeddingOption.Choices, Choice{
		ID:          "ollama",
		Label:       "Ollama (nomic-embed-code)",
		Description: "Local, fast, optimized for code. No API costs.",
		Recommended: ollamaAvailable && ollamaWarning == "",
		Available:   true, // Can still select even if not available yet
		Warning:     ollamaWarning,
	})

	// OpenAI option
	openaiAvailable := env.OpenAI.Available
	openaiWarning := ""
	if !openaiAvailable {
		openaiWarning = "Set OPENAI_API_KEY environment variable"
	}

	embeddingOption.Choices = append(embeddingOption.Choices, Choice{
		ID:          "openai",
		Label:       "OpenAI (text-embedding-3-small)",
		Description: "Cloud-based, high quality. Requires API key, has costs.",
		Recommended: !ollamaAvailable && openaiAvailable,
		Available:   true,
		Warning:     openaiWarning,
	})

	embeddingOption.Choices = append(embeddingOption.Choices, Choice{
		ID:          "openai-large",
		Label:       "OpenAI (text-embedding-3-large)",
		Description: "Highest quality embeddings. Higher cost.",
		Recommended: false,
		Available:   true,
		Warning:     openaiWarning,
	})

	// Set default based on availability
	if ollamaAvailable && ollamaWarning == "" {
		embeddingOption.Default = 0
	} else if openaiAvailable {
		embeddingOption.Default = 1
	}

	options = append(options, embeddingOption)

	// 2. Reranker
	rerankerOption := InitOption{
		ID:          "reranker",
		Name:        "Search Reranker",
		Description: "Reranking improves search accuracy by ~15% but uses more resources",
		Choices:     []Choice{},
		Default:     0,
	}

	hasReranker := false
	for _, m := range env.Ollama.Models {
		if strings.Contains(strings.ToLower(m.Name), "rerank") {
			hasReranker = true
			break
		}
	}

	rerankerWarning := ""
	if ollamaAvailable && !hasReranker {
		rerankerWarning = "Model not found. Run: ollama pull qwen3-reranker"
	} else if !ollamaAvailable {
		rerankerWarning = "Requires Ollama"
	}

	rerankerOption.Choices = append(rerankerOption.Choices, Choice{
		ID:          "enabled",
		Label:       "Enabled (qwen3-reranker)",
		Description: "Better search accuracy, uses ~600MB VRAM",
		Recommended: ollamaAvailable && hasReranker,
		Available:   true,
		Warning:     rerankerWarning,
	})

	rerankerOption.Choices = append(rerankerOption.Choices, Choice{
		ID:          "disabled",
		Label:       "Disabled",
		Description: "Faster, less resource usage, slightly lower accuracy",
		Recommended: !ollamaAvailable || env.Project.Complexity == "huge",
		Available:   true,
	})

	if ollamaAvailable && hasReranker {
		rerankerOption.Default = 0
	} else {
		rerankerOption.Default = 1
	}

	options = append(options, rerankerOption)

	// 3. Chunking Strategy
	chunkingOption := InitOption{
		ID:          "chunking",
		Name:        "Code Chunking",
		Description: "How to split code files for indexing",
		Choices:     []Choice{},
		Default:     0,
	}

	// Check which languages are detected
	langs := []string{}
	for lang := range env.Project.LanguagesDetected {
		langs = append(langs, lang)
	}
	langsStr := strings.Join(langs, ", ")

	chunkingOption.Choices = append(chunkingOption.Choices, Choice{
		ID:          "treesitter",
		Label:       "TreeSitter (semantic)",
		Description: fmt.Sprintf("AST-aware chunking. Detected: %s", langsStr),
		Recommended: true,
		Available:   true,
	})

	chunkingOption.Choices = append(chunkingOption.Choices, Choice{
		ID:          "simple",
		Label:       "Simple (line-based)",
		Description: "Fast, works with any language. Less accurate boundaries.",
		Recommended: false,
		Available:   true,
	})

	options = append(options, chunkingOption)

	// 4. Search Mode
	searchOption := InitOption{
		ID:          "search",
		Name:        "Search Mode",
		Description: "How to search the codebase",
		Choices:     []Choice{},
		Default:     0,
	}

	searchOption.Choices = append(searchOption.Choices, Choice{
		ID:          "hybrid",
		Label:       "Hybrid (BM25 + Vector)",
		Description: "Best accuracy. Combines keyword and semantic search.",
		Recommended: true,
		Available:   true,
	})

	searchOption.Choices = append(searchOption.Choices, Choice{
		ID:          "vector",
		Label:       "Vector only",
		Description: "Pure semantic search. Good for conceptual queries.",
		Recommended: false,
		Available:   true,
	})

	searchOption.Choices = append(searchOption.Choices, Choice{
		ID:          "bm25",
		Label:       "BM25 only",
		Description: "Keyword search only. Fast, no embeddings needed.",
		Recommended: false,
		Available:   true,
	})

	options = append(options, searchOption)

	// 5. Preset (quick option)
	presetOption := InitOption{
		ID:          "preset",
		Name:        "Configuration Preset",
		Description: "Quick configuration presets based on your needs",
		Choices:     []Choice{},
		Default:     0,
	}

	presetOption.Choices = append(presetOption.Choices, Choice{
		ID:          "recommended",
		Label:       "Recommended",
		Description: "Best balance of quality and performance for your system",
		Recommended: true,
		Available:   true,
	})

	presetOption.Choices = append(presetOption.Choices, Choice{
		ID:          "quality",
		Label:       "High Quality",
		Description: "Maximum search accuracy, more resources",
		Recommended: false,
		Available:   env.OpenAI.Available || (ollamaAvailable && hasReranker),
	})

	presetOption.Choices = append(presetOption.Choices, Choice{
		ID:          "fast",
		Label:       "Fast & Light",
		Description: "Minimal resources, good for large projects",
		Recommended: false,
		Available:   true,
	})

	presetOption.Choices = append(presetOption.Choices, Choice{
		ID:          "custom",
		Label:       "Custom",
		Description: "Configure each option individually",
		Recommended: false,
		Available:   true,
	})

	options = append(options, presetOption)

	return options
}

// ApplyPreset applies a preset configuration.
func (w *Wizard) ApplyPreset(env *DetectEnvironmentResult, preset string) *config.Config {
	cfg := config.DefaultConfig()

	switch preset {
	case "recommended":
		// Use recommendations from environment detection
		if env.Recommendations != nil {
			rec := env.Recommendations.Primary
			cfg.Embedding.Provider = rec.Embedding.Provider
			cfg.Embedding.Model = rec.Embedding.Model
			cfg.Embedding.BatchSize = rec.Embedding.BatchSize
			cfg.Reranker.Enabled = rec.Reranker.Enabled
			cfg.Reranker.Provider = rec.Reranker.Provider
			cfg.Reranker.Model = rec.Reranker.Model
			cfg.Reranker.Candidates = rec.Reranker.Candidates
			cfg.Chunking.Strategy = rec.Chunking.Strategy
			cfg.Chunking.MaxChunkSize = rec.Chunking.MaxChunkSize
			cfg.Limits.Workers = rec.Indexing.Workers
			cfg.Index.UseGitIgnore = rec.Indexing.UseGitIgnore
		}

	case "quality":
		// High quality settings
		if env.OpenAI.Available {
			cfg.Embedding.Provider = "openai"
			cfg.Embedding.Model = "text-embedding-3-large"
		}
		cfg.Reranker.Enabled = true
		cfg.Reranker.Candidates = 200
		cfg.Chunking.Strategy = "treesitter"
		cfg.Search.Mode = "hybrid"

	case "fast":
		// Fast/light settings
		cfg.Embedding.Provider = "ollama"
		cfg.Embedding.Model = "nomic-embed-code"
		cfg.Embedding.BatchSize = 64
		cfg.Reranker.Enabled = false
		cfg.Chunking.Strategy = "simple"
		cfg.Chunking.MaxChunkSize = 1500
		cfg.Limits.Workers = runtime.NumCPU()
		cfg.Search.Mode = "hybrid"

	default:
		// Use defaults
	}

	return cfg
}

// ApplySelections applies user selections to create a config.
func (w *Wizard) ApplySelections(env *DetectEnvironmentResult, selections map[string]string) *config.Config {
	cfg := config.DefaultConfig()

	// Check if using a preset
	if preset, ok := selections["preset"]; ok && preset != "custom" {
		return w.ApplyPreset(env, preset)
	}

	// Apply individual selections
	if embedding, ok := selections["embedding"]; ok {
		switch embedding {
		case "ollama":
			cfg.Embedding.Provider = "ollama"
			cfg.Embedding.Model = "nomic-embed-code"
			cfg.Embedding.Endpoint = "http://localhost:11434"
		case "openai":
			cfg.Embedding.Provider = "openai"
			cfg.Embedding.Model = "text-embedding-3-small"
		case "openai-large":
			cfg.Embedding.Provider = "openai"
			cfg.Embedding.Model = "text-embedding-3-large"
		}
	}

	if reranker, ok := selections["reranker"]; ok {
		cfg.Reranker.Enabled = reranker == "enabled"
		if cfg.Reranker.Enabled {
			cfg.Reranker.Provider = "ollama"
			cfg.Reranker.Model = "qwen3-reranker"
			cfg.Reranker.Endpoint = "http://localhost:11434"
		}
	}

	if chunking, ok := selections["chunking"]; ok {
		cfg.Chunking.Strategy = chunking
	}

	if search, ok := selections["search"]; ok {
		cfg.Search.Mode = search
	}

	return cfg
}

// FormatEnvironmentSummary returns a formatted summary of the detected environment.
func FormatEnvironmentSummary(env *DetectEnvironmentResult) string {
	var sb strings.Builder

	sb.WriteString("=== Environment ===\n")

	// Ollama
	if env.Ollama.Available {
		sb.WriteString(fmt.Sprintf("Ollama: ✓ Running at %s\n", env.Ollama.Endpoint))
		for _, m := range env.Ollama.Models {
			icon := "  "
			if m.Recommended {
				icon = "✓ "
			}
			sb.WriteString(fmt.Sprintf("  %s%s (%s, %s)\n", icon, m.Name, m.Type, m.Size))
		}
	} else {
		sb.WriteString("Ollama: ✗ Not running\n")
		if env.Ollama.Error != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", env.Ollama.Error))
		}
	}

	// OpenAI
	if env.OpenAI.Available {
		sb.WriteString("OpenAI: ✓ API key configured\n")
	} else {
		sb.WriteString("OpenAI: ✗ Not configured\n")
	}

	// System
	sb.WriteString(fmt.Sprintf("System: %s/%s, %d cores", env.System.OS, env.System.Arch, env.System.CPUCores))
	if env.System.TotalRAM != "" {
		sb.WriteString(fmt.Sprintf(", %s RAM", env.System.TotalRAM))
	}
	if env.System.HasGPU {
		sb.WriteString(fmt.Sprintf(", GPU: %s", env.System.GPUInfo))
	}
	sb.WriteString("\n")

	sb.WriteString("\n=== Project ===\n")
	sb.WriteString(fmt.Sprintf("Path: %s\n", env.Project.Path))

	if env.Project.IsGit {
		sb.WriteString("Git: ✓ Repository detected\n")
	}

	sb.WriteString(fmt.Sprintf("Files: %d (%s)\n", env.Project.FileCount, env.Project.EstimatedSize))
	sb.WriteString(fmt.Sprintf("Complexity: %s (est. %d chunks)\n",
		env.Project.Complexity, env.Project.EstimatedChunks))

	// Languages
	if len(env.Project.LanguagesDetected) > 0 {
		sb.WriteString("Languages:\n")
		for lang, count := range env.Project.LanguagesDetected {
			sb.WriteString(fmt.Sprintf("  %s: %d files\n", lang, count))
		}
	}

	// Recommendations
	if env.Recommendations != nil && len(env.Recommendations.Reasoning) > 0 {
		sb.WriteString("\n=== Notes ===\n")
		for _, r := range env.Recommendations.Reasoning {
			sb.WriteString(fmt.Sprintf("• %s\n", r))
		}
	}

	return sb.String()
}

// FormatConfigSummary returns a formatted summary of the configuration.
func FormatConfigSummary(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString("=== Configuration ===\n")
	sb.WriteString(fmt.Sprintf("Embedding: %s/%s\n", cfg.Embedding.Provider, cfg.Embedding.Model))

	if cfg.Reranker.Enabled {
		sb.WriteString(fmt.Sprintf("Reranker: %s/%s (top %d candidates)\n",
			cfg.Reranker.Provider, cfg.Reranker.Model, cfg.Reranker.Candidates))
	} else {
		sb.WriteString("Reranker: Disabled\n")
	}

	sb.WriteString(fmt.Sprintf("Chunking: %s (max %d tokens)\n",
		cfg.Chunking.Strategy, cfg.Chunking.MaxChunkSize))
	sb.WriteString(fmt.Sprintf("Search: %s (vector %.0f%%, BM25 %.0f%%)\n",
		cfg.Search.Mode, cfg.Search.VectorWeight*100, cfg.Search.BM25Weight*100))

	return sb.String()
}
