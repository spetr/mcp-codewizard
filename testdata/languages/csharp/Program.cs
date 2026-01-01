// Main module demonstrating various C# patterns for parser testing.
// Tests: entry points, method calls, properties, async/await.

using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace TestApp
{
    /// <summary>
    /// Main program class - tests entry point detection.
    /// </summary>
    public class Program
    {
        // Static constants - tests constant extraction
        public const int MaxRetries = 3;
        public const int DefaultTimeout = 30;

        // Static variables - tests field extraction
        private static string _appName = "TestApp";
        private static readonly string InternalVersion = "1.0.0";

        // Static logger - tests static initialization
        private static readonly Logger Logger = CreateLogger("main");

        /// <summary>
        /// Main entry point - should be marked as reachable.
        /// </summary>
        public static void Main(string[] args)
        {
            Console.WriteLine($"Starting {_appName}");

            // Method calls - tests reference extraction
            var config = LoadConfig();
            if (!Initialize(config))
            {
                Console.Error.WriteLine("Initialization failed");
                Environment.Exit(1);
            }

            // Method calls on objects
            var server = new Server(config);
            server.Start();

            // Using utility methods
            var result = Utils.ProcessData(new[] { "a", "b", "c" });
            Console.WriteLine(Utils.FormatOutput(result));

            // Calling transitive methods
            RunPipeline();
        }

        /// <summary>
        /// Async main for .NET Core - alternative entry point.
        /// </summary>
        public static async Task MainAsync(string[] args)
        {
            await Task.Run(() => Main(args));
        }

        /// <summary>
        /// Load configuration - called from Main, should be reachable.
        /// </summary>
        private static Config LoadConfig()
        {
            return new Config
            {
                Host = "localhost",
                Port = 8080,
                LogLevel = "info"
            };
        }

        /// <summary>
        /// Initialize application - called from Main, should be reachable.
        /// </summary>
        private static bool Initialize(Config config)
        {
            if (config == null)
            {
                return false;
            }
            SetupLogging(config.LogLevel);
            return true;
        }

        /// <summary>
        /// Internal helper - called from Initialize, should be reachable.
        /// </summary>
        private static void SetupLogging(string level)
        {
            Console.WriteLine($"Setting log level to: {level}");
        }

        /// <summary>
        /// Orchestrate data pipeline - tests transitive reachability.
        /// </summary>
        private static void RunPipeline()
        {
            var data = FetchData();
            var transformed = TransformData(data);
            SaveData(transformed);
        }

        /// <summary>
        /// Fetch data - called by RunPipeline, should be reachable.
        /// </summary>
        private static byte[] FetchData()
        {
            return System.Text.Encoding.UTF8.GetBytes("sample data");
        }

        /// <summary>
        /// Transform data - called by RunPipeline, should be reachable.
        /// </summary>
        private static byte[] TransformData(byte[] data)
        {
            var prefix = System.Text.Encoding.UTF8.GetBytes("transformed: ");
            var result = new byte[prefix.Length + data.Length];
            Array.Copy(prefix, 0, result, 0, prefix.Length);
            Array.Copy(data, 0, result, prefix.Length, data.Length);
            return result;
        }

        /// <summary>
        /// Save data - called by RunPipeline, should be reachable.
        /// </summary>
        private static void SaveData(byte[] data)
        {
            Console.WriteLine($"Saving: {System.Text.Encoding.UTF8.GetString(data)}");
        }

        /// <summary>
        /// Create logger - called from static init.
        /// </summary>
        private static Logger CreateLogger(string name)
        {
            return new Logger(name);
        }

        // ====================================================================
        // Dead code section - methods that are never called
        // ====================================================================

        /// <summary>
        /// This method is never called - DEAD CODE.
        /// </summary>
        private static void UnusedMethod()
        {
            Console.WriteLine("This is never executed");
        }

        /// <summary>
        /// Also never called - DEAD CODE.
        /// </summary>
        private static string AnotherUnused()
        {
            return "dead";
        }

        /// <summary>
        /// Starts a chain of dead code - DEAD CODE.
        /// </summary>
        private static void DeadChainStart()
        {
            DeadChainMiddle();
        }

        /// <summary>
        /// In the middle of dead chain - DEAD CODE (transitive).
        /// </summary>
        private static void DeadChainMiddle()
        {
            DeadChainEnd();
        }

        /// <summary>
        /// End of dead chain - DEAD CODE (transitive).
        /// </summary>
        private static void DeadChainEnd()
        {
            Console.WriteLine("End of dead chain");
        }
    }

    /// <summary>
    /// Simple logger class.
    /// </summary>
    public class Logger
    {
        private readonly string _prefix;
        private int _level = 1;

        public Logger(string prefix)
        {
            _prefix = prefix;
        }

        public void Info(string message)
        {
            Console.WriteLine($"[INFO] {_prefix}: {message}");
        }

        /// <summary>
        /// Debug method - DEAD CODE.
        /// </summary>
        public void Debug(string message)
        {
            if (_level >= 2)
            {
                Console.WriteLine($"[DEBUG] {_prefix}: {message}");
            }
        }

        /// <summary>
        /// Error method - DEAD CODE.
        /// </summary>
        public void Error(string message)
        {
            Console.WriteLine($"[ERROR] {_prefix}: {message}");
        }
    }
}
