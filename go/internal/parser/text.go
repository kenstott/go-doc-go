package parser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// TextElementType represents the type of text element
type TextElementType string

const (
	TextElementTypeRoot      TextElementType = "root"
	TextElementTypeParagraph TextElementType = "paragraph"
	TextElementTypeLine      TextElementType = "line"
	TextElementTypeRange     TextElementType = "range"
	TextElementTypeSubstring TextElementType = "substring"
)

// String returns the string representation of TextElementType
func (t TextElementType) String() string {
	return string(t)
}

// TextElement represents a parsed text element
type TextElement struct {
	ElementID       string                 `json:"element_id"`
	DocumentID      string                 `json:"doc_id"`
	ElementType     TextElementType        `json:"element_type"`
	Text            string                 `json:"text,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ParentID        string                 `json:"parent_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// TextRelationship represents a relationship between text elements
type TextRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// TextLink represents an extracted link
type TextLink struct {
	LinkID     string `json:"link_id"`
	LinkType   string `json:"link_type"`
	LinkTarget string `json:"link_target"`
	ElementID  string `json:"element_id"`
}

// TextParseRequest represents a request to parse text content
type TextParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// TextParseResponse represents the response from parsing text content
type TextParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []TextElement          `json:"elements"`
	Relationships []TextRelationship     `json:"relationships"`
	Links         []TextLink             `json:"links"`
}

// ToJSON converts the response to JSON string
func (r *TextParseResponse) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response to JSON: %w", err)
	}
	return string(data), nil
}

// TextParser handles parsing of text documents
type TextParser struct {
	MaxContentPreview     int
	ParagraphSeparator    string
	MinParagraphLength    int
	MaxElements           int
	ExtractDates          bool
	ExtractNumbers        bool
	EnableLinkExtraction  bool
	StripWhitespace       bool
	ElementIDCounter      int
	RelationshipIDCounter int
	LinkIDCounter         int
}

// NewTextParser creates a new text parser with default settings
func NewTextParser() *TextParser {
	return &TextParser{
		MaxContentPreview:     100,
		ParagraphSeparator:    "\n\n",
		MinParagraphLength:    5,
		MaxElements:           1000,
		ExtractDates:          true,
		ExtractNumbers:        true,
		EnableLinkExtraction:  true,
		StripWhitespace:       true,
		ElementIDCounter:      0,
		RelationshipIDCounter: 0,
		LinkIDCounter:         0,
	}
}

// Parse parses text content and returns structured elements
func (p *TextParser) Parse(request TextParseRequest) (*TextParseResponse, error) {
	if request.ID == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	content := request.Content
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Reset counters for each parse
	p.ElementIDCounter = 0
	p.RelationshipIDCounter = 0
	p.LinkIDCounter = 0

	response := &TextParseResponse{
		Document:      make(map[string]interface{}),
		Elements:      []TextElement{},
		Relationships: []TextRelationship{},
		Links:         []TextLink{},
	}

	// Create document metadata
	p.createDocumentMetadata(response, request)

	// Create root element
	rootElement := p.createRootElement(request.ID, content)
	response.Elements = append(response.Elements, rootElement)

	// Split content into paragraphs and create elements
	paragraphs := p.splitIntoParagraphs(content)
	for i, paragraph := range paragraphs {
		if len(paragraph) < p.MinParagraphLength {
			continue
		}

		// Create paragraph element
		paragraphElement := p.createParagraphElement(request.ID, paragraph, i, rootElement.ElementID)
		response.Elements = append(response.Elements, paragraphElement)

		// Create relationship
		relationship := p.createRelationship(rootElement.ElementID, paragraphElement.ElementID, "contains")
		response.Relationships = append(response.Relationships, relationship)

		// Extract links from paragraph
		if p.EnableLinkExtraction {
			links := p.extractLinks(paragraph, paragraphElement.ElementID)
			response.Links = append(response.Links, links...)
		}

		// Stop if we've reached max elements
		if len(response.Elements) >= p.MaxElements {
			break
		}
	}

	return response, nil
}

// createDocumentMetadata creates document-level metadata
func (p *TextParser) createDocumentMetadata(response *TextParseResponse, request TextParseRequest) {
	content := request.Content

	// Calculate basic statistics
	charCount := utf8.RuneCountInString(content)
	wordCount := len(strings.Fields(content))
	lineCount := strings.Count(content, "\n") + 1

	// Calculate paragraph count
	paragraphs := p.splitIntoParagraphs(content)
	paragraphCount := 0
	for _, paragraph := range paragraphs {
		if len(paragraph) >= p.MinParagraphLength {
			paragraphCount++
		}
	}

	response.Document = map[string]interface{}{
		"doc_id":   request.ID,
		"doc_type": "text",
		"metadata": map[string]interface{}{
			"character_count":  charCount,
			"word_count":       wordCount,
			"line_count":       lineCount,
			"paragraph_count":  paragraphCount,
			"source":           request.Metadata["source"],
			"filename":         request.Metadata["filename"],
			"content_type":     "text/plain",
			"parsing_method":   "go_text_parser",
			"paragraph_separator": p.ParagraphSeparator,
		},
	}
}

// createRootElement creates the root document element
func (p *TextParser) createRootElement(docID, content string) TextElement {
	elementID := p.generateElementID("txt_root")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	return TextElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     TextElementTypeRoot,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "full_document"},
		ContentHash:     contentHash,
		Metadata: map[string]interface{}{
			"element_type": "root",
			"length":       utf8.RuneCountInString(content),
		},
	}
}

// createParagraphElement creates a paragraph element
func (p *TextParser) createParagraphElement(docID, content string, index int, parentID string) TextElement {
	elementID := p.generateElementID("txt_para")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	// Extract metadata
	metadata := map[string]interface{}{
		"element_type":     "paragraph",
		"paragraph_index":  index,
		"length":           utf8.RuneCountInString(content),
		"word_count":       len(strings.Fields(content)),
		"line_count":       strings.Count(content, "\n") + 1,
	}

	// Add date extraction if enabled
	if p.ExtractDates {
		dates := p.extractDates(content)
		if len(dates) > 0 {
			metadata["dates"] = dates
		}
	}

	// Add number extraction if enabled
	if p.ExtractNumbers {
		numbers := p.extractNumbers(content)
		if len(numbers) > 0 {
			metadata["numbers"] = numbers
		}
	}

	return TextElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     TextElementTypeParagraph,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "paragraph", "index": index},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createRelationship creates a relationship between elements
func (p *TextParser) createRelationship(sourceID, targetID, relType string) TextRelationship {
	relationshipID := p.generateRelationshipID()

	return TextRelationship{
		RelationshipID:   relationshipID,
		SourceElementID:  sourceID,
		TargetElementID:  targetID,
		RelationshipType: relType,
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"relationship_type": relType},
	}
}

// splitIntoParagraphs splits content into paragraphs
func (p *TextParser) splitIntoParagraphs(content string) []string {
	// Normalize line endings
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Split by paragraph separator
	paragraphs := strings.Split(normalized, p.ParagraphSeparator)

	// Clean up paragraphs
	var result []string
	for _, paragraph := range paragraphs {
		if p.StripWhitespace {
			paragraph = strings.TrimSpace(paragraph)
		}
		if paragraph != "" {
			result = append(result, paragraph)
		}
	}

	return result
}

// extractLinks extracts links from content
func (p *TextParser) extractLinks(content, elementID string) []TextLink {
	var links []TextLink

	// URL patterns
	urlPattern := regexp.MustCompile(`https?://[^\s\)\]\}]+`)
	urls := urlPattern.FindAllString(content, -1)
	for _, url := range urls {
		linkID := p.generateLinkID()
		links = append(links, TextLink{
			LinkID:     linkID,
			LinkType:   "url",
			LinkTarget: url,
			ElementID:  elementID,
		})
	}

	// Email patterns
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailPattern.FindAllString(content, -1)
	for _, email := range emails {
		linkID := p.generateLinkID()
		links = append(links, TextLink{
			LinkID:     linkID,
			LinkType:   "email",
			LinkTarget: "mailto:" + email,
			ElementID:  elementID,
		})
	}

	// File path patterns (Unix and Windows)
	filePattern := regexp.MustCompile(`(?:\/[^\s\/]+)+\.[a-zA-Z0-9]+|[A-Za-z]:\\(?:[^\s\\]+\\)*[^\s\\]+\.[a-zA-Z0-9]+`)
	filePaths := filePattern.FindAllString(content, -1)
	for _, filePath := range filePaths {
		linkID := p.generateLinkID()
		links = append(links, TextLink{
			LinkID:     linkID,
			LinkType:   "file",
			LinkTarget: "file://" + filePath,
			ElementID:  elementID,
		})
	}

	return links
}

// extractDates extracts date patterns from content
func (p *TextParser) extractDates(content string) []string {
	var dates []string

	// Common date patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),                    // YYYY-MM-DD
		regexp.MustCompile(`\b\d{2}/\d{2}/\d{4}\b`),                    // MM/DD/YYYY
		regexp.MustCompile(`\b\d{2}-\d{2}-\d{4}\b`),                    // MM-DD-YYYY
		regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`),               // M/D/YYYY
		regexp.MustCompile(`\b[A-Za-z]{3,9}\s+\d{1,2},\s+\d{4}\b`),    // Month DD, YYYY
		regexp.MustCompile(`\b\d{1,2}\s+[A-Za-z]{3,9}\s+\d{4}\b`),     // DD Month YYYY
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllString(content, -1)
		dates = append(dates, matches...)
	}

	return dates
}

// extractNumbers extracts numeric patterns from content
func (p *TextParser) extractNumbers(content string) []interface{} {
	var numbers []interface{}

	// Integer pattern
	intPattern := regexp.MustCompile(`\b\d+\b`)
	ints := intPattern.FindAllString(content, -1)
	for _, intStr := range ints {
		if num, err := strconv.Atoi(intStr); err == nil && num > 0 {
			numbers = append(numbers, num)
		}
	}

	// Float pattern
	floatPattern := regexp.MustCompile(`\b\d+\.\d+\b`)
	floats := floatPattern.FindAllString(content, -1)
	for _, floatStr := range floats {
		if num, err := strconv.ParseFloat(floatStr, 64); err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

// generateElementID generates a unique element ID
func (p *TextParser) generateElementID(prefix string) string {
	p.ElementIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s_%d_%d", prefix, timestamp, p.ElementIDCounter)
}

// generateRelationshipID generates a unique relationship ID
func (p *TextParser) generateRelationshipID() string {
	p.RelationshipIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("txt_rel_%d_%d", timestamp, p.RelationshipIDCounter)
}

// generateLinkID generates a unique link ID
func (p *TextParser) generateLinkID() string {
	p.LinkIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("txt_link_%d_%d", timestamp, p.LinkIDCounter)
}

// calculateContentHash calculates MD5 hash of content
func (p *TextParser) calculateContentHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content for preview
func (p *TextParser) truncateContent(content string) string {
	// Remove extra whitespace and normalize
	content = strings.TrimSpace(content)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")

	if utf8.RuneCountInString(content) <= p.MaxContentPreview {
		return content
	}

	// Truncate by runes, not bytes
	runes := []rune(content)
	truncated := string(runes[:p.MaxContentPreview-3])

	// Try to break at word boundary
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > p.MaxContentPreview/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}