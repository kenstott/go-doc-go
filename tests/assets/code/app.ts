/**
 * Main application demonstrating utility functions.
 */

import { capitalize, reverseString, isPalindrome, sum, average, unique } from './utils';

function main(): void {
    console.log("Utility Functions Demo");
    console.log("=".repeat(40));

    // String operations
    console.log(`capitalize("hello") = ${capitalize("hello")}`);
    console.log(`reverseString("TypeScript") = ${reverseString("TypeScript")}`);
    console.log(`isPalindrome("racecar") = ${isPalindrome("racecar")}`);
    console.log(`isPalindrome("hello") = ${isPalindrome("hello")}`);

    // Array operations
    const numbers = [1, 2, 3, 4, 5];
    console.log(`sum([1, 2, 3, 4, 5]) = ${sum(numbers)}`);
    console.log(`average([1, 2, 3, 4, 5]) = ${average(numbers)}`);

    const duplicates = [1, 2, 2, 3, 3, 3, 4, 5, 5];
    console.log(`unique([1, 2, 2, 3, 3, 3, 4, 5, 5]) = ${unique(duplicates)}`);
}

main();
