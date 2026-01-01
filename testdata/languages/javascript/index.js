/**
 * Main module demonstrating various JavaScript patterns for parser testing.
 * Tests: entry points, imports, exports, function calls, classes.
 */

const { processData, formatOutput } = require('./utils');
const { Config, Server, createServer } = require('./models');

// Module-level constants - tests constant extraction
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 30;

// Module-level variables - tests variable extraction
let appName = 'TestApp';
const _internalVersion = '1.0.0';

// Module-level initialization - tests package-level reference tracking
const logger = createLogger('main');

/**
 * Main entry point - should be marked as reachable.
 */
function main() {
    console.log(`Starting ${appName}`);

    // Function calls - tests reference extraction
    const config = loadConfig();
    if (!initialize(config)) {
        console.error('Initialization failed');
        process.exit(1);
    }

    // Method calls
    const server = createServer(config);
    server.start();

    // Using imported function
    const result = processData(['a', 'b', 'c']);
    console.log(formatOutput(result));

    // Calling transitive functions
    runPipeline();
}

/**
 * Load configuration - called from main, should be reachable.
 * @returns {Config} Configuration object
 */
function loadConfig() {
    return new Config({
        host: 'localhost',
        port: 8080,
        logLevel: 'info'
    });
}

/**
 * Initialize application - called from main, should be reachable.
 * @param {Config} config - Configuration object
 * @returns {boolean} Success status
 */
function initialize(config) {
    if (!config) {
        return false;
    }
    setupLogging(config.logLevel);
    return true;
}

/**
 * Internal helper - called from initialize, should be reachable.
 * @param {string} level - Log level
 */
function setupLogging(level) {
    console.log(`Setting log level to: ${level}`);
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
function runPipeline() {
    const data = fetchData();
    const transformed = transformData(data);
    saveData(transformed);
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 * @returns {Buffer}
 */
function fetchData() {
    return Buffer.from('sample data');
}

/**
 * Transform data - called by runPipeline, should be reachable.
 * @param {Buffer} data
 * @returns {Buffer}
 */
function transformData(data) {
    return Buffer.concat([Buffer.from('transformed: '), data]);
}

/**
 * Save data - called by runPipeline, should be reachable.
 * @param {Buffer} data
 */
function saveData(data) {
    console.log(`Saving: ${data.toString()}`);
}

/**
 * Create a logger - called from module level, should be reachable.
 * @param {string} name
 * @returns {Logger}
 */
function createLogger(name) {
    return new Logger(name);
}

/**
 * Simple logger class.
 */
class Logger {
    constructor(name) {
        this.name = name;
    }

    info(message) {
        console.log(`[INFO] ${this.name}: ${message}`);
    }
}

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/**
 * This function is never called - DEAD CODE.
 */
function unusedFunction() {
    console.log('This is never executed');
}

/**
 * Also never called - DEAD CODE.
 * @returns {string}
 */
function anotherUnused() {
    return 'dead';
}

/**
 * Starts a chain of dead code - DEAD CODE.
 */
function deadChainStart() {
    deadChainMiddle();
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
function deadChainMiddle() {
    deadChainEnd();
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
function deadChainEnd() {
    console.log('End of dead chain');
}

/**
 * Private unused function - DEAD CODE.
 */
function _privateUnused() {
    // Empty
}

// ============================================================================
// Arrow functions - tests arrow function extraction
// ============================================================================

/**
 * Arrow function - DEAD CODE.
 */
const arrowFunction = (x) => x * 2;

/**
 * Arrow function with block - DEAD CODE.
 */
const arrowWithBlock = (x, y) => {
    const sum = x + y;
    return sum * 2;
};

/**
 * Async arrow function - DEAD CODE.
 */
const asyncArrow = async (url) => {
    const response = await fetch(url);
    return response.json();
};

// ============================================================================
// IIFE pattern - tests IIFE extraction
// ============================================================================

/**
 * IIFE - tests immediately invoked function expression.
 */
const modulePattern = (function() {
    let privateVar = 'private';

    function privateMethod() {
        return privateVar;
    }

    return {
        publicMethod: function() {
            return privateMethod();
        }
    };
})();

// ============================================================================
// Exports
// ============================================================================

module.exports = {
    main,
    loadConfig,
    initialize,
    Logger,
    MAX_RETRIES,
    DEFAULT_TIMEOUT
};

// Entry point
if (require.main === module) {
    main();
}
