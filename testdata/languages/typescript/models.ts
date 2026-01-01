/**
 * Models module - tests TypeScript class definitions.
 * Tests: classes, decorators, access modifiers, generics.
 */

import type { Handler, Request, Response, ConfigOptions, Repository } from './types';

/**
 * Configuration class - tests class extraction with types.
 */
export class Config {
    public host: string;
    public port: number;
    public logLevel: string;
    public options: Record<string, unknown>;

    constructor(options: Partial<ConfigOptions> = {}) {
        this.host = options.host || 'localhost';
        this.port = options.port || 8080;
        this.logLevel = 'info';
        this.options = {};
    }

    /**
     * Validate configuration.
     */
    validate(): boolean {
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
     */
    clone(): Config {
        const cloned = new Config({
            host: this.host,
            port: this.port
        });
        cloned.logLevel = this.logLevel;
        cloned.options = { ...this.options };
        return cloned;
    }

    /**
     * Static factory method.
     */
    static fromJSON(json: string): Config {
        const data = JSON.parse(json);
        return new Config(data);
    }
}

/**
 * Server class - tests class with access modifiers.
 */
export class Server {
    private _running: boolean = false;
    private readonly _logger: Logger;
    protected config: Config;

    constructor(config: Config) {
        this.config = config;
        this._logger = new Logger('server');
    }

    /**
     * Start the server - called from main, should be reachable.
     */
    public start(): void {
        this._running = true;
        this._logger.info(`Starting server on ${this.config.host}:${this.config.port}`);
        this._listen();
    }

    /**
     * Stop the server.
     */
    public stop(): void {
        this._running = false;
        this._logger.info('Stopping server');
    }

    /**
     * Internal listen method - called by start.
     */
    private _listen(): void {
        // Simulated listening
    }

    /**
     * Handle connection - DEAD CODE.
     */
    private _handleConnection(conn: unknown): void {
        // Handle connection
    }

    /**
     * Getter for running status.
     */
    get isRunning(): boolean {
        return this._running;
    }

    /**
     * Setter for running status.
     */
    set isRunning(value: boolean) {
        this._running = value;
    }
}

/**
 * Logger class.
 */
export class Logger {
    private prefix: string;
    private level: number = 1;

    constructor(prefix: string) {
        this.prefix = prefix;
    }

    /**
     * Log info message.
     */
    info(message: string): void {
        console.log(`[INFO] ${this.prefix}: ${message}`);
    }

    /**
     * Log debug message - DEAD CODE.
     */
    debug(message: string): void {
        if (this.level >= 2) {
            console.log(`[DEBUG] ${this.prefix}: ${message}`);
        }
    }

    /**
     * Log error message - DEAD CODE.
     */
    error(message: string): void {
        console.log(`[ERROR] ${this.prefix}: ${message}`);
    }
}

/**
 * Factory function for Server.
 */
export function createServer(config: Config): Server {
    return new Server(config);
}

// ============================================================================
// Abstract class - tests abstract class extraction
// ============================================================================

/**
 * Abstract base handler.
 */
export abstract class BaseHandler implements Handler {
    protected name_: string;

    constructor(name: string) {
        this.name_ = name;
    }

    /**
     * Abstract handle method.
     */
    abstract handle(request: Request): Promise<Response>;

    /**
     * Get handler name.
     */
    name(): string {
        return this.name_;
    }

    /**
     * Pre-process request.
     */
    protected preProcess(request: Request): Request {
        return request;
    }
}

/**
 * Echo handler - extends BaseHandler - DEAD CODE.
 */
export class EchoHandler extends BaseHandler {
    constructor() {
        super('echo');
    }

    async handle(request: Request): Promise<Response> {
        return {
            status: 200,
            body: request.body
        };
    }
}

/**
 * JSON handler - extends BaseHandler - DEAD CODE.
 */
export class JsonHandler extends BaseHandler {
    constructor() {
        super('json');
    }

    async handle(request: Request): Promise<Response> {
        const body = request.body ? JSON.parse(request.body.toString()) : {};
        return {
            status: 200,
            body: Buffer.from(JSON.stringify({ processed: true, data: body }))
        };
    }
}

// ============================================================================
// Generic class - tests generic class extraction
// ============================================================================

/**
 * Generic container class.
 */
export class Container<T> {
    private items: T[] = [];

    add(item: T): void {
        this.items.push(item);
    }

    get(index: number): T {
        return this.items[index];
    }

    all(): T[] {
        return [...this.items];
    }

    map<U>(fn: (item: T) => U): U[] {
        return this.items.map(fn);
    }

    filter(predicate: (item: T) => boolean): T[] {
        return this.items.filter(predicate);
    }
}

/**
 * Generic repository implementation - DEAD CODE.
 */
export class InMemoryRepository<T extends { id: string }> implements Repository<T> {
    private items: Map<string, T> = new Map();

    async findById(id: string): Promise<T | null> {
        return this.items.get(id) || null;
    }

    async findAll(): Promise<T[]> {
        return Array.from(this.items.values());
    }

    async save(entity: T): Promise<T> {
        this.items.set(entity.id, entity);
        return entity;
    }

    async delete(id: string): Promise<void> {
        this.items.delete(id);
    }
}

// ============================================================================
// Decorators - tests decorator extraction (experimental)
// ============================================================================

/**
 * Class decorator - DEAD CODE.
 */
function sealed(constructor: Function) {
    Object.seal(constructor);
    Object.seal(constructor.prototype);
}

/**
 * Method decorator - DEAD CODE.
 */
function log(
    target: any,
    propertyKey: string,
    descriptor: PropertyDescriptor
): PropertyDescriptor {
    const originalMethod = descriptor.value;
    descriptor.value = function(...args: any[]) {
        console.log(`Calling ${propertyKey} with:`, args);
        const result = originalMethod.apply(this, args);
        console.log(`Result:`, result);
        return result;
    };
    return descriptor;
}

/**
 * Property decorator - DEAD CODE.
 */
function required(target: any, propertyKey: string) {
    // Mark property as required
}

/**
 * Parameter decorator - DEAD CODE.
 */
function validate(target: any, propertyKey: string, parameterIndex: number) {
    // Validate parameter
}

/**
 * Decorated class - DEAD CODE.
 */
@sealed
export class DecoratedClass {
    @required
    private name: string;

    constructor(name: string) {
        this.name = name;
    }

    @log
    greet(@validate message: string): string {
        return `${this.name}: ${message}`;
    }
}

// ============================================================================
// Mixin pattern - tests mixin extraction
// ============================================================================

/**
 * Timestamped mixin.
 */
type Constructor<T = {}> = new (...args: any[]) => T;

function Timestamped<TBase extends Constructor>(Base: TBase) {
    return class extends Base {
        createdAt = new Date();
        updatedAt = new Date();

        touch() {
            this.updatedAt = new Date();
        }
    };
}

/**
 * Identifiable mixin.
 */
function Identifiable<TBase extends Constructor>(Base: TBase) {
    return class extends Base {
        id = Math.random().toString(36).substr(2, 9);
    };
}

/**
 * Base entity class.
 */
class BaseEntity {
    constructor(public name: string) {}
}

/**
 * Entity with mixins - DEAD CODE.
 */
export const Entity = Identifiable(Timestamped(BaseEntity));

// ============================================================================
// Unused classes - DEAD CODE
// ============================================================================

/**
 * Unused class - DEAD CODE.
 */
export class UnusedClass {
    private value: string;

    constructor(value: string) {
        this.value = value;
    }

    process(): void {
        console.log(`Processing: ${this.value}`);
    }
}

/**
 * Another unused class - DEAD CODE.
 */
export class AnotherUnusedClass<T> {
    private data: T;

    constructor(data: T) {
        this.data = data;
    }

    getData(): T {
        return this.data;
    }
}
