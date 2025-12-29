// mcp-codewizard is an MCP server for semantic code search and analysis.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	ollamaEmbed "github.com/spetr/mcp-codewizard/builtin/embedding/ollama"
	openaiEmbed "github.com/spetr/mcp-codewizard/builtin/embedding/openai"
	ollamaRerank "github.com/spetr/mcp-codewizard/builtin/reranker/ollama"
	"github.com/spetr/mcp-codewizard/builtin/reranker/none"
	simpleChunker "github.com/spetr/mcp-codewizard/builtin/chunking/simple"
	tsChunker "github.com/spetr/mcp-codewizard/builtin/chunking/treesitter"
	"github.com/spetr/mcp-codewizard/builtin/vectorstore/sqlitevec"
	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/internal/index"
	"github.com/spetr/mcp-codewizard/internal/mcp"
	"github.com/spetr/mcp-codewizard/internal/memory"
	"github.com/spetr/mcp-codewizard/internal/search"
	"github.com/spetr/mcp-codewizard/internal/wizard"
	"github.com/spetr/mcp-codewizard/pkg/plugin/host"
	"github.com/spetr/mcp-codewizard/pkg/plugin/shared"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

var (
	version   = "0.1.0"
	cfgFile   string
	logLevel  string
	logFormat string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mcp-codewizard",
	Short: "MCP server for semantic code search and analysis",
	Long: `mcp-codewizard is an MCP server that provides semantic code search
capabilities using vector embeddings and hybrid search (BM25 + vector).

It supports:
- Multiple embedding providers (Ollama, OpenAI)
- Reranking with Qwen3-Reranker
- TreeSitter-based code chunking
- Call graph and symbol analysis`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogging()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mcp-codewizard %s\n", version)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

var indexCmd = &cobra.Command{
	Use:   "index [path]",
	Short: "Index a codebase",
	Long:  `Index a codebase for semantic search. If no path is provided, indexes the current directory.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if dryRun {
			runDryRun(path)
		} else {
			runIndex(path, force)
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the codebase",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		mode, _ := cmd.Flags().GetString("mode")
		noRerank, _ := cmd.Flags().GetBool("no-rerank")

		runSearch(query, limit, mode, noRerank)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index status",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")
		runStatus(verbose)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		stdio, _ := cmd.Flags().GetBool("stdio")
		runServe(stdio)
	},
}

var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Watch for file changes and re-index automatically",
	Long:  `Watch for file changes and automatically re-index modified files. If no path is provided, watches the current directory.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		debounce, _ := cmd.Flags().GetInt("debounce")
		runWatch(path, debounce)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration",
	Run: func(cmd *cobra.Command, args []string) {
		runConfigInit()
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Run: func(cmd *cobra.Command, args []string) {
		runConfigValidate()
	},
}

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect environment and show recommendations",
	Run: func(cmd *cobra.Command, args []string) {
		runDetect()
	},
}

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Plugin management",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and loaded plugins",
	Run: func(cmd *cobra.Command, args []string) {
		runPluginList()
	},
}

var pluginLoadCmd = &cobra.Command{
	Use:   "load <name> <type>",
	Short: "Load a plugin (type: embedding, reranker)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runPluginLoad(args[0], args[1])
	},
}

// Memory commands
var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Project memory management (notes, decisions, issues, sessions)",
}

var memoryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show memory status and summary",
	Run: func(cmd *cobra.Command, args []string) {
		runMemoryStatus()
	},
}

var memoryNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "List project notes",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		tag, _ := cmd.Flags().GetString("tag")
		runMemoryNotesList(limit, tag)
	},
}

var memoryDecisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "List architecture decisions (ADRs)",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		status, _ := cmd.Flags().GetString("status")
		runMemoryDecisionsList(limit, status)
	},
}

var memoryIssuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List known issues",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		open, _ := cmd.Flags().GetBool("open")
		runMemoryIssuesList(limit, open)
	},
}

var memorySessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List session history",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runMemorySessions(limit)
	},
}

var memoryContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show current working context",
	Run: func(cmd *cobra.Command, args []string) {
		runMemoryContext()
	},
}

var memoryExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all memory as JSON",
	Run: func(cmd *cobra.Command, args []string) {
		runMemoryExport()
	},
}

var memoryClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all memory data",
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		runMemoryClear(force)
	},
}

var memoryInstallHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git hooks for memory sync",
	Run: func(cmd *cobra.Command, args []string) {
		runMemoryInstallHooks()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: .mcp-codewizard/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (text, json)")

	indexCmd.Flags().Bool("dry-run", false, "show what would be indexed")
	indexCmd.Flags().Bool("force", false, "force reindex all files")

	searchCmd.Flags().IntP("limit", "l", 10, "maximum results")
	searchCmd.Flags().StringP("mode", "m", "hybrid", "search mode (vector, bm25, hybrid)")
	searchCmd.Flags().Bool("no-rerank", false, "disable reranking")

	statusCmd.Flags().BoolP("verbose", "v", false, "show detailed statistics")

	serveCmd.Flags().Bool("stdio", false, "use stdio transport (for MCP)")

	watchCmd.Flags().Int("debounce", 500, "debounce time in milliseconds")

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)

	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginLoadCmd)

	// Memory command flags
	memoryNotesCmd.Flags().IntP("limit", "l", 20, "maximum notes to show")
	memoryNotesCmd.Flags().StringP("tag", "t", "", "filter by tag")

	memoryDecisionsCmd.Flags().IntP("limit", "l", 20, "maximum decisions to show")
	memoryDecisionsCmd.Flags().StringP("status", "s", "", "filter by status (proposed, accepted, deprecated)")

	memoryIssuesCmd.Flags().IntP("limit", "l", 20, "maximum issues to show")
	memoryIssuesCmd.Flags().Bool("open", false, "show only open issues")

	memorySessionsCmd.Flags().IntP("limit", "l", 10, "maximum sessions to show")

	memoryClearCmd.Flags().BoolP("force", "f", false, "force clear without confirmation")

	memoryCmd.AddCommand(memoryStatusCmd)
	memoryCmd.AddCommand(memoryNotesCmd)
	memoryCmd.AddCommand(memoryDecisionsCmd)
	memoryCmd.AddCommand(memoryIssuesCmd)
	memoryCmd.AddCommand(memorySessionsCmd)
	memoryCmd.AddCommand(memoryContextCmd)
	memoryCmd.AddCommand(memoryExportCmd)
	memoryCmd.AddCommand(memoryClearCmd)
	memoryCmd.AddCommand(memoryInstallHooksCmd)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(detectCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(memoryCmd)
}

func setupLogging() {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// createProviders creates all providers based on config.
func createProviders(cfg *config.Config) (provider.VectorStore, provider.EmbeddingProvider, provider.ChunkingStrategy, provider.Reranker, error) {
	// Create vector store
	store := sqlitevec.New()

	// Create embedding provider
	var embedding provider.EmbeddingProvider
	switch cfg.Embedding.Provider {
	case "ollama":
		embedding = ollamaEmbed.New(ollamaEmbed.Config{
			Model:     cfg.Embedding.Model,
			Endpoint:  cfg.Embedding.Endpoint,
			BatchSize: cfg.Embedding.BatchSize,
		})
	case "openai":
		embedding = openaiEmbed.New(openaiEmbed.Config{
			Model:     cfg.Embedding.Model,
			APIKey:    cfg.Embedding.APIKey,
			BatchSize: cfg.Embedding.BatchSize,
		})
	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Embedding.Provider)
	}

	// Create chunker
	var chunker provider.ChunkingStrategy
	switch cfg.Chunking.Strategy {
	case "treesitter":
		chunker = tsChunker.New(tsChunker.Config{
			MaxChunkSize: cfg.Chunking.MaxChunkSize,
		})
	case "simple":
		chunker = simpleChunker.New(simpleChunker.Config{
			MaxChunkSize: cfg.Chunking.MaxChunkSize,
		})
	default:
		chunker = simpleChunker.New(simpleChunker.Config{
			MaxChunkSize: cfg.Chunking.MaxChunkSize,
		})
	}

	// Create reranker
	var reranker provider.Reranker
	if cfg.Reranker.Enabled {
		switch cfg.Reranker.Provider {
		case "ollama":
			reranker = ollamaRerank.New(ollamaRerank.Config{
				Model:      cfg.Reranker.Model,
				Endpoint:   cfg.Reranker.Endpoint,
				MaxDocs:    cfg.Reranker.Candidates,
				Candidates: cfg.Reranker.Candidates,
			})
		case "none", "":
			reranker = none.New()
		}
	}

	return store, embedding, chunker, reranker, nil
}

func runDryRun(path string) {
	absPath, _ := filepath.Abs(path)
	slog.Info("dry-run mode", "path", absPath)

	// Detect environment
	wiz := wizard.New(absPath)
	result, err := wiz.DetectEnvironment(context.Background())
	if err != nil {
		slog.Error("detection failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Project Analysis ===")
	fmt.Printf("Path: %s\n", result.Project.Path)
	fmt.Printf("Files: %d\n", result.Project.FileCount)
	fmt.Printf("Complexity: %s\n", result.Project.Complexity)
	fmt.Printf("Estimated chunks: %d\n", result.Project.EstimatedChunks)

	fmt.Println("\nLanguages:")
	for lang, count := range result.Project.LanguagesDetected {
		fmt.Printf("  %s: %d files\n", lang, count)
	}

	if result.Recommendations != nil {
		fmt.Printf("\nEstimated indexing time: %s\n", result.Recommendations.Primary.Indexing.EstimatedTime)
		fmt.Printf("Estimated DB size: %s\n", result.Recommendations.Primary.Indexing.EstimatedDBSize)
	}
}

func runIndex(path string, force bool) {
	absPath, _ := filepath.Abs(path)
	slog.Info("indexing", "path", absPath, "force", force)

	cfg, warnings, err := config.Load(absPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	for _, w := range warnings {
		slog.Warn(w)
	}

	slog.Info("loaded config",
		"embedding", cfg.Embedding.Provider+"/"+cfg.Embedding.Model,
		"chunking", cfg.Chunking.Strategy,
		"reranker", cfg.Reranker.Enabled,
	)

	// Create providers
	store, embedding, chunker, _, err := createProviders(cfg)
	if err != nil {
		slog.Error("failed to create providers", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	interrupted := false
	go func() {
		sig := <-sigChan
		slog.Info("received interrupt signal, stopping indexing...", "signal", sig)
		interrupted = true
		cancel()
	}()

	// Cleanup on exit
	defer func() {
		signal.Stop(sigChan)
		store.Close()
		embedding.Close()
		chunker.Close()
		if interrupted {
			fmt.Println("\nIndexing interrupted. Progress saved - run again to resume.")
		}
	}()

	// Initialize store
	dbPath := config.IndexDBPath(absPath)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Warmup embedding
	if err := embedding.Warmup(ctx); err != nil {
		slog.Warn("embedding warmup failed", "error", err)
	}

	// Create and run indexer
	indexer := index.New(index.Config{
		ProjectDir: absPath,
		Config:     cfg,
		Store:      store,
		Embedding:  embedding,
		Chunker:    chunker,
		OnProgress: func(p types.IndexProgress) {
			if p.Phase != "" {
				fmt.Printf("\r[%s] Files: %d/%d, Chunks: %d/%d",
					p.Phase, p.ProcessedFiles, p.TotalFiles,
					p.ProcessedChunks, p.TotalChunks)
			}
		},
	})

	if err := indexer.Index(ctx, force); err != nil {
		if ctx.Err() != nil {
			// Context was cancelled - this is expected on interrupt
			slog.Info("indexing stopped by user")
		} else {
			slog.Error("indexing failed", "error", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println("\nIndexing complete!")

	// Show stats
	stats, _ := store.GetStats()
	if stats != nil {
		fmt.Printf("Files: %d, Chunks: %d, Symbols: %d\n",
			stats.IndexedFiles, stats.TotalChunks, stats.TotalSymbols)
	}
}

func runSearch(query string, limit int, mode string, noRerank bool) {
	cwd, _ := os.Getwd()
	slog.Debug("searching", "query", query, "limit", limit, "mode", mode, "no-rerank", noRerank)

	cfg, _, err := config.Load(cwd)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create providers
	store, embedding, _, reranker, err := createProviders(cfg)
	if err != nil {
		slog.Error("failed to create providers", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	defer embedding.Close()

	// Initialize store
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Create search engine
	engine := search.New(search.Config{
		Store:     store,
		Embedding: embedding,
		Reranker:  reranker,
	})

	// Determine search mode
	searchMode := types.SearchModeHybrid
	switch mode {
	case "vector":
		searchMode = types.SearchModeVector
	case "bm25":
		searchMode = types.SearchModeBM25
	}

	// Execute search
	ctx := context.Background()
	results, err := engine.Search(ctx, &types.SearchRequest{
		Query:       query,
		Limit:       limit,
		Mode:        searchMode,
		UseReranker: !noRerank && reranker != nil,
	})
	if err != nil {
		slog.Error("search failed", "error", err)
		os.Exit(1)
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	for i, r := range results {
		fmt.Printf("\n=== Result %d (score: %.3f) ===\n", i+1, r.Score)
		fmt.Printf("File: %s:%d-%d\n", r.Chunk.FilePath, r.Chunk.StartLine, r.Chunk.EndLine)
		if r.Chunk.Name != "" {
			fmt.Printf("Name: %s (%s)\n", r.Chunk.Name, r.Chunk.ChunkType)
		}
		fmt.Printf("\n%s\n", r.Chunk.Content)
	}
}

func runStatus(verbose bool) {
	cwd, _ := os.Getwd()

	cfg, _, err := config.Load(cwd)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create store
	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		fmt.Println("No index found. Run 'mcp-codewizard index' to create one.")
		return
	}
	defer store.Close()

	// Get stats
	stats, err := store.GetStats()
	if err != nil {
		slog.Error("failed to get stats", "error", err)
		os.Exit(1)
	}

	meta, _ := store.GetMetadata()

	fmt.Println("=== Index Status ===")
	fmt.Printf("Indexed files: %d\n", stats.IndexedFiles)
	fmt.Printf("Total chunks:  %d\n", stats.TotalChunks)
	fmt.Printf("Total symbols: %d\n", stats.TotalSymbols)
	fmt.Printf("Total refs:    %d\n", stats.TotalReferences)
	fmt.Printf("Database size: %s\n", formatBytes(stats.DBSizeBytes))

	if !stats.LastIndexed.IsZero() {
		fmt.Printf("Last indexed:  %s\n", stats.LastIndexed.Format("2006-01-02 15:04:05"))
	}

	if verbose && meta != nil {
		fmt.Println("\n=== Configuration ===")
		fmt.Printf("Embedding:  %s/%s\n", meta.EmbeddingProvider, meta.EmbeddingModel)
		fmt.Printf("Dimensions: %d\n", meta.EmbeddingDimensions)
		fmt.Printf("Chunking:   %s\n", meta.ChunkingStrategy)
		if meta.RerankerModel != "" {
			fmt.Printf("Reranker:   %s\n", meta.RerankerModel)
		}
		fmt.Printf("Tool:       %s\n", meta.ToolVersion)
	}

	if verbose {
		fmt.Println("\n=== Current Config ===")
		fmt.Printf("Embedding:  %s/%s\n", cfg.Embedding.Provider, cfg.Embedding.Model)
		fmt.Printf("Chunking:   %s\n", cfg.Chunking.Strategy)
		fmt.Printf("Reranker:   %v\n", cfg.Reranker.Enabled)
	}
}

func runServe(stdio bool) {
	cwd, _ := os.Getwd()
	slog.Info("starting MCP server", "stdio", stdio)

	cfg, warnings, err := config.Load(cwd)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	for _, w := range warnings {
		slog.Warn(w)
	}

	// Create providers
	store, embedding, chunker, reranker, err := createProviders(cfg)
	if err != nil {
		slog.Error("failed to create providers", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("received shutdown signal", "signal", sig)
		cancel()

		// Cleanup resources
		slog.Info("closing providers...")
		if err := store.Close(); err != nil {
			slog.Warn("failed to close store", "error", err)
		}
		if err := embedding.Close(); err != nil {
			slog.Warn("failed to close embedding", "error", err)
		}
		if err := chunker.Close(); err != nil {
			slog.Warn("failed to close chunker", "error", err)
		}
		slog.Info("shutdown complete")
		os.Exit(0)
	}()

	// Ensure cleanup on normal exit too
	defer func() {
		signal.Stop(sigChan)
		store.Close()
		embedding.Close()
		chunker.Close()
	}()

	// Initialize store
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Warmup providers
	if err := embedding.Warmup(ctx); err != nil {
		slog.Warn("embedding warmup failed", "error", err)
	}
	if reranker != nil {
		if err := reranker.Warmup(ctx); err != nil {
			slog.Warn("reranker warmup failed", "error", err)
		}
	}

	// Create MCP server
	server, err := mcp.New(mcp.Config{
		ProjectDir: cwd,
		Config:     cfg,
		Store:      store,
		Embedding:  embedding,
		Chunker:    chunker,
		Reranker:   reranker,
	})
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Run server
	if stdio {
		slog.Info("MCP server running (press Ctrl+C to stop)")
		if err := server.ServeStdio(); err != nil {
			if ctx.Err() != nil {
				// Context was cancelled, normal shutdown
				slog.Info("server stopped")
			} else {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Println("HTTP server not implemented yet. Use --stdio for MCP.")
		os.Exit(1)
	}
}

func runConfigInit() {
	cwd, _ := os.Getwd()
	cfg := config.DefaultConfig()

	if err := config.Save(cwd, cfg); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Created config at %s\n", config.ConfigPath(cwd))
}

func runConfigValidate() {
	cwd, _ := os.Getwd()

	cfg, warnings, err := config.Load(cwd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		fmt.Printf("Warning: %s\n", w)
	}

	errs := config.Validate(cfg)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("Error: %v\n", e)
		}
		os.Exit(1)
	}

	// Test providers
	wiz := wizard.New(cwd)
	result, err := wiz.ValidateConfig(context.Background(), cfg)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
		os.Exit(1)
	}

	for name, test := range result.Tests {
		fmt.Printf("[%s] %s: %s\n", test.Status, name, test.Message)
	}

	if result.Valid {
		fmt.Println("\nConfiguration is valid")
	} else {
		fmt.Println("\nConfiguration has errors")
		os.Exit(1)
	}
}

func runDetect() {
	cwd, _ := os.Getwd()

	wiz := wizard.New(cwd)
	result, err := wiz.DetectEnvironment(context.Background())
	if err != nil {
		slog.Error("detection failed", "error", err)
		os.Exit(1)
	}

	// Output as JSON
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func runWatch(path string, debounceMs int) {
	absPath, _ := filepath.Abs(path)
	slog.Info("watching for changes", "path", absPath, "debounce_ms", debounceMs)

	cfg, warnings, err := config.Load(absPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	for _, w := range warnings {
		slog.Warn(w)
	}

	// Create providers
	store, embedding, chunker, _, err := createProviders(cfg)
	if err != nil {
		slog.Error("failed to create providers", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Cleanup on exit
	defer func() {
		signal.Stop(sigChan)
		store.Close()
		embedding.Close()
		chunker.Close()
	}()

	// Initialize store
	dbPath := config.IndexDBPath(absPath)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Warmup embedding
	if err := embedding.Warmup(ctx); err != nil {
		slog.Warn("embedding warmup failed", "error", err)
	}

	// Create watcher
	watcher, err := index.NewWatcher(index.WatcherConfig{
		ProjectDir:   absPath,
		Config:       cfg,
		Store:        store,
		Embedding:    embedding,
		Chunker:      chunker,
		DebounceTime: time.Duration(debounceMs) * time.Millisecond,
		OnProgress: func(p types.IndexProgress) {
			if p.CurrentFile != "" {
				fmt.Printf("[watch] Indexed: %s\n", p.CurrentFile)
			}
		},
	})
	if err != nil {
		slog.Error("failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	fmt.Printf("Watching %s for changes (press Ctrl+C to stop)\n", absPath)

	// Run watcher (blocks until context is cancelled)
	if err := watcher.Watch(ctx); err != nil {
		if ctx.Err() != nil {
			slog.Info("watcher stopped")
		} else {
			slog.Error("watcher error", "error", err)
			os.Exit(1)
		}
	}
}

func runPluginList() {
	cwd, _ := os.Getwd()
	pluginsDir := filepath.Join(cwd, ".mcp-codewizard", "plugins")

	manager := host.NewManager(pluginsDir)

	// Discover available plugins
	available, err := manager.DiscoverPlugins()
	if err != nil {
		slog.Error("failed to discover plugins", "error", err)
		os.Exit(1)
	}

	fmt.Println("=== Available Plugins ===")
	fmt.Printf("Plugins directory: %s\n\n", pluginsDir)

	if len(available) == 0 {
		fmt.Println("No plugins found.")
		fmt.Println("\nTo install a plugin:")
		fmt.Println("  1. Build or download a plugin binary")
		fmt.Println("  2. Copy it to .mcp-codewizard/plugins/")
		fmt.Println("  3. Make it executable (chmod +x)")
		return
	}

	for _, name := range available {
		fmt.Printf("  - %s\n", name)
	}

	fmt.Println("\nTo load a plugin, use:")
	fmt.Println("  mcp-codewizard plugin load <name> <type>")
	fmt.Println("  where type is: embedding, reranker")
}

func runPluginLoad(name string, pluginType string) {
	cwd, _ := os.Getwd()
	pluginsDir := filepath.Join(cwd, ".mcp-codewizard", "plugins")

	manager := host.NewManager(pluginsDir)
	defer manager.UnloadAll()

	// Parse plugin type
	var pType shared.PluginType
	switch pluginType {
	case "embedding":
		pType = shared.PluginTypeEmbedding
	case "reranker":
		pType = shared.PluginTypeReranker
	default:
		slog.Error("invalid plugin type", "type", pluginType, "valid", "embedding, reranker")
		os.Exit(1)
	}

	// Load the plugin
	loaded, err := manager.LoadPlugin(name, pType)
	if err != nil {
		slog.Error("failed to load plugin", "name", name, "error", err)
		os.Exit(1)
	}

	fmt.Printf("Plugin loaded: %s (type: %s)\n", name, pluginType)

	// Test the plugin
	switch pType {
	case shared.PluginTypeEmbedding:
		if loaded.Embedding != nil {
			fmt.Printf("  Name: %s\n", loaded.Embedding.Name())
			fmt.Printf("  Dimensions: %d\n", loaded.Embedding.Dimensions())
			fmt.Printf("  Max Batch Size: %d\n", loaded.Embedding.MaxBatchSize())

			// Test embedding
			fmt.Println("\nTesting embedding...")
			embeddings, err := loaded.Embedding.Embed([]string{"Hello, world!"})
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				fmt.Printf("  Generated %d embedding(s) of dimension %d\n", len(embeddings), len(embeddings[0]))
			}
		}

	case shared.PluginTypeReranker:
		if loaded.Reranker != nil {
			fmt.Printf("  Name: %s\n", loaded.Reranker.Name())
			fmt.Printf("  Max Documents: %d\n", loaded.Reranker.MaxDocuments())

			// Test reranking
			fmt.Println("\nTesting reranker...")
			results, err := loaded.Reranker.Rerank("test query", []string{"doc1", "doc2"})
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			} else {
				fmt.Printf("  Reranked %d documents\n", len(results))
			}
		}
	}

	fmt.Println("\nPlugin test complete.")
}

// Memory command implementations

func runMemoryStatus() {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	summary := store.GetSummary()
	git := memory.NewGitIntegration(store)

	fmt.Println("=== Memory Status ===")
	fmt.Printf("Project: %s\n", filepath.Base(cwd))

	if git.IsGitRepo() {
		state := git.GetGitState()
		fmt.Printf("Branch:  %s\n", state.Branch)
		if state.IsDirty {
			fmt.Printf("Status:  %d uncommitted changes\n", len(state.DirtyFiles))
		}
	}

	fmt.Printf("\nNotes:     %d\n", summary.NotesCount)
	fmt.Printf("Decisions: %d\n", summary.DecisionsCount)
	fmt.Printf("Todos:     %d active\n", summary.ActiveTodos)
	fmt.Printf("Issues:    %d open\n", summary.OpenIssues)

	if summary.CurrentTask != "" {
		fmt.Printf("\nCurrent task: %s\n", summary.CurrentTask)
	}

	if len(summary.PendingTodos) > 0 {
		fmt.Println("\nHigh priority todos:")
		for _, todo := range summary.PendingTodos {
			fmt.Printf("  - %s (%s)\n", todo.Title, todo.ID)
		}
	}
}

func runMemoryNotesList(limit int, tag string) {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	req := memory.ListNotesRequest{Limit: limit}
	if tag != "" {
		req.Tags = []string{tag}
	}

	notes := store.ListNotes(req)

	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return
	}

	fmt.Printf("=== Notes (%d) ===\n\n", len(notes))
	for _, note := range notes {
		fmt.Printf("[%s] %s\n", note.ID, note.Title)
		if len(note.Tags) > 0 {
			fmt.Printf("  Tags: %v\n", note.Tags)
		}
		fmt.Printf("  Created: %s by %s\n", note.CreatedAt.Format("2006-01-02 15:04"), note.Author)
		fmt.Println()
	}
}

func runMemoryDecisionsList(limit int, status string) {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	req := memory.ListDecisionsRequest{Limit: limit}
	if status != "" {
		req.Status = []string{status}
	}

	decisions := store.ListDecisions(req)

	if len(decisions) == 0 {
		fmt.Println("No decisions found.")
		return
	}

	fmt.Printf("=== Architecture Decisions (%d) ===\n\n", len(decisions))
	for _, dec := range decisions {
		fmt.Printf("[%s] %s (%s)\n", dec.ID, dec.Title, dec.Status)
		fmt.Printf("  Created: %s by %s\n", dec.CreatedAt.Format("2006-01-02"), dec.Author)
		fmt.Println()
	}
}

func runMemoryIssuesList(limit int, openOnly bool) {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	req := memory.ListIssuesRequest{Limit: limit}
	if openOnly {
		req.Status = []string{"open", "investigating"}
	}

	issues := store.ListIssues(req)

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return
	}

	fmt.Printf("=== Issues (%d) ===\n\n", len(issues))
	for _, issue := range issues {
		severity := issue.Severity
		if severity == "critical" {
			severity = "🔴 " + severity
		} else if severity == "major" {
			severity = "🟠 " + severity
		} else {
			severity = "🟡 " + severity
		}

		fmt.Printf("[%s] %s (%s) - %s\n", issue.ID, issue.Title, severity, issue.Status)
		if issue.FilePath != "" {
			fmt.Printf("  File: %s\n", issue.FilePath)
		}
		fmt.Println()
	}
}

func runMemorySessions(limit int) {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	sessions, err := store.GetSessionHistory(limit)
	if err != nil {
		slog.Error("failed to get session history", "error", err)
		os.Exit(1)
	}

	if len(sessions) == 0 {
		fmt.Println("No session history found.")
		return
	}

	fmt.Printf("=== Session History (%d) ===\n\n", len(sessions))
	for _, session := range sessions {
		fmt.Printf("[%s] %s\n", session.EndedAt.Format("2006-01-02 15:04"), session.Agent)
		fmt.Printf("  Summary: %s\n", session.Summary)
		if len(session.NextSteps) > 0 {
			fmt.Println("  Next steps:")
			for _, step := range session.NextSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		fmt.Println()
	}
}

func runMemoryContext() {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	summary := store.GetContextSummary()
	fmt.Println(summary)
}

func runMemoryExport() {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	mem, err := store.Export()
	if err != nil {
		slog.Error("failed to export memory", "error", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(mem, "", "  ")
	fmt.Println(string(output))
}

func runMemoryClear(force bool) {
	if !force {
		fmt.Println("This will delete all notes, decisions, todos, and issues.")
		fmt.Print("Are you sure? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	if err := store.Clear(); err != nil {
		slog.Error("failed to clear memory", "error", err)
		os.Exit(1)
	}

	fmt.Println("Memory cleared.")
}

func runMemoryInstallHooks() {
	cwd, _ := os.Getwd()

	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	git := memory.NewGitIntegration(store)

	if !git.IsGitRepo() {
		fmt.Println("Not a git repository.")
		os.Exit(1)
	}

	if err := git.InstallHooks(); err != nil {
		slog.Error("failed to install hooks", "error", err)
		os.Exit(1)
	}

	fmt.Println("Git hooks installed:")
	fmt.Println("  - post-merge: syncs memory after merge")
	fmt.Println("  - post-checkout: updates context on branch switch")
	fmt.Println("  - pre-commit: validates memory files")
}
