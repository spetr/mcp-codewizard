package analysis

import (
	"sort"
	"strings"

	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// DeadCodeResult represents a potentially unused code symbol.
type DeadCodeResult struct {
	Symbol      *types.Symbol `json:"symbol"`
	Reason      string        `json:"reason"`
	Confidence  float32       `json:"confidence"` // 0-1, higher = more likely dead
	CallerCount int           `json:"caller_count"`
	DeadCallers int           `json:"dead_callers,omitempty"` // Number of callers that are also dead
}

// LanguageConfig contains language-specific settings for dead code detection.
type LanguageConfig struct {
	// EntryPointPatterns are function name patterns that are always considered entry points
	EntryPointPatterns []string

	// ExportedCheck determines if a symbol is exported/public in this language
	ExportedCheck func(sym *types.Symbol) bool

	// TestPatterns are patterns for test functions
	TestPatterns []string

	// MethodsAreInterfaceCandidates indicates if all methods might implement interfaces
	MethodsAreInterfaceCandidates bool
}

// DefaultLanguageConfigs provides language-specific configurations.
var DefaultLanguageConfigs = map[string]*LanguageConfig{
	"go": {
		EntryPointPatterns: []string{"main", "init", "ServeHTTP"},
		ExportedCheck: func(sym *types.Symbol) bool {
			if len(sym.Name) == 0 {
				return false
			}
			r := rune(sym.Name[0])
			return r >= 'A' && r <= 'Z'
		},
		TestPatterns:                  []string{"Test", "Benchmark", "Example", "Fuzz"},
		MethodsAreInterfaceCandidates: true,
	},
	"python": {
		EntryPointPatterns: []string{"main", "__main__", "__init__"},
		ExportedCheck: func(sym *types.Symbol) bool {
			// Python: not starting with underscore = public
			return !strings.HasPrefix(sym.Name, "_")
		},
		TestPatterns:                  []string{"test_", "Test"},
		MethodsAreInterfaceCandidates: false, // Python uses duck typing
	},
	"javascript": {
		EntryPointPatterns: []string{"main", "default"},
		ExportedCheck: func(sym *types.Symbol) bool {
			// JS: check visibility field or assume all named exports are public
			return sym.Visibility == "public" || sym.Visibility == ""
		},
		TestPatterns:                  []string{"test", "it", "describe", "Test"},
		MethodsAreInterfaceCandidates: false,
	},
	"typescript": {
		EntryPointPatterns: []string{"main", "default"},
		ExportedCheck: func(sym *types.Symbol) bool {
			return sym.Visibility == "public" || sym.Visibility == ""
		},
		TestPatterns:                  []string{"test", "it", "describe", "Test"},
		MethodsAreInterfaceCandidates: true, // TS has interfaces
	},
	"java": {
		EntryPointPatterns: []string{"main"},
		ExportedCheck: func(sym *types.Symbol) bool {
			return sym.Visibility == "public"
		},
		TestPatterns:                  []string{"test", "Test"},
		MethodsAreInterfaceCandidates: true,
	},
	"rust": {
		EntryPointPatterns: []string{"main"},
		ExportedCheck: func(sym *types.Symbol) bool {
			return sym.Visibility == "public" // pub keyword
		},
		TestPatterns:                  []string{"test_", "test"},
		MethodsAreInterfaceCandidates: true, // Rust has traits
	},
	"csharp": {
		EntryPointPatterns: []string{"Main"},
		ExportedCheck: func(sym *types.Symbol) bool {
			return sym.Visibility == "public"
		},
		TestPatterns:                  []string{"Test"},
		MethodsAreInterfaceCandidates: true,
	},
}

// DeadCodeAnalyzer detects potentially unused code using graph-based reachability analysis.
type DeadCodeAnalyzer struct {
	store provider.VectorStore

	// In-memory data structures
	symbols      map[string]*types.Symbol // ID → Symbol
	symbolsByName map[string][]*types.Symbol // name → Symbols (for resolution)

	// Call graph
	callees map[string]map[string]bool // caller → set of callees
	callers map[string]map[string]bool // callee → set of callers

	// Analysis results
	reachable map[string]bool

	// Language configs
	langConfigs map[string]*LanguageConfig
}

// NewDeadCodeAnalyzer creates a new dead code analyzer.
func NewDeadCodeAnalyzer(store provider.VectorStore) *DeadCodeAnalyzer {
	return &DeadCodeAnalyzer{
		store:         store,
		symbols:       make(map[string]*types.Symbol),
		symbolsByName: make(map[string][]*types.Symbol),
		callees:       make(map[string]map[string]bool),
		callers:       make(map[string]map[string]bool),
		reachable:     make(map[string]bool),
		langConfigs:   DefaultLanguageConfigs,
	}
}

// BuildGraph constructs the call graph from stored symbols and references.
func (d *DeadCodeAnalyzer) BuildGraph() error {
	// 1. Load all symbols
	symbols, err := d.store.FindSymbols("", "", 100000)
	if err != nil {
		return err
	}

	for _, sym := range symbols {
		d.symbols[sym.ID] = sym
		d.symbolsByName[sym.Name] = append(d.symbolsByName[sym.Name], sym)
	}

	// 2. Load all internal references
	refs, err := d.store.GetAllReferences(1000000)
	if err != nil {
		return err
	}

	// 3. Build graph from references
	for _, ref := range refs {
		fromID := d.resolveSymbol(ref.FromSymbol, ref.FilePath)
		toID := d.resolveSymbol(ref.ToSymbol, ref.FilePath)

		if fromID != "" && toID != "" {
			d.addEdge(fromID, toID)
		}
	}

	return nil
}

// resolveSymbol resolves a symbol name to its ID.
// Uses file path context to prefer symbols from the same package/directory.
func (d *DeadCodeAnalyzer) resolveSymbol(name string, contextFile string) string {
	// Direct ID match
	if _, ok := d.symbols[name]; ok {
		return name
	}

	// Handle qualified names (pkg.Symbol)
	shortName := name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		shortName = name[idx+1:]
	}

	candidates := d.symbolsByName[shortName]
	if len(candidates) == 0 {
		return "" // External symbol
	}

	if len(candidates) == 1 {
		return candidates[0].ID
	}

	// Multiple candidates - prefer same directory/package
	contextDir := ""
	if idx := strings.LastIndex(contextFile, "/"); idx >= 0 {
		contextDir = contextFile[:idx]
	}

	for _, sym := range candidates {
		symDir := ""
		if idx := strings.LastIndex(sym.FilePath, "/"); idx >= 0 {
			symDir = sym.FilePath[:idx]
		}
		if symDir == contextDir {
			return sym.ID
		}
	}

	// Fallback to first candidate
	return candidates[0].ID
}

// addEdge adds a directed edge from caller to callee.
func (d *DeadCodeAnalyzer) addEdge(from, to string) {
	if from == "" || to == "" || from == to {
		return
	}

	if d.callees[from] == nil {
		d.callees[from] = make(map[string]bool)
	}
	d.callees[from][to] = true

	if d.callers[to] == nil {
		d.callers[to] = make(map[string]bool)
	}
	d.callers[to][from] = true
}

// ComputeReachable computes all symbols reachable from entry points.
func (d *DeadCodeAnalyzer) ComputeReachable() {
	d.reachable = make(map[string]bool)

	// Find all entry points
	var worklist []string
	for id, sym := range d.symbols {
		if d.isEntryPoint(sym) {
			worklist = append(worklist, id)
		}
	}

	// BFS traversal
	for len(worklist) > 0 {
		id := worklist[0]
		worklist = worklist[1:]

		if d.reachable[id] {
			continue
		}
		d.reachable[id] = true

		// Add all callees
		for calleeID := range d.callees[id] {
			if !d.reachable[calleeID] {
				worklist = append(worklist, calleeID)
			}
		}
	}
}

// FindDeadCode finds all unreachable symbols.
func (d *DeadCodeAnalyzer) FindDeadCode(limit int) ([]*DeadCodeResult, error) {
	// Build graph if not already built
	if len(d.symbols) == 0 {
		if err := d.BuildGraph(); err != nil {
			return nil, err
		}
	}

	// Compute reachability
	d.ComputeReachable()

	var results []*DeadCodeResult
	for id, sym := range d.symbols {
		if d.reachable[id] {
			continue
		}

		// Skip test functions
		if d.isTestFunction(sym) {
			continue
		}

		// Calculate confidence
		confidence := d.calculateConfidence(sym)
		if confidence < 0.5 {
			continue
		}

		// Determine reason
		reason, deadCallers := d.determineReason(sym)

		results = append(results, &DeadCodeResult{
			Symbol:      sym,
			Reason:      reason,
			Confidence:  confidence,
			CallerCount: len(d.callers[id]),
			DeadCallers: deadCallers,
		})

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	// Sort by confidence descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// FindDeadFunctions finds only unreachable functions and methods.
func (d *DeadCodeAnalyzer) FindDeadFunctions(limit int) ([]*DeadCodeResult, error) {
	allDead, err := d.FindDeadCode(0)
	if err != nil {
		return nil, err
	}

	var results []*DeadCodeResult
	for _, r := range allDead {
		if r.Symbol.Kind == types.SymbolKindFunction || r.Symbol.Kind == types.SymbolKindMethod {
			results = append(results, r)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// FindUnusedTypes finds only unreachable types.
func (d *DeadCodeAnalyzer) FindUnusedTypes(limit int) ([]*DeadCodeResult, error) {
	allDead, err := d.FindDeadCode(0)
	if err != nil {
		return nil, err
	}

	var results []*DeadCodeResult
	for _, r := range allDead {
		if r.Symbol.Kind == types.SymbolKindType || r.Symbol.Kind == types.SymbolKindInterface {
			results = append(results, r)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// determineReason returns the reason why a symbol is considered dead and count of dead callers.
func (d *DeadCodeAnalyzer) determineReason(sym *types.Symbol) (string, int) {
	callerSet := d.callers[sym.ID]
	if len(callerSet) == 0 {
		if d.isExported(sym) {
			return "no callers found (exported - verify if public API)", 0
		}
		return "no callers found", 0
	}

	// Count dead callers
	deadCallers := 0
	for callerID := range callerSet {
		if !d.reachable[callerID] {
			deadCallers++
		}
	}

	if deadCallers == len(callerSet) {
		return "only called by dead code", deadCallers
	}

	// This shouldn't happen if reachability is correct
	return "unreachable", deadCallers
}

// calculateConfidence returns confidence score that the symbol is truly dead.
func (d *DeadCodeAnalyzer) calculateConfidence(sym *types.Symbol) float32 {
	confidence := float32(0.85)

	// Exported symbols might be used externally - lower confidence
	if d.isExported(sym) {
		confidence -= 0.3
	}

	// Constructor patterns (New*, Create*, Make*) are often called dynamically
	name := sym.Name
	if strings.HasPrefix(name, "New") || strings.HasPrefix(name, "Create") || strings.HasPrefix(name, "Make") {
		confidence -= 0.15
	}

	// Methods might implement interfaces
	if sym.Kind == types.SymbolKindMethod {
		cfg := d.getLanguageConfig(sym)
		if cfg != nil && cfg.MethodsAreInterfaceCandidates {
			confidence -= 0.2
		}
	}

	// Has callers but they're all dead - higher confidence (we see the chain)
	if callers := d.callers[sym.ID]; len(callers) > 0 {
		allDead := true
		for callerID := range callers {
			if d.reachable[callerID] {
				allDead = false
				break
			}
		}
		if allDead {
			confidence += 0.1 // More confident because we see the chain
		}
	}

	// Handler patterns might be registered dynamically
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "handler") || strings.Contains(lowerName, "middleware") {
		confidence -= 0.15
	}

	// Callback patterns
	if strings.HasPrefix(name, "on") || strings.HasPrefix(name, "On") {
		confidence -= 0.1
	}

	return max(min(confidence, 1.0), 0.0)
}

// isEntryPoint checks if a symbol is an entry point (root of reachability).
func (d *DeadCodeAnalyzer) isEntryPoint(sym *types.Symbol) bool {
	name := sym.Name
	lowerName := strings.ToLower(name)

	// Get language config
	cfg := d.getLanguageConfig(sym)

	// Check language-specific entry point patterns
	if cfg != nil {
		for _, pattern := range cfg.EntryPointPatterns {
			if name == pattern || lowerName == strings.ToLower(pattern) {
				return true
			}
		}
	}

	// Universal patterns
	if name == "main" || name == "init" || name == "__init__" || name == "__main__" {
		return true
	}

	// Test functions are entry points (so they reach tested code)
	if d.isTestFunction(sym) {
		return true
	}

	// Handler patterns
	if strings.HasPrefix(lowerName, "handle") || strings.HasSuffix(lowerName, "handler") {
		return true
	}

	// HTTP handlers
	if name == "ServeHTTP" || name == "ServeHTTPS" {
		return true
	}

	// Exported symbols in main packages might be entry points
	// But we don't treat all exported as entry points - that would defeat the purpose

	return false
}

// isTestFunction checks if a function is a test function.
func (d *DeadCodeAnalyzer) isTestFunction(sym *types.Symbol) bool {
	name := sym.Name
	cfg := d.getLanguageConfig(sym)

	// Check language-specific test patterns
	if cfg != nil {
		for _, pattern := range cfg.TestPatterns {
			if strings.HasPrefix(name, pattern) {
				return true
			}
		}
	}

	// Universal test patterns
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "test_") ||
		strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example") {
		return true
	}

	// Check file path for test files
	path := strings.ToLower(sym.FilePath)
	if strings.Contains(path, "_test.") || strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/") || strings.Contains(path, "/testing/") ||
		strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(path, ".test.ts") || strings.HasSuffix(path, "_test.py") {
		return true
	}

	return false
}

// isExported checks if a symbol is exported/public.
func (d *DeadCodeAnalyzer) isExported(sym *types.Symbol) bool {
	cfg := d.getLanguageConfig(sym)
	if cfg != nil && cfg.ExportedCheck != nil {
		return cfg.ExportedCheck(sym)
	}

	// Fallback: check visibility field
	if sym.Visibility == "public" {
		return true
	}

	// Go convention: starts with uppercase
	if len(sym.Name) > 0 {
		r := rune(sym.Name[0])
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}

	return false
}

// getLanguageConfig returns the language configuration for a symbol.
func (d *DeadCodeAnalyzer) getLanguageConfig(sym *types.Symbol) *LanguageConfig {
	// Detect language from file path
	lang := detectLanguageFromPath(sym.FilePath)
	if cfg, ok := d.langConfigs[lang]; ok {
		return cfg
	}

	// Check common aliases
	switch lang {
	case "jsx", "tsx":
		return d.langConfigs["javascript"]
	case "ts":
		return d.langConfigs["typescript"]
	case "cs":
		return d.langConfigs["csharp"]
	}

	return nil
}

// GetStats returns statistics about the call graph.
func (d *DeadCodeAnalyzer) GetStats() map[string]int {
	if len(d.symbols) == 0 {
		_ = d.BuildGraph()
		d.ComputeReachable()
	}

	deadCount := 0
	for id := range d.symbols {
		if !d.reachable[id] {
			deadCount++
		}
	}

	edgeCount := 0
	for _, callees := range d.callees {
		edgeCount += len(callees)
	}

	return map[string]int{
		"total_symbols":     len(d.symbols),
		"reachable_symbols": len(d.reachable),
		"dead_symbols":      deadCount,
		"total_edges":       edgeCount,
		"entry_points":      d.countEntryPoints(),
	}
}

func (d *DeadCodeAnalyzer) countEntryPoints() int {
	count := 0
	for _, sym := range d.symbols {
		if d.isEntryPoint(sym) {
			count++
		}
	}
	return count
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
