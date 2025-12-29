package analysis

import (
	"strings"

	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// DeadCodeResult represents a potentially unused code symbol.
type DeadCodeResult struct {
	Symbol     *types.Symbol `json:"symbol"`
	Reason     string        `json:"reason"`
	Confidence float32       `json:"confidence"` // 0-1, higher = more likely dead
	CallerCount int          `json:"caller_count"`
}

// DeadCodeAnalyzer detects potentially unused code.
type DeadCodeAnalyzer struct {
	store provider.VectorStore
}

// NewDeadCodeAnalyzer creates a new dead code analyzer.
func NewDeadCodeAnalyzer(store provider.VectorStore) *DeadCodeAnalyzer {
	return &DeadCodeAnalyzer{store: store}
}

// FindDeadCode finds potentially unused symbols.
func (d *DeadCodeAnalyzer) FindDeadCode(limit int) ([]*DeadCodeResult, error) {
	// Get all symbols
	symbols, err := d.store.FindSymbols("", "", 10000)
	if err != nil {
		return nil, err
	}

	var results []*DeadCodeResult

	for _, sym := range symbols {
		result := d.analyzeSymbol(sym)
		if result != nil {
			results = append(results, result)
		}

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// FindDeadFunctions specifically finds unused functions.
func (d *DeadCodeAnalyzer) FindDeadFunctions(limit int) ([]*DeadCodeResult, error) {
	// Get function symbols only
	symbols, err := d.store.FindSymbols("", types.SymbolKindFunction, 10000)
	if err != nil {
		return nil, err
	}

	// Also get methods
	methods, err := d.store.FindSymbols("", types.SymbolKindMethod, 10000)
	if err == nil {
		symbols = append(symbols, methods...)
	}

	var results []*DeadCodeResult

	for _, sym := range symbols {
		result := d.analyzeFunction(sym)
		if result != nil {
			results = append(results, result)
		}

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// FindUnusedTypes finds types that are never used.
func (d *DeadCodeAnalyzer) FindUnusedTypes(limit int) ([]*DeadCodeResult, error) {
	// Get type symbols
	symbols, err := d.store.FindSymbols("", types.SymbolKindType, 10000)
	if err != nil {
		return nil, err
	}

	var results []*DeadCodeResult

	for _, sym := range symbols {
		result := d.analyzeType(sym)
		if result != nil {
			results = append(results, result)
		}

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// analyzeSymbol checks if a symbol appears to be unused.
func (d *DeadCodeAnalyzer) analyzeSymbol(sym *types.Symbol) *DeadCodeResult {
	// Skip if it's an entry point or special function
	if d.isEntryPoint(sym) {
		return nil
	}

	// Skip if it's exported/public and might be used externally
	if sym.Visibility == "public" {
		return nil
	}

	// Get callers/references
	callers, err := d.store.GetCallers(sym.ID, 100)
	if err != nil {
		return nil
	}

	// No callers = potentially dead
	if len(callers) == 0 {
		confidence := d.calculateConfidence(sym, callers)
		if confidence > 0.5 {
			return &DeadCodeResult{
				Symbol:      sym,
				Reason:      "no references found",
				Confidence:  confidence,
				CallerCount: 0,
			}
		}
	}

	return nil
}

// analyzeFunction checks if a function appears to be unused.
func (d *DeadCodeAnalyzer) analyzeFunction(sym *types.Symbol) *DeadCodeResult {
	// Skip entry points
	if d.isEntryPoint(sym) {
		return nil
	}

	// Skip test functions
	if d.isTestFunction(sym) {
		return nil
	}

	// Skip interface implementations (harder to detect usage)
	if d.mightBeInterfaceImpl(sym) {
		return nil
	}

	// Get callers
	callers, err := d.store.GetCallers(sym.ID, 100)
	if err != nil {
		return nil
	}

	// Filter out self-references
	externalCallers := 0
	for _, ref := range callers {
		if ref.FromSymbol != sym.Name && ref.FromSymbol != sym.ID {
			externalCallers++
		}
	}

	if externalCallers == 0 {
		confidence := d.calculateFunctionConfidence(sym)

		// Only report if confidence is high enough
		if confidence >= 0.6 {
			reason := "no callers found"
			if len(callers) > 0 {
				reason = "only self-referential calls"
			}

			return &DeadCodeResult{
				Symbol:      sym,
				Reason:      reason,
				Confidence:  confidence,
				CallerCount: externalCallers,
			}
		}
	}

	return nil
}

// analyzeType checks if a type appears to be unused.
func (d *DeadCodeAnalyzer) analyzeType(sym *types.Symbol) *DeadCodeResult {
	// Skip exported types
	if sym.Visibility == "public" {
		return nil
	}

	// Get references to this type
	refs, err := d.store.GetCallers(sym.ID, 100)
	if err != nil {
		return nil
	}

	if len(refs) == 0 {
		return &DeadCodeResult{
			Symbol:      sym,
			Reason:      "type is never used",
			Confidence:  0.7,
			CallerCount: 0,
		}
	}

	return nil
}

// isEntryPoint checks if a symbol is an entry point.
func (d *DeadCodeAnalyzer) isEntryPoint(sym *types.Symbol) bool {
	name := strings.ToLower(sym.Name)

	// Main functions
	if name == "main" {
		return true
	}

	// Init functions
	if name == "init" {
		return true
	}

	// Common handler patterns
	if strings.HasPrefix(name, "handle") || strings.HasSuffix(name, "handler") {
		return true
	}

	// Exported API endpoints (Go convention)
	if len(sym.Name) > 0 && sym.Name[0] >= 'A' && sym.Name[0] <= 'Z' {
		// This is exported, might be used externally
		return false // Don't skip, but lower confidence
	}

	return false
}

// isTestFunction checks if a function is a test.
func (d *DeadCodeAnalyzer) isTestFunction(sym *types.Symbol) bool {
	name := sym.Name

	// Go tests
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example") {
		return true
	}

	// Python tests
	if strings.HasPrefix(name, "test_") {
		return true
	}

	// Check file path
	if strings.Contains(sym.FilePath, "_test.go") || strings.Contains(sym.FilePath, "test_") {
		return true
	}

	return false
}

// mightBeInterfaceImpl checks if a function might be an interface implementation.
func (d *DeadCodeAnalyzer) mightBeInterfaceImpl(sym *types.Symbol) bool {
	// Methods are often interface implementations
	if sym.Kind == types.SymbolKindMethod {
		return true
	}

	// Common interface method names
	commonInterfaceMethods := []string{
		"String", "Error", "Close", "Read", "Write",
		"ServeHTTP", "MarshalJSON", "UnmarshalJSON",
		"Scan", "Value", "Next", "Err",
	}

	for _, name := range commonInterfaceMethods {
		if sym.Name == name {
			return true
		}
	}

	return false
}

// calculateConfidence calculates how confident we are that code is dead.
func (d *DeadCodeAnalyzer) calculateConfidence(sym *types.Symbol, callers []*types.Reference) float32 {
	confidence := float32(0.5)

	// Private symbols are more likely to be truly dead
	if sym.Visibility == "private" {
		confidence += 0.2
	}

	// Functions with no callers are more likely dead
	if len(callers) == 0 {
		confidence += 0.2
	}

	// Small functions might be utility functions
	// (we don't have size info here, but could add it)

	return min(confidence, 1.0)
}

// calculateFunctionConfidence calculates confidence for function dead code.
func (d *DeadCodeAnalyzer) calculateFunctionConfidence(sym *types.Symbol) float32 {
	confidence := float32(0.6)

	// Private/unexported functions
	if sym.Visibility == "private" {
		confidence += 0.2
	}

	// Functions with underscore prefix (often internal)
	if strings.HasPrefix(sym.Name, "_") {
		confidence += 0.1
	}

	// Helper functions
	if strings.Contains(strings.ToLower(sym.Name), "helper") {
		confidence -= 0.1 // might be used via reflection
	}

	return min(max(confidence, 0), 1.0)
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
