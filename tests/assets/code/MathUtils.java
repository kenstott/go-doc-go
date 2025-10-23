/**
 * MathUtils class providing mathematical utility functions.
 */

package code;

import java.util.Arrays;

public class MathUtils {

    /**
     * Calculate the factorial of a number.
     * @param n The number to calculate factorial for
     * @return The factorial of n
     */
    public static long factorial(int n) {
        if (n < 0) {
            throw new IllegalArgumentException("Factorial is not defined for negative numbers");
        }
        if (n == 0 || n == 1) {
            return 1;
        }
        long result = 1;
        for (int i = 2; i <= n; i++) {
            result *= i;
        }
        return result;
    }

    /**
     * Check if a number is prime.
     * @param n The number to check
     * @return true if n is prime, false otherwise
     */
    public static boolean isPrime(int n) {
        if (n <= 1) {
            return false;
        }
        if (n <= 3) {
            return true;
        }
        if (n % 2 == 0 || n % 3 == 0) {
            return false;
        }
        for (int i = 5; i * i <= n; i += 6) {
            if (n % i == 0 || n % (i + 2) == 0) {
                return false;
            }
        }
        return true;
    }

    /**
     * Calculate the greatest common divisor of two numbers.
     * @param a First number
     * @param b Second number
     * @return GCD of a and b
     */
    public static int gcd(int a, int b) {
        a = Math.abs(a);
        b = Math.abs(b);
        while (b != 0) {
            int temp = b;
            b = a % b;
            a = temp;
        }
        return a;
    }

    /**
     * Calculate the average of an array of numbers.
     * @param numbers Array of integers
     * @return The average as a double
     */
    public static double average(int[] numbers) {
        if (numbers == null || numbers.length == 0) {
            throw new IllegalArgumentException("Array cannot be null or empty");
        }
        return Arrays.stream(numbers).average().orElse(0.0);
    }

    /**
     * Find the maximum value in an array.
     * @param numbers Array of integers
     * @return The maximum value
     */
    public static int max(int[] numbers) {
        if (numbers == null || numbers.length == 0) {
            throw new IllegalArgumentException("Array cannot be null or empty");
        }
        return Arrays.stream(numbers).max().orElse(Integer.MIN_VALUE);
    }

    /**
     * Calculate the power of a number.
     * @param base The base number
     * @param exponent The exponent
     * @return base raised to the power of exponent
     */
    public static double power(double base, int exponent) {
        return Math.pow(base, exponent);
    }
}
