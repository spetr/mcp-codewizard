/**
 * Main module demonstrating various C patterns for parser testing.
 * Tests: entry points, function calls, preprocessor macros.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "utils.h"

/* Constants - tests constant extraction */
#define MAX_RETRIES 3
#define DEFAULT_TIMEOUT 30
#define APP_NAME "TestApp"

/* Static variables - tests static variable extraction */
static const char* internal_version = "1.0.0";
static int initialized = 0;

/* Forward declarations */
static int initialize(const Config* config);
static void setup_logging(const char* level);
static void run_pipeline(void);
static char* fetch_data(void);
static char* transform_data(const char* data);
static void save_data(const char* data);

/**
 * Main entry point - should be marked as reachable.
 */
int main(int argc, char* argv[]) {
    printf("Starting %s\n", APP_NAME);

    /* Function calls - tests reference extraction */
    Config config;
    load_config(&config);

    if (!initialize(&config)) {
        fprintf(stderr, "Initialization failed\n");
        return 1;
    }

    /* Create and start server */
    Server server;
    server_init(&server, &config);
    server_start(&server);

    /* Using utility functions */
    char* result = process_data((const char*[]){"a", "b", "c"}, 3);
    char* output = format_output(result);
    printf("%s\n", output);
    free(result);
    free(output);

    /* Calling transitive functions */
    run_pipeline();

    /* Cleanup */
    server_stop(&server);
    return 0;
}

/**
 * Initialize application - called from main, should be reachable.
 */
static int initialize(const Config* config) {
    if (config == NULL) {
        return 0;
    }
    setup_logging(config->log_level);
    initialized = 1;
    return 1;
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
static void setup_logging(const char* level) {
    printf("Setting log level to: %s\n", level);
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
static void run_pipeline(void) {
    char* data = fetch_data();
    char* transformed = transform_data(data);
    save_data(transformed);
    free(data);
    free(transformed);
}

/**
 * Fetch data - called by run_pipeline, should be reachable.
 */
static char* fetch_data(void) {
    char* data = malloc(32);
    strcpy(data, "sample data");
    return data;
}

/**
 * Transform data - called by run_pipeline, should be reachable.
 */
static char* transform_data(const char* data) {
    const char* prefix = "transformed: ";
    size_t len = strlen(prefix) + strlen(data) + 1;
    char* result = malloc(len);
    snprintf(result, len, "%s%s", prefix, data);
    return result;
}

/**
 * Save data - called by run_pipeline, should be reachable.
 */
static void save_data(const char* data) {
    printf("Saving: %s\n", data);
}

/* ========================================================================== */
/* Dead code section - functions that are never called                        */
/* ========================================================================== */

/**
 * This function is never called - DEAD CODE.
 */
static void unused_function(void) {
    printf("This is never executed\n");
}

/**
 * Also never called - DEAD CODE.
 */
static char* another_unused(void) {
    return "dead";
}

/**
 * Starts a chain of dead code - DEAD CODE.
 */
static void dead_chain_start(void) {
    dead_chain_middle();
}

/**
 * In the middle of dead chain - DEAD CODE (transitive).
 */
static void dead_chain_middle(void) {
    dead_chain_end();
}

/**
 * End of dead chain - DEAD CODE (transitive).
 */
static void dead_chain_end(void) {
    printf("End of dead chain\n");
}

/* ========================================================================== */
/* Function pointer patterns                                                   */
/* ========================================================================== */

/* Callback type - tests typedef for function pointer */
typedef void (*Callback)(void* data);

/* Handler function pointer */
typedef int (*Handler)(const char* input, char* output, size_t output_size);

/**
 * Execute callback - DEAD CODE.
 */
static void execute_callback(Callback cb, void* data) {
    if (cb != NULL) {
        cb(data);
    }
}

/**
 * Sample callback - DEAD CODE.
 */
static void sample_callback(void* data) {
    printf("Callback executed with data: %p\n", data);
}

/* ========================================================================== */
/* Preprocessor macros                                                         */
/* ========================================================================== */

/* Simple macro */
#define MAX(a, b) ((a) > (b) ? (a) : (b))

/* Macro with multiple statements */
#define SAFE_FREE(ptr) do { \
    if ((ptr) != NULL) { \
        free(ptr); \
        (ptr) = NULL; \
    } \
} while(0)

/* Conditional compilation */
#ifdef DEBUG
static void debug_print(const char* msg) {
    printf("[DEBUG] %s\n", msg);
}
#endif

/* Variadic macro */
#define LOG(fmt, ...) printf("[LOG] " fmt "\n", ##__VA_ARGS__)
