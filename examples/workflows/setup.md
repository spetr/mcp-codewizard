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

=== mcp-codewizard Setup ===

Detecting environment... done

Step 1: AI Provider
-------------------
Which embedding provider do you want to use?

  * [1] Ollama - running at http://localhost:11434
    [2] OpenAI - OPENAI_API_KEY not set
    [3] Custom endpoint

Select provider [1]: 1

Step 2: API Endpoint
--------------------
Ollama endpoint [http://localhost:11434]:
Testing connection... ✓ Connected

Step 3: Embedding Model
-----------------------
Available embedding models:

  * [1] nomic-embed-code (274 MB)
    [2] mxbai-embed-large (670 MB)
    [3] all-minilm (45 MB)

Select model [1]: 1

Step 4: Reranker (Optional)
---------------------------
Reranking improves search accuracy by ~15% but uses additional resources.

Available reranker models:
  * [1] qwen3-reranker (600 MB)
    [2] Disable reranker

Select [1]: 1

=== Configuration Summary ===
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

The wizard:
1. **Detects available providers** - Shows Ollama/OpenAI availability status
2. **Tests connection** - Verifies the endpoint is reachable
3. **Lists available models** - Shows models from the connected provider
4. **Configures reranker** - Optional, shows available reranker models
5. **Starts indexing** - Optionally begins indexing immediately

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

## Option 4: Custom Endpoint (OpenAI-compatible)

Use a custom OpenAI-compatible API (e.g., Azure OpenAI, local LLM server):

```
$ mcp-codewizard init

=== mcp-codewizard Setup ===

Detecting environment... done

Step 1: AI Provider
-------------------
Which embedding provider do you want to use?

    [1] Ollama - not detected
    [2] OpenAI - OPENAI_API_KEY not set
  * [3] Custom endpoint

Select provider [1]: 3

Step 2: API Endpoint
--------------------
API endpoint URL: https://my-company.openai.azure.com

API Key: sk-...
Testing connection... ✓ Connected

Step 3: Embedding Model
-----------------------
Available embedding models:

  * [1] text-embedding-3-small (fast, good quality)
    [2] text-embedding-3-large (highest quality)
    [3] text-embedding-ada-002 (legacy)

Select model [1]: 1

Step 4: Reranker (Optional)
---------------------------
Reranker requires Ollama. Skipping.

=== Configuration Summary ===
Embedding: openai/text-embedding-3-small
Endpoint: https://my-company.openai.azure.com
Reranker: Disabled
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
