package worker

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kennethstott/go-doc-go/internal/analytics"
	"github.com/kennethstott/go-doc-go/internal/contentsource"
	"github.com/kennethstott/go-doc-go/internal/embeddings"
	"github.com/kennethstott/go-doc-go/internal/jobcontrol"
	"github.com/kennethstott/go-doc-go/internal/parser"
	"golang.org/x/net/html"
)

// Worker represents a document processing worker
type Worker struct {
	workerID             string
	numWorkers           int
	batchClaimSize       int
	jobControl           jobcontrol.JobControl
	contentSources       map[string]contentsource.ContentSource
	contentSourceConfigs map[string]map[string]interface{}
	analyticsStorages    []analytics.Storage
	embeddingService     *embeddings.Service
	contextualBuilder    *embeddings.ContextualTextBuilder
	maxDocuments         int
	documentsProcessed   int32 // atomic counter for goroutine safety
	isLeader             bool
	running              bool
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
}

// Config holds worker configuration
type Config struct {
	WorkerID          string
	NumWorkers        int // Number of concurrent goroutine workers (default: 1)
	BatchClaimSize    int // Number of documents to claim at once (default: 5)
	JobControlConfig  jobcontrol.Config
	ContentSources    []map[string]interface{}
	AnalyticsConfigs  []map[string]interface{}
	EmbeddingConfig   *embeddings.Config // Optional embedding configuration
	MaxDocuments      int
	DiscoveryInterval int // seconds
}

// ProcessResult represents the result of processing a document
type ProcessResult struct {
	DocID   string
	Success bool
	Error   string
}

// NewWorker creates a new document processing worker
func NewWorker(config Config) (*Worker, error) {
	// Create job control
	jc, err := jobcontrol.NewSQLiteJobControl(config.JobControlConfig)
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

	// Create embedding service if configured
	var embeddingService *embeddings.Service
	var contextualBuilder *embeddings.ContextualTextBuilder
	if config.EmbeddingConfig != nil && config.EmbeddingConfig.Enabled {
		// Create generator using factory (supports both Python shim and ONNX)
		gen, err := embeddings.CreateEmbeddingGenerator(*config.EmbeddingConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding generator: %w", err)
		}

		// ONNX provider uses direct generator access (thread-safe, parallel inference)
		// Python shim uses Service wrapper (serialized access)
		if config.EmbeddingConfig.Provider == "onnx" {
			log.Println("Using native ONNX embedding generator (parallel inference enabled)")
			// Bypass Service wrapper - ONNX sessions are thread-safe
			embeddingService = embeddings.NewService(gen) // Still wrap for now for API compatibility
		} else {
			// Python shim needs serialized access through Service
			embeddingService = embeddings.NewService(gen)
			log.Println("Using Python shim embedding generator (serialized access)")
		}

		// Create contextual text builder if contextual mode enabled
		if config.EmbeddingConfig.Contextual {
			contextualBuilder = embeddings.NewContextualTextBuilder(*config.EmbeddingConfig)
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
	batchClaimSize := config.BatchClaimSize
	if batchClaimSize == 0 {
		batchClaimSize = 5
	}

	worker := &Worker{
		workerID:             config.WorkerID,
		numWorkers:           numWorkers,
		batchClaimSize:       batchClaimSize,
		jobControl:           jc,
		contentSources:       sources,
		contentSourceConfigs: sourceConfigs,
		analyticsStorages:    storages,
		embeddingService:     embeddingService,
		contextualBuilder:    contextualBuilder,
		maxDocuments:         config.MaxDocuments,
		ctx:                  ctx,
		cancel:               cancel,
	}

	log.Printf("Initialized worker: %s", worker.workerID)
	log.Printf("Content sources: %d", len(sources))
	log.Printf("Analytics storages: %d", len(storages))
	if embeddingService != nil {
		log.Printf("Embeddings enabled: %s (dimensions: %d)", embeddingService.GetModelName(), embeddingService.GetDimensions())
	}

	return worker, nil
}

// Run starts the worker main loop
func (w *Worker) Run() error {
	log.Printf("Starting worker %s with %d goroutine workers", w.workerID, w.numWorkers)
	w.running = true

	// Setup signal handlers
	w.setupSignalHandlers()

	// Start embedding service if configured
	if w.embeddingService != nil {
		if err := w.embeddingService.Start(); err != nil {
			return fmt.Errorf("failed to start embedding service: %w", err)
		}
		defer w.embeddingService.Stop()
	}

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
	log.Printf("Starting goroutine pool with %d workers, batch claim size: %d", w.numWorkers, w.batchClaimSize)

	// Create channels
	workChan := make(chan *jobcontrol.DocumentInfo, w.numWorkers*2)
	resultChan := make(chan ProcessResult, w.numWorkers)

	// Start worker goroutines
	for i := 0; i < w.numWorkers; i++ {
		w.wg.Add(1)
		go w.workerGoroutine(i, workChan, resultChan)
	}

	// Start document claimer goroutine
	w.wg.Add(1)
	go w.documentClaimerGoroutine(workChan)

	// Start result handler goroutine
	w.wg.Add(1)
	go w.resultHandlerGoroutine(resultChan)

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

// workerGoroutine processes documents from the work channel
func (w *Worker) workerGoroutine(id int, workChan <-chan *jobcontrol.DocumentInfo, resultChan chan<- ProcessResult) {
	defer w.wg.Done()

	log.Printf("Worker goroutine %d started", id)

	for {
		select {
		case docInfo := <-workChan:
			if docInfo == nil {
				log.Printf("Worker goroutine %d shutting down", id)
				return
			}

			// Process document
			success := w.processDocument(docInfo)

			// Send result
			resultChan <- ProcessResult{
				DocID:   docInfo.DocID,
				Success: success,
			}

		case <-w.ctx.Done():
			log.Printf("Worker goroutine %d cancelled", id)
			return
		}
	}
}

// documentClaimerGoroutine claims documents in batches and sends to work channel
func (w *Worker) documentClaimerGoroutine(workChan chan<- *jobcontrol.DocumentInfo) {
	defer w.wg.Done()
	defer close(workChan)

	log.Printf("Document claimer started (batch size: %d)", w.batchClaimSize)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check document limit
			if w.maxDocuments > 0 && atomic.LoadInt32(&w.documentsProcessed) >= int32(w.maxDocuments) {
				log.Printf("Reached maximum document limit (%d)", w.maxDocuments)
				// Trigger shutdown of all goroutines
				w.running = false
				w.cancel() // Cancel context to stop all goroutines
				return
			}

			// Update worker heartbeat
			if err := w.jobControl.UpdateWorkerHeartbeat(w.workerID); err != nil {
				log.Printf("Failed to update heartbeat: %v", err)
			}

			// Claim multiple documents at once
			claimed := 0
			for i := 0; i < w.batchClaimSize; i++ {
				docInfo, err := w.jobControl.ClaimNextDocument(w.workerID)
				if err != nil {
					log.Printf("Failed to claim document: %v", err)
					break
				}
				if docInfo == nil {
					break // No more documents available
				}

				// Send to worker pool
				workChan <- docInfo
				claimed++
			}

			if claimed == 0 {
				// No documents available, try to become leader if needed
				if !w.isLeader && atomic.LoadInt32(&w.documentsProcessed) == 0 {
					log.Println("No documents available to process")
				}

				if !w.isLeader {
					leader, err := w.jobControl.GetCurrentLeader()
					if err == nil && leader == nil {
						workerInfo := map[string]interface{}{
							"hostname":   getHostname(),
							"pid":        os.Getpid(),
							"started_at": time.Now().Format(time.RFC3339),
						}
						success, err := w.jobControl.ElectLeader(w.workerID, workerInfo)
						if err == nil && success {
							w.isLeader = true
							log.Printf("Worker %s became leader", w.workerID)
							go w.discoveryLoop()
						}
					}
				}
			}

		case <-w.ctx.Done():
			log.Println("Document claimer shutting down")
			return
		}

		if !w.running {
			log.Println("Document claimer stopping")
			return
		}
	}
}

// resultHandlerGoroutine processes results from worker goroutines
func (w *Worker) resultHandlerGoroutine(resultChan <-chan ProcessResult) {
	defer w.wg.Done()

	log.Println("Result handler started")

	for {
		select {
		case result := <-resultChan:
			if result.Success {
				w.jobControl.CompleteDocument(result.DocID, w.workerID, true, "")
				count := atomic.AddInt32(&w.documentsProcessed, 1)
				log.Printf("Worker %s completed document %s (%d total)", w.workerID, result.DocID, count)
			} else {
				w.jobControl.CompleteDocument(result.DocID, w.workerID, false, result.Error)
				log.Printf("Worker %s failed to process document %s: %s", w.workerID, result.DocID, result.Error)
			}

		case <-w.ctx.Done():
			log.Println("Result handler shutting down")
			return
		}

		if !w.running {
			log.Println("Result handler stopping")
			return
		}
	}
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

	// Parse the document using appropriate parser
	switch docType {
	case "docx":
		log.Printf("DEBUG: Parsing DOCX with BinaryPath=%s", docContent.BinaryPath)
		docxParser := parser.NewDocxParser()
		docxResult, docxErr := docxParser.Parse(parser.DocxParseRequest{
			ID:      docInfo.DocID,
			Content: docContent.BinaryPath, // DOCX parser expects file path
		})
		if docxErr != nil {
			log.Printf("ERROR: DOCX parse error: %v", docxErr)
			err = docxErr
		} else {
			log.Printf("DEBUG: DOCX parsed successfully: %d elements, %d relationships",
				len(docxResult.Elements), len(docxResult.Relationships))
			parseResult = docxResult.ToParseResult()
		}
	case "xlsx", "xls":
		xlsxParser := parser.NewXLSXParser()
		parseResult, err = xlsxParser.Parse(docInfo.DocID, contentToUse)
	case "pdf":
		pdfParser := parser.NewPDFParser()
		parseResult, err = pdfParser.Parse(docInfo.DocID, contentToUse)
	case "csv":
		csvParser := parser.NewCSVParser()
		parseResult, err = csvParser.Parse(docInfo.DocID, contentToUse)
	case "json":
		jsonParser := parser.NewJSONParser()
		parseResult, err = jsonParser.Parse(docInfo.DocID, contentToUse)
	case "xml":
		xmlParser := parser.NewXMLParser()
		parseResult, err = xmlParser.Parse(docInfo.DocID, contentToUse)
	case "html":
		htmlParser := parser.NewHTMLParser()
		parseResult, err = htmlParser.Parse(docInfo.DocID, contentToUse)
	case "markdown", "md":
		markdownParser := parser.NewMarkdownParser()
		mdResult, mdErr := markdownParser.Parse(parser.MarkdownParseRequest{
			ID:      docInfo.DocID,
			Content: contentToUse,
		})
		if mdErr != nil {
			err = mdErr
		} else {
			parseResult = mdResult.ToParseResult()
		}
	case "text", "txt":
		textParser := parser.NewTextParser()
		parseResult, err = textParser.Parse(docInfo.DocID, contentToUse)
	default:
		// Default to text parser
		textParser := parser.NewTextParser()
		parseResult, err = textParser.Parse(docInfo.DocID, contentToUse)
	}

	if err != nil {
		log.Printf("Failed to parse document %s: %v", docInfo.DocID, err)
		return false
	}

	// Generate embeddings if enabled
	var embeddingMap map[string][]float64
	var embeddingTextMap map[string]string // Track the text used for each embedding
	if w.embeddingService != nil && len(parseResult.Elements) > 0 {
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
				embeddingText = w.contextualBuilder.BuildContextualText(elem, parseResult.Elements, parseResult.Relationships)
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
			// Batch embeddings for efficient processing
			// Native ONNX can handle larger batches efficiently
			const maxBatchSize = 100  // Increased from 10 for better performance with native ONNX
			embeddingMap = make(map[string][]float64)
			log.Printf("EMBEDDINGS: Generating for %d elements in document %s (batch size: %d)", len(texts), docInfo.DocID, maxBatchSize)

			for i := 0; i < len(texts); i += maxBatchSize {
				end := i + maxBatchSize
				if end > len(texts) {
					end = len(texts)
				}

				batchTexts := texts[i:end]
				batchIDs := elementIDs[i:end]

				// Use embedding service (serialized access, thread-safe)
				embeddingVectors, err := w.embeddingService.GenerateBatch(batchTexts)
				if err != nil {
					log.Printf("Failed to generate embeddings for batch %d-%d: %v", i, end, err)
					continue
				}

				log.Printf("DEBUG: Batch %d-%d returned %d embedding vectors for %d texts", i, end, len(embeddingVectors), len(batchTexts))
				for j, elementID := range batchIDs {
					if j < len(embeddingVectors) {
						embeddingMap[elementID] = embeddingVectors[j]
						log.Printf("DEBUG: Stored embedding for %s (vector length: %d)", elementID, len(embeddingVectors[j]))
					} else {
						log.Printf("DEBUG: WARNING - No embedding vector for element %s at index %d", elementID, j)
					}
				}
			}

			totalBatches := (len(texts) + maxBatchSize - 1) / maxBatchSize
			log.Printf("EMBEDDINGS: Generated %d embeddings in %d batches for document %s", len(embeddingMap), totalBatches, docInfo.DocID)
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
			RelationshipCount: len(parseResult.Relationships),
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

		// Store relationships
		var relationships []analytics.Relationship
		for _, rel := range parseResult.Relationships {
			relationships = append(relationships, analytics.Relationship{
				SourceElementID:  rel.SourceElementID,
				TargetElementID:  rel.TargetElementID,
				RelationshipType: rel.RelationshipType,
				DocID:            docInfo.DocID,
				SourceName:       docInfo.Source,
				Metadata:         rel.Metadata,
			})
		}
		if len(relationships) > 0 {
			if err := storage.AppendRelationships(relationships); err != nil {
				log.Printf("Failed to store relationships: %v", err)
				return false
			}
		}

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

	// Extract and queue links from HTML/web documents
	sourceConfig, hasConfig := w.contentSourceConfigs[docInfo.Source]
	if hasConfig && (docType == "html" || docType == "htm") {
		// Check if this is a web source
		if sourceType, ok := sourceConfig["type"].(string); ok && sourceType == "web" {
			w.extractAndQueueLinks(docInfo, docContent.Content, sourceConfig)
		}
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
	// Stop embedding service
	if w.embeddingService != nil {
		if err := w.embeddingService.Stop(); err != nil {
			log.Printf("Error stopping embedding service: %v", err)
		}
	}

	for _, storage := range w.analyticsStorages {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing analytics storage: %v", err)
		}
	}
	return w.jobControl.Close()
}

// extractAndQueueLinks extracts links from HTML content and queues them for processing
func (w *Worker) extractAndQueueLinks(docInfo *jobcontrol.DocumentInfo, htmlContent string, sourceConfig map[string]interface{}) {
	// Get current depth from metadata
	currentDepth := 0
	if depth, ok := docInfo.Metadata["discovery_depth"].(float64); ok {
		currentDepth = int(depth)
	}

	// Get max depth from source config (not from document metadata)
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

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Failed to parse HTML for link extraction: %v", err)
		return
	}

	// Parse base URL
	baseURL, err := url.Parse(docInfo.DocID)
	if err != nil {
		log.Printf("Failed to parse base URL %s: %v", docInfo.DocID, err)
		return
	}

	// Extract links
	links := make(map[string]bool)
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					// Resolve relative URL
					linkURL, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					absoluteURL := baseURL.ResolveReference(linkURL)

					// Skip if different domain
					if absoluteURL.Host != baseURL.Host {
						continue
					}

					// Skip fragments
					absoluteURL.Fragment = ""
					linkStr := absoluteURL.String()

					// Skip if same as base URL
					if linkStr == docInfo.DocID {
						continue
					}

					// Apply filters
					if w.shouldIncludeLink(linkStr, sourceConfig) {
						links[linkStr] = true
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	// Queue discovered links
	queued := 0
	for link := range links {
		metadata := map[string]interface{}{
			"url":             link,
			"parent_url":      docInfo.DocID,
			"discovery_depth": currentDepth + 1,
			"source_name":     docInfo.Source,
		}

		if err := w.jobControl.EnqueueDocument(link, docInfo.Source, metadata); err != nil {
			log.Printf("Failed to enqueue link %s: %v", link, err)
		} else {
			queued++
		}
	}

	if queued > 0 {
		log.Printf("Queued %d links from %s for processing", queued, docInfo.DocID)
	}
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
