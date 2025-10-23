/**
 * Main application demonstrating MathUtils usage.
 */

package code;

import code.MathUtils;
import java.util.Arrays;

public class Main {

    public static void main(String[] args) {
        System.out.println("MathUtils Demo");
        System.out.println("=".repeat(40));

        // Factorial calculation
        int n = 6;
        long factorial = MathUtils.factorial(n);
        System.out.printf("Factorial of %d: %d%n", n, factorial);

        // Prime number check
        int[] testNumbers = {2, 7, 15, 23, 100};
        System.out.println("\nPrime number checks:");
        for (int num : testNumbers) {
            boolean isPrime = MathUtils.isPrime(num);
            System.out.printf("  %d is %s%n", num, isPrime ? "prime" : "not prime");
        }

        // GCD calculation
        int a = 48, b = 18;
        int gcd = MathUtils.gcd(a, b);
        System.out.printf("\nGCD of %d and %d: %d%n", a, b, gcd);

        // Array operations
        int[] numbers = {5, 12, 8, 23, 17, 3, 9, 14};
        System.out.println("\nArray operations on: " + Arrays.toString(numbers));

        double avg = MathUtils.average(numbers);
        System.out.printf("  Average: %.2f%n", avg);

        int max = MathUtils.max(numbers);
        System.out.printf("  Maximum: %d%n", max);

        // Power calculation
        double base = 2.5;
        int exponent = 3;
        double result = MathUtils.power(base, exponent);
        System.out.printf("\n%.1f raised to the power of %d: %.3f%n", base, exponent, result);
    }
}
