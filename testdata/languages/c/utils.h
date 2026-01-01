/**
 * Utils header - tests header file extraction.
 * Tests: struct definitions, function prototypes, typedefs.
 */

#ifndef UTILS_H
#define UTILS_H

#include <stddef.h>

/* ========================================================================== */
/* Type definitions                                                            */
/* ========================================================================== */

/**
 * Configuration struct - tests struct extraction.
 */
typedef struct {
    char host[256];
    int port;
    char log_level[32];
} Config;

/**
 * Server struct - tests struct with function pointers.
 */
typedef struct Server {
    Config config;
    int running;
    void (*on_start)(struct Server* self);
    void (*on_stop)(struct Server* self);
} Server;

/**
 * Logger struct.
 */
typedef struct {
    char prefix[64];
    int level;
} Logger;

/**
 * Request struct.
 */
typedef struct {
    char method[16];
    char path[256];
    char* body;
    size_t body_len;
} Request;

/**
 * Response struct.
 */
typedef struct {
    int status;
    char* body;
    size_t body_len;
} Response;

/**
 * Handler function type.
 */
typedef Response* (*RequestHandler)(const Request* req);

/**
 * Generic container - tests void pointer pattern.
 */
typedef struct {
    void** items;
    size_t count;
    size_t capacity;
} Container;

/* ========================================================================== */
/* Enum definitions                                                            */
/* ========================================================================== */

/**
 * Log level enum.
 */
typedef enum {
    LOG_DEBUG = 0,
    LOG_INFO = 1,
    LOG_WARN = 2,
    LOG_ERROR = 3
} LogLevel;

/**
 * HTTP method enum.
 */
typedef enum {
    HTTP_GET,
    HTTP_POST,
    HTTP_PUT,
    HTTP_DELETE,
    HTTP_PATCH
} HttpMethod;

/**
 * Result status enum.
 */
typedef enum {
    RESULT_OK = 0,
    RESULT_ERROR = -1,
    RESULT_NOT_FOUND = -2,
    RESULT_TIMEOUT = -3
} ResultStatus;

/* ========================================================================== */
/* Function prototypes - reachable functions                                   */
/* ========================================================================== */

/**
 * Load configuration.
 */
void load_config(Config* config);

/**
 * Process string data.
 */
char* process_data(const char** items, size_t count);

/**
 * Format output string.
 */
char* format_output(const char* data);

/**
 * Server functions.
 */
void server_init(Server* server, const Config* config);
void server_start(Server* server);
void server_stop(Server* server);

/**
 * Logger functions.
 */
void logger_init(Logger* logger, const char* prefix);
void logger_info(const Logger* logger, const char* message);
void logger_debug(const Logger* logger, const char* message);
void logger_error(const Logger* logger, const char* message);

/* ========================================================================== */
/* Function prototypes - DEAD CODE                                             */
/* ========================================================================== */

/**
 * Hash string - DEAD CODE.
 */
char* hash_string(const char* s);

/**
 * Filter items - DEAD CODE.
 */
void** filter_items(void** items, size_t count, int (*predicate)(void*), size_t* result_count);

/**
 * Map items - DEAD CODE.
 */
void** map_items(void** items, size_t count, void* (*transform)(void*));

/**
 * Container functions - DEAD CODE.
 */
void container_init(Container* c);
void container_add(Container* c, void* item);
void* container_get(const Container* c, size_t index);
void container_free(Container* c);

/**
 * Cache functions - DEAD CODE.
 */
typedef struct Cache Cache;
Cache* cache_create(void);
void cache_destroy(Cache* cache);
void* cache_get(const Cache* cache, const char* key);
void cache_set(Cache* cache, const char* key, void* value);
void cache_delete(Cache* cache, const char* key);
void cache_clear(Cache* cache);

/* ========================================================================== */
/* Inline functions                                                            */
/* ========================================================================== */

/**
 * Max inline function.
 */
static inline int max_int(int a, int b) {
    return a > b ? a : b;
}

/**
 * Min inline function.
 */
static inline int min_int(int a, int b) {
    return a < b ? a : b;
}

/**
 * Is blank inline function.
 */
static inline int is_blank(const char* s) {
    return s == NULL || *s == '\0';
}

#endif /* UTILS_H */
