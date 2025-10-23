// Package parser provides text parsing utilities.
package parser

import (
	"fmt"
	"strings"
)

// Token represents a parsed token with type and value.
type Token struct {
	Type  string
	Value string
	Line  int
	Col   int
}

// ParseResult holds the result of parsing operations.
type ParseResult struct {
	Tokens []Token
	Errors []error
}

// Tokenize splits input text into tokens.
func Tokenize(input string) []Token {
	var tokens []Token
	lines := strings.Split(input, "\n")

	for lineNum, line := range lines {
		words := strings.Fields(line)
		col := 0
		for _, word := range words {
			col = strings.Index(line[col:], word) + col
			token := Token{
				Type:  classifyToken(word),
				Value: word,
				Line:  lineNum + 1,
				Col:   col + 1,
			}
			tokens = append(tokens, token)
			col += len(word)
		}
	}

	return tokens
}

// classifyToken determines the type of a token.
func classifyToken(word string) string {
	if strings.HasPrefix(word, "\"") && strings.HasSuffix(word, "\"") {
		return "STRING"
	}
	if _, err := fmt.Sscanf(word, "%d", new(int)); err == nil {
		return "NUMBER"
	}
	if strings.ContainsAny(word, "+-*/=") {
		return "OPERATOR"
	}
	if strings.ContainsAny(word, "(){},;") {
		return "PUNCTUATION"
	}
	return "IDENTIFIER"
}

// CountWords returns the number of words in the input.
func CountWords(input string) int {
	return len(strings.Fields(input))
}

// CountLines returns the number of lines in the input.
func CountLines(input string) int {
	if input == "" {
		return 0
	}
	return len(strings.Split(input, "\n"))
}

// FindPattern searches for a pattern in the input text.
func FindPattern(input, pattern string) []int {
	var positions []int
	searchIn := input
	offset := 0

	for {
		index := strings.Index(searchIn, pattern)
		if index == -1 {
			break
		}
		positions = append(positions, offset+index)
		offset += index + len(pattern)
		searchIn = searchIn[index+len(pattern):]
	}

	return positions
}

// Validate checks if input meets basic criteria.
func Validate(input string) error {
	if input == "" {
		return fmt.Errorf("input cannot be empty")
	}
	if len(input) > 10000 {
		return fmt.Errorf("input exceeds maximum length of 10000 characters")
	}
	return nil
}
