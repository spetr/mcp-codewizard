# mcp-codewizard

MCP server for semantic code search and analysis with plugin support.

## Overview

- **Go** + local vector DB (sqlite-vec)
- **Plugin architecture** for embedding, chunking, vector store, reranker
- **Hybrid search** (BM25 + vector + reranking)
- **Analysis** - call graph, entry points, import graph, git history

## Project Structure

```
mcp-codewizard/
├── cmd/mcp-codewizard/     # CLI + MCP server entry point
├── internal/
│   ├── config/             # Configuration
│   ├── index/              # Indexing logic + parallel processing
│   ├── store/              # Vector store abstraction
│   ├── analysis/           # Call graph, entry points, patterns
│   ├── search/             # Hybrid search
│   ├── wizard/             # Setup wizard
│   └── mcp/                # MCP server
├── pkg/
│   ├── plugin/             # go-plugin definitions
│   ├── provider/           # Provider interfaces (embedding, chunking, vectorstore)
│   └── types/              # Shared types (Chunk, Symbol, Reference)
├── builtin/
│   ├── embedding/          # ollama, openai
│   ├── chunking/           # treesitter, simple
│   ├── reranker/           # ollama, none
│   └── vectorstore/        # sqlitevec
└── testdata/               # Test fixtures
```

**Data:** `.mcp-codewizard/` contains `config.yaml`, `index.db`, `cache/`

## Plugin Types

| Type | Interface | Description |
|------|-----------|-------------|
| EmbeddingProvider | `Embed(texts) → [][]float32` | Generate embeddings |
| ChunkingStrategy | `Chunk(file) → []*Chunk` | Split code + extract symbols |
| Reranker | `Rerank(query, docs) → []RerankResult` | Re-rank results |
| VectorStore | `Search/Store/GetCallers` | Store and search vectors |

## MCP Tools

### Setup & Config
| Tool | Description |
|------|-------------|
| `detect_environment` | Detect providers (Ollama, models) and project |
| `get_config` | Get current configuration |
| `set_config` | Update configuration |
| `validate_config` | Validate config and test connections |

### Indexing & Search
| Tool | Description |
|------|-------------|
| `index_codebase` | Index project |
| `search_code` | Semantic code search |
| `get_chunk` | Get chunk with context |
| `get_status` | Index status |
| `clear_index` | Delete index |

### Analysis
| Tool | Description |
|------|-------------|
| `get_callers` | Who calls this symbol |
| `get_callees` | What this symbol calls |
| `get_symbols` | List symbols with filters |
| `get_entry_points` | Entry points (main, handlers) |
| `get_import_graph` | Dependency graph |
| `get_complexity` | Complexity metrics |
| `get_dead_code` | Unused code |

### Git History
| Tool | Description |
|------|-------------|
| `index_git_history` | Index git history |
| `search_history` | Semantic search in history |
| `get_chunk_history` | Chunk/function change history |
| `get_code_evolution` | Symbol evolution over time |
| `find_regression` | Find commits that may have introduced a bug |
| `get_commit_context` | Commit details |
| `get_contributor_insights` | Expert on given code area |

### Memory (Persistent Context)
| Tool | Description |
|------|-------------|
| `memory_store` | Store memory/knowledge |
| `memory_recall` | Semantic memory search |
| `memory_forget` | Delete memory |
| `memory_checkpoint` | Snapshot backup |
| `memory_restore` | Restore from checkpoint |

Categories: `decision`, `context`, `fact`, `note`, `error`, `review`

### Todo (Task Management)
| Tool | Description |
|------|-------------|
| `todo_create` | New task (supports subtasks) |
| `todo_list` | List with filters |
| `todo_search` | Semantic search |
| `todo_update` | Update status/priority |
| `todo_complete` | Mark as completed |

Priorities: `urgent`, `high`, `medium`, `low`
Statuses: `pending`, `in_progress`, `completed`, `cancelled`, `blocked`

## Configuration

```yaml
# .mcp-codewizard/config.yaml
embedding:
  provider: ollama           # ollama | openai
  model: nomic-embed-code
  endpoint: http://localhost:11434
  batch_size: 32

chunking:
  strategy: treesitter       # treesitter | simple
  max_chunk_size: 2000

reranker:
  enabled: true
  provider: ollama
  model: qwen3-reranker
  candidates: 100

search:
  mode: hybrid               # vector | bm25 | hybrid
  vector_weight: 0.7
  bm25_weight: 0.3

index:
  # Supports 35+ languages and formats - see full list in default config
  include: ["**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.rs", "**/*.java",
            "**/*.c", "**/*.cpp", "**/*.rb", "**/*.php", "**/*.cs", "**/*.kt",
            "**/*.swift", "**/*.scala", "**/*.dart", "**/*.vb", "**/*.pas",
            "**/*.ps1", "**/*.r", "**/*.html", "**/*.css", "**/*.yaml", "..."]
  exclude: ["**/vendor/**", "**/node_modules/**", "**/.git/**", "**/dist/**",
            "**/build/**", "**/target/**", "**/*.min.js", "**/package-lock.json", "..."]
  use_gitignore: true
```

## CLI Commands

```bash
mcp-codewizard index [path]           # index project
mcp-codewizard index --force          # reindex all
mcp-codewizard search "query"         # hybrid search
mcp-codewizard status                 # index status
mcp-codewizard serve --stdio          # MCP server for Claude Code
mcp-codewizard watch [path]           # watch mode
mcp-codewizard config validate        # validate config

# Register with AI CLI tools
mcp-codewizard register claude-code   # register with Claude Code
mcp-codewizard register gemini        # register with Gemini CLI
mcp-codewizard register codex         # register with Codex CLI (OpenAI)
mcp-codewizard register all           # register with all tools
mcp-codewizard register claude-code -g  # global (user-level) registration
```

## Ollama Setup

```bash
# Enable multiple models
export OLLAMA_NUM_PARALLEL=2
export OLLAMA_MAX_LOADED_MODELS=2
```

Required models: `nomic-embed-code` (~500MB), `qwen3-reranker` (~600MB)

## Key Types

Definitions in `pkg/types/`:
- **Chunk** - code block with ID `{filepath}:{startline}:{hash[:8]}`
- **Symbol** - function/type/variable with Kind, Signature, Visibility
- **Reference** - link between symbols (call, type_use, import)
- **Commit/Change** - git history with embeddings

## Dependencies (CGO)

- sqlite-vec (vector DB)
- go-tree-sitter (AST parsing)

Build: `CGO_ENABLED=1 go build`

## Conventions

- All code, comments, and documentation in English
- Public functions have doc comments
- Logging via `log/slog`
- Provider interfaces are stable
