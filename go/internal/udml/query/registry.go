package query

import (
	"context"
	"fmt"
	"sync"
)

// BackendFactory is a function that creates a new backend instance
type BackendFactory func(config BackendConfig) (QueryBackend, error)

// BackendRegistry manages available query backends
type BackendRegistry struct {
	mu        sync.RWMutex
	factories map[string]BackendFactory
	instances map[string]QueryBackend // Cached backend instances
}

// Global registry instance
var globalRegistry = NewBackendRegistry()

// NewBackendRegistry creates a new backend registry
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{
		factories: make(map[string]BackendFactory),
		instances: make(map[string]QueryBackend),
	}
}

// Register registers a backend factory
func (r *BackendRegistry) Register(backendType string, factory BackendFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[backendType]; exists {
		return fmt.Errorf("backend type %s is already registered", backendType)
	}

	r.factories[backendType] = factory
	return nil
}

// Unregister removes a backend factory
func (r *BackendRegistry) Unregister(backendType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.factories, backendType)
	delete(r.instances, backendType)
}

// Create creates a new backend instance (does not cache)
func (r *BackendRegistry) Create(config BackendConfig) (QueryBackend, error) {
	r.mu.RLock()
	factory, exists := r.factories[config.Type]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown backend type: %s (available: %v)", config.Type, r.ListBackends())
	}

	return factory(config)
}

// GetOrCreate gets a cached backend or creates a new one
func (r *BackendRegistry) GetOrCreate(config BackendConfig) (QueryBackend, error) {
	r.mu.RLock()
	instance, exists := r.instances[config.Type]
	r.mu.RUnlock()

	if exists {
		return instance, nil
	}

	// Create new instance
	backend, err := r.Create(config)
	if err != nil {
		return nil, err
	}

	// Initialize the backend
	ctx := context.Background()
	if err := backend.Initialize(ctx, config); err != nil {
		backend.Close()
		return nil, fmt.Errorf("failed to initialize backend %s: %w", config.Type, err)
	}

	// Cache the instance
	r.mu.Lock()
	r.instances[config.Type] = backend
	r.mu.Unlock()

	return backend, nil
}

// Get retrieves a cached backend instance
func (r *BackendRegistry) Get(backendType string) (QueryBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, exists := r.instances[backendType]
	if !exists {
		return nil, fmt.Errorf("backend %s not found in registry", backendType)
	}

	return instance, nil
}

// ListBackends returns all registered backend types
func (r *BackendRegistry) ListBackends() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backends := make([]string, 0, len(r.factories))
	for backendType := range r.factories {
		backends = append(backends, backendType)
	}
	return backends
}

// IsRegistered checks if a backend type is registered
func (r *BackendRegistry) IsRegistered(backendType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[backendType]
	return exists
}

// CloseAll closes all cached backend instances
func (r *BackendRegistry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for backendType, backend := range r.instances {
		if err := backend.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", backendType, err))
		}
	}

	// Clear instances
	r.instances = make(map[string]QueryBackend)

	if len(errs) > 0 {
		return fmt.Errorf("failed to close backends: %v", errs)
	}

	return nil
}

// Global registry functions

// Register registers a backend factory in the global registry
func Register(backendType string, factory BackendFactory) error {
	return globalRegistry.Register(backendType, factory)
}

// Unregister removes a backend factory from the global registry
func Unregister(backendType string) {
	globalRegistry.Unregister(backendType)
}

// CreateBackend creates a new backend instance from the global registry
func CreateBackend(config BackendConfig) (QueryBackend, error) {
	return globalRegistry.Create(config)
}

// GetOrCreateBackend gets a cached backend or creates a new one from the global registry
func GetOrCreateBackend(config BackendConfig) (QueryBackend, error) {
	return globalRegistry.GetOrCreate(config)
}

// GetBackend retrieves a cached backend from the global registry
func GetBackend(backendType string) (QueryBackend, error) {
	return globalRegistry.Get(backendType)
}

// ListBackends returns all registered backend types from the global registry
func ListBackends() []string {
	return globalRegistry.ListBackends()
}

// IsBackendRegistered checks if a backend type is registered in the global registry
func IsBackendRegistered(backendType string) bool {
	return globalRegistry.IsRegistered(backendType)
}

// CloseAllBackends closes all cached backend instances in the global registry
func CloseAllBackends() error {
	return globalRegistry.CloseAll()
}

// BackendInfo provides information about a registered backend
type BackendInfo struct {
	Type         string
	Name         string
	Version      string
	Capabilities BackendCapabilities
}

// GetBackendInfo retrieves information about a backend
func (r *BackendRegistry) GetBackendInfo(backendType string) (*BackendInfo, error) {
	backend, err := r.Get(backendType)
	if err != nil {
		return nil, err
	}

	return &BackendInfo{
		Type:    backendType,
		Name:    backend.GetName(),
		Version: backend.GetVersion(),
	}, nil
}

// MockBackend is a simple mock backend for testing
type MockBackend struct {
	name         string
	version      string
	capabilities BackendCapabilities
	initialized  bool
}

// NewMockBackend creates a new mock backend
func NewMockBackend(name string) *MockBackend {
	return &MockBackend{
		name:    name,
		version: "1.0.0",
		capabilities: BackendCapabilities{
			SupportsJSONPath:        true,
			SupportsSimilarity:      false,
			SupportsRegex:           true,
			SupportsFullTextSearch:  false,
			SupportsGraphTraversal:  false,
			SupportsPartitionPruning: true,
			SupportsVectorized:      true,
			SupportsParallel:        true,
		},
	}
}

func (m *MockBackend) GetName() string {
	return m.name
}

func (m *MockBackend) GetVersion() string {
	return m.version
}

func (m *MockBackend) Initialize(ctx context.Context, config BackendConfig) error {
	m.initialized = true
	return nil
}

func (m *MockBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
	return &NativeQuery{
		BackendType: m.name,
		QueryString: fmt.Sprintf("SELECT * FROM %s", expr.From),
		SourceExpr:  expr,
	}, nil
}

func (m *MockBackend) Execute(ctx context.Context, query *NativeQuery) (*QueryResult, error) {
	return &QueryResult{
		Columns: []ColumnInfo{
			{Name: "element_id", Type: "string", Nullable: false},
			{Name: "content", Type: "string", Nullable: true},
		},
		Rows:        []Row{},
		RowCount:    0,
		BackendType: m.name,
	}, nil
}

func (m *MockBackend) SupportsFeature(feature string) bool {
	switch feature {
	case "jsonpath":
		return m.capabilities.SupportsJSONPath
	case "similarity":
		return m.capabilities.SupportsSimilarity
	case "regex":
		return m.capabilities.SupportsRegex
	case "full_text_search":
		return m.capabilities.SupportsFullTextSearch
	case "graph_traversal":
		return m.capabilities.SupportsGraphTraversal
	default:
		return false
	}
}

func (m *MockBackend) Explain(ctx context.Context, query *NativeQuery) (*QueryPlan, error) {
	return &QueryPlan{
		BackendName: m.name,
		RawPlan:     "Mock execution plan",
		Steps: []QueryPlanStep{
			{Operator: "SCAN", Description: "Scan elements table", EstimatedRows: 1000, EstimatedCost: 1.0},
		},
	}, nil
}

func (m *MockBackend) Close() error {
	m.initialized = false
	return nil
}
