<?php
/**
 * Models module - tests classes, interfaces, traits, and enums.
 * Tests: class extraction, interface extraction, trait extraction.
 */

declare(strict_types=1);

namespace App;

/**
 * Log level enum.
 */
enum LogLevel: int
{
    case Debug = 0;
    case Info = 1;
    case Warn = 2;
    case Error = 3;
}

/**
 * HTTP method enum.
 */
enum HttpMethod: string
{
    case Get = 'GET';
    case Post = 'POST';
    case Put = 'PUT';
    case Delete = 'DELETE';
    case Patch = 'PATCH';
}

/**
 * Configuration class - tests class with constructor property promotion.
 */
class Config
{
    public function __construct(
        public string $host = 'localhost',
        public int $port = 8080,
        public string $logLevel = 'info',
        public array $options = []
    ) {}

    public function validate(): bool
    {
        if (empty($this->host)) {
            return false;
        }
        if ($this->port <= 0 || $this->port > 65535) {
            return false;
        }
        return true;
    }

    public function clone(): self
    {
        return new self($this->host, $this->port, $this->logLevel, $this->options);
    }
}

/**
 * Handler interface - tests interface extraction.
 */
interface Handler
{
    public function handle(string $input): string;
    public function getName(): string;
}

/**
 * Loggable trait - tests trait extraction.
 */
trait Loggable
{
    protected Logger $logger;

    protected function initLogger(string $prefix): void
    {
        $this->logger = new Logger($prefix);
    }

    protected function log(string $message): void
    {
        $this->logger->info($message);
    }
}

/**
 * Logger class.
 */
class Logger
{
    private LogLevel $level = LogLevel::Info;

    public function __construct(private string $prefix) {}

    public function info(string $message): void
    {
        echo "[INFO] {$this->prefix}: {$message}\n";
    }

    // DEAD CODE
    public function debug(string $message): void
    {
        if ($this->level->value <= LogLevel::Debug->value) {
            echo "[DEBUG] {$this->prefix}: {$message}\n";
        }
    }

    // DEAD CODE
    public function error(string $message): void
    {
        echo "[ERROR] {$this->prefix}: {$message}\n";
    }

    public function setLevel(LogLevel $level): void
    {
        $this->level = $level;
    }
}

/**
 * Server class - tests class with methods.
 */
class Server
{
    use Loggable;

    private bool $running = false;

    public function __construct(private Config $config)
    {
        $this->initLogger('server');
    }

    private function listen(): void
    {
        // Simulated listening
    }

    // DEAD CODE
    private function handleConnection(mixed $connection): void
    {
        // Handle connection
    }

    public function start(): void
    {
        $this->running = true;
        $this->log("Starting server on {$this->config->host}:{$this->config->port}");
        $this->listen();
    }

    public function stop(): void
    {
        $this->running = false;
        $this->log('Stopping server');
    }

    public function isRunning(): bool
    {
        return $this->running;
    }
}

/**
 * Echo handler - DEAD CODE.
 */
class EchoHandler implements Handler
{
    public function handle(string $input): string
    {
        return $input;
    }

    public function getName(): string
    {
        return 'echo';
    }
}

/**
 * Upper handler - DEAD CODE.
 */
class UpperHandler implements Handler
{
    public function handle(string $input): string
    {
        return strtoupper($input);
    }

    public function getName(): string
    {
        return 'upper';
    }
}

/**
 * Generic container class.
 * @template T
 */
class Container
{
    /** @var array<T> */
    private array $items = [];

    /**
     * @param T $item
     */
    public function add(mixed $item): void
    {
        $this->items[] = $item;
    }

    /**
     * @return T|null
     */
    public function get(int $index): mixed
    {
        return $this->items[$index] ?? null;
    }

    /**
     * @return array<T>
     */
    public function all(): array
    {
        return $this->items;
    }

    public function count(): int
    {
        return count($this->items);
    }

    /**
     * @template U
     * @param callable(T): U $mapper
     * @return Container<U>
     */
    public function map(callable $mapper): Container
    {
        $result = new Container();
        foreach ($this->items as $item) {
            $result->add($mapper($item));
        }
        return $result;
    }
}

/**
 * Pair class - DEAD CODE.
 * @template TFirst
 * @template TSecond
 */
class Pair
{
    /**
     * @param TFirst $first
     * @param TSecond $second
     */
    public function __construct(
        public mixed $first,
        public mixed $second
    ) {}
}

/**
 * Cache class - DEAD CODE.
 */
class Cache
{
    private array $data = [];

    public function set(string $key, mixed $value): void
    {
        $this->data[$key] = $value;
    }

    public function get(string $key): mixed
    {
        return $this->data[$key] ?? null;
    }

    public function delete(string $key): void
    {
        unset($this->data[$key]);
    }

    public function clear(): void
    {
        $this->data = [];
    }
}
