package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "interview":
		runInterview(os.Args[1:])
	case "extract":
		runExtract(os.Args[1:])
	case "analyze-graph":
		runAnalyzeGraph(os.Args[1:])
	case "cache":
		runCache(os.Args[1:])
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Ontology Management CLI

A unified command-line tool for creating, refining, and extracting ontologies
from document corpora.

USAGE:
  ontology <command> [options]

COMMANDS:
  interview      Create or refine an ontology schema through LLM-guided interview
  extract        Extract entities and relationships using an ontology schema
  analyze-graph  Analyze extracted ontology as knowledge graph with community detection
  cache          Manage content cache (clear, stats)
  help           Show this help message
  version        Show version information

Run 'ontology <command> --help' for more information on a specific command.

EXAMPLES:
  # Create a new ontology
  ontology interview --config config.toml

  # Extract entities using ontology
  ontology extract --config config.toml --schema ontology.json --job-db extraction.db

  # Analyze extracted ontology as knowledge graph
  ontology analyze-graph --config config.toml --run-id extraction_12345 --output-gob graph.gob

  # View cache statistics
  ontology cache stats

  # Clear content cache
  ontology cache clear

ENVIRONMENT:
  ANTHROPIC_API_KEY    Required for 'interview' command

For detailed documentation, visit:
  https://github.com/go-doc-go/docs
`)
}

func printVersion() {
	fmt.Println("Ontology CLI v1.0.0")
	fmt.Println("Go-Doc-Go Document Analysis System")
}
