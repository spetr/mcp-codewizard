//! Utils module - tests Rust utility functions, iterators, Result handling.

use std::collections::HashMap;
use std::hash::Hash;

/// Process string data - called from main, should be reachable.
pub fn process_data(items: &[&str]) -> String {
    items
        .iter()
        .map(|s| s.to_uppercase())
        .collect::<Vec<_>>()
        .join(", ")
}

/// Format output string - called from main, should be reachable.
pub fn format_output(data: &str) -> String {
    format!("Result: {}", data)
}

/// Compute SHA256 hash - DEAD CODE.
pub fn hash_string(s: &str) -> String {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::Hasher;

    let mut hasher = DefaultHasher::new();
    hasher.write(s.as_bytes());
    format!("{:x}", hasher.finish())
}

/// Filter items by predicate - DEAD CODE.
pub fn filter_items<T, F>(items: &[T], predicate: F) -> Vec<&T>
where
    F: Fn(&T) -> bool,
{
    items.iter().filter(|item| predicate(item)).collect()
}

/// Map items with transform - DEAD CODE.
pub fn map_items<T, U, F>(items: &[T], transform: F) -> Vec<U>
where
    F: Fn(&T) -> U,
{
    items.iter().map(transform).collect()
}

/// Reduce items to single value - DEAD CODE.
pub fn reduce_items<T, U, F>(items: &[T], initial: U, reducer: F) -> U
where
    F: Fn(U, &T) -> U,
{
    items.iter().fold(initial, reducer)
}

// ============================================================================
// Higher-order functions - tests function parameter extraction
// ============================================================================

/// Compose functions - DEAD CODE.
pub fn compose<T, F, G>(f: F, g: G) -> impl Fn(T) -> T
where
    F: Fn(T) -> T,
    G: Fn(T) -> T,
{
    move |x| g(f(x))
}

/// Pipe functions - DEAD CODE.
pub fn pipe<T, F, G>(f: F, g: G) -> impl Fn(T) -> T
where
    F: Fn(T) -> T,
    G: Fn(T) -> T,
{
    move |x| g(f(x))
}

/// Memoize function - DEAD CODE.
pub fn memoize<K, V, F>(f: F) -> impl FnMut(K) -> V
where
    K: Eq + Hash + Clone,
    V: Clone,
    F: Fn(K) -> V,
{
    let mut cache: HashMap<K, V> = HashMap::new();
    move |key: K| {
        if let Some(value) = cache.get(&key) {
            return value.clone();
        }
        let value = f(key.clone());
        cache.insert(key, value.clone());
        value
    }
}

// ============================================================================
// Result utilities - tests Result/Option handling
// ============================================================================

/// Safe division - DEAD CODE.
pub fn safe_divide(a: i32, b: i32) -> Option<i32> {
    if b == 0 {
        None
    } else {
        Some(a / b)
    }
}

/// Parse or default - DEAD CODE.
pub fn parse_or_default<T: std::str::FromStr + Default>(s: &str) -> T {
    s.parse().unwrap_or_default()
}

/// Try multiple parsers - DEAD CODE.
pub fn try_parse<T, E>(s: &str, parsers: &[fn(&str) -> Result<T, E>]) -> Option<T> {
    for parser in parsers {
        if let Ok(value) = parser(s) {
            return Some(value);
        }
    }
    None
}

/// Chain results - DEAD CODE.
pub fn chain_results<T, E, F>(results: Vec<Result<T, E>>, combiner: F) -> Result<Vec<T>, E>
where
    F: Fn(Vec<T>) -> Vec<T>,
{
    let collected: Result<Vec<T>, E> = results.into_iter().collect();
    collected.map(combiner)
}

// ============================================================================
// Iterator utilities - tests iterator extraction
// ============================================================================

/// Chunk iterator - DEAD CODE.
pub fn chunks<T>(items: Vec<T>, size: usize) -> Vec<Vec<T>> {
    items
        .into_iter()
        .collect::<Vec<_>>()
        .chunks(size)
        .map(|chunk| chunk.to_vec())
        .collect()
}

/// Zip with index - DEAD CODE.
pub fn enumerate_items<T>(items: &[T]) -> Vec<(usize, &T)> {
    items.iter().enumerate().collect()
}

/// Flatten nested - DEAD CODE.
pub fn flatten<T: Clone>(nested: &[Vec<T>]) -> Vec<T> {
    nested.iter().flatten().cloned().collect()
}

/// Unique items - DEAD CODE.
pub fn unique<T: Eq + Hash + Clone>(items: &[T]) -> Vec<T> {
    let mut seen = std::collections::HashSet::new();
    items
        .iter()
        .filter(|item| seen.insert((*item).clone()))
        .cloned()
        .collect()
}

/// Group by key - DEAD CODE.
pub fn group_by<T, K, F>(items: Vec<T>, key_fn: F) -> HashMap<K, Vec<T>>
where
    K: Eq + Hash,
    F: Fn(&T) -> K,
{
    let mut groups: HashMap<K, Vec<T>> = HashMap::new();
    for item in items {
        let key = key_fn(&item);
        groups.entry(key).or_default().push(item);
    }
    groups
}

// ============================================================================
// String utilities - DEAD CODE
// ============================================================================

/// Is blank check - DEAD CODE.
pub fn is_blank(s: &str) -> bool {
    s.trim().is_empty()
}

/// Truncate string - DEAD CODE.
pub fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len])
    }
}

/// Split and trim - DEAD CODE.
pub fn split_and_trim(s: &str, delimiter: char) -> Vec<String> {
    s.split(delimiter)
        .map(|part| part.trim().to_string())
        .filter(|part| !part.is_empty())
        .collect()
}

// ============================================================================
// Cache struct - DEAD CODE
// ============================================================================

/// Simple cache - DEAD CODE.
pub struct Cache<K, V> {
    data: HashMap<K, V>,
}

impl<K: Eq + Hash, V> Cache<K, V> {
    pub fn new() -> Self {
        Cache {
            data: HashMap::new(),
        }
    }

    pub fn get(&self, key: &K) -> Option<&V> {
        self.data.get(key)
    }

    pub fn set(&mut self, key: K, value: V) {
        self.data.insert(key, value);
    }

    pub fn has(&self, key: &K) -> bool {
        self.data.contains_key(key)
    }

    pub fn remove(&mut self, key: &K) -> Option<V> {
        self.data.remove(key)
    }

    pub fn clear(&mut self) {
        self.data.clear();
    }
}

impl<K: Eq + Hash, V> Default for Cache<K, V> {
    fn default() -> Self {
        Cache::new()
    }
}

// ============================================================================
// Builder pattern - DEAD CODE
// ============================================================================

/// Query builder - DEAD CODE.
pub struct QueryBuilder {
    table: String,
    columns: Vec<String>,
    conditions: Vec<String>,
    limit: Option<usize>,
}

impl QueryBuilder {
    pub fn new(table: &str) -> Self {
        QueryBuilder {
            table: table.to_string(),
            columns: Vec::new(),
            conditions: Vec::new(),
            limit: None,
        }
    }

    pub fn select(mut self, columns: &[&str]) -> Self {
        self.columns.extend(columns.iter().map(|s| s.to_string()));
        self
    }

    pub fn where_clause(mut self, condition: &str) -> Self {
        self.conditions.push(condition.to_string());
        self
    }

    pub fn limit(mut self, n: usize) -> Self {
        self.limit = Some(n);
        self
    }

    pub fn build(&self) -> String {
        let mut sql = String::from("SELECT ");

        if self.columns.is_empty() {
            sql.push('*');
        } else {
            sql.push_str(&self.columns.join(", "));
        }

        sql.push_str(" FROM ");
        sql.push_str(&self.table);

        if !self.conditions.is_empty() {
            sql.push_str(" WHERE ");
            sql.push_str(&self.conditions.join(" AND "));
        }

        if let Some(limit) = self.limit {
            sql.push_str(&format!(" LIMIT {}", limit));
        }

        sql
    }
}

// ============================================================================
// Private helpers - DEAD CODE
// ============================================================================

/// Private helper - DEAD CODE.
fn _private_helper(x: i32, y: i32) -> i32 {
    x + y
}

/// Another private helper - DEAD CODE.
fn _another_private() {
    // Empty
}
