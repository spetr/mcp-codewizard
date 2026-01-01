/**
 * Models module - tests class definitions, inheritance, static methods.
 */

/**
 * Configuration class - tests class extraction.
 */
class Config {
    /**
     * Create a config instance.
     * @param {Object} options - Configuration options
     */
    constructor(options = {}) {
        this.host = options.host || 'localhost';
        this.port = options.port || 8080;
        this.logLevel = options.logLevel || 'info';
        this.options = options.options || {};
    }

    /**
     * Validate configuration.
     * @returns {boolean}
     */
    validate() {
        if (!this.host) {
            return false;
        }
        if (this.port <= 0 || this.port > 65535) {
            return false;
        }
        return true;
    }

    /**
     * Clone the configuration.
     * @returns {Config}
     */
    clone() {
        return new Config({
            host: this.host,
            port: this.port,
            logLevel: this.logLevel,
            options: { ...this.options }
        });
    }

    /**
     * Static factory method.
     * @param {string} json - JSON string
     * @returns {Config}
     */
    static fromJSON(json) {
        const data = JSON.parse(json);
        return new Config(data);
    }
}

/**
 * Server class - tests class with methods.
 */
class Server {
    /**
     * Create a server instance.
     * @param {Config} config - Configuration
     */
    constructor(config) {
        this.config = config;
        this._running = false;
        this._logger = new Logger('server');
    }

    /**
     * Start the server - called from main, should be reachable.
     */
    start() {
        this._running = true;
        this._logger.info(`Starting server on ${this.config.host}:${this.config.port}`);
        this._listen();
    }

    /**
     * Stop the server - public method.
     */
    stop() {
        this._running = false;
        this._logger.info('Stopping server');
    }

    /**
     * Internal listen method - called by start, should be reachable.
     * @private
     */
    _listen() {
        // Simulated listening
    }

    /**
     * Handle connection - DEAD CODE (never called).
     * @param {Object} conn - Connection object
     * @private
     */
    _handleConnection(conn) {
        // Handle connection
    }

    /**
     * Check if server is running.
     * @returns {boolean}
     */
    get isRunning() {
        return this._running;
    }

    /**
     * Set running status.
     * @param {boolean} value
     */
    set isRunning(value) {
        this._running = value;
    }
}

/**
 * Logger class - used by Server.
 */
class Logger {
    /**
     * Create a logger.
     * @param {string} prefix - Log prefix
     */
    constructor(prefix) {
        this.prefix = prefix;
        this.level = 1;
    }

    /**
     * Log info message - called from Server.start.
     * @param {string} message
     */
    info(message) {
        console.log(`[INFO] ${this.prefix}: ${message}`);
    }

    /**
     * Log debug message - DEAD CODE.
     * @param {string} message
     */
    debug(message) {
        if (this.level >= 2) {
            console.log(`[DEBUG] ${this.prefix}: ${message}`);
        }
    }

    /**
     * Log error message - DEAD CODE.
     * @param {string} message
     */
    error(message) {
        console.log(`[ERROR] ${this.prefix}: ${message}`);
    }
}

/**
 * Factory function for Server - called from main.
 * @param {Config} config
 * @returns {Server}
 */
function createServer(config) {
    return new Server(config);
}

// ============================================================================
// Inheritance - tests class inheritance extraction
// ============================================================================

/**
 * Base handler class.
 */
class BaseHandler {
    constructor(name) {
        this.name = name;
    }

    /**
     * Handle request - to be overridden.
     * @param {Object} request
     * @returns {Object}
     */
    handle(request) {
        throw new Error('Not implemented');
    }

    /**
     * Get handler name.
     * @returns {string}
     */
    getName() {
        return this.name;
    }
}

/**
 * Echo handler - extends BaseHandler - DEAD CODE.
 */
class EchoHandler extends BaseHandler {
    constructor() {
        super('echo');
    }

    /**
     * Handle request by echoing body.
     * @param {Object} request
     * @returns {Object}
     */
    handle(request) {
        return {
            status: 200,
            body: request.body
        };
    }
}

/**
 * JSON handler - extends BaseHandler - DEAD CODE.
 */
class JsonHandler extends BaseHandler {
    constructor() {
        super('json');
    }

    /**
     * Handle JSON request.
     * @param {Object} request
     * @returns {Object}
     */
    handle(request) {
        const data = JSON.parse(request.body);
        return {
            status: 200,
            body: JSON.stringify({ processed: true, data })
        };
    }
}

// ============================================================================
// ES6+ features - tests modern JavaScript extraction
// ============================================================================

/**
 * Class with private fields (ES2022).
 */
class ModernClass {
    #privateField = 'private';
    static #staticPrivate = 'static private';

    constructor(value) {
        this.#privateField = value;
    }

    getPrivate() {
        return this.#privateField;
    }

    static getStaticPrivate() {
        return ModernClass.#staticPrivate;
    }
}

/**
 * Class with static block (ES2022).
 */
class ClassWithStaticBlock {
    static data = [];

    static {
        this.data.push('initialized');
    }

    static getData() {
        return this.data;
    }
}

// ============================================================================
// Unused classes - DEAD CODE
// ============================================================================

/**
 * Class that is never instantiated - DEAD CODE.
 */
class UnusedClass {
    constructor(value) {
        this.value = value;
    }

    process() {
        console.log(`Processing: ${this.value}`);
    }
}

/**
 * Another unused class - DEAD CODE.
 */
class AnotherUnusedClass {
    constructor() {
        this.data = {};
    }

    getData() {
        return this.data;
    }
}

// ============================================================================
// Exports
// ============================================================================

module.exports = {
    Config,
    Server,
    Logger,
    createServer,
    BaseHandler,
    EchoHandler,
    JsonHandler,
    ModernClass,
    ClassWithStaticBlock
};
