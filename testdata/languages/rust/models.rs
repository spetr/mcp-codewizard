//! Models module - tests Rust struct definitions, traits, impls.

use std::collections::HashMap;
use std::fmt;

/// Configuration struct - tests struct extraction.
#[derive(Debug, Clone)]
pub struct Config {
    pub host: String,
    pub port: u16,
    pub log_level: String,
    pub options: HashMap<String, String>,
}

impl Config {
    /// Create new config - constructor pattern.
    pub fn new(host: &str, port: u16, log_level: &str) -> Self {
        Config {
            host: host.to_string(),
            port,
            log_level: log_level.to_string(),
            options: HashMap::new(),
        }
    }

    /// Validate configuration.
    pub fn validate(&self) -> bool {
        if self.host.is_empty() {
            return false;
        }
        if self.port == 0 {
            return false;
        }
        true
    }

    /// Builder pattern method - DEAD CODE.
    pub fn with_option(mut self, key: &str, value: &str) -> Self {
        self.options.insert(key.to_string(), value.to_string());
        self
    }
}

impl Default for Config {
    fn default() -> Self {
        Config::new("localhost", 8080, "info")
    }
}

/// Server struct - tests struct with methods.
pub struct Server {
    config: Config,
    running: bool,
    logger: Logger,
}

impl Server {
    /// Create new server.
    pub fn new(config: Config) -> Self {
        Server {
            config,
            running: false,
            logger: Logger::new("server"),
        }
    }

    /// Start the server - called from main, should be reachable.
    pub fn start(&self) {
        self.logger.info(&format!(
            "Starting server on {}:{}",
            self.config.host, self.config.port
        ));
        self.listen();
    }

    /// Stop the server.
    pub fn stop(&mut self) {
        self.running = false;
        self.logger.info("Stopping server");
    }

    /// Check if running.
    pub fn is_running(&self) -> bool {
        self.running
    }

    /// Internal listen method - called by start, should be reachable.
    fn listen(&self) {
        // Simulated listening
    }

    /// Handle connection - DEAD CODE.
    fn handle_connection(&self, _conn: &dyn std::io::Read) {
        // Handle connection
    }
}

/// Logger struct.
pub struct Logger {
    prefix: String,
    level: u8,
}

impl Logger {
    /// Create new logger.
    pub fn new(prefix: &str) -> Self {
        Logger {
            prefix: prefix.to_string(),
            level: 1,
        }
    }

    /// Log info message.
    pub fn info(&self, message: &str) {
        println!("[INFO] {}: {}", self.prefix, message);
    }

    /// Log debug message - DEAD CODE.
    pub fn debug(&self, message: &str) {
        if self.level >= 2 {
            println!("[DEBUG] {}: {}", self.prefix, message);
        }
    }

    /// Log error message - DEAD CODE.
    pub fn error(&self, message: &str) {
        println!("[ERROR] {}: {}", self.prefix, message);
    }
}

// ============================================================================
// Traits - tests trait extraction
// ============================================================================

/// Handler trait - tests trait with methods.
pub trait Handler {
    /// Handle request.
    fn handle(&self, request: &Request) -> Response;

    /// Get handler name.
    fn name(&self) -> &str;

    /// Pre-process request - default implementation.
    fn pre_process(&self, request: Request) -> Request {
        request
    }
}

/// Request struct.
pub struct Request {
    pub method: String,
    pub path: String,
    pub body: Vec<u8>,
}

/// Response struct.
pub struct Response {
    pub status: u16,
    pub body: Vec<u8>,
}

/// Echo handler - implements Handler - DEAD CODE.
pub struct EchoHandler {
    name: String,
}

impl EchoHandler {
    pub fn new() -> Self {
        EchoHandler {
            name: "echo".to_string(),
        }
    }
}

impl Handler for EchoHandler {
    fn handle(&self, request: &Request) -> Response {
        Response {
            status: 200,
            body: request.body.clone(),
        }
    }

    fn name(&self) -> &str {
        &self.name
    }
}

/// JSON handler - implements Handler - DEAD CODE.
pub struct JsonHandler {
    name: String,
}

impl JsonHandler {
    pub fn new() -> Self {
        JsonHandler {
            name: "json".to_string(),
        }
    }
}

impl Handler for JsonHandler {
    fn handle(&self, _request: &Request) -> Response {
        Response {
            status: 200,
            body: b"{\"processed\": true}".to_vec(),
        }
    }

    fn name(&self) -> &str {
        &self.name
    }
}

// ============================================================================
// Generic struct - tests generic struct extraction
// ============================================================================

/// Generic container - tests generic struct.
pub struct Container<T> {
    items: Vec<T>,
}

impl<T> Container<T> {
    pub fn new() -> Self {
        Container { items: Vec::new() }
    }

    pub fn add(&mut self, item: T) {
        self.items.push(item);
    }

    pub fn get(&self, index: usize) -> Option<&T> {
        self.items.get(index)
    }

    pub fn all(&self) -> &[T] {
        &self.items
    }

    pub fn len(&self) -> usize {
        self.items.len()
    }

    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }
}

impl<T: Clone> Container<T> {
    /// Map items - with Clone bound.
    pub fn map<U, F>(&self, f: F) -> Container<U>
    where
        F: Fn(&T) -> U,
    {
        Container {
            items: self.items.iter().map(f).collect(),
        }
    }
}

impl<T> Default for Container<T> {
    fn default() -> Self {
        Container::new()
    }
}

// ============================================================================
// Enum - tests enum extraction
// ============================================================================

/// Log level enum.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3,
}

/// HTTP method enum.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum HttpMethod {
    Get,
    Post,
    Put,
    Delete,
    Patch,
}

impl fmt::Display for HttpMethod {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            HttpMethod::Get => write!(f, "GET"),
            HttpMethod::Post => write!(f, "POST"),
            HttpMethod::Put => write!(f, "PUT"),
            HttpMethod::Delete => write!(f, "DELETE"),
            HttpMethod::Patch => write!(f, "PATCH"),
        }
    }
}

/// Result enum with data - tests enum with variants.
#[derive(Debug)]
pub enum ProcessResult<T, E> {
    Success(T),
    Failure(E),
    Pending,
}

impl<T, E> ProcessResult<T, E> {
    pub fn is_success(&self) -> bool {
        matches!(self, ProcessResult::Success(_))
    }

    pub fn unwrap(self) -> T
    where
        E: fmt::Debug,
    {
        match self {
            ProcessResult::Success(v) => v,
            ProcessResult::Failure(e) => panic!("called unwrap on Failure: {:?}", e),
            ProcessResult::Pending => panic!("called unwrap on Pending"),
        }
    }
}

// ============================================================================
// Type aliases - tests type alias extraction
// ============================================================================

/// Result type alias.
pub type AppResult<T> = Result<T, Box<dyn std::error::Error>>;

/// Handler function type.
pub type HandlerFn = fn(&Request) -> Response;

/// Callback type.
pub type Callback<T> = Box<dyn Fn(T) + Send + Sync>;

// ============================================================================
// Unused structs - DEAD CODE
// ============================================================================

/// Unused struct - DEAD CODE.
pub struct UnusedStruct {
    value: String,
}

impl UnusedStruct {
    pub fn new(value: &str) -> Self {
        UnusedStruct {
            value: value.to_string(),
        }
    }

    pub fn process(&self) {
        println!("Processing: {}", self.value);
    }
}

/// Another unused struct - DEAD CODE.
pub struct AnotherUnusedStruct<T> {
    data: T,
}

impl<T> AnotherUnusedStruct<T> {
    pub fn new(data: T) -> Self {
        AnotherUnusedStruct { data }
    }

    pub fn get_data(&self) -> &T {
        &self.data
    }
}

// ============================================================================
// Derive macro usage - tests derive extraction
// ============================================================================

/// Point struct with derives.
#[derive(Debug, Clone, Copy, PartialEq, Default)]
pub struct Point {
    pub x: i32,
    pub y: i32,
}

impl Point {
    pub fn new(x: i32, y: i32) -> Self {
        Point { x, y }
    }

    pub fn distance_from_origin(&self) -> f64 {
        ((self.x.pow(2) + self.y.pow(2)) as f64).sqrt()
    }
}

/// User struct with serde derives (if available).
#[derive(Debug, Clone)]
pub struct User {
    pub id: String,
    pub name: String,
    pub email: String,
}
