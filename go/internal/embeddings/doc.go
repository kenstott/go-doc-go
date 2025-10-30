// Package embeddings provides vector embedding generation for semantic search and analysis.
//
// The embeddings package implements the EmbeddingGenerator interface for converting
// text into dense vector representations (embeddings) suitable for semantic similarity
// search, clustering, and machine learning applications.
//
// # Embedding Providers
//
// **ONNX (Recommended)**:
//   - Local ONNX model execution
//   - No external API dependencies
//   - Thread-safe with session pooling
//   - Fast inference (100-1000 embeddings/sec)
//   - Models: all-MiniLM-L6-v2 (384 dims, default), others
//
// # Usage
//
// Create an embedding generator:
//
//	config := &embeddings.Config{
//		Enabled:    true,
//		Provider:   "onnx",
//		Model:      "all-MiniLM-L6-v2",
//		ModelPath:  "./models/all-MiniLM-L6-v2",
//		Dimensions: 384,
//		BatchSize:  32,
//		PoolSize:   4,  // Match number of workers
//	}
//
//	generator, err := embeddings.NewEmbeddingGenerator(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer generator.Close()
//
// Generate embeddings:
//
//	// Single text
//	vec, err := generator.Generate("machine learning algorithms")
//	fmt.Printf("Embedding: %d dimensions\n", len(vec))
//
//	// Batch processing (more efficient)
//	texts := []string{
//		"neural networks",
//		"deep learning",
//		"artificial intelligence",
//	}
//	vectors, err := generator.GenerateBatch(texts)
//	fmt.Printf("Generated %d vectors\n", len(vectors))
//
// # Contextual Text Building
//
// The ContextualTextBuilder enriches element text with surrounding context
// for better embeddings:
//
//	builder := embeddings.NewContextualTextBuilder()
//	contextualText := builder.BuildContextualText(
//		element,           // Current element
//		parentElements,    // Parent elements for context
//		siblingElements,   // Sibling elements for context
//	)
//
// **Context Strategy**:
//   - Include parent content (e.g., section headings)
//   - Include N predecessor siblings
//   - Include N successor siblings
//   - Respects max token limits
//
// Example contextual text:
//
//	Section: Machine Learning
//	Previous: Neural networks are computational models...
//	Current: Deep learning uses multiple layers...
//	Next: Training requires large datasets...
//
// This produces better embeddings than isolated element text.
//
// # Configuration
//
// TOML configuration:
//
//	[embedding]
//	enabled = true
//	provider = "onnx"
//	model = "all-MiniLM-L6-v2"
//	model_path = "./models/all-MiniLM-L6-v2"
//	dimensions = 384
//	batch_size = 32
//	pool_size = 4
//	contextual = true
//	predecessor_count = 1
//	successor_count = 1
//
// **Configuration Options**:
//   - enabled: Enable embedding generation
//   - provider: Embedding provider ("onnx")
//   - model: Model name
//   - model_path: Path to ONNX model files
//   - dimensions: Embedding vector dimensions
//   - batch_size: Batch size for inference
//   - pool_size: ONNX session pool size (should match worker count)
//   - contextual: Include context from parent/siblings
//   - predecessor_count: Number of previous siblings to include
//   - successor_count: Number of following siblings to include
//
// # ONNX Model Setup
//
// Download and setup the ONNX model:
//
//	# Create model directory
//	mkdir -p models/all-MiniLM-L6-v2
//
//	# Download model files (model.onnx, tokenizer.json, config.json)
//	# From Hugging Face or other source
//
//	# Set model_path in config
//	model_path = "./models/all-MiniLM-L6-v2"
//
// Required files:
//   - model.onnx: ONNX model file
//   - tokenizer.json: Tokenizer configuration
//   - config.json: Model configuration
//
// # Batch Processing
//
// Batch processing is significantly faster than individual generation:
//
//	texts := make([]string, 1000)
//	// ... populate texts
//
//	// Efficient: 1 batch call
//	vectors, err := generator.GenerateBatch(texts)
//
//	// Inefficient: 1000 individual calls
//	for _, text := range texts {
//		vec, _ := generator.Generate(text)  // Slow!
//	}
//
// Batch processing amortizes:
//   - Tokenization overhead
//   - Model inference overhead
//   - Memory allocation
//
// **Recommended batch sizes**:
//   - Small texts (< 100 tokens): batch_size = 64
//   - Medium texts (100-500 tokens): batch_size = 32
//   - Large texts (> 500 tokens): batch_size = 16
//
// # Thread Safety
//
// ONNX generator uses session pooling for thread safety:
//
//	// Multiple goroutines can use the same generator
//	var wg sync.WaitGroup
//	for i := 0; i < 10; i++ {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			vec, _ := generator.Generate("concurrent text")
//		}()
//	}
//	wg.Wait()
//
// Pool size should match worker count:
//   - pool_size = 4 workers → 4 ONNX sessions
//   - Each session handles one request at a time
//   - Requests queue if all sessions busy
//
// # Performance Characteristics
//
// **ONNX Provider**:
//   - Throughput: 100-1000 embeddings/second (depends on text length, batch size)
//   - Latency: 10-100ms per batch of 32 texts
//   - Memory: ~500MB for model + ~50MB per session
//   - CPU: Uses all available cores
//
// **Optimization Tips**:
//   - Use batch processing (GenerateBatch vs. Generate)
//   - Set batch_size to match hardware (CPU cores, memory)
//   - Set pool_size to match worker count
//   - Use shorter contextual text (< 512 tokens)
//   - Preload models at startup
//
// # Embedding Dimensions
//
// Common embedding models:
//
//   - all-MiniLM-L6-v2: 384 dimensions (default, balanced)
//   - all-mpnet-base-v2: 768 dimensions (higher quality, slower)
//   - paraphrase-multilingual: 384 dimensions (multilingual)
//
// Higher dimensions:
//   - Better semantic representation
//   - Higher storage cost
//   - Slower similarity computations
//
// # Similarity Search
//
// Embeddings enable semantic similarity search:
//
//	queryVec, _ := generator.Generate("machine learning")
//
//	// Cosine similarity
//	similarity := cosineSimilarity(queryVec, docVec)
//	if similarity > 0.7 {  // 70% similar
//		fmt.Println("Semantically similar!")
//	}
//
// Similarity ranges:
//   - 0.9-1.0: Nearly identical
//   - 0.7-0.9: Highly similar
//   - 0.5-0.7: Moderately similar
//   - < 0.5: Weakly similar or unrelated
//
// # Storage Integration
//
// Embeddings are stored in the UDML-V layer:
//
//	embeddings := []analytics.Embedding{
//		{
//			ElementID:  "elem-123",
//			DocID:      "doc-456",
//			SourceName: "wikipedia",
//			Embedding:  vector,
//			Text:       contextualText,
//		},
//	}
//
//	err := storage.AppendEmbeddings(embeddings)
//
// Stored embeddings enable:
//   - Semantic search via SearchSemanticSimilarity
//   - Document clustering
//   - Similarity-based relationship detection
//   - Recommendation systems
//
// # Error Handling
//
// Embedding generation returns errors for:
//
//   - Model loading failures
//   - Invalid model path
//   - Out of memory
//   - Text too long (> max tokens)
//   - ONNX runtime errors
//
// Errors are wrapped with context using fmt.Errorf with %w.
//
// # Resource Management
//
// Properly close generators to release resources:
//
//	generator, err := embeddings.NewEmbeddingGenerator(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer generator.Close()  // Important!
//
// Close() releases:
//   - ONNX sessions
//   - Model memory
//   - Thread pools
//
// # Factory Pattern
//
// Create generators dynamically:
//
//	config := &embeddings.Config{
//		Provider: "onnx",
//		// ... other config
//	}
//
//	generator, err := embeddings.NewEmbeddingGenerator(config)
//
// The factory selects the appropriate implementation based on provider.
//
// # See Also
//
//   - Analytics Storage: go/internal/analytics/ (embedding persistence)
//   - Worker: go/internal/worker/ (embedding generation pipeline)
//   - Configuration: docs/configuration/README.md (embedding section)
//   - Semantic Search: docs/features/embeddings/ (usage guide)
//   - Examples: examples/semantic-search/
package embeddings
