package com.example.app;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

/**
 * Configuration class - tests class extraction.
 */
public class Config {
    private final String host;
    private final int port;
    private final String logLevel;
    private Map<String, Object> options;

    public Config(String host, int port, String logLevel) {
        this.host = host;
        this.port = port;
        this.logLevel = logLevel;
        this.options = new HashMap<>();
    }

    // Getters
    public String getHost() { return host; }
    public int getPort() { return port; }
    public String getLogLevel() { return logLevel; }
    public Map<String, Object> getOptions() { return options; }

    /**
     * Validate configuration.
     */
    public boolean validate() {
        if (host == null || host.isEmpty()) {
            return false;
        }
        if (port <= 0 || port > 65535) {
            return false;
        }
        return true;
    }

    /**
     * Clone the configuration.
     */
    public Config clone() {
        Config cloned = new Config(this.host, this.port, this.logLevel);
        cloned.options = new HashMap<>(this.options);
        return cloned;
    }

    /**
     * Builder pattern - tests inner class.
     */
    public static class Builder {
        private String host = "localhost";
        private int port = 8080;
        private String logLevel = "info";
        private Map<String, Object> options = new HashMap<>();

        public Builder host(String host) {
            this.host = host;
            return this;
        }

        public Builder port(int port) {
            this.port = port;
            return this;
        }

        public Builder logLevel(String logLevel) {
            this.logLevel = logLevel;
            return this;
        }

        public Builder option(String key, Object value) {
            this.options.put(key, value);
            return this;
        }

        public Config build() {
            Config config = new Config(host, port, logLevel);
            config.options = this.options;
            return config;
        }
    }
}

/**
 * Server class - tests class with methods.
 */
class Server {
    private final Config config;
    private boolean running = false;
    private final Logger logger;

    public Server(Config config) {
        this.config = config;
        this.logger = new Logger("server");
    }

    /**
     * Start the server - called from main, should be reachable.
     */
    public void start() {
        this.running = true;
        logger.info("Starting server on " + config.getHost() + ":" + config.getPort());
        listen();
    }

    /**
     * Stop the server.
     */
    public void stop() {
        this.running = false;
        logger.info("Stopping server");
    }

    /**
     * Check if running.
     */
    public boolean isRunning() {
        return running;
    }

    /**
     * Internal listen method - called by start, should be reachable.
     */
    private void listen() {
        // Simulated listening
    }

    /**
     * Handle connection - DEAD CODE.
     */
    private void handleConnection(Object connection) {
        // Handle connection
    }
}

// ============================================================================
// Interface - tests interface extraction
// ============================================================================

/**
 * Handler interface.
 */
interface Handler {
    /**
     * Handle request.
     */
    Response handle(Request request);

    /**
     * Get handler name.
     */
    String getName();
}

/**
 * Request class.
 */
class Request {
    private final String method;
    private final String path;
    private final byte[] body;

    public Request(String method, String path, byte[] body) {
        this.method = method;
        this.path = path;
        this.body = body;
    }

    public String getMethod() { return method; }
    public String getPath() { return path; }
    public byte[] getBody() { return body; }
}

/**
 * Response class.
 */
class Response {
    private final int status;
    private final byte[] body;

    public Response(int status, byte[] body) {
        this.status = status;
        this.body = body;
    }

    public int getStatus() { return status; }
    public byte[] getBody() { return body; }
}

// ============================================================================
// Abstract class - tests abstract class extraction
// ============================================================================

/**
 * Abstract base handler.
 */
abstract class BaseHandler implements Handler {
    protected final String name;

    protected BaseHandler(String name) {
        this.name = name;
    }

    @Override
    public String getName() {
        return name;
    }

    /**
     * Pre-process request.
     */
    protected Request preProcess(Request request) {
        return request;
    }
}

/**
 * Echo handler - extends BaseHandler - DEAD CODE.
 */
class EchoHandler extends BaseHandler {
    public EchoHandler() {
        super("echo");
    }

    @Override
    public Response handle(Request request) {
        return new Response(200, request.getBody());
    }
}

/**
 * JSON handler - extends BaseHandler - DEAD CODE.
 */
class JsonHandler extends BaseHandler {
    public JsonHandler() {
        super("json");
    }

    @Override
    public Response handle(Request request) {
        // Simplified JSON handling
        return new Response(200, "{\"processed\": true}".getBytes());
    }
}

// ============================================================================
// Generic class - tests generic class extraction
// ============================================================================

/**
 * Generic container class.
 */
class Container<T> {
    private final java.util.List<T> items = new java.util.ArrayList<>();

    public void add(T item) {
        items.add(item);
    }

    public T get(int index) {
        return items.get(index);
    }

    public java.util.List<T> all() {
        return new java.util.ArrayList<>(items);
    }

    public int size() {
        return items.size();
    }
}

/**
 * Generic repository interface - DEAD CODE.
 */
interface Repository<T> {
    Optional<T> findById(String id);
    java.util.List<T> findAll();
    T save(T entity);
    void delete(String id);
}

/**
 * In-memory repository - DEAD CODE.
 */
class InMemoryRepository<T> implements Repository<T> {
    private final Map<String, T> items = new HashMap<>();

    @Override
    public Optional<T> findById(String id) {
        return Optional.ofNullable(items.get(id));
    }

    @Override
    public java.util.List<T> findAll() {
        return new java.util.ArrayList<>(items.values());
    }

    @Override
    public T save(T entity) {
        // Simplified - would need ID extraction
        return entity;
    }

    @Override
    public void delete(String id) {
        items.remove(id);
    }
}

// ============================================================================
// Enum - tests enum extraction
// ============================================================================

/**
 * Log level enum.
 */
enum LogLevel {
    DEBUG(0),
    INFO(1),
    WARN(2),
    ERROR(3);

    private final int value;

    LogLevel(int value) {
        this.value = value;
    }

    public int getValue() {
        return value;
    }
}

/**
 * HTTP method enum.
 */
enum HttpMethod {
    GET, POST, PUT, DELETE, PATCH
}

// ============================================================================
// Record (Java 16+) - tests record extraction
// ============================================================================

/**
 * Point record - tests record extraction.
 */
record Point(int x, int y) {
    /**
     * Distance from origin.
     */
    public double distanceFromOrigin() {
        return Math.sqrt(x * x + y * y);
    }
}

/**
 * User record - DEAD CODE.
 */
record User(String id, String name, String email) {}

// ============================================================================
// Unused classes - DEAD CODE
// ============================================================================

/**
 * Unused class - DEAD CODE.
 */
class UnusedClass {
    private final String value;

    public UnusedClass(String value) {
        this.value = value;
    }

    public void process() {
        System.out.println("Processing: " + value);
    }
}

/**
 * Another unused class - DEAD CODE.
 */
class AnotherUnusedClass<T> {
    private T data;

    public AnotherUnusedClass(T data) {
        this.data = data;
    }

    public T getData() {
        return data;
    }
}
