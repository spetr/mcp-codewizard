/**
 * Main module demonstrating various Swift patterns for parser testing.
 * Tests: entry points, classes, structs, protocols.
 */

import Foundation

// Constants - tests constant extraction
let MAX_RETRIES = 3
let DEFAULT_TIMEOUT = 30
let APP_NAME = "TestApp"

// Global variables
private var appVersion = "1.0.0"
private var initialized = false

/**
 * Main entry point - should be marked as reachable.
 */
func main() {
    print("Starting \(APP_NAME)")

    // Function calls - tests reference extraction
    let config = loadConfig()
    guard initialize(config: config) else {
        fputs("Initialization failed\n", stderr)
        exit(1)
    }

    // Create and start server
    let server = Server(config: config)
    server.start()

    // Using utility functions
    let items = ["a", "b", "c"]
    let result = processData(items: items)
    let output = formatOutput(data: result)
    print(output)

    // Calling transitive functions
    runPipeline()

    // Cleanup
    server.stop()
}

/**
 * Load configuration - called from main, should be reachable.
 */
func loadConfig() -> Config {
    return Config(host: "localhost", port: 8080, logLevel: "info")
}

/**
 * Initialize application - called from main, should be reachable.
 */
func initialize(config: Config) -> Bool {
    setupLogging(level: config.logLevel)
    initialized = true
    return true
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
private func setupLogging(level: String) {
    print("Setting log level to: \(level)")
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
func runPipeline() {
    let data = fetchData()
    let transformed = transformData(data: data)
    saveData(data: transformed)
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 */
func fetchData() -> String {
    return "sample data"
}

/**
 * Transform data - called by runPipeline, should be reachable.
 */
func transformData(data: String) -> String {
    return "transformed: \(data)"
}

/**
 * Save data - called by runPipeline, should be reachable.
 */
func saveData(data: String) {
    print("Saving: \(data)")
}

/**
 * Process string data - called from main, should be reachable.
 */
func processData(items: [String]) -> String {
    return items.map { $0.uppercased() }.joined(separator: ", ")
}

/**
 * Format output string - called from main, should be reachable.
 */
func formatOutput(data: String) -> String {
    return "Result: \(data)"
}

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/**
 * This function is never called - DEAD CODE.
 */
func unusedFunction() {
    print("This is never executed")
}

/**
 * Also never called - DEAD CODE.
 */
func anotherUnused() -> String {
    return "dead"
}

/**
 * Starts a chain of dead code - DEAD CODE.
 */
func deadChainStart() {
    deadChainMiddle()
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
private func deadChainMiddle() {
    deadChainEnd()
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
private func deadChainEnd() {
    print("End of dead chain")
}

// Run main
main()
