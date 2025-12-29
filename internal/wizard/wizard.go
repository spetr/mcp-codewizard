// Package wizard implements the setup wizard for environment detection and configuration.
package wizard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/builtin/chunking/simple"
	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// DetectEnvironmentResult contains detected environment information.
type DetectEnvironmentResult struct {
	Ollama  OllamaInfo  `json:"ollama"`
	OpenAI  OpenAIInfo  `json:"openai"`
	System  SystemInfo  `json:"system"`
	Project ProjectInfo `json:"project"`

	ExistingConfig *config.Config       `json:"existing_config,omitempty"`
	ExistingIndex  *types.IndexMetadata `json:"existing_index,omitempty"`

	Recommendations *ModelRecommendations `json:"recommendations"`
}

// OllamaInfo contains Ollama detection results.
type OllamaInfo struct {
	Available    bool        `json:"available"`
	Endpoint     string      `json:"endpoint"`
	Models       []ModelInfo `json:"models,omitempty"`
	TotalVRAM    string      `json:"total_vram,omitempty"`
	AvailableRAM string      `json:"available_ram,omitempty"`
	Error        string      `json:"error,omitempty"`
}

// ModelInfo contains information about an Ollama model.
type ModelInfo struct {
	Name        string `json:"name"`
	Size        string `json:"size"`
	Type        string `json:"type"` // "embedding", "reranker", "llm"
	Loaded      bool   `json:"loaded"`
	Recommended bool   `json:"recommended"`
}

// OpenAIInfo contains OpenAI availability.
type OpenAIInfo struct {
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

// SystemInfo contains system information.
type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	CPUCores     int    `json:"cpu_cores"`
	TotalRAM     string `json:"total_ram"`
	AvailableRAM string `json:"available_ram"`
	HasGPU       bool   `json:"has_gpu"`
	GPUInfo      string `json:"gpu_info,omitempty"`
}

// ProjectInfo contains project information.
type ProjectInfo struct {
	Path              string                    `json:"path"`
	IsGit             bool                      `json:"is_git"`
	LanguagesDetected map[string]int            `json:"languages_detected"`
	FileCount         int                       `json:"file_count"`
	TotalLines        int                       `json:"total_lines"`
	EstimatedSize     string                    `json:"estimated_size"`
	EstimatedChunks   int                       `json:"estimated_chunks"`
	Complexity        types.ProjectComplexity   `json:"complexity"`
}

// ModelRecommendations contains recommended settings.
type ModelRecommendations struct {
	Primary     RecommendationSet  `json:"primary"`
	LowMemory   *RecommendationSet `json:"low_memory,omitempty"`
	HighQuality *RecommendationSet `json:"high_quality,omitempty"`
	FastIndex   *RecommendationSet `json:"fast_index,omitempty"`
	Reasoning   []string           `json:"reasoning"`
}

// RecommendationSet contains a complete recommendation set.
type RecommendationSet struct {
	Embedding EmbeddingRecommendation `json:"embedding"`
	Reranker  RerankerRecommendation  `json:"reranker"`
	Chunking  ChunkingRecommendation  `json:"chunking"`
	Indexing  IndexingRecommendation  `json:"indexing"`
}

// EmbeddingRecommendation contains embedding model recommendation.
type EmbeddingRecommendation struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Dimensions    int    `json:"dimensions"`
	BatchSize     int    `json:"batch_size"`
	CodeOptimized bool   `json:"code_optimized"`
	Reason        string `json:"reason"`
}

// RerankerRecommendation contains reranker recommendation.
type RerankerRecommendation struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	Candidates int    `json:"candidates"`
	Reason     string `json:"reason"`
}

// ChunkingRecommendation contains chunking recommendation.
type ChunkingRecommendation struct {
	Strategy     string   `json:"strategy"`
	MaxChunkSize int      `json:"max_chunk_size"`
	Languages    []string `json:"languages"`
	Reason       string   `json:"reason"`
}

// IndexingRecommendation contains indexing recommendation.
type IndexingRecommendation struct {
	Workers         int    `json:"workers"`
	EstimatedTime   string `json:"estimated_time"`
	EstimatedDBSize string `json:"estimated_db_size"`
	UseGitIgnore    bool   `json:"use_gitignore"`
	Reason          string `json:"reason"`
}

// Wizard handles environment detection and recommendations.
type Wizard struct {
	projectDir string
}

// New creates a new wizard.
func New(projectDir string) *Wizard {
	return &Wizard{projectDir: projectDir}
}

// DetectEnvironment detects the environment and generates recommendations.
func (w *Wizard) DetectEnvironment(ctx context.Context) (*DetectEnvironmentResult, error) {
	result := &DetectEnvironmentResult{}

	// Detect Ollama
	result.Ollama = w.detectOllama(ctx)

	// Detect OpenAI
	result.OpenAI = w.detectOpenAI()

	// Detect system
	result.System = w.detectSystem()

	// Detect project
	result.Project = w.detectProject()

	// Load existing config if present
	cfg, _, err := config.Load(w.projectDir)
	if err == nil {
		result.ExistingConfig = cfg
	}

	// Generate recommendations
	result.Recommendations = w.generateRecommendations(result)

	return result, nil
}

// detectOllama checks Ollama availability.
func (w *Wizard) detectOllama(ctx context.Context) OllamaInfo {
	info := OllamaInfo{
		Endpoint: "http://localhost:11434",
	}

	// Check if Ollama is running
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", info.Endpoint+"/api/version", nil)
	resp, err := client.Do(req)
	if err != nil {
		info.Error = "Ollama not running at " + info.Endpoint
		return info
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("Ollama returned status %d", resp.StatusCode)
		return info
	}

	info.Available = true

	// Get models
	req, _ = http.NewRequestWithContext(ctx, "GET", info.Endpoint+"/api/tags", nil)
	resp, err = client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()

		var tagsResp struct {
			Models []struct {
				Name       string `json:"name"`
				Size       int64  `json:"size"`
				ModifiedAt string `json:"modified_at"`
			} `json:"models"`
		}

		if json.NewDecoder(resp.Body).Decode(&tagsResp) == nil {
			for _, m := range tagsResp.Models {
				modelType := detectModelType(m.Name)
				info.Models = append(info.Models, ModelInfo{
					Name:        m.Name,
					Size:        formatBytes(m.Size),
					Type:        modelType,
					Recommended: isRecommendedModel(m.Name),
				})
			}
		}
	}

	return info
}

// detectOpenAI checks OpenAI availability.
func (w *Wizard) detectOpenAI() OpenAIInfo {
	info := OpenAIInfo{}

	// Check for API key in environment
	if os.Getenv("OPENAI_API_KEY") != "" {
		info.Available = true
	} else {
		info.Error = "OPENAI_API_KEY not set"
	}

	return info
}

// detectSystem gets system information.
func (w *Wizard) detectSystem() SystemInfo {
	info := SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	// Try to get memory info
	if runtime.GOOS == "darwin" {
		// macOS
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			var memBytes int64
			if _, err := fmt.Sscanf(string(out), "%d", &memBytes); err == nil {
				info.TotalRAM = formatBytes(memBytes)
			}
		}

		// Check for GPU (Apple Silicon or discrete)
		out, err = exec.Command("system_profiler", "SPDisplaysDataType", "-json").Output()
		if err == nil {
			info.HasGPU = true
			if strings.Contains(string(out), "Apple") {
				info.GPUInfo = "Apple Silicon GPU"
			}
		}
	} else if runtime.GOOS == "linux" {
		// Linux
		out, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					var kb int64
					if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &kb); err == nil {
						info.TotalRAM = formatBytes(kb * 1024)
					}
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					var kb int64
					if _, err := fmt.Sscanf(line, "MemAvailable: %d kB", &kb); err == nil {
						info.AvailableRAM = formatBytes(kb * 1024)
					}
				}
			}
		}

		// Check for NVIDIA GPU
		if _, err := exec.LookPath("nvidia-smi"); err == nil {
			info.HasGPU = true
			info.GPUInfo = "NVIDIA GPU detected"
		}
	}

	return info
}

// detectProject analyzes the project.
func (w *Wizard) detectProject() ProjectInfo {
	info := ProjectInfo{
		Path:              w.projectDir,
		LanguagesDetected: make(map[string]int),
	}

	// Check if git repo
	if _, err := os.Stat(filepath.Join(w.projectDir, ".git")); err == nil {
		info.IsGit = true
	}

	// Scan files
	var totalSize int64
	var totalLines int

	_ = filepath.WalkDir(w.projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories and common excludes
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" ||
				name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Detect language
		lang := simple.DetectLanguage(path)
		if lang != "text" {
			info.LanguagesDetected[lang]++
			info.FileCount++

			// Get file size and estimate lines
			if fi, err := d.Info(); err == nil {
				totalSize += fi.Size()
				totalLines += int(fi.Size() / 50) // rough estimate
			}
		}

		return nil
	})

	info.TotalLines = totalLines
	info.EstimatedSize = formatBytes(totalSize)

	// Estimate chunks (rough: 1 chunk per 50 lines average)
	info.EstimatedChunks = totalLines / 50
	if info.EstimatedChunks < info.FileCount {
		info.EstimatedChunks = info.FileCount
	}

	// Determine complexity
	info.Complexity = types.DetermineComplexity(info.FileCount)

	return info
}

// generateRecommendations generates configuration recommendations.
func (w *Wizard) generateRecommendations(env *DetectEnvironmentResult) *ModelRecommendations {
	rec := &ModelRecommendations{
		Reasoning: []string{},
	}

	// Default primary recommendation
	rec.Primary = RecommendationSet{
		Embedding: EmbeddingRecommendation{
			Provider:      "ollama",
			Model:         "nomic-embed-code",
			Dimensions:    768,
			BatchSize:     32,
			CodeOptimized: true,
			Reason:        "Code-optimized embedding model, runs locally",
		},
		Reranker: RerankerRecommendation{
			Enabled:    true,
			Provider:   "ollama",
			Model:      "qwen3-reranker",
			Candidates: 100,
			Reason:     "Improves search accuracy by ~15%",
		},
		Chunking: ChunkingRecommendation{
			Strategy:     "treesitter",
			MaxChunkSize: 2000,
			Languages:    []string{},
			Reason:       "AST-aware chunking for better code understanding",
		},
		Indexing: IndexingRecommendation{
			Workers:       runtime.NumCPU(),
			UseGitIgnore:  env.Project.IsGit,
			EstimatedTime: estimateIndexTime(env.Project.FileCount),
			EstimatedDBSize: estimateDBSize(env.Project.EstimatedChunks),
			Reason:        fmt.Sprintf("Parallel indexing on %d cores", runtime.NumCPU()),
		},
	}

	// Add detected languages
	for lang := range env.Project.LanguagesDetected {
		rec.Primary.Chunking.Languages = append(rec.Primary.Chunking.Languages, lang)
	}

	// Adjust based on Ollama availability
	if !env.Ollama.Available {
		if env.OpenAI.Available {
			rec.Primary.Embedding.Provider = "openai"
			rec.Primary.Embedding.Model = "text-embedding-3-small"
			rec.Primary.Embedding.Reason = "Using OpenAI (Ollama not available)"
			rec.Primary.Reranker.Enabled = false
			rec.Primary.Reranker.Reason = "Disabled (requires Ollama)"
			rec.Reasoning = append(rec.Reasoning, "Ollama not available, using OpenAI")
		} else {
			rec.Reasoning = append(rec.Reasoning,
				"WARNING: No embedding provider available. Install Ollama or set OPENAI_API_KEY")
		}
	} else {
		// Check if recommended models are available
		hasEmbedding := false
		hasReranker := false
		for _, m := range env.Ollama.Models {
			if strings.Contains(m.Name, "nomic-embed") || strings.Contains(m.Name, "embed") {
				hasEmbedding = true
			}
			if strings.Contains(m.Name, "rerank") {
				hasReranker = true
			}
		}

		if !hasEmbedding {
			rec.Reasoning = append(rec.Reasoning,
				"Run: ollama pull nomic-embed-code")
		}
		if !hasReranker {
			rec.Reasoning = append(rec.Reasoning,
				"Run: ollama pull qwen3-reranker (optional, for better search)")
		}
	}

	// Adjust for project size
	switch env.Project.Complexity {
	case types.ComplexitySmall:
		rec.Reasoning = append(rec.Reasoning, "Small project - full quality settings")
	case types.ComplexityMedium:
		rec.Reasoning = append(rec.Reasoning, "Medium project - standard settings")
	case types.ComplexityLarge:
		rec.Reasoning = append(rec.Reasoning,
			"Large project - consider using exclude patterns")
		rec.Primary.Reranker.Candidates = 50 // Reduce for speed
	case types.ComplexityHuge:
		rec.Reasoning = append(rec.Reasoning,
			"WARNING: Very large project (>10k files). Consider indexing specific directories.")
		rec.Primary.Reranker.Enabled = false
		rec.Primary.Reranker.Reason = "Disabled for performance"
	}

	// Low memory alternative
	rec.LowMemory = &RecommendationSet{
		Embedding: EmbeddingRecommendation{
			Provider:   "ollama",
			Model:      "nomic-embed-code",
			BatchSize:  8,
			Reason:     "Smaller batch size for low memory",
		},
		Reranker: RerankerRecommendation{
			Enabled: false,
			Reason:  "Disabled to save ~600MB VRAM",
		},
		Chunking: ChunkingRecommendation{
			Strategy:     "simple",
			MaxChunkSize: 1500,
			Reason:       "Simpler chunking uses less memory",
		},
		Indexing: IndexingRecommendation{
			Workers:      runtime.NumCPU() / 2,
			UseGitIgnore: true,
			Reason:       "Fewer workers for lower memory usage",
		},
	}

	// High quality alternative
	rec.HighQuality = &RecommendationSet{
		Embedding: EmbeddingRecommendation{
			Provider:   "openai",
			Model:      "text-embedding-3-large",
			Dimensions: 3072,
			BatchSize:  32,
			Reason:     "Higher quality embeddings",
		},
		Reranker: RerankerRecommendation{
			Enabled:    true,
			Provider:   "ollama",
			Model:      "qwen3-reranker",
			Candidates: 200,
			Reason:     "More candidates for better accuracy",
		},
		Chunking: ChunkingRecommendation{
			Strategy: "treesitter",
			Reason:   "AST-aware chunking",
		},
		Indexing: IndexingRecommendation{
			Workers: runtime.NumCPU(),
			Reason:  "Full parallel indexing",
		},
	}

	return rec
}

// Helper functions

func detectModelType(name string) string {
	name = strings.ToLower(name)
	if strings.Contains(name, "embed") {
		return "embedding"
	}
	if strings.Contains(name, "rerank") {
		return "reranker"
	}
	return "llm"
}

func isRecommendedModel(name string) bool {
	name = strings.ToLower(name)
	recommended := []string{"nomic-embed", "qwen3-rerank"}
	for _, r := range recommended {
		if strings.Contains(name, r) {
			return true
		}
	}
	return false
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func estimateIndexTime(fileCount int) string {
	// Rough estimate: 100 files per minute
	minutes := fileCount / 100
	if minutes < 1 {
		return "< 1 minute"
	}
	if minutes == 1 {
		return "~1 minute"
	}
	return fmt.Sprintf("~%d minutes", minutes)
}

func estimateDBSize(chunkCount int) string {
	// Rough estimate: 2KB per chunk (content + embedding)
	bytes := int64(chunkCount) * 2 * 1024
	return formatBytes(bytes)
}

// ValidateConfig validates configuration and tests providers.
func (w *Wizard) ValidateConfig(ctx context.Context, cfg *config.Config) (*ValidateResult, error) {
	result := &ValidateResult{
		Valid:    true,
		Tests:    make(map[string]TestResult),
	}

	// Validate config structure
	errs := config.Validate(cfg)
	for _, err := range errs {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	// Test Ollama connection
	if cfg.Embedding.Provider == "ollama" {
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", cfg.Embedding.Endpoint+"/api/version", nil)
		resp, err := client.Do(req)
		if err != nil {
			result.Tests["ollama_connection"] = TestResult{
				Status:  "error",
				Message: "Cannot connect to Ollama: " + err.Error(),
			}
			result.Valid = false
		} else {
			resp.Body.Close()
			result.Tests["ollama_connection"] = TestResult{
				Status:  "ok",
				Message: "Connected to Ollama",
			}
		}
	}

	// Test embedding model
	if cfg.Embedding.Provider == "ollama" && result.Tests["ollama_connection"].Status == "ok" {
		reqBody := map[string]any{"name": cfg.Embedding.Model}
		jsonBody, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "POST", cfg.Embedding.Endpoint+"/api/show", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)

		if err != nil || resp.StatusCode != http.StatusOK {
			result.Tests["embedding_model"] = TestResult{
				Status:  "error",
				Message: fmt.Sprintf("Model %s not found. Run: ollama pull %s", cfg.Embedding.Model, cfg.Embedding.Model),
			}
			result.Warnings = append(result.Warnings, "Embedding model not available")
		} else {
			resp.Body.Close()
			result.Tests["embedding_model"] = TestResult{
				Status:  "ok",
				Message: "Model " + cfg.Embedding.Model + " available",
			}
		}
	}

	return result, nil
}

// ValidateResult contains validation results.
type ValidateResult struct {
	Valid    bool                  `json:"valid"`
	Errors   []string              `json:"errors"`
	Warnings []string              `json:"warnings"`
	Tests    map[string]TestResult `json:"tests"`
}

// TestResult contains a single test result.
type TestResult struct {
	Status  string `json:"status"` // "ok", "error", "warning", "skipped"
	Message string `json:"message"`
}
