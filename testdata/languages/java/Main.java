package com.example.app;

import java.util.Arrays;
import java.util.List;

/**
 * Main class demonstrating various Java patterns for parser testing.
 * Tests: entry points, method calls, static fields.
 */
public class Main {

    // Static constants - tests constant extraction
    public static final int MAX_RETRIES = 3;
    public static final int DEFAULT_TIMEOUT = 30;

    // Static variables - tests variable extraction
    private static String appName = "TestApp";
    private static final String INTERNAL_VERSION = "1.0.0";

    // Static initialization - tests static reference tracking
    private static final Logger logger = createLogger("main");

    /**
     * Main entry point - should be marked as reachable.
     */
    public static void main(String[] args) {
        System.out.println("Starting " + appName);

        // Method calls - tests reference extraction
        Config config = loadConfig();
        if (!initialize(config)) {
            System.err.println("Initialization failed");
            System.exit(1);
        }

        // Method calls on objects
        Server server = createServer(config);
        server.start();

        // Using utility methods
        String result = Utils.processData(Arrays.asList("a", "b", "c"));
        System.out.println(Utils.formatOutput(result));

        // Calling transitive methods
        runPipeline();
    }

    /**
     * Load configuration - called from main, should be reachable.
     */
    private static Config loadConfig() {
        return new Config("localhost", 8080, "info");
    }

    /**
     * Initialize application - called from main, should be reachable.
     */
    private static boolean initialize(Config config) {
        if (config == null) {
            return false;
        }
        setupLogging(config.getLogLevel());
        return true;
    }

    /**
     * Internal helper - called from initialize, should be reachable.
     */
    private static void setupLogging(String level) {
        System.out.println("Setting log level to: " + level);
    }

    /**
     * Orchestrate data pipeline - tests transitive reachability.
     */
    private static void runPipeline() {
        byte[] data = fetchData();
        byte[] transformed = transformData(data);
        saveData(transformed);
    }

    /**
     * Fetch data - called by runPipeline, should be reachable.
     */
    private static byte[] fetchData() {
        return "sample data".getBytes();
    }

    /**
     * Transform data - called by runPipeline, should be reachable.
     */
    private static byte[] transformData(byte[] data) {
        byte[] prefix = "transformed: ".getBytes();
        byte[] result = new byte[prefix.length + data.length];
        System.arraycopy(prefix, 0, result, 0, prefix.length);
        System.arraycopy(data, 0, result, prefix.length, data.length);
        return result;
    }

    /**
     * Save data - called by runPipeline, should be reachable.
     */
    private static void saveData(byte[] data) {
        System.out.println("Saving: " + new String(data));
    }

    /**
     * Create server - called from main.
     */
    private static Server createServer(Config config) {
        return new Server(config);
    }

    /**
     * Create logger - called from static init.
     */
    private static Logger createLogger(String name) {
        return new Logger(name);
    }

    // ========================================================================
    // Dead code section - methods that are never called
    // ========================================================================

    /**
     * This method is never called - DEAD CODE.
     */
    private static void unusedMethod() {
        System.out.println("This is never executed");
    }

    /**
     * Also never called - DEAD CODE.
     */
    private static String anotherUnused() {
        return "dead";
    }

    /**
     * Starts a chain of dead code - DEAD CODE.
     */
    private static void deadChainStart() {
        deadChainMiddle();
    }

    /**
     * In the middle of dead chain - DEAD CODE (transitive).
     */
    private static void deadChainMiddle() {
        deadChainEnd();
    }

    /**
     * End of dead chain - DEAD CODE (transitive).
     */
    private static void deadChainEnd() {
        System.out.println("End of dead chain");
    }
}

/**
 * Simple logger class.
 */
class Logger {
    private final String name;
    private int level = 1;

    public Logger(String name) {
        this.name = name;
    }

    public void info(String message) {
        System.out.println("[INFO] " + name + ": " + message);
    }

    /**
     * Debug method - DEAD CODE.
     */
    public void debug(String message) {
        if (level >= 2) {
            System.out.println("[DEBUG] " + name + ": " + message);
        }
    }

    /**
     * Error method - DEAD CODE.
     */
    public void error(String message) {
        System.out.println("[ERROR] " + name + ": " + message);
    }
}
