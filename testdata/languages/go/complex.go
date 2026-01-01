// Package main - complex.go tests complexity analysis patterns.
// Contains functions with various cyclomatic complexity levels.
package main

import (
	"errors"
	"fmt"
)

// SimpleFunction has complexity 1 - no branches.
// Expected: cyclomatic complexity = 1, nesting = 0
func SimpleFunction(x int) int {
	return x * 2
}

// LinearFunction has complexity 1 - sequential statements.
// Expected: cyclomatic complexity = 1, nesting = 0
func LinearFunction(a, b, c int) int {
	sum := a + b
	sum += c
	result := sum * 2
	return result
}

// SingleIfFunction has complexity 2 - one if statement.
// Expected: cyclomatic complexity = 2, nesting = 1
func SingleIfFunction(x int) string {
	if x > 0 {
		return "positive"
	}
	return "non-positive"
}

// IfElseFunction has complexity 2 - if/else.
// Expected: cyclomatic complexity = 2, nesting = 1
func IfElseFunction(x int) string {
	if x > 0 {
		return "positive"
	} else {
		return "non-positive"
	}
}

// MultipleIfFunction has complexity 4 - multiple independent ifs.
// Expected: cyclomatic complexity = 4, nesting = 1
func MultipleIfFunction(a, b, c bool) int {
	count := 0
	if a {
		count++
	}
	if b {
		count++
	}
	if c {
		count++
	}
	return count
}

// NestedIfFunction has complexity 3 - nested ifs.
// Expected: cyclomatic complexity = 3, nesting = 2
func NestedIfFunction(x, y int) string {
	if x > 0 {
		if y > 0 {
			return "both positive"
		}
		return "only x positive"
	}
	return "x not positive"
}

// DeeplyNestedFunction has high nesting depth.
// Expected: cyclomatic complexity = 5, nesting = 4
func DeeplyNestedFunction(a, b, c, d bool) string {
	if a {
		if b {
			if c {
				if d {
					return "all true"
				}
				return "d false"
			}
			return "c false"
		}
		return "b false"
	}
	return "a false"
}

// SwitchFunction has complexity based on cases.
// Expected: cyclomatic complexity = 5 (4 cases + default)
func SwitchFunction(day int) string {
	switch day {
	case 1:
		return "Monday"
	case 2:
		return "Tuesday"
	case 3:
		return "Wednesday"
	case 4, 5:
		return "Thursday or Friday"
	default:
		return "Weekend"
	}
}

// ForLoopFunction has complexity 2 - one loop.
// Expected: cyclomatic complexity = 2, nesting = 1
func ForLoopFunction(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	return sum
}

// ForRangeFunction has complexity 2 - range loop.
// Expected: cyclomatic complexity = 2, nesting = 1
func ForRangeFunction(items []int) int {
	sum := 0
	for _, item := range items {
		sum += item
	}
	return sum
}

// NestedLoopFunction has complexity 3 - nested loops.
// Expected: cyclomatic complexity = 3, nesting = 2
func NestedLoopFunction(n, m int) int {
	sum := 0
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			sum += i * j
		}
	}
	return sum
}

// LoopWithCondition has complexity 3 - loop + condition.
// Expected: cyclomatic complexity = 3, nesting = 2
func LoopWithCondition(items []int) int {
	sum := 0
	for _, item := range items {
		if item > 0 {
			sum += item
		}
	}
	return sum
}

// LogicalAndFunction has complexity 2 - && operator.
// Expected: cyclomatic complexity = 2 (short-circuit)
func LogicalAndFunction(a, b bool) bool {
	return a && b
}

// LogicalOrFunction has complexity 2 - || operator.
// Expected: cyclomatic complexity = 2 (short-circuit)
func LogicalOrFunction(a, b bool) bool {
	return a || b
}

// ComplexLogicalFunction has complexity 4 - multiple logical ops.
// Expected: cyclomatic complexity = 4
func ComplexLogicalFunction(a, b, c bool) bool {
	return (a && b) || (b && c) || (a && c)
}

// ErrorHandlingFunction has complexity 4 - multiple error checks.
// Expected: cyclomatic complexity = 4, nesting = 1
func ErrorHandlingFunction(x, y int) (int, error) {
	if x < 0 {
		return 0, errors.New("x must be non-negative")
	}
	if y < 0 {
		return 0, errors.New("y must be non-negative")
	}
	if y == 0 {
		return 0, errors.New("y must not be zero")
	}
	return x / y, nil
}

// HighComplexityFunction has high cyclomatic complexity.
// Expected: cyclomatic complexity >= 10
func HighComplexityFunction(op string, a, b int) (int, error) {
	switch op {
	case "add":
		return a + b, nil
	case "sub":
		return a - b, nil
	case "mul":
		return a * b, nil
	case "div":
		if b == 0 {
			return 0, errors.New("division by zero")
		}
		return a / b, nil
	case "mod":
		if b == 0 {
			return 0, errors.New("modulo by zero")
		}
		return a % b, nil
	case "pow":
		if b < 0 {
			return 0, errors.New("negative exponent")
		}
		result := 1
		for i := 0; i < b; i++ {
			result *= a
		}
		return result, nil
	case "max":
		if a > b {
			return a, nil
		}
		return b, nil
	case "min":
		if a < b {
			return a, nil
		}
		return b, nil
	default:
		return 0, fmt.Errorf("unknown operation: %s", op)
	}
}

// TypeSwitchFunction has complexity based on type cases.
// Expected: cyclomatic complexity = 4
func TypeSwitchFunction(v interface{}) string {
	switch v := v.(type) {
	case int:
		return fmt.Sprintf("int: %d", v)
	case string:
		return fmt.Sprintf("string: %s", v)
	case bool:
		return fmt.Sprintf("bool: %t", v)
	default:
		return "unknown type"
	}
}

// DeferFunction tests defer statement.
// Expected: cyclomatic complexity = 2
func DeferFunction(shouldPanic bool) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = "recovered"
		}
	}()

	if shouldPanic {
		panic("intentional panic")
	}

	return "normal"
}

// GoroutineFunction tests goroutine spawning.
// Expected: cyclomatic complexity = 2
func GoroutineFunction(items []int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for _, item := range items {
			ch <- item
		}
	}()
	return ch
}

// SelectFunction tests select statement.
// Expected: cyclomatic complexity = 4
func SelectFunction(ch1, ch2 <-chan int, done <-chan struct{}) int {
	select {
	case v := <-ch1:
		return v
	case v := <-ch2:
		return v * 2
	case <-done:
		return -1
	default:
		return 0
	}
}

// ExtremelyComplexFunction is intentionally complex for testing.
// Expected: cyclomatic complexity >= 15, nesting >= 4
func ExtremelyComplexFunction(category string, items []Item, threshold int) ([]Item, error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}

	var result []Item
	for _, item := range items {
		if item.Category != category {
			continue
		}

		switch item.Status {
		case "active":
			if item.Value > threshold {
				if item.Priority == "high" {
					result = append(result, item)
				} else if item.Priority == "medium" {
					if item.Value > threshold*2 {
						result = append(result, item)
					}
				}
			}
		case "pending":
			if item.CreatedAt > 0 {
				for _, tag := range item.Tags {
					if tag == "important" {
						result = append(result, item)
						break
					}
				}
			}
		case "archived":
			// Skip archived items
			continue
		default:
			if item.Value > 0 && item.Priority != "" {
				result = append(result, item)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no items found for category %s", category)
	}

	return result, nil
}

// Item represents a data item for testing.
type Item struct {
	Category  string
	Status    string
	Value     int
	Priority  string
	CreatedAt int64
	Tags      []string
}
