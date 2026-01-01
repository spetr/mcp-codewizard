/**
 * Models module - tests classes, data classes, sealed classes, and enums.
 * Tests: class extraction, interface extraction, enum extraction.
 */

package app

/**
 * Log level enum.
 */
enum class LogLevel(val level: Int) {
    DEBUG(0),
    INFO(1),
    WARN(2),
    ERROR(3)
}

/**
 * HTTP method enum.
 */
enum class HttpMethod {
    GET, POST, PUT, DELETE, PATCH
}

/**
 * Configuration data class - tests data class extraction.
 */
data class Config(
    val host: String = "localhost",
    val port: Int = 8080,
    val logLevel: String = "info",
    val options: Map<String, String> = emptyMap()
) {
    fun validate(): Boolean {
        if (host.isEmpty()) return false
        if (port <= 0 || port > 65535) return false
        return true
    }

    fun clone(): Config = copy()
}

/**
 * Handler interface - tests interface extraction.
 */
interface Handler {
    fun handle(input: String): String
    val name: String
}

/**
 * Logger class.
 */
class Logger(private val prefix: String) {
    var level: LogLevel = LogLevel.INFO

    fun info(message: String) {
        println("[INFO] $prefix: $message")
    }

    // DEAD CODE
    fun debug(message: String) {
        if (level.level <= LogLevel.DEBUG.level) {
            println("[DEBUG] $prefix: $message")
        }
    }

    // DEAD CODE
    fun error(message: String) {
        println("[ERROR] $prefix: $message")
    }
}

/**
 * Server class - tests class with methods.
 */
class Server(private val config: Config) {
    private var running = false
    private val logger = Logger("server")

    private fun listen() {
        // Simulated listening
    }

    // DEAD CODE
    private fun handleConnection(connection: Any?) {
        // Handle connection
    }

    fun start() {
        running = true
        logger.info("Starting server on ${config.host}:${config.port}")
        listen()
    }

    fun stop() {
        running = false
        logger.info("Stopping server")
    }

    fun isRunning(): Boolean = running
}

/**
 * Echo handler - DEAD CODE.
 */
class EchoHandler : Handler {
    override fun handle(input: String): String = input
    override val name: String = "echo"
}

/**
 * Upper handler - DEAD CODE.
 */
class UpperHandler : Handler {
    override fun handle(input: String): String = input.uppercase()
    override val name: String = "upper"
}

/**
 * Generic container class.
 */
class Container<T> {
    private val items = mutableListOf<T>()

    fun add(item: T) {
        items.add(item)
    }

    fun get(index: Int): T = items[index]

    fun all(): List<T> = items.toList()

    val size: Int get() = items.size

    fun <U> map(mapper: (T) -> U): Container<U> {
        val result = Container<U>()
        items.forEach { result.add(mapper(it)) }
        return result
    }
}

/**
 * Pair data class - DEAD CODE.
 */
@Suppress("unused")
data class Pair<A, B>(val first: A, val second: B)

/**
 * Sealed result class - tests sealed class extraction.
 */
sealed class Result<out T> {
    data class Success<T>(val value: T) : Result<T>()
    data class Failure(val error: Throwable) : Result<Nothing>()
    data object Loading : Result<Nothing>()
}

/**
 * Cache class - DEAD CODE.
 */
@Suppress("unused")
class Cache {
    private val data = mutableMapOf<String, Any?>()

    fun set(key: String, value: Any?) {
        data[key] = value
    }

    @Suppress("UNCHECKED_CAST")
    fun <T> get(key: String): T? = data[key] as? T

    fun delete(key: String) {
        data.remove(key)
    }

    fun clear() {
        data.clear()
    }
}

/**
 * Object singleton - tests object declaration.
 */
object AppState {
    var isInitialized = false
    val version = "1.0.0"

    fun reset() {
        isInitialized = false
    }
}

/**
 * Companion object example - tests companion object.
 */
class Factory {
    companion object {
        fun create(): Factory = Factory()
        const val DEFAULT_NAME = "factory"
    }

    fun produce(): String = "product"
}
