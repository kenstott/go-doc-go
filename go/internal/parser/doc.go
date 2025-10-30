// Package parser provides document parsing for various file formats.
//
// The parser package implements format-specific parsers for extracting structured
// information from documents. All parsers implement the Parser interface and return
// Universal Elements compatible with the UDML (Universal Document Markup Language)
// specification.
//
// # Supported Formats
//
// The package includes parsers for:
//
// **Code Files** (11 languages):
//   - Go (.go)
//   - Java (.java)
//   - Python (.py)
//   - JavaScript/TypeScript (.js, .ts, .jsx, .tsx)
//   - Ruby (.rb)
//   - Rust (.rs)
//   - C/C++ (.c, .cpp, .h, .hpp)
//   - Bash (.sh)
//   - HCL (.hcl, .tf)
//   - Protobuf (.proto)
//   - Dockerfile
//   - PHP (.php)
//
// **Data Formats**:
//   - JSON (.json)
//   - CSV (.csv)
//   - XML (.xml)
//   - YAML (.yaml, .yml)
//   - Parquet (.parquet)
//
// **Documents**:
//   - PDF (.pdf)
//   - HTML (.html, .htm)
//   - Markdown (.md)
//   - Plain Text (.txt)
//
// **Office Documents**:
//   - Word (.docx)
//   - Excel (.xlsx)
//   - PowerPoint (.pptx)
//
// # Usage
//
// Create a parser registry and register parsers:
//
//	registry := parser.NewParserRegistry()
//	registry.Register(parser.NewPdfParser())
//	registry.Register(parser.NewDocxParser())
//
//	// Get parser for a specific file
//	parser, err := registry.GetParserForFile("document.pdf")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Parse the document
//	req := parser.ParseRequest{
//		ID:      "doc123",
//		Content: "path/to/document.pdf",
//		Config:  parser.DefaultParserConfig(),
//	}
//	result, err := parser.Parse(context.Background(), req)
//	if err != nil {
//		log.Fatalf("parse failed: %v", err)
//	}
//
//	fmt.Printf("Extracted %d elements\n", len(result.Elements))
//
// # Universal Elements
//
// All parsers produce Element structs with consistent fields:
//
//   - ElementID: Unique identifier
//   - ElementType: Type from element taxonomy (heading, paragraph, table, etc.)
//   - ElementCategory: Category (container, content, structure, component, metadata)
//   - Content: Full element content
//   - ContentPreview: Truncated preview (default 100 chars)
//   - ParentID: Parent element ID (for hierarchy)
//   - Position: Sibling order position
//   - Depth: Nesting depth in document hierarchy
//
// # Query-Optimized Fields
//
// UDML Phase 1 adds promoted fields for common queries:
//
//   - PageNumber: Page location (PDF, DOCX, PPTX)
//   - SectionLevel: Heading hierarchy level
//   - RowIndex, ColumnIndex: Table cell positions
//   - TemporalType: Date/time type classification
//   - TagName: HTML/XML tag identifier
//
// UDML Phase 2 adds code-specific fields:
//
//   - FunctionName: Function/method name
//   - ClassName: Class/struct/interface name
//   - Namespace: Package/module path
//   - LineNumber: Source code line number
//   - Signature: Function signature or type definition
//
// These promoted fields enable 60-1000x faster queries compared to JSON extraction.
//
// # Element Types
//
// Common element types produced by parsers:
//
//   - heading: Document headings (h1-h6)
//   - paragraph: Text paragraphs
//   - table: Tabular data
//   - table_row, table_cell: Table structure
//   - list_item: List items
//   - image: Images and figures
//   - hyperlink: Links and URLs
//   - code_block: Code snippets
//   - code_function: Function definitions
//   - code_class: Class definitions
//   - code_import: Import/dependency statements
//   - code_comment: Code comments
//
// See element_taxonomy.json for the complete element type reference.
//
// # Configuration
//
// Parsers accept a ParserConfig for customization:
//
//	config := &parser.ParserConfig{
//		MaxContentPreview: 200,        // Content preview length
//		ExtractImages:     true,       // Extract image elements
//		ExtractTables:     true,       // Extract table structure
//		ExtractHeaders:    true,       // Extract headers/footers
//		ExtractComments:   false,      // Extract comments
//		MaxImageSize:      10 * 1024 * 1024,  // 10MB limit
//		ExcludePatterns:   []string{"*.test.*"},  // Exclude patterns
//	}
//
// # Parser Interface
//
// To implement a custom parser:
//
//	type CustomParser struct{}
//
//	func (p *CustomParser) GetName() string {
//		return "custom"
//	}
//
//	func (p *CustomParser) GetSupportedFormats() []string {
//		return []string{".custom"}
//	}
//
//	func (p *CustomParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
//		// Parse logic here
//		return &ParseResult{
//			Document: Document{ID: req.ID, DocType: "custom"},
//			Elements: []Element{...},
//		}, nil
//	}
//
//	func (p *CustomParser) SupportsStreaming() bool {
//		return false
//	}
//
//	func (p *CustomParser) Close() error {
//		return nil
//	}
//
// # Error Handling
//
// Parsers return errors for:
//
//   - Invalid file formats
//   - Corrupted documents
//   - Unsupported features
//   - I/O failures
//
// Errors are wrapped with context using fmt.Errorf with %w for error chain inspection.
//
// # Performance
//
// Parser performance characteristics:
//
//   - **PDF**: ~1s per 10MB file, memory usage ~5x file size
//   - **DOCX**: ~500ms per 5MB file, memory usage ~3x file size
//   - **Code**: ~100ms per 1MB file, memory usage ~2x file size
//   - **JSON/CSV**: ~200ms per 10MB file, memory usage ~4x file size
//
// Most parsers support concurrent execution. Use multiple goroutines with
// separate parser instances for parallel processing.
//
// # Code Parsers
//
// Code parsers extract semantic elements:
//
//   - Imports and dependencies (for discovery/crawling)
//   - Function and method definitions
//   - Class and interface definitions
//   - Comments and documentation
//   - Code blocks and statements
//
// Code parsers populate code-specific fields (FunctionName, ClassName, Namespace,
// LineNumber, Signature) for efficient querying.
//
// # Document Hierarchy
//
// Parsers build parent-child relationships via the ParentID field:
//
//	Document (root)
//	├─ Section (heading)
//	│  ├─ Paragraph
//	│  └─ Table
//	│     ├─ TableRow
//	│     │  ├─ TableCell
//	│     │  └─ TableCell
//	│     └─ TableRow
//	└─ Section (heading)
//	   └─ Paragraph
//
// The Depth field indicates nesting level, and Position indicates sibling order.
//
// # Registry Pattern
//
// The ParserRegistry provides parser discovery and selection:
//
//	// Get all registered parsers
//	allParsers := registry.GetAll()
//
//	// List parser names
//	names := registry.ListParsers()
//
//	// Get specific parser
//	pdfParser, _ := registry.GetParser("pdf")
//
//	// Get parser by file extension
//	parser, _ := registry.GetParserForFile("report.xlsx")
//
// # Content Input
//
// Parsers accept content as:
//
//   - File path string: "path/to/document.pdf"
//   - io.Reader: bytes.NewReader(data) or open file
//
// The ParseRequest.Content field is interface{} to support both:
//
//	// File path
//	req := ParseRequest{
//		Content: "/tmp/document.pdf",
//	}
//
//	// Reader
//	req := ParseRequest{
//		Content: bytes.NewReader(documentBytes),
//	}
//
// # See Also
//
//   - Universal Element Types: docs/universal-element-types.md
//   - UDML Specification: docs/features/udml/
//   - Parser Configuration: docs/configuration/README.md
//   - Adding Custom Parsers: docs/development/adding-parsers.md
package parser
