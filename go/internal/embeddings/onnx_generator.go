package embeddings

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	ort "github.com/yalue/onnxruntime_go"
)

// SessionWrapper wraps a reusable ONNX session with pre-allocated tensors
type SessionWrapper struct {
	session            *ort.AdvancedSession
	inputTensor        *ort.Tensor[int64]
	maskTensor         *ort.Tensor[int64]
	outputTensor       *ort.Tensor[float32]
	inputData          []int64
	maskData           []int64
	batchSize          int64
	seqLength          int64
}

// OnnxEmbeddingGenerator implements native Go embedding generation using ONNX Runtime
type OnnxEmbeddingGenerator struct {
	config          Config
	modelPath       string
	modelDir        string
	tokenizer       *SimpleTokenizer
	dimensions      int
	maxSeqLength    int
	coreMLAttempted bool // Track if we've already tried CoreML to avoid log spam
	sessionPool     chan *SessionWrapper // Pool of reusable sessions with fixed shapes
	poolSize        int
	fixedBatchSize  int64
	fixedSeqLength  int64
}

// SimpleTokenizer implements basic BERT-style tokenization
type SimpleTokenizer struct {
	vocab          map[string]int64
	idToToken      map[int64]string
	clsToken       int64
	sepToken       int64
	padToken       int64
	unkToken       int64
	maxSeqLength   int
}

// NewOnnxEmbeddingGenerator creates a new ONNX-based embedding generator
func NewOnnxEmbeddingGenerator(config Config, modelDir string) (*OnnxEmbeddingGenerator, error) {
	// Set library path based on platform
	// Try to find ONNX Runtime library in common locations
	libraryPath := os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH")
	if libraryPath == "" {
		// Try Python venv CoreML-enabled library first (for better performance on macOS)
		homeDir, _ := os.UserHomeDir()
		venvPath := filepath.Join(homeDir, "PycharmProjects", "doculyzer-go-conversion", ".venv", "lib", "python3.12", "site-packages", "onnxruntime", "capi", "libonnxruntime.1.23.0.dylib")
		if _, err := os.Stat(venvPath); err == nil {
			libraryPath = venvPath
			log.Printf("Using CoreML-enabled ONNX Runtime from Python venv")
		} else {
			// Fall back to Homebrew locations
			homebrewPath := "/opt/homebrew/lib/libonnxruntime.dylib"
			if _, err := os.Stat(homebrewPath); err == nil {
				libraryPath = homebrewPath
			} else {
				// Try user's homebrew
				userBrewPath := filepath.Join(homeDir, "homebrew", "lib", "libonnxruntime.dylib")
				if _, err := os.Stat(userBrewPath); err == nil {
					libraryPath = userBrewPath
				}
			}
		}
	}

	if libraryPath != "" {
		log.Printf("Setting ONNX Runtime library path: %s", libraryPath)
		ort.SetSharedLibraryPath(libraryPath)
	}

	// Initialize ONNX Runtime
	err := ort.InitializeEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Resolve absolute model path
	absModelDir := modelDir
	if !filepath.IsAbs(modelDir) {
		// If relative path, resolve from project root
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		// If running from go/ directory and path starts with go/, strip it
		if filepath.Base(cwd) == "go" && strings.HasPrefix(modelDir, "go/") {
			modelDir = strings.TrimPrefix(modelDir, "go/")
		}
		absModelDir = filepath.Join(cwd, modelDir)
	}

	// Load model configuration
	configPath := filepath.Join(absModelDir, "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model config: %w", err)
	}

	var modelConfig struct {
		ModelType              string `json:"model_type"`
		HiddenSize             int    `json:"hidden_size"`
		MaxPositionEmbeddings  int    `json:"max_position_embeddings"`
	}
	if err := json.Unmarshal(configData, &modelConfig); err != nil {
		return nil, fmt.Errorf("failed to parse model config: %w", err)
	}

	// Use standard transformer config fields
	embeddingDimension := modelConfig.HiddenSize
	maxSeqLength := modelConfig.MaxPositionEmbeddings

	if embeddingDimension == 0 {
		embeddingDimension = config.Dimensions // Fall back to config
	}
	if maxSeqLength == 0 {
		maxSeqLength = config.ChunkSize // Fall back to config
	}

	// Load tokenizer
	tokenizer, err := LoadSimpleTokenizer(filepath.Join(absModelDir, "vocab.txt"), maxSeqLength)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	modelPath := filepath.Join(absModelDir, "model.onnx")

	log.Printf("Loaded ONNX embedding model: %s (dimensions: %d, max_seq_length: %d)",
		modelConfig.ModelType, embeddingDimension, maxSeqLength)

	// Fixed dimensions for session reuse
	// Use batch_size=1 for optimal CoreML performance (batching at ONNX level slower than parallel single items)
	const fixedBatchSize = 1
	fixedSeqLength := int64(maxSeqLength)

	// Create session pool with pre-allocated sessions and tensors
	// This is THE KEY to performance - sessions are expensive to create with CoreML
	poolSize := config.PoolSize
	if poolSize == 0 {
		poolSize = 4 // Default to 4 if not specified
	}
	sessionPool := make(chan *SessionWrapper, poolSize)

	log.Printf("Creating session pool with %d reusable ONNX sessions (batch_size=%d, seq_length=%d)...",
		poolSize, fixedBatchSize, fixedSeqLength)

	for i := 0; i < poolSize; i++ {
		// Create session options with platform-specific GPU acceleration
		sessionOptions, err := ort.NewSessionOptions()
		if err != nil {
			return nil, fmt.Errorf("failed to create session options: %w", err)
		}

		// Enable GPU acceleration based on platform
		if i == 0 {
			// Log GPU setup only for first session to avoid spam
			if runtime.GOOS == "darwin" {
				// macOS - use CoreML for Apple Silicon GPU/Neural Engine
				coreMLOptions := map[string]string{
					"ModelFormat":                        "NeuralNetwork", // Use NeuralNetwork format for stability
					"MLComputeUnits":                     "ALL",           // Enable all compute units (CPU, GPU, Neural Engine)
					"RequireStaticInputShapes":           "0",             // Allow dynamic shapes (important for BERT)
					"EnableOnSubgraphs":                  "0",             // Don't enable on subgraphs
					"AllowLowPrecisionAccumulationOnGPU": "1",             // Enable low precision acceleration
				}
				log.Printf("Attempting to enable CoreML execution provider (macOS GPU/Neural Engine)...")
				if err := sessionOptions.AppendExecutionProviderCoreMLV2(coreMLOptions); err != nil {
					log.Printf("WARNING: CoreML execution provider NOT available: %v", err)
					log.Printf("INFO: Falling back to CPU execution provider (no GPU acceleration)")
				} else {
					log.Printf("SUCCESS: CoreML execution provider enabled with GPU/Neural Engine (MLComputeUnits=ALL)")
				}
			} else if runtime.GOOS == "linux" {
				// Linux - try CUDA for NVIDIA GPU acceleration
				log.Printf("Attempting to enable CUDA execution provider (NVIDIA GPU)...")
				cudaProviderOptions, err := ort.NewCUDAProviderOptions()
				if err != nil {
					log.Printf("WARNING: Failed to create CUDA provider options: %v", err)
					log.Printf("INFO: Falling back to CPU execution provider (no GPU acceleration)")
				} else {
					cudaSettings := map[string]string{
						"device_id":                       "0",  // Use first GPU
						"gpu_mem_limit":                   "0",  // No memory limit (use all available)
						"arena_extend_strategy":           "kSameAsRequested",
						"cudnn_conv_algo_search":          "EXHAUSTIVE",
						"do_copy_in_default_stream":       "1",
						"cudnn_conv_use_max_workspace":    "1",
					}
					if err := cudaProviderOptions.Update(cudaSettings); err != nil {
						log.Printf("WARNING: Failed to update CUDA options: %v", err)
						cudaProviderOptions.Destroy()
					} else if err := sessionOptions.AppendExecutionProviderCUDA(cudaProviderOptions); err != nil {
						log.Printf("WARNING: CUDA execution provider NOT available: %v", err)
						log.Printf("INFO: Falling back to CPU execution provider (no GPU acceleration)")
						log.Printf("INFO: To enable GPU on Linux, install CUDA toolkit and ONNX Runtime with CUDA support")
						cudaProviderOptions.Destroy()
					} else {
						log.Printf("SUCCESS: CUDA execution provider enabled (NVIDIA GPU acceleration)")
						cudaProviderOptions.Destroy()
					}
				}
			} else {
				// Other platforms - CPU only
				log.Printf("Platform %s - using CPU execution provider", runtime.GOOS)
			}
		} else {
			// Silently enable GPU for remaining sessions
			if runtime.GOOS == "darwin" {
				coreMLOptions := map[string]string{
					"ModelFormat":                        "NeuralNetwork",
					"MLComputeUnits":                     "ALL",
					"RequireStaticInputShapes":           "0",
					"EnableOnSubgraphs":                  "0",
					"AllowLowPrecisionAccumulationOnGPU": "1",
				}
				sessionOptions.AppendExecutionProviderCoreMLV2(coreMLOptions)
			} else if runtime.GOOS == "linux" {
				cudaProviderOptions, err := ort.NewCUDAProviderOptions()
				if err == nil {
					cudaSettings := map[string]string{
						"device_id":                       "0",
						"gpu_mem_limit":                   "0",
						"arena_extend_strategy":           "kSameAsRequested",
						"cudnn_conv_algo_search":          "EXHAUSTIVE",
						"do_copy_in_default_stream":       "1",
						"cudnn_conv_use_max_workspace":    "1",
					}
					cudaProviderOptions.Update(cudaSettings)
					sessionOptions.AppendExecutionProviderCUDA(cudaProviderOptions)
					cudaProviderOptions.Destroy()
				}
			}
		}

		// Pre-allocate tensors with FIXED shapes
		inputData := make([]int64, fixedBatchSize*fixedSeqLength)
		maskData := make([]int64, fixedBatchSize*fixedSeqLength)

		inputTensor, err := ort.NewTensor(ort.NewShape(fixedBatchSize, fixedSeqLength), inputData)
		if err != nil {
			sessionOptions.Destroy()
			return nil, fmt.Errorf("failed to create input tensor: %w", err)
		}

		maskTensor, err := ort.NewTensor(ort.NewShape(fixedBatchSize, fixedSeqLength), maskData)
		if err != nil {
			inputTensor.Destroy()
			sessionOptions.Destroy()
			return nil, fmt.Errorf("failed to create mask tensor: %w", err)
		}

		outputShape := ort.NewShape(fixedBatchSize, fixedSeqLength, int64(embeddingDimension))
		outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
		if err != nil {
			inputTensor.Destroy()
			maskTensor.Destroy()
			sessionOptions.Destroy()
			return nil, fmt.Errorf("failed to create output tensor: %w", err)
		}

		// Create the session ONCE with these fixed tensors
		// This is where CoreML initialization happens - very expensive!
		// Note: Most sentence-transformers models only need input_ids and attention_mask
		session, err := ort.NewAdvancedSession(
			modelPath,
			[]string{"input_ids", "attention_mask"},
			[]string{"last_hidden_state"},
			[]ort.Value{inputTensor, maskTensor},
			[]ort.Value{outputTensor},
			sessionOptions,
		)
		if err != nil {
			inputTensor.Destroy()
			maskTensor.Destroy()
			outputTensor.Destroy()
			sessionOptions.Destroy()
			return nil, fmt.Errorf("failed to create ONNX session: %w", err)
		}

		if i == 0 {
			log.Printf("First session created successfully (this initialized CoreML)")
		}

		wrapper := &SessionWrapper{
			session:      session,
			inputTensor:  inputTensor,
			maskTensor:   maskTensor,
			outputTensor: outputTensor,
			inputData:    inputData,
			maskData:     maskData,
			batchSize:    fixedBatchSize,
			seqLength:    fixedSeqLength,
		}

		sessionPool <- wrapper
	}

	log.Printf("Session pool ready with %d reusable sessions (CoreML initialized)", poolSize)

	gen := &OnnxEmbeddingGenerator{
		config:          config,
		modelPath:       modelPath,
		modelDir:        absModelDir,
		tokenizer:       tokenizer,
		dimensions:      embeddingDimension,
		maxSeqLength:    maxSeqLength,
		sessionPool:     sessionPool,
		poolSize:        poolSize,
		fixedBatchSize:  fixedBatchSize,
		fixedSeqLength:  fixedSeqLength,
	}

	return gen, nil
}

// getSessionOptions creates session options with optimizations
func (g *OnnxEmbeddingGenerator) getSessionOptions() (*ort.SessionOptions, error) {
	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}

	// Try to use CoreML execution provider for Apple Silicon GPU
	log.Printf("Attempting to enable CoreML execution provider...")
	if err := sessionOptions.AppendExecutionProviderCoreML(0); err != nil {
		log.Printf("WARNING: CoreML execution provider NOT available: %v", err)
		log.Printf("INFO: Falling back to CPU execution provider (no GPU acceleration)")
		log.Printf("INFO: This will be slower than GPU-accelerated inference")
	} else {
		log.Printf("SUCCESS: CoreML execution provider enabled - using GPU/Neural Engine acceleration")
	}

	return sessionOptions, nil
}

// Generate generates an embedding for a single text
func (g *OnnxEmbeddingGenerator) Generate(text string) ([]float64, error) {
	if text == "" {
		return make([]float64, g.dimensions), nil
	}

	// Tokenize
	inputIDs, attentionMask := g.tokenizer.Tokenize(text)
	seqLen := int64(len(inputIDs))

	// Create input tensors (ONNX expects int64)
	inputTensor, err := ort.NewTensor(ort.NewShape(1, seqLen), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	maskTensor, err := ort.NewTensor(ort.NewShape(1, seqLen), attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	// Create output tensor (pre-allocated)
	// Output shape: [batch_size, sequence_length, hidden_size] = [1, seqLen, dimensions]
	outputShape := ort.NewShape(1, seqLen, int64(g.dimensions))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	// Set up session options for CoreML
	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer sessionOptions.Destroy()

	// Try to use CoreML execution provider for Apple Silicon GPU
	if err := sessionOptions.AppendExecutionProviderCoreML(0); err != nil {
		// Silently fall back to CPU
	}

	// Create session - each session is independent, no mutex needed
	// Sessions are created per-call for thread safety (no shared state)
	// Note: Most sentence-transformers models only need input_ids and attention_mask
	session, err := ort.NewAdvancedSession(
		g.modelPath,
		[]string{"input_ids", "attention_mask"},
		[]string{"last_hidden_state"},
		[]ort.Value{inputTensor, maskTensor},
		[]ort.Value{outputTensor},
		sessionOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}
	defer session.Destroy()

	// Run inference
	err = session.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	// Extract output data
	outputData := outputTensor.GetData()

	// Mean pooling over sequence dimension
	embedding := g.meanPooling(outputData, attentionMask, g.dimensions)

	// Normalize embedding
	embedding = g.normalize(embedding)

	// Convert to float64
	result := make([]float64, len(embedding))
	for i, v := range embedding {
		result[i] = float64(v)
	}

	return result, nil
}

// GenerateBatch generates embeddings for multiple texts using pre-created reusable sessions
func (g *OnnxEmbeddingGenerator) GenerateBatch(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	fixedBatchSize := int(g.fixedBatchSize)

	// Handle batches larger than fixed size by processing sequentially
	// Each worker goroutine already provides parallelism - no need to spawn more goroutines here
	if len(texts) > fixedBatchSize {
		// Split into chunks and process sequentially (worker-level parallelism is enough)
		allResults := make([][]float64, 0, len(texts))

		for i := 0; i < len(texts); i += fixedBatchSize {
			end := i + fixedBatchSize
			if end > len(texts) {
				end = len(texts)
			}

			// Process this chunk sequentially (no additional goroutines)
			chunkVectors, err := g.generateBatchSingle(texts[i:end])
			if err != nil {
				return nil, err
			}

			allResults = append(allResults, chunkVectors...)
		}

		return allResults, nil
	}

	// Single chunk - process directly
	return g.generateBatchSingle(texts)
}

// generateBatchSingle processes a single batch (size <= fixedBatchSize) using one session from the pool
func (g *OnnxEmbeddingGenerator) generateBatchSingle(texts []string) ([][]float64, error) {
	originalCount := len(texts)
	fixedBatchSize := int(g.fixedBatchSize)

	// Get a session wrapper from the pool (blocks if all in use)
	wrapper := <-g.sessionPool
	defer func() {
		// Return wrapper to pool for reuse
		g.sessionPool <- wrapper
	}()

	// Tokenize and fill the pre-allocated tensor data
	batchInputIDs := make([][]int64, fixedBatchSize)
	batchAttentionMasks := make([][]int64, fixedBatchSize)

	for i := 0; i < fixedBatchSize; i++ {
		if i < len(texts) && texts[i] != "" {
			inputIDs, attentionMask := g.tokenizer.Tokenize(texts[i])
			batchInputIDs[i] = inputIDs
			batchAttentionMasks[i] = attentionMask
		} else {
			// Empty text or padding gets zero embedding
			batchInputIDs[i] = []int64{g.tokenizer.clsToken, g.tokenizer.sepToken}
			batchAttentionMasks[i] = []int64{1, 1}
		}
	}

	// Fill the wrapper's pre-allocated tensor data (avoiding new allocations)
	startFill := time.Now()
	for i := 0; i < fixedBatchSize; i++ {
		offset := int(int64(i) * wrapper.seqLength)
		seqLen := len(batchInputIDs[i])

		// Clear previous data
		for j := 0; j < int(wrapper.seqLength); j++ {
			wrapper.inputData[offset+j] = g.tokenizer.padToken
			wrapper.maskData[offset+j] = 0
		}

		// Copy actual tokens
		copy(wrapper.inputData[offset:offset+seqLen], batchInputIDs[i])
		copy(wrapper.maskData[offset:offset+seqLen], batchAttentionMasks[i])
	}
	log.Printf("TIMING: Tensor fill took %v", time.Since(startFill))

	// Run inference using the pre-created session (THIS is the key optimization!)
	// No session creation overhead - just pure inference
	startInference := time.Now()
	err := wrapper.session.Run()
	log.Printf("TIMING: ONNX inference took %v", time.Since(startInference))
	if err != nil {
		return nil, fmt.Errorf("failed to run batch inference: %w", err)
	}

	// Extract output data from pre-allocated output tensor
	outputData := wrapper.outputTensor.GetData()

	// Process each item in the batch (but only return originalCount results)
	results := make([][]float64, originalCount)
	for i := 0; i < originalCount; i++ {
		// Extract hidden states for this batch item
		offset := i * int(wrapper.seqLength) * g.dimensions
		hiddenStates := outputData[offset : offset+int(wrapper.seqLength)*g.dimensions]

		// Mean pooling over sequence dimension
		embedding := g.meanPooling(hiddenStates, batchAttentionMasks[i], g.dimensions)

		// Normalize embedding
		embedding = g.normalize(embedding)

		// Convert to float64
		result := make([]float64, len(embedding))
		for j, v := range embedding {
			result[j] = float64(v)
		}

		results[i] = result
	}

	return results, nil
}

// GetDimensions returns the embedding dimension
func (g *OnnxEmbeddingGenerator) GetDimensions() int {
	return g.dimensions
}

// GetModelName returns the model name
func (g *OnnxEmbeddingGenerator) GetModelName() string {
	return g.config.Model
}

// Close releases resources
func (g *OnnxEmbeddingGenerator) Close() error {
	// Clean up session pool
	close(g.sessionPool)
	for wrapper := range g.sessionPool {
		if wrapper.session != nil {
			wrapper.session.Destroy()
		}
		if wrapper.inputTensor != nil {
			wrapper.inputTensor.Destroy()
		}
		if wrapper.maskTensor != nil {
			wrapper.maskTensor.Destroy()
		}
		if wrapper.outputTensor != nil {
			wrapper.outputTensor.Destroy()
		}
	}
	return nil
}

// meanPooling performs mean pooling over sequence dimension
func (g *OnnxEmbeddingGenerator) meanPooling(hiddenStates []float32, attentionMask []int64, hiddenSize int) []float32 {
	seqLen := len(attentionMask)

	// Initialize sum vector
	pooled := make([]float32, hiddenSize)

	// Sum vectors, weighted by attention mask
	var totalMask int64
	for i := 0; i < seqLen; i++ {
		if attentionMask[i] > 0 {
			for j := 0; j < hiddenSize; j++ {
				pooled[j] += hiddenStates[i*hiddenSize+j]
			}
			totalMask++
		}
	}

	// Divide by number of attended tokens
	if totalMask > 0 {
		for j := 0; j < hiddenSize; j++ {
			pooled[j] /= float32(totalMask)
		}
	}

	return pooled
}

// normalize performs L2 normalization
func (g *OnnxEmbeddingGenerator) normalize(vec []float32) []float32 {
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		result := make([]float32, len(vec))
		for i, v := range vec {
			result[i] = v / norm
		}
		return result
	}

	return vec
}

// LoadSimpleTokenizer loads a BERT-style tokenizer from vocab file
func LoadSimpleTokenizer(vocabPath string, maxSeqLength int) (*SimpleTokenizer, error) {
	vocab := make(map[string]int64)
	idToToken := make(map[int64]string)

	file, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vocab file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var id int64
	for scanner.Scan() {
		token := scanner.Text()
		vocab[token] = id
		idToToken[id] = token
		id++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading vocab file: %w", err)
	}

	// Find special tokens
	clsToken, ok := vocab["[CLS]"]
	if !ok {
		return nil, fmt.Errorf("missing [CLS] token in vocab")
	}

	sepToken, ok := vocab["[SEP]"]
	if !ok {
		return nil, fmt.Errorf("missing [SEP] token in vocab")
	}

	padToken, ok := vocab["[PAD]"]
	if !ok {
		return nil, fmt.Errorf("missing [PAD] token in vocab")
	}

	unkToken, ok := vocab["[UNK]"]
	if !ok {
		return nil, fmt.Errorf("missing [UNK] token in vocab")
	}

	log.Printf("Loaded tokenizer with %d tokens (max_seq_length: %d)", len(vocab), maxSeqLength)

	return &SimpleTokenizer{
		vocab:        vocab,
		idToToken:    idToToken,
		clsToken:     clsToken,
		sepToken:     sepToken,
		padToken:     padToken,
		unkToken:     unkToken,
		maxSeqLength: maxSeqLength,
	}, nil
}

// Tokenize converts text to token IDs and attention mask
func (t *SimpleTokenizer) Tokenize(text string) ([]int64, []int64) {
	// Simple whitespace + punctuation tokenization
	// This is a basic implementation - production would use WordPiece
	tokens := t.basicTokenize(text)

	// Convert to IDs
	ids := make([]int64, 0, len(tokens)+2)
	ids = append(ids, t.clsToken)

	for _, token := range tokens {
		if id, ok := t.vocab[token]; ok {
			ids = append(ids, id)
		} else {
			// Try lowercase
			if id, ok := t.vocab[strings.ToLower(token)]; ok {
				ids = append(ids, id)
			} else {
				ids = append(ids, t.unkToken)
			}
		}

		if len(ids) >= t.maxSeqLength-1 {
			break
		}
	}

	ids = append(ids, t.sepToken)

	// Create attention mask
	attentionMask := make([]int64, len(ids))
	for i := range attentionMask {
		attentionMask[i] = 1
	}

	// No padding needed if we're using dynamic shapes

	return ids, attentionMask
}

// basicTokenize performs basic whitespace and punctuation tokenization
func (t *SimpleTokenizer) basicTokenize(text string) []string {
	// Simple split on whitespace and punctuation
	text = strings.ToLower(text)

	// Replace punctuation with spaces
	replacer := strings.NewReplacer(
		".", " . ",
		",", " , ",
		"!", " ! ",
		"?", " ? ",
		":", " : ",
		";", " ; ",
		"(", " ( ",
		")", " ) ",
		"[", " [ ",
		"]", " ] ",
		"{", " { ",
		"}", " } ",
		"-", " - ",
	)
	text = replacer.Replace(text)

	// Split on whitespace and filter empty strings
	parts := strings.Fields(text)

	return parts
}
