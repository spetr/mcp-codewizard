/**
 * Types module - tests TypeScript type definitions.
 * Tests: interfaces, types, enums, type aliases.
 */

// ============================================================================
// Interfaces - tests interface extraction
// ============================================================================

/**
 * Handler interface - tests interface with methods.
 */
export interface Handler {
    /** Handle a request. */
    handle(request: Request): Promise<Response>;
    /** Get handler name. */
    name(): string;
}

/**
 * Request interface.
 */
export interface Request {
    method: string;
    path: string;
    body?: Buffer;
    headers?: Record<string, string>;
}

/**
 * Response interface.
 */
export interface Response {
    status: number;
    body?: Buffer;
    headers?: Record<string, string>;
}

/**
 * Generic repository interface - tests generic interface.
 */
export interface Repository<T> {
    findById(id: string): Promise<T | null>;
    findAll(): Promise<T[]>;
    save(entity: T): Promise<T>;
    delete(id: string): Promise<void>;
}

/**
 * Interface with index signature.
 */
export interface Dictionary<T> {
    [key: string]: T;
}

/**
 * Interface extending another interface.
 */
export interface ExtendedHandler extends Handler {
    middleware(): Middleware[];
    validate(request: Request): boolean;
}

/**
 * Interface with optional and readonly properties.
 */
export interface ConfigOptions {
    readonly version: string;
    host?: string;
    port?: number;
    timeout?: number;
    retries?: number;
    options?: Record<string, unknown>;
}

/**
 * Interface with call signature.
 */
export interface Transformer {
    (input: string): string;
    name: string;
    description?: string;
}

/**
 * Interface with construct signature.
 */
export interface PluginConstructor {
    new (options: Record<string, unknown>): Plugin;
    version: string;
}

// ============================================================================
// Type aliases - tests type alias extraction
// ============================================================================

/**
 * Simple type alias.
 */
export type ID = string;

/**
 * Union type alias.
 */
export type StringOrNumber = string | number;

/**
 * Intersection type alias.
 */
export type HandlerWithMiddleware = Handler & { middleware: Middleware[] };

/**
 * Conditional type alias.
 */
export type Nullable<T> = T | null;

/**
 * Mapped type alias.
 */
export type Readonly<T> = { readonly [K in keyof T]: T[K] };

/**
 * Template literal type.
 */
export type EventName = `on${Capitalize<string>}`;

/**
 * Utility type usage.
 */
export type PartialConfig = Partial<ConfigOptions>;
export type RequiredConfig = Required<ConfigOptions>;
export type ConfigKeys = keyof ConfigOptions;

/**
 * Complex conditional type.
 */
export type UnwrapPromise<T> = T extends Promise<infer U> ? U : T;

/**
 * Recursive type.
 */
export type JSONValue =
    | string
    | number
    | boolean
    | null
    | JSONValue[]
    | { [key: string]: JSONValue };

// ============================================================================
// Enums - tests enum extraction
// ============================================================================

/**
 * Numeric enum.
 */
export enum LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3
}

/**
 * String enum.
 */
export enum HttpMethod {
    GET = 'GET',
    POST = 'POST',
    PUT = 'PUT',
    DELETE = 'DELETE',
    PATCH = 'PATCH'
}

/**
 * Const enum - tests const enum extraction.
 */
export const enum StatusCode {
    OK = 200,
    Created = 201,
    BadRequest = 400,
    NotFound = 404,
    InternalError = 500
}

/**
 * Heterogeneous enum.
 */
export enum MixedEnum {
    No = 0,
    Yes = 'YES',
    Maybe = 1
}

// ============================================================================
// Class types - for type checking
// ============================================================================

/**
 * Middleware type.
 */
export type Middleware = (
    req: Request,
    res: Response,
    next: () => void
) => void | Promise<void>;

/**
 * Plugin interface.
 */
export interface Plugin {
    name: string;
    version: string;
    init(): Promise<void>;
    destroy(): Promise<void>;
}

/**
 * Event handler type.
 */
export type EventHandler<T = unknown> = (event: T) => void | Promise<void>;

/**
 * Async function type.
 */
export type AsyncFunction<T = unknown, R = void> = (arg: T) => Promise<R>;

// ============================================================================
// Utility interfaces - DEAD CODE (never implemented)
// ============================================================================

/**
 * Cache interface - DEAD CODE.
 */
export interface Cache<T> {
    get(key: string): T | undefined;
    set(key: string, value: T, ttl?: number): void;
    delete(key: string): boolean;
    clear(): void;
}

/**
 * Queue interface - DEAD CODE.
 */
export interface Queue<T> {
    enqueue(item: T): void;
    dequeue(): T | undefined;
    peek(): T | undefined;
    isEmpty(): boolean;
    size(): number;
}

/**
 * Stack interface - DEAD CODE.
 */
export interface Stack<T> {
    push(item: T): void;
    pop(): T | undefined;
    peek(): T | undefined;
    isEmpty(): boolean;
    size(): number;
}

/**
 * Observer interface - DEAD CODE.
 */
export interface Observer<T> {
    next(value: T): void;
    error(error: Error): void;
    complete(): void;
}

/**
 * Observable interface - DEAD CODE.
 */
export interface Observable<T> {
    subscribe(observer: Observer<T>): Subscription;
}

/**
 * Subscription interface - DEAD CODE.
 */
export interface Subscription {
    unsubscribe(): void;
}

// ============================================================================
// Type guards and branded types
// ============================================================================

/**
 * Branded type for validated email.
 */
export type Email = string & { readonly __brand: 'email' };

/**
 * Branded type for positive integer.
 */
export type PositiveInteger = number & { readonly __brand: 'positive' };

/**
 * Branded type for UUID.
 */
export type UUID = string & { readonly __brand: 'uuid' };

/**
 * Create validated email.
 */
export function createEmail(value: string): Email {
    if (!value.includes('@')) {
        throw new Error('Invalid email');
    }
    return value as Email;
}

/**
 * Create positive integer.
 */
export function createPositiveInteger(value: number): PositiveInteger {
    if (value <= 0 || !Number.isInteger(value)) {
        throw new Error('Must be positive integer');
    }
    return value as PositiveInteger;
}

// ============================================================================
// Namespace - tests namespace extraction
// ============================================================================

/**
 * Validation namespace.
 */
export namespace Validation {
    export interface Rule {
        name: string;
        validate(value: unknown): boolean;
        message: string;
    }

    export type Result = {
        valid: boolean;
        errors: string[];
    };

    export function createRule(
        name: string,
        validate: (value: unknown) => boolean,
        message: string
    ): Rule {
        return { name, validate, message };
    }
}

/**
 * Http namespace.
 */
export namespace Http {
    export type Headers = Record<string, string>;
    export type QueryParams = Record<string, string | string[]>;

    export interface Request {
        url: string;
        method: HttpMethod;
        headers?: Headers;
        query?: QueryParams;
        body?: unknown;
    }

    export interface Response<T = unknown> {
        status: number;
        headers: Headers;
        data: T;
    }
}
