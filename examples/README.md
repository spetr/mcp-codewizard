# mcp-codewizard Examples

This directory contains examples showing how to use mcp-codewizard with AI coding assistants.

## Directory Structure

```
examples/
├── prompts/           # Individual prompt examples
│   ├── search.md      # Semantic code search
│   ├── analysis.md    # Code analysis (complexity, dead code, blame)
│   └── navigation.md  # Symbol and call graph navigation
└── workflows/         # Complete workflow scenarios
    ├── setup.md       # First-time project setup
    ├── onboarding.md  # Getting familiar with a new codebase
    ├── debugging.md   # Finding and fixing bugs
    └── refactoring.md # Safe code refactoring
```

## Quick Reference

### Most Useful Prompts

| Task | Prompt |
|------|--------|
| Setup project | "Initialize this project for code search" |
| Find code | "Search for code that handles user authentication" |
| Understand function | "What does the `ProcessPayment` function do and who calls it?" |
| Find callers | "Show me all places that call the `validateToken` function" |
| Check complexity | "What are the most complex functions in this project?" |
| Find dead code | "Are there any unused functions in the codebase?" |
| Blame | "Who wrote the error handling in `api/handlers.go`?" |

### Search Tips

1. **Be descriptive** - "HTTP middleware for rate limiting" works better than "rate limit"
2. **Use domain terms** - Use the same terminology as the codebase
3. **Combine with filters** - "Find authentication code in Go files only"
4. **Ask for context** - "Show me the function with surrounding code"

### Analysis Tips

1. **Start broad** - "What are the most complex files?" then drill down
2. **Combine tools** - Use complexity + dead code together for refactoring candidates
3. **Use blame for context** - Understand why code was written before changing it

## How It Works

When you ask your AI assistant a question about code, mcp-codewizard:

1. **Parses your question** to understand the intent
2. **Searches semantically** using vector similarity + keyword matching
3. **Reranks results** for better relevance (if enabled)
4. **Returns context** including surrounding code and metadata

The AI assistant then uses this information to answer your question.

## Getting Started

1. Make sure mcp-codewizard is configured in your AI assistant
2. Initialize your project: `mcp-codewizard init` (or ask your AI: "Initialize this project for code search")
3. Start asking questions!

### First Time Setup

The easiest way to get started is to ask your AI assistant:

> "Set up code search for this project"

The AI will use the `init_project` tool to:
- Detect your environment (Ollama, OpenAI, GPU, RAM)
- Analyze your project (languages, file count, complexity)
- Recommend optimal settings
- Create configuration and start indexing

### Presets

| Preset | Description | Use Case |
|--------|-------------|----------|
| `recommended` | Best balance of quality and speed | Most projects |
| `quality` | Maximum search accuracy | When precision matters |
| `fast` | Minimal resources, quick indexing | Large codebases, limited RAM |

See individual prompt files for detailed examples.
