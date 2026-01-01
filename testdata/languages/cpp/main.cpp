/**
 * Main module demonstrating various C++ patterns for parser testing.
 * Tests: entry points, classes, templates, namespaces.
 */

#include <iostream>
#include <string>
#include <vector>
#include <memory>
#include <functional>
#include <map>

// Constants - tests constant extraction
constexpr int MAX_RETRIES = 3;
constexpr int DEFAULT_TIMEOUT = 30;
const char* const APP_NAME = "TestApp";

// Namespace - tests namespace extraction
namespace app {

/**
 * Configuration class - tests class extraction.
 */
class Config {
public:
    std::string host = "localhost";
    int port = 8080;
    std::string log_level = "info";
    std::map<std::string, std::string> options;

    Config() = default;
    Config(const std::string& h, int p, const std::string& level)
        : host(h), port(p), log_level(level) {}

    bool validate() const {
        if (host.empty()) return false;
        if (port <= 0 || port > 65535) return false;
        return true;
    }

    Config clone() const {
        Config c(host, port, log_level);
        c.options = options;
        return c;
    }
};

/**
 * Logger class.
 */
class Logger {
private:
    std::string prefix_;
    int level_ = 1;

public:
    explicit Logger(const std::string& prefix) : prefix_(prefix) {}

    void info(const std::string& message) const {
        std::cout << "[INFO] " << prefix_ << ": " << message << std::endl;
    }

    // DEAD CODE
    void debug(const std::string& message) const {
        if (level_ >= 2) {
            std::cout << "[DEBUG] " << prefix_ << ": " << message << std::endl;
        }
    }

    // DEAD CODE
    void error(const std::string& message) const {
        std::cout << "[ERROR] " << prefix_ << ": " << message << std::endl;
    }
};

/**
 * Server class - tests class with methods.
 */
class Server {
private:
    Config config_;
    bool running_ = false;
    Logger logger_{"server"};

    void listen() {
        // Simulated listening
    }

    // DEAD CODE
    void handleConnection(void* conn) {
        // Handle connection
    }

public:
    explicit Server(const Config& config) : config_(config) {}

    void start() {
        running_ = true;
        logger_.info("Starting server on " + config_.host + ":" + std::to_string(config_.port));
        listen();
    }

    void stop() {
        running_ = false;
        logger_.info("Stopping server");
    }

    bool isRunning() const { return running_; }
};

// Utility functions
std::string processData(const std::vector<std::string>& items);
std::string formatOutput(const std::string& data);

} // namespace app

// Global logger
static app::Logger mainLogger("main");

// Forward declarations
static app::Config loadConfig();
static bool initialize(const app::Config& config);
static void setupLogging(const std::string& level);
static void runPipeline();
static std::vector<char> fetchData();
static std::vector<char> transformData(const std::vector<char>& data);
static void saveData(const std::vector<char>& data);

/**
 * Main entry point - should be marked as reachable.
 */
int main(int argc, char* argv[]) {
    std::cout << "Starting " << APP_NAME << std::endl;

    // Function calls - tests reference extraction
    auto config = loadConfig();
    if (!initialize(config)) {
        std::cerr << "Initialization failed" << std::endl;
        return 1;
    }

    // Method calls on objects
    app::Server server(config);
    server.start();

    // Using utility functions
    auto result = app::processData({"a", "b", "c"});
    std::cout << app::formatOutput(result) << std::endl;

    // Calling transitive functions
    runPipeline();

    // Cleanup
    server.stop();
    return 0;
}

/**
 * Load configuration - called from main, should be reachable.
 */
static app::Config loadConfig() {
    return app::Config("localhost", 8080, "info");
}

/**
 * Initialize application - called from main, should be reachable.
 */
static bool initialize(const app::Config& config) {
    setupLogging(config.log_level);
    return true;
}

/**
 * Internal helper - called from initialize, should be reachable.
 */
static void setupLogging(const std::string& level) {
    std::cout << "Setting log level to: " << level << std::endl;
}

/**
 * Orchestrate data pipeline - tests transitive reachability.
 */
static void runPipeline() {
    auto data = fetchData();
    auto transformed = transformData(data);
    saveData(transformed);
}

/**
 * Fetch data - called by runPipeline, should be reachable.
 */
static std::vector<char> fetchData() {
    std::string s = "sample data";
    return std::vector<char>(s.begin(), s.end());
}

/**
 * Transform data - called by runPipeline, should be reachable.
 */
static std::vector<char> transformData(const std::vector<char>& data) {
    std::string prefix = "transformed: ";
    std::vector<char> result(prefix.begin(), prefix.end());
    result.insert(result.end(), data.begin(), data.end());
    return result;
}

/**
 * Save data - called by runPipeline, should be reachable.
 */
static void saveData(const std::vector<char>& data) {
    std::cout << "Saving: " << std::string(data.begin(), data.end()) << std::endl;
}

// Implement namespace functions
namespace app {

std::string processData(const std::vector<std::string>& items) {
    std::string result;
    for (size_t i = 0; i < items.size(); ++i) {
        std::string upper;
        for (char c : items[i]) {
            upper += static_cast<char>(std::toupper(c));
        }
        result += upper;
        if (i < items.size() - 1) {
            result += ", ";
        }
    }
    return result;
}

std::string formatOutput(const std::string& data) {
    return "Result: " + data;
}

} // namespace app

// ============================================================================
// Template classes - tests template extraction
// ============================================================================

/**
 * Generic container - tests template class.
 */
template<typename T>
class Container {
private:
    std::vector<T> items_;

public:
    void add(const T& item) {
        items_.push_back(item);
    }

    T& get(size_t index) {
        return items_.at(index);
    }

    const std::vector<T>& all() const {
        return items_;
    }

    size_t size() const {
        return items_.size();
    }

    // Map function
    template<typename U>
    Container<U> map(std::function<U(const T&)> mapper) const {
        Container<U> result;
        for (const auto& item : items_) {
            result.add(mapper(item));
        }
        return result;
    }
};

/**
 * Generic pair - DEAD CODE.
 */
template<typename T, typename U>
struct Pair {
    T first;
    U second;

    Pair(const T& f, const U& s) : first(f), second(s) {}
};

// ============================================================================
// Interface-like abstract class - tests abstract class extraction
// ============================================================================

/**
 * Handler interface - tests pure virtual.
 */
class IHandler {
public:
    virtual ~IHandler() = default;
    virtual std::string handle(const std::string& input) = 0;
    virtual std::string name() const = 0;
};

/**
 * Echo handler - DEAD CODE.
 */
class EchoHandler : public IHandler {
public:
    std::string handle(const std::string& input) override {
        return input;
    }

    std::string name() const override {
        return "echo";
    }
};

/**
 * Upper handler - DEAD CODE.
 */
class UpperHandler : public IHandler {
public:
    std::string handle(const std::string& input) override {
        std::string result;
        for (char c : input) {
            result += static_cast<char>(std::toupper(c));
        }
        return result;
    }

    std::string name() const override {
        return "upper";
    }
};

// ============================================================================
// Lambda and functional - tests lambda extraction
// ============================================================================

/**
 * Create adder - DEAD CODE.
 */
auto makeAdder(int x) {
    return [x](int y) { return x + y; };
}

/**
 * Apply function twice - DEAD CODE.
 */
template<typename F>
auto applyTwice(F f, int x) {
    return f(f(x));
}

// ============================================================================
// Dead code section
// ============================================================================

/**
 * Unused function - DEAD CODE.
 */
static void unusedFunction() {
    std::cout << "This is never executed" << std::endl;
}

/**
 * Another unused - DEAD CODE.
 */
static std::string anotherUnused() {
    return "dead";
}

/**
 * Dead chain start - DEAD CODE.
 */
static void deadChainStart() {
    // Would call deadChainMiddle
}

/**
 * Dead chain middle - DEAD CODE.
 */
static void deadChainMiddle() {
    // Would call deadChainEnd
}

/**
 * Dead chain end - DEAD CODE.
 */
static void deadChainEnd() {
    std::cout << "End of dead chain" << std::endl;
}

// ============================================================================
// Smart pointer usage - tests modern C++
// ============================================================================

/**
 * Cache class using smart pointers - DEAD CODE.
 */
class Cache {
private:
    std::map<std::string, std::shared_ptr<void>> data_;

public:
    template<typename T>
    void set(const std::string& key, std::shared_ptr<T> value) {
        data_[key] = value;
    }

    template<typename T>
    std::shared_ptr<T> get(const std::string& key) {
        auto it = data_.find(key);
        if (it != data_.end()) {
            return std::static_pointer_cast<T>(it->second);
        }
        return nullptr;
    }

    void remove(const std::string& key) {
        data_.erase(key);
    }

    void clear() {
        data_.clear();
    }
};

// ============================================================================
// Enum class - tests C++11 enum
// ============================================================================

/**
 * Log level enum class.
 */
enum class LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3
};

/**
 * HTTP method enum class.
 */
enum class HttpMethod {
    Get,
    Post,
    Put,
    Delete,
    Patch
};
