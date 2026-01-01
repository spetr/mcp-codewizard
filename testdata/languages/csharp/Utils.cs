// Utils module - tests C# utility methods, LINQ, async patterns.

using System;
using System.Collections.Generic;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using System.Threading.Tasks;

namespace TestApp
{
    /// <summary>
    /// Utils class - tests utility methods.
    /// </summary>
    public static class Utils
    {
        /// <summary>
        /// Process string data - called from Main, should be reachable.
        /// </summary>
        public static string ProcessData(string[] items)
        {
            // Using lambda - tests lambda extraction
            Func<string, string> mapper = s => s.ToUpper();

            return string.Join(", ", items.Select(mapper));
        }

        /// <summary>
        /// Format output string - called from Main, should be reachable.
        /// </summary>
        public static string FormatOutput(string data)
        {
            return $"Result: {data}";
        }

        /// <summary>
        /// Compute SHA256 hash - DEAD CODE.
        /// </summary>
        public static string HashString(string s)
        {
            using var sha256 = SHA256.Create();
            var bytes = sha256.ComputeHash(Encoding.UTF8.GetBytes(s));
            return BitConverter.ToString(bytes).Replace("-", "").ToLower();
        }

        /// <summary>
        /// Filter items by predicate - DEAD CODE.
        /// </summary>
        public static IEnumerable<T> FilterItems<T>(IEnumerable<T> items, Func<T, bool> predicate)
        {
            return items.Where(predicate);
        }

        /// <summary>
        /// Map items with transform - DEAD CODE.
        /// </summary>
        public static IEnumerable<U> MapItems<T, U>(IEnumerable<T> items, Func<T, U> transform)
        {
            return items.Select(transform);
        }

        /// <summary>
        /// Reduce items to single value - DEAD CODE.
        /// </summary>
        public static U ReduceItems<T, U>(IEnumerable<T> items, U initial, Func<U, T, U> reducer)
        {
            return items.Aggregate(initial, reducer);
        }

        // ====================================================================
        // Higher-order functions - tests delegate extraction
        // ====================================================================

        /// <summary>
        /// Compose functions - DEAD CODE.
        /// </summary>
        public static Func<T, T> Compose<T>(params Func<T, T>[] functions)
        {
            return x => functions.Aggregate(x, (acc, f) => f(acc));
        }

        /// <summary>
        /// Memoize function - DEAD CODE.
        /// </summary>
        public static Func<TKey, TValue> Memoize<TKey, TValue>(Func<TKey, TValue> func)
            where TKey : notnull
        {
            var cache = new Dictionary<TKey, TValue>();
            return key =>
            {
                if (!cache.TryGetValue(key, out var value))
                {
                    value = func(key);
                    cache[key] = value;
                }
                return value;
            };
        }

        /// <summary>
        /// Retry function - DEAD CODE.
        /// </summary>
        public static T Retry<T>(Func<T> func, int attempts, int delayMs)
        {
            Exception? lastError = null;
            for (int i = 0; i < attempts; i++)
            {
                try
                {
                    return func();
                }
                catch (Exception e)
                {
                    lastError = e;
                    if (i < attempts - 1)
                    {
                        System.Threading.Thread.Sleep(delayMs);
                    }
                }
            }
            throw new Exception($"Failed after {attempts} attempts", lastError);
        }

        // ====================================================================
        // Async utilities - DEAD CODE
        // ====================================================================

        /// <summary>
        /// Async retry - DEAD CODE.
        /// </summary>
        public static async Task<T> RetryAsync<T>(
            Func<Task<T>> func,
            int attempts,
            int delayMs)
        {
            Exception? lastError = null;
            for (int i = 0; i < attempts; i++)
            {
                try
                {
                    return await func();
                }
                catch (Exception e)
                {
                    lastError = e;
                    if (i < attempts - 1)
                    {
                        await Task.Delay(delayMs);
                    }
                }
            }
            throw new Exception($"Failed after {attempts} attempts", lastError);
        }

        /// <summary>
        /// With timeout - DEAD CODE.
        /// </summary>
        public static async Task<T> WithTimeout<T>(Task<T> task, int timeoutMs)
        {
            var timeoutTask = Task.Delay(timeoutMs);
            var completedTask = await Task.WhenAny(task, timeoutTask);

            if (completedTask == timeoutTask)
            {
                throw new TimeoutException($"Operation timed out after {timeoutMs}ms");
            }

            return await task;
        }

        /// <summary>
        /// When all with limit - DEAD CODE.
        /// </summary>
        public static async Task<T[]> WhenAllWithLimit<T>(
            IEnumerable<Func<Task<T>>> tasks,
            int maxParallel)
        {
            var semaphore = new System.Threading.SemaphoreSlim(maxParallel);
            var taskList = tasks.Select(async taskFunc =>
            {
                await semaphore.WaitAsync();
                try
                {
                    return await taskFunc();
                }
                finally
                {
                    semaphore.Release();
                }
            });

            return await Task.WhenAll(taskList);
        }

        // ====================================================================
        // LINQ utilities - DEAD CODE
        // ====================================================================

        /// <summary>
        /// Chunk items - DEAD CODE.
        /// </summary>
        public static IEnumerable<IEnumerable<T>> Chunk<T>(IEnumerable<T> items, int size)
        {
            return items
                .Select((item, index) => new { item, index })
                .GroupBy(x => x.index / size)
                .Select(g => g.Select(x => x.item));
        }

        /// <summary>
        /// Distinct by key - DEAD CODE.
        /// </summary>
        public static IEnumerable<T> DistinctBy<T, TKey>(
            IEnumerable<T> items,
            Func<T, TKey> keySelector)
        {
            var seen = new HashSet<TKey>();
            foreach (var item in items)
            {
                if (seen.Add(keySelector(item)))
                {
                    yield return item;
                }
            }
        }

        /// <summary>
        /// Flatten - DEAD CODE.
        /// </summary>
        public static IEnumerable<T> Flatten<T>(IEnumerable<IEnumerable<T>> nested)
        {
            return nested.SelectMany(x => x);
        }

        /// <summary>
        /// Zip with - DEAD CODE.
        /// </summary>
        public static IEnumerable<TResult> ZipWith<T1, T2, TResult>(
            IEnumerable<T1> first,
            IEnumerable<T2> second,
            Func<T1, T2, TResult> resultSelector)
        {
            return first.Zip(second, resultSelector);
        }

        // ====================================================================
        // String utilities - DEAD CODE
        // ====================================================================

        /// <summary>
        /// Is blank check - DEAD CODE.
        /// </summary>
        public static bool IsBlank(string? s)
        {
            return string.IsNullOrWhiteSpace(s);
        }

        /// <summary>
        /// Truncate string - DEAD CODE.
        /// </summary>
        public static string Truncate(string s, int maxLength)
        {
            if (s == null || s.Length <= maxLength)
            {
                return s ?? string.Empty;
            }
            return s.Substring(0, maxLength) + "...";
        }

        /// <summary>
        /// Safe substring - DEAD CODE.
        /// </summary>
        public static string SafeSubstring(string s, int start, int length)
        {
            if (string.IsNullOrEmpty(s) || start >= s.Length)
            {
                return string.Empty;
            }
            return s.Substring(start, Math.Min(length, s.Length - start));
        }

        // ====================================================================
        // Private helpers - DEAD CODE
        // ====================================================================

        /// <summary>
        /// Private helper - DEAD CODE.
        /// </summary>
        private static int PrivateHelper(int x, int y)
        {
            return x + y;
        }

        /// <summary>
        /// Another private helper - DEAD CODE.
        /// </summary>
        private static void AnotherPrivate()
        {
            // Empty
        }
    }

    // ========================================================================
    // Cache class - DEAD CODE
    // ========================================================================

    /// <summary>
    /// Simple cache - DEAD CODE.
    /// </summary>
    public class Cache<TKey, TValue> where TKey : notnull
    {
        private readonly Dictionary<TKey, TValue> _data = new();

        public TValue? Get(TKey key)
        {
            _data.TryGetValue(key, out var value);
            return value;
        }

        public void Set(TKey key, TValue value)
        {
            _data[key] = value;
        }

        public bool Has(TKey key)
        {
            return _data.ContainsKey(key);
        }

        public void Remove(TKey key)
        {
            _data.Remove(key);
        }

        public void Clear()
        {
            _data.Clear();
        }
    }

    // ========================================================================
    // Extension methods - DEAD CODE
    // ========================================================================

    /// <summary>
    /// String extensions - DEAD CODE.
    /// </summary>
    public static class StringExtensions
    {
        public static bool IsNullOrEmpty(this string? s)
        {
            return string.IsNullOrEmpty(s);
        }

        public static string OrDefault(this string? s, string defaultValue)
        {
            return string.IsNullOrEmpty(s) ? defaultValue : s;
        }

        public static int? ToIntOrNull(this string s)
        {
            return int.TryParse(s, out var result) ? result : null;
        }
    }

    /// <summary>
    /// Enumerable extensions - DEAD CODE.
    /// </summary>
    public static class EnumerableExtensions
    {
        public static bool IsEmpty<T>(this IEnumerable<T> items)
        {
            return !items.Any();
        }

        public static T? FirstOrNull<T>(this IEnumerable<T> items) where T : class
        {
            return items.FirstOrDefault();
        }

        public static IEnumerable<T> WhereNotNull<T>(this IEnumerable<T?> items) where T : class
        {
            return items.Where(x => x != null)!;
        }
    }
}
