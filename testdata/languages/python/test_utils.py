"""
Test module - tests test function detection.
All functions here should be excluded from dead code analysis.
"""

import pytest
import unittest
from typing import List

from utils import process_data, hash_string, filter_items
from complex import (
    simple_function,
    nested_if_function,
    high_complexity_function
)


class TestProcessData(unittest.TestCase):
    """Test case for process_data - should be excluded."""

    def test_basic(self):
        """Test basic functionality."""
        result = process_data(["hello", "world"])
        self.assertEqual(result, "HELLO, WORLD")

    def test_empty(self):
        """Test empty list."""
        result = process_data([])
        self.assertEqual(result, "")

    def test_single_item(self):
        """Test single item."""
        result = process_data(["test"])
        self.assertEqual(result, "TEST")


class TestHashString(unittest.TestCase):
    """Test case for hash_string - should be excluded."""

    def test_hash_length(self):
        """Test hash output length."""
        result = hash_string("test")
        self.assertEqual(len(result), 64)

    def test_deterministic(self):
        """Test hash is deterministic."""
        result1 = hash_string("test")
        result2 = hash_string("test")
        self.assertEqual(result1, result2)


class TestFilterItems(unittest.TestCase):
    """Test case for filter_items - should be excluded."""

    def test_filter_positive(self):
        """Test filtering positive numbers."""
        items = [-1, 0, 1, 2, 3]
        result = filter_items(items, lambda x: x > 0)
        self.assertEqual(result, [1, 2, 3])


class TestComplexFunctions(unittest.TestCase):
    """Test case for complex functions."""

    def test_simple_function(self):
        """Test simple function."""
        self.assertEqual(simple_function(5), 10)

    def test_nested_if_function(self):
        """Test nested if function."""
        self.assertEqual(nested_if_function(1, 1), "both positive")
        self.assertEqual(nested_if_function(1, -1), "only x positive")
        self.assertEqual(nested_if_function(-1, 1), "x not positive")

    def test_high_complexity_function(self):
        """Test high complexity function."""
        self.assertEqual(high_complexity_function("add", 2, 3), 5)
        self.assertEqual(high_complexity_function("mul", 2, 3), 6)
        self.assertIsNone(high_complexity_function("div", 5, 0))


# ============================================================================
# Pytest style tests - should be excluded
# ============================================================================

def test_process_data_pytest():
    """Pytest style test for process_data."""
    result = process_data(["a", "b", "c"])
    assert result == "A, B, C"


def test_hash_string_pytest():
    """Pytest style test for hash_string."""
    result = hash_string("pytest")
    assert len(result) == 64


@pytest.fixture
def sample_data() -> List[str]:
    """Pytest fixture - should be excluded."""
    return ["apple", "banana", "cherry"]


@pytest.fixture
def config_data():
    """Another pytest fixture."""
    return {"host": "localhost", "port": 8080}


def test_with_fixture(sample_data):
    """Test using fixture."""
    result = process_data(sample_data)
    assert "APPLE" in result


@pytest.mark.parametrize("input_val,expected", [
    (5, 10),
    (0, 0),
    (-5, -10),
])
def test_simple_function_parametrized(input_val, expected):
    """Parametrized test - should be excluded."""
    assert simple_function(input_val) == expected


@pytest.mark.slow
def test_slow_operation():
    """Marked test - should be excluded."""
    result = process_data(["x"] * 1000)
    assert len(result) > 0


@pytest.mark.skip(reason="not implemented")
def test_skipped():
    """Skipped test - should be excluded."""
    pass


@pytest.mark.xfail(reason="known bug")
def test_expected_failure():
    """Expected failure test - should be excluded."""
    assert False


class TestWithSetup:
    """Test class with setup/teardown."""

    def setup_method(self):
        """Setup method - should be excluded."""
        self.data = ["a", "b", "c"]

    def teardown_method(self):
        """Teardown method - should be excluded."""
        self.data = None

    def test_with_setup_data(self):
        """Test using setup data."""
        result = process_data(self.data)
        assert result == "A, B, C"


# ============================================================================
# Test helpers - should be excluded
# ============================================================================

def _test_helper(name: str) -> None:
    """Private test helper function."""
    print(f"Running test: {name}")


def setup_test_environment() -> dict:
    """Setup test environment - should be excluded."""
    return {"env": "test"}


def teardown_test_environment(env: dict) -> None:
    """Teardown test environment - should be excluded."""
    env.clear()


def create_mock_data() -> List[str]:
    """Create mock data for tests - should be excluded."""
    return ["mock1", "mock2", "mock3"]


def assert_equal_lists(actual: List, expected: List) -> None:
    """Custom assertion helper - should be excluded."""
    assert len(actual) == len(expected)
    for a, e in zip(actual, expected):
        assert a == e


# ============================================================================
# Module-level test setup
# ============================================================================

def setup_module():
    """Module setup - should be excluded."""
    print("Setting up test module")


def teardown_module():
    """Module teardown - should be excluded."""
    print("Tearing down test module")


if __name__ == "__main__":
    unittest.main()
