"""
Complex module - tests complexity analysis patterns.
Contains functions with various cyclomatic complexity levels.
"""

from typing import Any, Dict, List, Optional


def simple_function(x: int) -> int:
    """Simple function - complexity 1, no branches."""
    return x * 2


def linear_function(a: int, b: int, c: int) -> int:
    """Linear function - complexity 1, sequential statements."""
    total = a + b
    total += c
    result = total * 2
    return result


def single_if_function(x: int) -> str:
    """Single if - complexity 2."""
    if x > 0:
        return "positive"
    return "non-positive"


def if_else_function(x: int) -> str:
    """If/else - complexity 2."""
    if x > 0:
        return "positive"
    else:
        return "non-positive"


def if_elif_else_function(x: int) -> str:
    """If/elif/else - complexity 3."""
    if x > 0:
        return "positive"
    elif x < 0:
        return "negative"
    else:
        return "zero"


def multiple_if_function(a: bool, b: bool, c: bool) -> int:
    """Multiple independent ifs - complexity 4."""
    count = 0
    if a:
        count += 1
    if b:
        count += 1
    if c:
        count += 1
    return count


def nested_if_function(x: int, y: int) -> str:
    """Nested ifs - complexity 3, nesting 2."""
    if x > 0:
        if y > 0:
            return "both positive"
        return "only x positive"
    return "x not positive"


def deeply_nested_function(a: bool, b: bool, c: bool, d: bool) -> str:
    """Deeply nested - complexity 5, nesting 4."""
    if a:
        if b:
            if c:
                if d:
                    return "all true"
                return "d false"
            return "c false"
        return "b false"
    return "a false"


def match_function(day: int) -> str:
    """Match statement (Python 3.10+) - complexity varies by cases."""
    match day:
        case 1:
            return "Monday"
        case 2:
            return "Tuesday"
        case 3:
            return "Wednesday"
        case 4 | 5:
            return "Thursday or Friday"
        case _:
            return "Weekend"


def for_loop_function(n: int) -> int:
    """For loop - complexity 2, nesting 1."""
    total = 0
    for i in range(n):
        total += i
    return total


def for_each_function(items: List[int]) -> int:
    """For-each loop - complexity 2, nesting 1."""
    total = 0
    for item in items:
        total += item
    return total


def while_loop_function(n: int) -> int:
    """While loop - complexity 2."""
    total = 0
    i = 0
    while i < n:
        total += i
        i += 1
    return total


def nested_loop_function(n: int, m: int) -> int:
    """Nested loops - complexity 3, nesting 2."""
    total = 0
    for i in range(n):
        for j in range(m):
            total += i * j
    return total


def loop_with_condition(items: List[int]) -> int:
    """Loop with condition - complexity 3, nesting 2."""
    total = 0
    for item in items:
        if item > 0:
            total += item
    return total


def loop_with_break(items: List[int], target: int) -> int:
    """Loop with break - complexity 3."""
    for i, item in enumerate(items):
        if item == target:
            return i
    return -1


def loop_with_continue(items: List[int]) -> int:
    """Loop with continue - complexity 3."""
    total = 0
    for item in items:
        if item < 0:
            continue
        total += item
    return total


def logical_and_function(a: bool, b: bool) -> bool:
    """Logical AND - complexity 2 (short-circuit)."""
    return a and b


def logical_or_function(a: bool, b: bool) -> bool:
    """Logical OR - complexity 2 (short-circuit)."""
    return a or b


def complex_logical_function(a: bool, b: bool, c: bool) -> bool:
    """Complex logical - complexity 4."""
    return (a and b) or (b and c) or (a and c)


def ternary_function(x: int) -> str:
    """Ternary operator - complexity 2."""
    return "positive" if x > 0 else "non-positive"


def try_except_function(x: int) -> int:
    """Try/except - complexity 2."""
    try:
        return 100 // x
    except ZeroDivisionError:
        return 0


def try_multiple_except(x: Any) -> int:
    """Try with multiple except - complexity 4."""
    try:
        return int(x)
    except ValueError:
        return 0
    except TypeError:
        return -1
    except Exception:
        return -2


def try_except_finally(x: int) -> int:
    """Try/except/finally - complexity 2."""
    result = 0
    try:
        result = 100 // x
    except ZeroDivisionError:
        result = 0
    finally:
        print("cleanup")
    return result


def with_statement_function(filename: str) -> str:
    """With statement - complexity 1 (no branching)."""
    with open(filename) as f:
        return f.read()


def comprehension_with_condition(items: List[int]) -> List[int]:
    """List comprehension with condition - complexity 2."""
    return [x for x in items if x > 0]


def nested_comprehension(matrix: List[List[int]]) -> List[int]:
    """Nested comprehension - complexity 3."""
    return [x for row in matrix for x in row if x > 0]


def generator_expression_function(items: List[int]) -> int:
    """Generator expression - complexity 2."""
    return sum(x for x in items if x > 0)


def high_complexity_function(
    op: str,
    a: int,
    b: int
) -> Optional[int]:
    """High complexity function - complexity >= 10."""
    if op == "add":
        return a + b
    elif op == "sub":
        return a - b
    elif op == "mul":
        return a * b
    elif op == "div":
        if b == 0:
            return None
        return a // b
    elif op == "mod":
        if b == 0:
            return None
        return a % b
    elif op == "pow":
        if b < 0:
            return None
        result = 1
        for _ in range(b):
            result *= a
        return result
    elif op == "max":
        return a if a > b else b
    elif op == "min":
        return a if a < b else b
    else:
        return None


def extremely_complex_function(
    category: str,
    items: List[Dict[str, Any]],
    threshold: int
) -> List[Dict[str, Any]]:
    """Extremely complex function - complexity >= 15, nesting >= 4."""
    if not items:
        raise ValueError("empty items")

    result = []
    for item in items:
        if item.get("category") != category:
            continue

        status = item.get("status")
        if status == "active":
            if item.get("value", 0) > threshold:
                if item.get("priority") == "high":
                    result.append(item)
                elif item.get("priority") == "medium":
                    if item.get("value", 0) > threshold * 2:
                        result.append(item)
        elif status == "pending":
            if item.get("created_at"):
                for tag in item.get("tags", []):
                    if tag == "important":
                        result.append(item)
                        break
        elif status == "archived":
            continue
        else:
            if item.get("value", 0) > 0 and item.get("priority"):
                result.append(item)

    if not result:
        raise ValueError(f"no items found for category {category}")

    return result


async def async_complex_function(
    urls: List[str],
    timeout: int
) -> List[str]:
    """Async complex function - complexity >= 5."""
    results = []
    for url in urls:
        if not url:
            continue
        try:
            # Simulated async operation
            if timeout > 0:
                results.append(f"fetched: {url}")
            else:
                results.append(f"timeout: {url}")
        except Exception as e:
            results.append(f"error: {e}")
    return results


class ComplexClass:
    """Class with complex methods for testing."""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self._initialized = False

    def complex_method(
        self,
        items: List[int],
        mode: str
    ) -> int:
        """Complex method - high complexity."""
        if not self._initialized:
            return -1

        total = 0
        for item in items:
            if mode == "sum":
                total += item
            elif mode == "product":
                if total == 0:
                    total = 1
                total *= item
            elif mode == "max":
                if item > total:
                    total = item
            elif mode == "min":
                if total == 0 or item < total:
                    total = item
            else:
                continue

        return total
