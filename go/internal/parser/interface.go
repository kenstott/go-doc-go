package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Parser defines the interface all format-specific parsers must implement
type Parser interface {
	// GetName returns the parser identifier (e.g., "pdf", "docx", "csv")
	GetName() string

	// GetSupportedFormats returns file extensions this parser handles
	GetSupportedFormats() []string

	// Parse parses content and returns UDML elements
	Parse(ctx context.Context, req ParseRequest) (*ParseResult, error)

	// SupportsStreaming indicates if parser can handle streaming content
	SupportsStreaming() bool

	// Close releases any resources held by the parser
	Close() error
}

// ParseRequest encapsulates parsing input
type ParseRequest struct {
	ID       string                 // Document ID
	Content  interface{}            // File path string or io.Reader
	Metadata map[string]interface{} // Initial metadata
	Config   *ParserConfig          // Parser-specific configuration
}

// ParserConfig holds parser-specific configuration
type ParserConfig struct {
	MaxContentPreview int      // Content preview length (default: 100)
	ExtractImages     bool     // Extract image elements
	ExtractTables     bool     // Extract table structure
	ExtractHeaders    bool     // Extract headers/footers (DOCX/PPTX)
	ExtractComments   bool     // Extract comments/annotations
	MaxImageSize      int      // Maximum image size in bytes
	ExcludePatterns   []string // Patterns for excluding hyperlinks (from discovery config)
}

// DefaultParserConfig returns default configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		MaxContentPreview: 100,
		ExtractImages:     true,
		ExtractTables:     true,
		ExtractHeaders:    true,
		ExtractComments:   false,
		MaxImageSize:      10 * 1024 * 1024, // 10MB
	}
}

// ParserRegistry manages available parsers
type ParserRegistry struct {
	parsers map[string]Parser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser to the registry
func (r *ParserRegistry) Register(parser Parser) {
	r.parsers[parser.GetName()] = parser
}

// GetParser retrieves a parser by name
func (r *ParserRegistry) GetParser(name string) (Parser, error) {
	parser, exists := r.parsers[name]
	if !exists {
		return nil, fmt.Errorf("parser not found: %s", name)
	}
	return parser, nil
}

// GetParserForFile returns the appropriate parser for a file based on extension or filename
func (r *ParserRegistry) GetParserForFile(filename string) (Parser, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	basename := strings.ToLower(filepath.Base(filename))

	// Try to find a parser that supports this extension or filename
	for _, parser := range r.parsers {
		for _, format := range parser.GetSupportedFormats() {
			formatLower := strings.ToLower(format)
			// Handle both .ext and ext formats, and also full filename matches (e.g., "Dockerfile")
			if ext == formatLower || ext == "."+formatLower || basename == formatLower {
				return parser, nil
			}
		}
	}

	return nil, fmt.Errorf("no parser found for file: %s (extension: %s)", filename, ext)
}

// GetAll returns all registered parsers
func (r *ParserRegistry) GetAll() []Parser {
	parsers := make([]Parser, 0, len(r.parsers))
	for _, parser := range r.parsers {
		parsers = append(parsers, parser)
	}
	return parsers
}

// ListParsers returns names of all registered parsers
func (r *ParserRegistry) ListParsers() []string {
	names := make([]string, 0, len(r.parsers))
	for name := range r.parsers {
		names = append(names, name)
	}
	return names
}
