package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kennethstott/go-doc-go/internal/analytics"
	"github.com/kennethstott/go-doc-go/internal/contentsource"
	"github.com/kennethstott/go-doc-go/internal/jobcontrol"
	"github.com/kennethstott/go-doc-go/internal/parser"
)

// Worker represents a document processing worker
type Worker struct {
	workerID           string
	numWorkers         int
	batchClaimSize     int
	jobControl         jobcontrol.JobControl
	contentSources     map[string]contentsource.ContentSource
	analyticsStorages  []analytics.Storage
	maxDocuments       int
	documentsProcessed int32 // atomic counter for goroutine safety
	isLeader           bool
	running            bool
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
}

// Config holds worker configuration
type Config struct {
	WorkerID          string
	NumWorkers        int // Number of concurrent goroutine workers (default: 1)
	BatchClaimSize    int // Number of documents to claim at once (default: 5)
	JobControlConfig  jobcontrol.Config
	ContentSources    []map[string]interface{}
	AnalyticsConfigs  []map[string]interface{}
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

	// Create content sources
	sources := make(map[string]contentsource.ContentSource)
	for _, sourceConfig := range config.ContentSources {
		name, ok := sourceConfig["name"].(string)
		if !ok {
			continue // Skip sources without names
		}
		source, err := contentsource.NewPythonShimSource(name, sourceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create content source %s: %w", name, err)
		}
		sources[name] = source
	}

	// Create analytics storages
	var storages []analytics.Storage
	for _, analyticsConfig := range config.AnalyticsConfigs {
		storage, err := analytics.NewPythonShimStorage(analyticsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create analytics storage: %w", err)
		}
		storages = append(storages, storage)
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
		workerID:          config.WorkerID,
		numWorkers:        numWorkers,
		batchClaimSize:    batchClaimSize,
		jobControl:        jc,
		contentSources:    sources,
		analyticsStorages: storages,
		maxDocuments:      config.MaxDocuments,
		ctx:               ctx,
		cancel:            cancel,
	}

	log.Printf("Initialized worker: %s", worker.workerID)
	log.Printf("Content sources: %d", len(sources))
	log.Printf("Analytics storages: %d", len(storages))

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

	// Attempt leader election
	success, err := w.jobControl.ElectLeader(w.workerID, workerInfo)
	if err != nil {
		log.Printf("Leader election failed: %v", err)
	} else if success {
		w.isLeader = true
		log.Printf("Worker %s became leader", w.workerID)
		// Start discovery in background
		go w.discoveryLoop()
	} else {
		log.Printf("Worker %s is a follower", w.workerID)
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
		}
	}

	// Parse the document using appropriate parser
	// Note: DOCX and Markdown have different interfaces, to be added later
	switch docType {
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

		// Store elements
		var elements []analytics.Element
		for _, elem := range parseResult.Elements {
			elements = append(elements, analytics.Element{
				ElementID:      elem.ElementID,
				DocID:          docInfo.DocID,
				SourceName:     docInfo.Source,
				ElementType:    elem.ElementType,
				ContentPreview: elem.ContentPreview,
				ParentID:       elem.ParentID,
				Metadata:       elem.Metadata,
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
	}

	log.Printf("Successfully processed document %s", docInfo.DocID)
	return true
}

// discoveryLoop runs document discovery for the leader
func (w *Worker) discoveryLoop() {
	log.Printf("Leader %s starting discovery loop", w.workerID)

	for w.running && w.isLeader {
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

		// Sleep for a while (check every minute)
		time.Sleep(60 * time.Second)
	}

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
	if len(filename) < 4 {
		return "text"
	}

	ext := filename[len(filename)-4:]
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
	case ".md":
		return "markdown"
	default:
		return "text"
	}
}
