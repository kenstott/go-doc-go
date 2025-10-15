package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/catalogs"
)

// exportCatalogs exports all built-in catalogs to YAML files
func main() {
	// Get output directory from args or use default
	outputDir := "../../examples/ontologies"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Get all domain names
	domains := catalogs.ListDomains()

	fmt.Printf("Exporting %d built-in catalogs to %s\n", len(domains), outputDir)

	// Export each catalog
	for _, domain := range domains {
		catalog, exists := catalogs.GetCatalog(domain)
		if !exists {
			log.Printf("Warning: Domain %s not found", domain)
			continue
		}

		outputPath := filepath.Join(outputDir, domain+".yaml")

		if err := catalogs.SaveToFile(catalog, outputPath); err != nil {
			log.Fatalf("Failed to export %s: %v", domain, err)
		}

		fmt.Printf("  ✓ Exported %s → %s\n", domain, outputPath)
	}

	fmt.Println("\nExport complete!")
	fmt.Println("\nTo use these catalogs, call:")
	fmt.Println("  catalogs.RegisterFromDirectory(\"./examples/ontologies\")")
}
