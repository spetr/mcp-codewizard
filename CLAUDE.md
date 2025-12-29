# mcp-codewizard

MCP server pro sémantické vyhledávání a analýzu kódu s podporou pluginů.

## Přehled projektu

Inspirováno [zilliztech/claude-context](https://github.com/zilliztech/claude-context), ale s těmito rozdíly:
- **Go** místo Node.js
- **Lokální vektorová databáze** (sqlite-vec) místo cloud (Zilliz)
- **Plugin architektura** pro embedding providery, chunking strategie a vector stores
- **Rozšířená analýza** - call graph, entry points, import graph, pattern detection

## Architektura

### Plugin systém

**Dvouvrstvá architektura:**

1. **Built-in providery** (kompilované přímo do binárky)
   - Ollama embedding
   - OpenAI embedding
   - TreeSitter chunking (CGO)
   - Simple chunking
   - sqlite-vec vector store

2. **Externí pluginy** via [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)
   - Pro third-party rozšíření
   - RPC přes gRPC
   - Separátní binárky

**Plugin typy:**
1. **EmbeddingProvider** - generování embeddingů
2. **ChunkingStrategy** - dělení kódu na chunky + extrakce symbolů/referencí
3. **VectorStore** - ukládání a vyhledávání vektorů
4. **Reranker** - re-ranking výsledků pro lepší přesnost

**Registrace:**
```go
// Built-in se registrují při startu
registry.RegisterEmbedding("ollama", builtin.NewOllamaProvider)
registry.RegisterEmbedding("openai", builtin.NewOpenAIProvider)

// Externí pluginy se načítají z .mcp-codewizard/plugins/
registry.LoadExternalPlugins(pluginsDir)
```

### Datová struktura

```
.mcp-codewizard/
├── config.yaml          # Hlavní konfigurace
├── index.db             # SQLite + sqlite-vec (vektory + metadata)
├── cache/               # File hash cache pro inkrementální indexaci
└── plugins/             # Externí pluginy (optional)
```

### Struktura projektu

```
mcp-codewizard/
├── cmd/
│   └── mcp-codewizard/
│       └── main.go              # CLI + MCP server entry point
├── internal/
│   ├── config/                  # Konfigurace (.mcp-codewizard/config.yaml)
│   ├── index/                   # Hlavní indexační logika + parallel processing
│   ├── store/                   # Abstrakce nad vector store
│   ├── analysis/                # Call graph, entry points, patterns
│   ├── search/                  # Hybridní search (BM25 + vector)
│   ├── wizard/                  # Setup wizard (detect, recommend, validate)
│   └── mcp/                     # MCP server (používá official go-sdk)
├── pkg/
│   ├── plugin/                  # go-plugin definice pro externí pluginy
│   │   ├── shared/              # Shared proto/interfaces
│   │   └── host/                # Plugin host (loader)
│   ├── provider/                # Provider interfaces
│   │   ├── embedding.go         # EmbeddingProvider interface
│   │   ├── chunking.go          # ChunkingStrategy interface
│   │   └── vectorstore.go       # VectorStore interface
│   └── types/                   # Shared typy (Chunk, Symbol, Reference, etc.)
├── builtin/                     # Built-in implementace
│   ├── embedding/
│   │   ├── ollama/              # Ollama provider
│   │   └── openai/              # OpenAI provider
│   ├── chunking/
│   │   ├── treesitter/          # TreeSitter (CGO)
│   │   └── simple/              # Simple line-based
│   ├── reranker/
│   │   ├── ollama/              # Qwen3-Reranker přes Ollama
│   │   └── none/                # No-op (passthrough)
│   └── vectorstore/
│       └── sqlitevec/           # sqlite-vec (CGO)
├── testdata/                    # Test fixtures
│   ├── projects/                # Malé testovací projekty
│   │   ├── go-simple/
│   │   ├── python-simple/
│   │   └── mixed/
│   └── search_quality/          # Golden test dataset
│       └── queries.yaml
└── go.mod
```

## Datové modely

### SourceFile

```go
type SourceFile struct {
    Path     string
    Content  []byte
    Language string    // go, python, javascript, typescript, ...
    Hash     string    // SHA256 pro inkrementální indexaci
}
```

### Chunk

```go
type Chunk struct {
    ID            string    // {filepath}:{startline}:{hash[:8]}
    FilePath      string
    Language      string
    Content       string
    ChunkType     ChunkType // function, class, method, block, file
    Name          string    // název funkce/třídy/metody
    ParentName    string    // parent scope
    StartLine     int
    EndLine       int
    Hash          string    // SHA256 obsahu
}

type ChunkType string

const (
    ChunkTypeFunction ChunkType = "function"
    ChunkTypeClass    ChunkType = "class"
    ChunkTypeMethod   ChunkType = "method"
    ChunkTypeBlock    ChunkType = "block"
    ChunkTypeFile     ChunkType = "file"
)
```

### Symbol

```go
type Symbol struct {
    ID         string
    Name       string
    Kind       SymbolKind  // function, type, variable, const, interface
    FilePath   string
    StartLine  int
    EndLine    int
    Signature  string      // pro funkce: func(x int) error
    Visibility string      // public, private
    DocComment string
}
```

### Reference

```go
type Reference struct {
    ID         string
    FromSymbol string      // symbol ID který odkazuje
    ToSymbol   string      // symbol ID na který se odkazuje (může být external)
    Kind       RefKind     // call, type_use, import, implement
    FilePath   string
    Line       int
    IsExternal bool        // true pokud ToSymbol není v našem indexu (např. fmt.Println)
}

type RefKind string

const (
    RefKindCall      RefKind = "call"
    RefKindTypeUse   RefKind = "type_use"
    RefKindImport    RefKind = "import"
    RefKindImplement RefKind = "implement"
)
```

### CodePattern

```go
type CodePattern struct {
    ID         string
    Name       string      // repository, middleware, singleton, factory
    FilePath   string
    StartLine  int
    EndLine    int
    Confidence float32
    Evidence   []string
}
```

### Git History Models (Temporální vyhledávání)

```go
// Commit reprezentuje jeden git commit
type Commit struct {
    Hash             string    // Plný SHA hash
    ShortHash        string    // Zkrácený hash (7 znaků)
    Author           string
    AuthorEmail      string
    Date             time.Time
    Message          string
    MessageEmbedding []float32 // Embedding commit message pro sémantické hledání
    ParentHash       string    // Hash rodičovského commitu
    FilesChanged     int
    Insertions       int
    Deletions        int
    IsMerge          bool
    Tags             []string
}

// Change reprezentuje změnu souboru v commitu
type Change struct {
    ID                string
    CommitHash        string
    FilePath          string
    ChangeType        ChangeType // A=added, M=modified, D=deleted, R=renamed
    OldPath           string     // Pro přejmenované soubory
    DiffContent       string     // Obsah diffu
    DiffEmbedding     []float32  // Embedding diffu pro sémantické hledání
    Additions         int
    Deletions         int
    AffectedFunctions []string   // Funkce ovlivněné touto změnou
    AffectedChunkIDs  []string   // Chunky ovlivněné touto změnou
}

// ChunkHistoryEntry mapuje chunk na commit kde byl změněn
type ChunkHistoryEntry struct {
    ChunkID     string
    CommitHash  string
    ChangeType  string     // "created", "modified", "deleted"
    DiffSummary string
    Date        time.Time
    Author      string
}

// HistorySearchRequest pro vyhledávání v historii
type HistorySearchRequest struct {
    Query           string       // Sémantický dotaz
    QueryVec        []float32    // Embedding dotazu
    TimeFrom        *time.Time   // Filtr: změny od
    TimeTo          *time.Time   // Filtr: změny do
    Authors         []string     // Filtr: autoři
    Paths           []string     // Filtr: cesty (glob patterns)
    ChangeTypes     []ChangeType // Filtr: typy změn
    Functions       []string     // Filtr: konkrétní funkce
    Limit           int
    TimeDecayFactor float32      // Penalizace starších změn (0-1)
}

// RegressionCandidate - podezřelý commit pro regresi
type RegressionCandidate struct {
    Commit      *Commit
    Changes     []*Change
    Score       float32   // 0-1, vyšší = větší pravděpodobnost
    Reasoning   string    // Proč je podezřelý
    DiffPreview string    // Náhled relevantního diffu
}

// ChunkEvolution - evoluce chunku v čase
type ChunkEvolution struct {
    ChunkID      string
    CreatedAt    time.Time
    CreatedBy    string
    LastModified time.Time
    ModifyCount  int
    Authors      []string
    History      []ChunkHistoryEntry
}
```

## Plugin Interfaces

### EmbeddingProvider

```go
type EmbeddingProvider interface {
    Name() string
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
    MaxBatchSize() int
}
```

### ChunkingStrategy

```go
type ChunkingStrategy interface {
    Name() string
    Chunk(file *SourceFile) ([]*Chunk, error)
    SupportedLanguages() []string
    ExtractSymbols(file *SourceFile) ([]*Symbol, error)
    ExtractReferences(file *SourceFile) ([]*Reference, error)
}
```

### Reranker

```go
type Reranker interface {
    Name() string
    Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)
    MaxDocuments() int  // max počet dokumentů pro reranking (typicky 100-200)
}

type RerankResult struct {
    Index int       // původní pozice v input slice
    Score float32   // relevance score (vyšší = relevantnější)
}
```

### VectorStore

```go
type VectorStore interface {
    Name() string
    Init(path string) error
    Close() error

    // Chunk operations
    StoreChunks(chunks []*ChunkWithEmbedding) error
    DeleteChunksByFile(filePath string) error

    // Search - hybridní (BM25 + vector)
    Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error)

    // Symbol operations
    StoreSymbols(symbols []*Symbol) error
    GetSymbol(id string) (*Symbol, error)
    FindSymbols(query string, kind SymbolKind, limit int) ([]*Symbol, error)

    // Reference operations
    StoreReferences(refs []*Reference) error
    GetCallers(symbolID string, limit int) ([]*Reference, error)
    GetCallees(symbolID string, limit int) ([]*Reference, error)

    // Maintenance
    GetStats() (*StoreStats, error)
}

type SearchRequest struct {
    Query      string      // textový dotaz (pro BM25 a reranking)
    QueryVec   []float32   // embedding dotazu (pro vector search)
    Limit      int
    Filters    *SearchFilters
    Mode       SearchMode  // vector, bm25, hybrid

    // Hybrid search weights
    VectorWeight float32   // default 0.7
    BM25Weight   float32   // default 0.3

    // Reranking
    UseReranker      bool  // default true pokud je reranker configured
    RerankCandidates int   // kolik kandidátů pro reranking (default 100)

    // Context
    IncludeContext bool    // vrátit okolní řádky
    ContextLines   int     // default 5
}

type SearchMode string

const (
    SearchModeVector SearchMode = "vector"
    SearchModeBM25   SearchMode = "bm25"
    SearchModeHybrid SearchMode = "hybrid"  // default
)

type SearchFilters struct {
    Languages  []string   // filter by language
    ChunkTypes []ChunkType
    FilePaths  []string   // glob patterns
}

type SearchResult struct {
    Chunk         *Chunk
    Score         float32   // finální skóre (po rerankingu pokud enabled)
    VectorScore   float32   // pro debugging
    BM25Score     float32   // pro debugging
    RerankScore   float32   // pro debugging (0 pokud reranking disabled)
    ContextBefore string    // řádky před chunkem (pokud IncludeContext=true)
    ContextAfter  string    // řádky za chunkem
}

type StoreStats struct {
    TotalChunks    int
    TotalSymbols   int
    TotalReferences int
    IndexedFiles   int
    LastIndexed    time.Time
    DBSizeBytes    int64
}
```

## MCP Tools

### Setup & Konfigurace (Fáze 1)

| Tool | Popis |
|------|-------|
| `detect_environment` | Detekuj dostupné providery (Ollama, modely) a projekt |
| `get_config` | Vrať aktuální konfiguraci |
| `set_config` | Uprav hodnoty v konfiguraci |
| `validate_config` | Validuj config a testuj spojení s providery |

### Základní (Fáze 1)

| Tool | Popis |
|------|-------|
| `index_codebase` | Indexuj aktuální projekt |
| `search_code` | Sémantické vyhledávání v kódu |
| `get_chunk` | Vrať konkrétní chunk s kontextem |
| `clear_index` | Smaž index |
| `get_status` | Stav indexace |

### Analýza (Fáze 2-4)

| Tool | Popis |
|------|-------|
| `get_callers` | Kdo volá daný symbol |
| `get_callees` | Co daný symbol volá |
| `get_entry_points` | Hlavní vstupní body (main, handlers) |
| `get_import_graph` | Graf závislostí mezi moduly |
| `get_symbols` | Seznam symbolů s filtry |

### Pokročilé (Fáze 5+)

| Tool | Popis |
|------|-------|
| `get_type_at` | Typ výrazu na dané pozici |
| `get_complexity` | Metriky složitosti |
| `get_patterns` | Detekované design patterns |
| `get_dead_code` | Nepoužívaný kód |
| `analyze_impact` | Dopad změny symbolu |

### Git Historie (Temporální vyhledávání)

Tyto nástroje umožňují vyhledávání v historii kódu - sledování změn v čase, hledání regresí a analýzu evoluce kódu.

| Tool | Popis |
|------|-------|
| `index_git_history` | Indexuj git historii (commity, změny, diff embeddings) |
| `search_history` | Sémantické vyhledávání v historii změn |
| `get_chunk_history` | Historie změn konkrétního chunku/funkce |
| `get_code_evolution` | Evoluce symbolu v čase (kdy vznikl, kdo měnil) |
| `find_regression` | Najdi commity které mohly zavést chybu |
| `get_commit_context` | Detaily commitu včetně změn a affected funkcí |
| `get_contributor_insights` | Kdo je expert na danou část kódu |
| `get_git_history_status` | Stav indexace git historie |

#### Příklady použití

##### 1. Indexace git historie

Před použitím temporálního vyhledávání je nutné naindexovat git historii:

```
User: "Naindexuj git historii tohoto projektu"

Tool: index_git_history
Params: {
  "force": false  // inkrementální - jen nové commity
}

Response:
{
  "success": true,
  "commits_indexed": 156,
  "changes_indexed": 423,
  "time_range": "2024-01-15 to 2024-12-29",
  "last_commit": "abc123d - Fix null pointer in parser"
}
```

##### 2. Hledání regrese (Bug Hunting)

Když víte, že chyba vznikla někdy mezi dvěma verzemi:

```
User: "Aplikace padá při přihlášení. Fungovalo to ve verzi 1.0, ale v 1.1 už ne."

Tool: find_regression
Params: {
  "description": "crash on login, null pointer, authentication failure",
  "known_good": "v1.0",
  "known_bad": "v1.1",
  "limit": 5
}

Response:
{
  "description": "crash on login, null pointer, authentication failure",
  "range": "v1.0..v1.1",
  "count": 3,
  "candidates": [
    {
      "rank": 1,
      "severity": "HIGH",
      "score": 0.85,
      "commit": "def456",
      "date": "2024-12-15 14:30",
      "author": "Jan Novák",
      "message": "Refactor auth middleware",
      "reasoning": "Změny v auth.go ovlivňují login flow",
      "changed_files": ["internal/auth/middleware.go", "internal/auth/session.go"],
      "diff_preview": "-    if user != nil {\n+    if user.IsValid() {"
    },
    ...
  ],
  "recommendation": "Start investigating commit def456 - Refactor auth middleware"
}
```

##### 3. Evoluce symbolu/funkce

Zjistěte historii konkrétní funkce nebo typu:

```
User: "Kdy vznikla funkce ValidateToken a kdo na ní pracoval?"

Tool: get_code_evolution
Params: {
  "symbol": "ValidateToken"
}

Response:
{
  "symbol": "ValidateToken",
  "file": "internal/auth/token.go",
  "created_at": "2024-03-15",
  "created_by": "Petr Svoboda",
  "last_modified": "2024-12-10",
  "modify_count": 8,
  "authors": ["Petr Svoboda", "Jan Novák", "Eva Malá"],
  "history": [
    {
      "commit": "abc123",
      "date": "2024-12-10",
      "author": "Jan Novák",
      "change_type": "M",
      "summary": "Add token expiration check"
    },
    {
      "commit": "def456",
      "date": "2024-11-20",
      "author": "Eva Malá",
      "change_type": "M",
      "summary": "Fix JWT validation"
    },
    ...
  ]
}
```

##### 4. Sémantické vyhledávání v historii

Hledejte změny podle popisu, ne podle klíčových slov:

```
User: "Najdi všechny změny související s databázovým připojením za poslední 2 měsíce"

Tool: search_history
Params: {
  "query": "database connection pooling, SQL queries, db initialization",
  "time_from": "2024-10-29",
  "limit": 10
}

Response:
{
  "query": "database connection pooling...",
  "results": [
    {
      "commit": "abc123d",
      "date": "2024-12-20",
      "author": "Jan Novák",
      "message": "Increase connection pool size for production",
      "score": 0.92,
      "changes": [
        {
          "file": "internal/db/pool.go",
          "type": "M",
          "affected_functions": ["NewPool", "GetConnection"]
        }
      ]
    },
    ...
  ]
}
```

##### 5. Filtrování podle autora nebo cesty

```
User: "Co změnil Jan Novák v autentizačním modulu?"

Tool: search_history
Params: {
  "query": "authentication changes",
  "authors": ["Jan Novák"],
  "paths": ["internal/auth/**"],
  "limit": 20
}
```

##### 6. Historie konkrétního chunku

Detailní historie změn jednoho bloku kódu:

```
User: "Ukaž mi historii změn funkce handleRequest v handlers.go"

Tool: get_chunk_history
Params: {
  "file": "internal/api/handlers.go",
  "function": "handleRequest",
  "limit": 10
}

Response:
{
  "chunk_id": "internal/api/handlers.go:45:abc123",
  "function": "handleRequest",
  "total_changes": 12,
  "history": [
    {
      "commit": "abc123d",
      "date": "2024-12-28",
      "author": "Eva Malá",
      "change_type": "modified",
      "lines_added": 5,
      "lines_removed": 2,
      "diff_summary": "Added error logging"
    },
    ...
  ]
}
```

##### 7. Detail commitu

Získejte kompletní kontext commitu:

```
User: "Ukaž mi detaily commitu abc123d včetně všech změn"

Tool: get_commit_context
Params: {
  "commit": "abc123d",
  "include_diff": true
}

Response:
{
  "hash": "abc123def456789...",
  "short_hash": "abc123d",
  "author": "Jan Novák",
  "author_email": "jan@example.com",
  "date": "2024-12-28 15:30:00",
  "message": "Fix connection leak in database pool\n\nThe connection was not properly returned to pool on error.",
  "parent": "xyz789a",
  "is_merge": false,
  "files_changed": 3,
  "insertions": 25,
  "deletions": 8,
  "changes": [
    {
      "file": "internal/db/pool.go",
      "change_type": "M",
      "additions": 15,
      "deletions": 5,
      "affected_functions": ["GetConnection", "releaseConnection"],
      "diff": "@@ -45,10 +45,20 @@\n func GetConnection()..."
    },
    ...
  ]
}
```

##### 8. Experti na kód (Contributor Insights)

Zjistěte, kdo je expert na danou část kódu:

```
User: "Kdo rozumí autentizačnímu modulu nejlépe?"

Tool: get_contributor_insights
Params: {
  "paths": ["internal/auth/**"]
}

Response:
{
  "path_filter": "internal/auth/**",
  "insights": [
    {
      "author": "Petr Svoboda",
      "email": "petr@example.com",
      "commit_count": 45,
      "lines_changed": 1250,
      "first_commit": "2024-01-15",
      "last_commit": "2024-12-20",
      "expertise_score": 0.85,
      "top_files": ["auth/token.go", "auth/middleware.go", "auth/session.go"]
    },
    {
      "author": "Jan Novák",
      "email": "jan@example.com",
      "commit_count": 23,
      "lines_changed": 580,
      "expertise_score": 0.65,
      ...
    }
  ]
}
```

##### 9. Stav git indexace

Zkontrolujte stav indexace historie:

```
User: "Jaký je stav indexace git historie?"

Tool: get_git_history_status

Response:
{
  "indexed": true,
  "total_commits": 156,
  "total_changes": 423,
  "chunk_history_entries": 1250,
  "first_commit": "2024-01-15",
  "last_commit": "2024-12-29",
  "last_indexed_commit": "abc123d",
  "commits_with_embeddings": 156,
  "changes_with_embeddings": 89,
  "index_size_mb": 12.5
}
```

##### 10. Kombinované workflow: Debugging regrese

Kompletní příklad hledání a analýzy chyby:

```
# Krok 1: Najdi podezřelé commity
User: "API vrací 500 na /users endpoint. Minulý týden to fungovalo."

Tool: find_regression
Params: {
  "description": "500 error on /users endpoint, API failure",
  "known_good": "HEAD~20",
  "known_bad": "HEAD"
}
→ Podezřelý commit: def456 "Refactor user service"

# Krok 2: Prozkoumej commit
Tool: get_commit_context
Params: {"commit": "def456", "include_diff": true}
→ Změny v user_service.go a repository.go

# Krok 3: Kdo je expert na tento kód?
Tool: get_contributor_insights
Params: {"paths": ["internal/user/**"]}
→ Expert: Petr Svoboda

# Krok 4: Historie problematické funkce
Tool: get_chunk_history
Params: {"file": "internal/user/service.go", "function": "GetUsers"}
→ Funkce byla změněna 3x za poslední týden

# Závěr: Commit def456 pravděpodobně zavedl chybu v GetUsers,
# kontaktujte Petra Svobodu pro review.
```

##### 11. Refactoring tracking

Sledujte průběh refactoringu:

```
User: "Ukaž mi všechny změny související s migrací na nové API v posledním měsíci"

Tool: search_history
Params: {
  "query": "API migration, new endpoints, deprecated methods removal",
  "time_from": "2024-11-29",
  "change_types": ["M", "A", "D"]
}

# Poté pro každý zajímavý commit:
Tool: get_commit_context
Params: {"commit": "<hash>", "include_diff": true}
```

### Memory Tools (Persistent Context)

Nástroje pro ukládání a vyhledávání znalostí/kontextu s podporou sémantického vyhledávání.
Paměť je uložena projektově a je izolována podle větví (channel = git branch).

| Tool | Popis |
|------|-------|
| `memory_store` | Ulož novou paměť/znalost |
| `memory_recall` | Vyhledej paměti pomocí sémantického search |
| `memory_forget` | Smaž konkrétní paměť |
| `memory_checkpoint` | Vytvoř snapshot paměti (záloha) |
| `memory_restore` | Obnov paměti z checkpointu |
| `memory_stats` | Statistiky paměti |

#### Kategorie pamětí

- `decision` - Rozhodnutí (architektura, design choices)
- `context` - Kontext projektu (jak věci fungují)
- `fact` - Fakta (API klíče, endpointy, konfigurace)
- `note` - Poznámky (obecné)
- `error` - Chyby a jejich řešení
- `review` - Code review poznatky

#### Příklady použití

```
# Ulož důležité rozhodnutí
Tool: memory_store
Params: {
  "content": "Rozhodli jsme se použít JWT pro autentizaci kvůli stateless API",
  "category": "decision",
  "tags": ["auth", "api", "architecture"],
  "importance": 0.9
}

# Vyhledej relevantní kontext
Tool: memory_recall
Params: {
  "query": "jak funguje autentizace v API",
  "limit": 5
}

# Vytvoř zálohu před velkými změnami
Tool: memory_checkpoint
Params: {
  "name": "before-refactoring",
  "description": "Snapshot před refaktoringem auth modulu"
}
```

### Todo Tools (Task Management)

Nástroje pro správu úkolů s podporou hierarchie (subtasky), priorit a sémantického vyhledávání.
Úkoly jsou izolované podle větví (channel = git branch).

| Tool | Popis |
|------|-------|
| `todo_create` | Vytvoř nový úkol |
| `todo_list` | Seznam úkolů s filtry |
| `todo_search` | Sémantické vyhledávání úkolů |
| `todo_update` | Aktualizuj úkol (status, priority, progress) |
| `todo_complete` | Označ úkol jako dokončený |
| `todo_delete` | Smaž úkol |
| `todo_stats` | Statistiky úkolů |

#### Priority

- `urgent` - Kritické, okamžitě
- `high` - Vysoká priorita
- `medium` - Standardní (default)
- `low` - Nízká priorita

#### Statusy

- `pending` - Čeká na zpracování
- `in_progress` - Právě se řeší
- `completed` - Dokončeno
- `cancelled` - Zrušeno
- `blocked` - Blokováno

#### Příklady použití

```
# Vytvoř úkol s subtasky
Tool: todo_create
Params: {
  "title": "Implementovat user authentication",
  "description": "Přidat JWT auth do API",
  "priority": "high",
  "tags": ["auth", "api"]
}
→ id: "todo_123"

Tool: todo_create
Params: {
  "title": "Vytvořit login endpoint",
  "parent_id": "todo_123",
  "priority": "medium"
}

Tool: todo_create
Params: {
  "title": "Přidat middleware pro validaci tokenů",
  "parent_id": "todo_123",
  "priority": "medium"
}

# Seznam aktivních úkolů
Tool: todo_list
Params: {
  "statuses": ["pending", "in_progress"],
  "limit": 20
}

# Označ úkol jako rozpracovaný
Tool: todo_update
Params: {
  "id": "todo_123",
  "status": "in_progress",
  "progress": 30
}

# Vyhledej úkoly
Tool: todo_search
Params: {
  "query": "authentication API security"
}

# Statistiky
Tool: todo_stats
Params: {}
→ {
  "total": 15,
  "by_status": {"pending": 8, "in_progress": 3, "completed": 4},
  "by_priority": {"high": 5, "medium": 7, "low": 3},
  "overdue": 2
}
```

## Setup Wizard (interaktivní konfigurace přes MCP)

AI agent může provést uživatele nastavením přímo z promptu:

### detect_environment

```go
type DetectEnvironmentResult struct {
    Ollama struct {
        Available    bool          `json:"available"`
        Endpoint     string        `json:"endpoint"`
        Models       []ModelInfo   `json:"models"`
        TotalVRAM    string        `json:"total_vram,omitempty"`    // "8GB"
        AvailableRAM string        `json:"available_ram,omitempty"`
        Error        string        `json:"error,omitempty"`
    } `json:"ollama"`

    OpenAI struct {
        Available bool   `json:"available"`
        Error     string `json:"error,omitempty"`
    } `json:"openai"`

    System struct {
        OS           string `json:"os"`
        Arch         string `json:"arch"`
        CPUCores     int    `json:"cpu_cores"`
        TotalRAM     string `json:"total_ram"`
        AvailableRAM string `json:"available_ram"`
        HasGPU       bool   `json:"has_gpu"`
        GPUInfo      string `json:"gpu_info,omitempty"`
    } `json:"system"`

    Project struct {
        Path              string            `json:"path"`
        IsGit             bool              `json:"is_git"`
        LanguagesDetected map[string]int    `json:"languages_detected"`
        FileCount         int               `json:"file_count"`
        TotalLines        int               `json:"total_lines"`
        EstimatedSize     string            `json:"estimated_size"`
        EstimatedChunks   int               `json:"estimated_chunks"`
        Complexity        ProjectComplexity `json:"complexity"`  // small, medium, large, huge
    } `json:"project"`

    ExistingConfig *Config        `json:"existing_config"`
    ExistingIndex  *IndexMetadata `json:"existing_index"`

    Recommendations *ModelRecommendations `json:"recommendations"`
}

type ModelInfo struct {
    Name       string `json:"name"`
    Size       string `json:"size"`        // "600MB"
    Type       string `json:"type"`        // "embedding", "reranker", "llm"
    Loaded     bool   `json:"loaded"`      // již v paměti?
    Recommended bool  `json:"recommended"` // doporučený pro tento projekt?
}

type ProjectComplexity string

const (
    ComplexitySmall  ProjectComplexity = "small"   // < 100 souborů
    ComplexityMedium ProjectComplexity = "medium"  // 100-1000 souborů
    ComplexityLarge  ProjectComplexity = "large"   // 1000-10000 souborů
    ComplexityHuge   ProjectComplexity = "huge"    // > 10000 souborů
)
```

### ModelRecommendations

```go
type ModelRecommendations struct {
    // Primární doporučení
    Primary RecommendationSet `json:"primary"`

    // Alternativy
    LowMemory  *RecommendationSet `json:"low_memory,omitempty"`   // pro málo RAM/VRAM
    HighQuality *RecommendationSet `json:"high_quality,omitempty"` // nejlepší kvalita
    FastIndex  *RecommendationSet `json:"fast_index,omitempty"`   // rychlá indexace

    // Vysvětlení
    Reasoning []string `json:"reasoning"`  // proč tato doporučení
}

type RecommendationSet struct {
    Embedding EmbeddingRecommendation `json:"embedding"`
    Reranker  RerankerRecommendation  `json:"reranker"`
    Chunking  ChunkingRecommendation  `json:"chunking"`
    Indexing  IndexingRecommendation  `json:"indexing"`
}

type EmbeddingRecommendation struct {
    Provider    string `json:"provider"`     // "ollama", "openai"
    Model       string `json:"model"`        // "nomic-embed-code"
    Dimensions  int    `json:"dimensions"`   // 768
    BatchSize   int    `json:"batch_size"`   // 32
    Reason      string `json:"reason"`       // "Optimální pro code, běží lokálně"

    // Pro code-specific projekty
    CodeOptimized bool `json:"code_optimized"`
}

type RerankerRecommendation struct {
    Enabled    bool   `json:"enabled"`
    Provider   string `json:"provider,omitempty"`
    Model      string `json:"model,omitempty"`
    Candidates int    `json:"candidates"`    // 100
    Reason     string `json:"reason"`
}

type ChunkingRecommendation struct {
    Strategy     string   `json:"strategy"`       // "treesitter", "simple"
    MaxChunkSize int      `json:"max_chunk_size"` // 2000
    Languages    []string `json:"languages"`      // podporované jazyky v projektu
    Reason       string   `json:"reason"`
}

type IndexingRecommendation struct {
    Workers          int    `json:"workers"`           // paralelní workers
    EstimatedTime    string `json:"estimated_time"`    // "~5 minutes"
    EstimatedDBSize  string `json:"estimated_db_size"` // "~50MB"
    UseGitIgnore     bool   `json:"use_gitignore"`
    Reason           string `json:"reason"`
}
```

### Logika doporučení

```go
func generateRecommendations(env *DetectEnvironmentResult) *ModelRecommendations {
    rec := &ModelRecommendations{}

    // Základní doporučení podle velikosti projektu a dostupných zdrojů
    switch {
    case env.Project.Complexity == ComplexitySmall:
        // Malý projekt: plná kvalita
        rec.Primary = fullQualitySet()

    case env.Project.Complexity == ComplexityLarge && !env.System.HasGPU:
        // Velký projekt bez GPU: optimalizovat pro rychlost
        rec.Primary = balancedSet()
        rec.Reasoning = append(rec.Reasoning,
            "Velký projekt bez GPU - doporučuji menší batch size")

    case env.Project.Complexity == ComplexityHuge:
        // Obrovský projekt: varování + optimalizace
        rec.Primary = fastIndexSet()
        rec.Reasoning = append(rec.Reasoning,
            "Velmi velký projekt (>10k souborů) - doporučuji inkrementální indexaci")
    }

    // Doporučení podle jazyků
    if hasCodeLanguages(env.Project.LanguagesDetected) {
        rec.Primary.Embedding.CodeOptimized = true
        rec.Primary.Embedding.Model = "nomic-embed-code"
        rec.Reasoning = append(rec.Reasoning,
            "Detekován kód - používám code-optimized embedding model")
    }

    // Doporučení podle dostupné paměti
    if env.System.AvailableRAM < "4GB" || !env.System.HasGPU {
        rec.LowMemory = lowMemorySet()
        rec.Reasoning = append(rec.Reasoning,
            "Omezená paměť - připravena low-memory alternativa")
    }

    // Alternativy
    rec.HighQuality = highQualitySet()  // voyage-code-3 + větší reranker
    rec.FastIndex = fastIndexSet()       // bez rerankeru, větší batch

    return rec
}
```

### Příklad výstupu detect_environment

```json
{
  "ollama": {
    "available": true,
    "endpoint": "http://localhost:11434",
    "models": [
      {"name": "nomic-embed-code", "size": "500MB", "type": "embedding", "recommended": true},
      {"name": "qwen3-reranker", "size": "600MB", "type": "reranker", "recommended": true},
      {"name": "llama3", "size": "4GB", "type": "llm", "recommended": false}
    ],
    "total_vram": "8GB"
  },
  "system": {
    "os": "darwin",
    "arch": "arm64",
    "cpu_cores": 10,
    "total_ram": "32GB",
    "available_ram": "16GB",
    "has_gpu": true,
    "gpu_info": "Apple M1 Pro"
  },
  "project": {
    "path": "/home/user/myproject",
    "is_git": true,
    "languages_detected": {"go": 150, "python": 30, "markdown": 20},
    "file_count": 200,
    "total_lines": 45000,
    "estimated_size": "2.5MB",
    "estimated_chunks": 800,
    "complexity": "medium"
  },
  "recommendations": {
    "primary": {
      "embedding": {
        "provider": "ollama",
        "model": "nomic-embed-code",
        "dimensions": 768,
        "batch_size": 32,
        "code_optimized": true,
        "reason": "Optimální pro Go/Python kód, běží lokálně na GPU"
      },
      "reranker": {
        "enabled": true,
        "provider": "ollama",
        "model": "qwen3-reranker",
        "candidates": 100,
        "reason": "Zlepší přesnost search o ~15%, dostatek VRAM"
      },
      "chunking": {
        "strategy": "treesitter",
        "max_chunk_size": 2000,
        "languages": ["go", "python"],
        "reason": "TreeSitter dostupný pro Go i Python"
      },
      "indexing": {
        "workers": 8,
        "estimated_time": "~2 minuty",
        "estimated_db_size": "~25MB",
        "use_gitignore": true,
        "reason": "Střední projekt, paralelní indexace na 8 jádrech"
      }
    },
    "low_memory": {
      "embedding": {"provider": "ollama", "model": "nomic-embed-code", "batch_size": 8},
      "reranker": {"enabled": false, "reason": "Ušetří ~600MB VRAM"},
      "chunking": {"strategy": "simple"},
      "indexing": {"workers": 4}
    },
    "high_quality": {
      "embedding": {"provider": "openai", "model": "text-embedding-3-large"},
      "reranker": {"enabled": true, "model": "qwen3-reranker-4b", "candidates": 200},
      "chunking": {"strategy": "treesitter"},
      "indexing": {"workers": 8}
    },
    "reasoning": [
      "Detekován Go/Python projekt - používám code-optimized modely",
      "Dostatek VRAM (8GB) - povolený reranker pro lepší kvalitu",
      "Střední velikost projektu - standardní nastavení",
      "TreeSitter podporuje všechny detekované jazyky"
    ]
  }
}
```

### set_config

```go
type SetConfigRequest struct {
    // Dot notation pro nested hodnoty: "reranker.enabled", "index.include"
    Changes map[string]any `json:"changes"`
}

type SetConfigResult struct {
    Success      bool     `json:"success"`
    Changed      []string `json:"changed"`       // které hodnoty se změnily
    Warnings     []string `json:"warnings"`      // např. "search quality may decrease"
    NeedsReindex bool     `json:"needs_reindex"` // změna vyžaduje reindexaci
    Config       *Config  `json:"config"`        // nová konfigurace
}
```

### validate_config

```go
type ValidateConfigResult struct {
    Valid    bool              `json:"valid"`
    Errors   []string          `json:"errors"`
    Warnings []string          `json:"warnings"`
    Tests    map[string]TestResult `json:"tests"`
}

type TestResult struct {
    Status  string `json:"status"`   // "ok", "error", "warning", "skipped"
    Message string `json:"message"`
}
```

### Příklady interakce

**První nastavení:**
```
User: "Nastav indexování pro tento projekt"

AI: [detect_environment]

AI: "Analyzoval jsem tvůj projekt:

📊 Projekt: 200 souborů (Go 75%, Python 15%), ~45k řádků
💻 Systém: Apple M1 Pro, 32GB RAM, 8GB VRAM
🔧 Ollama: běží, modely nomic-embed-code a qwen3-reranker dostupné

Doporučuji nastavení 'primary':
• Embedding: nomic-embed-code (lokální, optimalizovaný pro kód)
• Reranker: Qwen3 (zlepší přesnost o ~15%)
• Chunking: TreeSitter (Go + Python podporován)
• Odhadovaný čas indexace: ~2 minuty

Alternativy:
• 'low_memory' - bez rerankeru, ušetří 600MB VRAM
• 'high_quality' - OpenAI embeddings + větší reranker

Které nastavení použít?"

User: "Použij primary"

AI: [set_config] → [validate_config] → [index_codebase]
```

**Výběr podle potřeb:**
```
User: "Mám starší notebook, málo paměti"

AI: [detect_environment]

AI: "Vidím že máš 8GB RAM a integrovanou grafiku.

Doporučuji 'low_memory' nastavení:
• Embedding: nomic-embed-code s batch_size 8
• Reranker: vypnutý (ušetří 600MB)
• Chunking: simple (rychlejší, méně paměti)
• Workers: 4 (šetříme CPU)

Search bude o něco méně přesný, ale poběží plynule. OK?"
```

**Troubleshooting:**
```
User: "Search nefunguje"

AI: [validate_config]

AI: "Našel jsem problém:

❌ Ollama: neběží na localhost:11434
✅ Config: validní
✅ Index: existuje (800 chunků)

Řešení: Spusť 'ollama serve' v terminálu.

Nebo chceš přepnout na OpenAI embeddings? (vyžaduje API klíč)"
```

**Změna modelu:**
```
User: "Chci lepší kvalitu, mám OpenAI API klíč"

AI: [detect_environment]

AI: "Pro vyšší kvalitu doporučuji:

Aktuální → Nové nastavení:
• Embedding: nomic-embed-code → text-embedding-3-large (+20% kvalita)
• Reranker: qwen3-reranker → voyage-rerank-2 (nejlepší pro kód)
• Cena: ~$0.02 za reindexaci tohoto projektu

⚠️ Změna embedding modelu vyžaduje reindexaci celého projektu.

Pokračovat?"

User: "Ano"

AI: [set_config] → [validate_config] → [index_codebase --force]
```

**Přidání jazyka:**
```
User: "Přidej Rust soubory"

AI: [get_config] → [detect_environment]

AI: "Přidávám Rust:

• Nalezeno: 45 nových .rs souborů
• TreeSitter: Rust grammar dostupný ✅
• Odhadovaný čas: +30 sekund

Přidám '**/*.rs' do indexu a spustím reindexaci?"
```

**Optimalizace pro velký projekt:**
```
User: "Mám monorepo s 50k soubory"

AI: [detect_environment]

AI: "⚠️ Velmi velký projekt detekován.

Doporučení pro monorepo:
• Indexovat jen relevantní složky (ne celé repo)
• Použít --exclude pro vendor/, node_modules/, generated/
• Rozdělit na menší indexy podle týmů/modulů?

Aktuální odhad:
• Čas indexace: ~45 minut
• Velikost DB: ~500MB
• Doporučený reranker: vypnout (příliš pomalý pro 50k chunků)

Chceš pomoct s nastavením exclude patterns?"
```

## Konfigurace

```yaml
# .mcp-codewizard/config.yaml

embedding:
  provider: ollama           # ollama | openai | voyage | jina
  model: nomic-embed-code    # model specifický pro provider
  endpoint: http://localhost:11434  # Ollama endpoint
  batch_size: 32             # dokumentů na batch
  # api_key: ${OPENAI_API_KEY}  # pro API providery

chunking:
  strategy: treesitter       # treesitter | simple (fallback)
  max_chunk_size: 2000       # max tokenů na chunk

reranker:
  enabled: true
  provider: ollama           # ollama | none
  model: qwen3-reranker      # Qwen3-Reranker-0.6B
  endpoint: http://localhost:11434  # Ollama endpoint (může být jiný než embedding)
  candidates: 100            # kolik kandidátů pro reranking

search:
  mode: hybrid               # vector | bm25 | hybrid
  vector_weight: 0.7
  bm25_weight: 0.3
  default_limit: 10

vectorstore:
  provider: sqlitevec        # sqlitevec | (budoucí: lancedb, qdrant)

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
    - "**/*.h"
  exclude:
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/.git/**"
    - "**/dist/**"
    - "**/build/**"
    - "**/*.min.js"
    - "**/*.generated.*"
  use_gitignore: true        # respektovat .gitignore (použije git ls-files)

limits:
  max_file_size: 1MB         # přeskočit větší soubory
  max_files: 50000           # max počet souborů
  max_chunk_tokens: 2000     # max velikost jednoho chunku
  timeout: 30m               # max doba indexace

analysis:
  extract_symbols: true
  extract_references: true
  detect_patterns: false     # zatím disabled

logging:
  level: info                # debug | info | warn | error
  format: text               # text | json
```

## Implementační fáze

### Fáze 1: Základ (MVP)
- [x] Projekt setup (go.mod, struktura)
- [x] Provider interfaces a typy (pkg/provider, pkg/types)
- [x] sqlite-vec vector store (builtin/vectorstore/sqlitevec)
- [x] Ollama embedding provider (builtin/embedding/ollama)
- [x] Ollama reranker provider (builtin/reranker/ollama) - Qwen3-Reranker
- [x] Simple chunking (builtin/chunking/simple)
- [x] TreeSitter chunking s fallback (builtin/chunking/treesitter)
- [x] Hybridní search (BM25 + vector + reranking)
- [x] Parallel indexer s progress reporting
- [x] Inkrementální indexace (file hash cache)
- [x] MCP server s basic tools (modelcontextprotocol/go-sdk)
- [x] MCP Setup Wizard tools (detect_environment, get/set/validate_config)
- [x] CLI (cobra): index, search, status, serve, config
- [x] Graceful shutdown + checkpoint resume

### Fáze 2: Analýza
- [x] Extrakce symbolů (TreeSitter)
- [x] Extrakce referencí
- [x] Call graph (get_callers, get_callees)
- [x] Import graph
- [x] Entry points detection

### Fáze 3: Rozšíření
- [x] OpenAI embedding provider
- [x] Watch mode (fsnotify)
- [x] go-plugin pro externí pluginy
- [x] Více jazyků (Rust, Java, C/C++)

### Fáze 4+: Pokročilé (budoucnost)
- [ ] Voyage/Jina embedding providery
- [ ] Type flow analysis
- [ ] Pattern detection
- [x] Complexity metrics
- [x] Dead code detection
- [x] Git blame integration

### Fáze 5: Git Historie (Temporální vyhledávání)
- [x] Git history typy (Commit, Change, HistorySearchRequest)
- [x] GitHistoryStore interface
- [x] SQLite schema pro git historii (commits, changes, chunk_history)
- [x] GitHistoryAnalyzer (git log, git diff parsing)
- [x] Store implementace (CRUD + search v historii)
- [x] MCP Tools (search_history, get_chunk_history, find_regression, atd.)
- [x] Integrace do indexeru + inkrementální git indexace
- [x] Základní testy

## Příkazy

```bash
# Indexace
mcp-codewizard index [path]           # indexuj projekt
mcp-codewizard index --dry-run        # ukaž co by se indexovalo
mcp-codewizard index --force          # reindexuj vše (ignoruj cache)

# Vyhledávání (CLI)
mcp-codewizard search "query"                    # hybrid search
mcp-codewizard search "query" --mode vector      # jen vector search
mcp-codewizard search "query" --no-rerank        # bez rerankingu
mcp-codewizard search "query" --limit 20         # víc výsledků

# Status
mcp-codewizard status                 # stav indexu
mcp-codewizard status --verbose       # detailní statistiky

# MCP server
mcp-codewizard serve                  # spustí dlouhoběžící daemon
mcp-codewizard serve --stdio          # MCP přes stdio (pro Claude Code)

# Watch mode (automatická reindexace)
mcp-codewizard watch [path]           # sleduj změny a reindexuj
mcp-codewizard watch --debounce 1000  # debounce v ms (default: 500)

# Konfigurace
mcp-codewizard config init            # vytvoř default config
mcp-codewizard config validate        # zkontroluj config

# Plugin management
mcp-codewizard plugin list                           # seznam dostupných pluginů
mcp-codewizard plugin load <name> <type>             # načti plugin (type: embedding, reranker)
```

### Globální flagy

```bash
--config, -c     # cesta ke config souboru (default: .mcp-codewizard/config.yaml)
--log-level      # debug | info | warn | error
--log-format     # text | json
```

## Parallel Indexing

Indexace využívá všechna dostupná jádra:

```go
type IndexerConfig struct {
    Workers        int           // default: runtime.NumCPU()
    BatchSize      int           // chunks per embedding batch (default: 32)
    ProgressFunc   func(IndexProgress)
}

type IndexProgress struct {
    Phase          string        // "scanning", "chunking", "embedding", "storing"
    TotalFiles     int
    ProcessedFiles int
    TotalChunks    int
    ProcessedChunks int
    CurrentFile    string
    Error          error         // non-fatal error (např. nelze parsovat soubor)
}
```

**Pipeline:**
```
Files → [Chunking Workers] → Chunks → [Embedding Batches] → [Store]
              ↓                              ↓
         Symbols/Refs                   Parallel API calls
```

## Inkrementální indexace

Cache klíč pro každý soubor:
```
{file_path} → {
    file_hash: SHA256(content),
    config_hash: SHA256(embedding_model + chunking_strategy + chunk_size),
    indexed_at: timestamp
}
```

Pokud se změní embedding model nebo chunking strategie → reindexace všeho.

## Index Metadata

```go
type IndexMetadata struct {
    SchemaVersion  int           // pro detekci incompatible changes
    CreatedAt      time.Time
    LastUpdated    time.Time
    ToolVersion    string        // "0.1.0"
    ConfigHash     string        // hash konfigurace

    // Provider info
    EmbeddingProvider string     // "ollama"
    EmbeddingModel    string     // "nomic-embed-code"
    EmbeddingDimensions int      // 768
    ChunkingStrategy  string     // "treesitter"
    RerankerModel     string     // "qwen3-reranker" (nebo "")

    // Stats
    Stats StoreStats
}
```

Při startu: pokud `SchemaVersion` neodpovídá → vynutit reindexaci.

## Ollama konfigurace

### Paralelní běh modelů

Pro současný běh embedding a reranker modelu je potřeba nastavit Ollama:

```bash
# Povolit více modelů v paměti
export OLLAMA_NUM_PARALLEL=2
export OLLAMA_MAX_LOADED_MODELS=2
```

**Paměťové požadavky:**
| Model | VRAM/RAM |
|-------|----------|
| nomic-embed-code | ~500MB |
| qwen3-reranker-0.6B | ~600MB |
| **Celkem** | **~1.1GB** |

### Konfigurace endpointů

```yaml
# .mcp-codewizard/config.yaml

embedding:
  provider: ollama
  model: nomic-embed-code
  endpoint: http://localhost:11434    # default Ollama

reranker:
  enabled: true
  provider: ollama
  model: qwen3-reranker
  endpoint: http://localhost:11434    # stejná instance
  # endpoint: http://localhost:11435  # nebo separátní instance
```

### Separátní Ollama instance (optional)

Pro lepší izolaci nebo distribuci zátěže:

```bash
# Instance 1: Embedding (port 11434)
OLLAMA_HOST=127.0.0.1:11434 ollama serve

# Instance 2: Reranker (port 11435)
OLLAMA_HOST=127.0.0.1:11435 ollama serve
```

### Low-memory konfigurace

Pokud máš málo paměti (< 4GB VRAM):

```yaml
# Varianta 1: Disable reranker
reranker:
  enabled: false

# Varianta 2: Menší batch, méně kandidátů
reranker:
  enabled: true
  candidates: 30      # méně kandidátů = rychlejší

embedding:
  batch_size: 16      # menší batch = méně paměti
```

### Warm-up

Při startu MCP serveru se modely pre-loadují:

```go
func (s *Server) Start() error {
    // Pre-load modely do paměti Ollama
    if err := s.embedding.Warmup(ctx); err != nil {
        slog.Warn("embedding warmup failed", "error", err)
    }
    if s.reranker != nil {
        if err := s.reranker.Warmup(ctx); err != nil {
            slog.Warn("reranker warmup failed", "error", err)
        }
    }
    return s.serve()
}
```

### Batch optimalizace

Pipeline je navržený pro minimální model switching:

```
Indexing:
  [Všechny soubory] → Chunking → [Všechny chunky] → Embedding batch → Store
                                       ↑
                                  jeden model load

Search:
  Query → Embedding → Vector+BM25 search → [Top 100] → Rerank batch → [Top 10]
              ↑                                              ↑
         jeden call                                    jeden model load
```

## Závislosti

### Core
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) - oficiální MCP SDK
- [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin) - externí plugin systém

### Built-in providers (CGO)
- [asg017/sqlite-vec-go-bindings](https://github.com/asg017/sqlite-vec) - vektorová DB + FTS5
- [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter) - AST parsing

### Embedding clients
- [ollama/ollama/api](https://github.com/ollama/ollama) - Ollama client
- [sashabaranov/go-openai](https://github.com/sashabaranov/go-openai) - OpenAI client

### Utilities
- [spf13/cobra](https://github.com/spf13/cobra) - CLI
- [spf13/viper](https://github.com/spf13/viper) - config
- [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) - file watching (optional)

## Error Handling

### Graceful degradation
- Pokud Ollama neběží → jasná chybová hláška s instrukcemi
- Pokud embedding selže pro jeden soubor → přeskočit, logovat, pokračovat
- Pokud TreeSitter nepodporuje jazyk → fallback na Simple chunking

### Error typy
```go
var (
    ErrProviderNotAvailable = errors.New("provider not available")
    ErrIndexNotFound        = errors.New("index not found")
    ErrInvalidConfig        = errors.New("invalid configuration")
    ErrParseError           = errors.New("failed to parse file")
)
```

## Testing

### Unit testy
- Každý provider má vlastní testy
- Mock interfaces pro testování bez externích závislostí

### Integration testy
- Testovací fixtures (malé projekty v různých jazycích)
- Testy pro celý pipeline: index → search → results

### Benchmarks
```go
func BenchmarkIndexing(b *testing.B)     // chunks/sec
func BenchmarkSearch(b *testing.B)       // queries/sec
func BenchmarkHybridSearch(b *testing.B) // hybrid vs pure vector
func BenchmarkReranking(b *testing.B)    // rerank latency
```

### Search Quality Tests

```yaml
# testdata/search_quality/queries.yaml
- query: "HTTP handler for user authentication"
  project: go-simple
  must_include:
    - file: "handlers/auth.go"
      function: "HandleLogin"
  must_not_include:
    - file: "utils/string.go"
  min_score: 0.7

- query: "database connection pooling"
  project: go-simple
  must_include:
    - file: "db/pool.go"
```

Spuštění: `go test -tags=quality ./internal/search/...`

## Konvence

- Kód v angličtině, komentáře mohou být česky
- Všechny public funkce mají dokumentační komentáře
- Testy pro kritickou funkcionalitu
- Provider interfaces jsou stabilní, změny vyžadují major version bump
- Logování přes `log/slog` (structured logging)
