package com.example.app;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.*;
import java.util.function.*;
import java.util.stream.Collectors;

/**
 * Utils class - tests utility methods, lambdas, streams.
 */
public class Utils {

    /**
     * Process string data - called from main, should be reachable.
     */
    public static String processData(List<String> items) {
        // Using lambda - tests lambda extraction
        Function<String, String> mapper = s -> s.toUpperCase();

        return items.stream()
                .map(mapper)
                .collect(Collectors.joining(", "));
    }

    /**
     * Format output string - called from main, should be reachable.
     */
    public static String formatOutput(String data) {
        return "Result: " + data;
    }

    /**
     * Compute SHA256 hash - DEAD CODE.
     */
    public static String hashString(String s) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(s.getBytes());
            StringBuilder hexString = new StringBuilder();
            for (byte b : hash) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) hexString.append('0');
                hexString.append(hex);
            }
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }

    /**
     * Filter items by predicate - DEAD CODE.
     */
    public static <T> List<T> filterItems(List<T> items, Predicate<T> predicate) {
        return items.stream()
                .filter(predicate)
                .collect(Collectors.toList());
    }

    /**
     * Map items with transform - DEAD CODE.
     */
    public static <T, R> List<R> mapItems(List<T> items, Function<T, R> transform) {
        return items.stream()
                .map(transform)
                .collect(Collectors.toList());
    }

    /**
     * Reduce items to single value - DEAD CODE.
     */
    public static <T> T reduceItems(List<T> items, T identity, BinaryOperator<T> reducer) {
        return items.stream()
                .reduce(identity, reducer);
    }

    // ========================================================================
    // Higher-order functions - tests function parameter extraction
    // ========================================================================

    /**
     * Compose functions - DEAD CODE.
     */
    public static <T> Function<T, T> compose(Function<T, T>... functions) {
        return Arrays.stream(functions)
                .reduce(Function.identity(), Function::andThen);
    }

    /**
     * Memoize function - DEAD CODE.
     */
    public static <T, R> Function<T, R> memoize(Function<T, R> fn) {
        Map<T, R> cache = new HashMap<>();
        return t -> cache.computeIfAbsent(t, fn);
    }

    /**
     * Retry function - DEAD CODE.
     */
    public static <T> T retry(Supplier<T> fn, int attempts, long delayMs) {
        Exception lastError = null;
        for (int i = 0; i < attempts; i++) {
            try {
                return fn.get();
            } catch (Exception e) {
                lastError = e;
                if (i < attempts - 1) {
                    try {
                        Thread.sleep(delayMs);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw new RuntimeException(ie);
                    }
                }
            }
        }
        throw new RuntimeException("Failed after " + attempts + " attempts", lastError);
    }

    // ========================================================================
    // Optional utilities - DEAD CODE
    // ========================================================================

    /**
     * Safe get from map - DEAD CODE.
     */
    public static <K, V> Optional<V> safeGet(Map<K, V> map, K key) {
        return Optional.ofNullable(map.get(key));
    }

    /**
     * Get or default - DEAD CODE.
     */
    public static <K, V> V getOrDefault(Map<K, V> map, K key, V defaultValue) {
        return map.getOrDefault(key, defaultValue);
    }

    /**
     * Safe cast - DEAD CODE.
     */
    @SuppressWarnings("unchecked")
    public static <T> Optional<T> safeCast(Object obj, Class<T> clazz) {
        if (clazz.isInstance(obj)) {
            return Optional.of((T) obj);
        }
        return Optional.empty();
    }

    // ========================================================================
    // String utilities - DEAD CODE
    // ========================================================================

    /**
     * Is blank check - DEAD CODE.
     */
    public static boolean isBlank(String s) {
        return s == null || s.trim().isEmpty();
    }

    /**
     * Truncate string - DEAD CODE.
     */
    public static String truncate(String s, int maxLength) {
        if (s == null || s.length() <= maxLength) {
            return s;
        }
        return s.substring(0, maxLength) + "...";
    }

    /**
     * Join with prefix and suffix - DEAD CODE.
     */
    public static String joinWithDelimiter(
            List<String> items,
            String delimiter,
            String prefix,
            String suffix) {
        return items.stream()
                .collect(Collectors.joining(delimiter, prefix, suffix));
    }

    // ========================================================================
    // Collection utilities - DEAD CODE
    // ========================================================================

    /**
     * Partition list - DEAD CODE.
     */
    public static <T> Map<Boolean, List<T>> partition(List<T> items, Predicate<T> predicate) {
        return items.stream()
                .collect(Collectors.partitioningBy(predicate));
    }

    /**
     * Group by - DEAD CODE.
     */
    public static <T, K> Map<K, List<T>> groupBy(List<T> items, Function<T, K> keyExtractor) {
        return items.stream()
                .collect(Collectors.groupingBy(keyExtractor));
    }

    /**
     * Flatten nested list - DEAD CODE.
     */
    public static <T> List<T> flatten(List<List<T>> nestedList) {
        return nestedList.stream()
                .flatMap(List::stream)
                .collect(Collectors.toList());
    }

    /**
     * Zip two lists - DEAD CODE.
     */
    public static <A, B, C> List<C> zip(List<A> a, List<B> b, BiFunction<A, B, C> zipper) {
        int size = Math.min(a.size(), b.size());
        List<C> result = new ArrayList<>(size);
        for (int i = 0; i < size; i++) {
            result.add(zipper.apply(a.get(i), b.get(i)));
        }
        return result;
    }

    // ========================================================================
    // Private helpers - DEAD CODE
    // ========================================================================

    /**
     * Private helper - DEAD CODE.
     */
    private static int privateHelper(int x, int y) {
        return x + y;
    }

    /**
     * Another private helper - DEAD CODE.
     */
    private static void anotherPrivate() {
        // Empty
    }
}

// ============================================================================
// Cache class - DEAD CODE
// ============================================================================

/**
 * Simple cache - DEAD CODE.
 */
class Cache<K, V> {
    private final Map<K, V> data = new HashMap<>();

    public Optional<V> get(K key) {
        return Optional.ofNullable(data.get(key));
    }

    public void set(K key, V value) {
        data.put(key, value);
    }

    public boolean has(K key) {
        return data.containsKey(key);
    }

    public void delete(K key) {
        data.remove(key);
    }

    public void clear() {
        data.clear();
    }
}

// ============================================================================
// Builder class - DEAD CODE
// ============================================================================

/**
 * Query builder - DEAD CODE.
 */
class QueryBuilder {
    private String table;
    private List<String> columns = new ArrayList<>();
    private List<String> conditions = new ArrayList<>();
    private Integer limit;

    public QueryBuilder(String table) {
        this.table = table;
    }

    public QueryBuilder select(String... cols) {
        columns.addAll(Arrays.asList(cols));
        return this;
    }

    public QueryBuilder where(String condition) {
        conditions.add(condition);
        return this;
    }

    public QueryBuilder limit(int n) {
        this.limit = n;
        return this;
    }

    public String build() {
        StringBuilder sql = new StringBuilder("SELECT ");
        sql.append(columns.isEmpty() ? "*" : String.join(", ", columns));
        sql.append(" FROM ").append(table);

        if (!conditions.isEmpty()) {
            sql.append(" WHERE ").append(String.join(" AND ", conditions));
        }

        if (limit != null) {
            sql.append(" LIMIT ").append(limit);
        }

        return sql.toString();
    }
}
