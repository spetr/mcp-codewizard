/**
 * Utils implementation - tests function implementations.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "utils.h"

/* ========================================================================== */
/* Configuration functions                                                     */
/* ========================================================================== */

/**
 * Load configuration - called from main, should be reachable.
 */
void load_config(Config* config) {
    strncpy(config->host, "localhost", sizeof(config->host) - 1);
    config->port = 8080;
    strncpy(config->log_level, "info", sizeof(config->log_level) - 1);
}

/* ========================================================================== */
/* Data processing functions                                                   */
/* ========================================================================== */

/**
 * Process string data - called from main, should be reachable.
 */
char* process_data(const char** items, size_t count) {
    if (items == NULL || count == 0) {
        char* result = malloc(1);
        result[0] = '\0';
        return result;
    }

    /* Calculate total length */
    size_t total_len = 0;
    for (size_t i = 0; i < count; i++) {
        total_len += strlen(items[i]);
    }
    total_len += (count - 1) * 2; /* ", " between items */
    total_len += 1; /* null terminator */

    /* Build result */
    char* result = malloc(total_len);
    result[0] = '\0';

    for (size_t i = 0; i < count; i++) {
        /* Convert to uppercase */
        for (const char* p = items[i]; *p; p++) {
            char c = *p;
            if (c >= 'a' && c <= 'z') {
                c -= 32;
            }
            char tmp[2] = {c, '\0'};
            strcat(result, tmp);
        }
        if (i < count - 1) {
            strcat(result, ", ");
        }
    }

    return result;
}

/**
 * Format output string - called from main, should be reachable.
 */
char* format_output(const char* data) {
    const char* prefix = "Result: ";
    size_t len = strlen(prefix) + strlen(data) + 1;
    char* result = malloc(len);
    snprintf(result, len, "%s%s", prefix, data);
    return result;
}

/* ========================================================================== */
/* Server functions                                                            */
/* ========================================================================== */

static Logger server_logger;

/**
 * Server init - called from main, should be reachable.
 */
void server_init(Server* server, const Config* config) {
    memcpy(&server->config, config, sizeof(Config));
    server->running = 0;
    server->on_start = NULL;
    server->on_stop = NULL;
    logger_init(&server_logger, "server");
}

/**
 * Server start - called from main, should be reachable.
 */
void server_start(Server* server) {
    server->running = 1;
    char msg[128];
    snprintf(msg, sizeof(msg), "Starting server on %s:%d",
             server->config.host, server->config.port);
    logger_info(&server_logger, msg);

    if (server->on_start != NULL) {
        server->on_start(server);
    }
}

/**
 * Server stop - called from main, should be reachable.
 */
void server_stop(Server* server) {
    server->running = 0;
    logger_info(&server_logger, "Stopping server");

    if (server->on_stop != NULL) {
        server->on_stop(server);
    }
}

/* ========================================================================== */
/* Logger functions                                                            */
/* ========================================================================== */

/**
 * Logger init.
 */
void logger_init(Logger* logger, const char* prefix) {
    strncpy(logger->prefix, prefix, sizeof(logger->prefix) - 1);
    logger->level = LOG_INFO;
}

/**
 * Logger info - called from server functions, should be reachable.
 */
void logger_info(const Logger* logger, const char* message) {
    printf("[INFO] %s: %s\n", logger->prefix, message);
}

/**
 * Logger debug - DEAD CODE.
 */
void logger_debug(const Logger* logger, const char* message) {
    if (logger->level >= LOG_DEBUG) {
        printf("[DEBUG] %s: %s\n", logger->prefix, message);
    }
}

/**
 * Logger error - DEAD CODE.
 */
void logger_error(const Logger* logger, const char* message) {
    printf("[ERROR] %s: %s\n", logger->prefix, message);
}

/* ========================================================================== */
/* Hash function - DEAD CODE                                                   */
/* ========================================================================== */

/**
 * Hash string - DEAD CODE.
 */
char* hash_string(const char* s) {
    /* Simple hash for testing */
    unsigned long hash = 5381;
    int c;
    while ((c = *s++)) {
        hash = ((hash << 5) + hash) + c;
    }

    char* result = malloc(17);
    snprintf(result, 17, "%016lx", hash);
    return result;
}

/* ========================================================================== */
/* Filter/Map functions - DEAD CODE                                            */
/* ========================================================================== */

/**
 * Filter items - DEAD CODE.
 */
void** filter_items(void** items, size_t count, int (*predicate)(void*), size_t* result_count) {
    /* Count matching items */
    size_t matches = 0;
    for (size_t i = 0; i < count; i++) {
        if (predicate(items[i])) {
            matches++;
        }
    }

    /* Allocate result */
    void** result = malloc(matches * sizeof(void*));
    size_t j = 0;
    for (size_t i = 0; i < count; i++) {
        if (predicate(items[i])) {
            result[j++] = items[i];
        }
    }

    *result_count = matches;
    return result;
}

/**
 * Map items - DEAD CODE.
 */
void** map_items(void** items, size_t count, void* (*transform)(void*)) {
    void** result = malloc(count * sizeof(void*));
    for (size_t i = 0; i < count; i++) {
        result[i] = transform(items[i]);
    }
    return result;
}

/* ========================================================================== */
/* Container functions - DEAD CODE                                             */
/* ========================================================================== */

/**
 * Container init - DEAD CODE.
 */
void container_init(Container* c) {
    c->items = NULL;
    c->count = 0;
    c->capacity = 0;
}

/**
 * Container add - DEAD CODE.
 */
void container_add(Container* c, void* item) {
    if (c->count >= c->capacity) {
        size_t new_capacity = c->capacity == 0 ? 4 : c->capacity * 2;
        c->items = realloc(c->items, new_capacity * sizeof(void*));
        c->capacity = new_capacity;
    }
    c->items[c->count++] = item;
}

/**
 * Container get - DEAD CODE.
 */
void* container_get(const Container* c, size_t index) {
    if (index >= c->count) {
        return NULL;
    }
    return c->items[index];
}

/**
 * Container free - DEAD CODE.
 */
void container_free(Container* c) {
    free(c->items);
    c->items = NULL;
    c->count = 0;
    c->capacity = 0;
}

/* ========================================================================== */
/* Cache implementation - DEAD CODE                                            */
/* ========================================================================== */

typedef struct CacheEntry {
    char* key;
    void* value;
    struct CacheEntry* next;
} CacheEntry;

struct Cache {
    CacheEntry** buckets;
    size_t bucket_count;
};

static size_t cache_hash(const char* key, size_t bucket_count) {
    unsigned long hash = 5381;
    int c;
    while ((c = *key++)) {
        hash = ((hash << 5) + hash) + c;
    }
    return hash % bucket_count;
}

/**
 * Cache create - DEAD CODE.
 */
Cache* cache_create(void) {
    Cache* cache = malloc(sizeof(Cache));
    cache->bucket_count = 16;
    cache->buckets = calloc(cache->bucket_count, sizeof(CacheEntry*));
    return cache;
}

/**
 * Cache destroy - DEAD CODE.
 */
void cache_destroy(Cache* cache) {
    for (size_t i = 0; i < cache->bucket_count; i++) {
        CacheEntry* entry = cache->buckets[i];
        while (entry != NULL) {
            CacheEntry* next = entry->next;
            free(entry->key);
            free(entry);
            entry = next;
        }
    }
    free(cache->buckets);
    free(cache);
}

/**
 * Cache get - DEAD CODE.
 */
void* cache_get(const Cache* cache, const char* key) {
    size_t index = cache_hash(key, cache->bucket_count);
    CacheEntry* entry = cache->buckets[index];
    while (entry != NULL) {
        if (strcmp(entry->key, key) == 0) {
            return entry->value;
        }
        entry = entry->next;
    }
    return NULL;
}

/**
 * Cache set - DEAD CODE.
 */
void cache_set(Cache* cache, const char* key, void* value) {
    size_t index = cache_hash(key, cache->bucket_count);

    /* Check if key exists */
    CacheEntry* entry = cache->buckets[index];
    while (entry != NULL) {
        if (strcmp(entry->key, key) == 0) {
            entry->value = value;
            return;
        }
        entry = entry->next;
    }

    /* Add new entry */
    CacheEntry* new_entry = malloc(sizeof(CacheEntry));
    new_entry->key = strdup(key);
    new_entry->value = value;
    new_entry->next = cache->buckets[index];
    cache->buckets[index] = new_entry;
}

/**
 * Cache delete - DEAD CODE.
 */
void cache_delete(Cache* cache, const char* key) {
    size_t index = cache_hash(key, cache->bucket_count);
    CacheEntry** pp = &cache->buckets[index];
    while (*pp != NULL) {
        CacheEntry* entry = *pp;
        if (strcmp(entry->key, key) == 0) {
            *pp = entry->next;
            free(entry->key);
            free(entry);
            return;
        }
        pp = &entry->next;
    }
}

/**
 * Cache clear - DEAD CODE.
 */
void cache_clear(Cache* cache) {
    for (size_t i = 0; i < cache->bucket_count; i++) {
        CacheEntry* entry = cache->buckets[i];
        while (entry != NULL) {
            CacheEntry* next = entry->next;
            free(entry->key);
            free(entry);
            entry = next;
        }
        cache->buckets[i] = NULL;
    }
}

/* ========================================================================== */
/* Private helpers - DEAD CODE                                                 */
/* ========================================================================== */

/**
 * Private helper - DEAD CODE.
 */
static int private_helper(int x, int y) {
    return x + y;
}

/**
 * Another private helper - DEAD CODE.
 */
static void another_private(void) {
    /* Empty */
}
