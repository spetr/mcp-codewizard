// Package main - utils.go tests utility functions, closures, and various patterns.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// ProcessData processes string data - called from main, should be reachable.
// Tests: function with slice parameter and return value.
func ProcessData(items []string) string {
	// Using closure - tests closure extraction
	mapper := func(s string) string {
		return strings.ToUpper(s)
	}

	var results []string
	for _, item := range items {
		results = append(results, mapper(item))
	}

	return strings.Join(results, ", ")
}

// HashString computes SHA256 hash - DEAD CODE (never called).
func HashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// FilterStrings filters strings by predicate - DEAD CODE.
// Tests: function type parameter.
func FilterStrings(items []string, predicate func(string) bool) []string {
	var result []string
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// MapStrings transforms strings - DEAD CODE.
func MapStrings(items []string, transform func(string) string) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}
	return result
}

// ReduceStrings reduces strings to single value - DEAD CODE.
func ReduceStrings(items []string, initial string, reducer func(string, string) string) string {
	result := initial
	for _, item := range items {
		result = reducer(result, item)
	}
	return result
}

// SortByLength sorts strings by length - tests sort.Interface pattern.
type SortByLength []string

func (s SortByLength) Len() int           { return len(s) }
func (s SortByLength) Less(i, j int) bool { return len(s[i]) < len(s[j]) }
func (s SortByLength) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortStringsByLength sorts strings by their length - DEAD CODE.
func SortStringsByLength(items []string) []string {
	sorted := make(SortByLength, len(items))
	copy(sorted, items)
	sort.Sort(sorted)
	return sorted
}

// MustParse parses or panics - tests panic pattern.
func MustParse(s string) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %v", s, err))
	}
	return result
}

// SafeParse parses with error handling - DEAD CODE.
func SafeParse(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("parse error: %w", err)
	}
	return result, nil
}

// Retry retries a function with exponential backoff - DEAD CODE.
// Tests: complex function signature with function parameter.
func Retry(attempts int, fn func() error) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", attempts, lastErr)
}

// WithTimeout wraps function with timeout - DEAD CODE.
// Tests: channel usage, goroutine pattern.
func WithTimeout(timeout int, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	default:
		return fmt.Errorf("timeout after %d ms", timeout)
	}
}

// Cache provides simple caching - tests struct with methods.
type Cache struct {
	data map[string]interface{}
}

// NewCache creates a new cache - DEAD CODE.
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]interface{}),
	}
}

// Get retrieves value from cache - DEAD CODE.
func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.data[key]
	return val, ok
}

// Set stores value in cache - DEAD CODE.
func (c *Cache) Set(key string, value interface{}) {
	c.data[key] = value
}

// Delete removes value from cache - DEAD CODE.
func (c *Cache) Delete(key string) {
	delete(c.data, key)
}

// Clear removes all values - DEAD CODE.
func (c *Cache) Clear() {
	c.data = make(map[string]interface{})
}

// Builder pattern - tests method chaining.
type QueryBuilder struct {
	table   string
	columns []string
	where   []string
	limit   int
}

// NewQueryBuilder creates a new builder - DEAD CODE.
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{table: table}
}

// Select adds columns - DEAD CODE.
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = append(qb.columns, columns...)
	return qb
}

// Where adds condition - DEAD CODE.
func (qb *QueryBuilder) Where(condition string) *QueryBuilder {
	qb.where = append(qb.where, condition)
	return qb
}

// Limit sets limit - DEAD CODE.
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

// Build generates SQL - DEAD CODE.
func (qb *QueryBuilder) Build() string {
	sql := "SELECT "
	if len(qb.columns) == 0 {
		sql += "*"
	} else {
		sql += strings.Join(qb.columns, ", ")
	}
	sql += " FROM " + qb.table

	if len(qb.where) > 0 {
		sql += " WHERE " + strings.Join(qb.where, " AND ")
	}

	if qb.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	return sql
}

// privateHelper is a private helper - DEAD CODE.
func privateHelper(x, y int) int {
	return x + y
}

// init function - tests init detection.
func init() {
	fmt.Println("utils.go initialized")
}
