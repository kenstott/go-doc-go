package main

import (
	"fmt"
	"log"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
)

func main() {
	xlsxParser := parser.NewXLSXParser()
	result, err := xlsxParser.Parse("test_sample.xlsx", "/Users/kennethstott/PycharmProjects/doculyzer-go-conversion/tests/assets/test_sample.xlsx")

	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Count element types
	elementTypes := make(map[string]int)
	for _, elem := range result.Elements {
		elementTypes[elem.ElementType]++
	}

	fmt.Println("=== Go XLSX Parser ===")
	fmt.Printf("Total elements: %d\n", len(result.Elements))
	fmt.Printf("Total relationships: %d\n", len(result.Relationships))
	fmt.Printf("Total links: %d\n", len(result.Links))
	fmt.Println("\nElement types:")
	for elemType, count := range elementTypes {
		fmt.Printf("  %s: %d\n", elemType, count)
	}

	// Show first 5 elements
	fmt.Println("\nFirst 5 elements:")
	for i, elem := range result.Elements {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s: %s\n", elem.ElementType, elem.ElementID)
	}
}
