"""
Utils module - tests utility functions, decorators, closures.
"""

import functools
import hashlib
import time
from typing import Any, Callable, Dict, List, Optional, TypeVar

T = TypeVar('T')
F = TypeVar('F', bound=Callable[..., Any])


def process_data(items: List[str]) -> str:
    """Process string data - called from main, should be reachable."""
    # Using lambda - tests lambda extraction
    mapper = lambda s: s.upper()

    results = [mapper(item) for item in items]
    return ", ".join(results)


def format_output(data: str) -> str:
    """Format output string - called from main, should be reachable."""
    return f"Result: {data}"


def hash_string(s: str) -> str:
    """Compute SHA256 hash - DEAD CODE."""
    return hashlib.sha256(s.encode()).hexdigest()


def filter_items(items: List[T], predicate: Callable[[T], bool]) -> List[T]:
    """Filter items by predicate - DEAD CODE."""
    return [item for item in items if predicate(item)]


def map_items(items: List[T], transform: Callable[[T], T]) -> List[T]:
    """Transform items - DEAD CODE."""
    return [transform(item) for item in items]


def reduce_items(
    items: List[T],
    reducer: Callable[[T, T], T],
    initial: T
) -> T:
    """Reduce items to single value - DEAD CODE."""
    result = initial
    for item in items:
        result = reducer(result, item)
    return result


# ============================================================================
# Decorators - tests decorator extraction
# ============================================================================

def timer(func: F) -> F:
    """Decorator that times function execution - tests decorator pattern."""
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        start = time.time()
        result = func(*args, **kwargs)
        elapsed = time.time() - start
        print(f"{func.__name__} took {elapsed:.4f}s")
        return result
    return wrapper  # type: ignore


def retry(attempts: int = 3, delay: float = 1.0):
    """Decorator factory for retry logic - tests decorator factory."""
    def decorator(func: F) -> F:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            last_error = None
            for i in range(attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    last_error = e
                    if i < attempts - 1:
                        time.sleep(delay)
            raise last_error  # type: ignore
        return wrapper  # type: ignore
    return decorator


def deprecated(message: str = ""):
    """Mark function as deprecated - DEAD CODE decorator."""
    def decorator(func: F) -> F:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            import warnings
            warnings.warn(
                f"{func.__name__} is deprecated. {message}",
                DeprecationWarning,
                stacklevel=2
            )
            return func(*args, **kwargs)
        return wrapper  # type: ignore
    return decorator


def memoize(func: F) -> F:
    """Memoization decorator - DEAD CODE."""
    cache: Dict[str, Any] = {}

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        key = str(args) + str(kwargs)
        if key not in cache:
            cache[key] = func(*args, **kwargs)
        return cache[key]
    return wrapper  # type: ignore


# ============================================================================
# Context managers - tests context manager extraction
# ============================================================================

class Timer:
    """Context manager for timing - tests __enter__/__exit__ pattern."""

    def __init__(self, name: str = ""):
        self.name = name
        self.start: float = 0
        self.elapsed: float = 0

    def __enter__(self) -> 'Timer':
        self.start = time.time()
        return self

    def __exit__(self, *args) -> None:
        self.elapsed = time.time() - self.start
        if self.name:
            print(f"{self.name}: {self.elapsed:.4f}s")


class DatabaseConnection:
    """Database connection context manager - DEAD CODE."""

    def __init__(self, connection_string: str):
        self.connection_string = connection_string
        self._connected = False

    def __enter__(self) -> 'DatabaseConnection':
        self._connect()
        return self

    def __exit__(self, *args) -> None:
        self._disconnect()

    def _connect(self) -> None:
        self._connected = True

    def _disconnect(self) -> None:
        self._connected = False

    def execute(self, query: str) -> List[Dict[str, Any]]:
        """Execute query - DEAD CODE."""
        return []


# ============================================================================
# Cache class - DEAD CODE
# ============================================================================

class Cache:
    """Simple cache implementation - DEAD CODE."""

    def __init__(self):
        self._data: Dict[str, Any] = {}

    def get(self, key: str) -> Optional[Any]:
        """Get value from cache."""
        return self._data.get(key)

    def set(self, key: str, value: Any) -> None:
        """Set value in cache."""
        self._data[key] = value

    def delete(self, key: str) -> None:
        """Delete value from cache."""
        self._data.pop(key, None)

    def clear(self) -> None:
        """Clear all values."""
        self._data.clear()


# ============================================================================
# Builder pattern - DEAD CODE
# ============================================================================

class QueryBuilder:
    """SQL query builder - DEAD CODE."""

    def __init__(self, table: str):
        self._table = table
        self._columns: List[str] = []
        self._where: List[str] = []
        self._limit: Optional[int] = None

    def select(self, *columns: str) -> 'QueryBuilder':
        """Add columns to select."""
        self._columns.extend(columns)
        return self

    def where(self, condition: str) -> 'QueryBuilder':
        """Add WHERE condition."""
        self._where.append(condition)
        return self

    def limit(self, n: int) -> 'QueryBuilder':
        """Set LIMIT."""
        self._limit = n
        return self

    def build(self) -> str:
        """Build SQL query."""
        sql = "SELECT "
        sql += ", ".join(self._columns) if self._columns else "*"
        sql += f" FROM {self._table}"

        if self._where:
            sql += " WHERE " + " AND ".join(self._where)

        if self._limit:
            sql += f" LIMIT {self._limit}"

        return sql


# ============================================================================
# Async functions - tests async/await extraction
# ============================================================================

async def async_fetch(url: str) -> bytes:
    """Async fetch function - DEAD CODE."""
    # Simulated async operation
    return b"response"


async def async_process(data: bytes) -> str:
    """Async process function - DEAD CODE."""
    return data.decode()


async def async_pipeline(url: str) -> str:
    """Async pipeline - DEAD CODE."""
    data = await async_fetch(url)
    result = await async_process(data)
    return result


# ============================================================================
# Generator functions - tests generator extraction
# ============================================================================

def generate_numbers(n: int):
    """Generator function - DEAD CODE."""
    for i in range(n):
        yield i


def generate_fibonacci(n: int):
    """Fibonacci generator - DEAD CODE."""
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b


# ============================================================================
# Private helpers - DEAD CODE
# ============================================================================

def _private_helper(x: int, y: int) -> int:
    """Private helper function - DEAD CODE."""
    return x + y


def _another_private() -> None:
    """Another private function - DEAD CODE."""
    pass
