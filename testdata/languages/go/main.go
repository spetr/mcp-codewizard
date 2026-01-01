// Package main demonstrates various Go patterns for parser testing.
// This file tests: entry points, function calls, variable declarations.
package main

import (
	"fmt"
	"os"
)

// Global variables - tests package-level initialization tracking
var (
	// AppName is the application name (exported).
	AppName = "TestApp"
	// appVersion is internal version (unexported).
	appVersion = "1.0.0"
	// logger uses function call in init - tests reference extraction
	logger = NewLogger("main")
)

// Constants - tests constant extraction
const (
	MaxRetries   = 3
	defaultDelay = 100
)

// main is the entry point - should be marked as reachable.
func main() {
	fmt.Println("Starting", AppName)

	// Direct function calls - tests reference extraction
	config := LoadConfig()
	if err := Initialize(config); err != nil {
		fmt.Fprintf(os.Stderr, "Init failed: %v\n", err)
		os.Exit(1)
	}

	// Method call on returned value
	server := NewServer(config)
	server.Start()

	// Using helper from utils.go
	result := ProcessData([]string{"a", "b", "c"})
	fmt.Println("Result:", result)

	// Calling function that calls other functions (transitive)
	RunPipeline()
}

// Initialize sets up the application - called from main, should be reachable.
func Initialize(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	setupLogging(cfg.LogLevel)
	return nil
}

// LoadConfig loads configuration - called from main, should be reachable.
func LoadConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     8080,
		LogLevel: "info",
	}
}

// setupLogging is internal helper - called from Initialize, should be reachable.
func setupLogging(level string) {
	fmt.Println("Log level:", level)
}

// RunPipeline orchestrates the data pipeline - tests transitive reachability.
func RunPipeline() {
	data := fetchData()
	transformed := transformData(data)
	saveData(transformed)
}

// fetchData fetches data - called by RunPipeline, should be reachable.
func fetchData() []byte {
	return []byte("sample data")
}

// transformData transforms data - called by RunPipeline, should be reachable.
func transformData(data []byte) []byte {
	return append([]byte("transformed: "), data...)
}

// saveData saves data - called by RunPipeline, should be reachable.
func saveData(data []byte) {
	fmt.Println("Saving:", string(data))
}

// unusedEntryHelper is never called - DEAD CODE.
func unusedEntryHelper() {
	fmt.Println("This is never called")
}

// anotherUnused is also never called - DEAD CODE.
func anotherUnused() string {
	return "dead"
}

// deadChainStart starts a chain of dead code - DEAD CODE.
func deadChainStart() {
	deadChainMiddle()
}

// deadChainMiddle is in the middle of dead chain - DEAD CODE (transitive).
func deadChainMiddle() {
	deadChainEnd()
}

// deadChainEnd is at the end of dead chain - DEAD CODE (transitive).
func deadChainEnd() {
	fmt.Println("End of dead chain")
}
