<?php
/**
 * Main module demonstrating various PHP patterns for parser testing.
 * Tests: entry points, classes, functions, namespaces.
 */

declare(strict_types=1);

namespace App;

require_once __DIR__ . '/Utils.php';
require_once __DIR__ . '/Models.php';

// Constants - tests constant extraction
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 30;
const APP_NAME = 'TestApp';

// Global variables
$appVersion = '1.0.0';
$initialized = false;

/**
 * Main entry point - should be marked as reachable.
 */
function main(): void
{
    global $initialized;

    echo "Starting " . APP_NAME . "\n";

    // Function calls - tests reference extraction
    $config = loadConfig();
    if (!initialize($config)) {
        fwrite(STDERR, "Initialization failed\n");
        exit(1);
    }

    // Create and start server
    $server = new Server($config);
    $server->start();

    // Using utility functions
    $items = ['a', 'b', 'c'];
    $result = Utils::processData($items);
    $output = Utils::formatOutput($result);
    echo $output . "\n";

    // Calling transitive functions
    runPipeline();

    // Cleanup
    $server->stop();
}

/**
 * Load configuration - called from main, should be reachable.
 */
function loadConfig(): Config
{
    return new Config('localhost', 8080, 'info');
}

/**
 * Initialize application - called from main, should be reachable.
 */
function initialize(Config $config): bool
{
    global $initialized;
    setupLogging($config->logLevel);
    $initialized = true;
    return true;
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
function setupLogging(string $level): void
{
    echo "Setting log level to: {$level}\n";
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
function runPipeline(): void
{
    $data = fetchData();
    $transformed = transformData($data);
    saveData($transformed);
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 */
function fetchData(): string
{
    return 'sample data';
}

/**
 * Transform data - called by runPipeline, should be reachable.
 */
function transformData(string $data): string
{
    return "transformed: {$data}";
}

/**
 * Save data - called by runPipeline, should be reachable.
 */
function saveData(string $data): void
{
    echo "Saving: {$data}\n";
}

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/**
 * This function is never called - DEAD CODE.
 */
function unusedFunction(): void
{
    echo "This is never executed\n";
}

/**
 * Also never called - DEAD CODE.
 */
function anotherUnused(): string
{
    return 'dead';
}

/**
 * Starts a chain of dead code - DEAD CODE.
 */
function deadChainStart(): void
{
    deadChainMiddle();
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
function deadChainMiddle(): void
{
    deadChainEnd();
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
function deadChainEnd(): void
{
    echo "End of dead chain\n";
}

// Run main if executed directly
if (php_sapi_name() === 'cli' && basename(__FILE__) === basename($_SERVER['SCRIPT_NAME'])) {
    main();
}
