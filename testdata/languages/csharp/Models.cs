// Models module - tests C# class definitions, interfaces, records.

using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace TestApp
{
    /// <summary>
    /// Configuration class - tests class with properties.
    /// </summary>
    public class Config
    {
        public string Host { get; set; } = "localhost";
        public int Port { get; set; } = 8080;
        public string LogLevel { get; set; } = "info";
        public Dictionary<string, object> Options { get; set; } = new();

        /// <summary>
        /// Validate configuration.
        /// </summary>
        public bool Validate()
        {
            if (string.IsNullOrEmpty(Host))
            {
                return false;
            }
            if (Port <= 0 || Port > 65535)
            {
                return false;
            }
            return true;
        }

        /// <summary>
        /// Clone the configuration.
        /// </summary>
        public Config Clone()
        {
            return new Config
            {
                Host = this.Host,
                Port = this.Port,
                LogLevel = this.LogLevel,
                Options = new Dictionary<string, object>(this.Options)
            };
        }
    }

    /// <summary>
    /// Server class - tests class with methods.
    /// </summary>
    public class Server
    {
        private readonly Config _config;
        private bool _running;
        private readonly Logger _logger;

        public Server(Config config)
        {
            _config = config;
            _logger = new Logger("server");
        }

        /// <summary>
        /// Start the server - called from Main, should be reachable.
        /// </summary>
        public void Start()
        {
            _running = true;
            _logger.Info($"Starting server on {_config.Host}:{_config.Port}");
            Listen();
        }

        /// <summary>
        /// Stop the server.
        /// </summary>
        public void Stop()
        {
            _running = false;
            _logger.Info("Stopping server");
        }

        /// <summary>
        /// Check if running.
        /// </summary>
        public bool IsRunning => _running;

        /// <summary>
        /// Internal listen method - called by Start, should be reachable.
        /// </summary>
        private void Listen()
        {
            // Simulated listening
        }

        /// <summary>
        /// Handle connection - DEAD CODE.
        /// </summary>
        private void HandleConnection(object connection)
        {
            // Handle connection
        }
    }

    // ========================================================================
    // Interface - tests interface extraction
    // ========================================================================

    /// <summary>
    /// Handler interface.
    /// </summary>
    public interface IHandler
    {
        /// <summary>
        /// Handle request.
        /// </summary>
        Response Handle(Request request);

        /// <summary>
        /// Get handler name.
        /// </summary>
        string Name { get; }
    }

    /// <summary>
    /// Async handler interface - tests generic interface.
    /// </summary>
    public interface IAsyncHandler<TRequest, TResponse>
    {
        Task<TResponse> HandleAsync(TRequest request);
    }

    /// <summary>
    /// Request class.
    /// </summary>
    public class Request
    {
        public string Method { get; set; }
        public string Path { get; set; }
        public byte[] Body { get; set; }
    }

    /// <summary>
    /// Response class.
    /// </summary>
    public class Response
    {
        public int Status { get; set; }
        public byte[] Body { get; set; }
    }

    // ========================================================================
    // Abstract class - tests abstract class extraction
    // ========================================================================

    /// <summary>
    /// Abstract base handler.
    /// </summary>
    public abstract class BaseHandler : IHandler
    {
        protected readonly string _name;

        protected BaseHandler(string name)
        {
            _name = name;
        }

        public abstract Response Handle(Request request);

        public string Name => _name;

        /// <summary>
        /// Pre-process request.
        /// </summary>
        protected virtual Request PreProcess(Request request)
        {
            return request;
        }
    }

    /// <summary>
    /// Echo handler - extends BaseHandler - DEAD CODE.
    /// </summary>
    public class EchoHandler : BaseHandler
    {
        public EchoHandler() : base("echo") { }

        public override Response Handle(Request request)
        {
            return new Response
            {
                Status = 200,
                Body = request.Body
            };
        }
    }

    /// <summary>
    /// JSON handler - extends BaseHandler - DEAD CODE.
    /// </summary>
    public class JsonHandler : BaseHandler
    {
        public JsonHandler() : base("json") { }

        public override Response Handle(Request request)
        {
            return new Response
            {
                Status = 200,
                Body = System.Text.Encoding.UTF8.GetBytes("{\"processed\": true}")
            };
        }
    }

    // ========================================================================
    // Generic class - tests generic class extraction
    // ========================================================================

    /// <summary>
    /// Generic container class.
    /// </summary>
    public class Container<T>
    {
        private readonly List<T> _items = new();

        public void Add(T item)
        {
            _items.Add(item);
        }

        public T Get(int index)
        {
            return _items[index];
        }

        public IReadOnlyList<T> All => _items.AsReadOnly();

        public int Count => _items.Count;

        /// <summary>
        /// Map items - DEAD CODE.
        /// </summary>
        public Container<U> Map<U>(Func<T, U> mapper)
        {
            var result = new Container<U>();
            foreach (var item in _items)
            {
                result.Add(mapper(item));
            }
            return result;
        }
    }

    /// <summary>
    /// Generic repository interface - DEAD CODE.
    /// </summary>
    public interface IRepository<T> where T : class
    {
        Task<T?> FindByIdAsync(string id);
        Task<IEnumerable<T>> FindAllAsync();
        Task<T> SaveAsync(T entity);
        Task DeleteAsync(string id);
    }

    /// <summary>
    /// In-memory repository - DEAD CODE.
    /// </summary>
    public class InMemoryRepository<T> : IRepository<T> where T : class
    {
        private readonly Dictionary<string, T> _items = new();

        public Task<T?> FindByIdAsync(string id)
        {
            _items.TryGetValue(id, out var item);
            return Task.FromResult(item);
        }

        public Task<IEnumerable<T>> FindAllAsync()
        {
            return Task.FromResult<IEnumerable<T>>(_items.Values);
        }

        public Task<T> SaveAsync(T entity)
        {
            // Simplified - would need ID extraction
            return Task.FromResult(entity);
        }

        public Task DeleteAsync(string id)
        {
            _items.Remove(id);
            return Task.CompletedTask;
        }
    }

    // ========================================================================
    // Record types (C# 9+) - tests record extraction
    // ========================================================================

    /// <summary>
    /// Point record - tests record extraction.
    /// </summary>
    public record Point(int X, int Y)
    {
        public double DistanceFromOrigin()
        {
            return Math.Sqrt(X * X + Y * Y);
        }
    }

    /// <summary>
    /// User record - DEAD CODE.
    /// </summary>
    public record User(string Id, string Name, string Email);

    /// <summary>
    /// Immutable config record - DEAD CODE.
    /// </summary>
    public record ConfigRecord
    {
        public string Host { get; init; } = "localhost";
        public int Port { get; init; } = 8080;
    }

    // ========================================================================
    // Enum - tests enum extraction
    // ========================================================================

    /// <summary>
    /// Log level enum.
    /// </summary>
    public enum LogLevel
    {
        Debug = 0,
        Info = 1,
        Warn = 2,
        Error = 3
    }

    /// <summary>
    /// HTTP method enum.
    /// </summary>
    public enum HttpMethod
    {
        Get,
        Post,
        Put,
        Delete,
        Patch
    }

    // ========================================================================
    // Struct - tests struct extraction
    // ========================================================================

    /// <summary>
    /// Color struct - tests struct extraction.
    /// </summary>
    public struct Color
    {
        public byte R { get; set; }
        public byte G { get; set; }
        public byte B { get; set; }

        public Color(byte r, byte g, byte b)
        {
            R = r;
            G = g;
            B = b;
        }

        public string ToHex()
        {
            return $"#{R:X2}{G:X2}{B:X2}";
        }
    }

    // ========================================================================
    // Unused classes - DEAD CODE
    // ========================================================================

    /// <summary>
    /// Unused class - DEAD CODE.
    /// </summary>
    public class UnusedClass
    {
        private readonly string _value;

        public UnusedClass(string value)
        {
            _value = value;
        }

        public void Process()
        {
            Console.WriteLine($"Processing: {_value}");
        }
    }

    /// <summary>
    /// Another unused class - DEAD CODE.
    /// </summary>
    public class AnotherUnusedClass<T>
    {
        private T _data;

        public AnotherUnusedClass(T data)
        {
            _data = data;
        }

        public T Data => _data;
    }
}
