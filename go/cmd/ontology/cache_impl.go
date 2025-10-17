package main

import (
	"fmt"
	"os"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
)

// runCache manages the disk-based content cache
func runCache(args []string) {
	if len(args) < 2 {
		printCacheUsage()
		os.Exit(1)
	}

	subcommand := args[1]

	// Initialize cache
	config := parser.DefaultCacheConfig()
	cache, err := parser.NewContentCache(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	switch subcommand {
	case "stats":
		count, size, err := cache.GetStats()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting cache stats: %v\n", err)
			os.Exit(1)
		}

		sizeMB := float64(size) / (1024 * 1024)
		fmt.Printf("Content Cache Statistics:\n")
		fmt.Printf("  Cache directory: %s\n", config.CacheDir)
		fmt.Printf("  Cached items: %d\n", count)
		fmt.Printf("  Total size: %.2f MB\n", sizeMB)
		fmt.Printf("  Max size: %d MB\n", config.MaxCacheSizeMB)
		fmt.Printf("  Max item size: %d MB\n", config.MaxItemSizeMB)

	case "clear":
		if err := cache.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Cache cleared successfully")

	case "help", "--help", "-h":
		printCacheUsage()

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown cache subcommand '%s'\n\n", subcommand)
		printCacheUsage()
		os.Exit(1)
	}
}

func printCacheUsage() {
	fmt.Fprintf(os.Stderr, `Manage Content Cache

USAGE:
  ontology cache <subcommand>

SUBCOMMANDS:
  stats    Show cache statistics (size, item count, location)
  clear    Clear all cached content
  help     Show this help message

EXAMPLES:
  # View cache statistics
  ontology cache stats

  # Clear cache
  ontology cache clear

The content cache stores fetched web content (HTML, etc.) to avoid
repeated downloads. It uses an LRU eviction policy with configurable
size limits (default: 500MB max cache, 10MB max item).

Cache location: $HOME/.cache/go-doc-go/content/
`)
}
