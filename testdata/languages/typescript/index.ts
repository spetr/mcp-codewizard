/**
 * Main module demonstrating various TypeScript patterns for parser testing.
 * Tests: entry points, imports, exports, types, interfaces.
 */

import { processData, formatOutput } from './utils';
import { Config, Server, createServer, Logger } from './models';
import type { Handler, Request, Response } from './types';

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
function main(): void {
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
 */
function loadConfig(): Config {
    return new Config({
        host: 'localhost',
        port: 8080,
        logLevel: 'info'
    });
}

/**
 * Initialize application - called from main, should be reachable.
 */
function initialize(config: Config): boolean {
    if (!config) {
        return false;
    }
    setupLogging(config.logLevel);
    return true;
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
function setupLogging(level: string): void {
    console.log(`Setting log level to: ${level}`);
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
function runPipeline(): void {
    const data = fetchData();
    const transformed = transformData(data);
    saveData(transformed);
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 */
function fetchData(): Buffer {
    return Buffer.from('sample data');
}

/**
 * Transform data - called by runPipeline, should be reachable.
 */
function transformData(data: Buffer): Buffer {
    return Buffer.concat([Buffer.from('transformed: '), data]);
}

/**
 * Save data - called by runPipeline, should be reachable.
 */
function saveData(data: Buffer): void {
    console.log(`Saving: ${data.toString()}`);
}

/**
 * Create a logger - called from module level, should be reachable.
 */
function createLogger(name: string): Logger {
    return new Logger(name);
}

// ============================================================================
// Generic functions - tests TypeScript generics extraction
// ============================================================================

/**
 * Generic identity function - tests type parameter extraction.
 */
function identity<T>(value: T): T {
    return value;
}

/**
 * Generic with constraint - tests type constraint extraction.
 */
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
    return obj[key];
}

/**
 * Generic with multiple type parameters - DEAD CODE.
 */
function map<T, U>(items: T[], fn: (item: T) => U): U[] {
    return items.map(fn);
}

/**
 * Generic with default type - DEAD CODE.
 */
function createContainer<T = string>(value: T): { value: T } {
    return { value };
}

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/**
 * This function is never called - DEAD CODE.
 */
function unusedFunction(): void {
    console.log('This is never executed');
}

/**
 * Also never called - DEAD CODE.
 */
function anotherUnused(): string {
    return 'dead';
}

/**
 * Starts a chain of dead code - DEAD CODE.
 */
function deadChainStart(): void {
    deadChainMiddle();
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
function deadChainMiddle(): void {
    deadChainEnd();
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
function deadChainEnd(): void {
    console.log('End of dead chain');
}

/**
 * Private unused function - DEAD CODE.
 */
function _privateUnused(): void {
    // Empty
}

// ============================================================================
// Arrow functions with types - tests typed arrow function extraction
// ============================================================================

/**
 * Typed arrow function - DEAD CODE.
 */
const typedArrow = (x: number): number => x * 2;

/**
 * Arrow with generics - DEAD CODE.
 */
const genericArrow = <T>(x: T): T => x;

/**
 * Async typed arrow - DEAD CODE.
 */
const asyncTypedArrow = async (url: string): Promise<object> => {
    const response = await fetch(url);
    return response.json();
};

// ============================================================================
// Type assertions and guards - tests TypeScript-specific patterns
// ============================================================================

/**
 * Type guard function - DEAD CODE.
 */
function isString(value: unknown): value is string {
    return typeof value === 'string';
}

/**
 * Type assertion function - DEAD CODE.
 */
function assertNonNull<T>(value: T | null | undefined): asserts value is T {
    if (value === null || value === undefined) {
        throw new Error('Value is null or undefined');
    }
}

/**
 * Function with overloads - DEAD CODE.
 */
function processValue(value: string): string;
function processValue(value: number): number;
function processValue(value: string | number): string | number {
    if (typeof value === 'string') {
        return value.toUpperCase();
    }
    return value * 2;
}

// ============================================================================
// Exports
// ============================================================================

export {
    main,
    loadConfig,
    initialize,
    identity,
    getProperty,
    MAX_RETRIES,
    DEFAULT_TIMEOUT
};

// Default export
export default main;

// Entry point
if (require.main === module) {
    main();
}
