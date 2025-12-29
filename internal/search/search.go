// Package search implements hybrid search with reranking.
package search

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Engine handles search operations.
type Engine struct {
	store     provider.VectorStore
	embedding provider.EmbeddingProvider
	reranker  provider.Reranker // may be nil
}

// Config contains search engine configuration.
type Config struct {
	Store     provider.VectorStore
	Embedding provider.EmbeddingProvider
	Reranker  provider.Reranker // optional
}

// New creates a new search engine.
func New(cfg Config) *Engine {
	return &Engine{
		store:     cfg.Store,
		embedding: cfg.Embedding,
		reranker:  cfg.Reranker,
	}
}

// Search performs a search with the given request.
func (e *Engine) Search(ctx context.Context, req *types.SearchRequest) ([]*types.SearchResult, error) {
	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Mode == "" {
		req.Mode = types.SearchModeHybrid
	}
	if req.VectorWeight == 0 && req.BM25Weight == 0 {
		req.VectorWeight = 0.7
		req.BM25Weight = 0.3
	}

	// Generate query embedding for vector search
	if req.Mode == types.SearchModeVector || req.Mode == types.SearchModeHybrid {
		if len(req.QueryVec) == 0 && req.Query != "" {
			embeddings, err := e.embedding.Embed(ctx, []string{req.Query})
			if err != nil {
				return nil, fmt.Errorf("failed to embed query: %w", err)
			}
			req.QueryVec = embeddings[0]
		}
	}

	// Determine if we should rerank
	useReranker := e.reranker != nil && req.UseReranker
	if req.RerankCandidates == 0 {
		req.RerankCandidates = 100
	}

	// Get initial candidates
	candidates, err := e.store.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Apply reranking if enabled
	if useReranker && len(candidates) > 0 {
		candidates, err = e.rerank(ctx, req.Query, candidates)
		if err != nil {
			// Log warning but return non-reranked results
			slog.Warn("reranking failed, returning non-reranked results", "error", err)
		}
	}

	// Limit results
	if len(candidates) > req.Limit {
		candidates = candidates[:req.Limit]
	}

	// Add context if requested
	if req.IncludeContext {
		for _, result := range candidates {
			e.addContext(result, req.ContextLines)
		}
	}

	return candidates, nil
}

// rerank reranks candidates using the reranker.
func (e *Engine) rerank(ctx context.Context, query string, candidates []*types.SearchResult) ([]*types.SearchResult, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	// Prepare documents for reranking
	docs := make([]string, len(candidates))
	for i, c := range candidates {
		docs[i] = c.Chunk.Content
	}

	// Get rerank scores
	rerankResults, err := e.reranker.Rerank(ctx, query, docs)
	if err != nil {
		return nil, err
	}

	// Apply rerank scores
	scoreMap := make(map[int]float32)
	for _, rr := range rerankResults {
		scoreMap[rr.Index] = rr.Score
	}

	for i, c := range candidates {
		if score, ok := scoreMap[i]; ok {
			c.RerankScore = score
			// Combine with original score (weighted)
			c.Score = c.Score*0.3 + score*0.7
		}
	}

	// Re-sort by final score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates, nil
}

// addContext adds surrounding lines to a search result.
func (e *Engine) addContext(result *types.SearchResult, contextLines int) {
	if contextLines == 0 {
		contextLines = 5
	}

	// Read the file
	file, err := os.Open(result.Chunk.FilePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	startLine := result.Chunk.StartLine
	endLine := result.Chunk.EndLine

	// Get context before
	contextStart := startLine - contextLines - 1
	if contextStart < 0 {
		contextStart = 0
	}
	if contextStart < startLine-1 {
		result.ContextBefore = strings.Join(allLines[contextStart:startLine-1], "\n")
	}

	// Get context after
	contextEnd := endLine + contextLines
	if contextEnd > len(allLines) {
		contextEnd = len(allLines)
	}
	if endLine < contextEnd {
		result.ContextAfter = strings.Join(allLines[endLine:contextEnd], "\n")
	}
}

// SearchSymbols searches for symbols by name and kind.
func (e *Engine) SearchSymbols(query string, kind types.SymbolKind, limit int) ([]*types.Symbol, error) {
	return e.store.FindSymbols(query, kind, limit)
}

// SearchSymbolsAdvanced searches for symbols with additional filtering options.
func (e *Engine) SearchSymbolsAdvanced(query string, kind types.SymbolKind, minLines int, sortBy string, limit int) ([]*types.Symbol, error) {
	return e.store.FindSymbolsAdvanced(query, kind, minLines, sortBy, limit)
}

// FindLongFunctions returns functions sorted by line count (longest first).
func (e *Engine) FindLongFunctions(minLines int, limit int) ([]*types.Symbol, error) {
	return e.store.FindLongFunctions(minLines, limit)
}

// ComplexityCandidate represents a function with high complexity metrics.
type ComplexityCandidate struct {
	Name                 string `json:"name"`
	FilePath             string `json:"file_path"`
	StartLine            int    `json:"start_line"`
	EndLine              int    `json:"end_line"`
	LineCount            int    `json:"line_count"`
	CyclomaticComplexity int    `json:"cyclomatic_complexity"`
	CognitiveComplexity  int    `json:"cognitive_complexity"`
	MaxNesting           int    `json:"max_nesting"`
}

// FindHighComplexityFunctions finds functions with complexity above thresholds.
// This is a simplified implementation that uses line count as a proxy for complexity.
// For accurate complexity metrics, use the analysis package on specific files.
func (e *Engine) FindHighComplexityFunctions(maxComplexity, maxNesting, limit int) ([]*ComplexityCandidate, error) {
	// Get long functions as candidates (longer functions tend to have higher complexity)
	// Minimum 30 lines to be considered for complexity analysis
	symbols, err := e.store.FindLongFunctions(30, limit*3)
	if err != nil {
		return nil, err
	}

	var candidates []*ComplexityCandidate
	for _, sym := range symbols {
		// Estimate complexity based on line count as a simple heuristic
		// Actual cyclomatic complexity requires parsing the code
		estimatedComplexity := sym.LineCount / 5 // rough estimate: 1 complexity point per 5 lines
		estimatedNesting := 2                     // default estimate

		// Longer functions typically have higher nesting
		if sym.LineCount > 100 {
			estimatedNesting = 4
		} else if sym.LineCount > 50 {
			estimatedNesting = 3
		}

		// Only include if exceeds thresholds
		if estimatedComplexity > maxComplexity || estimatedNesting > maxNesting {
			candidates = append(candidates, &ComplexityCandidate{
				Name:                 sym.Name,
				FilePath:             sym.FilePath,
				StartLine:            sym.StartLine,
				EndLine:              sym.EndLine,
				LineCount:            sym.LineCount,
				CyclomaticComplexity: estimatedComplexity,
				CognitiveComplexity:  estimatedComplexity, // simplified
				MaxNesting:           estimatedNesting,
			})
		}

		if len(candidates) >= limit {
			break
		}
	}

	return candidates, nil
}

// GetCallers returns callers of a symbol.
func (e *Engine) GetCallers(symbolID string, limit int) ([]*types.Reference, error) {
	return e.store.GetCallers(symbolID, limit)
}

// GetCallees returns callees of a symbol.
func (e *Engine) GetCallees(symbolID string, limit int) ([]*types.Reference, error) {
	return e.store.GetCallees(symbolID, limit)
}

// GetSymbol retrieves a symbol by ID.
func (e *Engine) GetSymbol(id string) (*types.Symbol, error) {
	return e.store.GetSymbol(id)
}

// EntryPoint represents a detected entry point in the codebase.
type EntryPoint struct {
	Symbol     *types.Symbol `json:"symbol"`
	Type       string        `json:"type"`       // main, handler, test, init, cli
	Confidence float32       `json:"confidence"` // 0-1
}

// GetEntryPoints finds entry points in the codebase (main functions, handlers, tests, etc).
func (e *Engine) GetEntryPoints(limit int) ([]*EntryPoint, error) {
	var entryPoints []*EntryPoint

	// Find main functions (highest confidence)
	mainSymbols, err := e.store.FindSymbols("main", types.SymbolKindFunction, 100)
	if err != nil {
		return nil, err
	}
	for _, sym := range mainSymbols {
		if sym.Name == "main" {
			entryPoints = append(entryPoints, &EntryPoint{
				Symbol:     sym,
				Type:       "main",
				Confidence: 1.0,
			})
		}
	}

	// Find HTTP handlers (look for common patterns)
	handlerPatterns := []string{"Handle", "ServeHTTP", "Handler"}
	for _, pattern := range handlerPatterns {
		symbols, err := e.store.FindSymbols(pattern, types.SymbolKindFunction, 100)
		if err != nil {
			continue
		}
		for _, sym := range symbols {
			// Check if it looks like a handler
			if strings.Contains(sym.Name, pattern) || strings.HasSuffix(sym.Name, pattern) {
				entryPoints = append(entryPoints, &EntryPoint{
					Symbol:     sym,
					Type:       "handler",
					Confidence: 0.8,
				})
			}
		}
	}

	// Find test functions
	testSymbols, err := e.store.FindSymbols("Test", types.SymbolKindFunction, 100)
	if err == nil {
		for _, sym := range testSymbols {
			if strings.HasPrefix(sym.Name, "Test") {
				entryPoints = append(entryPoints, &EntryPoint{
					Symbol:     sym,
					Type:       "test",
					Confidence: 1.0,
				})
			}
		}
	}

	// Find init functions
	initSymbols, err := e.store.FindSymbols("init", types.SymbolKindFunction, 100)
	if err == nil {
		for _, sym := range initSymbols {
			if sym.Name == "init" {
				entryPoints = append(entryPoints, &EntryPoint{
					Symbol:     sym,
					Type:       "init",
					Confidence: 1.0,
				})
			}
		}
	}

	// Find CLI entry points (Execute, Run, Cmd)
	cliPatterns := []string{"Execute", "Run", "Cmd"}
	for _, pattern := range cliPatterns {
		symbols, err := e.store.FindSymbols(pattern, types.SymbolKindFunction, 50)
		if err != nil {
			continue
		}
		for _, sym := range symbols {
			if strings.Contains(sym.Name, pattern) {
				entryPoints = append(entryPoints, &EntryPoint{
					Symbol:     sym,
					Type:       "cli",
					Confidence: 0.6,
				})
			}
		}
	}

	// Sort by confidence and limit
	sort.Slice(entryPoints, func(i, j int) bool {
		return entryPoints[i].Confidence > entryPoints[j].Confidence
	})

	if limit > 0 && len(entryPoints) > limit {
		entryPoints = entryPoints[:limit]
	}

	return entryPoints, nil
}

// ImportEdge represents an import relationship between files/packages.
type ImportEdge struct {
	From       string `json:"from"`        // Source file path
	To         string `json:"to"`          // Imported package/module
	Line       int    `json:"line"`        // Line where import occurs
	IsExternal bool   `json:"is_external"` // Is it an external package?
}

// ImportGraph represents the import graph of the codebase.
type ImportGraph struct {
	Edges      []*ImportEdge     `json:"edges"`
	FileCount  int               `json:"file_count"`   // Number of files in the graph
	Imports    map[string]int    `json:"imports"`      // Package -> import count
	ImportedBy map[string][]string `json:"imported_by"` // Package -> files that import it
}

// GetImportGraph builds the import graph from import references.
func (e *Engine) GetImportGraph(limit int) (*ImportGraph, error) {
	// Get all import references
	refs, err := e.store.FindReferencesByKind(types.RefKindImport, limit)
	if err != nil {
		return nil, err
	}

	graph := &ImportGraph{
		Edges:      make([]*ImportEdge, 0),
		Imports:    make(map[string]int),
		ImportedBy: make(map[string][]string),
	}

	filesSet := make(map[string]bool)

	for _, ref := range refs {
		edge := &ImportEdge{
			From:       ref.FromSymbol, // Using FromSymbol as file path or package
			To:         ref.ToSymbol,   // Imported package/module
			Line:       ref.Line,
			IsExternal: ref.IsExternal,
		}
		graph.Edges = append(graph.Edges, edge)

		// Track file
		filesSet[ref.FilePath] = true

		// Count imports
		graph.Imports[ref.ToSymbol]++

		// Track what imports what
		if _, ok := graph.ImportedBy[ref.ToSymbol]; !ok {
			graph.ImportedBy[ref.ToSymbol] = []string{}
		}
		// Add file if not already in list
		found := false
		for _, f := range graph.ImportedBy[ref.ToSymbol] {
			if f == ref.FilePath {
				found = true
				break
			}
		}
		if !found {
			graph.ImportedBy[ref.ToSymbol] = append(graph.ImportedBy[ref.ToSymbol], ref.FilePath)
		}
	}

	graph.FileCount = len(filesSet)

	return graph, nil
}
