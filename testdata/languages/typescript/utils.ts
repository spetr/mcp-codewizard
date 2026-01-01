/**
 * Utils module - tests TypeScript utility functions, generics, async patterns.
 */

import * as crypto from 'crypto';

/**
 * Process string data - called from main, should be reachable.
 */
export function processData(items: string[]): string {
    const mapper = (s: string): string => s.toUpperCase();
    const results = items.map(mapper);
    return results.join(', ');
}

/**
 * Format output string - called from main, should be reachable.
 */
export function formatOutput(data: string): string {
    return `Result: ${data}`;
}

/**
 * Compute SHA256 hash - DEAD CODE.
 */
export function hashString(s: string): string {
    return crypto.createHash('sha256').update(s).digest('hex');
}

/**
 * Filter items by predicate - DEAD CODE.
 */
export function filterItems<T>(items: T[], predicate: (item: T) => boolean): T[] {
    return items.filter(predicate);
}

/**
 * Map items with transform - DEAD CODE.
 */
export function mapItems<T, U>(items: T[], transform: (item: T) => U): U[] {
    return items.map(transform);
}

/**
 * Reduce items to single value - DEAD CODE.
 */
export function reduceItems<T, U>(
    items: T[],
    reducer: (acc: U, item: T) => U,
    initial: U
): U {
    return items.reduce(reducer, initial);
}

// ============================================================================
// Higher-order functions with types - tests typed HOF extraction
// ============================================================================

/**
 * Compose functions - DEAD CODE.
 */
export function compose<T>(...fns: Array<(arg: T) => T>): (arg: T) => T {
    return (x: T) => fns.reduceRight((acc, fn) => fn(acc), x);
}

/**
 * Pipe functions - DEAD CODE.
 */
export function pipe<T>(...fns: Array<(arg: T) => T>): (arg: T) => T {
    return (x: T) => fns.reduce((acc, fn) => fn(acc), x);
}

/**
 * Curry a function - DEAD CODE.
 */
export function curry<A, B, C>(fn: (a: A, b: B) => C): (a: A) => (b: B) => C {
    return (a: A) => (b: B) => fn(a, b);
}

/**
 * Debounce function - DEAD CODE.
 */
export function debounce<T extends (...args: any[]) => any>(
    fn: T,
    delay: number
): (...args: Parameters<T>) => void {
    let timeoutId: NodeJS.Timeout;
    return function debounced(...args: Parameters<T>) {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => fn(...args), delay);
    };
}

/**
 * Throttle function - DEAD CODE.
 */
export function throttle<T extends (...args: any[]) => any>(
    fn: T,
    limit: number
): (...args: Parameters<T>) => ReturnType<T> | undefined {
    let lastCall = 0;
    return function throttled(...args: Parameters<T>): ReturnType<T> | undefined {
        const now = Date.now();
        if (now - lastCall >= limit) {
            lastCall = now;
            return fn(...args);
        }
        return undefined;
    };
}

/**
 * Memoize function - DEAD CODE.
 */
export function memoize<T extends (...args: any[]) => any>(
    fn: T
): (...args: Parameters<T>) => ReturnType<T> {
    const cache = new Map<string, ReturnType<T>>();
    return function memoized(...args: Parameters<T>): ReturnType<T> {
        const key = JSON.stringify(args);
        if (cache.has(key)) {
            return cache.get(key)!;
        }
        const result = fn(...args);
        cache.set(key, result);
        return result;
    };
}

// ============================================================================
// Async utilities with types - tests async/await extraction
// ============================================================================

/**
 * Async fetch wrapper - DEAD CODE.
 */
export async function fetchJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
    }
    return response.json() as Promise<T>;
}

/**
 * Async with retry - DEAD CODE.
 */
export async function withRetry<T>(
    fn: () => Promise<T>,
    attempts: number = 3,
    delay: number = 1000
): Promise<T> {
    let lastError: Error | undefined;
    for (let i = 0; i < attempts; i++) {
        try {
            return await fn();
        } catch (error) {
            lastError = error as Error;
            if (i < attempts - 1) {
                await sleep(delay);
            }
        }
    }
    throw lastError;
}

/**
 * Sleep utility - DEAD CODE.
 */
export function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Promise all with limit - DEAD CODE.
 */
export async function promiseAllWithLimit<T>(
    promises: Promise<T>[],
    limit: number
): Promise<T[]> {
    const results: Promise<T>[] = [];
    const executing: Promise<T>[] = [];

    for (const promise of promises) {
        const p = Promise.resolve(promise).then(result => {
            executing.splice(executing.indexOf(p), 1);
            return result;
        });
        results.push(p);
        executing.push(p);

        if (executing.length >= limit) {
            await Promise.race(executing);
        }
    }

    return Promise.all(results);
}

/**
 * Timeout wrapper - DEAD CODE.
 */
export function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
    const timeout = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('Timeout')), ms);
    });
    return Promise.race([promise, timeout]);
}

// ============================================================================
// Result type - tests union type handling
// ============================================================================

/**
 * Result type for error handling.
 */
export type Result<T, E = Error> =
    | { ok: true; value: T }
    | { ok: false; error: E };

/**
 * Create success result.
 */
export function ok<T>(value: T): Result<T, never> {
    return { ok: true, value };
}

/**
 * Create error result.
 */
export function err<E>(error: E): Result<never, E> {
    return { ok: false, error };
}

/**
 * Map result value - DEAD CODE.
 */
export function mapResult<T, U, E>(
    result: Result<T, E>,
    fn: (value: T) => U
): Result<U, E> {
    if (result.ok) {
        return ok(fn(result.value));
    }
    return result;
}

/**
 * Chain results - DEAD CODE.
 */
export function chainResult<T, U, E>(
    result: Result<T, E>,
    fn: (value: T) => Result<U, E>
): Result<U, E> {
    if (result.ok) {
        return fn(result.value);
    }
    return result;
}

// ============================================================================
// Cache class with generics - DEAD CODE
// ============================================================================

/**
 * Generic cache with TTL - DEAD CODE.
 */
export class Cache<T> {
    private data: Map<string, { value: T; expires: number }> = new Map();
    private defaultTTL: number;

    constructor(defaultTTL: number = 60000) {
        this.defaultTTL = defaultTTL;
    }

    get(key: string): T | undefined {
        const entry = this.data.get(key);
        if (!entry) {
            return undefined;
        }
        if (Date.now() > entry.expires) {
            this.data.delete(key);
            return undefined;
        }
        return entry.value;
    }

    set(key: string, value: T, ttl?: number): void {
        const expires = Date.now() + (ttl || this.defaultTTL);
        this.data.set(key, { value, expires });
    }

    has(key: string): boolean {
        return this.get(key) !== undefined;
    }

    delete(key: string): boolean {
        return this.data.delete(key);
    }

    clear(): void {
        this.data.clear();
    }
}

// ============================================================================
// Generator functions with types - tests generator extraction
// ============================================================================

/**
 * Number generator - DEAD CODE.
 */
export function* generateNumbers(n: number): Generator<number, void, unknown> {
    for (let i = 0; i < n; i++) {
        yield i;
    }
}

/**
 * Fibonacci generator - DEAD CODE.
 */
export function* generateFibonacci(n: number): Generator<number, void, unknown> {
    let a = 0, b = 1;
    for (let i = 0; i < n; i++) {
        yield a;
        [a, b] = [b, a + b];
    }
}

/**
 * Async generator - DEAD CODE.
 */
export async function* fetchAll<T>(urls: string[]): AsyncGenerator<T, void, unknown> {
    for (const url of urls) {
        yield await fetchJSON<T>(url);
    }
}

// ============================================================================
// Private helpers - DEAD CODE
// ============================================================================

/**
 * Private helper function - DEAD CODE.
 */
function _privateHelper(x: number, y: number): number {
    return x + y;
}

/**
 * Another private function - DEAD CODE.
 */
function _anotherPrivate(): void {
    // Empty
}
