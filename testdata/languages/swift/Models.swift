/**
 * Models module - tests classes, structs, protocols, and enums.
 * Tests: struct extraction, protocol extraction, enum extraction.
 */

import Foundation

/**
 * Log level enum.
 */
enum LogLevel: Int {
    case debug = 0
    case info = 1
    case warn = 2
    case error = 3
}

/**
 * HTTP method enum.
 */
enum HttpMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
    case patch = "PATCH"
}

/**
 * Configuration struct - tests struct extraction.
 */
struct Config {
    var host: String = "localhost"
    var port: Int = 8080
    var logLevel: String = "info"
    var options: [String: String] = [:]

    func validate() -> Bool {
        if host.isEmpty { return false }
        if port <= 0 || port > 65535 { return false }
        return true
    }

    func clone() -> Config {
        return Config(host: host, port: port, logLevel: logLevel, options: options)
    }
}

/**
 * Handler protocol - tests protocol extraction.
 */
protocol Handler {
    func handle(input: String) -> String
    var name: String { get }
}

/**
 * Logger class.
 */
class Logger {
    private let prefix: String
    var level: LogLevel = .info

    init(prefix: String) {
        self.prefix = prefix
    }

    func info(_ message: String) {
        print("[INFO] \(prefix): \(message)")
    }

    // DEAD CODE
    func debug(_ message: String) {
        if level.rawValue <= LogLevel.debug.rawValue {
            print("[DEBUG] \(prefix): \(message)")
        }
    }

    // DEAD CODE
    func error(_ message: String) {
        print("[ERROR] \(prefix): \(message)")
    }
}

/**
 * Server class - tests class with methods.
 */
class Server {
    private let config: Config
    private var running = false
    private let logger = Logger(prefix: "server")

    init(config: Config) {
        self.config = config
    }

    private func listen() {
        // Simulated listening
    }

    // DEAD CODE
    private func handleConnection(_ connection: Any?) {
        // Handle connection
    }

    func start() {
        running = true
        logger.info("Starting server on \(config.host):\(config.port)")
        listen()
    }

    func stop() {
        running = false
        logger.info("Stopping server")
    }

    func isRunning() -> Bool {
        return running
    }
}

/**
 * Echo handler - DEAD CODE.
 */
class EchoHandler: Handler {
    func handle(input: String) -> String {
        return input
    }

    var name: String { "echo" }
}

/**
 * Upper handler - DEAD CODE.
 */
class UpperHandler: Handler {
    func handle(input: String) -> String {
        return input.uppercased()
    }

    var name: String { "upper" }
}

/**
 * Generic container class.
 */
class Container<T> {
    private var items: [T] = []

    func add(_ item: T) {
        items.append(item)
    }

    func get(_ index: Int) -> T {
        return items[index]
    }

    func all() -> [T] {
        return items
    }

    var count: Int { items.count }

    func map<U>(_ transform: (T) -> U) -> Container<U> {
        let result = Container<U>()
        for item in items {
            result.add(transform(item))
        }
        return result
    }
}

/**
 * Pair struct - DEAD CODE.
 */
struct Pair<First, Second> {
    var first: First
    var second: Second
}

/**
 * Result enum - tests associated values.
 */
enum Result<T> {
    case success(T)
    case failure(Error)
    case loading
}

/**
 * Cache class - DEAD CODE.
 */
class Cache {
    private var data: [String: Any] = [:]

    func set(_ key: String, value: Any) {
        data[key] = value
    }

    func get<T>(_ key: String) -> T? {
        return data[key] as? T
    }

    func delete(_ key: String) {
        data.removeValue(forKey: key)
    }

    func clear() {
        data.removeAll()
    }
}

/**
 * App state singleton - tests static properties.
 */
class AppState {
    static let shared = AppState()
    var isInitialized = false
    let version = "1.0.0"

    private init() {}

    func reset() {
        isInitialized = false
    }
}
