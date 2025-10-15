package worker

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
	"github.com/kennethstott/doculyzer-go-conversion/internal/contentsource"
	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/jobcontrol"
	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/kennethstott/doculyzer-go-conversion/internal/resolver"
)

// EmbeddingBatchItem represents a single embedding request
type EmbeddingBatchItem struct {
	DocID     string
	ElementID string
	Text      string
}

// EmbeddingBatchAggregator collects embeddings across documents for efficient batching
type EmbeddingBatchAggregator struct {
	mu            sync.Mutex
	items         []EmbeddingBatchItem
	itemIndex     map[string]int // ElementID -> index in items
	maxBatchSize  int
	generator     embeddings.EmbeddingGenerator
	flushCallback func(map[string][]float64) // Called when batch is flushed
}

// Worker represents a document processing worker
type Worker struct {
	workerID             string
	numWorkers           int
	jobControl           jobcontrol.JobControl
	contentSources       map[string]contentsource.ContentSource
	contentSourceConfigs map[string]map[string]interface{}
	analyticsStorages    []analytics.Storage
	embeddingGenerator   embeddings.EmbeddingGenerator // Native ONNX generator (thread-safe)
	contextualBuilder    *embeddings.ContextualTextBuilder
	embeddingAggregator  *EmbeddingBatchAggregator      // Cross-document batch aggregator
	maxDocuments         int
	documentsProcessed   int32 // atomic counter for goroutine safety
	isLeader             bool
	running              bool
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup

	// Neo4j export state
	neo4jExportConfig     *Neo4jExportConfig
	lastQueueEmptyTime    *time.Time
	lastNeo4jExportTime   *time.Time

	// Semantic analysis state
	semanticConfig           *SemanticConfig
	semanticAnalyzer         *SemanticAnalyzer
	semanticAnalysisRunning  atomic.Bool
	lastSemanticAnalysisTime *time.Time

	// Ontology extraction state
	ontologyConfig            *OntologyConfig
	ontologyExtractor         *OntologyExtractor
	ontologyExtractionRunning atomic.Bool
	lastOntologyExtractionTime *time.Time
}

// Neo4jExportConfig holds Neo4j export configuration
type Neo4jExportConfig struct {
	Enabled            bool
	EmptyQueueWaitTime int    // Seconds to wait before triggering export
	SourceAnalytics    string // Source analytics type (e.g., "parquet")
	SourcePath         string // Path to source analytics
	Connection         map[string]interface{}
	BatchSize          int
}

// NewEmbeddingBatchAggregator creates a new batch aggregator
func NewEmbeddingBatchAggregator(maxBatchSize int, generator embeddings.EmbeddingGenerator) *EmbeddingBatchAggregator {
	return &EmbeddingBatchAggregator{
		items:        make([]EmbeddingBatchItem, 0, maxBatchSize),
		itemIndex:    make(map[string]int),
		maxBatchSize: maxBatchSize,
		generator:    generator,
	}
}

// Add adds an embedding request to the batch
func (a *EmbeddingBatchAggregator) Add(docID, elementID, text string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add to batch
	a.items = append(a.items, EmbeddingBatchItem{
		DocID:     docID,
		ElementID: elementID,
		Text:      text,
	})
	a.itemIndex[elementID] = len(a.items) - 1
}

// Flush processes all accumulated embeddings and returns results
func (a *EmbeddingBatchAggregator) Flush() (map[string][]float64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.items) == 0 {
		return make(map[string][]float64), nil
	}

	log.Printf("CROSS-DOC BATCH: Flushing %d embeddings across multiple documents", len(a.items))

	// Extract texts
	texts := make([]string, len(a.items))
	for i, item := range a.items {
		texts[i] = item.Text
	}

	// Generate embeddings in one or more batches
	startTime := time.Now()
	vectors, err := a.generator.GenerateBatch(texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	duration := time.Since(startTime)

	log.Printf("CROSS-DOC BATCH: Generated %d embeddings in %v (%.2f embeddings/sec)",
		len(vectors), duration, float64(len(vectors))/duration.Seconds())

	// Build result map
	result := make(map[string][]float64, len(a.items))
	for i, item := range a.items {
		if i < len(vectors) {
			result[item.ElementID] = vectors[i]
		}
	}

	// Clear batch
	a.items = a.items[:0]
	a.itemIndex = make(map[string]int)

	return result, nil
}

// ShouldFlush returns true if the batch is full
func (a *EmbeddingBatchAggregator) ShouldFlush() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.items) >= a.maxBatchSize
}

// Size returns current batch size
func (a *EmbeddingBatchAggregator) Size() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.items)
}

// SemanticConfig holds configuration for cross-document semantic relationship detection
type SemanticConfig struct {
	Enabled                  bool
	SimilarityThreshold      float64
	QueueIdleTriggerMinutes  int
	RateLimitBatchSize       int
	RateLimitSleepMs         int
	MinIntervalMinutes       int
}

// OntologyConfig holds configuration for ontology-based entity extraction
type OntologyConfig struct {
	Enabled                 bool
	SchemaPath              string  // Path to ontology schema files (YAML/JSON)
	DiversityThreshold      float64 // Cosine similarity threshold for diversity filtering (0.0-1.0, lower = more diverse samples)
	QueueIdleTriggerMinutes int
	MinIntervalMinutes      int
}

// Config holds worker configuration
type Config struct {
	WorkerID          string
	NumWorkers        int // Number of concurrent goroutine workers (default: 1)
	JobControlConfig  jobcontrol.Config
	ContentSources    []map[string]interface{}
	AnalyticsConfigs  []map[string]interface{}
	EmbeddingConfig   *embeddings.Config // Optional embedding configuration
	MaxDocuments      int
	DiscoveryInterval int // seconds
	Neo4jExportConfig *Neo4jExportConfig // Optional Neo4j export configuration
	SemanticConfig    *SemanticConfig    // Optional semantic relationship detection configuration
	OntologyConfig    *OntologyConfig    // Optional ontology extraction configuration
}

// NewWorker creates a new document processing worker
func NewWorker(config Config) (*Worker, error) {
	// Create job control using factory (supports SQLite and PostgreSQL)
	jc, err := jobcontrol.NewJobControl(config.JobControlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create job control: %w", err)
	}

	// Create content sources and store configs
	sources := make(map[string]contentsource.ContentSource)
	sourceConfigs := make(map[string]map[string]interface{})
	for _, sourceConfig := range config.ContentSources {
		name, ok := sourceConfig["name"].(string)
		if !ok {
			continue // Skip sources without names
		}
		// Use factory to create native Go sources where available (file, web, s3)
		// Falls back to Python shim for unsupported types
		source, err := contentsource.NewContentSource(sourceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create content source %s: %w", name, err)
		}
		sources[name] = source
		sourceConfigs[name] = sourceConfig
	}

	// Create analytics storages
	var storages []analytics.Storage
	for _, analyticsConfig := range config.AnalyticsConfigs {
		storage, err := analytics.NewStorage(analyticsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create analytics storage: %w", err)
		}
		storages = append(storages, storage)
	}

	// Create embedding generator if configured (ONNX only)
	var embeddingGenerator embeddings.EmbeddingGenerator
	var contextualBuilder *embeddings.ContextualTextBuilder
	var embeddingAggregator *EmbeddingBatchAggregator
	if config.EmbeddingConfig != nil && config.EmbeddingConfig.Enabled {
		// Create ONNX generator (only supported provider in Go)
		gen, err := embeddings.CreateEmbeddingGenerator(*config.EmbeddingConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding generator: %w", err)
		}
		embeddingGenerator = gen

		// Create cross-document batch aggregator for efficient batching
		batchSize := gen.GetBatchSize()
		if batchSize == 0 {
			batchSize = 32
		}
		embeddingAggregator = NewEmbeddingBatchAggregator(batchSize, gen)
		log.Printf("Cross-document batch aggregator enabled (batch size: %d)", batchSize)

		// Create contextual text builder if contextual mode enabled
		if config.EmbeddingConfig.Contextual {
			// Create parsers for content resolution
			htmlParser := parser.NewHTMLParser()
			xmlParser := parser.NewXMLParser()
			jsonParser := parser.NewJSONParser()

			// Create content resolver with HTML, XML, and JSON parsers
			parserResolvers := map[string]resolver.ParserResolver{
				"html": htmlParser,
				"xml":  xmlParser,
				"json": jsonParser,
			}
			contentResolver := resolver.NewContentResolver(parserResolvers)

			contextualBuilder = embeddings.NewContextualTextBuilder(*config.EmbeddingConfig, contentResolver)
			log.Printf("Contextual embeddings enabled (predecessors: %d, successors: %d)",
				config.EmbeddingConfig.PredecessorCount, config.EmbeddingConfig.SuccessorCount)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults
	numWorkers := config.NumWorkers
	if numWorkers == 0 {
		numWorkers = 1
	}

	// Create semantic analyzer if configured
	var semanticAnalyzer *SemanticAnalyzer
	if config.SemanticConfig != nil && config.SemanticConfig.Enabled {
		semanticAnalyzer = NewSemanticAnalyzer(config.SemanticConfig, storages)
		log.Printf("Semantic analysis enabled (threshold: %.2f, idle trigger: %d min)",
			config.SemanticConfig.SimilarityThreshold, config.SemanticConfig.QueueIdleTriggerMinutes)
	}

	// Create ontology extractor if configured
	var ontologyExtractor *OntologyExtractor
	if config.OntologyConfig != nil && config.OntologyConfig.Enabled {
		extractor, err := NewOntologyExtractor(config.OntologyConfig, storages)
		if err != nil {
			log.Printf("WARNING: Failed to create ontology extractor: %v", err)
			log.Printf("Ontology extraction will be disabled")
		} else {
			ontologyExtractor = extractor
			log.Printf("Ontology extraction enabled (schema path: %s, idle trigger: %d min)",
				config.OntologyConfig.SchemaPath, config.OntologyConfig.QueueIdleTriggerMinutes)
		}
	}

	worker := &Worker{
		workerID:             config.WorkerID,
		numWorkers:           numWorkers,
		jobControl:           jc,
		contentSources:       sources,
		contentSourceConfigs: sourceConfigs,
		analyticsStorages:    storages,
		embeddingGenerator:   embeddingGenerator,
		contextualBuilder:    contextualBuilder,
		embeddingAggregator:  embeddingAggregator,
		maxDocuments:         config.MaxDocuments,
		ctx:                  ctx,
		cancel:               cancel,
		neo4jExportConfig:    config.Neo4jExportConfig,
		semanticConfig:       config.SemanticConfig,
		semanticAnalyzer:     semanticAnalyzer,
		ontologyConfig:       config.OntologyConfig,
		ontologyExtractor:    ontologyExtractor,
	}

	log.Printf("Initialized worker: %s", worker.workerID)
	log.Printf("Content sources: %d", len(sources))
	log.Printf("Analytics storages: %d", len(storages))
	if embeddingGenerator != nil {
		log.Printf("Embeddings enabled (ONNX): %s (dimensions: %d)", embeddingGenerator.GetModelName(), embeddingGenerator.GetDimensions())
	}
	if config.Neo4jExportConfig != nil && config.Neo4jExportConfig.Enabled {
		log.Printf("Neo4j export enabled (wait time: %d seconds)", config.Neo4jExportConfig.EmptyQueueWaitTime)
	}
	if semanticAnalyzer != nil {
		log.Printf("Semantic analysis enabled (queue idle: %d min, min interval: %d min)",
			config.SemanticConfig.QueueIdleTriggerMinutes, config.SemanticConfig.MinIntervalMinutes)
	}
	if ontologyExtractor != nil {
		log.Printf("Ontology extraction enabled (queue idle: %d min, min interval: %d min)",
			config.OntologyConfig.QueueIdleTriggerMinutes, config.OntologyConfig.MinIntervalMinutes)
	}

	return worker, nil
}

// Run starts the worker main loop
func (w *Worker) Run() error {
	log.Printf("Starting worker %s with %d goroutine workers", w.workerID, w.numWorkers)
	w.running = true

	// Setup signal handlers
	w.setupSignalHandlers()

	// Register worker
	workerInfo := map[string]interface{}{
		"hostname":   getHostname(),
		"pid":        os.Getpid(),
		"started_at": time.Now().Format(time.RFC3339),
	}
	if err := w.jobControl.RegisterWorker(w.workerID, workerInfo); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Attempt leader election (global for backward compatibility)
	success, err := w.jobControl.ElectLeader(w.workerID, workerInfo)
	if err != nil {
		log.Printf("Global leader election failed: %v", err)
	} else if success {
		w.isLeader = true
		log.Printf("Worker %s became global leader", w.workerID)
		// NOTE: Don't start global discoveryLoop here - per-source loops handle discovery
		// The global discoveryLoop is only for backward compatibility when per-source loops aren't used
	} else {
		log.Printf("Worker %s is a follower", w.workerID)
	}

	// Start per-source discovery goroutines
	// Each content source gets its own discovery loop with separate leader election
	// This is the primary discovery mechanism - it handles all sources
	for sourceName := range w.contentSources {
		w.wg.Add(1)
		go w.perSourceDiscoveryLoop(sourceName)
	}

	// Use goroutine pool if numWorkers > 1, otherwise single-threaded
	if w.numWorkers > 1 {
		return w.runWithPool()
	}

	// Single-threaded processing loop (backward compatible)
	for w.running {
		// Check document limit
		if w.maxDocuments > 0 && atomic.LoadInt32(&w.documentsProcessed) >= int32(w.maxDocuments) {
			log.Printf("Reached maximum document limit (%d)", w.maxDocuments)
			w.running = false
			w.cancel() // Cancel context to stop discovery goroutines
			break
		}

		// Update worker heartbeat
		if err := w.jobControl.UpdateWorkerHeartbeat(w.workerID); err != nil {
			log.Printf("Failed to update heartbeat: %v", err)
		}

		// Claim next document
		docInfo, err := w.jobControl.ClaimNextDocument(w.workerID)
		if err != nil {
			log.Printf("Failed to claim document: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if docInfo != nil {
			// Process the document
			success := w.processDocument(docInfo)

			if success {
				w.jobControl.CompleteDocument(docInfo.DocID, w.workerID, true, "")
				atomic.AddInt32(&w.documentsProcessed, 1)
				log.Printf("Worker %s completed document %s (%d total)", w.workerID, docInfo.DocID, atomic.LoadInt32(&w.documentsProcessed))
			} else {
				w.jobControl.CompleteDocument(docInfo.DocID, w.workerID, false, "Processing failed")
				log.Printf("Worker %s failed to process document %s", w.workerID, docInfo.DocID)
			}
		} else {
			// No documents available
			if atomic.LoadInt32(&w.documentsProcessed) == 0 && !w.isLeader {
				log.Println("No documents available to process")
			}
			time.Sleep(5 * time.Second)

			// Try to become leader if none exists
			if !w.isLeader {
				leader, err := w.jobControl.GetCurrentLeader()
				if err == nil && leader == nil {
					success, err := w.jobControl.ElectLeader(w.workerID, workerInfo)
					if err == nil && success {
						w.isLeader = true
						log.Printf("Worker %s became leader", w.workerID)
						go w.discoveryLoop()
					}
				}
			}
		}
	}

	// Cleanup
	if w.isLeader {
		log.Printf("Releasing leadership for worker %s", w.workerID)
		w.jobControl.ReleaseLeadership(w.workerID)
	}

	log.Printf("Worker %s stopping. Processed %d documents", w.workerID, atomic.LoadInt32(&w.documentsProcessed))
	return nil
}

// runWithPool runs the worker with a goroutine pool
func (w *Worker) runWithPool() error {
	log.Printf("Starting worker pool with %d goroutines", w.numWorkers)

	// Start worker goroutines - each claims and processes its own documents
	for i := 0; i < w.numWorkers; i++ {
		w.wg.Add(1)
		go w.independentWorkerGoroutine(i)
	}

	// Wait for all goroutines to complete
	w.wg.Wait()

	// Cleanup
	if w.isLeader {
		log.Printf("Releasing leadership for worker %s", w.workerID)
		w.jobControl.ReleaseLeadership(w.workerID)
	}

	log.Printf("Worker %s stopping. Processed %d documents", w.workerID, atomic.LoadInt32(&w.documentsProcessed))
	return nil
}

// independentWorkerGoroutine - each goroutine claims and processes its own documents
func (w *Worker) independentWorkerGoroutine(id int) {
	defer w.wg.Done()

	log.Printf("Worker goroutine %d started", id)

	// Track last heartbeat time
	lastHeartbeat := time.Now()
	heartbeatInterval := 30 * time.Second

	for w.running {
		// Check document limit
		if w.maxDocuments > 0 && atomic.LoadInt32(&w.documentsProcessed) >= int32(w.maxDocuments) {
			log.Printf("Worker goroutine %d: reached max documents", id)
			return
		}

		// Check context cancellation
		select {
		case <-w.ctx.Done():
			log.Printf("Worker goroutine %d cancelled", id)
			return
		default:
		}

		// Update heartbeat periodically (only every 30 seconds, not on every iteration)
		if time.Since(lastHeartbeat) > heartbeatInterval {
			if err := w.jobControl.UpdateWorkerHeartbeat(w.workerID); err != nil {
				log.Printf("Worker goroutine %d: failed to update heartbeat: %v", id, err)
			}
			lastHeartbeat = time.Now()
		}

		// Claim next document
		docInfo, err := w.jobControl.ClaimNextDocument(w.workerID)
		if err != nil {
			log.Printf("Worker goroutine %d: failed to claim document: %v", id, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if docInfo == nil {
			// No documents available - short sleep and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Log the claim
		log.Printf("Worker goroutine %d claimed document %s", id, docInfo.DocID)

		// Process document with timing and heartbeat ticker
		startTime := time.Now()

		// Start heartbeat ticker during processing to keep claim alive
		// Heartbeat every 30 seconds to prevent claim timeout (default timeout: 300s)
		heartbeatTicker := time.NewTicker(30 * time.Second)
		heartbeatDone := make(chan struct{})

		go func() {
			tickCount := 0
			for {
				select {
				case <-heartbeatTicker.C:
					tickCount++
					log.Printf("Worker goroutine %d: heartbeat tick %d for document %s", id, tickCount, docInfo.DocID)

					// Update BOTH worker heartbeat AND document claim timestamp
					if err := w.jobControl.UpdateWorkerHeartbeat(w.workerID); err != nil {
						log.Printf("Worker goroutine %d: failed to update worker heartbeat: %v", id, err)
					}

					// CRITICAL: Update the document's claimed_at timestamp to prevent timeout
					if err := w.jobControl.UpdateDocumentClaimHeartbeat(docInfo.DocID, w.workerID); err != nil {
						log.Printf("Worker goroutine %d: failed to update document claim heartbeat for %s: %v", id, docInfo.DocID, err)
					} else {
						log.Printf("Worker goroutine %d: successfully updated claim heartbeat for %s", id, docInfo.DocID)
					}
				case <-heartbeatDone:
					log.Printf("Worker goroutine %d: stopping heartbeat for document %s after %d ticks", id, docInfo.DocID, tickCount)
					return
				}
			}
		}()

		success := w.processDocument(docInfo)

		// Stop heartbeat ticker
		heartbeatTicker.Stop()
		close(heartbeatDone)

		processingTime := time.Since(startTime)

		// Complete document
		if success {
			w.jobControl.CompleteDocument(docInfo.DocID, w.workerID, true, "")
			count := atomic.AddInt32(&w.documentsProcessed, 1)
			log.Printf("Worker goroutine %d completed document %s in %.2fs (%d total)", id, docInfo.DocID, processingTime.Seconds(), count)
		} else {
			w.jobControl.CompleteDocument(docInfo.DocID, w.workerID, false, "Processing failed")
			log.Printf("Worker goroutine %d failed to process document %s after %.2fs", id, docInfo.DocID, processingTime.Seconds())
		}
	}

	log.Printf("Worker goroutine %d stopped", id)
}


// processDocument processes a single document
func (w *Worker) processDocument(docInfo *jobcontrol.DocumentInfo) bool {
	log.Printf("Processing document %s from source %s", docInfo.DocID, docInfo.Source)

	// Get content source
	source, ok := w.contentSources[docInfo.Source]
	if !ok {
		log.Printf("Unknown content source: %s", docInfo.Source)
		return false
	}

	// Fetch document content
	sourceID, ok := docInfo.Metadata["source_id"].(string)
	if !ok {
		sourceID = docInfo.DocID
	}

	docContent, err := source.FetchDocument(sourceID)
	if err != nil {
		log.Printf("Failed to fetch document %s: %v", docInfo.DocID, err)
		return false
	}

	// Determine parser based on content
	var parseResult *parser.ParseResult

	// Check for binary_path (for binary files like XLSX, PDF)
	contentToUse := docContent.Content
	if docContent.BinaryPath != "" {
		contentToUse = docContent.BinaryPath
	}

	// Try to parse based on doc_type or file extension
	docType := docContent.DocType
	if docType == "" {
		// Try to infer from metadata
		if filename, ok := docContent.Metadata["filename"].(string); ok {
			docType = inferDocType(filename)
		} else if url, ok := docContent.Metadata["url"].(string); ok {
			// Infer from URL
			docType = inferDocType(url)
		} else {
			// Try to infer from doc_id if it looks like a URL or file path
			docType = inferDocType(docInfo.DocID)
		}

		// If still empty and this is a web source, default to html
		if docType == "" || docType == "text" {
			if sourceConfig, ok := w.contentSourceConfigs[docInfo.Source]; ok {
				if sourceType, ok := sourceConfig["type"].(string); ok && sourceType == "web" {
					docType = "html"
				}
			}
		}
	}

	log.Printf("DEBUG: Determined docType=%s for document %s", docType, docInfo.DocID)

	// Parse the document using appropriate parser with unified Parser interface
	ctx := context.Background()
	switch docType {
	case "docx":
		log.Printf("DEBUG: Parsing DOCX with BinaryPath=%s", docContent.BinaryPath)
		docxParser := parser.NewDocxParser()
		parseResult, err = docxParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: docContent.BinaryPath, // DOCX parser expects file path
			Config:  parser.DefaultParserConfig(),
		})
	case "pptx", "ppt":
		log.Printf("DEBUG: Parsing PPTX with BinaryPath=%s", docContent.BinaryPath)
		pptxParser := parser.NewPptxParser()
		parseResult, err = pptxParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: docContent.BinaryPath, // PPTX parser expects file path
			Config:  parser.DefaultParserConfig(),
		})
	case "xlsx", "xls":
		xlsxParser := parser.NewXLSXParser()
		parseResult, err = xlsxParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "pdf":
		pdfParser := parser.NewPDFParser()
		parseResult, err = pdfParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "csv":
		csvParser := parser.NewCSVParser()
		parseResult, err = csvParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "json":
		jsonParser := parser.NewJSONParser()
		parseResult, err = jsonParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "xml":
		xmlParser := parser.NewXMLParser()
		parseResult, err = xmlParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "html":
		htmlParser := parser.NewHTMLParser()
		parseResult, err = htmlParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "markdown", "md":
		markdownParser := parser.NewMarkdownParser()
		parseResult, err = markdownParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	case "text", "txt":
		textParser := parser.NewTextParser()
		parseResult, err = textParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	default:
		// Default to text parser
		textParser := parser.NewTextParser()
		parseResult, err = textParser.Parse(ctx, parser.ParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
			Config:  parser.DefaultParserConfig(),
		})
	}

	if err != nil {
		log.Printf("Failed to parse document %s: %v", docInfo.DocID, err)
		return false
	}

	// Note: Relationships and Links were removed during UDML migration
	// They are now handled by UDML graph extraction instead

	// Generate embeddings if enabled
	var embeddingMap map[string][]float64
	var embeddingTextMap map[string]string // Track the text used for each embedding
	log.Printf("DEBUG: Embedding check - embeddingGenerator=%v, elements=%d", w.embeddingGenerator != nil, len(parseResult.Elements))
	if w.embeddingGenerator != nil && len(parseResult.Elements) > 0 {
		// Collect texts for batch embedding (with contextual text if enabled)
		var texts []string
		var elementIDs []string
		embeddingTextMap = make(map[string]string)

		for _, elem := range parseResult.Elements {
			// Check if element should be embedded
			if !embeddings.ShouldEmbed(elem, parseResult.Elements) {
				continue
			}

			var embeddingText string

			// Use contextual text if builder is configured
			if w.contextualBuilder != nil {
				embeddingText = w.contextualBuilder.BuildContextualText(elem, parseResult.Elements)
			} else {
				// Simple mode: use content preview
				embeddingText = elem.ContentPreview
				if elem.Content != "" {
					embeddingText = elem.Content
				}
			}

			if embeddingText != "" {
				texts = append(texts, embeddingText)
				elementIDs = append(elementIDs, elem.ElementID)
				// Store the embedding text for this element
				embeddingTextMap[elem.ElementID] = embeddingText
			}
		}

		if len(texts) > 0 {
			log.Printf("EMBEDDINGS: Adding %d elements from document %s to cross-document batch (current size: %d/%d)",
				len(texts), docInfo.DocID, w.embeddingAggregator.Size(), w.embeddingAggregator.maxBatchSize)

			// Add embedding requests to cross-document aggregator
			embeddingMap = make(map[string][]float64)
			for i, text := range texts {
				elementID := elementIDs[i]
				w.embeddingAggregator.Add(docInfo.DocID, elementID, text)
			}

			// Strategy: Always flush to get embeddings for this document immediately
			// This ensures we can store complete data, while still benefiting from
			// cross-document batching when multiple small documents accumulate
			log.Printf("EMBEDDINGS: Flushing batch with %d total items (includes current document)",
				w.embeddingAggregator.Size())
			flushedEmbeddings, err := w.embeddingAggregator.Flush()
			if err != nil {
				log.Printf("Failed to flush embeddings: %v", err)
			} else {
				// Store flushed embeddings that belong to this document
				for _, elementID := range elementIDs {
					if embedding, ok := flushedEmbeddings[elementID]; ok {
						embeddingMap[elementID] = embedding
					}
				}
				log.Printf("EMBEDDINGS: Document %s received %d/%d embeddings from flush (total flushed: %d)",
					docInfo.DocID, len(embeddingMap), len(elementIDs), len(flushedEmbeddings))
			}
		}
	}

	// Store in analytics outputs
	for _, storage := range w.analyticsStorages {
		// Store document
		docs := []analytics.Document{{
			DocID:             docInfo.DocID,
			SourceName:        docInfo.Source,
			Title:             getStringFromMap(docContent.Metadata, "title"),
			URL:               getStringFromMap(docContent.Metadata, "url"),
			ContentType:       docContent.DocType,
			ProcessedAt:       time.Now(),
			ElementCount:      len(parseResult.Elements),
			RelationshipCount: 0, // Relationships removed during UDML migration
		}}
		if err := storage.AppendDocuments(docs); err != nil {
			log.Printf("Failed to store documents: %v", err)
			return false
		}

		// Store elements with complete schema
		var elements []analytics.Element
		for docPosition, elem := range parseResult.Elements {
			// Calculate element order (position among siblings with same parent)
			elementOrder := 0.0
			for i, e := range parseResult.Elements {
				if i >= docPosition {
					break
				}
				if e.ParentID == elem.ParentID {
					elementOrder++
				}
			}

			// Get content hash from ContentLocation if available
			contentHash := ""
			if elem.ContentLocation != nil {
				if hash, ok := elem.ContentLocation["content_hash"].(string); ok {
					contentHash = hash
				}
			}

			elements = append(elements, analytics.Element{
				ElementID:        elem.ElementID,
				DocID:            docInfo.DocID,
				SourceName:       docInfo.Source,
				ElementType:      elem.ElementType,
				ElementCategory:  parser.GetElementCategory(elem.ElementType), // Enrich with category
				Content:          elem.Content,
				ContentPreview:   elem.ContentPreview,
				ContentHash:      contentHash,
				ContentLocation:  elem.ContentLocation,
				ParentID:         elem.ParentID,
				Metadata:         elem.Metadata,
				ElementOrder:     elementOrder,
				DocumentPosition: float64(docPosition),
			})
		}
		if len(elements) > 0 {
			if err := storage.AppendElements(elements); err != nil {
				log.Printf("Failed to store elements: %v", err)
				return false
			}
		}

		// Note: Relationships and Links storage removed during UDML migration
		// UDML graph extraction will handle these via separate process

		// Store embeddings
		if embeddingMap != nil && len(embeddingMap) > 0 {
			var embeddingsToStore []analytics.Embedding
			for elementID, embedding := range embeddingMap {
				// Get the embedding text that was actually used (contextual or content preview)
				embeddingText := embeddingTextMap[elementID]

				embeddingsToStore = append(embeddingsToStore, analytics.Embedding{
					ElementID:  elementID,
					DocID:      docInfo.DocID,
					SourceName: docInfo.Source,
					Embedding:  embedding,
					Text:       embeddingText, // The contextual text used for embedding
				})
			}

			if err := storage.AppendEmbeddings(embeddingsToStore); err != nil {
				log.Printf("Failed to store embeddings: %v", err)
				// Don't return false - embeddings are optional
			}
		}
	}

	// Queue links discovered during parsing (works for all document types)
	// Parsers create link elements with element_type="link" and metadata["link_target"]
	sourceConfig, hasConfig := w.contentSourceConfigs[docInfo.Source]
	if hasConfig {
		w.queueDiscoveredLinks(docInfo, parseResult, sourceConfig)
	}

	log.Printf("Successfully processed document %s", docInfo.DocID)
	return true
}

// discoveryLoop runs document discovery for the leader
// perSourceDiscoveryLoop runs discovery for a single content source with per-source leader election
func (w *Worker) perSourceDiscoveryLoop(sourceName string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in perSourceDiscoveryLoop for %s: %v", sourceName, r)
		}
		w.wg.Done()
	}()

	log.Printf("Worker %s starting per-source discovery loop for %s", w.workerID, sourceName)

	source := w.contentSources[sourceName]
	sourceConfig := w.contentSourceConfigs[sourceName]
	isSourceLeader := false

	// Get discovery interval from source config (default: 3600 seconds = 1 hour)
	discoveryInterval := 3600 // Default: 1 hour
	if refreshInterval, ok := sourceConfig["refresh_interval"].(int); ok {
		discoveryInterval = refreshInterval
	} else if refreshInterval, ok := sourceConfig["refresh_interval"].(int64); ok {
		discoveryInterval = int(refreshInterval)
	} else if refreshInterval, ok := sourceConfig["refresh_interval"].(float64); ok {
		discoveryInterval = int(refreshInterval)
	}
	log.Printf("Worker %s using discovery interval of %d seconds for source %s", w.workerID, discoveryInterval, sourceName)

	// Try to become leader for this source
	workerInfo := map[string]interface{}{
		"hostname":   getHostname(),
		"pid":        os.Getpid(),
		"started_at": time.Now().Format(time.RFC3339),
	}

	log.Printf("DEBUG: Worker %s entering discovery loop for %s", w.workerID, sourceName)

	for {
		log.Printf("DEBUG: Worker %s discovery loop iteration for %s (running=%v, isSourceLeader=%v)", w.workerID, sourceName, w.running, isSourceLeader)

		select {
		case <-w.ctx.Done():
			// Context cancelled, exit immediately
			log.Printf("Worker %s discovery loop for %s cancelled", w.workerID, sourceName)
			goto cleanup
		default:
			// Continue with discovery
		}

		if !w.running {
			log.Printf("Worker %s discovery loop for %s stopping (running=false)", w.workerID, sourceName)
			break
		}

		// Attempt to become leader for this source if not already
		if !isSourceLeader {
			log.Printf("DEBUG: Worker %s attempting to elect source leader for %s", w.workerID, sourceName)
			success, err := w.jobControl.ElectSourceLeader(sourceName, w.workerID, fmt.Sprintf("%v", workerInfo))
			if err != nil {
				log.Printf("Source leader election failed for %s: %v", sourceName, err)
			} else if success {
				isSourceLeader = true
				log.Printf("Worker %s became leader for source %s", w.workerID, sourceName)
			} else {
				// Not leader, check again later (with context-aware sleep)
				select {
				case <-time.After(30 * time.Second):
				case <-w.ctx.Done():
					goto cleanup
				}
				continue
			}
		}

		// Update source leader heartbeat
		if isSourceLeader {
			log.Printf("DEBUG: Updating heartbeat for %s", sourceName)
			if err := w.jobControl.UpdateSourceLeaderHeartbeat(sourceName, w.workerID); err != nil {
				log.Printf("Failed to update source leader heartbeat for %s: %v", sourceName, err)
				// Lost leadership, try to re-elect
				isSourceLeader = false
				continue
			}
			log.Printf("DEBUG: Heartbeat updated successfully for %s", sourceName)
		}

		// Discover and queue documents from this source
		log.Printf("DEBUG: Calling ListDocuments for %s", sourceName)
		documents, err := source.ListDocuments()
		log.Printf("DEBUG: ListDocuments returned %d documents, err=%v", len(documents), err)
		if err != nil {
			log.Printf("Failed to list documents from %s: %v", sourceName, err)
			select {
			case <-time.After(60 * time.Second):
			case <-w.ctx.Done():
				goto cleanup
			}
			continue
		}

		queued := 0
		for _, docInfo := range documents {
			// Check if already queued
			isQueued, err := w.jobControl.IsDocumentQueued(docInfo.ID)
			if err != nil {
				log.Printf("Error checking if document queued: %v", err)
				continue
			}
			if isQueued {
				continue
			}

			// Check if changed (using metadata from job control)
			metadata, err := w.jobControl.GetDocumentMetadata(docInfo.ID)
			if err != nil {
				log.Printf("Error getting document metadata: %v", err)
			}

			var lastModified interface{}
			if metadata != nil && metadata.LastModified != nil {
				lastModified = metadata.LastModified
			}

			changed, err := source.HasChanged(docInfo.ID, lastModified)
			if err != nil {
				log.Printf("Error checking if document changed: %v", err)
				changed = true // Assume changed if we can't check
			}

			if changed {
				if err := w.jobControl.EnqueueDocument(docInfo.ID, sourceName, docInfo.Metadata); err != nil {
					log.Printf("Failed to enqueue document %s: %v", docInfo.ID, err)
					continue
				}
				queued++
			}
		}

		if queued > 0 {
			log.Printf("Source leader for %s queued %d new documents", sourceName, queued)
		}

		// Check queue status for Neo4j export and finalization tasks (only if leader)
		if isSourceLeader {
			w.checkQueueForNeo4jExport()

			// Also check finalization independently (for when Neo4j export is disabled)
			status, err := w.jobControl.GetProcessingStatus()
			if err == nil {
				pendingDocs := status.Documents["pending"]
				processingDocs := status.Documents["processing"]
				isQueueEmpty := (pendingDocs == 0 && processingDocs == 0)
				currentTime := time.Now()

				// Track queue empty time
				if isQueueEmpty {
					if w.lastQueueEmptyTime == nil {
						w.lastQueueEmptyTime = &currentTime
						log.Println("Queue became empty, starting idle timer for finalization tasks")
					}
				} else {
					// Reset timer if queue is not empty
					if w.lastQueueEmptyTime != nil {
						w.lastQueueEmptyTime = nil
					}
				}

				w.checkQueueForFinalization(isQueueEmpty, currentTime)
			}
		}

		// Sleep for configured interval (with context-aware sleep)
		select {
		case <-time.After(time.Duration(discoveryInterval) * time.Second):
		case <-w.ctx.Done():
			goto cleanup
		}
	}

cleanup:

	// Cleanup: release source leadership
	if isSourceLeader {
		log.Printf("Worker %s releasing source leadership for %s", w.workerID, sourceName)
		if err := w.jobControl.ReleaseSourceLeadership(sourceName, w.workerID); err != nil {
			log.Printf("Failed to release source leadership for %s: %v", sourceName, err)
		}
	}

	log.Printf("Worker %s stopping per-source discovery loop for %s", w.workerID, sourceName)
}

// discoveryLoop is kept for backward compatibility but is now deprecated in favor of perSourceDiscoveryLoop
func (w *Worker) discoveryLoop() {
	log.Printf("Leader %s starting discovery loop", w.workerID)

	for {
		select {
		case <-w.ctx.Done():
			log.Printf("Leader %s discovery loop cancelled", w.workerID)
			goto cleanup
		default:
			// Continue
		}

		if !w.running || !w.isLeader {
			log.Printf("Leader %s stopping discovery loop (running=%v, isLeader=%v)", w.workerID, w.running, w.isLeader)
			break
		}

		// Update leader heartbeat
		if err := w.jobControl.UpdateLeaderHeartbeat(w.workerID); err != nil {
			log.Printf("Failed to update leader heartbeat: %v", err)
		}

		// Discover and queue documents from all sources
		totalQueued := 0
		for sourceName, source := range w.contentSources {
			documents, err := source.ListDocuments()
			if err != nil {
				log.Printf("Failed to list documents from %s: %v", sourceName, err)
				continue
			}

			queued := 0
			for _, docInfo := range documents {
				// Check if already queued
				isQueued, err := w.jobControl.IsDocumentQueued(docInfo.ID)
				if err != nil {
					log.Printf("Error checking if document queued: %v", err)
					continue
				}
				if isQueued {
					continue
				}

				// Check if changed (using metadata from job control)
				metadata, err := w.jobControl.GetDocumentMetadata(docInfo.ID)
				if err != nil {
					log.Printf("Error getting document metadata: %v", err)
				}

				var lastModified interface{}
				if metadata != nil && metadata.LastModified != nil {
					lastModified = metadata.LastModified
				}

				changed, err := source.HasChanged(docInfo.ID, lastModified)
				if err != nil {
					log.Printf("Error checking if document changed: %v", err)
					changed = true // Assume changed if we can't check
				}

				if changed {
					if err := w.jobControl.EnqueueDocument(docInfo.ID, sourceName, docInfo.Metadata); err != nil {
						log.Printf("Failed to enqueue document %s: %v", docInfo.ID, err)
						continue
					}
					queued++
				}
			}

			if queued > 0 {
				log.Printf("Leader queued %d new documents from %s", queued, sourceName)
			}
			totalQueued += queued
		}

		if totalQueued > 0 {
			log.Printf("Leader queued %d total new documents", totalQueued)
		}

		// Check queue status for Neo4j export
		w.checkQueueForNeo4jExport()

		// Context-aware sleep
		select {
		case <-time.After(60 * time.Second):
		case <-w.ctx.Done():
			goto cleanup
		}
	}

cleanup:
	log.Printf("Leader %s stopping discovery loop", w.workerID)
}

// setupSignalHandlers sets up graceful shutdown handlers
func (w *Worker) setupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		log.Printf("Worker %s received signal %v, requesting shutdown", w.workerID, sig)
		w.Stop()
	}()
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.running = false
	w.cancel()
}

// Close closes the worker and releases resources
func (w *Worker) Close() error {
	for _, storage := range w.analyticsStorages {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing analytics storage: %v", err)
		}
	}
	return w.jobControl.Close()
}

// queueDiscoveredLinks queues links discovered during parsing
// All parsers create link elements with element_type="link" and metadata["link_target"]
func (w *Worker) queueDiscoveredLinks(docInfo *jobcontrol.DocumentInfo, parseResult *parser.ParseResult, sourceConfig map[string]interface{}) {
	if sourceConfig == nil {
		return
	}

	// Get current depth from metadata
	currentDepth := 0
	if depth, ok := docInfo.Metadata["discovery_depth"].(float64); ok {
		currentDepth = int(depth)
	}

	// Get max depth from source config
	maxDepth := 1 // default
	if configDepth, ok := sourceConfig["max_link_depth"].(int); ok {
		maxDepth = configDepth
	} else if configDepth, ok := sourceConfig["max_link_depth"].(float64); ok {
		maxDepth = int(configDepth)
	}

	// Check if we've reached max depth
	if currentDepth >= maxDepth {
		log.Printf("Not extracting links from %s - at max depth %d", docInfo.DocID, maxDepth)
		return
	}

	// Find all link elements and queue their targets
	queued := 0
	for _, elem := range parseResult.Elements {
		// Check if this is a link element
		if elem.ElementType != "link" {
			continue
		}

		// Extract link target from metadata
		linkTarget, ok := elem.Metadata["link_target"].(string)
		if !ok || linkTarget == "" {
			continue
		}

		// Skip mailto, javascript, and anchor-only links
		if strings.HasPrefix(linkTarget, "mailto:") ||
		   strings.HasPrefix(linkTarget, "javascript:") ||
		   strings.HasPrefix(linkTarget, "#") {
			continue
		}

		// Skip links to common image file extensions
		lowerTarget := strings.ToLower(linkTarget)
		if strings.HasSuffix(lowerTarget, ".png") || strings.HasSuffix(lowerTarget, ".jpg") ||
		   strings.HasSuffix(lowerTarget, ".jpeg") || strings.HasSuffix(lowerTarget, ".gif") ||
		   strings.HasSuffix(lowerTarget, ".bmp") || strings.HasSuffix(lowerTarget, ".svg") ||
		   strings.HasSuffix(lowerTarget, ".webp") || strings.HasSuffix(lowerTarget, ".ico") {
			continue
		}

		// Resolve relative links to absolute URLs
		targetURL := linkTarget
		hasProtocol := strings.Contains(targetURL, "://")

		if !hasProtocol {
			// This is a relative link - resolve it against the parent document URL
			resolvedURL := w.resolveRelativeLink(docInfo.DocID, targetURL)
			if resolvedURL != "" {
				targetURL = resolvedURL
			}
		}

		// Apply filters if this is a URL with a protocol
		if strings.Contains(targetURL, "://") {
			if !w.shouldIncludeLink(targetURL, sourceConfig) {
				continue
			}
		}

		// Get link type from metadata (if available)
		linkType, _ := elem.Metadata["link_type"].(string)
		if linkType == "" {
			linkType = "discovered"
		}

		// Queue the link for processing
		metadata := map[string]interface{}{
			"url":             targetURL,
			"parent_url":      docInfo.DocID,
			"discovery_depth": currentDepth + 1,
			"source_name":     docInfo.Source,
			"link_type":       linkType,
		}

		if err := w.jobControl.EnqueueDocument(targetURL, docInfo.Source, metadata); err != nil {
			log.Printf("Failed to enqueue link %s: %v", targetURL, err)
		} else {
			queued++
		}
	}

	if queued > 0 {
		log.Printf("Queued %d discovered links from %s for processing", queued, docInfo.DocID)
	}
}

// resolveRelativeLink converts a relative link to an absolute URL based on the parent document
func (w *Worker) resolveRelativeLink(parentDocID, relativeLink string) string {
	// If parent doesn't have a protocol, it's a file path - resolve relative to its directory
	if !strings.Contains(parentDocID, "://") {
		// File path - resolve relative to parent directory
		parentDir := filepath.Dir(parentDocID)
		resolvedPath := filepath.Join(parentDir, relativeLink)
		return resolvedPath
	}

	// Parse parent URL
	baseURL, err := url.Parse(parentDocID)
	if err != nil {
		log.Printf("Failed to parse parent URL %s: %v", parentDocID, err)
		return relativeLink
	}

	// Parse relative URL
	relURL, err := url.Parse(relativeLink)
	if err != nil {
		log.Printf("Failed to parse relative URL %s: %v", relativeLink, err)
		return relativeLink
	}

	// Resolve relative URL against base URL
	resolvedURL := baseURL.ResolveReference(relURL)
	return resolvedURL.String()
}

// shouldIncludeLink checks if a link should be included based on config patterns
func (w *Worker) shouldIncludeLink(link string, sourceConfig map[string]interface{}) bool {
	// Check exclude patterns first
	if excludePatternsRaw, ok := sourceConfig["exclude_patterns"]; ok {
		if excludePatterns, ok := excludePatternsRaw.([]interface{}); ok {
			for _, patternRaw := range excludePatterns {
				if pattern, ok := patternRaw.(string); ok {
					if strings.Contains(link, pattern) {
						return false
					}
				}
			}
		}
	}

	// Check include patterns
	if includePatternsRaw, ok := sourceConfig["include_patterns"]; ok {
		if includePatterns, ok := includePatternsRaw.([]interface{}); ok {
			for _, patternRaw := range includePatterns {
				if pattern, ok := patternRaw.(string); ok {
					// Check if pattern is a regex (starts with ^)
					if strings.HasPrefix(pattern, "^") {
						// Use simple prefix matching for now
						// Full regex would require regexp package
						cleanPattern := strings.TrimPrefix(pattern, "^")
						if strings.HasPrefix(link, cleanPattern) {
							return true
						}
					} else if strings.Contains(link, pattern) {
						return true
					}
				}
			}
			// If include patterns exist but none matched, exclude
			return false
		}
	}

	// If no include patterns specified, allow by default
	return true
}

// Helper functions

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func inferDocType(filename string) string {
	// Simple inference based on extension
	// Find the last dot to get the extension
	lastDot := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 || lastDot == len(filename)-1 {
		return "text"
	}

	ext := filename[lastDot:]
	switch ext {
	case ".xlsx", ".xls":
		return "xlsx"
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".md", ".markdown":
		return "markdown"
	case ".txt":
		return "text"
	default:
		return "text"
	}
}

// checkQueueForNeo4jExport checks if queue is empty and triggers Neo4j export if conditions are met
func (w *Worker) checkQueueForNeo4jExport() {
	// Only perform checks if Neo4j export is enabled
	if w.neo4jExportConfig == nil || !w.neo4jExportConfig.Enabled {
		return
	}

	// Get processing status
	status, err := w.jobControl.GetProcessingStatus()
	if err != nil {
		log.Printf("Warning: failed to get processing status for Neo4j export check: %v", err)
		return
	}

	// Check if queue is empty
	pendingDocs := status.Documents["pending"]
	processingDocs := status.Documents["processing"]

	isQueueEmpty := (pendingDocs == 0 && processingDocs == 0)
	currentTime := time.Now()

	if isQueueEmpty {
		if w.lastQueueEmptyTime == nil {
			// Queue just became empty
			w.lastQueueEmptyTime = &currentTime
			log.Println("Queue is now empty, starting timer for Neo4j export")
		} else {
			// Check if queue has been empty long enough
			timeSinceEmpty := currentTime.Sub(*w.lastQueueEmptyTime).Seconds()
			if timeSinceEmpty >= float64(w.neo4jExportConfig.EmptyQueueWaitTime) {
				// Check if enough time has passed since last export
				timeSinceLastExport := float64(999999)
				if w.lastNeo4jExportTime != nil {
					timeSinceLastExport = currentTime.Sub(*w.lastNeo4jExportTime).Seconds()
				}

				if timeSinceLastExport >= float64(w.neo4jExportConfig.EmptyQueueWaitTime) {
					log.Println("Queue has been empty long enough, triggering Neo4j export")
					w.triggerNeo4jExport()
					w.lastNeo4jExportTime = &currentTime
					w.lastQueueEmptyTime = nil // Reset timer
				} else {
					log.Printf("Queue empty but exported recently, waiting %.1f seconds", float64(w.neo4jExportConfig.EmptyQueueWaitTime)-timeSinceLastExport)
				}
			}
		}

		// Also check for finalization triggers (semantic analysis and ontology extraction)
		w.checkQueueForFinalization(isQueueEmpty, currentTime)
	} else {
		// Queue is not empty, reset timer
		if w.lastQueueEmptyTime != nil {
			log.Println("Queue is no longer empty, resetting export timer")
			w.lastQueueEmptyTime = nil
		}
	}
}

// checkQueueForFinalization checks if queue is idle and triggers finalization tasks (semantic + ontology)
func (w *Worker) checkQueueForFinalization(isQueueEmpty bool, currentTime time.Time) {
	// Trigger semantic analysis first (if enabled)
	w.checkQueueForSemanticAnalysis(isQueueEmpty, currentTime)

	// Then trigger ontology extraction (if enabled)
	w.checkQueueForOntologyExtraction(isQueueEmpty, currentTime)
}

// checkQueueForSemanticAnalysis checks if queue is idle and triggers semantic analysis
func (w *Worker) checkQueueForSemanticAnalysis(isQueueEmpty bool, currentTime time.Time) {
	// Only perform checks if semantic analysis is enabled
	if w.semanticConfig == nil || !w.semanticConfig.Enabled || w.semanticAnalyzer == nil {
		return
	}

	// Check if already running
	if w.semanticAnalysisRunning.Load() {
		return
	}

	// Check if queue is empty
	if !isQueueEmpty {
		return
	}

	// Calculate time since queue became empty
	// Use 30 seconds for testing when QueueIdleTriggerMinutes is 0
	idleTriggerSeconds := float64(w.semanticConfig.QueueIdleTriggerMinutes * 60)
	if idleTriggerSeconds == 0 {
		idleTriggerSeconds = 30 // 30 seconds for immediate testing
	}
	if w.lastQueueEmptyTime != nil {
		timeSinceEmpty := currentTime.Sub(*w.lastQueueEmptyTime).Seconds()
		if timeSinceEmpty < idleTriggerSeconds {
			return // Not idle long enough yet
		}
	}

	// Check minimum interval since last analysis
	minIntervalSeconds := float64(w.semanticConfig.MinIntervalMinutes * 60)
	if w.lastSemanticAnalysisTime != nil {
		timeSinceLastAnalysis := currentTime.Sub(*w.lastSemanticAnalysisTime).Seconds()
		if timeSinceLastAnalysis < minIntervalSeconds {
			log.Printf("SEMANTIC ANALYSIS: Skipping (ran %.1f min ago, min interval: %d min)",
				timeSinceLastAnalysis/60, w.semanticConfig.MinIntervalMinutes)
			return
		}
	}

	// Trigger semantic analysis in background
	log.Println("SEMANTIC ANALYSIS: Queue idle long enough, triggering analysis")
	w.wg.Add(1)
	go w.runSemanticAnalysis()
}

// runSemanticAnalysis runs semantic relationship detection in background
func (w *Worker) runSemanticAnalysis() {
	defer w.wg.Done()

	// Set running flag
	if !w.semanticAnalysisRunning.CompareAndSwap(false, true) {
		log.Println("SEMANTIC ANALYSIS: Already running, skipping")
		return
	}
	defer w.semanticAnalysisRunning.Store(false)

	// Update last analysis time
	now := time.Now()
	w.lastSemanticAnalysisTime = &now

	// Run analysis
	if err := w.semanticAnalyzer.AnalyzeAndStore(); err != nil {
		log.Printf("SEMANTIC ANALYSIS: Failed: %v", err)
	}
}

// checkQueueForOntologyExtraction checks if queue is idle and triggers ontology extraction
func (w *Worker) checkQueueForOntologyExtraction(isQueueEmpty bool, currentTime time.Time) {
	// Only perform checks if ontology extraction is enabled
	if w.ontologyConfig == nil || !w.ontologyConfig.Enabled || w.ontologyExtractor == nil {
		return
	}

	// Check if already running
	if w.ontologyExtractionRunning.Load() {
		return
	}

	// Check if queue is empty
	if !isQueueEmpty {
		return
	}

	// Calculate time since queue became empty
	// Use 30 seconds for testing when QueueIdleTriggerMinutes is 0
	idleTriggerSeconds := float64(w.ontologyConfig.QueueIdleTriggerMinutes * 60)
	if idleTriggerSeconds == 0 {
		idleTriggerSeconds = 30 // 30 seconds for immediate testing
	}
	if w.lastQueueEmptyTime != nil {
		timeSinceEmpty := currentTime.Sub(*w.lastQueueEmptyTime).Seconds()
		if timeSinceEmpty < idleTriggerSeconds {
			return // Not idle long enough yet
		}
	}

	// Check minimum interval since last extraction
	minIntervalSeconds := float64(w.ontologyConfig.MinIntervalMinutes * 60)
	if w.lastOntologyExtractionTime != nil {
		timeSinceLastExtraction := currentTime.Sub(*w.lastOntologyExtractionTime).Seconds()
		if timeSinceLastExtraction < minIntervalSeconds {
			log.Printf("ONTOLOGY EXTRACTION: Skipping (ran %.1f min ago, min interval: %d min)",
				timeSinceLastExtraction/60, w.ontologyConfig.MinIntervalMinutes)
			return
		}
	}

	// Trigger ontology extraction in background
	log.Println("ONTOLOGY EXTRACTION: Queue idle long enough, triggering extraction")
	w.wg.Add(1)
	go w.runOntologyExtraction()
}

// runOntologyExtraction runs ontology entity extraction in background
func (w *Worker) runOntologyExtraction() {
	defer w.wg.Done()

	// Set running flag
	if !w.ontologyExtractionRunning.CompareAndSwap(false, true) {
		log.Println("ONTOLOGY EXTRACTION: Already running, skipping")
		return
	}
	defer w.ontologyExtractionRunning.Store(false)

	// Update last extraction time
	now := time.Now()
	w.lastOntologyExtractionTime = &now

	// Run extraction
	if err := w.ontologyExtractor.ExtractAndStore(); err != nil {
		log.Printf("ONTOLOGY EXTRACTION: Failed: %v", err)
	}
}

// triggerNeo4jExport triggers Neo4j graph export
func (w *Worker) triggerNeo4jExport() {
	log.Println("========================================")
	log.Println("TRIGGERING NEO4J GRAPH EXPORT")
	log.Println("========================================")

	// Import export package
	// Dynamically build configuration from worker settings
	sourceAnalytics := w.neo4jExportConfig.SourceAnalytics
	if sourceAnalytics == "" {
		sourceAnalytics = "parquet"
	}

	// Find source path from analytics configs
	sourcePath := w.neo4jExportConfig.SourcePath
	if sourcePath == "" {
		// Use configured source path from Neo4j export config
		// Fallback to /tmp if not available
		sourcePath = "/tmp"
		log.Printf("Warning: No source path configured for Neo4j export, using fallback: %s", sourcePath)
	}

	// This will be implemented in the next step - for now just log
	log.Printf("Would export from %s at %s to Neo4j", sourceAnalytics, sourcePath)
	log.Printf("Neo4j connection: %v", w.neo4jExportConfig.Connection)

	// TODO: Implement actual export using graph exporter
	// exporter, err := export.NewGraphExporter(export.Config{
	// 	SourceType:       sourceAnalytics,
	// 	SourcePath:       sourcePath,
	// 	TargetType:       "neo4j",
	// 	TargetConnection: w.neo4jExportConfig.Connection,
	// 	BatchSize:        w.neo4jExportConfig.BatchSize,
	// })
	// if err != nil {
	// 	log.Printf("ERROR: Failed to create graph exporter: %v", err)
	// 	return
	// }
	// defer exporter.Close()
	//
	// result := exporter.ExportToNeo4j()
	// if result.Success {
	// 	log.Printf("✓ Neo4j export completed successfully")
	// 	log.Printf("  Documents: %d", result.DocumentCount)
	// 	log.Printf("  Elements: %d", result.ElementCount)
	// 	log.Printf("  Relationships: %d", result.RelationshipCount)
	// } else {
	// 	log.Printf("ERROR: Neo4j export failed: %s", result.Error)
	// }

	log.Println("========================================")
}
