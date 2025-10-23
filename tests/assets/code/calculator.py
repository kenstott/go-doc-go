"""
Calculator module providing basic mathematical operations.
"""

import math
from typing import Union


class Calculator:
    """A simple calculator class for basic arithmetic operations."""

    def add(self, a: Union[int, float], b: Union[int, float]) -> Union[int, float]:
        """Add two numbers."""
        return a + b

    def subtract(self, a: Union[int, float], b: Union[int, float]) -> Union[int, float]:
        """Subtract b from a."""
        return a - b

    def multiply(self, a: Union[int, float], b: Union[int, float]) -> Union[int, float]:
        """Multiply two numbers."""
        return a * b

    def divide(self, a: Union[int, float], b: Union[int, float]) -> float:
        """Divide a by b. Raises ValueError if b is zero."""
        if b == 0:
            raise ValueError("Cannot divide by zero")
        return a / b

    def power(self, base: Union[int, float], exponent: Union[int, float]) -> float:
        """Calculate base raised to exponent power."""
        return math.pow(base, exponent)

    def square_root(self, n: Union[int, float]) -> float:
        """Calculate the square root of n."""
        if n < 0:
            raise ValueError("Cannot calculate square root of negative number")
        return math.sqrt(n)
