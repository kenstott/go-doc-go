package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockParser is a test implementation of the Parser interface
type MockParser struct {
	name              string
	supportedFormats  []string
	supportsStreaming bool
	parseFunc         func(ctx context.Context, req parser.ParseRequest) (*parser.ParseResult, error)
	closeFunc         func() error
}

func (m *MockParser) GetName() string {
	return m.name
}

func (m *MockParser) GetSupportedFormats() []string {
	return m.supportedFormats
}

func (m *MockParser) Parse(ctx context.Context, req parser.ParseRequest) (*parser.ParseResult, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, req)
	}
	return &parser.ParseResult{
		Document: parser.Document{
			ID:      req.ID,
			DocType: m.name,
		},
		Elements: []parser.Element{},
	}, nil
}

func (m *MockParser) SupportsStreaming() bool {
	return m.supportsStreaming
}

func (m *MockParser) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// TestParserInterface validates that parsers implement the interface correctly
func TestParserInterface(t *testing.T) {
	mockParser := &MockParser{
		name:              "test",
		supportedFormats:  []string{".txt", ".test"},
		supportsStreaming: false,
	}

	t.Run("GetName returns parser name", func(t *testing.T) {
		assert.Equal(t, "test", mockParser.GetName())
	})

	t.Run("GetSupportedFormats returns format list", func(t *testing.T) {
		formats := mockParser.GetSupportedFormats()
		assert.Len(t, formats, 2)
		assert.Contains(t, formats, ".txt")
		assert.Contains(t, formats, ".test")
	})

	t.Run("SupportsStreaming returns boolean", func(t *testing.T) {
		assert.False(t, mockParser.SupportsStreaming())
	})

	t.Run("Parse accepts ParseRequest and returns ParseResult", func(t *testing.T) {
		ctx := context.Background()
		req := parser.ParseRequest{
			ID:      "test-doc-1",
			Content: "test content",
			Config:  parser.DefaultParserConfig(),
		}

		result, err := mockParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-doc-1", result.Document.ID)
	})

	t.Run("Close releases resources", func(t *testing.T) {
		err := mockParser.Close()
		assert.NoError(t, err)
	})
}

// TestParserRegistry validates ParserRegistry functionality
func TestParserRegistry(t *testing.T) {
	t.Run("NewParserRegistry creates empty registry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		assert.NotNil(t, registry)
		assert.Empty(t, registry.ListParsers())
	})

	t.Run("Register adds parser to registry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		mockParser := &MockParser{
			name:             "test",
			supportedFormats: []string{".txt"},
		}

		registry.Register(mockParser)
		parsers := registry.ListParsers()
		assert.Len(t, parsers, 1)
		assert.Contains(t, parsers, "test")
	})

	t.Run("GetParser retrieves registered parser", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		mockParser := &MockParser{
			name:             "test",
			supportedFormats: []string{".txt"},
		}

		registry.Register(mockParser)
		retrieved, err := registry.GetParser("test")
		require.NoError(t, err)
		assert.Equal(t, "test", retrieved.GetName())
	})

	t.Run("GetParser returns error for unknown parser", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		_, err := registry.GetParser("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parser not found")
	})

	t.Run("GetParserForFile finds parser by extension", func(t *testing.T) {
		registry := parser.NewParserRegistry()

		pdfParser := &MockParser{
			name:             "pdf",
			supportedFormats: []string{".pdf"},
		}
		csvParser := &MockParser{
			name:             "csv",
			supportedFormats: []string{".csv"},
		}

		registry.Register(pdfParser)
		registry.Register(csvParser)

		// Test with .pdf extension
		retrieved, err := registry.GetParserForFile("document.pdf")
		require.NoError(t, err)
		assert.Equal(t, "pdf", retrieved.GetName())

		// Test with .csv extension
		retrieved, err = registry.GetParserForFile("data.csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", retrieved.GetName())
	})

	t.Run("GetParserForFile handles extension without dot", func(t *testing.T) {
		registry := parser.NewParserRegistry()

		txtParser := &MockParser{
			name:             "txt",
			supportedFormats: []string{"txt"}, // No dot prefix
		}

		registry.Register(txtParser)

		retrieved, err := registry.GetParserForFile("file.txt")
		require.NoError(t, err)
		assert.Equal(t, "txt", retrieved.GetName())
	})

	t.Run("GetParserForFile returns error for unsupported extension", func(t *testing.T) {
		registry := parser.NewParserRegistry()

		pdfParser := &MockParser{
			name:             "pdf",
			supportedFormats: []string{".pdf"},
		}
		registry.Register(pdfParser)

		_, err := registry.GetParserForFile("document.docx")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no parser found")
	})

	t.Run("GetAll returns all registered parsers", func(t *testing.T) {
		registry := parser.NewParserRegistry()

		parser1 := &MockParser{name: "parser1", supportedFormats: []string{".p1"}}
		parser2 := &MockParser{name: "parser2", supportedFormats: []string{".p2"}}
		parser3 := &MockParser{name: "parser3", supportedFormats: []string{".p3"}}

		registry.Register(parser1)
		registry.Register(parser2)
		registry.Register(parser3)

		allParsers := registry.GetAll()
		assert.Len(t, allParsers, 3)
	})
}

// TestParserConfig validates parser configuration
func TestParserConfig(t *testing.T) {
	t.Run("DefaultParserConfig returns sensible defaults", func(t *testing.T) {
		config := parser.DefaultParserConfig()

		assert.Equal(t, 100, config.MaxContentPreview)
		assert.True(t, config.ExtractImages)
		assert.True(t, config.ExtractTables)
		assert.True(t, config.ExtractHeaders)
		assert.False(t, config.ExtractComments)
		assert.Equal(t, 10*1024*1024, config.MaxImageSize) // 10MB
	})

	t.Run("ParserConfig can be customized", func(t *testing.T) {
		config := &parser.ParserConfig{
			MaxContentPreview: 200,
			ExtractImages:     false,
			ExtractTables:     true,
			ExtractHeaders:    false,
			ExtractComments:   true,
			MaxImageSize:      5 * 1024 * 1024, // 5MB
		}

		assert.Equal(t, 200, config.MaxContentPreview)
		assert.False(t, config.ExtractImages)
		assert.True(t, config.ExtractComments)
	})
}

// TestParseRequest validates ParseRequest structure
func TestParseRequest(t *testing.T) {
	t.Run("ParseRequest can be created with minimal fields", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "doc-123",
			Content: "/path/to/file.pdf",
		}

		assert.Equal(t, "doc-123", req.ID)
		assert.Equal(t, "/path/to/file.pdf", req.Content)
		assert.Nil(t, req.Metadata)
		assert.Nil(t, req.Config)
	})

	t.Run("ParseRequest can include metadata and config", func(t *testing.T) {
		metadata := map[string]interface{}{
			"source": "test",
			"author": "John Doe",
		}
		config := parser.DefaultParserConfig()

		req := parser.ParseRequest{
			ID:       "doc-456",
			Content:  "/path/to/file.pdf",
			Metadata: metadata,
			Config:   config,
		}

		assert.Equal(t, "doc-456", req.ID)
		assert.NotNil(t, req.Metadata)
		assert.Equal(t, "test", req.Metadata["source"])
		assert.NotNil(t, req.Config)
		assert.Equal(t, 100, req.Config.MaxContentPreview)
	})
}
