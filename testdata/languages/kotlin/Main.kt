/**
 * Main module demonstrating various Kotlin patterns for parser testing.
 * Tests: entry points, classes, data classes, sealed classes.
 */

package app

// Constants - tests constant extraction
const val MAX_RETRIES = 3
const val DEFAULT_TIMEOUT = 30
const val APP_NAME = "TestApp"

// Top-level variables
private var appVersion = "1.0.0"
private var initialized = false

/**
 * Main entry point - should be marked as reachable.
 */
fun main(args: Array<String>) {
    println("Starting $APP_NAME")

    // Function calls - tests reference extraction
    val config = loadConfig()
    if (!initialize(config)) {
        System.err.println("Initialization failed")
        System.exit(1)
    }

    // Create and start server
    val server = Server(config)
    server.start()

    // Using utility functions
    val items = listOf("a", "b", "c")
    val result = processData(items)
    val output = formatOutput(result)
    println(output)

    // Calling transitive functions
    runPipeline()

    // Cleanup
    server.stop()
}

/**
 * Load configuration - called from main, should be reachable.
 */
fun loadConfig(): Config = Config("localhost", 8080, "info")

/**
 * Initialize application - called from main, should be reachable.
 */
fun initialize(config: Config): Boolean {
    setupLogging(config.logLevel)
    initialized = true
    return true
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
private fun setupLogging(level: String) {
    println("Setting log level to: $level")
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
fun runPipeline() {
    val data = fetchData()
    val transformed = transformData(data)
    saveData(transformed)
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 */
fun fetchData(): String = "sample data"

/**
 * Transform data - called by runPipeline, should be reachable.
 */
fun transformData(data: String): String = "transformed: $data"

/**
 * Save data - called by runPipeline, should be reachable.
 */
fun saveData(data: String) {
    println("Saving: $data")
}

/**
 * Process string data - called from main, should be reachable.
 */
fun processData(items: List<String>): String =
    items.map { it.uppercase() }.joinToString(", ")

/**
 * Format output string - called from main, should be reachable.
 */
fun formatOutput(data: String): String = "Result: $data"

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/**
 * This function is never called - DEAD CODE.
 */
@Suppress("unused")
fun unusedFunction() {
    println("This is never executed")
}

/**
 * Also never called - DEAD CODE.
 */
@Suppress("unused")
fun anotherUnused(): String = "dead"

/**
 * Starts a chain of dead code - DEAD CODE.
 */
@Suppress("unused")
fun deadChainStart() {
    deadChainMiddle()
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
@Suppress("unused")
private fun deadChainMiddle() {
    deadChainEnd()
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
@Suppress("unused")
private fun deadChainEnd() {
    println("End of dead chain")
}
