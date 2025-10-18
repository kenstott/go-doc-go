package parser

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// HTMLElementType represents the type of an HTML element
type HTMLElementType string

// HTML element type constants - UDML canonical types from element_taxonomy.json
const (
	HTMLElementTypeRoot        HTMLElementType = "root"
	HTMLElementTypeHeader      HTMLElementType = "header"
	HTMLElementTypeHeading     HTMLElementType = "heading"
	HTMLElementTypeParagraph   HTMLElementType = "paragraph"
	HTMLElementTypeList        HTMLElementType = "list"
	HTMLElementTypeListItem    HTMLElementType = "list_item"
	HTMLElementTypeTable       HTMLElementType = "table"
	HTMLElementTypeTableRow    HTMLElementType = "table_row"
	HTMLElementTypeTableHeader HTMLElementType = "table_header"
	HTMLElementTypeTableCell   HTMLElementType = "table_cell"
	HTMLElementTypeImage       HTMLElementType = "image"
	HTMLElementTypeCodeBlock   HTMLElementType = "code_block"
	HTMLElementTypeBlockquote  HTMLElementType = "blockquote"
	HTMLElementTypeDiv         HTMLElementType = "div"
	HTMLElementTypeArticle     HTMLElementType = "article"
	HTMLElementTypeSection     HTMLElementType = "section"
	HTMLElementTypeSpan        HTMLElementType = "span"
	HTMLElementTypeBody        HTMLElementType = "body"
	HTMLElementTypeText        HTMLElementType = "text"
	HTMLElementTypeHyperlink   HTMLElementType = "hyperlink" // UDML canonical: hyperlink (not "link")
	HTMLElementTypeTemporal    HTMLElementType = "temporal"
	HTMLElementTypeDiagram     HTMLElementType = "diagram"   // For figure, chart
	HTMLElementTypeCaption     HTMLElementType = "caption"
)

// HTMLElement represents a parsed HTML element
type HTMLElement struct {
	ElementID       string                 `json:"element_id"`
	ElementType     HTMLElementType        `json:"element_type"`
	ParentID        string                 `json:"parent_id,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ElementOrder    int                    `json:"element_order"`
	DocumentOrder   int                    `json:"document_position"`
	Metadata        map[string]interface{} `json:"metadata"`
	Text            string                 `json:"text,omitempty"`
	Content         string                 `json:"content,omitempty"`
}

// HTMLParseRequest represents the input for HTML parsing (deprecated - use parser.ParseRequest)
// Kept temporarily during migration to new Parser interface
type HTMLParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ParseResponse represents the output of HTML parsing
type ParseResponse struct {
	Document map[string]interface{} `json:"document"`
	Elements []HTMLElement          `json:"elements"`
	Dates    map[string]interface{} `json:"dates,omitempty"`
}

// HTMLParser handles HTML document parsing
type HTMLParser struct {
	MaxContentPreview int
	ExtractDates      bool
	EnableCaching     bool
	StoreFullContent  bool              // If true, populate Content field with full (untruncated) text
	Headers           map[string]string // HTTP headers to use when fetching URLs
	documentCache     map[string]string // In-memory cache of HTML content by source URL
	diskCache         *ContentCache     // Disk-based persistent cache
	cacheMu           sync.RWMutex      // Mutex for thread-safe cache access
}

// NewHTMLParser creates a new HTML parser instance
func NewHTMLParser() *HTMLParser {
	// Initialize disk cache with default config
	diskCache, err := NewContentCache(DefaultCacheConfig())
	if err != nil {
		// Log warning but continue - memory cache will still work
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize disk cache: %v\n", err)
		diskCache = nil
	}

	return &HTMLParser{
		MaxContentPreview: 100,
		ExtractDates:      true,
		StoreFullContent:  false, // Default: don't store full content
		EnableCaching:     true,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (compatible; Go-Doc-Go/1.0; +https://github.com/go-doc-go)",
		},
		documentCache: make(map[string]string),
		diskCache:     diskCache,
	}
}

// NewHTMLParserWithCache creates a new HTML parser with custom cache config
func NewHTMLParserWithCache(cacheConfig ContentCacheConfig) (*HTMLParser, error) {
	diskCache, err := NewContentCache(cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize disk cache: %w", err)
	}

	return &HTMLParser{
		MaxContentPreview: 100,
		ExtractDates:      true,
		StoreFullContent:  false,
		EnableCaching:     true,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (compatible; Go-Doc-Go/1.0; +https://github.com/go-doc-go)",
		},
		documentCache: make(map[string]string),
		diskCache:     diskCache,
	}, nil
}

// Parser interface implementation

// GetName returns the parser name
func (p *HTMLParser) GetName() string {
	return "html"
}

// GetSupportedFormats returns supported file formats
func (p *HTMLParser) GetSupportedFormats() []string {
	return []string{".html", ".htm", "html"}
}

// Parse implements the Parser interface for HTML documents
func (p *HTMLParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract content from request
	var htmlContent string
	switch v := req.Content.(type) {
	case string:
		htmlContent = v
	case []byte:
		htmlContent = string(v)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", req.Content)
	}

	// Cache the HTML content for later resolution
	p.cacheMu.Lock()
	p.documentCache[req.ID] = htmlContent
	p.cacheMu.Unlock()

	// Create HTML-specific request
	htmlRequest := HTMLParseRequest{
		ID:       req.ID,
		Content:  htmlContent,
		Metadata: req.Metadata,
	}

	// Parse using existing implementation
	response, err := p.parseHTML(htmlRequest)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult
	return p.convertToParseResult(response), nil
}

// SupportsStreaming returns whether the parser supports streaming
func (p *HTMLParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *HTMLParser) Close() error {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.documentCache = make(map[string]string)
	return nil
}

// ParseLegacy is the legacy interface that converts HTML content to ParseResult (deprecated)
func (p *HTMLParser) ParseLegacy(docID string, content interface{}) (*ParseResult, error) {
	// Handle different input types
	var htmlContent string
	switch v := content.(type) {
	case string:
		htmlContent = v
	case []byte:
		htmlContent = string(v)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Cache the HTML content for later resolution (contextual embeddings)
	p.cacheMu.Lock()
	p.documentCache[docID] = htmlContent
	p.cacheMu.Unlock()

	// Create request structure for compatibility
	request := HTMLParseRequest{
		ID:       docID,
		Content:  htmlContent,
		Metadata: make(map[string]interface{}),
	}

	// Parse using existing implementation
	response, err := p.parseHTML(request)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult
	return p.convertToParseResult(response), nil
}

// parseHTML parses an HTML document into structured elements (internal implementation)
func (p *HTMLParser) parseHTML(request HTMLParseRequest) (*ParseResponse, error) {
	// Parse HTML content
	doc, err := html.Parse(strings.NewReader(request.Content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Initialize response
	response := &ParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "html",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements: []HTMLElement{},
	}

	// Create root element
	rootElement := HTMLElement{
		ElementID:       p.generateID("root_"),
		ElementType:     HTMLElementTypeRoot,
		ContentPreview:  p.truncateContent(request.Content),
		ContentLocation: p.createContentLocation(request.ID, string(HTMLElementTypeRoot), ""),
		ContentHash:     p.generateHash(request.Content),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, rootElement)

	// Parse document elements
	elementCounter := 1
	p.parseNode(doc, &response.Elements, rootElement.ElementID, request.ID, &elementCounter)

	return response, nil
}

// parseNode recursively parses HTML nodes and returns the ID of the element created (if any)
func (p *HTMLParser) parseNode(node *html.Node, elements *[]HTMLElement,
	parentID, sourceID string, counter *int) string {

	if node.Type == html.DocumentNode {
		// Document node - recurse into children without creating element
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			p.parseNode(child, elements, parentID, sourceID, counter)
		}
		return ""
	} else if node.Type == html.ElementNode {
		elementType := p.getElementType(node.Data)

		// Skip script, style, and other non-content elements
		if p.shouldSkipElement(node.Data) {
			return ""
		}

		// Special handling for anchor tags - create link elements
		if strings.ToLower(node.Data) == "a" {
			return p.parseAnchorTag(node, elements, parentID, sourceID, counter)
		}

		// Check if this tag should get a dedicated element (semantic tags only)
		// This matches Python's selective element creation in html.py:942-944
		if !p.shouldCreateElement(node.Data) {
			// Don't create element, but process children with same parent
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				p.parseNode(child, elements, parentID, sourceID, counter)
			}
			return ""
		}

		// Extract text content for this element
		textContent := p.extractTextContent(node) // Truncated preview

		// NOTE: We create elements even if they have no direct text content
		// Structural elements (ul, ol, section, nav, etc.) are important for hierarchy
		// The embedding system will skip empty ancestors when building context
		// This is the CORRECT behavior - Python's filtering at html.py:1050 is a bug

		// Create element for all whitelisted semantic tags
		metadata := p.extractMetadata(node)

		// Optionally get full text for temporal extraction
		fullText := ""
		if p.StoreFullContent {
			fullText = p.extractFullTextContent(node)
		}

		// Extract temporal metadata from text content (use full text if available, otherwise preview)
		contentForTemporal := textContent
		if fullText != "" {
			contentForTemporal = fullText
		}
		if contentForTemporal != "" {
			ProcessTemporalContent(contentForTemporal, metadata)
		}

		element := HTMLElement{
			ElementID:       p.generateID(fmt.Sprintf("%s_", strings.ToLower(string(elementType)))),
			ElementType:     elementType,
			ParentID:        parentID,
			ContentPreview:  textContent, // Always populate with truncated preview
			ContentLocation: p.createContentLocationForNode(sourceID, elementType, node),
			ContentHash:     p.generateHash(textContent),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata:        metadata,
		}

		// Optionally populate Content field with full (untruncated) text
		if fullText != "" {
			element.Content = fullText
		}

		*elements = append(*elements, element)
		*counter++

		// Recursively parse children
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			p.parseNode(child, elements, element.ElementID, sourceID, counter)
		}

		return element.ElementID
	} else if node.Type == html.TextNode {
		// Don't create separate text node elements
		// Text is captured by parent semantic elements via extractTextContent
		// This matches Python's approach which aggregates text into parent elements
		return ""
	}
	return ""
}

// getElementType maps HTML tag names to UDML canonical element types
// All mappings must use types defined in element_taxonomy.json
func (p *HTMLParser) getElementType(tagName string) HTMLElementType {
	switch strings.ToLower(tagName) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return HTMLElementTypeHeading // UDML: heading (not header)
	case "header":
		return HTMLElementTypeHeader // HTML5 header element
	case "p":
		return HTMLElementTypeParagraph
	case "ul", "ol":
		return HTMLElementTypeList
	case "li":
		return HTMLElementTypeListItem
	case "table":
		return HTMLElementTypeTable
	case "tr":
		return HTMLElementTypeTableRow
	case "th":
		return HTMLElementTypeTableHeader
	case "td":
		return HTMLElementTypeTableCell
	case "img":
		return HTMLElementTypeImage
	case "pre", "code":
		return HTMLElementTypeCodeBlock
	case "blockquote":
		return HTMLElementTypeBlockquote
	case "div":
		return HTMLElementTypeDiv
	case "article":
		return HTMLElementTypeArticle
	case "section":
		return HTMLElementTypeSection
	case "nav", "aside", "main":
		return HTMLElementTypeSection // Map nav/aside/main to section
	case "span":
		return HTMLElementTypeSpan
	case "body":
		return HTMLElementTypeBody
	case "a":
		return HTMLElementTypeHyperlink // UDML: hyperlink (not link)
	case "figure":
		return HTMLElementTypeDiagram // Map figure to diagram
	case "figcaption", "caption":
		return HTMLElementTypeCaption
	default:
		// Handle namespace-prefixed elements (XBRL, iXBRL, etc.)
		if strings.Contains(tagName, ":") {
			return HTMLElementType(strings.ReplaceAll(tagName, ":", "_"))
		}
		// For unmapped tags, use div as fallback (generic container)
		return HTMLElementTypeDiv
	}
}

// shouldSkipElement determines if an element should be skipped during parsing
func (p *HTMLParser) shouldSkipElement(tagName string) bool {
	skipTags := map[string]bool{
		"script":   true,
		"style":    true,
		"noscript": true,
		"meta":     true,
		"link":     true,
		"title":    true,
		"head":     true,
	}
	return skipTags[strings.ToLower(tagName)]
}

// shouldCreateElement determines if we should create a dedicated element for this tag
// This includes all semantic content tags AND structural/container tags for hierarchy
// NOTE: This is the CORRECT design - Python has bugs where it:
//   1. Doesn't create li elements
//   2. Filters out empty container elements (ul, ol, section, nav)
// We create ALL whitelisted elements for proper hierarchy tracking
func (p *HTMLParser) shouldCreateElement(tagName string) bool {
	// Whitelist of semantic tags that should get dedicated elements
	// Based on Python's whitelist in html.py:942-944 but ADDS 'li' (bug fix)
	semanticTags := map[string]bool{
		// Headers and headings
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"header": true,
		// Text content
		"p": true,
		// Lists - containers AND items
		"ul": true, "ol": true, "li": true,
		// Code blocks
		"pre": true, "code": true,
		// Quotes
		"blockquote": true,
		// Tables - ONLY table container, NOT tr/td/th (those are handled in processTable)
		"table": true,
		// Media
		"img": true,
		"figure": true, "figcaption": true, "caption": true,
		// Structural/container elements (important for hierarchy)
		"body": true, "div": true, "article": true, "section": true, "nav": true, "aside": true, "main": true,
	}
	return semanticTags[strings.ToLower(tagName)]
}

// extractTextContent extracts text content from an HTML node
// This matches Python's approach in html.py:1033-1047:
// - Get only DIRECT text content for most elements (prevents container pollution)
// - Get ALL text for specific elements like p, h1-h6, li, th, td, blockquote, pre, code
func (p *HTMLParser) extractTextContent(node *html.Node) string {
	tagName := strings.ToLower(node.Data)

	// For specific element types, include all descendant text
	// This matches Python's list at html.py:1044
	includeAllText := map[string]bool{
		"p": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "th": true, "td": true, "blockquote": true, "pre": true, "code": true,
	}

	var text strings.Builder
	if includeAllText[tagName] {
		// Get all text from descendants
		p.extractTextRecursive(node, &text)
	} else {
		// Get only direct text children (not from deeper descendants)
		// This matches Python's approach at html.py:1035-1041
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				textContent := strings.TrimSpace(child.Data)
				if textContent != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(textContent)
				}
			}
		}
	}

	content := strings.TrimSpace(text.String())
	return p.truncateContent(content)
}

// extractFullTextContent extracts untruncated text content from an HTML node
// Used when StoreFullContent is true to populate the Content field
func (p *HTMLParser) extractFullTextContent(node *html.Node) string {
	tagName := strings.ToLower(node.Data)

	// For specific element types, include all descendant text
	includeAllText := map[string]bool{
		"p": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"li": true, "th": true, "td": true, "blockquote": true, "pre": true, "code": true,
	}

	var text strings.Builder
	if includeAllText[tagName] {
		// Get all text from descendants
		p.extractTextRecursive(node, &text)
	} else {
		// Get only direct text children
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				textContent := strings.TrimSpace(child.Data)
				if textContent != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(textContent)
				}
			}
		}
	}

	return strings.TrimSpace(text.String())
}

// extractTextRecursive recursively extracts text from HTML nodes
func (p *HTMLParser) extractTextRecursive(node *html.Node, text *strings.Builder) {
	if node.Type == html.TextNode {
		textContent := strings.TrimSpace(node.Data)
		if textContent != "" {
			if text.Len() > 0 {
				text.WriteString(" ")
			}
			text.WriteString(textContent)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.extractTextRecursive(child, text)
	}
}

// parseAnchorTag creates a hyperlink element for an HTML anchor tag
// Hyperlink elements are containers that can have text/image children
func (p *HTMLParser) parseAnchorTag(node *html.Node, elements *[]HTMLElement,
	parentID, sourceID string, counter *int) string {

	// Find href attribute
	var href string
	for _, attr := range node.Attr {
		if attr.Key == "href" {
			href = attr.Val
			break
		}
	}

	// Extract text content for preview (truncated)
	linkText := p.extractTextContent(node)

	// Create hyperlink element (UDML canonical type)
	linkElement := HTMLElement{
		ElementID:       p.generateID("hyperlink_"),
		ElementType:     HTMLElementTypeHyperlink,
		ParentID:        parentID,
		ContentPreview:  linkText,
		ContentLocation: p.createContentLocationForNode(sourceID, HTMLElementTypeHyperlink, node),
		ContentHash:     p.generateHash(linkText),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"link_target": href,
			"link_type":   "html",
		},
	}

	// Store full text if enabled
	if p.StoreFullContent {
		fullText := p.extractFullTextContent(node)
		linkElement.Content = fullText
	}

	*elements = append(*elements, linkElement)
	*counter++

	// Recursively parse children (text, images, etc.) with hyperlink as parent
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.parseNode(child, elements, linkElement.ElementID, sourceID, counter)
	}

	return linkElement.ElementID
}

// extractMetadata extracts metadata from HTML attributes
func (p *HTMLParser) extractMetadata(node *html.Node) map[string]interface{} {
	metadata := make(map[string]interface{})

	for _, attr := range node.Attr {
		// Store important attributes
		switch attr.Key {
		case "id", "class", "colspan", "rowspan":
			metadata[attr.Key] = attr.Val
		default:
			if strings.HasPrefix(attr.Key, "data-") {
				metadata[attr.Key] = attr.Val
			}
		}
	}

	// Extract header level for h1-h6 tags
	tagName := strings.ToLower(node.Data)
	if len(tagName) == 2 && tagName[0] == 'h' && tagName[1] >= '1' && tagName[1] <= '6' {
		level := int(tagName[1] - '0')
		metadata["level"] = level
		metadata["tag"] = tagName
	}

	// Set default values for table cells if not present
	if node.Data == "td" || node.Data == "th" {
		if _, exists := metadata["colspan"]; !exists {
			metadata["colspan"] = "1"
		}
		if _, exists := metadata["rowspan"]; !exists {
			metadata["rowspan"] = "1"
		}
	}

	return metadata
}

// createContentLocation creates a content location object
func (p *HTMLParser) createContentLocation(source string, elementType string, selector string) map[string]interface{} {
	return map[string]interface{}{
		"source":   source,
		"type":     elementType,
		"selector": selector,
	}
}

// createContentLocationForNode creates a content location for a specific node
func (p *HTMLParser) createContentLocationForNode(source string, elementType HTMLElementType, node *html.Node) map[string]interface{} {
	selector := p.generateCSSSelector(node)
	return p.createContentLocation(source, string(elementType), selector)
}

// generateCSSSelector generates a CSS selector for an HTML node
func (p *HTMLParser) generateCSSSelector(node *html.Node) string {
	if node == nil || node.Type != html.ElementNode {
		return ""
	}

	var parts []string
	current := node

	for current != nil && current.Type == html.ElementNode {
		part := current.Data

		// Add ID if present
		for _, attr := range current.Attr {
			if attr.Key == "id" && attr.Val != "" {
				part = fmt.Sprintf("%s#%s", part, attr.Val)
				parts = append([]string{part}, parts...)
				// ID is unique, so we can stop here
				return strings.Join(parts, " > ")
			}
		}

		// Add classes if present
		var classes []string
		for _, attr := range current.Attr {
			if attr.Key == "class" && attr.Val != "" {
				classNames := strings.Fields(attr.Val)
				for _, className := range classNames {
					if className != "" {
						classes = append(classes, "."+className)
					}
				}
			}
		}
		if len(classes) > 0 {
			part = part + strings.Join(classes, "")
		}

		// Add nth-of-type if needed
		if current.Parent != nil {
			position := p.getNthOfType(current)
			if position > 1 {
				part = fmt.Sprintf("%s:nth-of-type(%d)", part, position)
			}
		}

		parts = append([]string{part}, parts...)
		current = current.Parent
	}

	return strings.Join(parts, " > ")
}

// getNthOfType calculates the nth-of-type position for an element
func (p *HTMLParser) getNthOfType(node *html.Node) int {
	if node.Parent == nil {
		return 1
	}

	position := 1
	for sibling := node.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		if sibling == node {
			break
		}
		if sibling.Type == html.ElementNode && sibling.Data == node.Data {
			position++
		}
	}

	return position
}

// Helper functions

// generateID generates a unique ID with the given prefix
func (p *HTMLParser) generateID(prefix string) string {
	return generateID(prefix)
}

// generateHash generates an MD5 hash of the content
func (p *HTMLParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content to the maximum preview length
func (p *HTMLParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// ToJSON converts the parse response to JSON
func (r *ParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates an HTMLParseRequest from JSON
func (r *HTMLParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// convertToParseResult converts ParseResponse to universal ParseResult format
func (p *HTMLParser) convertToParseResult(response *ParseResponse) *ParseResult {
	result := &ParseResult{
		Document: Document{
			ID:      response.Document["doc_id"].(string),
			DocType: response.Document["doc_type"].(string),
		},
		Elements: make([]Element, 0, len(response.Elements)),
	}

	// Convert metadata if present
	if meta, ok := response.Document["metadata"]; ok {
		result.Document.Metadata = meta.(map[string]interface{})
	}

	// Convert elements
	for i, htmlElem := range response.Elements {
		element := Element{
			ElementID:       htmlElem.ElementID,
			ElementType:     string(htmlElem.ElementType),
			Content:         "", // Leave empty - rely on ContentLocation resolution for HTML
			ContentPreview:  htmlElem.ContentPreview,
			ParentID:        htmlElem.ParentID,
			Position:        i,
			Depth:           calculateDepth(htmlElem.ElementType),
			ContentLocation: htmlElem.ContentLocation,
			Metadata:        htmlElem.Metadata,
		}

		// UDML Phase 1: Populate promoted fields

		// TagName - store the HTML tag name for all HTML elements
		tagName := string(htmlElem.ElementType)
		element.TagName = &tagName

		// SectionLevel - for heading elements (h1-h6)
		if htmlElem.ElementType == HTMLElementTypeHeading {
			// Extract level from element type or ID
			level := extractHeaderLevel(htmlElem.ElementID, htmlElem.Metadata)
			if level > 0 {
				element.SectionLevel = &level
			}
		}

		// RowIndex and ColumnIndex - for table cells
		if htmlElem.ElementType == HTMLElementTypeTableCell || htmlElem.ElementType == HTMLElementTypeTableHeader {
			if row, ok := htmlElem.Metadata["row"].(int); ok {
				element.RowIndex = &row
			}
			if col, ok := htmlElem.Metadata["col"].(int); ok {
				element.ColumnIndex = &col
			}
		}

		// TemporalType - from temporal metadata
		if temporalType, ok := htmlElem.Metadata["temporal_type"].(string); ok && temporalType != "" {
			element.TemporalType = &temporalType
		}

		// ElementCategory
		element.ElementCategory = GetElementCategory(element.ElementType)

		result.Elements = append(result.Elements, element)
	}

	return result
}

// calculateDepth calculates element depth based on HTML element type
func calculateDepth(elementType HTMLElementType) int {
	switch elementType {
	case HTMLElementTypeRoot:
		return 0
	case HTMLElementTypeBody:
		return 1
	case HTMLElementTypeHeader, HTMLElementTypeHeading, HTMLElementTypeParagraph, HTMLElementTypeList, HTMLElementTypeTable,
		 HTMLElementTypeDiv, HTMLElementTypeArticle, HTMLElementTypeSection:
		return 2
	case HTMLElementTypeListItem, HTMLElementTypeTableRow:
		return 3
	case HTMLElementTypeTableCell, HTMLElementTypeTableHeader, HTMLElementTypeSpan, HTMLElementTypeText:
		return 4
	default:
		return 2 // Default depth for unknown elements
	}
}

// extractHeaderLevel extracts the heading level (1-6) from header element metadata
func extractHeaderLevel(elementID string, metadata map[string]interface{}) int {
	// Try to get from metadata first
	if level, ok := metadata["level"].(int); ok && level >= 1 && level <= 6 {
		return level
	}

	// Extract from element ID (e.g., "h1_123" or "header_h2_456")
	idLower := strings.ToLower(elementID)
	for level := 1; level <= 6; level++ {
		tag := fmt.Sprintf("h%d", level)
		if strings.Contains(idLower, tag) {
			return level
		}
	}

	return 0 // Unknown level
}

// SupportsLocation checks if this parser can resolve the given content location
func (p *HTMLParser) SupportsLocation(contentLocation map[string]interface{}) bool {
	if contentLocation == nil {
		return false
	}

	// Check if source field exists
	source, ok := contentLocation["source"].(string)
	if !ok || source == "" {
		return false
	}

	// Support both URLs and local files
	// For URLs, check if it looks like HTML (has CSS selector)
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// CSS selectors are HTML-specific, so presence of selector indicates HTML
		_, hasSelector := contentLocation["selector"]
		return hasSelector
	}

	// For local files, check if file exists and is HTML
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return false
	}

	// Check if it's an HTML file
	ext := strings.ToLower(filepath.Ext(source))
	return ext == ".html" || ext == ".htm"
}

// resolveElementSelection is a helper that finds an element using content_location
// Returns the goquery selection for the element
func (p *HTMLParser) resolveElementSelection(contentLocation map[string]interface{}) (*goquery.Selection, error) {
	if contentLocation == nil {
		return nil, fmt.Errorf("content_location is nil")
	}

	// Get source file path or URL
	source, ok := contentLocation["source"].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("missing or invalid 'source' field in content_location")
	}

	// Get CSS selector - must be specific
	selector, ok := contentLocation["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("missing or invalid 'selector' field in content_location")
	}

	var htmlContent string

	// Check if source is a URL (web content)
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Try memory cache first (fastest)
		p.cacheMu.RLock()
		cached, found := p.documentCache[source]
		p.cacheMu.RUnlock()

		if found {
			htmlContent = cached
		} else {
			// Try disk cache second (fast, persistent)
			if p.diskCache != nil {
				diskCached, diskFound := p.diskCache.Get(source)
				if diskFound {
					htmlContent = diskCached
					// Populate memory cache for next time
					p.cacheMu.Lock()
					p.documentCache[source] = htmlContent
					p.cacheMu.Unlock()
					goto parseHTML
				}
			}

			// Not in any cache - fetch from URL
			req, err := http.NewRequest("GET", source, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request for %s: %w", source, err)
			}

			// Add configured headers (including User-Agent)
			for k, v := range p.Headers {
				req.Header.Set(k, v)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch HTML from URL %s: %w", source, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("HTTP %d when fetching URL: %s", resp.StatusCode, source)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body from %s: %w", source, err)
			}

			htmlContent = string(bodyBytes)

			// Cache in both memory and disk
			p.cacheMu.Lock()
			p.documentCache[source] = htmlContent
			p.cacheMu.Unlock()

			if p.diskCache != nil {
				// Async disk write to avoid blocking
				go func(url, content string) {
					if err := p.diskCache.Put(url, content); err != nil {
						// Log warning but don't fail the request
						fmt.Fprintf(os.Stderr, "Warning: failed to write to disk cache: %v\n", err)
					}
				}(source, htmlContent)
			}
		}
	parseHTML:
	} else {
		// Local file - read from disk
		fileContent, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("failed to read source file %s: %w", source, err)
		}
		htmlContent = string(fileContent)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find element using CSS selector
	selection := doc.Find(selector)

	// Ensure selector resolves to exactly one element
	if selection.Length() == 0 {
		return nil, fmt.Errorf("selector '%s' matched no elements", selector)
	}
	if selection.Length() > 1 {
		return nil, fmt.Errorf("selector '%s' matched %d elements (must match exactly 1)", selector, selection.Length())
	}

	return selection, nil
}

// ResolveElementContent extracts raw HTML content for a specific element using content_location
func (p *HTMLParser) ResolveElementContent(contentLocation map[string]interface{}, sourceContent string) (string, error) {
	selection, err := p.resolveElementSelection(contentLocation)
	if err != nil {
		return "", err
	}

	// Get HTML content of the element
	html, err := selection.Html()
	if err != nil {
		return "", fmt.Errorf("failed to extract HTML: %w", err)
	}

	return html, nil
}

// ResolveElementText extracts plain text for a specific element using content_location
func (p *HTMLParser) ResolveElementText(contentLocation map[string]interface{}, sourceContent string) (string, error) {
	selection, err := p.resolveElementSelection(contentLocation)
	if err != nil {
		return "", err
	}

	// Extract text content
	text := selection.Text()

	// Clean up whitespace
	text = strings.TrimSpace(text)

	return text, nil
}