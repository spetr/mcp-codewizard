/**
 * Utils module - tests utility functions and extension functions.
 * Tests: top-level functions, extension functions, higher-order functions.
 */

package app

// ============================================================================
// Extension functions - tests extension function extraction
// ============================================================================

/**
 * Check if string is blank - extension function.
 */
fun String?.isBlank(): Boolean = this == null || this.trim().isEmpty()

/**
 * Truncate string - extension function, DEAD CODE.
 */
@Suppress("unused")
fun String.truncate(maxLength: Int): String =
    if (length <= maxLength) this else "${take(maxLength)}..."

/**
 * Capitalize string - extension function, DEAD CODE.
 */
@Suppress("unused")
fun String.capitalizeFirst(): String =
    if (isEmpty()) this else this[0].uppercaseChar() + substring(1).lowercase()

// ============================================================================
// Higher-order functions - tests lambda and function type extraction
// ============================================================================

/**
 * Filter strings - DEAD CODE.
 */
@Suppress("unused")
fun filterStrings(items: List<String>, predicate: (String) -> Boolean): List<String> =
    items.filter(predicate)

/**
 * Map strings - DEAD CODE.
 */
@Suppress("unused")
fun mapStrings(items: List<String>, transform: (String) -> String): List<String> =
    items.map(transform)

/**
 * Reduce strings - DEAD CODE.
 */
@Suppress("unused")
fun reduceStrings(items: List<String>, seed: String, accumulator: (String, String) -> String): String =
    items.fold(seed, accumulator)

/**
 * Chunk list - DEAD CODE.
 */
@Suppress("unused")
fun <T> chunkList(items: List<T>, size: Int): List<List<T>> =
    items.chunked(size)

/**
 * Flatten nested lists - DEAD CODE.
 */
@Suppress("unused")
fun <T> flattenList(items: List<List<T>>): List<T> =
    items.flatten()

/**
 * Group by key - DEAD CODE.
 */
@Suppress("unused")
fun <T, K> groupByKey(items: List<T>, keySelector: (T) -> K): Map<K, List<T>> =
    items.groupBy(keySelector)

/**
 * Sort by comparator - DEAD CODE.
 */
@Suppress("unused")
fun <T> sortBy(items: List<T>, comparator: Comparator<T>): List<T> =
    items.sortedWith(comparator)

/**
 * Distinct by key - DEAD CODE.
 */
@Suppress("unused")
fun <T, K> distinctBy(items: List<T>, keySelector: (T) -> K): List<T> =
    items.distinctBy(keySelector)

/**
 * Take while predicate - DEAD CODE.
 */
@Suppress("unused")
fun <T> takeWhilePredicate(items: List<T>, predicate: (T) -> Boolean): List<T> =
    items.takeWhile(predicate)

// ============================================================================
// Inline functions - tests inline function extraction
// ============================================================================

/**
 * Inline measure time - DEAD CODE.
 */
@Suppress("unused")
inline fun <T> measureTimeMillis(block: () -> T): kotlin.Pair<T, Long> {
    val start = System.currentTimeMillis()
    val result = block()
    val end = System.currentTimeMillis()
    return result to (end - start)
}

/**
 * Inline try or null - DEAD CODE.
 */
@Suppress("unused")
inline fun <T> tryOrNull(block: () -> T): T? = try {
    block()
} catch (e: Exception) {
    null
}

/**
 * Inline run if not null - DEAD CODE.
 */
@Suppress("unused")
inline fun <T, R> T?.runIfNotNull(block: T.() -> R): R? = this?.run(block)

// ============================================================================
// Utility functions with generics - tests generic function extraction
// ============================================================================

/**
 * Hash string - DEAD CODE.
 */
@Suppress("unused")
fun hashString(s: String): String {
    var hash = 5381L
    for (c in s) {
        hash = ((hash shl 5) + hash) + c.code
    }
    return hash.toString(16).padStart(16, '0')
}

/**
 * Safe cast - DEAD CODE.
 */
@Suppress("unused")
inline fun <reified T> safeCast(value: Any?): T? = value as? T

/**
 * Create pair - DEAD CODE.
 */
@Suppress("unused")
fun <A, B> createPair(first: A, second: B): kotlin.Pair<A, B> = first to second

/**
 * Swap pair - DEAD CODE.
 */
@Suppress("unused")
fun <A, B> kotlin.Pair<A, B>.swap(): kotlin.Pair<B, A> = second to first
