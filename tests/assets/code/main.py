"""
Main application demonstrating Calculator usage.
"""

from calculator import Calculator


def main():
    """Main entry point for the calculator application."""
    calc = Calculator()

    print("Calculator Demo")
    print("=" * 40)

    # Perform calculations
    print(f"5 + 3 = {calc.add(5, 3)}")
    print(f"10 - 4 = {calc.subtract(10, 4)}")
    print(f"7 * 6 = {calc.multiply(7, 6)}")
    print(f"20 / 4 = {calc.divide(20, 4)}")
    print(f"2^8 = {calc.power(2, 8)}")
    print(f"√16 = {calc.square_root(16)}")

    # Error handling example
    try:
        result = calc.divide(10, 0)
    except ValueError as e:
        print(f"Error: {e}")


if __name__ == "__main__":
    main()
