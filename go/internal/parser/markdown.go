package parser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// MarkdownElementType represents the type of markdown element
type MarkdownElementType string

const (
	MarkdownElementTypeRoot       MarkdownElementType = "root"
	MarkdownElementTypeHeader     MarkdownElementType = "header"
	MarkdownElementTypeParagraph  MarkdownElementType = "paragraph"
	MarkdownElementTypeCodeBlock  MarkdownElementType = "code_block"
	MarkdownElementTypeList       MarkdownElementType = "list"
	MarkdownElementTypeListItem   MarkdownElementType = "list_item"
	MarkdownElementTypeBlockquote MarkdownElementType = "blockquote"
	MarkdownElementTypeTable      MarkdownElementType = "table"
	MarkdownElementTypeTableRow   MarkdownElementType = "table_row"
	MarkdownElementTypeTableCell  MarkdownElementType = "table_cell"
	MarkdownElementTypeFrontMatter MarkdownElementType = "front_matter"
)

// String returns the string representation of MarkdownElementType
func (t MarkdownElementType) String() string {
	return string(t)
}

// MarkdownElement represents a parsed markdown element
type MarkdownElement struct {
	ElementID       string                 `json:"element_id"`
	DocumentID      string                 `json:"doc_id"`
	ElementType     MarkdownElementType    `json:"element_type"`
	Text            string                 `json:"text,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ParentID        string                 `json:"parent_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// MarkdownRelationship represents a relationship between markdown elements
type MarkdownRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// MarkdownLink represents an extracted link
type MarkdownLink struct {
	LinkID     string `json:"link_id"`
	LinkType   string `json:"link_type"`
	LinkTarget string `json:"link_target"`
	ElementID  string `json:"element_id"`
	LinkText   string `json:"link_text,omitempty"`
}

// MarkdownParseRequest represents a request to parse markdown content
type MarkdownParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// MarkdownParseResponse represents the response from parsing markdown content
type MarkdownParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []MarkdownElement      `json:"elements"`
	Relationships []MarkdownRelationship `json:"relationships"`
	Links         []MarkdownLink         `json:"links"`
}

// ToJSON converts the response to JSON string
func (r *MarkdownParseResponse) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response to JSON: %w", err)
	}
	return string(data), nil
}

// ToParseResult converts MarkdownParseResponse to the standard ParseResult format
func (r *MarkdownParseResponse) ToParseResult() *ParseResult {
	// Convert document metadata
	var doc Document
	if docID, ok := r.Document["doc_id"].(string); ok {
		doc.ID = docID
	}
	if title, ok := r.Document["title"].(string); ok {
		doc.Title = title
	}
	if contentType, ok := r.Document["content_type"].(string); ok {
		doc.DocType = contentType
	}
	doc.Metadata = r.Document

	// Convert elements
	elements := make([]Element, len(r.Elements))
	for i, mdElem := range r.Elements {
		elements[i] = Element{
			ElementID:       mdElem.ElementID,
			ElementType:     string(mdElem.ElementType),
			Content:         mdElem.Text,
			ContentPreview:  mdElem.ContentPreview,
			ParentID:        mdElem.ParentID,
			Metadata:        mdElem.Metadata,
			ContentLocation: mdElem.ContentLocation,
		}
	}

	// Convert relationships
	relationships := make([]Relationship, len(r.Relationships))
	for i, mdRel := range r.Relationships {
		relationships[i] = Relationship{
			SourceElementID:  mdRel.SourceElementID,
			TargetElementID:  mdRel.TargetElementID,
			RelationshipType: mdRel.RelationshipType,
			Metadata:         mdRel.Metadata,
		}
	}

	// Convert links
	links := make([]Link, len(r.Links))
	for i, mdLink := range r.Links {
		links[i] = Link{
			SourceElementID: mdLink.ElementID,
			LinkTarget:      mdLink.LinkTarget,
			LinkText:        mdLink.LinkText,
			LinkType:        mdLink.LinkType,
		}
	}

	return &ParseResult{
		Document:      doc,
		Elements:      elements,
		Relationships: relationships,
		Links:         links,
	}
}

// MarkdownParser handles parsing of markdown documents
type MarkdownParser struct {
	MaxContentPreview     int
	ExtractFrontMatter    bool
	ParagraphThreshold    int
	MaxElements           int
	ExtractDates          bool
	ExtractNumbers        bool
	EnableLinkExtraction  bool
	StripWhitespace       bool
	ElementIDCounter      int
	RelationshipIDCounter int
	LinkIDCounter         int
}

// NewMarkdownParser creates a new markdown parser with default settings
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		MaxContentPreview:     100,
		ExtractFrontMatter:    true,
		ParagraphThreshold:    1,
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

// Parse parses markdown content and returns structured elements
func (p *MarkdownParser) Parse(request MarkdownParseRequest) (*MarkdownParseResponse, error) {
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

	response := &MarkdownParseResponse{
		Document:      make(map[string]interface{}),
		Elements:      []MarkdownElement{},
		Relationships: []MarkdownRelationship{},
		Links:         []MarkdownLink{},
	}

	// Extract front matter if enabled
	var frontMatter map[string]interface{}
	if p.ExtractFrontMatter {
		content, frontMatter = p.extractFrontMatter(content)
	}

	// Create document metadata
	p.createDocumentMetadata(response, request, frontMatter)

	// Create root element
	rootElement := p.createRootElement(request.ID, content)
	response.Elements = append(response.Elements, rootElement)

	// Process front matter element if present
	if frontMatter != nil {
		frontMatterElement := p.createFrontMatterElement(request.ID, frontMatter, rootElement.ElementID)
		response.Elements = append(response.Elements, frontMatterElement)

		// Create relationship
		relationship := p.createRelationship(rootElement.ElementID, frontMatterElement.ElementID, "contains")
		response.Relationships = append(response.Relationships, relationship)
	}

	// Parse markdown content into elements
	lines := strings.Split(content, "\n")
	p.parseMarkdownContent(lines, request.ID, rootElement.ElementID, response)

	return response, nil
}

// extractFrontMatter extracts YAML front matter from content
func (p *MarkdownParser) extractFrontMatter(content string) (string, map[string]interface{}) {
	// Check for YAML front matter (--- at start and end)
	frontMatterPattern := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)
	matches := frontMatterPattern.FindStringSubmatch(content)

	if len(matches) != 3 {
		return content, nil
	}

	yamlContent := matches[1]
	remainingContent := matches[2]

	// Parse YAML
	var frontMatter map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &frontMatter)
	if err != nil {
		// If YAML parsing fails, return original content
		return content, nil
	}

	return remainingContent, frontMatter
}

// createDocumentMetadata creates document-level metadata
func (p *MarkdownParser) createDocumentMetadata(response *MarkdownParseResponse, request MarkdownParseRequest, frontMatter map[string]interface{}) {
	content := request.Content

	// Calculate basic statistics
	charCount := utf8.RuneCountInString(content)
	wordCount := len(strings.Fields(content))
	lineCount := strings.Count(content, "\n") + 1

	// Count different element types
	headerCount := len(regexp.MustCompile(`^#{1,6}\s+`).FindAllString(content, -1))
	codeBlockCount := len(regexp.MustCompile("```").FindAllString(content, -1)) / 2
	listItemCount := len(regexp.MustCompile(`(?m)^[\s]*[-\*\+]\s+`).FindAllString(content, -1))

	metadata := map[string]interface{}{
		"character_count":  charCount,
		"word_count":       wordCount,
		"line_count":       lineCount,
		"header_count":     headerCount,
		"code_block_count": codeBlockCount,
		"list_item_count":  listItemCount,
		"source":           request.Metadata["source"],
		"filename":         request.Metadata["filename"],
		"content_type":     "text/markdown",
		"parsing_method":   "go_markdown_parser",
		"has_front_matter": frontMatter != nil,
	}

	// Add front matter metadata if present
	if frontMatter != nil {
		metadata["front_matter"] = frontMatter
	}

	response.Document = map[string]interface{}{
		"doc_id":   request.ID,
		"doc_type": "markdown",
		"metadata": metadata,
	}
}

// createRootElement creates the root document element
func (p *MarkdownParser) createRootElement(docID, content string) MarkdownElement {
	elementID := p.generateElementID("md_root")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeRoot,
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

// createFrontMatterElement creates a front matter element
func (p *MarkdownParser) createFrontMatterElement(docID string, frontMatter map[string]interface{}, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_front")

	// Convert front matter back to YAML for display
	yamlBytes, _ := yaml.Marshal(frontMatter)
	yamlContent := string(yamlBytes)

	contentHash := p.calculateContentHash(yamlContent)
	preview := p.truncateContent(yamlContent)

	metadata := map[string]interface{}{
		"element_type": "front_matter",
	}

	// Add front matter fields to metadata
	for key, value := range frontMatter {
		metadata[key] = value
	}

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeFrontMatter,
		Text:            yamlContent,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "front_matter"},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// parseMarkdownContent parses markdown content into elements
func (p *MarkdownParser) parseMarkdownContent(lines []string, docID, parentID string, response *MarkdownParseResponse) {
	i := 0
	var previousElementID string // Track previous element for sibling relationships

	// Track section hierarchy for proper parent-child relationships (like Python)
	currentParent := parentID
	type section struct {
		id    string
		level int
	}
	sectionStack := []section{{id: parentID, level: 0}}

	for i < len(lines) && len(response.Elements) < p.MaxElements {
		line := lines[i]

		if p.StripWhitespace {
			line = strings.TrimSpace(line)
		}

		// Skip empty lines
		if line == "" {
			i++
			continue
		}

		// Check for headers
		if match := regexp.MustCompile(`^(#{1,6})\s+(.+)$`).FindStringSubmatch(line); match != nil {
			level := len(match[1])

			// Pop stack until we find a header with lower level (like Python does)
			for len(sectionStack) > 1 && sectionStack[len(sectionStack)-1].level >= level {
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			// Current parent is the top of the stack
			currentParent = sectionStack[len(sectionStack)-1].id

			element := p.createHeaderElement(docID, match[2], level, i, currentParent)
			response.Elements = append(response.Elements, element)

			// Create parent-child relationship
			relationship := p.createRelationship(currentParent, element.ElementID, "contains")
			response.Relationships = append(response.Relationships, relationship)

			// Push this header onto the stack and make it the new current parent
			sectionStack = append(sectionStack, section{id: element.ElementID, level: level})
			currentParent = element.ElementID

			// Create sibling relationships
			if previousElementID != "" {
				prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
				response.Relationships = append(response.Relationships, prevSibRel)

				nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
				response.Relationships = append(response.Relationships, nextSibRel)
			}
			previousElementID = element.ElementID

			// Extract links from header
			if p.EnableLinkExtraction {
				links := p.extractLinks(match[2], element.ElementID)
				response.Links = append(response.Links, links...)
			}

			i++
			continue
		}

		// Check for code blocks
		if strings.HasPrefix(line, "```") {
			codeContent, language, newIndex := p.parseCodeBlock(lines, i)
			if codeContent != "" {
				element := p.createCodeBlockElement(docID, codeContent, language, i, currentParent)
				response.Elements = append(response.Elements, element)

				// Create parent-child relationship
				relationship := p.createRelationship(currentParent, element.ElementID, "contains")
				response.Relationships = append(response.Relationships, relationship)

				// Create sibling relationships
				if previousElementID != "" {
					prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
					response.Relationships = append(response.Relationships, prevSibRel)

					nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
					response.Relationships = append(response.Relationships, nextSibRel)
				}
				previousElementID = element.ElementID

				i = newIndex
				continue
			}
		}

		// Check for blockquotes
		if strings.HasPrefix(line, ">") {
			quoteContent, newIndex := p.parseBlockquote(lines, i)
			if quoteContent != "" {
				element := p.createBlockquoteElement(docID, quoteContent, i, currentParent)
				response.Elements = append(response.Elements, element)

				// Create parent-child relationship
				relationship := p.createRelationship(currentParent, element.ElementID, "contains")
				response.Relationships = append(response.Relationships, relationship)

				// Create sibling relationships
				if previousElementID != "" {
					prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
					response.Relationships = append(response.Relationships, prevSibRel)

					nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
					response.Relationships = append(response.Relationships, nextSibRel)
				}
				previousElementID = element.ElementID

				// Extract links from blockquote
				if p.EnableLinkExtraction {
					links := p.extractLinks(quoteContent, element.ElementID)
					response.Links = append(response.Links, links...)
				}
			}
			i = newIndex
			continue
		}

		// Check for lists
		if match := regexp.MustCompile(`^(\s*)([-\*\+]|\d+\.)\s+(.+)$`).FindStringSubmatch(line); match != nil {
			listContent, newIndex := p.parseList(lines, i)
			if listContent != "" {
				element := p.createListElement(docID, listContent, i, currentParent)
				response.Elements = append(response.Elements, element)

				// Create parent-child relationship
				relationship := p.createRelationship(currentParent, element.ElementID, "contains")
				response.Relationships = append(response.Relationships, relationship)

				// Create sibling relationships
				if previousElementID != "" {
					prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
					response.Relationships = append(response.Relationships, prevSibRel)

					nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
					response.Relationships = append(response.Relationships, nextSibRel)
				}
				previousElementID = element.ElementID

				// Extract links from list
				if p.EnableLinkExtraction {
					links := p.extractLinks(listContent, element.ElementID)
					response.Links = append(response.Links, links...)
				}

				i = newIndex
				continue
			}
		}

		// Check for tables
		if strings.Contains(line, "|") && i+1 < len(lines) && strings.Contains(lines[i+1], "|") {
			tableContent, newIndex := p.parseTable(lines, i)
			if tableContent != "" {
				element := p.createTableElement(docID, tableContent, i, currentParent)
				response.Elements = append(response.Elements, element)

				// Create relationship
				relationship := p.createRelationship(currentParent, element.ElementID, "contains")
				response.Relationships = append(response.Relationships, relationship)

				// Create sibling relationships
				if previousElementID != "" {
					prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
					response.Relationships = append(response.Relationships, prevSibRel)

					nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
					response.Relationships = append(response.Relationships, nextSibRel)
				}
				previousElementID = element.ElementID

				// Extract links from table
				if p.EnableLinkExtraction {
					links := p.extractLinks(tableContent, element.ElementID)
					response.Links = append(response.Links, links...)
				}

				i = newIndex
				continue
			}
		}

		// Default: treat as paragraph
		paragraphContent, newIndex := p.parseParagraph(lines, i)
		if paragraphContent != "" && len(paragraphContent) >= p.ParagraphThreshold {
			element := p.createParagraphElement(docID, paragraphContent, i, currentParent)
			response.Elements = append(response.Elements, element)

			// Create relationship
			relationship := p.createRelationship(currentParent, element.ElementID, "contains")
			response.Relationships = append(response.Relationships, relationship)

			// Create sibling relationships
			if previousElementID != "" {
				prevSibRel := p.createRelationship(element.ElementID, previousElementID, "previous_sibling")
				response.Relationships = append(response.Relationships, prevSibRel)

				nextSibRel := p.createRelationship(previousElementID, element.ElementID, "next_sibling")
				response.Relationships = append(response.Relationships, nextSibRel)
			}
			previousElementID = element.ElementID

			// Extract links from paragraph
			if p.EnableLinkExtraction {
				links := p.extractLinks(paragraphContent, element.ElementID)
				response.Links = append(response.Links, links...)
			}
		}

		i = newIndex
	}
}

// createHeaderElement creates a header element
func (p *MarkdownParser) createHeaderElement(docID, text string, level, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_header")
	contentHash := p.calculateContentHash(text)
	preview := p.truncateContent(text)

	metadata := map[string]interface{}{
		"element_type": "header",
		"level":        level,
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(text),
	}

	// Add date extraction if enabled
	if p.ExtractDates {
		dates := p.extractDates(text)
		if len(dates) > 0 {
			metadata["dates"] = dates
		}
	}

	// Add number extraction if enabled
	if p.ExtractNumbers {
		numbers := p.extractNumbers(text)
		if len(numbers) > 0 {
			metadata["numbers"] = numbers
		}
	}

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeHeader,
		Text:            text,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "header", "level": level, "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createParagraphElement creates a paragraph element
func (p *MarkdownParser) createParagraphElement(docID, text string, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_para")
	contentHash := p.calculateContentHash(text)
	preview := p.truncateContent(text)

	metadata := map[string]interface{}{
		"element_type": "paragraph",
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(text),
		"word_count":   len(strings.Fields(text)),
	}

	// Add date extraction if enabled
	if p.ExtractDates {
		dates := p.extractDates(text)
		if len(dates) > 0 {
			metadata["dates"] = dates
		}
	}

	// Add number extraction if enabled
	if p.ExtractNumbers {
		numbers := p.extractNumbers(text)
		if len(numbers) > 0 {
			metadata["numbers"] = numbers
		}
	}

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeParagraph,
		Text:            text,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "paragraph", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createCodeBlockElement creates a code block element
func (p *MarkdownParser) createCodeBlockElement(docID, content, language string, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_code")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	metadata := map[string]interface{}{
		"element_type": "code_block",
		"language":     language,
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(content),
		"line_count":   strings.Count(content, "\n") + 1,
	}

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeCodeBlock,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "code_block", "language": language, "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createListElement creates a list element
func (p *MarkdownParser) createListElement(docID, content string, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_list")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	// Count list items
	itemPattern := regexp.MustCompile(`(?m)^[\s]*([-\*\+]|\d+\.)\s+`)
	itemCount := len(itemPattern.FindAllString(content, -1))

	metadata := map[string]interface{}{
		"element_type": "list",
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(content),
		"item_count":   itemCount,
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

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeList,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "list", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createBlockquoteElement creates a blockquote element
func (p *MarkdownParser) createBlockquoteElement(docID, content string, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_quote")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	metadata := map[string]interface{}{
		"element_type": "blockquote",
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(content),
		"word_count":   len(strings.Fields(content)),
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

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeBlockquote,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "blockquote", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createTableElement creates a table element
func (p *MarkdownParser) createTableElement(docID, content string, lineNumber int, parentID string) MarkdownElement {
	elementID := p.generateElementID("md_table")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	// Count rows and estimate columns
	lines := strings.Split(content, "\n")
	rowCount := 0
	colCount := 0
	foundHeader := false

	for _, line := range lines {
		if strings.Contains(line, "|") {
			if strings.Contains(line, "---") {
				// This is the separator line, skip it
				continue
			}

			if !foundHeader {
				// First data line after potential header
				foundHeader = true
				// Count columns by splitting on | and removing empty entries
				parts := strings.Split(line, "|")
				actualCols := 0
				for _, part := range parts {
					if strings.TrimSpace(part) != "" {
						actualCols++
					}
				}
				colCount = actualCols
			} else {
				// Count data rows (excluding header)
				rowCount++
			}
		}
	}

	metadata := map[string]interface{}{
		"element_type": "table",
		"line_number":  lineNumber,
		"length":       utf8.RuneCountInString(content),
		"row_count":    rowCount,
		"column_count": colCount,
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

	return MarkdownElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     MarkdownElementTypeTable,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "table", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// parseCodeBlock parses a code block starting at the given index
func (p *MarkdownParser) parseCodeBlock(lines []string, startIndex int) (string, string, int) {
	if startIndex >= len(lines) {
		return "", "", startIndex + 1
	}

	startLine := lines[startIndex]
	if !strings.HasPrefix(startLine, "```") {
		return "", "", startIndex + 1
	}

	// Extract language
	language := strings.TrimSpace(startLine[3:])

	var codeLines []string
	i := startIndex + 1

	// Find closing ```
	for i < len(lines) {
		if strings.HasPrefix(lines[i], "```") {
			break
		}
		codeLines = append(codeLines, lines[i])
		i++
	}

	if i < len(lines) {
		i++ // Skip closing ```
	}

	return strings.Join(codeLines, "\n"), language, i
}

// parseBlockquote parses a blockquote starting at the given index
func (p *MarkdownParser) parseBlockquote(lines []string, startIndex int) (string, int) {
	var quoteLines []string
	i := startIndex

	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, ">") {
			// Remove > and optional space
			cleanLine := strings.TrimPrefix(line, ">")
			if strings.HasPrefix(cleanLine, " ") {
				cleanLine = cleanLine[1:]
			}
			quoteLines = append(quoteLines, cleanLine)
			i++
		} else if strings.TrimSpace(line) == "" {
			// Empty line ends the blockquote
			break
		} else {
			// Non-blockquote line ends the blockquote
			break
		}
	}

	return strings.Join(quoteLines, "\n"), i
}

// parseList parses a list starting at the given index
func (p *MarkdownParser) parseList(lines []string, startIndex int) (string, int) {
	var listLines []string
	i := startIndex

	// Pattern for list items (with indentation support)
	listPattern := regexp.MustCompile(`^(\s*)([-\*\+]|\d+\.)\s+(.+)$`)

	// Determine the list type from the first line
	firstMatch := listPattern.FindStringSubmatch(lines[startIndex])
	if firstMatch == nil {
		return "", startIndex
	}

	isOrdered := strings.Contains(firstMatch[2], ".")
	baseIndent := len(firstMatch[1])

	for i < len(lines) {
		line := lines[i]

		if match := listPattern.FindStringSubmatch(line); match != nil {
			// Check if it's the same type of list
			currentIsOrdered := strings.Contains(match[2], ".")
			currentIndent := len(match[1])

			// If list type changes and we're at the same indentation level, stop
			if currentIsOrdered != isOrdered && currentIndent <= baseIndent {
				break
			}

			listLines = append(listLines, line)
			i++
		} else if strings.TrimSpace(line) == "" {
			// Empty lines might be part of the list
			if i+1 < len(lines) {
				if nextMatch := listPattern.FindStringSubmatch(lines[i+1]); nextMatch != nil {
					nextIsOrdered := strings.Contains(nextMatch[2], ".")
					nextIndent := len(nextMatch[1])
					// Continue if it's the same list type or nested
					if nextIsOrdered == isOrdered || nextIndent > baseIndent {
						listLines = append(listLines, line)
						i++
					} else {
						break
					}
				} else {
					break
				}
			} else {
				break
			}
		} else if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			// Indented lines are part of the list item
			listLines = append(listLines, line)
			i++
		} else {
			// Non-list line ends the list
			break
		}
	}

	return strings.Join(listLines, "\n"), i
}

// parseTable parses a table starting at the given index
func (p *MarkdownParser) parseTable(lines []string, startIndex int) (string, int) {
	var tableLines []string
	i := startIndex

	for i < len(lines) {
		line := lines[i]
		if strings.Contains(line, "|") {
			tableLines = append(tableLines, line)
			i++
		} else if strings.TrimSpace(line) == "" {
			// Empty lines might be part of the table formatting
			if i+1 < len(lines) && strings.Contains(lines[i+1], "|") {
				tableLines = append(tableLines, line)
				i++
			} else {
				break
			}
		} else {
			// Non-table line ends the table
			break
		}
	}

	return strings.Join(tableLines, "\n"), i
}

// parseParagraph parses a paragraph starting at the given index
func (p *MarkdownParser) parseParagraph(lines []string, startIndex int) (string, int) {
	var paragraphLines []string
	i := startIndex

	for i < len(lines) {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		// End paragraph on empty line
		if trimmedLine == "" {
			i++
			break
		}

		// End paragraph on headers
		if regexp.MustCompile(`^#{1,6}\s+`).MatchString(trimmedLine) {
			break
		}

		// End paragraph on code blocks
		if strings.HasPrefix(trimmedLine, "```") {
			break
		}

		// End paragraph on lists
		if regexp.MustCompile(`^(\s*)([-\*\+]|\d+\.)\s+`).MatchString(trimmedLine) {
			break
		}

		// End paragraph on blockquotes
		if strings.HasPrefix(trimmedLine, ">") {
			break
		}

		// End paragraph on tables
		if strings.Contains(trimmedLine, "|") {
			break
		}

		paragraphLines = append(paragraphLines, line)
		i++
	}

	return strings.Join(paragraphLines, "\n"), i
}

// createRelationship creates a relationship between elements
func (p *MarkdownParser) createRelationship(sourceID, targetID, relType string) MarkdownRelationship {
	relationshipID := p.generateRelationshipID()

	return MarkdownRelationship{
		RelationshipID:   relationshipID,
		SourceElementID:  sourceID,
		TargetElementID:  targetID,
		RelationshipType: relType,
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"relationship_type": relType},
	}
}

// extractLinks extracts links from content
func (p *MarkdownParser) extractLinks(content, elementID string) []MarkdownLink {
	var links []MarkdownLink

	// Markdown links [text](url)
	markdownLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := markdownLinkPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			linkID := p.generateLinkID()
			linkType := "markdown"
			if strings.HasPrefix(match[2], "http") {
				linkType = "url"
			} else if strings.HasPrefix(match[2], "mailto:") {
				linkType = "email"
			} else if strings.HasSuffix(match[2], ".md") {
				linkType = "document"
			}

			links = append(links, MarkdownLink{
				LinkID:     linkID,
				LinkType:   linkType,
				LinkTarget: match[2],
				ElementID:  elementID,
				LinkText:   match[1],
			})
		}
	}

	// Wiki-style links [[Page]]
	wikiLinkPattern := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	wikiMatches := wikiLinkPattern.FindAllStringSubmatch(content, -1)
	for _, match := range wikiMatches {
		if len(match) == 2 {
			linkID := p.generateLinkID()
			links = append(links, MarkdownLink{
				LinkID:     linkID,
				LinkType:   "wiki",
				LinkTarget: match[1],
				ElementID:  elementID,
				LinkText:   match[1],
			})
		}
	}

	// URL patterns
	urlPattern := regexp.MustCompile(`https?://[^\s\)\]\}]+`)
	urls := urlPattern.FindAllString(content, -1)
	for _, url := range urls {
		linkID := p.generateLinkID()
		links = append(links, MarkdownLink{
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
		links = append(links, MarkdownLink{
			LinkID:     linkID,
			LinkType:   "email",
			LinkTarget: "mailto:" + email,
			ElementID:  elementID,
		})
	}

	return links
}

// extractDates extracts date patterns from content
func (p *MarkdownParser) extractDates(content string) []string {
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
func (p *MarkdownParser) extractNumbers(content string) []interface{} {
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
func (p *MarkdownParser) generateElementID(prefix string) string {
	p.ElementIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s_%d_%d", prefix, timestamp, p.ElementIDCounter)
}

// generateRelationshipID generates a unique relationship ID
func (p *MarkdownParser) generateRelationshipID() string {
	p.RelationshipIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("md_rel_%d_%d", timestamp, p.RelationshipIDCounter)
}

// generateLinkID generates a unique link ID
func (p *MarkdownParser) generateLinkID() string {
	p.LinkIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("md_link_%d_%d", timestamp, p.LinkIDCounter)
}

// calculateContentHash calculates MD5 hash of content
func (p *MarkdownParser) calculateContentHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content for preview
func (p *MarkdownParser) truncateContent(content string) string {
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

