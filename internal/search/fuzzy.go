// Package search implements hybrid search with reranking.
package search

import (
	"sort"
	"strings"
	"unicode"

	"github.com/spetr/mcp-codewizard/pkg/types"
)

// FuzzyMatch represents a fuzzy search match.
type FuzzyMatch struct {
	Symbol     *types.Symbol `json:"symbol"`
	Score      float32       `json:"score"`       // 0-1, higher is better
	MatchType  string        `json:"match_type"`  // exact, prefix, contains, fuzzy
	Highlights []int         `json:"highlights"`  // Indices of matched characters
}

// FuzzySearchSymbols performs fuzzy search on symbol names.
func (e *Engine) FuzzySearchSymbols(query string, kind types.SymbolKind, limit int) ([]*FuzzyMatch, error) {
	if limit == 0 {
		limit = 20
	}

	// Get all symbols (or filtered by kind)
	symbols, err := e.store.FindSymbolsAdvanced("", kind, 0, "", 1000)
	if err != nil {
		return nil, err
	}

	// Normalize query
	queryLower := strings.ToLower(query)
	queryTokens := tokenize(query)

	var matches []*FuzzyMatch

	for _, sym := range symbols {
		nameLower := strings.ToLower(sym.Name)

		var score float32
		var matchType string
		var highlights []int

		// 1. Exact match (highest priority)
		if nameLower == queryLower {
			score = 1.0
			matchType = "exact"
			highlights = makeRange(0, len(sym.Name))
		} else if strings.HasPrefix(nameLower, queryLower) {
			// 2. Prefix match
			score = 0.9
			matchType = "prefix"
			highlights = makeRange(0, len(query))
		} else if strings.Contains(nameLower, queryLower) {
			// 3. Contains match
			idx := strings.Index(nameLower, queryLower)
			score = 0.7
			matchType = "contains"
			highlights = makeRange(idx, idx+len(query))
		} else {
			// 4. Fuzzy match - try multiple strategies
			fuzzyScore, fuzzyHighlights := fuzzyMatch(queryLower, nameLower)
			if fuzzyScore > 0.3 {
				score = fuzzyScore * 0.6 // Cap fuzzy matches at 0.6
				matchType = "fuzzy"
				highlights = fuzzyHighlights
			}

			// 5. CamelCase/snake_case token matching
			nameTokens := tokenize(sym.Name)
			tokenScore := tokenMatch(queryTokens, nameTokens)
			if tokenScore > score {
				score = tokenScore * 0.8 // Token matches capped at 0.8
				matchType = "token"
				highlights = findTokenHighlights(query, sym.Name)
			}
		}

		if score > 0.3 { // Threshold
			matches = append(matches, &FuzzyMatch{
				Symbol:     sym,
				Score:      score,
				MatchType:  matchType,
				Highlights: highlights,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Apply limit
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// fuzzyMatch calculates fuzzy similarity using longest common subsequence.
func fuzzyMatch(query, target string) (float32, []int) {
	if len(query) == 0 || len(target) == 0 {
		return 0, nil
	}

	// Find longest common subsequence
	lcs, indices := longestCommonSubsequence(query, target)
	if len(lcs) == 0 {
		return 0, nil
	}

	// Score based on:
	// 1. Ratio of matched characters to query length
	// 2. Ratio of matched characters to target length
	// 3. Bonus for consecutive matches
	matchRatio := float32(len(lcs)) / float32(len(query))
	targetRatio := float32(len(lcs)) / float32(len(target))

	// Calculate consecutive bonus
	consecutiveBonus := float32(0)
	for i := 1; i < len(indices); i++ {
		if indices[i] == indices[i-1]+1 {
			consecutiveBonus += 0.05
		}
	}

	score := (matchRatio*0.6 + targetRatio*0.3 + consecutiveBonus*0.1)
	if score > 1.0 {
		score = 1.0
	}

	return score, indices
}

// longestCommonSubsequence finds LCS and returns matched indices in target.
func longestCommonSubsequence(s1, s2 string) (string, []int) {
	m, n := len(s1), len(s2)
	if m == 0 || n == 0 {
		return "", nil
	}

	// DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS and indices
	lcs := make([]byte, 0, dp[m][n])
	indices := make([]int, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if s1[i-1] == s2[j-1] {
			lcs = append([]byte{s1[i-1]}, lcs...)
			indices = append([]int{j - 1}, indices...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return string(lcs), indices
}

// tokenize splits a name into tokens (handles camelCase and snake_case).
func tokenize(name string) []string {
	var tokens []string
	var current strings.Builder

	for i, r := range name {
		if r == '_' || r == '-' || r == '.' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.ToLower(current.String()))
				current.Reset()
			}
			continue
		}

		// CamelCase boundary
		if unicode.IsUpper(r) && i > 0 {
			if current.Len() > 0 {
				tokens = append(tokens, strings.ToLower(current.String()))
				current.Reset()
			}
		}

		current.WriteRune(unicode.ToLower(r))
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// tokenMatch calculates how well query tokens match target tokens.
func tokenMatch(queryTokens, targetTokens []string) float32 {
	if len(queryTokens) == 0 || len(targetTokens) == 0 {
		return 0
	}

	matched := 0
	for _, qt := range queryTokens {
		for _, tt := range targetTokens {
			if strings.HasPrefix(tt, qt) {
				matched++
				break
			}
		}
	}

	return float32(matched) / float32(len(queryTokens))
}

// findTokenHighlights finds character indices that match tokens.
func findTokenHighlights(query, target string) []int {
	var highlights []int
	queryLower := strings.ToLower(query)
	targetLower := strings.ToLower(target)

	qi := 0
	for ti := 0; ti < len(target) && qi < len(query); ti++ {
		if targetLower[ti] == queryLower[qi] {
			highlights = append(highlights, ti)
			qi++
		}
	}

	return highlights
}

// makeRange creates a slice of integers from start to end (exclusive).
func makeRange(start, end int) []int {
	if end <= start {
		return nil
	}
	result := make([]int, end-start)
	for i := range result {
		result[i] = start + i
	}
	return result
}

// FuzzySearchFiles performs fuzzy search on file paths.
func (e *Engine) FuzzySearchFiles(query string, limit int) ([]string, error) {
	if limit == 0 {
		limit = 20
	}

	// Get all indexed file paths
	fileHashes, err := e.store.GetAllFileHashes()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	type fileMatch struct {
		path  string
		score float32
	}

	var matches []fileMatch

	for path := range fileHashes {
		pathLower := strings.ToLower(path)
		fileName := pathLower
		if idx := strings.LastIndex(pathLower, "/"); idx >= 0 {
			fileName = pathLower[idx+1:]
		}

		var score float32

		// Check filename first (higher priority)
		if strings.Contains(fileName, queryLower) {
			if fileName == queryLower {
				score = 1.0
			} else if strings.HasPrefix(fileName, queryLower) {
				score = 0.9
			} else {
				score = 0.7
			}
		} else if strings.Contains(pathLower, queryLower) {
			// Check full path
			score = 0.5
		} else {
			// Fuzzy match on filename
			fuzzyScore, _ := fuzzyMatch(queryLower, fileName)
			score = fuzzyScore * 0.4
		}

		if score > 0.2 {
			matches = append(matches, fileMatch{path: path, score: score})
		}
	}

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Extract paths
	result := make([]string, 0, limit)
	for i := 0; i < len(matches) && i < limit; i++ {
		result = append(result, matches[i].path)
	}

	return result, nil
}
