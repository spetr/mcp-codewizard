#!/usr/bin/env python3
"""
Main module demonstrating various Python patterns for parser testing.
Tests: entry points, imports, function calls, decorators.
"""

import os
import sys
from typing import List, Optional

from utils import process_data, format_output
from models import Config, Server, create_server

# Module-level constants - tests constant extraction
MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30

# Module-level variables - tests variable extraction
app_name = "TestApp"
_internal_version = "1.0.0"

# Module-level initialization - tests package-level reference tracking
logger = create_logger("main")


def main() -> None:
    """Main entry point - should be marked as reachable."""
    print(f"Starting {app_name}")

    # Function calls - tests reference extraction
    config = load_config()
    if not initialize(config):
        print("Initialization failed", file=sys.stderr)
        sys.exit(1)

    # Method calls
    server = create_server(config)
    server.start()

    # Using imported function
    result = process_data(["a", "b", "c"])
    print(format_output(result))

    # Calling transitive functions
    run_pipeline()


def load_config() -> Config:
    """Load configuration - called from main, should be reachable."""
    return Config(
        host="localhost",
        port=8080,
        log_level="info"
    )


def initialize(config: Config) -> bool:
    """Initialize application - called from main, should be reachable."""
    if config is None:
        return False
    _setup_logging(config.log_level)
    return True


def _setup_logging(level: str) -> None:
    """Internal helper - called from initialize, should be reachable."""
    print(f"Setting log level to: {level}")


def run_pipeline() -> None:
    """Orchestrate data pipeline - tests transitive reachability."""
    data = _fetch_data()
    transformed = _transform_data(data)
    _save_data(transformed)


def _fetch_data() -> bytes:
    """Fetch data - called by run_pipeline, should be reachable."""
    return b"sample data"


def _transform_data(data: bytes) -> bytes:
    """Transform data - called by run_pipeline, should be reachable."""
    return b"transformed: " + data


def _save_data(data: bytes) -> None:
    """Save data - called by run_pipeline, should be reachable."""
    print(f"Saving: {data.decode()}")


def create_logger(name: str):
    """Create a logger - called from module level, should be reachable."""
    return Logger(name)


class Logger:
    """Simple logger class."""

    def __init__(self, name: str):
        self.name = name

    def info(self, message: str) -> None:
        print(f"[INFO] {self.name}: {message}")


# ============================================================================
# Dead code section - functions that are never called
# ============================================================================

def unused_function() -> None:
    """This function is never called - DEAD CODE."""
    print("This is never executed")


def another_unused() -> str:
    """Also never called - DEAD CODE."""
    return "dead"


def dead_chain_start() -> None:
    """Starts a chain of dead code - DEAD CODE."""
    dead_chain_middle()


def dead_chain_middle() -> None:
    """In the middle of dead chain - DEAD CODE (transitive)."""
    dead_chain_end()


def dead_chain_end() -> None:
    """End of dead chain - DEAD CODE (transitive)."""
    print("End of dead chain")


def _private_unused() -> None:
    """Private unused function - DEAD CODE."""
    pass


# ============================================================================
# Entry point guard
# ============================================================================

if __name__ == "__main__":
    main()
