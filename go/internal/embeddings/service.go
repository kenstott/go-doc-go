package embeddings

import (
	"fmt"
	"log"
	"sync"
)

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Texts        []string
	ResponseChan chan EmbeddingResponse
}

// EmbeddingResponse represents the response from embedding generation
type EmbeddingResponse struct {
	Embeddings [][]float64
	Error      error
}

// Service provides serialized access to embedding generation
// This ensures only one embedding operation happens at a time,
// preventing multiple Python processes from spawning and exhausting memory
type Service struct {
	generator   EmbeddingGenerator
	requestChan chan EmbeddingRequest
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewService creates a new embedding service
func NewService(generator EmbeddingGenerator) *Service {
	return &Service{
		generator:   generator,
		requestChan: make(chan EmbeddingRequest, 10), // Buffer 10 requests
		stopChan:    make(chan struct{}),
		running:     false,
	}
}

// Start starts the embedding service
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("service already running")
	}

	s.running = true
	s.wg.Add(1)
	go s.processLoop()

	log.Println("Embedding service started")
	return nil
}

// Stop stops the embedding service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("service not running")
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()

	// Close the generator
	if s.generator != nil {
		if err := s.generator.Close(); err != nil {
			return fmt.Errorf("failed to close generator: %w", err)
		}
	}

	log.Println("Embedding service stopped")
	return nil
}

// GenerateBatch generates embeddings for multiple texts (thread-safe)
func (s *Service) GenerateBatch(texts []string) ([][]float64, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("service not running")
	}
	s.mu.Unlock()

	// Create response channel
	responseChan := make(chan EmbeddingResponse, 1)

	// Send request
	request := EmbeddingRequest{
		Texts:        texts,
		ResponseChan: responseChan,
	}

	select {
	case s.requestChan <- request:
		// Request sent
	case <-s.stopChan:
		return nil, fmt.Errorf("service stopped")
	}

	// Wait for response
	select {
	case response := <-responseChan:
		return response.Embeddings, response.Error
	case <-s.stopChan:
		return nil, fmt.Errorf("service stopped")
	}
}

// processLoop is the main processing loop (runs in a single goroutine)
func (s *Service) processLoop() {
	defer s.wg.Done()

	log.Println("Embedding service process loop started")

	for {
		select {
		case request := <-s.requestChan:
			// Process request serially
			embeddings, err := s.generator.GenerateBatch(request.Texts)

			// Send response
			response := EmbeddingResponse{
				Embeddings: embeddings,
				Error:      err,
			}

			select {
			case request.ResponseChan <- response:
				// Response sent
			case <-s.stopChan:
				return
			}

		case <-s.stopChan:
			log.Println("Embedding service process loop stopping")
			return
		}
	}
}

// GetDimensions returns the embedding dimensions
func (s *Service) GetDimensions() int {
	if s.generator == nil {
		return 0
	}
	return s.generator.GetDimensions()
}

// GetModelName returns the model name
func (s *Service) GetModelName() string {
	if s.generator == nil {
		return ""
	}
	return s.generator.GetModelName()
}
