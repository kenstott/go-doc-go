/**
 * Main entry point demonstrating helper functions.
 */

const { formatDate, isValidEmail, randomNumber, truncate } = require('./helpers');

function main() {
    console.log("Helper Functions Demo");
    console.log("=".repeat(40));

    // Date formatting
    const today = new Date();
    console.log(`Today's date: ${formatDate(today)}`);

    // Email validation
    console.log(`isValidEmail("test@example.com") = ${isValidEmail("test@example.com")}`);
    console.log(`isValidEmail("invalid.email") = ${isValidEmail("invalid.email")}`);

    // Random number generation
    console.log(`Random number (1-100): ${randomNumber(1, 100)}`);
    console.log(`Random number (1-100): ${randomNumber(1, 100)}`);

    // String truncation
    const longString = "This is a very long string that needs to be truncated";
    console.log(`truncate("${longString}", 30) = ${truncate(longString, 30)}`);
}

main();
