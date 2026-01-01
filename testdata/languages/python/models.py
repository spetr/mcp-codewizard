"""
Models module - tests class definitions, dataclasses, protocols.
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Protocol, TypeVar, Generic

# Type variable for generics
T = TypeVar('T')


@dataclass
class Config:
    """Configuration dataclass - tests dataclass extraction."""
    host: str
    port: int
    log_level: str = "info"
    options: Dict[str, Any] = field(default_factory=dict)

    def validate(self) -> bool:
        """Validate configuration."""
        if not self.host:
            return False
        if self.port <= 0 or self.port > 65535:
            return False
        return True

    def clone(self) -> 'Config':
        """Create a copy of the config."""
        return Config(
            host=self.host,
            port=self.port,
            log_level=self.log_level,
            options=self.options.copy()
        )


class Server:
    """Server class - tests class with methods."""

    def __init__(self, config: Config):
        """Initialize server with config."""
        self.config = config
        self._running = False
        self._logger = Logger("server")

    def start(self) -> None:
        """Start the server - called from main, should be reachable."""
        self._running = True
        self._logger.info(f"Starting server on {self.config.host}:{self.config.port}")
        self._listen()

    def stop(self) -> None:
        """Stop the server - public method."""
        self._running = False
        self._logger.info("Stopping server")

    def _listen(self) -> None:
        """Internal listen method - called by start, should be reachable."""
        pass

    def _handle_connection(self, conn: Any) -> None:
        """Handle connection - DEAD CODE (never called)."""
        pass


class Logger:
    """Logger class - used by Server."""

    def __init__(self, prefix: str):
        self.prefix = prefix
        self.level = 1

    def info(self, message: str) -> None:
        """Log info message - called from Server.start."""
        print(f"[INFO] {self.prefix}: {message}")

    def debug(self, message: str) -> None:
        """Log debug message - DEAD CODE."""
        if self.level >= 2:
            print(f"[DEBUG] {self.prefix}: {message}")

    def error(self, message: str) -> None:
        """Log error message - DEAD CODE."""
        print(f"[ERROR] {self.prefix}: {message}")


def create_server(config: Config) -> Server:
    """Factory function for Server - called from main."""
    return Server(config)


# ============================================================================
# Protocol (structural subtyping) - tests Protocol extraction
# ============================================================================

class Handler(Protocol):
    """Handler protocol - tests Protocol extraction."""

    def handle(self, request: 'Request') -> 'Response':
        """Handle a request."""
        ...

    def name(self) -> str:
        """Return handler name."""
        ...


@dataclass
class Request:
    """Request dataclass."""
    method: str
    path: str
    body: bytes = b""


@dataclass
class Response:
    """Response dataclass."""
    status: int
    body: bytes = b""


class EchoHandler:
    """Echo handler implementing Handler protocol - DEAD CODE."""

    def __init__(self):
        self._name = "echo"

    def handle(self, request: Request) -> Response:
        """Handle request by echoing body."""
        return Response(status=200, body=request.body)

    def name(self) -> str:
        """Return handler name."""
        return self._name


# ============================================================================
# Abstract base class - tests ABC extraction
# ============================================================================

class BaseProcessor(ABC):
    """Abstract processor - tests ABC extraction."""

    def __init__(self, name: str):
        self.name = name

    @abstractmethod
    def process(self, data: bytes) -> bytes:
        """Process data - abstract method."""
        pass

    def pre_process(self, data: bytes) -> bytes:
        """Pre-process data - concrete method."""
        return data.strip()


class JsonProcessor(BaseProcessor):
    """JSON processor - implements BaseProcessor - DEAD CODE."""

    def process(self, data: bytes) -> bytes:
        """Process JSON data."""
        return b'{"processed": true}'


class XmlProcessor(BaseProcessor):
    """XML processor - implements BaseProcessor - DEAD CODE."""

    def process(self, data: bytes) -> bytes:
        """Process XML data."""
        return b'<processed>true</processed>'


# ============================================================================
# Generic class - tests generic type extraction
# ============================================================================

class Container(Generic[T]):
    """Generic container - tests generic class extraction."""

    def __init__(self):
        self._items: List[T] = []

    def add(self, item: T) -> None:
        """Add item to container."""
        self._items.append(item)

    def get(self, index: int) -> T:
        """Get item by index."""
        return self._items[index]

    def all(self) -> List[T]:
        """Get all items."""
        return self._items.copy()


# ============================================================================
# Unused classes - DEAD CODE
# ============================================================================

class UnusedClass:
    """Class that is never instantiated - DEAD CODE."""

    def __init__(self, value: str):
        self.value = value

    def process(self) -> None:
        """Process method - DEAD CODE."""
        print(f"Processing: {self.value}")


@dataclass
class UnusedDataclass:
    """Dataclass that is never used - DEAD CODE."""
    field1: str
    field2: int


# Type alias - tests type alias extraction
ConfigDict = Dict[str, Any]
StringList = List[str]
