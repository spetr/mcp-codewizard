//! Main module demonstrating various Rust patterns for parser testing.
//! Tests: entry points, function calls, modules, traits.

mod utils;
mod models;

use models::{Config, Server, Logger};
use utils::{process_data, format_output};

/// Maximum retries constant - tests constant extraction.
const MAX_RETRIES: u32 = 3;

/// Default timeout constant.
const DEFAULT_TIMEOUT: u32 = 30;

/// Application name - tests static variable extraction.
static APP_NAME: &str = "TestApp";

/// Internal version (private).
static INTERNAL_VERSION: &str = "1.0.0";

/// Logger instance - tests lazy_static pattern.
lazy_static::lazy_static! {
    static ref LOGGER: Logger = create_logger("main");
}

/// Main entry point - should be marked as reachable.
fn main() {
    println!("Starting {}", APP_NAME);

    // Function calls - tests reference extraction
    let config = load_config();
    if !initialize(&config) {
        eprintln!("Initialization failed");
        std::process::exit(1);
    }

    // Method calls
    let server = Server::new(config.clone());
    server.start();

    // Using utility functions
    let result = process_data(&["a", "b", "c"]);
    println!("{}", format_output(&result));

    // Calling transitive functions
    run_pipeline();
}

/// Load configuration - called from main, should be reachable.
fn load_config() -> Config {
    Config::new("localhost", 8080, "info")
}

/// Initialize application - called from main, should be reachable.
fn initialize(config: &Config) -> bool {
    setup_logging(&config.log_level);
    true
}

/// Internal helper - called from initialize, should be reachable.
fn setup_logging(level: &str) {
    println!("Setting log level to: {}", level);
}

/// Orchestrate data pipeline - tests transitive reachability.
fn run_pipeline() {
    let data = fetch_data();
    let transformed = transform_data(&data);
    save_data(&transformed);
}

/// Fetch data - called by run_pipeline, should be reachable.
fn fetch_data() -> Vec<u8> {
    b"sample data".to_vec()
}

/// Transform data - called by run_pipeline, should be reachable.
fn transform_data(data: &[u8]) -> Vec<u8> {
    let mut result = b"transformed: ".to_vec();
    result.extend_from_slice(data);
    result
}

/// Save data - called by run_pipeline, should be reachable.
fn save_data(data: &[u8]) {
    println!("Saving: {}", String::from_utf8_lossy(data));
}

/// Create logger - called from static init.
fn create_logger(name: &str) -> Logger {
    Logger::new(name)
}

// ============================================================================
// Generic functions - tests Rust generics extraction
// ============================================================================

/// Generic identity function - tests type parameter extraction.
fn identity<T>(value: T) -> T {
    value
}

/// Generic with trait bound - tests where clause extraction.
fn print_debug<T: std::fmt::Debug>(value: &T) {
    println!("{:?}", value);
}

/// Generic with multiple bounds - DEAD CODE.
fn process_item<T>(item: T) -> String
where
    T: std::fmt::Display + Clone,
{
    format!("{}", item)
}

/// Generic with lifetime - DEAD CODE.
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}

// ============================================================================
// Dead code section - functions that are never called
// ============================================================================

/// This function is never called - DEAD CODE.
fn unused_function() {
    println!("This is never executed");
}

/// Also never called - DEAD CODE.
fn another_unused() -> String {
    "dead".to_string()
}

/// Starts a chain of dead code - DEAD CODE.
fn dead_chain_start() {
    dead_chain_middle();
}

/// In the middle of dead chain - DEAD CODE (transitive).
fn dead_chain_middle() {
    dead_chain_end();
}

/// End of dead chain - DEAD CODE (transitive).
fn dead_chain_end() {
    println!("End of dead chain");
}

/// Private unused function - DEAD CODE.
fn _private_unused() {
    // Empty
}

// ============================================================================
// Async functions - tests async extraction
// ============================================================================

/// Async fetch function - DEAD CODE.
async fn async_fetch(url: &str) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    // Simulated async operation
    Ok(b"response".to_vec())
}

/// Async process function - DEAD CODE.
async fn async_process(data: Vec<u8>) -> String {
    String::from_utf8_lossy(&data).to_string()
}

/// Async pipeline - DEAD CODE.
async fn async_pipeline(url: &str) -> Result<String, Box<dyn std::error::Error>> {
    let data = async_fetch(url).await?;
    Ok(async_process(data).await)
}

// ============================================================================
// Closures and function pointers - tests closure extraction
// ============================================================================

/// Function returning closure - DEAD CODE.
fn make_adder(x: i32) -> impl Fn(i32) -> i32 {
    move |y| x + y
}

/// Function accepting closure - DEAD CODE.
fn apply_twice<F>(f: F, x: i32) -> i32
where
    F: Fn(i32) -> i32,
{
    f(f(x))
}

/// Function pointer type - DEAD CODE.
type Operation = fn(i32, i32) -> i32;

/// Using function pointer - DEAD CODE.
fn calculate(op: Operation, a: i32, b: i32) -> i32 {
    op(a, b)
}

// ============================================================================
// Tests module - should be excluded
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_load_config() {
        let config = load_config();
        assert_eq!(config.host, "localhost");
        assert_eq!(config.port, 8080);
    }

    #[test]
    fn test_initialize() {
        let config = load_config();
        assert!(initialize(&config));
    }

    #[test]
    fn test_fetch_data() {
        let data = fetch_data();
        assert!(!data.is_empty());
    }

    #[test]
    fn test_transform_data() {
        let data = b"test";
        let result = transform_data(data);
        assert!(result.starts_with(b"transformed: "));
    }

    #[test]
    fn test_identity() {
        assert_eq!(identity(42), 42);
        assert_eq!(identity("hello"), "hello");
    }

    #[test]
    #[should_panic]
    fn test_panic() {
        panic!("Expected panic");
    }

    #[test]
    #[ignore]
    fn test_ignored() {
        // This test is ignored
    }

    /// Test helper - should be excluded.
    fn setup_test() -> Config {
        Config::new("test", 9999, "debug")
    }

    /// Another test helper.
    fn assert_config_valid(config: &Config) {
        assert!(!config.host.is_empty());
        assert!(config.port > 0);
    }
}
