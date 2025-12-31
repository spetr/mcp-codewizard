# Configuration Reference

mcp-codewizard uses a YAML configuration file stored at `.mcp-codewizard/config.yaml` in your project root.

## Quick Start

```bash
# Interactive setup (recommended)
mcp-codewizard init

# Or with preset
mcp-codewizard init --preset recommended
```

## Complete Configuration

```yaml
# .mcp-codewizard/config.yaml

# =============================================================================
# Embedding Configuration
# =============================================================================
embedding:
  provider: ollama              # ollama | openai | voyage | jina
  model: nomic-embed-code       # Model name
  endpoint: http://localhost:11434  # API endpoint
  api_key: ""                   # API key (required for openai/voyage/jina)
  batch_size: 32                # Documents per batch (reduce for less memory)

# =============================================================================
# Chunking Configuration
# =============================================================================
chunking:
  strategy: treesitter          # treesitter | simple
  max_chunk_size: 2000          # Maximum tokens per chunk

# =============================================================================
# Reranker Configuration
# =============================================================================
reranker:
  enabled: true                 # Enable/disable reranking
  provider: ollama              # ollama | none
  model: qwen3-reranker         # Model name
  endpoint: http://localhost:11434  # API endpoint
  candidates: 100               # Number of candidates to rerank

# =============================================================================
# Search Configuration
# =============================================================================
search:
  mode: hybrid                  # vector | bm25 | hybrid
  vector_weight: 0.7            # Weight for vector search (0.0-1.0)
  bm25_weight: 0.3              # Weight for BM25 search (0.0-1.0)
  default_limit: 10             # Default number of results

# =============================================================================
# Vector Store Configuration
# =============================================================================
vectorstore:
  provider: sqlitevec           # sqlitevec (only option currently)

# =============================================================================
# Index Configuration
# =============================================================================
index:
  include:                      # Glob patterns to include
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
    - "**/*.h"
    - "**/*.rb"
    - "**/*.php"
    - "**/*.cs"
    - "**/*.kt"
    - "**/*.swift"
    - "**/*.scala"
    - "**/*.lua"
    - "**/*.sql"
    - "**/*.proto"
    - "**/*.sh"
    - "**/*.md"
    - "**/*.yaml"
    - "**/*.yml"
    - "**/*.json"
    - "**/*.html"
    - "**/*.css"
    - "**/Dockerfile"
  exclude:                      # Glob patterns to exclude
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/.git/**"
    - "**/dist/**"
    - "**/build/**"
    - "**/target/**"
    - "**/bin/**"
    - "**/*.min.js"
    - "**/*.min.css"
    - "**/*.generated.*"
    - "**/package-lock.json"
    - "**/yarn.lock"
    - "**/go.sum"
    - "**/Cargo.lock"
  use_gitignore: true           # Respect .gitignore patterns

# =============================================================================
# Resource Limits
# =============================================================================
limits:
  max_file_size: 1MB            # Maximum file size to index
  max_files: 50000              # Maximum number of files
  max_chunk_tokens: 2000        # Maximum tokens per chunk
  timeout: 30m                  # Indexing timeout
  memory_limit: ""              # Memory limit (e.g., "4GB")
  workers: 0                    # Parallel workers (0 = auto, uses all CPUs)

# =============================================================================
# Analysis Configuration
# =============================================================================
analysis:
  extract_symbols: true         # Extract function/class symbols
  extract_references: true      # Extract call references
  detect_patterns: false        # Detect code patterns (experimental)

# =============================================================================
# Logging Configuration
# =============================================================================
logging:
  level: info                   # debug | info | warn | error
  format: text                  # text | json
```

## Section Details

### Embedding

Controls how code is converted to vector embeddings for semantic search.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `provider` | string | `ollama` | Embedding provider: `ollama`, `openai`, `voyage`, `jina` |
| `model` | string | `nomic-embed-code` | Model name |
| `endpoint` | string | `http://localhost:11434` | API endpoint URL |
| `api_key` | string | `""` | API key (required for cloud providers) |
| `batch_size` | int | `32` | Documents per embedding batch |

**Provider Examples:**

```yaml
# Ollama (local, free)
embedding:
  provider: ollama
  model: nomic-embed-code
  endpoint: http://localhost:11434

# OpenAI (cloud, paid)
embedding:
  provider: openai
  model: text-embedding-3-small
  # Set OPENAI_API_KEY environment variable

# OpenAI with custom endpoint
embedding:
  provider: openai
  model: text-embedding-3-large
  endpoint: https://api.openai.com/v1
  api_key: sk-...
```

### Chunking

Controls how source code is split into searchable chunks.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `strategy` | string | `treesitter` | Chunking strategy: `treesitter`, `simple` |
| `max_chunk_size` | int | `2000` | Maximum tokens per chunk |

**Strategies:**

- **treesitter** - AST-aware chunking, respects function/class boundaries, extracts symbols
- **simple** - Line-based chunking, faster but less accurate

### Reranker

Optional second-stage ranking for improved search quality.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable reranking |
| `provider` | string | `ollama` | Provider: `ollama`, `none` |
| `model` | string | `qwen3-reranker` | Model name |
| `endpoint` | string | `http://localhost:11434` | API endpoint |
| `candidates` | int | `100` | Candidates to fetch before reranking |

**Disable reranking** (faster, less accurate):

```yaml
reranker:
  enabled: false
```

### Search

Controls search behavior.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mode` | string | `hybrid` | Search mode: `vector`, `bm25`, `hybrid` |
| `vector_weight` | float | `0.7` | Weight for vector similarity (0.0-1.0) |
| `bm25_weight` | float | `0.3` | Weight for BM25 text matching (0.0-1.0) |
| `default_limit` | int | `10` | Default number of results |

**Search Modes:**

- **hybrid** - Combines vector + BM25, best quality
- **vector** - Semantic similarity only, good for conceptual queries
- **bm25** - Keyword matching only, good for exact terms

### Index

Controls which files are indexed.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `include` | []string | (see below) | Glob patterns to include |
| `exclude` | []string | (see below) | Glob patterns to exclude |
| `use_gitignore` | bool | `true` | Respect `.gitignore` patterns |

**Default Include Patterns:**

Core languages: `*.go`, `*.py`, `*.js`, `*.ts`, `*.jsx`, `*.tsx`, `*.rs`, `*.java`, `*.c`, `*.cpp`, `*.h`, `*.hpp`

Additional: `*.rb`, `*.php`, `*.cs`, `*.kt`, `*.swift`, `*.scala`, `*.lua`, `*.sql`, `*.proto`, `*.sh`

New in v0.2.0: `*.dart`, `*.vb`, `*.pas`, `*.ps1`, `*.r`, `*.R`

Config/markup: `*.html`, `*.css`, `*.yaml`, `*.yml`, `*.json`, `*.toml`, `*.md`, `*.tf`, `*.hcl`, `Dockerfile`

**Default Exclude Patterns:**

```yaml
exclude:
  - "**/vendor/**"
  - "**/node_modules/**"
  - "**/.git/**"
  - "**/dist/**"
  - "**/build/**"
  - "**/target/**"
  - "**/*.min.js"
  - "**/*.min.css"
  - "**/package-lock.json"
  - "**/go.sum"
  - "**/Cargo.lock"
```

### Limits

Resource constraints for indexing.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max_file_size` | string | `1MB` | Skip files larger than this |
| `max_files` | int | `50000` | Maximum files to index |
| `max_chunk_tokens` | int | `2000` | Maximum tokens per chunk |
| `timeout` | duration | `30m` | Indexing timeout |
| `memory_limit` | string | `""` | Memory limit (empty = unlimited) |
| `workers` | int | `0` | Parallel workers (0 = auto) |

### Analysis

Controls code analysis features.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `extract_symbols` | bool | `true` | Extract functions, classes, etc. |
| `extract_references` | bool | `true` | Extract call references |
| `detect_patterns` | bool | `false` | Detect code patterns (experimental) |

### Logging

Controls log output.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `format` | string | `text` | Output format: `text`, `json` |

## Configuration Presets

Use `mcp-codewizard init --preset <name>` for quick setup:

| Preset | Description |
|--------|-------------|
| `recommended` | Best balance of quality and speed |
| `quality` | Maximum accuracy, slower indexing |
| `fast` | Minimal resources, faster indexing |

## Environment Variables

Some settings can be overridden via environment variables:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `OLLAMA_HOST` | Ollama endpoint (e.g., `http://remote:11434`) |

## Ollama Setup

For best results with Ollama:

```bash
# Enable multiple models in parallel
export OLLAMA_NUM_PARALLEL=2
export OLLAMA_MAX_LOADED_MODELS=2

# Pull required models
ollama pull nomic-embed-code    # ~500MB, for embeddings
ollama pull qwen3-reranker      # ~600MB, for reranking
```

## Validation

Validate your configuration:

```bash
mcp-codewizard config validate
```

This checks:
- Provider availability
- Model existence
- Endpoint connectivity
- Pattern validity

## Data Directory

All data is stored in `.mcp-codewizard/` in your project root:

```
.mcp-codewizard/
├── config.yaml      # Configuration file
├── index.db         # SQLite database with vectors, symbols, FTS index
└── cache/           # Temporary cache files
```

Add `.mcp-codewizard/` to your `.gitignore` to avoid committing the index.
