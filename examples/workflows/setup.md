# Project Setup Workflow

How to use mcp-codewizard to set up a new project for semantic code search.

## Scenario

You have a codebase you want to index for AI-assisted development. This workflow shows how to configure mcp-codewizard optimally for your environment and project.

## Option 1: Ask Your AI Assistant

The easiest way to set up a project is to ask your AI assistant directly:

```
User: Set up code search for this project

AI: [Uses init_project tool]

    === Environment ===
    Ollama: Running at http://localhost:11434
      - nomic-embed-code (embedding, 500MB)
      - qwen3-reranker (reranker, 600MB)
    OpenAI: Not configured
    System: darwin/arm64, 10 cores, 32GB RAM, Apple Silicon GPU

    === Project ===
    Path: /home/user/my-project
    Files: 245 (Go 65%, Python 20%, TypeScript 15%)
    Complexity: medium (est. 1,200 chunks)

    === Recommendations ===
    - Using Ollama with nomic-embed-code for local embeddings
    - Reranker enabled for better accuracy
    - TreeSitter chunking for Go/Python/TypeScript
    - Estimated indexing time: ~3 minutes

    Configuration saved. Starting indexing...

    Indexing complete: 245 files, 1,187 chunks, 892 symbols
```

### With Specific Preferences

```
User: Initialize this project for code search. I have limited RAM, use minimal resources.

AI: [Uses init_project with preset="fast"]

    Using 'fast' preset for minimal resource usage:
    - Embedding: ollama/nomic-embed-code (batch size: 64)
    - Reranker: disabled (saves 600MB RAM)
    - Chunking: simple (faster, less memory)
    - Workers: 4 (reduced parallelism)

    Configuration saved to .mcp-codewizard/config.yaml
```

```
User: Set up code search with maximum quality, I don't mind the cost.

AI: [Uses init_project with preset="quality"]

    Using 'quality' preset for maximum accuracy:
    - Embedding: openai/text-embedding-3-large
    - Reranker: enabled with 200 candidates
    - Chunking: treesitter (semantic boundaries)

    Note: Requires OPENAI_API_KEY environment variable.
```

## Option 2: CLI Interactive Wizard

Run the interactive wizard from the command line:

```bash
$ mcp-codewizard init

Detecting environment...

=== Environment ===
Ollama: Running at http://localhost:11434
  - nomic-embed-code (embedding)
  - qwen3-reranker (reranker)
OpenAI: Not configured
System: darwin/arm64, 10 cores, 32GB RAM

=== Project ===
Path: /home/user/my-project
Files: 245 (Go, Python, TypeScript)
Complexity: medium (est. 1,200 chunks)

=== Configuration ===
Choose a preset:

  *[1] Recommended - Best balance of quality and performance for your system
   [2] High Quality - Maximum search accuracy, more resources
   [3] Fast & Light - Minimal resources, good for large projects
   [4] Custom - Configure each option individually

Select preset [1]: 1

Using 'recommended' preset.

=== Configuration ===
Embedding: ollama/nomic-embed-code
Reranker: ollama/qwen3-reranker (top 100 candidates)
Chunking: treesitter (max 2000 tokens)
Search: hybrid (vector 70%, BM25 30%)

Configuration saved to .mcp-codewizard/config.yaml

Start indexing now? (Y/n): y

[embedding] Files: 245/245, Chunks: 1187/1187
Indexing complete!
Files: 245, Chunks: 1187, Symbols: 892
```

## Option 3: CLI with Preset

Skip prompts entirely with a preset:

```bash
# Recommended settings (auto-detected based on your environment)
mcp-codewizard init --preset recommended

# Maximum quality (requires OpenAI API or full Ollama setup)
mcp-codewizard init --preset quality

# Minimal resources (for large codebases or limited RAM)
mcp-codewizard init --preset fast

# Skip indexing (just create config)
mcp-codewizard init --preset recommended --no-index
```

## Option 4: Custom Configuration

Choose each option individually:

```
$ mcp-codewizard init

Select preset [1]: 4   # Custom

=== Custom Configuration ===
(* = recommended)

Embedding Provider:
  Choose how to generate code embeddings for semantic search

  *[1] Ollama (nomic-embed-code)
      Local, fast, optimized for code. No API costs.
   [2] OpenAI (text-embedding-3-small) [Set OPENAI_API_KEY environment variable]
      Cloud-based, high quality. Requires API key, has costs.
   [3] OpenAI (text-embedding-3-large) [Set OPENAI_API_KEY environment variable]
      Highest quality embeddings. Higher cost.

Select [1]: 1

Search Reranker:
  Reranking improves search accuracy by ~15% but uses more resources

  *[1] Enabled (qwen3-reranker)
      Better search accuracy, uses ~600MB VRAM
   [2] Disabled
      Faster, less resource usage, slightly lower accuracy

Select [1]: 1

Code Chunking:
  How to split code files for indexing

  *[1] TreeSitter (semantic)
      AST-aware chunking. Detected: go, python, typescript
   [2] Simple (line-based)
      Fast, works with any language. Less accurate boundaries.

Select [1]: 1

Search Mode:
  How to search the codebase

  *[1] Hybrid (BM25 + Vector)
      Best accuracy. Combines keyword and semantic search.
   [2] Vector only
      Pure semantic search. Good for conceptual queries.
   [3] BM25 only
      Keyword search only. Fast, no embeddings needed.

Select [1]: 1

=== Configuration ===
Embedding: ollama/nomic-embed-code
Reranker: ollama/qwen3-reranker (top 100 candidates)
Chunking: treesitter (max 2000 tokens)
Search: hybrid (vector 70%, BM25 30%)

Configuration saved.
```

## Presets Explained

| Preset | Embedding | Reranker | Chunking | Best For |
|--------|-----------|----------|----------|----------|
| `recommended` | Auto-detected (Ollama or OpenAI) | Enabled if available | TreeSitter | Most projects |
| `quality` | OpenAI large or Ollama | Enabled, 200 candidates | TreeSitter | When accuracy matters |
| `fast` | Ollama | Disabled | Simple | Large codebases, limited RAM |

## Troubleshooting

### Ollama Not Running

```
User: Set up code search

AI: [Uses init_project]

    === Environment ===
    Ollama: Not running
      Warning: Start Ollama with: ollama serve

    Recommendations:
    - Start Ollama: ollama serve
    - Pull embedding model: ollama pull nomic-embed-code
    - Or set OPENAI_API_KEY for cloud embeddings
```

### Missing Models

```
User: Initialize code search

AI: [Uses init_project]

    === Environment ===
    Ollama: Running
      Warning: nomic-embed-code not found
      Run: ollama pull nomic-embed-code

    Using fallback configuration without reranking.
```

### Large Codebase

```
User: Set up code search for this monorepo

AI: [Uses init_project]

    === Project ===
    Files: 45,000
    Complexity: huge (est. 120,000 chunks)

    Warning: Very large project detected.

    Recommendations:
    - Use 'fast' preset to reduce memory usage
    - Consider excluding vendor/, node_modules/, generated/
    - Index specific directories instead of entire repo
    - Estimated time: ~45 minutes

    Want me to use the fast preset?
```

## Next Steps

After setup is complete:

1. **Search code**: "Find the authentication handler"
2. **Navigate symbols**: "Who calls the validateToken function?"
3. **Analyze complexity**: "What are the most complex functions?"

See other workflow examples:
- [onboarding.md](onboarding.md) - Getting familiar with a codebase
- [debugging.md](debugging.md) - Finding and fixing bugs
- [refactoring.md](refactoring.md) - Safe code changes
