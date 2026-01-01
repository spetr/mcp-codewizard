/**
 * Utils module - tests utility functions and extensions.
 * Tests: top-level functions, extensions, higher-order functions.
 */

import Foundation

// ============================================================================
// String extensions - tests extension extraction
// ============================================================================

extension String {
    /**
     * Check if string is blank - extension method.
     */
    var isBlank: Bool {
        return trimmingCharacters(in: .whitespaces).isEmpty
    }

    /**
     * Truncate string - extension method, DEAD CODE.
     */
    func truncate(_ maxLength: Int) -> String {
        if count <= maxLength { return self }
        return String(prefix(maxLength)) + "..."
    }

    /**
     * Capitalize first letter - extension method, DEAD CODE.
     */
    var capitalizedFirst: String {
        if isEmpty { return self }
        return prefix(1).uppercased() + dropFirst().lowercased()
    }
}

extension Optional where Wrapped == String {
    /**
     * Check if optional string is blank - extension method.
     */
    var isBlank: Bool {
        return self?.isBlank ?? true
    }
}

// ============================================================================
// Array extensions - tests generic extension extraction
// ============================================================================

extension Array {
    /**
     * Safe subscript - DEAD CODE.
     */
    subscript(safe index: Int) -> Element? {
        return indices.contains(index) ? self[index] : nil
    }

    /**
     * Chunk array - DEAD CODE.
     */
    func chunked(into size: Int) -> [[Element]] {
        stride(from: 0, to: count, by: size).map {
            Array(self[$0..<Swift.min($0 + size, count)])
        }
    }
}

extension Array where Element: Hashable {
    /**
     * Distinct elements - DEAD CODE.
     */
    func distinct() -> [Element] {
        var seen = Set<Element>()
        return filter { seen.insert($0).inserted }
    }
}

// ============================================================================
// Higher-order functions - tests closure extraction
// ============================================================================

/**
 * Filter strings - DEAD CODE.
 */
func filterStrings(_ items: [String], predicate: (String) -> Bool) -> [String] {
    return items.filter(predicate)
}

/**
 * Map strings - DEAD CODE.
 */
func mapStrings(_ items: [String], transform: (String) -> String) -> [String] {
    return items.map(transform)
}

/**
 * Reduce strings - DEAD CODE.
 */
func reduceStrings(_ items: [String], seed: String, accumulator: (String, String) -> String) -> String {
    return items.reduce(seed, accumulator)
}

/**
 * Group by key - DEAD CODE.
 */
func groupByKey<T, K: Hashable>(_ items: [T], keySelector: (T) -> K) -> [K: [T]] {
    return Dictionary(grouping: items, by: keySelector)
}

/**
 * Sort by comparator - DEAD CODE.
 */
func sortBy<T>(_ items: [T], comparator: (T, T) -> Bool) -> [T] {
    return items.sorted(by: comparator)
}

/**
 * Distinct by key - DEAD CODE.
 */
func distinctBy<T, K: Hashable>(_ items: [T], keySelector: (T) -> K) -> [T] {
    var seen = Set<K>()
    return items.filter { seen.insert(keySelector($0)).inserted }
}

/**
 * Take while predicate - DEAD CODE.
 */
func takeWhilePredicate<T>(_ items: [T], predicate: (T) -> Bool) -> [T] {
    var result: [T] = []
    for item in items {
        if !predicate(item) { break }
        result.append(item)
    }
    return result
}

// ============================================================================
// Utility functions - tests generic function extraction
// ============================================================================

/**
 * Hash string - DEAD CODE.
 */
func hashString(_ s: String) -> String {
    var hash: UInt64 = 5381
    for char in s.unicodeScalars {
        hash = ((hash &<< 5) &+ hash) &+ UInt64(char.value)
    }
    return String(format: "%016llx", hash)
}

/**
 * Safe cast - DEAD CODE.
 */
func safeCast<T>(_ value: Any?) -> T? {
    return value as? T
}

/**
 * Create pair - DEAD CODE.
 */
func createPair<A, B>(_ first: A, _ second: B) -> (A, B) {
    return (first, second)
}

/**
 * Measure time - DEAD CODE.
 */
func measureTime<T>(_ block: () -> T) -> (T, TimeInterval) {
    let start = Date()
    let result = block()
    let elapsed = Date().timeIntervalSince(start)
    return (result, elapsed)
}

/**
 * Try or nil - DEAD CODE.
 */
func tryOrNil<T>(_ block: () throws -> T) -> T? {
    return try? block()
}

/**
 * String helper class - DEAD CODE.
 */
class StringHelper {
    /**
     * Join strings - DEAD CODE.
     */
    static func join(_ items: [String], separator: String = ", ") -> String {
        return items.joined(separator: separator)
    }

    /**
     * Split string - DEAD CODE.
     */
    static func split(_ s: String, separator: String = ",") -> [String] {
        return s.components(separatedBy: separator).map { $0.trimmingCharacters(in: .whitespaces) }
    }

    /**
     * Repeat string - DEAD CODE.
     */
    static func `repeat`(_ s: String, count: Int) -> String {
        return String(repeating: s, count: count)
    }
}
