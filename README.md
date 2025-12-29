# mcp-codewizard

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)

**Semantic code search and analysis for AI coding assistants.**

mcp-codewizard is an MCP (Model Context Protocol) server that provides intelligent code search and analysis capabilities to AI assistants like Claude. It indexes your codebase locally, enabling semantic search, call graph navigation, complexity analysis, and more.

## Why mcp-codewizard?

When working with AI coding assistants on large codebases, simple text search isn't enough. You need:

- **Semantic understanding** - Find code by meaning, not just keywords
- **Code navigation** - Understand how functions call each other
- **Local processing** - Keep your code private, no cloud uploads
- **Deep analysis** - Complexity metrics, dead code detection, patterns

mcp-codewizard solves all of this by creating a local semantic index of your code that AI assistants can query through the MCP protocol.

## Features

### Core Capabilities

| Feature | Description |
|---------|-------------|
| **Semantic Search** | Find code by meaning using hybrid search (BM25 + vector + reranking) |
| **Symbol Navigation** | Browse functions, types, variables with full metadata |
| **Call Graph** | See who calls a function and what it calls |
| **Code Context** | Get surrounding lines for better understanding |

### Analysis Tools

| Feature | Description |
|---------|-------------|
| **Complexity Metrics** | Cyclomatic complexity, cognitive complexity, nesting depth |
| **Dead Code Detection** | Find unused functions and unreachable code |
| **Git Blame Integration** | See who wrote each line and when |
| **Entry Points** | Identify main functions, HTTP handlers, CLI commands |

### Technical Highlights

- **Plugin Architecture** - Swap embedding providers, chunking strategies, vector stores
- **Incremental Indexing** - Only re-index changed files
- **Parallel Processing** - Fast indexing using all CPU cores
- **TreeSitter Parsing** - Language-aware code chunking for 34 languages
- **Local-First** - Everything runs on your machine with Ollama

## Quick Start

### 1. Install Prerequisites

```bash
# Install Ollama (for embeddings)
curl -fsSL https://ollama.com/install.sh | sh

# Pull the embedding model
ollama pull nomic-embed-text
```

### 2. Install mcp-codewizard

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.sh | bash

# Windows (PowerShell)
# irm https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.ps1 | iex
```

### 3. Initialize and Index Your Project

```bash
cd /path/to/your/project

# Interactive setup wizard (recommended for first time)
mcp-codewizard init

# Or quick setup with recommended settings
mcp-codewizard init --preset recommended
```

### 4. Configure Your AI Assistant

Add to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "codeindex": {
      "command": "/path/to/mcp-codewizard",
      "args": ["serve", "--stdio"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

## Installation

### Quick Install (Recommended)

**Linux / macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.ps1 | iex
```

**Update to latest version:**
```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.sh | bash -s update

# Windows (PowerShell) - just run installer again
irm https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.ps1 | iex
```

### Download Binary

Pre-built binaries for all platforms are available on the [Releases](https://github.com/spetr/mcp-codewizard/releases) page:

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | x64 | `mcp-codewizard-vX.X.X-linux-amd64.tar.gz` |
| Linux | ARM64 | `mcp-codewizard-vX.X.X-linux-arm64.tar.gz` |
| macOS | x64 (Intel) | `mcp-codewizard-vX.X.X-darwin-amd64.tar.gz` |
| macOS | ARM64 (Apple Silicon) | `mcp-codewizard-vX.X.X-darwin-arm64.tar.gz` |
| Windows | x64 | `mcp-codewizard-vX.X.X-windows-amd64.zip` |
| Windows | ARM64 | `mcp-codewizard-vX.X.X-windows-arm64.zip` |

### Requirements

- **Ollama** (recommended) or OpenAI API key
- **~500MB disk** for embedding model
- **~50MB per 10k LOC** for index storage

### Build from Source

Requires **Go 1.21+** with CGO enabled.

```bash
git clone https://github.com/spetr/mcp-codewizard
cd mcp-codewizard

# Build with CGO (required for sqlite-vec and TreeSitter)
CGO_ENABLED=1 go build -o mcp-codewizard ./cmd/mcp-codewizard

# Verify installation
./mcp-codewizard version
```

### Docker (Coming Soon)

```bash
docker run -v $(pwd):/project ghcr.io/spetr/mcp-codewizard serve --stdio
```

## Configuration

Configuration is stored in `.mcp-codewizard/config.yaml` in your project root.

### Basic Configuration

```yaml
# .mcp-codewizard/config.yaml

embedding:
  provider: ollama
  model: nomic-embed-text
  endpoint: http://localhost:11434
  batch_size: 32

chunking:
  strategy: treesitter    # or "simple" for basic line-based
  max_chunk_size: 2000

search:
  mode: hybrid            # vector, bm25, or hybrid
  default_limit: 10

index:
  include:
    - "**/*.go"
    - "**/*.py"
    - "**/*.js"
    - "**/*.ts"
    - "**/*.jsx"
    - "**/*.tsx"
    - "**/*.rs"
    - "**/*.java"
    - "**/*.c"
    - "**/*.cpp"
    - "**/*.rb"
    - "**/*.php"
    - "**/*.cs"
    - "**/*.kt"
    - "**/*.swift"
    - "**/*.scala"
    - "**/*.lua"
    - "**/*.sql"
    - "**/*.proto"
    - "**/*.md"
    - "**/*.sh"
    - "**/*.css"
    - "**/Dockerfile"
    - "**/*.yaml"
    - "**/*.yml"
    - "**/*.tf"
    - "**/*.hcl"
    - "**/*.html"
    - "**/*.svelte"
  exclude:
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/.git/**"
  use_gitignore: true
```

### With Reranking (Better Quality)

```yaml
reranker:
  enabled: true
  provider: ollama
  model: qwen3-reranker
  candidates: 100
```

### With OpenAI

```yaml
embedding:
  provider: openai
  model: text-embedding-3-small
  # Set OPENAI_API_KEY environment variable
```

## CLI Usage

### Initialization (New Projects)

```bash
# Interactive setup wizard - detects environment, recommends settings
mcp-codewizard init

# Quick setup with preset (skips prompts)
mcp-codewizard init --preset recommended   # Best balance
mcp-codewizard init --preset quality       # Maximum accuracy
mcp-codewizard init --preset fast          # Minimal resources

# Initialize without starting indexing
mcp-codewizard init --preset recommended --no-index

# Output as JSON (for automation/MCP)
mcp-codewizard init --json
```

The wizard will:
1. Detect your environment (Ollama, OpenAI, GPU, RAM)
2. Analyze your project (languages, size, complexity)
3. Recommend optimal settings
4. Create configuration and optionally start indexing

### Indexing

```bash
# Index current directory
mcp-codewizard index

# Force full re-index
mcp-codewizard index --force

# Show what would be indexed
mcp-codewizard index --dry-run

# Index specific directory
mcp-codewizard index /path/to/project
```

### Searching

```bash
# Semantic search
mcp-codewizard search "HTTP authentication handler"

# Search with filters
mcp-codewizard search "database connection" --lang go --limit 20

# Vector-only search (faster, less accurate)
mcp-codewizard search "error handling" --mode vector
```

### Status

```bash
# Show index statistics
mcp-codewizard status

# Detailed stats
mcp-codewizard status --verbose
```

### Configuration

```bash
# Initialize config with auto-detection
mcp-codewizard config init

# Validate configuration
mcp-codewizard config validate
```

### MCP Server

```bash
# Start as stdio server (for AI assistants)
mcp-codewizard serve --stdio

# Start as HTTP server
mcp-codewizard serve --http :8080
```

## MCP Tools Reference

When used with an AI assistant, these tools are available:

### Setup & Configuration

| Tool | Description |
|------|-------------|
| `init_project` | Interactive setup wizard - detect environment, configure, optionally index |
| `detect_environment` | Detect available providers, project structure, get recommendations |
| `get_config` | Get current configuration |
| `validate_config` | Validate config and test provider connections |
| `apply_recommendation` | Apply a configuration preset (primary, low_memory, high_quality) |

### Indexing & Search

| Tool | Description |
|------|-------------|
| `index_codebase` | Index or re-index the project |
| `search_code` | Semantic code search with filters |
| `get_chunk` | Get a specific code chunk with context |
| `get_status` | Get index statistics |
| `clear_index` | Delete the index |

#### index_codebase Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `force` | boolean | Force re-index all files (ignore cache) |
| `ignore_patterns` | array | Additional patterns to exclude (e.g., `["**/test/**", "*.spec.ts"]`) |
| `custom_extensions` | array | Additional file extensions to include (e.g., `[".vue", ".svelte"]`) |

#### search_code Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | string | Search query (required) |
| `limit` | number | Maximum results (default: 10) |
| `mode` | string | Search mode: `vector`, `bm25`, `hybrid` (default) |
| `no_rerank` | boolean | Disable reranking |
| `include_context` | boolean | Include surrounding lines |
| `languages` | array | Filter by languages (e.g., `["go", "python"]`) |

### Symbol Navigation

| Tool | Description |
|------|-------------|
| `get_symbols` | List symbols (functions, types, etc.) with filters |
| `get_callers` | Find all callers of a function |
| `get_callees` | Find all functions called by a function |

### Analysis

| Tool | Description |
|------|-------------|
| `get_complexity` | Get complexity metrics for a file or function |
| `get_dead_code` | Find unused/unreachable code |
| `get_blame` | Get git blame information for code |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Assistant                              │
│                    (Claude, GPT, etc.)                          │
└─────────────────────────────────┬───────────────────────────────┘
                                  │ MCP Protocol
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      mcp-codewizard                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  MCP Server  │  │   Indexer    │  │    Search Engine     │  │
│  │   (stdio)    │  │  (parallel)  │  │ (hybrid BM25+vector) │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                            │                    │                │
│  ┌─────────────────────────┴────────────────────┴─────────────┐ │
│  │                    Provider Layer                          │ │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌────────────┐   │ │
│  │  │Embedding│  │ Chunking │  │Reranker │  │VectorStore │   │ │
│  │  │ Ollama  │  │TreeSitter│  │  Qwen3  │  │ sqlite-vec │   │ │
│  │  │ OpenAI  │  │  Simple  │  │  None   │  │            │   │ │
│  │  └─────────┘  └──────────┘  └─────────┘  └────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    .mcp-codewizard/                               │
│  ┌──────────────┐  ┌──────────────────────────────────────────┐ │
│  │ config.yaml  │  │              index.db                    │ │
│  │              │  │  • Chunks + embeddings (sqlite-vec)      │ │
│  │              │  │  • Symbols + references                  │ │
│  │              │  │  • BM25 full-text index (FTS5)          │ │
│  └──────────────┘  └──────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works

1. **Indexing Pipeline**
   ```
   Source Files → TreeSitter Parser → Chunks + Symbols + References
                                           │
                                           ▼
                              Ollama/OpenAI Embedding API
                                           │
                                           ▼
                              sqlite-vec Vector Storage
   ```

2. **Search Pipeline**
   ```
   Query → Embedding → Vector Search ─┐
                                      ├─→ Score Fusion → Reranking → Results
          BM25 Full-Text Search ──────┘
   ```

3. **Incremental Updates**
   - File hashes are cached
   - Only changed files are re-indexed
   - Configuration changes trigger full re-index

### Plugin System

mcp-codewizard uses a plugin architecture for extensibility:

```go
// Register a custom embedding provider
provider.RegisterEmbedding("my-provider", func(cfg provider.EmbeddingConfig) (provider.EmbeddingProvider, error) {
    return mypackage.New(cfg), nil
})
```

**Built-in Providers:**

| Type | Providers |
|------|-----------|
| Embedding | `ollama`, `openai` |
| Chunking | `treesitter`, `simple` |
| Reranker | `ollama`, `none` |
| VectorStore | `sqlitevec` |

## Supported Languages

TreeSitter chunking provides full AST-aware parsing for 34 languages:

| Language | Extensions | Symbol Extraction |
|----------|------------|-------------------|
| Go | `.go` | Functions, types, methods, interfaces |
| Python | `.py` | Functions, classes, methods |
| JavaScript | `.js`, `.jsx` | Functions, classes, arrow functions |
| TypeScript | `.ts`, `.tsx` | Functions, classes, interfaces, types |
| Rust | `.rs` | Functions, structs, impls, traits, enums |
| Java | `.java` | Classes, methods, interfaces, enums |
| C | `.c`, `.h` | Functions, structs, typedefs |
| C++ | `.cpp`, `.hpp`, `.cc`, `.cxx` | Functions, classes, structs |
| Ruby | `.rb` | Classes, methods, modules |
| PHP | `.php` | Classes, functions, methods, traits, interfaces |
| C# | `.cs` | Classes, methods, interfaces, structs, records, enums |
| Kotlin | `.kt`, `.kts` | Functions, classes, objects, interfaces |
| Swift | `.swift` | Functions, classes, structs, protocols, extensions |
| Scala | `.scala`, `.sc` | Functions, classes, objects, traits |
| Lua | `.lua` | Functions, local functions, variables |
| SQL | `.sql` | Tables, views, functions, procedures, triggers |
| Protobuf | `.proto` | Messages, enums, services, RPCs |
| Markdown | `.md` | Headings, code blocks |
| Bash/Shell | `.sh`, `.bash` | Functions |
| CSS | `.css` | Rules, keyframes, media queries |
| Dockerfile | `Dockerfile` | FROM, RUN, COPY instructions |
| YAML | `.yaml`, `.yml` | Top-level keys |
| HCL/Terraform | `.tf`, `.hcl` | Resources, variables, outputs, modules |
| Elixir | `.ex`, `.exs` | Modules, functions, macros, protocols |
| Elm | `.elm` | Modules, functions, types, type aliases |
| Groovy | `.groovy`, `.gradle` | Classes, methods, closures |
| OCaml | `.ml`, `.mli` | Modules, functions, types, values |
| Pascal/Delphi | `.pas`, `.dpr`, `.pp` | Functions, procedures, classes, interfaces |
| TOML | `.toml` | Tables, arrays, key-value pairs |
| Cue | `.cue` | Definitions, fields, packages |
| **HTML** | `.html`, `.htm`, `.xhtml` | Embedded JavaScript extraction |
| **Svelte** | `.svelte` | Embedded JavaScript + expressions |

### Embedded JavaScript Support

HTML, Svelte, and PHP files contain embedded JavaScript that is automatically extracted and indexed:

- **HTML/XHTML**: Extracts `<script>` tags (inline and module)
- **Svelte**: Extracts `<script>` tags and `{expression}` syntax
- **PHP**: Extracts JavaScript from HTML parts, handles `<?php ?>`, `<? ?>`, and `<?= ?>` tags

This ensures that JavaScript code in template files is searchable and analyzed just like standalone JS files.

All languages support:
- **Chunking** - Semantic code splitting based on AST structure
- **Symbol extraction** - Functions, classes, methods, types
- **Reference extraction** - Function calls, imports, type usage
- **Call graph analysis** - Callers and callees for navigation

## Examples

See the [examples/](examples/) directory for:

- **Search Prompts** - How to ask your AI assistant to search code
- **Analysis Prompts** - Complexity, dead code, blame queries
- **Workflow Examples** - Onboarding, debugging, refactoring scenarios

## Troubleshooting

### Ollama Connection Failed

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama
ollama serve
```

### Model Not Found

```bash
# List available models
ollama list

# Pull required model
ollama pull nomic-embed-text
```

### Slow Indexing

```yaml
# Reduce batch size for less memory usage
embedding:
  batch_size: 16

# Or use simple chunking (faster, less accurate)
chunking:
  strategy: simple
```

### Large Codebase

```yaml
# Exclude more directories
index:
  exclude:
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/dist/**"
    - "**/build/**"
    - "**/*.generated.*"
    - "**/testdata/**"
```

## Development

### Running Tests

```bash
CGO_ENABLED=1 go test -tags "fts5" ./...
```

### Building

```bash
CGO_ENABLED=1 go build -tags "fts5" -o bin/mcp-codewizard ./cmd/mcp-codewizard
```

### Project Structure

```
mcp-codewizard/
├── cmd/mcp-codewizard/     # CLI entry point
├── internal/
│   ├── config/            # Configuration handling
│   ├── index/             # Parallel indexing
│   ├── search/            # Hybrid search engine
│   ├── analysis/          # Complexity, dead code, blame
│   ├── mcp/               # MCP server implementation
│   └── wizard/            # Setup wizard
├── pkg/
│   ├── provider/          # Provider interfaces
│   ├── types/             # Shared types
│   └── plugin/            # External plugin support
├── builtin/               # Built-in providers
│   ├── embedding/         # Ollama, OpenAI
│   ├── chunking/          # TreeSitter, Simple
│   ├── reranker/          # Ollama, None
│   └── vectorstore/       # sqlite-vec
└── examples/              # Usage examples
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Business Source License 1.1 - see [LICENSE](LICENSE) for details.

**Free for:** Personal use, development, testing, non-commercial projects.

**Commercial use:** Requires a license. Contact: https://github.com/spetr/mcp-codewizard

**Change Date:** 2028-12-29 (converts to Apache 2.0)

## Acknowledgments

- Inspired by [zilliztech/claude-context](https://github.com/zilliztech/claude-context)
- Built with [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- Uses [sqlite-vec](https://github.com/asg017/sqlite-vec) for vector storage
- Parsing powered by [go-tree-sitter](https://github.com/smacker/go-tree-sitter)
