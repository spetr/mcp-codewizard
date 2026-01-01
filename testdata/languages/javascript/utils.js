/**
 * Utils module - tests utility functions, closures, async patterns.
 */

const crypto = require('crypto');

/**
 * Process string data - called from main, should be reachable.
 * @param {string[]} items
 * @returns {string}
 */
function processData(items) {
    // Using arrow function - tests closure/arrow extraction
    const mapper = (s) => s.toUpperCase();

    const results = items.map(mapper);
    return results.join(', ');
}

/**
 * Format output string - called from main, should be reachable.
 * @param {string} data
 * @returns {string}
 */
function formatOutput(data) {
    return `Result: ${data}`;
}

/**
 * Compute SHA256 hash - DEAD CODE.
 * @param {string} s
 * @returns {string}
 */
function hashString(s) {
    return crypto.createHash('sha256').update(s).digest('hex');
}

/**
 * Filter items by predicate - DEAD CODE.
 * @param {Array} items
 * @param {Function} predicate
 * @returns {Array}
 */
function filterItems(items, predicate) {
    return items.filter(predicate);
}

/**
 * Map items with transform - DEAD CODE.
 * @param {Array} items
 * @param {Function} transform
 * @returns {Array}
 */
function mapItems(items, transform) {
    return items.map(transform);
}

/**
 * Reduce items to single value - DEAD CODE.
 * @param {Array} items
 * @param {Function} reducer
 * @param {*} initial
 * @returns {*}
 */
function reduceItems(items, reducer, initial) {
    return items.reduce(reducer, initial);
}

// ============================================================================
// Higher-order functions - tests function parameter extraction
// ============================================================================

/**
 * Compose functions - DEAD CODE.
 * @param  {...Function} fns
 * @returns {Function}
 */
function compose(...fns) {
    return (x) => fns.reduceRight((acc, fn) => fn(acc), x);
}

/**
 * Pipe functions - DEAD CODE.
 * @param  {...Function} fns
 * @returns {Function}
 */
function pipe(...fns) {
    return (x) => fns.reduce((acc, fn) => fn(acc), x);
}

/**
 * Curry a function - DEAD CODE.
 * @param {Function} fn
 * @returns {Function}
 */
function curry(fn) {
    return function curried(...args) {
        if (args.length >= fn.length) {
            return fn.apply(this, args);
        }
        return (...moreArgs) => curried.apply(this, args.concat(moreArgs));
    };
}

/**
 * Debounce function - DEAD CODE.
 * @param {Function} fn
 * @param {number} delay
 * @returns {Function}
 */
function debounce(fn, delay) {
    let timeoutId;
    return function debounced(...args) {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => fn.apply(this, args), delay);
    };
}

/**
 * Throttle function - DEAD CODE.
 * @param {Function} fn
 * @param {number} limit
 * @returns {Function}
 */
function throttle(fn, limit) {
    let lastCall = 0;
    return function throttled(...args) {
        const now = Date.now();
        if (now - lastCall >= limit) {
            lastCall = now;
            return fn.apply(this, args);
        }
    };
}

/**
 * Memoize function - DEAD CODE.
 * @param {Function} fn
 * @returns {Function}
 */
function memoize(fn) {
    const cache = new Map();
    return function memoized(...args) {
        const key = JSON.stringify(args);
        if (cache.has(key)) {
            return cache.get(key);
        }
        const result = fn.apply(this, args);
        cache.set(key, result);
        return result;
    };
}

// ============================================================================
// Async utilities - tests async/await extraction
// ============================================================================

/**
 * Async fetch wrapper - DEAD CODE.
 * @param {string} url
 * @returns {Promise<Object>}
 */
async function fetchJSON(url) {
    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
    }
    return response.json();
}

/**
 * Async with retry - DEAD CODE.
 * @param {Function} fn
 * @param {number} attempts
 * @param {number} delay
 * @returns {Promise<*>}
 */
async function withRetry(fn, attempts = 3, delay = 1000) {
    let lastError;
    for (let i = 0; i < attempts; i++) {
        try {
            return await fn();
        } catch (error) {
            lastError = error;
            if (i < attempts - 1) {
                await sleep(delay);
            }
        }
    }
    throw lastError;
}

/**
 * Sleep utility - DEAD CODE.
 * @param {number} ms
 * @returns {Promise<void>}
 */
function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Promise all with limit - DEAD CODE.
 * @param {Array<Promise>} promises
 * @param {number} limit
 * @returns {Promise<Array>}
 */
async function promiseAllWithLimit(promises, limit) {
    const results = [];
    const executing = [];

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
 * @param {Promise} promise
 * @param {number} ms
 * @returns {Promise}
 */
function withTimeout(promise, ms) {
    const timeout = new Promise((_, reject) => {
        setTimeout(() => reject(new Error('Timeout')), ms);
    });
    return Promise.race([promise, timeout]);
}

// ============================================================================
// Cache class - DEAD CODE
// ============================================================================

/**
 * Simple cache - DEAD CODE.
 */
class Cache {
    constructor() {
        this.data = new Map();
    }

    get(key) {
        return this.data.get(key);
    }

    set(key, value) {
        this.data.set(key, value);
    }

    has(key) {
        return this.data.has(key);
    }

    delete(key) {
        this.data.delete(key);
    }

    clear() {
        this.data.clear();
    }
}

/**
 * LRU Cache - DEAD CODE.
 */
class LRUCache {
    constructor(capacity) {
        this.capacity = capacity;
        this.cache = new Map();
    }

    get(key) {
        if (!this.cache.has(key)) {
            return undefined;
        }
        const value = this.cache.get(key);
        this.cache.delete(key);
        this.cache.set(key, value);
        return value;
    }

    set(key, value) {
        if (this.cache.has(key)) {
            this.cache.delete(key);
        } else if (this.cache.size >= this.capacity) {
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        this.cache.set(key, value);
    }
}

// ============================================================================
// Generator functions - tests generator extraction
// ============================================================================

/**
 * Number generator - DEAD CODE.
 * @param {number} n
 */
function* generateNumbers(n) {
    for (let i = 0; i < n; i++) {
        yield i;
    }
}

/**
 * Fibonacci generator - DEAD CODE.
 * @param {number} n
 */
function* generateFibonacci(n) {
    let a = 0, b = 1;
    for (let i = 0; i < n; i++) {
        yield a;
        [a, b] = [b, a + b];
    }
}

/**
 * Async generator - DEAD CODE.
 * @param {string[]} urls
 */
async function* fetchAll(urls) {
    for (const url of urls) {
        yield await fetchJSON(url);
    }
}

// ============================================================================
// Private helpers - DEAD CODE
// ============================================================================

/**
 * Private helper function - DEAD CODE.
 * @param {number} x
 * @param {number} y
 * @returns {number}
 */
function _privateHelper(x, y) {
    return x + y;
}

/**
 * Another private function - DEAD CODE.
 */
function _anotherPrivate() {
    // Empty
}

// ============================================================================
// Exports
// ============================================================================

module.exports = {
    processData,
    formatOutput,
    hashString,
    filterItems,
    mapItems,
    reduceItems,
    compose,
    pipe,
    curry,
    debounce,
    throttle,
    memoize,
    fetchJSON,
    withRetry,
    sleep,
    promiseAllWithLimit,
    withTimeout,
    Cache,
    LRUCache,
    generateNumbers,
    generateFibonacci
};
