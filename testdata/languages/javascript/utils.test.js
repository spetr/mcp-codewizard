/**
 * Test module - tests test function detection.
 * All functions here should be excluded from dead code analysis.
 */

const { processData, hashString, filterItems } = require('./utils');
const { Config, Server, Logger } = require('./models');

// ============================================================================
// Jest-style tests - should be excluded
// ============================================================================

describe('processData', () => {
    test('should process basic data', () => {
        const result = processData(['hello', 'world']);
        expect(result).toBe('HELLO, WORLD');
    });

    test('should handle empty array', () => {
        const result = processData([]);
        expect(result).toBe('');
    });

    test('should handle single item', () => {
        const result = processData(['test']);
        expect(result).toBe('TEST');
    });

    it('should work with numbers converted to strings', () => {
        const result = processData(['1', '2', '3']);
        expect(result).toBe('1, 2, 3');
    });
});

describe('hashString', () => {
    test('should return 64 character hash', () => {
        const result = hashString('test');
        expect(result).toHaveLength(64);
    });

    test('should be deterministic', () => {
        const result1 = hashString('test');
        const result2 = hashString('test');
        expect(result1).toBe(result2);
    });
});

describe('filterItems', () => {
    test('should filter positive numbers', () => {
        const items = [-1, 0, 1, 2, 3];
        const result = filterItems(items, x => x > 0);
        expect(result).toEqual([1, 2, 3]);
    });

    test('should return empty array when nothing matches', () => {
        const items = [1, 2, 3];
        const result = filterItems(items, x => x > 10);
        expect(result).toEqual([]);
    });
});

describe('Config', () => {
    test('should create with defaults', () => {
        const config = new Config();
        expect(config.host).toBe('localhost');
        expect(config.port).toBe(8080);
    });

    test('should accept custom options', () => {
        const config = new Config({ host: '0.0.0.0', port: 3000 });
        expect(config.host).toBe('0.0.0.0');
        expect(config.port).toBe(3000);
    });

    test('should validate correctly', () => {
        const validConfig = new Config({ host: 'localhost', port: 8080 });
        expect(validConfig.validate()).toBe(true);

        const invalidConfig = new Config({ host: '', port: 8080 });
        expect(invalidConfig.validate()).toBe(false);
    });

    test('should clone correctly', () => {
        const original = new Config({ host: 'localhost', port: 8080 });
        const cloned = original.clone();
        expect(cloned).not.toBe(original);
        expect(cloned.host).toBe(original.host);
    });
});

describe('Server', () => {
    let server;
    let config;

    beforeEach(() => {
        config = new Config({ host: 'localhost', port: 8080 });
        server = new Server(config);
    });

    afterEach(() => {
        server = null;
    });

    test('should start correctly', () => {
        server.start();
        expect(server.isRunning).toBe(true);
    });

    test('should stop correctly', () => {
        server.start();
        server.stop();
        expect(server.isRunning).toBe(false);
    });
});

describe('Logger', () => {
    test('should log info messages', () => {
        const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
        const logger = new Logger('test');
        logger.info('test message');
        expect(consoleSpy).toHaveBeenCalledWith('[INFO] test: test message');
        consoleSpy.mockRestore();
    });
});

// ============================================================================
// Mocha-style tests - should be excluded
// ============================================================================

describe('Mocha style tests', function() {
    it('should work with mocha style', function() {
        const result = processData(['a']);
        expect(result).toBe('A');
    });

    context('when processing multiple items', function() {
        it('should join with comma', function() {
            const result = processData(['a', 'b']);
            expect(result).toBe('A, B');
        });
    });
});

// ============================================================================
// Test helpers - should be excluded
// ============================================================================

/**
 * Create mock config for tests.
 * @returns {Config}
 */
function createMockConfig() {
    return new Config({
        host: 'test-host',
        port: 9999,
        logLevel: 'debug'
    });
}

/**
 * Create mock server for tests.
 * @param {Config} config
 * @returns {Server}
 */
function createMockServer(config) {
    return new Server(config || createMockConfig());
}

/**
 * Setup test environment.
 * @returns {Object}
 */
function setupTestEnvironment() {
    return {
        config: createMockConfig(),
        logger: new Logger('test')
    };
}

/**
 * Teardown test environment.
 * @param {Object} env
 */
function teardownTestEnvironment(env) {
    // Cleanup
}

/**
 * Wait for condition - test utility.
 * @param {Function} condition
 * @param {number} timeout
 * @returns {Promise<void>}
 */
async function waitFor(condition, timeout = 5000) {
    const start = Date.now();
    while (Date.now() - start < timeout) {
        if (condition()) {
            return;
        }
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error('Timeout waiting for condition');
}

/**
 * Assert arrays are equal - custom assertion.
 * @param {Array} actual
 * @param {Array} expected
 */
function assertArraysEqual(actual, expected) {
    expect(actual).toHaveLength(expected.length);
    actual.forEach((item, index) => {
        expect(item).toBe(expected[index]);
    });
}

// ============================================================================
// Test fixtures
// ============================================================================

const testFixtures = {
    emptyArray: [],
    stringArray: ['a', 'b', 'c'],
    numberArray: [1, 2, 3],
    mixedArray: ['a', 1, null, undefined],
    config: { host: 'localhost', port: 8080 }
};

// ============================================================================
// Module exports for test utilities
// ============================================================================

module.exports = {
    createMockConfig,
    createMockServer,
    setupTestEnvironment,
    teardownTestEnvironment,
    waitFor,
    assertArraysEqual,
    testFixtures
};
