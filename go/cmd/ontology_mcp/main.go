package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/mcp"
	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <parquet_path> [model_path]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Starts an MCP server for exploring UDML corpus during ontology generation.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  parquet_path   Path to UDML Parquet storage directory\n")
		fmt.Fprintf(os.Stderr, "  model_path     Path to ONNX embedding model (optional, for semantic search)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  %s ./output/udml.parquet ./assets/all-MiniLM-L6-v2-onnx\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "MCP Tools Provided:\n")
		fmt.Fprintf(os.Stderr, "  - search_corpus: Semantic/keyword/regex search\n")
		fmt.Fprintf(os.Stderr, "  - analyze_patterns: Regex pattern analysis\n")
		fmt.Fprintf(os.Stderr, "  - compute_frequencies: Term frequency counting\n")
		fmt.Fprintf(os.Stderr, "  - find_cooccurrences: Entity co-occurrence analysis\n")
		fmt.Fprintf(os.Stderr, "  - get_element_context: Retrieve element context\n")
		fmt.Fprintf(os.Stderr, "  - aggregate_statistics: Corpus statistics\n")
		os.Exit(1)
	}

	parquetPath := os.Args[1]

	// Initialize querier
	querier, err := query.NewQuerier(parquetPath)
	if err != nil {
		log.Fatalf("Failed to create querier: %v", err)
	}
	defer querier.Close()

	// Initialize embedding generator (optional)
	var embGen embeddings.EmbeddingGenerator
	if len(os.Args) > 2 {
		modelPath := os.Args[2]
		log.Printf("Loading embedding model from: %s", modelPath)

		embGen, err = embeddings.NewONNXEmbedding(modelPath, embeddings.ModelConfig{
			ModelName: "all-MiniLM-L6-v2",
		})
		if err != nil {
			log.Printf("Warning: Failed to load embedding model: %v", err)
			log.Printf("Semantic search will not be available")
		} else {
			defer embGen.Close()
			log.Printf("Embedding model loaded successfully (dimensions: %d)", embGen.GetDimensions())
		}
	}

	// Create MCP corpus explorer
	explorer := mcp.NewOntologyCorpusExplorer(querier, embGen)
	mcpServer := explorer.CreateMCPServer()

	// Start server on stdio
	log.Println("Starting Ontology Corpus Explorer MCP server...")
	log.Println("Server ready to accept tool calls from LLM client")

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
