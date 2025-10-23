/**
 * Utility module providing string and array helper functions.
 */

/**
 * Capitalizes the first letter of a string.
 */
export function capitalize(str: string): string {
    if (!str) return str;
    return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Reverses a string.
 */
export function reverseString(str: string): string {
    return str.split('').reverse().join('');
}

/**
 * Checks if a string is a palindrome.
 */
export function isPalindrome(str: string): boolean {
    const cleaned = str.toLowerCase().replace(/[^a-z0-9]/g, '');
    return cleaned === cleaned.split('').reverse().join('');
}

/**
 * Returns the sum of all numbers in an array.
 */
export function sum(numbers: number[]): number {
    return numbers.reduce((acc, num) => acc + num, 0);
}

/**
 * Returns the average of all numbers in an array.
 */
export function average(numbers: number[]): number {
    if (numbers.length === 0) {
        throw new Error("Cannot calculate average of empty array");
    }
    return sum(numbers) / numbers.length;
}

/**
 * Removes duplicates from an array.
 */
export function unique<T>(arr: T[]): T[] {
    return Array.from(new Set(arr));
}
