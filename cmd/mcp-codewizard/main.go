// mcp-codewizard is an MCP server for semantic code search and analysis.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
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
	"github.com/spetr/mcp-codewizard/internal/analysis"
	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/internal/github"
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
	version   = "0.2.0"
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

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize project with interactive setup wizard",
	Long: `Initialize a new project with an interactive setup wizard.

The wizard will:
1. Detect your environment (Ollama, OpenAI, system resources)
2. Analyze your project (languages, size, complexity)
3. Recommend optimal settings
4. Let you choose a preset or customize settings
5. Create the configuration file
6. Optionally start indexing

Examples:
  mcp-codewizard init           # Initialize current directory
  mcp-codewizard init ./myapp   # Initialize specific directory
  mcp-codewizard init --preset recommended  # Skip prompts, use recommended
  mcp-codewizard init --preset fast         # Skip prompts, use fast preset`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		preset, _ := cmd.Flags().GetString("preset")
		skipIndex, _ := cmd.Flags().GetBool("no-index")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		runInit(path, preset, skipIndex, jsonOutput)
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

// Register commands for AI CLI tools
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register mcp-codewizard with AI CLI tools",
	Long: `Register mcp-codewizard as an MCP server with various AI CLI tools.

Supported tools:
  claude-code  - Claude Code (Anthropic)
  gemini       - Gemini CLI (Google)
  codex        - Codex CLI (OpenAI)

Examples:
  mcp-codewizard register claude-code
  mcp-codewizard register gemini
  mcp-codewizard register codex
  mcp-codewizard register --all`,
}

var registerClaudeCodeCmd = &cobra.Command{
	Use:   "claude-code",
	Short: "Register with Claude Code",
	Long:  `Register mcp-codewizard as an MCP server in Claude Code configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		runRegisterClaudeCode(global)
	},
}

var registerGeminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Register with Gemini CLI",
	Long:  `Register mcp-codewizard as an MCP server in Gemini CLI configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		runRegisterGemini(global)
	},
}

var registerCodexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Register with Codex CLI",
	Long:  `Register mcp-codewizard as an MCP server in Codex CLI (OpenAI) configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		runRegisterCodex(global)
	},
}

var registerAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Register with all supported AI CLI tools",
	Run: func(cmd *cobra.Command, args []string) {
		global, _ := cmd.Flags().GetBool("global")
		runRegisterAll(global)
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize external data sources",
}

var syncGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Sync GitHub issues and PRs to memory",
	Long: `Fetch GitHub issues and pull requests using gh CLI and store them in memory.

Requires: gh CLI installed and authenticated (gh auth login)

Examples:
  mcp-codewizard sync github              # Sync open issues
  mcp-codewizard sync github --all        # Sync all issues (open + closed)
  mcp-codewizard sync github --prs        # Include pull requests
  mcp-codewizard sync github --limit 50   # Limit to 50 items`,
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		prs, _ := cmd.Flags().GetBool("prs")
		limit, _ := cmd.Flags().GetInt("limit")
		labels, _ := cmd.Flags().GetStringSlice("labels")
		assignee, _ := cmd.Flags().GetString("assignee")

		runSyncGithub(all, prs, limit, labels, assignee)
	},
}

// Call command - generic MCP tool invocation
var callCmd = &cobra.Command{
	Use:   "call <tool> [json-args]",
	Short: "Call any MCP tool directly (for debugging)",
	Long: `Call any MCP tool by name with JSON arguments.

Examples:
  mcp-codewizard call get_callers '{"symbol": "main"}'
  mcp-codewizard call get_dead_code '{"type": "functions", "limit": 10}'
  mcp-codewizard call get_symbols '{"kind": "function", "min_lines": 50}'
  mcp-codewizard call search_history '{"query": "fix bug"}'`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		tool := args[0]
		jsonArgs := "{}"
		if len(args) > 1 {
			jsonArgs = args[1]
		}
		runCall(tool, jsonArgs)
	},
}

// Chunk command
var chunkCmd = &cobra.Command{
	Use:   "chunk <id>",
	Short: "Get chunk content with context",
	Long: `Get a specific chunk by ID with surrounding context lines.

Examples:
  mcp-codewizard chunk "main.go:10:abc123"
  mcp-codewizard chunk "main.go:10:abc123" --context 10`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextLines, _ := cmd.Flags().GetInt("context")
		runChunk(args[0], contextLines)
	},
}

// Clear command
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the index",
	Long:  `Remove all indexed data. This will require re-indexing.`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		runClear(force)
	},
}

// Analysis commands
var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "Code analysis tools",
	Long:  `Various code analysis tools: callers, callees, symbols, dead code, complexity, etc.`,
}

var analysisCallersCmd = &cobra.Command{
	Use:   "callers <symbol>",
	Short: "Find all callers of a symbol",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisCallers(args[0], limit)
	},
}

var analysisCalleesCmd = &cobra.Command{
	Use:   "callees <symbol>",
	Short: "Find all functions called by a symbol",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisCallees(args[0], limit)
	},
}

var analysisSymbolsCmd = &cobra.Command{
	Use:   "symbols",
	Short: "List symbols with filters",
	Run: func(cmd *cobra.Command, args []string) {
		kind, _ := cmd.Flags().GetString("kind")
		minLines, _ := cmd.Flags().GetInt("min-lines")
		sortBy, _ := cmd.Flags().GetString("sort")
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisSymbols(kind, minLines, sortBy, limit)
	},
}

var analysisDeadCodeCmd = &cobra.Command{
	Use:   "dead-code",
	Short: "Find potentially unused code",
	Run: func(cmd *cobra.Command, args []string) {
		codeType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisDeadCode(codeType, limit)
	},
}

var analysisRefactoringCmd = &cobra.Command{
	Use:   "refactoring",
	Short: "Find refactoring candidates",
	Run: func(cmd *cobra.Command, args []string) {
		minLines, _ := cmd.Flags().GetInt("min-lines")
		maxComplexity, _ := cmd.Flags().GetInt("max-complexity")
		maxNesting, _ := cmd.Flags().GetInt("max-nesting")
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisRefactoring(minLines, maxComplexity, maxNesting, limit)
	},
}

var analysisComplexityCmd = &cobra.Command{
	Use:   "complexity <file>",
	Short: "Analyze code complexity",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startLine, _ := cmd.Flags().GetInt("start-line")
		endLine, _ := cmd.Flags().GetInt("end-line")
		runAnalysisComplexity(args[0], startLine, endLine)
	},
}

var analysisEntryPointsCmd = &cobra.Command{
	Use:   "entry-points",
	Short: "Find entry points (main, handlers, etc.)",
	Run: func(cmd *cobra.Command, args []string) {
		entryType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisEntryPoints(entryType, limit)
	},
}

var analysisImportsCmd = &cobra.Command{
	Use:   "imports",
	Short: "Show import/dependency graph",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runAnalysisImports(limit)
	},
}

// Git commands
var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git history analysis tools",
	Long:  `Tools for analyzing git history: search commits, blame, evolution, regression finding.`,
}

var gitIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index git history for semantic search",
	Run: func(cmd *cobra.Command, args []string) {
		since, _ := cmd.Flags().GetString("since")
		force, _ := cmd.Flags().GetBool("force")
		runGitIndex(since, force)
	},
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show git history index status",
	Run: func(cmd *cobra.Command, args []string) {
		runGitStatus()
	},
}

var gitSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Semantic search in git history",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		includeDiff, _ := cmd.Flags().GetBool("diff")
		runGitSearch(args[0], limit, includeDiff)
	},
}

var gitBlameCmd = &cobra.Command{
	Use:   "blame <file>",
	Short: "Show blame information for a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startLine, _ := cmd.Flags().GetInt("start-line")
		endLine, _ := cmd.Flags().GetInt("end-line")
		runGitBlame(args[0], startLine, endLine)
	},
}

var gitEvolutionCmd = &cobra.Command{
	Use:   "evolution <symbol>",
	Short: "Show how a symbol evolved over time",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runGitEvolution(args[0], limit)
	},
}

var gitRegressionCmd = &cobra.Command{
	Use:   "regression <pattern>",
	Short: "Find commits that may have introduced a bug",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		knownGood, _ := cmd.Flags().GetString("good")
		knownBad, _ := cmd.Flags().GetString("bad")
		limit, _ := cmd.Flags().GetInt("limit")
		runGitRegression(args[0], knownGood, knownBad, limit)
	},
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit <hash>",
	Short: "Show commit details and context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		includeDiff, _ := cmd.Flags().GetBool("diff")
		runGitCommit(args[0], includeDiff)
	},
}

var gitContributorsCmd = &cobra.Command{
	Use:   "contributors",
	Short: "Show contributor insights",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		runGitContributors(file)
	},
}

var gitHistoryCmd = &cobra.Command{
	Use:   "history <chunk-or-symbol>",
	Short: "Show change history for a chunk or symbol",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runGitHistory(args[0], limit)
	},
}

// Todo commands
var todoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Task management",
	Long:  `Manage tasks and todos with priorities and statuses.`,
}

var todoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List todos",
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetStringSlice("status")
		priority, _ := cmd.Flags().GetString("priority")
		limit, _ := cmd.Flags().GetInt("limit")
		includeCompleted, _ := cmd.Flags().GetBool("completed")
		runTodoList(status, priority, limit, includeCompleted)
	},
}

var todoAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new todo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetString("priority")
		parent, _ := cmd.Flags().GetString("parent")
		file, _ := cmd.Flags().GetString("file")
		line, _ := cmd.Flags().GetInt("line")
		runTodoAdd(args[0], description, priority, parent, file, line)
	},
}

var todoSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search todos semantically",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		runTodoSearch(args[0], limit)
	},
}

var todoUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a todo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")
		progress, _ := cmd.Flags().GetInt("progress")
		title, _ := cmd.Flags().GetString("title")
		runTodoUpdate(args[0], status, priority, progress, title)
	},
}

var todoCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a todo as completed",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runTodoComplete(args[0])
	},
}

var todoDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a todo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runTodoDelete(args[0])
	},
}

var fuzzyCmd = &cobra.Command{
	Use:   "fuzzy <query>",
	Short: "Fuzzy search for symbols or files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kind, _ := cmd.Flags().GetString("kind")
		searchType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		runFuzzySearch(args[0], kind, searchType, limit)
	},
}

var summaryCmd = &cobra.Command{
	Use:   "summary <file>",
	Short: "Get file summary (imports, exports, functions, complexity)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		quick, _ := cmd.Flags().GetBool("quick")
		runFileSummary(args[0], quick)
	},
}

var todoStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show todo statistics",
	Run: func(cmd *cobra.Command, args []string) {
		runTodoStats()
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

	initCmd.Flags().String("preset", "", "use preset (recommended, quality, fast)")
	initCmd.Flags().Bool("no-index", false, "skip indexing after init")
	initCmd.Flags().Bool("json", false, "output as JSON (for MCP integration)")

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

	// Register command flags
	registerClaudeCodeCmd.Flags().BoolP("global", "g", false, "register globally (user-level config)")
	registerGeminiCmd.Flags().BoolP("global", "g", false, "register globally (user-level config)")
	registerCodexCmd.Flags().BoolP("global", "g", false, "register globally (user-level config)")
	registerAllCmd.Flags().BoolP("global", "g", false, "register globally (user-level config)")

	registerCmd.AddCommand(registerClaudeCodeCmd)
	registerCmd.AddCommand(registerGeminiCmd)
	registerCmd.AddCommand(registerCodexCmd)
	registerCmd.AddCommand(registerAllCmd)

	// Sync command flags
	syncGithubCmd.Flags().Bool("all", false, "sync all issues (open + closed)")
	syncGithubCmd.Flags().Bool("prs", false, "include pull requests")
	syncGithubCmd.Flags().IntP("limit", "l", 100, "maximum items to sync")
	syncGithubCmd.Flags().StringSlice("labels", nil, "filter by labels")
	syncGithubCmd.Flags().StringP("assignee", "a", "", "filter by assignee")

	syncCmd.AddCommand(syncGithubCmd)

	// Chunk command flags
	chunkCmd.Flags().IntP("context", "c", 5, "number of context lines")

	// Clear command flags
	clearCmd.Flags().BoolP("force", "f", false, "force clear without confirmation")

	// Analysis command flags
	analysisCallersCmd.Flags().IntP("limit", "l", 50, "maximum results")
	analysisCalleesCmd.Flags().IntP("limit", "l", 50, "maximum results")
	analysisSymbolsCmd.Flags().StringP("kind", "k", "", "filter by kind (function, type, method, variable)")
	analysisSymbolsCmd.Flags().Int("min-lines", 0, "minimum line count")
	analysisSymbolsCmd.Flags().StringP("sort", "s", "", "sort by (lines, name)")
	analysisSymbolsCmd.Flags().IntP("limit", "l", 50, "maximum results")
	analysisDeadCodeCmd.Flags().StringP("type", "t", "functions", "type of dead code (functions, types)")
	analysisDeadCodeCmd.Flags().IntP("limit", "l", 20, "maximum results")
	analysisRefactoringCmd.Flags().Int("min-lines", 50, "minimum line count")
	analysisRefactoringCmd.Flags().Int("max-complexity", 10, "maximum acceptable complexity")
	analysisRefactoringCmd.Flags().Int("max-nesting", 4, "maximum acceptable nesting")
	analysisRefactoringCmd.Flags().IntP("limit", "l", 20, "maximum results")
	analysisComplexityCmd.Flags().Int("start-line", 0, "start line")
	analysisComplexityCmd.Flags().Int("end-line", 0, "end line")
	analysisEntryPointsCmd.Flags().StringP("type", "t", "", "filter by type (main, handler, init)")
	analysisEntryPointsCmd.Flags().IntP("limit", "l", 100, "maximum results")
	analysisImportsCmd.Flags().IntP("limit", "l", 1000, "maximum edges")

	analysisCmd.AddCommand(analysisCallersCmd)
	analysisCmd.AddCommand(analysisCalleesCmd)
	analysisCmd.AddCommand(analysisSymbolsCmd)
	analysisCmd.AddCommand(analysisDeadCodeCmd)
	analysisCmd.AddCommand(analysisRefactoringCmd)
	analysisCmd.AddCommand(analysisComplexityCmd)
	analysisCmd.AddCommand(analysisEntryPointsCmd)
	analysisCmd.AddCommand(analysisImportsCmd)

	// Git command flags
	gitIndexCmd.Flags().String("since", "30 days ago", "index commits since this date")
	gitIndexCmd.Flags().Bool("force", false, "force reindex")
	gitSearchCmd.Flags().IntP("limit", "l", 10, "maximum results")
	gitSearchCmd.Flags().Bool("diff", false, "include diff in results")
	gitBlameCmd.Flags().Int("start-line", 0, "start line")
	gitBlameCmd.Flags().Int("end-line", 0, "end line")
	gitEvolutionCmd.Flags().IntP("limit", "l", 20, "maximum commits")
	gitRegressionCmd.Flags().String("good", "", "known good commit")
	gitRegressionCmd.Flags().String("bad", "HEAD", "known bad commit")
	gitRegressionCmd.Flags().IntP("limit", "l", 10, "maximum results")
	gitCommitCmd.Flags().Bool("diff", false, "include diff")
	gitContributorsCmd.Flags().StringP("file", "f", "", "filter by file path")
	gitHistoryCmd.Flags().IntP("limit", "l", 20, "maximum commits")

	gitCmd.AddCommand(gitIndexCmd)
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitSearchCmd)
	gitCmd.AddCommand(gitBlameCmd)
	gitCmd.AddCommand(gitEvolutionCmd)
	gitCmd.AddCommand(gitRegressionCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitContributorsCmd)
	gitCmd.AddCommand(gitHistoryCmd)

	// Todo command flags
	todoListCmd.Flags().StringSliceP("status", "s", nil, "filter by status (pending, in_progress, done, cancelled)")
	todoListCmd.Flags().StringP("priority", "p", "", "filter by priority (urgent, high, medium, low)")
	todoListCmd.Flags().IntP("limit", "l", 20, "maximum results")
	todoListCmd.Flags().Bool("completed", false, "include completed todos")
	todoAddCmd.Flags().StringP("description", "d", "", "task description")
	todoAddCmd.Flags().StringP("priority", "p", "medium", "priority (urgent, high, medium, low)")
	todoAddCmd.Flags().String("parent", "", "parent todo ID")
	todoAddCmd.Flags().StringP("file", "f", "", "related file path")
	todoAddCmd.Flags().Int("line", 0, "line number in file")
	todoSearchCmd.Flags().IntP("limit", "l", 10, "maximum results")
	todoUpdateCmd.Flags().StringP("status", "s", "", "new status")
	todoUpdateCmd.Flags().StringP("priority", "p", "", "new priority")
	todoUpdateCmd.Flags().Int("progress", -1, "progress percentage (0-100)")
	todoUpdateCmd.Flags().StringP("title", "t", "", "new title")

	todoCmd.AddCommand(todoListCmd)
	todoCmd.AddCommand(todoAddCmd)
	todoCmd.AddCommand(todoSearchCmd)
	todoCmd.AddCommand(todoUpdateCmd)
	todoCmd.AddCommand(todoCompleteCmd)
	todoCmd.AddCommand(todoDeleteCmd)
	todoCmd.AddCommand(todoStatsCmd)

	// Fuzzy search flags
	fuzzyCmd.Flags().StringP("kind", "k", "", "symbol kind filter (function, type, method, variable)")
	fuzzyCmd.Flags().StringP("type", "t", "symbols", "search type (symbols, files)")
	fuzzyCmd.Flags().IntP("limit", "l", 20, "maximum results")

	// Summary flags
	summaryCmd.Flags().BoolP("quick", "q", false, "quick summary (only line counts)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(detectCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(chunkCmd)
	rootCmd.AddCommand(clearCmd)
	rootCmd.AddCommand(analysisCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(todoCmd)
	rootCmd.AddCommand(fuzzyCmd)
	rootCmd.AddCommand(summaryCmd)
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
			BaseURL:   cfg.Embedding.Endpoint,
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

func runInit(path string, preset string, skipIndex bool, jsonOutput bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		slog.Error("invalid path", "error", err)
		os.Exit(1)
	}

	// Check if config already exists
	configPath := config.ConfigPath(absPath)
	if _, err := os.Stat(configPath); err == nil {
		if !jsonOutput {
			fmt.Printf("Config already exists at %s\n", configPath)
			fmt.Print("Overwrite? (y/N): ")
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Aborted.")
				return
			}
		}
	}

	wiz := wizard.New(absPath)
	ctx := context.Background()

	// JSON output mode (for MCP integration)
	if jsonOutput {
		fmt.Println("Detecting environment...")
		env, err := wiz.DetectEnvironment(ctx)
		if err != nil {
			slog.Error("detection failed", "error", err)
			os.Exit(1)
		}

		state := wizard.InitWizardState{
			Environment: env,
			Options:     wiz.GetInitOptions(env),
			Ready:       false,
		}

		if preset != "" {
			cfg := wiz.ApplyPreset(env, preset)
			state.Config = cfg
			state.Selections = map[string]string{"preset": preset}
			state.Ready = true
		}

		output, _ := json.MarshalIndent(state, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Interactive mode - start with AI provider setup
	fmt.Println("\n=== mcp-codewizard Setup ===")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cfg := config.DefaultConfig()

	// Quick environment detection for display
	fmt.Print("Detecting environment... ")
	env, _ := wiz.DetectEnvironment(ctx)
	fmt.Println("done")
	fmt.Println()

	// If preset specified, use it directly
	if preset != "" {
		cfg = wiz.ApplyPreset(env, preset)
		fmt.Printf("Using '%s' preset.\n", preset)
	} else {
		// Step 1: Choose AI Provider
		fmt.Println("Step 1: AI Provider")
		fmt.Println("-------------------")
		fmt.Println("Which embedding provider do you want to use?")
		fmt.Println()

		// Show provider options with detection status
		ollamaStatus := "not detected"
		ollamaMarker := "  "
		if env.Ollama.Available {
			ollamaStatus = fmt.Sprintf("running at %s", env.Ollama.Endpoint)
			ollamaMarker = "* "
		}

		openaiStatus := "OPENAI_API_KEY not set"
		openaiMarker := "  "
		if env.OpenAI.Available {
			openaiStatus = "API key configured"
			if !env.Ollama.Available {
				openaiMarker = "* "
			}
		}

		fmt.Printf("  %s[1] Ollama - %s\n", ollamaMarker, ollamaStatus)
		fmt.Printf("  %s[2] OpenAI - %s\n", openaiMarker, openaiStatus)
		fmt.Println("    [3] Custom endpoint")
		fmt.Println()

		defaultProvider := 1
		if !env.Ollama.Available && env.OpenAI.Available {
			defaultProvider = 2
		}
		fmt.Printf("Select provider [%d]: ", defaultProvider)

		providerInput, _ := reader.ReadString('\n')
		providerInput = strings.TrimSpace(providerInput)
		providerSelection := defaultProvider
		if providerInput != "" {
			fmt.Sscanf(providerInput, "%d", &providerSelection)
		}

		var provider string
		var endpoint string
		var apiKey string
		var connResult *wizard.ConnectionTestResult

		switch providerSelection {
		case 2:
			provider = "openai"
			endpoint = "https://api.openai.com/v1"
		case 3:
			provider = "openai" // Custom uses OpenAI-compatible API
			endpoint = ""
		default:
			provider = "ollama"
			endpoint = "http://localhost:11434"
		}

		fmt.Println()

		// Step 2: Configure endpoint
		fmt.Println("Step 2: API Endpoint")
		fmt.Println("--------------------")

		if provider == "ollama" {
			fmt.Printf("Ollama endpoint [%s]: ", endpoint)
			endpointInput, _ := reader.ReadString('\n')
			endpointInput = strings.TrimSpace(endpointInput)
			if endpointInput != "" {
				endpoint = endpointInput
			}

			// Test connection
			fmt.Print("Testing connection... ")
			connResult = wiz.TestOllamaConnection(ctx, endpoint)
			if connResult.Connected {
				fmt.Println("✓ Connected")
			} else {
				fmt.Printf("✗ Failed: %s\n", connResult.Error)
				fmt.Println("You can continue with this endpoint, but indexing may fail.")
			}
		} else {
			if endpoint == "" {
				fmt.Print("API endpoint URL: ")
				endpointInput, _ := reader.ReadString('\n')
				endpoint = strings.TrimSpace(endpointInput)
				if endpoint == "" {
					endpoint = "https://api.openai.com/v1"
				}
			} else {
				fmt.Printf("OpenAI endpoint [%s]: ", endpoint)
				endpointInput, _ := reader.ReadString('\n')
				endpointInput = strings.TrimSpace(endpointInput)
				if endpointInput != "" {
					endpoint = endpointInput
				}
			}

			// Get API key
			fmt.Println()
			if env.OpenAI.Available {
				fmt.Print("API Key [press Enter to use OPENAI_API_KEY]: ")
			} else {
				fmt.Print("API Key: ")
			}
			apiKeyInput, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(apiKeyInput)
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}

			// Test connection
			fmt.Print("Testing connection... ")
			connResult = wiz.TestOpenAIConnection(ctx, endpoint, apiKey)
			if connResult.Connected {
				fmt.Println("✓ Connected")
			} else {
				fmt.Printf("✗ Failed: %s\n", connResult.Error)
				fmt.Println("You can continue, but indexing may fail.")
			}
		}

		fmt.Println()

		// Step 3: Select embedding model
		fmt.Println("Step 3: Embedding Model")
		fmt.Println("-----------------------")

		var selectedModel string
		embeddingModels := connResult.GetEmbeddingModels()

		if len(embeddingModels) > 0 {
			fmt.Println("Available embedding models:")
			fmt.Println()

			defaultModelIdx := 0
			for i, m := range embeddingModels {
				marker := "  "
				if m.Recommended {
					marker = "* "
					defaultModelIdx = i
				}
				sizeInfo := ""
				if m.Size != "" {
					sizeInfo = fmt.Sprintf(" (%s)", m.Size)
				}
				fmt.Printf("  %s[%d] %s%s\n", marker, i+1, m.Name, sizeInfo)
			}
			fmt.Println()
			fmt.Printf("Select model [%d]: ", defaultModelIdx+1)

			modelInput, _ := reader.ReadString('\n')
			modelInput = strings.TrimSpace(modelInput)
			modelSelection := defaultModelIdx + 1
			if modelInput != "" {
				fmt.Sscanf(modelInput, "%d", &modelSelection)
			}

			if modelSelection >= 1 && modelSelection <= len(embeddingModels) {
				selectedModel = embeddingModels[modelSelection-1].Name
			} else {
				selectedModel = embeddingModels[defaultModelIdx].Name
			}
		} else {
			// No models detected, ask for manual input
			defaultModel := "nomic-embed-code"
			if provider == "openai" {
				defaultModel = "text-embedding-3-small"
			}
			fmt.Printf("Model name [%s]: ", defaultModel)
			modelInput, _ := reader.ReadString('\n')
			selectedModel = strings.TrimSpace(modelInput)
			if selectedModel == "" {
				selectedModel = defaultModel
			}
		}

		fmt.Println()

		// Step 4: Reranker (optional)
		fmt.Println("Step 4: Reranker (Optional)")
		fmt.Println("---------------------------")
		fmt.Println("Reranking improves search accuracy by ~15% but uses additional resources.")
		fmt.Println()

		enableReranker := false
		rerankerModel := ""

		if provider == "ollama" && connResult.Connected {
			rerankerModels := connResult.GetRerankerModels()
			if len(rerankerModels) > 0 {
				fmt.Println("Available reranker models:")
				for i, m := range rerankerModels {
					marker := "  "
					if m.Recommended {
						marker = "* "
					}
					fmt.Printf("  %s[%d] %s (%s)\n", marker, i+1, m.Name, m.Size)
				}
				fmt.Printf("    [%d] Disable reranker\n", len(rerankerModels)+1)
				fmt.Println()
				fmt.Print("Select [1]: ")

				rerankerInput, _ := reader.ReadString('\n')
				rerankerInput = strings.TrimSpace(rerankerInput)
				rerankerSelection := 1
				if rerankerInput != "" {
					fmt.Sscanf(rerankerInput, "%d", &rerankerSelection)
				}

				if rerankerSelection >= 1 && rerankerSelection <= len(rerankerModels) {
					enableReranker = true
					rerankerModel = rerankerModels[rerankerSelection-1].Name
				}
			} else {
				fmt.Println("No reranker models found. You can install one with:")
				fmt.Println("  ollama pull qwen3-reranker")
				fmt.Println()
				fmt.Print("Enable reranker anyway? (y/N): ")
				rerankerInput, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(rerankerInput)) == "y" {
					enableReranker = true
					rerankerModel = "qwen3-reranker"
				}
			}
		} else {
			fmt.Println("Reranker requires Ollama. Skipping.")
		}

		fmt.Println()

		// Apply configuration
		cfg.Embedding.Provider = provider
		cfg.Embedding.Endpoint = endpoint
		cfg.Embedding.Model = selectedModel
		if apiKey != "" && apiKey != os.Getenv("OPENAI_API_KEY") {
			cfg.Embedding.APIKey = apiKey
		}

		cfg.Reranker.Enabled = enableReranker
		if enableReranker {
			cfg.Reranker.Provider = "ollama"
			cfg.Reranker.Model = rerankerModel
			cfg.Reranker.Endpoint = endpoint
		}

		// Use sensible defaults for other settings
		cfg.Chunking.Strategy = "treesitter"
		cfg.Search.Mode = "hybrid"
	}

	// Show summary
	fmt.Println("=== Configuration Summary ===")
	fmt.Println()
	fmt.Println(wizard.FormatConfigSummary(cfg))

	// Save configuration
	if err := config.Save(absPath, cfg); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration saved to %s\n", configPath)

	// Optionally start indexing
	if !skipIndex {
		fmt.Println()
		fmt.Print("Start indexing now? (Y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response == "" || response == "y" || response == "Y" {
			fmt.Println()
			runIndex(absPath, false)
		}
	}
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
		_, _ = fmt.Scanln(&response)
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

// Register command implementations

// MCPServerConfig represents the MCP server configuration in JSON configs
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ClaudeCodeConfig represents Claude Code's MCP config structure
type ClaudeCodeConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// GeminiConfig represents Gemini CLI's MCP config structure
type GeminiConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// CodexConfig represents Codex CLI's MCP config structure
type CodexConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

func getBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func getClaudeCodeConfigPath(global bool) string {
	if global {
		home, _ := os.UserHomeDir()
		// Claude Code uses ~/.claude.json for user config
		return filepath.Join(home, ".claude.json")
	}
	// Project-level config uses .mcp.json
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".mcp.json")
}

func getGeminiConfigPath(global bool) string {
	if global {
		home, _ := os.UserHomeDir()
		// Gemini CLI uses ~/.gemini/settings.json
		return filepath.Join(home, ".gemini", "settings.json")
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".gemini.json")
}

func getCodexConfigPath(global bool) string {
	if global {
		home, _ := os.UserHomeDir()
		// Codex CLI uses ~/.codex/config.json
		return filepath.Join(home, ".codex", "config.json")
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".codex.json")
}

func loadOrCreateJSONConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return config, nil
}

func saveJSONConfig(path string, config map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func runRegisterClaudeCode(global bool) {
	binaryPath, err := getBinaryPath()
	if err != nil {
		slog.Error("failed to get binary path", "error", err)
		os.Exit(1)
	}

	configPath := getClaudeCodeConfigPath(global)
	config, err := loadOrCreateJSONConfig(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Get or create mcpServers
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Add mcp-codewizard server
	mcpServers["mcp-codewizard"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"serve", "--stdio"},
	}

	config["mcpServers"] = mcpServers

	if err := saveJSONConfig(configPath, config); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Registered mcp-codewizard with Claude Code\n")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("\nRestart Claude Code to apply changes.\n")
}

func runRegisterGemini(global bool) {
	binaryPath, err := getBinaryPath()
	if err != nil {
		slog.Error("failed to get binary path", "error", err)
		os.Exit(1)
	}

	configPath := getGeminiConfigPath(global)
	config, err := loadOrCreateJSONConfig(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Get or create mcpServers
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Add mcp-codewizard server
	mcpServers["mcp-codewizard"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"serve", "--stdio"},
	}

	config["mcpServers"] = mcpServers

	if err := saveJSONConfig(configPath, config); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Registered mcp-codewizard with Gemini CLI\n")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("\nRestart Gemini CLI to apply changes.\n")
}

func runRegisterCodex(global bool) {
	binaryPath, err := getBinaryPath()
	if err != nil {
		slog.Error("failed to get binary path", "error", err)
		os.Exit(1)
	}

	configPath := getCodexConfigPath(global)
	config, err := loadOrCreateJSONConfig(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Get or create mcpServers
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Add mcp-codewizard server
	mcpServers["mcp-codewizard"] = map[string]interface{}{
		"command": binaryPath,
		"args":    []string{"serve", "--stdio"},
	}

	config["mcpServers"] = mcpServers

	if err := saveJSONConfig(configPath, config); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Registered mcp-codewizard with Codex CLI\n")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("\nRestart Codex CLI to apply changes.\n")
}

func runRegisterAll(global bool) {
	fmt.Println("Registering mcp-codewizard with all supported AI CLI tools...")
	fmt.Println()

	// Try Claude Code
	fmt.Print("Claude Code: ")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("failed (%v)\n", r)
			}
		}()
		binaryPath, _ := getBinaryPath()
		configPath := getClaudeCodeConfigPath(global)
		config, _ := loadOrCreateJSONConfig(configPath)
		mcpServers, ok := config["mcpServers"].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}
		mcpServers["mcp-codewizard"] = map[string]interface{}{
			"command": binaryPath,
			"args":    []string{"serve", "--stdio"},
		}
		config["mcpServers"] = mcpServers
		if err := saveJSONConfig(configPath, config); err != nil {
			fmt.Printf("failed (%v)\n", err)
			return
		}
		fmt.Printf("OK (%s)\n", configPath)
	}()

	// Try Gemini
	fmt.Print("Gemini CLI:  ")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("failed (%v)\n", r)
			}
		}()
		binaryPath, _ := getBinaryPath()
		configPath := getGeminiConfigPath(global)
		config, _ := loadOrCreateJSONConfig(configPath)
		mcpServers, ok := config["mcpServers"].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}
		mcpServers["mcp-codewizard"] = map[string]interface{}{
			"command": binaryPath,
			"args":    []string{"serve", "--stdio"},
		}
		config["mcpServers"] = mcpServers
		if err := saveJSONConfig(configPath, config); err != nil {
			fmt.Printf("failed (%v)\n", err)
			return
		}
		fmt.Printf("OK (%s)\n", configPath)
	}()

	// Try Codex
	fmt.Print("Codex CLI:   ")
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("failed (%v)\n", r)
			}
		}()
		binaryPath, _ := getBinaryPath()
		configPath := getCodexConfigPath(global)
		config, _ := loadOrCreateJSONConfig(configPath)
		mcpServers, ok := config["mcpServers"].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}
		mcpServers["mcp-codewizard"] = map[string]interface{}{
			"command": binaryPath,
			"args":    []string{"serve", "--stdio"},
		}
		config["mcpServers"] = mcpServers
		if err := saveJSONConfig(configPath, config); err != nil {
			fmt.Printf("failed (%v)\n", err)
			return
		}
		fmt.Printf("OK (%s)\n", configPath)
	}()

	fmt.Println()
	fmt.Println("Restart the CLI tools to apply changes.")
}

// runSyncGithub syncs GitHub issues and PRs to memory.
func runSyncGithub(all, prs bool, limit int, labels []string, assignee string) {
	setupLogging()
	cwd, _ := os.Getwd()

	// Check if gh is available
	client := github.New()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		slog.Error("gh CLI not available or not authenticated")
		fmt.Println("Error: gh CLI is not installed or not authenticated.")
		fmt.Println("Install: https://cli.github.com/")
		fmt.Println("Authenticate: gh auth login")
		os.Exit(1)
	}

	// Get repo info
	repo, err := client.GetRepoInfo(ctx)
	if err != nil {
		slog.Error("failed to get repo info", "error", err)
		fmt.Println("Error: Not in a GitHub repository or failed to get repo info.")
		os.Exit(1)
	}

	fmt.Printf("Syncing GitHub issues from %s...\n", repo)

	// Build sync config
	cfg := github.SyncConfig{
		Limit:      limit,
		Labels:     labels,
		Assignee:   assignee,
		IncludePRs: prs,
	}

	if all {
		cfg.States = []string{"all"}
	} else {
		cfg.States = []string{"open"}
	}

	// Sync
	requests, result, err := client.Sync(ctx, cfg)
	if err != nil {
		slog.Error("sync failed", "error", err)
		os.Exit(1)
	}

	if len(requests) == 0 {
		fmt.Println("No issues or PRs found.")
		return
	}

	// Open memory store
	store, err := memory.NewStore(memory.StoreConfig{ProjectDir: cwd})
	if err != nil {
		slog.Error("failed to open memory store", "error", err)
		os.Exit(1)
	}

	// Store issues
	for _, req := range requests {
		if _, err := store.AddIssue(req); err != nil {
			slog.Warn("failed to store issue", "title", req.Title, "error", err)
		}
	}

	fmt.Printf("\nSynced %d issues", result.IssuesSynced)
	if prs {
		fmt.Printf(" and %d PRs", result.PRsSynced)
	}
	fmt.Println(" to memory.")

	if len(result.Errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	fmt.Println("\nUse 'mcp-codewizard memory issues' to view synced issues.")
}

// runCall invokes an MCP tool directly (for debugging)
// Note: This is a simplified implementation that maps tool names to direct function calls
func runCall(tool string, jsonArgs string) {
	cwd, _ := os.Getwd()

	// Parse JSON args
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(jsonArgs), &args); err != nil {
		slog.Error("invalid JSON arguments", "error", err)
		os.Exit(1)
	}

	// Load config
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

	// Initialize store
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Create search engine
	searchEngine := search.New(search.Config{
		Store:     store,
		Embedding: embedding,
		Reranker:  reranker,
	})

	// Create MCP server for tool handling
	mcpServer, err := mcp.New(mcp.Config{
		ProjectDir: cwd,
		Config:     cfg,
		Store:      store,
		Embedding:  embedding,
		Reranker:   reranker,
	})
	if err != nil {
		slog.Error("failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Map tool name to simplified command output
	// Since MCP doesn't expose CallTool directly, we handle common cases here
	fmt.Printf("Tool: %s\nArgs: %s\n\n", tool, jsonArgs)
	fmt.Println("Note: Direct tool invocation is limited. Use the specific subcommands instead:")
	fmt.Println("  analysis callers/callees/symbols/dead-code/complexity/entry-points/imports")
	fmt.Println("  git index/search/blame/evolution/regression/commit/contributors/history")
	fmt.Println("  todo list/add/search/update/complete/delete/stats")
	fmt.Println("  chunk <id>")
	fmt.Println()

	// Show search engine and MCP server are available
	_ = searchEngine
	_ = mcpServer
}

// runChunk gets a chunk with context
func runChunk(chunkID string, contextLines int) {
	cwd, _ := os.Getwd()

	// Create store
	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Get chunk
	chunk, err := store.GetChunk(chunkID)
	if err != nil {
		slog.Error("failed to get chunk", "id", chunkID, "error", err)
		os.Exit(1)
	}

	if chunk == nil {
		fmt.Println("Chunk not found:", chunkID)
		os.Exit(1)
	}

	// Print chunk info
	fmt.Printf("File: %s\n", chunk.FilePath)
	fmt.Printf("Lines: %d-%d\n", chunk.StartLine, chunk.EndLine)
	fmt.Printf("Type: %s\n", chunk.ChunkType)
	if chunk.Name != "" {
		fmt.Printf("Name: %s\n", chunk.Name)
	}
	fmt.Println()

	// Read file for context
	content, err := os.ReadFile(chunk.FilePath)
	if err != nil {
		// Just print chunk content without context
		fmt.Println(chunk.Content)
		return
	}

	lines := splitLines(string(content))
	startCtx := chunk.StartLine - contextLines - 1
	if startCtx < 0 {
		startCtx = 0
	}
	endCtx := chunk.EndLine + contextLines
	if endCtx > len(lines) {
		endCtx = len(lines)
	}

	for i := startCtx; i < endCtx; i++ {
		lineNum := i + 1
		prefix := "  "
		if lineNum >= chunk.StartLine && lineNum <= chunk.EndLine {
			prefix = "> "
		}
		fmt.Printf("%s%4d: %s\n", prefix, lineNum, lines[i])
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// runClear clears the index
func runClear(force bool) {
	cwd, _ := os.Getwd()

	if !force {
		fmt.Print("This will delete all indexed data. Continue? [y/N] ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}

	// Get all files and delete their chunks
	hashes, err := store.GetAllFileHashes()
	if err != nil {
		slog.Error("failed to get files", "error", err)
		store.Close()
		os.Exit(1)
	}

	for filePath := range hashes {
		if err := store.DeleteChunksByFile(filePath); err != nil {
			slog.Warn("failed to delete chunks", "file", filePath, "error", err)
		}
		if err := store.DeleteFileCache(filePath); err != nil {
			slog.Warn("failed to delete cache", "file", filePath, "error", err)
		}
	}

	store.Close()
	fmt.Println("Index cleared.")
}

// Analysis functions

func runAnalysisCallers(symbol string, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	refs, err := searchEngine.GetCallers(symbol, limit)
	if err != nil {
		slog.Error("failed to get callers", "error", err)
		os.Exit(1)
	}

	if len(refs) == 0 {
		fmt.Printf("No callers found for '%s'\n", symbol)
		return
	}

	fmt.Printf("Callers of '%s' (%d found):\n\n", symbol, len(refs))
	for _, ref := range refs {
		fmt.Printf("  %s:%d - %s\n", ref.FilePath, ref.Line, ref.FromSymbol)
	}
}

func runAnalysisCallees(symbol string, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	refs, err := searchEngine.GetCallees(symbol, limit)
	if err != nil {
		slog.Error("failed to get callees", "error", err)
		os.Exit(1)
	}

	if len(refs) == 0 {
		fmt.Printf("No callees found for '%s'\n", symbol)
		return
	}

	fmt.Printf("Functions called by '%s' (%d found):\n\n", symbol, len(refs))
	for _, ref := range refs {
		fmt.Printf("  %s:%d - %s\n", ref.FilePath, ref.Line, ref.ToSymbol)
	}
}

func runAnalysisSymbols(kind string, minLines int, sortBy string, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	symbols, err := searchEngine.SearchSymbolsAdvanced("", types.SymbolKind(kind), minLines, sortBy, limit)
	if err != nil {
		slog.Error("failed to get symbols", "error", err)
		os.Exit(1)
	}

	if len(symbols) == 0 {
		fmt.Println("No symbols found")
		return
	}

	fmt.Printf("Symbols (%d found):\n\n", len(symbols))
	for _, sym := range symbols {
		fmt.Printf("  %-12s %-40s %s:%d (%d lines)\n",
			sym.Kind, sym.Name, sym.FilePath, sym.StartLine, sym.LineCount)
	}
}

func runAnalysisDeadCode(codeType string, limit int) {
	cwd, _ := os.Getwd()

	// Create store for dead code analysis
	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Use analysis package for dead code detection
	deadCodeAnalyzer := analysis.NewDeadCodeAnalyzer(store)

	var results []*analysis.DeadCodeResult
	var err error

	switch codeType {
	case "functions":
		results, err = deadCodeAnalyzer.FindDeadFunctions(limit)
	case "types":
		results, err = deadCodeAnalyzer.FindUnusedTypes(limit)
	case "all":
		results, err = deadCodeAnalyzer.FindDeadCode(limit)
	default:
		results, err = deadCodeAnalyzer.FindDeadFunctions(limit)
	}

	if err != nil {
		slog.Error("failed to get dead code", "error", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No dead code candidates found")
		return
	}

	fmt.Printf("Potentially unused %s (%d found):\n\n", codeType, len(results))
	for _, dc := range results {
		fmt.Printf("  %-40s %s:%d (%s)\n", dc.Symbol.Name, dc.Symbol.FilePath, dc.Symbol.StartLine, dc.Reason)
	}
}

func runAnalysisRefactoring(minLines, maxComplexity, maxNesting, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	result := map[string]interface{}{
		"thresholds": map[string]int{
			"min_lines":      minLines,
			"max_complexity": maxComplexity,
			"max_nesting":    maxNesting,
		},
	}

	// Get long functions
	longFunctions, err := searchEngine.FindLongFunctions(minLines, limit)
	if err != nil {
		slog.Warn("failed to find long functions", "error", err)
	} else {
		var longFormatted []map[string]interface{}
		for _, sym := range longFunctions {
			longFormatted = append(longFormatted, map[string]interface{}{
				"name":       sym.Name,
				"file":       sym.FilePath,
				"start_line": sym.StartLine,
				"end_line":   sym.EndLine,
				"line_count": sym.LineCount,
			})
		}
		result["long_functions"] = longFormatted
	}

	// Get high complexity functions
	highComplexity, err := searchEngine.FindHighComplexityFunctions(maxComplexity, maxNesting, limit)
	if err == nil && len(highComplexity) > 0 {
		var complexFormatted []map[string]interface{}
		for _, item := range highComplexity {
			complexFormatted = append(complexFormatted, map[string]interface{}{
				"name":                  item.Name,
				"file":                  item.FilePath,
				"start_line":            item.StartLine,
				"end_line":              item.EndLine,
				"line_count":            item.LineCount,
				"cyclomatic_complexity": item.CyclomaticComplexity,
				"cognitive_complexity":  item.CognitiveComplexity,
				"max_nesting":           item.MaxNesting,
			})
		}
		result["high_complexity"] = complexFormatted
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonResult))
}

func runAnalysisComplexity(file string, startLine, endLine int) {
	cwd, _ := os.Getwd()
	absPath := file
	if !filepath.IsAbs(file) {
		absPath = filepath.Join(cwd, file)
	}

	// Use analysis package for complexity
	complexityAnalyzer := analysis.NewComplexityAnalyzer()

	var result interface{}
	var err error

	if startLine > 0 && endLine > 0 {
		result, err = complexityAnalyzer.AnalyzeRange(absPath, startLine, endLine)
	} else {
		result, err = complexityAnalyzer.AnalyzeFile(absPath)
	}

	if err != nil {
		slog.Error("failed to get complexity", "error", err)
		os.Exit(1)
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonResult))
}

func runAnalysisEntryPoints(entryType string, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	entryPoints, err := searchEngine.GetEntryPoints(limit)
	if err != nil {
		slog.Error("failed to get entry points", "error", err)
		os.Exit(1)
	}

	// Filter by type if specified
	if entryType != "" {
		var filtered []*search.EntryPoint
		for _, ep := range entryPoints {
			if ep.Type == entryType {
				filtered = append(filtered, ep)
			}
		}
		entryPoints = filtered
	}

	if len(entryPoints) == 0 {
		fmt.Println("No entry points found")
		return
	}

	fmt.Printf("Entry points (%d found):\n\n", len(entryPoints))
	for _, ep := range entryPoints {
		fmt.Printf("  %-12s %-40s %s:%d\n", ep.Type, ep.Symbol.Name, ep.Symbol.FilePath, ep.Symbol.StartLine)
	}
}

func runAnalysisImports(limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	graph, err := searchEngine.GetImportGraph(limit)
	if err != nil {
		slog.Error("failed to get import graph", "error", err)
		os.Exit(1)
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	fmt.Println(string(result))
}

// Helper to create search engine
func createSearchEngine() *search.Engine {
	cwd, _ := os.Getwd()

	cfg, _, err := config.Load(cwd)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
		return nil
	}

	store, embedding, _, reranker, err := createProviders(cfg)
	if err != nil {
		slog.Error("failed to create providers", "error", err)
		os.Exit(1)
		return nil
	}

	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
		return nil
	}

	searchEngine := search.New(search.Config{
		Store:     store,
		Embedding: embedding,
		Reranker:  reranker,
	})

	return searchEngine
}

// Git functions

func runGitIndex(since string, force bool) {
	runCall("index_git_history", fmt.Sprintf(`{"since": "%s", "force": %t}`, since, force))
}

func runGitStatus() {
	runCall("get_git_history_status", "{}")
}

func runGitSearch(query string, limit int, includeDiff bool) {
	args := fmt.Sprintf(`{"query": "%s", "limit": %d, "include_diff": %t}`, query, limit, includeDiff)
	runCall("search_history", args)
}

func runGitBlame(file string, startLine, endLine int) {
	cwd, _ := os.Getwd()
	absPath := file
	if !filepath.IsAbs(file) {
		absPath = filepath.Join(cwd, file)
	}
	args := fmt.Sprintf(`{"file": "%s", "start_line": %d, "end_line": %d}`, absPath, startLine, endLine)
	runCall("get_blame", args)
}

func runGitEvolution(symbol string, limit int) {
	args := fmt.Sprintf(`{"symbol": "%s", "limit": %d}`, symbol, limit)
	runCall("get_code_evolution", args)
}

func runGitRegression(pattern, knownGood, knownBad string, limit int) {
	args := fmt.Sprintf(`{"pattern": "%s", "known_good": "%s", "known_bad": "%s", "limit": %d}`,
		pattern, knownGood, knownBad, limit)
	runCall("find_regression", args)
}

func runGitCommit(hash string, includeDiff bool) {
	args := fmt.Sprintf(`{"commit": "%s", "include_diff": %t}`, hash, includeDiff)
	runCall("get_commit_context", args)
}

func runGitContributors(file string) {
	args := "{}"
	if file != "" {
		cwd, _ := os.Getwd()
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(cwd, file)
		}
		args = fmt.Sprintf(`{"file": "%s"}`, absPath)
	}
	runCall("get_contributor_insights", args)
}

func runGitHistory(target string, limit int) {
	// Try as chunk ID first, then as symbol
	args := fmt.Sprintf(`{"chunk_id": "%s", "limit": %d}`, target, limit)
	runCall("get_chunk_history", args)
}

// Todo functions

func runTodoList(statuses []string, priority string, limit int, includeCompleted bool) {
	cwd, _ := os.Getwd()

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Convert statuses
	var todoStatuses []types.TodoStatus
	for _, s := range statuses {
		todoStatuses = append(todoStatuses, types.TodoStatus(s))
	}

	// Include completed if flag is set
	if includeCompleted && len(todoStatuses) == 0 {
		todoStatuses = []types.TodoStatus{
			types.TodoStatusPending,
			types.TodoStatusInProgress,
			types.TodoStatusCompleted,
		}
	}

	// List todos
	ctx := context.Background()
	todos, err := store.ListTodos(ctx, "", todoStatuses, limit)
	if err != nil {
		slog.Error("failed to list todos", "error", err)
		os.Exit(1)
	}

	// Filter by priority if specified
	if priority != "" {
		var filtered []*types.TodoItem
		for _, todo := range todos {
			if todo.Priority == types.TodoPriority(priority) {
				filtered = append(filtered, todo)
			}
		}
		todos = filtered
	}

	if len(todos) == 0 {
		fmt.Println("No todos found")
		return
	}

	fmt.Printf("Todos (%d found):\n\n", len(todos))
	for _, todo := range todos {
		statusIcon := "○"
		switch todo.Status {
		case types.TodoStatusInProgress:
			statusIcon = "◐"
		case types.TodoStatusCompleted:
			statusIcon = "●"
		case types.TodoStatusCancelled:
			statusIcon = "✗"
		case types.TodoStatusBlocked:
			statusIcon = "⊘"
		}
		prioIcon := ""
		switch todo.Priority {
		case types.TodoPriorityUrgent:
			prioIcon = "!!!"
		case types.TodoPriorityHigh:
			prioIcon = "!!"
		case types.TodoPriorityMedium:
			prioIcon = "!"
		}
		fmt.Printf("  %s %-3s [%s] %s\n", statusIcon, prioIcon, todo.ID, todo.Title)
		if todo.Progress > 0 {
			fmt.Printf("      Progress: %d%%\n", todo.Progress)
		}
	}
}

func runTodoAdd(title, description, priority, parent, file string, line int) {
	cwd, _ := os.Getwd()

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Generate a unique ID
	todoID := fmt.Sprintf("todo_%d", time.Now().UnixNano())

	todo := &types.TodoItem{
		ID:          todoID,
		Title:       title,
		Description: description,
		Priority:    types.TodoPriority(priority),
		Status:      types.TodoStatusPending,
		ParentID:    parent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add file reference as related chunk if provided
	if file != "" {
		absFile := file
		if !filepath.IsAbs(file) {
			absFile = filepath.Join(cwd, file)
		}
		chunkRef := absFile
		if line > 0 {
			chunkRef = fmt.Sprintf("%s:%d", absFile, line)
		}
		todo.RelatedChunks = []string{chunkRef}
	}

	if err := store.StoreTodo(todo); err != nil {
		slog.Error("failed to create todo", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Created todo: %s\n", todo.ID)
}

func runTodoSearch(query string, limit int) {
	args := fmt.Sprintf(`{"query": "%s", "limit": %d}`, query, limit)
	runCall("todo_search", args)
}

func runTodoUpdate(id, status, priority string, progress int, title string) {
	cwd, _ := os.Getwd()

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	todo, err := store.GetTodo(id)
	if err != nil {
		slog.Error("failed to get todo", "error", err)
		os.Exit(1)
	}

	if todo == nil {
		fmt.Printf("Todo not found: %s\n", id)
		os.Exit(1)
	}

	if status != "" {
		todo.Status = types.TodoStatus(status)
	}
	if priority != "" {
		todo.Priority = types.TodoPriority(priority)
	}
	if progress >= 0 {
		todo.Progress = progress
	}
	if title != "" {
		todo.Title = title
	}
	todo.UpdatedAt = time.Now()

	if err := store.UpdateTodo(todo); err != nil {
		slog.Error("failed to update todo", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Updated todo: %s\n", id)
}

func runTodoComplete(id string) {
	runTodoUpdate(id, string(types.TodoStatusCompleted), "", -1, "")
}

func runTodoDelete(id string) {
	cwd, _ := os.Getwd()

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.DeleteTodo(id); err != nil {
		slog.Error("failed to delete todo", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted todo: %s\n", id)
}

func runTodoStats() {
	cwd, _ := os.Getwd()

	store := sqlitevec.New()
	dbPath := config.IndexDBPath(cwd)
	if err := store.Init(dbPath); err != nil {
		slog.Error("failed to init store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	stats, err := store.GetTodoStats("")
	if err != nil {
		slog.Error("failed to get todo stats", "error", err)
		os.Exit(1)
	}

	jsonResult, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Println(string(jsonResult))
}

func runFuzzySearch(query, kind, searchType string, limit int) {
	searchEngine := createSearchEngine()
	if searchEngine == nil {
		return
	}

	if searchType == "files" {
		files, err := searchEngine.FuzzySearchFiles(query, limit)
		if err != nil {
			slog.Error("fuzzy file search failed", "error", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Println("No matching files found")
			return
		}

		fmt.Printf("Files matching '%s' (%d found):\n\n", query, len(files))
		for _, f := range files {
			fmt.Printf("  %s\n", f)
		}
		return
	}

	// Symbol search
	var symbolKind types.SymbolKind
	if kind != "" {
		symbolKind = types.SymbolKind(kind)
	}

	matches, err := searchEngine.FuzzySearchSymbols(query, symbolKind, limit)
	if err != nil {
		slog.Error("fuzzy search failed", "error", err)
		os.Exit(1)
	}

	if len(matches) == 0 {
		fmt.Println("No matching symbols found")
		return
	}

	fmt.Printf("Symbols matching '%s' (%d found):\n\n", query, len(matches))
	for _, m := range matches {
		fmt.Printf("  [%.0f%% %s] %-12s %-40s %s:%d\n",
			m.Score*100, m.MatchType, m.Symbol.Kind, m.Symbol.Name,
			m.Symbol.FilePath, m.Symbol.StartLine)
	}
}

func runFileSummary(file string, quick bool) {
	cwd, _ := os.Getwd()

	// Resolve file path
	filePath := filepath.Join(cwd, file)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if _, err := os.Stat(file); err == nil {
			filePath = file
		} else {
			slog.Error("file not found", "file", file)
			os.Exit(1)
		}
	}

	analyzer := analysis.NewSummaryAnalyzer()

	var summary *analysis.FileSummary
	var err error

	if quick {
		summary, err = analyzer.QuickSummary(filePath)
	} else {
		summary, err = analyzer.AnalyzeFile(filePath)
	}

	if err != nil {
		slog.Error("failed to analyze file", "error", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("File: %s\n", summary.FileName)
	fmt.Printf("Path: %s\n", summary.FilePath)
	fmt.Printf("Language: %s\n", summary.Language)
	fmt.Printf("Size: %d bytes\n", summary.Size)
	fmt.Printf("Modified: %s\n\n", summary.ModifiedAt.Format("2006-01-02 15:04:05"))

	fmt.Printf("Lines:\n")
	fmt.Printf("  Total:    %d\n", summary.TotalLines)
	fmt.Printf("  Code:     %d\n", summary.CodeLines)
	fmt.Printf("  Comments: %d\n", summary.CommentLines)
	fmt.Printf("  Blank:    %d\n\n", summary.BlankLines)

	if quick {
		return
	}

	// Detailed info
	if len(summary.Imports) > 0 {
		fmt.Printf("Imports (%d):\n", len(summary.Imports))
		for _, imp := range summary.Imports {
			local := ""
			if imp.IsLocal {
				local = " [local]"
			}
			fmt.Printf("  %s%s\n", imp.Path, local)
		}
		fmt.Println()
	}

	if len(summary.Exports) > 0 {
		fmt.Printf("Exports (%d):\n", len(summary.Exports))
		for _, exp := range summary.Exports {
			fmt.Printf("  %-10s %s (line %d)\n", exp.Kind, exp.Name, exp.Line)
		}
		fmt.Println()
	}

	if len(summary.Functions) > 0 {
		fmt.Printf("Functions (%d):\n", len(summary.Functions))
		fmt.Printf("  Average lines: %.1f\n", summary.AverageFunction)
		fmt.Printf("  Longest: %d lines\n\n", summary.LongestFunction)

		for _, fn := range summary.Functions {
			exported := ""
			if fn.IsExported {
				exported = " [exported]"
			}
			fmt.Printf("  %-40s %4d lines  complexity: %d%s\n",
				fn.Name, fn.LineCount, fn.Complexity, exported)
		}
		fmt.Println()
	}

	if len(summary.Types) > 0 {
		fmt.Printf("Types (%d):\n", len(summary.Types))
		for _, t := range summary.Types {
			exported := ""
			if t.IsExported {
				exported = " [exported]"
			}
			fmt.Printf("  %-10s %s (line %d)%s\n", t.Kind, t.Name, t.StartLine, exported)
		}
		fmt.Println()
	}

	if summary.Complexity != nil {
		fmt.Printf("Complexity:\n")
		fmt.Printf("  Cyclomatic:  %d\n", summary.Complexity.CyclomaticComplexity)
		fmt.Printf("  Cognitive:   %d\n", summary.Complexity.CognitiveComplexity)
		fmt.Printf("  Max nesting: %d\n", summary.Complexity.MaxNestingDepth)
		fmt.Printf("  Rating:      %s\n", summary.Complexity.ComplexityRating)
	}
}
