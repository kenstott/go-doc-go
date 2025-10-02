package parser

import (
	"archive/zip"
	"crypto/md5"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// DocxElement represents a parsed DOCX element
type DocxElement struct {
	ElementID       string                 `json:"element_id"`
	ElementType     string                 `json:"element_type"`
	ParentID        string                 `json:"parent_id,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ElementOrder    int                    `json:"element_order"`
	DocumentOrder   int                    `json:"document_position"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// DocxRelationship represents a relationship between elements
type DocxRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_id"`
	TargetElementID  string                 `json:"target_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// DocxLink represents an extracted link
type DocxLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// DocxParseRequest represents the input for DOCX parsing
type DocxParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// DocxParseResponse represents the output of DOCX parsing
type DocxParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []DocxElement          `json:"elements"`
	Links         []DocxLink             `json:"links"`
	Relationships []DocxRelationship     `json:"relationships"`
}

// ToParseResult converts DocxParseResponse to the standard ParseResult format
func (r *DocxParseResponse) ToParseResult() *ParseResult {
	// Convert document metadata
	var doc Document
	if docID, ok := r.Document["doc_id"].(string); ok {
		doc.ID = docID
	}
	if title, ok := r.Document["title"].(string); ok {
		doc.Title = title
	}
	if docType, ok := r.Document["doc_type"].(string); ok {
		doc.DocType = docType
	}
	doc.Metadata = r.Document

	// Convert elements
	elements := make([]Element, len(r.Elements))
	for i, docxElem := range r.Elements {
		elements[i] = Element{
			ElementID:       docxElem.ElementID,
			ElementType:     docxElem.ElementType,
			Content:         "", // DOCX elements use ContentLocation
			ContentPreview:  docxElem.ContentPreview,
			ParentID:        docxElem.ParentID,
			Metadata:        docxElem.Metadata,
			ContentLocation: docxElem.ContentLocation,
		}
	}

	// Convert relationships
	relationships := make([]Relationship, len(r.Relationships))
	for i, docxRel := range r.Relationships {
		relationships[i] = Relationship{
			SourceElementID:  docxRel.SourceElementID,
			TargetElementID:  docxRel.TargetElementID,
			RelationshipType: docxRel.RelationshipType,
			Metadata:         docxRel.Metadata,
		}
	}

	// Convert links
	links := make([]Link, len(r.Links))
	for i, docxLink := range r.Links {
		links[i] = Link{
			SourceElementID: docxLink.SourceID,
			LinkTarget:      docxLink.LinkTarget,
			LinkText:        docxLink.LinkText,
			LinkType:        docxLink.LinkType,
		}
	}

	return &ParseResult{
		Document:      doc,
		Elements:      elements,
		Relationships: relationships,
		Links:         links,
	}
}

// XML structures for DOCX parsing
type WordDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    WordBody `xml:"body"`
}

type WordBody struct {
	Paragraphs []WordParagraph `xml:"p"`
	Tables     []WordTable     `xml:"tbl"`
}

type WordParagraph struct {
	Runs []WordRun `xml:"r"`
	Ppr  WordPpr   `xml:"pPr"`
}

type WordRun struct {
	Text []string `xml:"t"`
}

type WordPpr struct {
	PStyle WordPStyle `xml:"pStyle"`
}

type WordPStyle struct {
	Val string `xml:"val,attr"`
}

type WordTable struct {
	Rows []WordTableRow `xml:"tr"`
}

type WordTableRow struct {
	Cells []WordTableCell `xml:"tc"`
}

type WordTableCell struct {
	Paragraphs []WordParagraph `xml:"p"`
	TcPr       WordTcPr        `xml:"tcPr"`
}

type WordTcPr struct {
	GridSpan WordGridSpan `xml:"gridSpan"`
	VMerge   WordVMerge   `xml:"vMerge"`
}

type WordGridSpan struct {
	Val int `xml:"val,attr"`
}

type WordVMerge struct {
	Val string `xml:"val,attr"` // "restart" for start of merge, empty or "continue" for continuation
}

// DocxParser handles DOCX document parsing
type DocxParser struct {
	MaxContentPreview     int
	ExtractHeadersFooters bool
	ExtractComments       bool
	ExtractStyles         bool
	MaxImageSize          int
	ExtractDates          bool
}

// NewDocxParser creates a new DOCX parser instance
func NewDocxParser() *DocxParser {
	return &DocxParser{
		MaxContentPreview:     100,
		ExtractHeadersFooters: true,
		ExtractComments:       true,
		ExtractStyles:         true,
		MaxImageSize:          1024 * 1024, // 1MB
	}
}

// Parse parses a DOCX document into structured elements
func (p *DocxParser) Parse(request DocxParseRequest) (*DocxParseResponse, error) {
	// Open DOCX file as ZIP archive
	reader, err := zip.OpenReader(request.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer reader.Close()

	// Find and read document.xml and relationships
	var docXML []byte
	var relsXML []byte
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open document.xml: %w", err)
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read document.xml: %w", err)
			}
		} else if file.Name == "word/_rels/document.xml.rels" {
			rc, err := file.Open()
			if err != nil {
				// Relationships file is optional
				continue
			}
			relsXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				// Just log and continue
				continue
			}
		}
	}

	if docXML == nil {
		return nil, fmt.Errorf("document.xml not found in DOCX file")
	}

	// Parse relationships to extract hyperlinks
	hyperlinks := make(map[string]string) // relID -> target URL
	if len(relsXML) > 0 {
		p.parseRelationships(relsXML, hyperlinks)
	}

	// Parse XML
	var wordDoc WordDocument
	err = xml.Unmarshal(docXML, &wordDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document.xml: %w", err)
	}

	// Initialize response
	response := &DocxParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "docx",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []DocxElement{},
		Links:         []DocxLink{},
		Relationships: []DocxRelationship{},
	}

	// Create root element
	rootElement := DocxElement{
		ElementID:       p.generateID("root_"),
		ElementType:     "root",
		ContentPreview:  p.truncateContent(fmt.Sprintf("Document: %s", request.ID)),
		ContentLocation: p.createContentLocation(request.ID, "root", ""),
		ContentHash:     p.generateHash(request.Content),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, rootElement)

	// Parse document elements
	elementCounter := 1

	// Create container elements (matching Python structure)
	// 1. Headers container
	headersElement := DocxElement{
		ElementID:       p.generateID("headers_"),
		ElementType:     "headers",
		ParentID:        rootElement.ElementID,
		ContentPreview:  "Document headers",
		ContentLocation: p.createContentLocation(request.ID, "headers", ""),
		ContentHash:     p.generateHash("headers"),
		ElementOrder:    elementCounter,
		DocumentOrder:   elementCounter,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, headersElement)
	elementCounter++

	// 2. Footers container
	footersElement := DocxElement{
		ElementID:       p.generateID("footers_"),
		ElementType:     "footers",
		ParentID:        rootElement.ElementID,
		ContentPreview:  "Document footers",
		ContentLocation: p.createContentLocation(request.ID, "footers", ""),
		ContentHash:     p.generateHash("footers"),
		ElementOrder:    elementCounter,
		DocumentOrder:   elementCounter,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, footersElement)
	elementCounter++

	// 3. Parse main document body
	p.parseWordDocument(&wordDoc, &response.Elements, &response.Links, &response.Relationships,
		rootElement.ElementID, request.ID, &elementCounter)

	// 4. Comments container
	commentsElement := DocxElement{
		ElementID:       p.generateID("comments_"),
		ElementType:     "comments",
		ParentID:        rootElement.ElementID,
		ContentPreview:  "Document comments",
		ContentLocation: p.createContentLocation(request.ID, "comments", ""),
		ContentHash:     p.generateHash("comments"),
		ElementOrder:    elementCounter,
		DocumentOrder:   elementCounter,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, commentsElement)
	elementCounter++

	// Extract document metadata
	p.extractDocumentMetadata(&wordDoc, response.Document)

	// Extract hyperlinks from relationships
	if len(hyperlinks) > 0 {
		p.extractHyperlinksFromText(string(docXML), hyperlinks, response)
	}

	return response, nil
}

// parseWordDocument parses the Word document structure
func (p *DocxParser) parseWordDocument(doc *WordDocument, elements *[]DocxElement, links *[]DocxLink,
	relationships *[]DocxRelationship, parentID, sourceID string, counter *int) {

	// Create body element
	bodyElement := DocxElement{
		ElementID:       p.generateID("body_"),
		ElementType:     "body",
		ParentID:        parentID,
		ContentPreview:  "Document body",
		ContentLocation: p.createContentLocation(sourceID, "body", ""),
		ContentHash:     p.generateHash("body"),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata:        make(map[string]interface{}),
	}
	*elements = append(*elements, bodyElement)
	*counter++

	// Create relationship from root to body
	relationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  parentID,
		TargetElementID:  bodyElement.ElementID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, relationship)

	// Create inverse relationship
	inverseRelationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  bodyElement.ElementID,
		TargetElementID:  parentID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, inverseRelationship)

	// Parse paragraphs
	currentParent := bodyElement.ElementID
	sectionStack := []map[string]interface{}{
		{"id": bodyElement.ElementID, "level": 0},
	}

	// Process document content by mixing paragraphs and tables in order
	// Since XML parsing separates them, we'll process them sequentially

	// Process paragraphs
	for i, para := range doc.Body.Paragraphs {
		p.processParagraph(&para, i, elements, links, relationships, &currentParent, &sectionStack, sourceID, counter)
	}

	// Process tables
	for i, table := range doc.Body.Tables {
		p.processTable(&table, i, elements, links, relationships, currentParent, sourceID, counter)
	}
}

// processParagraph processes a paragraph element
func (p *DocxParser) processParagraph(para *WordParagraph, index int, elements *[]DocxElement, links *[]DocxLink,
	relationships *[]DocxRelationship, currentParent *string, sectionStack *[]map[string]interface{},
	sourceID string, counter *int) {

	text := p.extractParagraphText(para)
	if strings.TrimSpace(text) == "" {
		return // Skip empty paragraphs
	}

	// Determine element type and check for headers
	elementType := "paragraph"
	level := 0

	// Check if this is a header based on style
	if p.isHeaderParagraph(para, text) {
		elementType = "header"
		level = p.determineHeaderLevel(para, text)

		// Update section hierarchy - pop sections at same or higher level
		for len(*sectionStack) > 1 && (*sectionStack)[len(*sectionStack)-1]["level"].(int) >= level {
			*sectionStack = (*sectionStack)[:len(*sectionStack)-1]
		}
		// Parent is the top of the section stack (could be body or a parent header)
		*currentParent = (*sectionStack)[len(*sectionStack)-1]["id"].(string)
	}

	// Create paragraph/header element
	elementID := p.generateID(fmt.Sprintf("%s_", strings.ToLower(elementType)))
	element := DocxElement{
		ElementID:       elementID,
		ElementType:     elementType,
		ParentID:        *currentParent,
		ContentPreview:  p.truncateContent(text),
		ContentLocation: p.createContentLocationForParagraph(sourceID, elementType, index, level),
		ContentHash:     p.generateHash(text),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"index": index,
			"text":  text,
		},
	}

	// Extract temporal metadata if enabled
	if p.ExtractDates && text != "" {
		ProcessTemporalContent(text, element.Metadata)
	}

	if elementType == "header" {
		element.Metadata["level"] = level
		// Add to section stack and make it the current parent
		*sectionStack = append(*sectionStack, map[string]interface{}{
			"id":    elementID,
			"level": level,
		})
		// Headers become parents for subsequent content until a same/higher level header
		*currentParent = elementID
	}

	*elements = append(*elements, element)
	*counter++

	// Create relationship
	relType := "contains"
	if elementType == "paragraph" {
		relType = "contains_text"
	}

	relationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  element.ParentID,
		TargetElementID:  elementID,
		RelationshipType: relType,
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"index": index},
	}
	*relationships = append(*relationships, relationship)

	// Create inverse relationship
	inverseRelationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  elementID,
		TargetElementID:  element.ParentID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, inverseRelationship)

	// Extract links from paragraph
	p.extractLinksFromText(text, elementID, links)
}

// processTable processes a table element
func (p *DocxParser) processTable(table *WordTable, index int, elements *[]DocxElement, links *[]DocxLink,
	relationships *[]DocxRelationship, parentID, sourceID string, counter *int) {

	// Create table element
	tableID := p.generateID("table_")
	tableElement := DocxElement{
		ElementID:       tableID,
		ElementType:     "table",
		ParentID:        parentID,
		ContentPreview:  "Table",
		ContentLocation: p.createContentLocationForTable(sourceID, index),
		ContentHash:     p.generateHash(fmt.Sprintf("table_%d", index)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"index":   index,
			"rows":    len(table.Rows),
			"columns": p.getTableColumnCount(table),
		},
	}
	*elements = append(*elements, tableElement)
	*counter++

	// Create relationship from parent to table
	relationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  parentID,
		TargetElementID:  tableID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"index": index},
	}
	*relationships = append(*relationships, relationship)

	// Create inverse relationship
	inverseRelationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  tableID,
		TargetElementID:  parentID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, inverseRelationship)

	// Process table rows
	for rowIdx, row := range table.Rows {
		p.processTableRow(&row, rowIdx, index, elements, links, relationships, tableID, sourceID, counter)
	}
}

// processTableRow processes a table row
func (p *DocxParser) processTableRow(row *WordTableRow, rowIdx, tableIdx int, elements *[]DocxElement, links *[]DocxLink,
	relationships *[]DocxRelationship, tableID, sourceID string, counter *int) {

	// Create row element
	rowID := p.generateID("row_")
	rowElement := DocxElement{
		ElementID:       rowID,
		ElementType:     "table_row",
		ParentID:        tableID,
		ContentPreview:  fmt.Sprintf("Row %d", rowIdx+1),
		ContentLocation: p.createContentLocationForTableRow(sourceID, tableIdx, rowIdx),
		ContentHash:     p.generateHash(fmt.Sprintf("row_%d_%d", tableIdx, rowIdx)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"row":         rowIdx,
			"table_index": tableIdx,
		},
	}
	*elements = append(*elements, rowElement)
	*counter++

	// Create relationship from table to row
	relationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  tableID,
		TargetElementID:  rowID,
		RelationshipType: "contains_table_row",
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"row_index": rowIdx},
	}
	*relationships = append(*relationships, relationship)

	// Create inverse relationship
	inverseRelationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  rowID,
		TargetElementID:  tableID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, inverseRelationship)

	// Process table cells
	for colIdx, cell := range row.Cells {
		p.processTableCell(&cell, rowIdx, colIdx, tableIdx, elements, links, relationships, rowID, sourceID, counter)
	}
}

// processTableCell processes a table cell
func (p *DocxParser) processTableCell(cell *WordTableCell, rowIdx, colIdx, tableIdx int, elements *[]DocxElement, links *[]DocxLink,
	relationships *[]DocxRelationship, rowID, sourceID string, counter *int) {

	// Check if this is a vertically merged cell that should be skipped
	if cell.TcPr.VMerge.Val == "continue" || (cell.TcPr.VMerge.Val == "" && cell.TcPr.VMerge != (WordVMerge{})) {
		// This cell is part of a vertical merge continuation, skip it
		return
	}

	// Extract cell text
	cellText := p.extractCellText(cell)
	if strings.TrimSpace(cellText) == "" && cell.TcPr.GridSpan.Val == 0 && cell.TcPr.VMerge.Val != "restart" {
		return // Skip empty cells that aren't merged
	}

	// Determine cell type (header if first row)
	cellType := "table_cell"
	if rowIdx == 0 {
		cellType = "table_header"
	}

	// Check for horizontal merge (colspan)
	colspan := 1
	if cell.TcPr.GridSpan.Val > 0 {
		colspan = cell.TcPr.GridSpan.Val
	}

	// Check for vertical merge start (rowspan)
	rowspan := 1
	if cell.TcPr.VMerge.Val == "restart" {
		// Count how many consecutive cells have vMerge continue
		rowspan = p.calculateRowSpan(cell, rowIdx, colIdx, tableIdx)
	}

	// Mark as merged cell if spans are greater than 1
	if colspan > 1 || rowspan > 1 {
		cellType = "merged_cell"
	}

	// Create cell element
	cellID := p.generateID("cell_")
	cellElement := DocxElement{
		ElementID:       cellID,
		ElementType:     cellType,
		ParentID:        rowID,
		ContentPreview:  p.truncateContent(cellText),
		ContentLocation: p.createContentLocationForTableCell(sourceID, tableIdx, rowIdx, colIdx, cellType),
		ContentHash:     p.generateHash(cellText),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"row":         rowIdx,
			"col":         colIdx,
			"table_index": tableIdx,
			"text":        cellText,
			"colspan":     colspan,
			"rowspan":     rowspan,
		},
	}

	// Extract temporal metadata if enabled
	if p.ExtractDates && cellText != "" {
		ProcessTemporalContent(cellText, cellElement.Metadata)
	}

	*elements = append(*elements, cellElement)
	*counter++

	// Create relationship from row to cell
	relType := "contains_table_cell"
	if cellType == "table_header" {
		relType = "contains_table_header"
	}

	relationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  rowID,
		TargetElementID:  cellID,
		RelationshipType: relType,
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"col_index": colIdx},
	}
	*relationships = append(*relationships, relationship)

	// Create inverse relationship
	inverseRelationship := DocxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  cellID,
		TargetElementID:  rowID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, inverseRelationship)

	// Extract links from cell
	p.extractLinksFromText(cellText, cellID, links)
}

// Helper methods

// calculateRowSpan calculates the rowspan for a vertically merged cell
func (p *DocxParser) calculateRowSpan(cell *WordTableCell, rowIdx, colIdx, tableIdx int) int {
	// This is a simplified implementation
	// In a full implementation, you would need to access the table structure
	// to check subsequent rows for vMerge="continue" in the same column
	// For now, return 1 as default
	return 1
}

// extractParagraphText extracts text from a paragraph
func (p *DocxParser) extractParagraphText(para *WordParagraph) string {
	var texts []string
	for _, run := range para.Runs {
		for _, text := range run.Text {
			if strings.TrimSpace(text) != "" {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, " ")
}

// extractCellText extracts text from a table cell
func (p *DocxParser) extractCellText(cell *WordTableCell) string {
	var texts []string
	for _, para := range cell.Paragraphs {
		text := p.extractParagraphText(&para)
		if strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, " ")
}

// isHeaderParagraph determines if a paragraph is a header
func (p *DocxParser) isHeaderParagraph(para *WordParagraph, text string) bool {
	// Check for heading style
	if para.Ppr.PStyle.Val != "" {
		style := strings.ToLower(para.Ppr.PStyle.Val)
		if strings.Contains(style, "heading") || strings.Contains(style, "title") {
			return true
		}
	}

	// Simple heuristics for header detection
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}

	// Headers are typically short
	if len(text) > 100 {
		return false
	}

	// Check for title case or all caps (common in headers)
	titleCaseCount := 0
	for _, word := range words {
		if len(word) > 0 && strings.ToUpper(word[:1]) == word[:1] {
			titleCaseCount++
		}
	}

	// If most words are title case, likely a header
	return float64(titleCaseCount)/float64(len(words)) > 0.6
}

// determineHeaderLevel determines the level of a header
func (p *DocxParser) determineHeaderLevel(para *WordParagraph, text string) int {
	// Check style first
	if para.Ppr.PStyle.Val != "" {
		style := strings.ToLower(para.Ppr.PStyle.Val)
		if strings.Contains(style, "heading") {
			// Extract number from heading style
			re := regexp.MustCompile(`heading\s*(\d+)`)
			matches := re.FindStringSubmatch(style)
			if len(matches) > 1 {
				if level := parseInt(matches[1]); level > 0 && level <= 6 {
					return level
				}
			}
		}
		if strings.Contains(style, "title") {
			return 1
		}
	}

	// Default heuristics
	if len(text) > 50 {
		return 3
	} else if len(text) > 30 {
		return 2
	}
	return 1
}

// getTableColumnCount gets the maximum number of columns in a table
func (p *DocxParser) getTableColumnCount(table *WordTable) int {
	maxCols := 0
	for _, row := range table.Rows {
		if len(row.Cells) > maxCols {
			maxCols = len(row.Cells)
		}
	}
	return maxCols
}

// extractLinksFromText extracts links from text
func (p *DocxParser) extractLinksFromText(text, elementID string, links *[]DocxLink) {
	// Simple URL detection in text
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlRegex.FindAllString(text, -1)

	for _, url := range urls {
		link := DocxLink{
			SourceID:   elementID,
			LinkText:   url,
			LinkTarget: url,
			LinkType:   "url",
		}
		*links = append(*links, link)
	}
}

// extractDocumentMetadata extracts metadata from the document
func (p *DocxParser) extractDocumentMetadata(doc *WordDocument, document map[string]interface{}) {
	// Extract basic document statistics
	metadata := document["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
		document["metadata"] = metadata
	}

	// Count paragraphs and tables
	metadata["paragraph_count"] = len(doc.Body.Paragraphs)
	metadata["table_count"] = len(doc.Body.Tables)

	// Estimate word count
	wordCount := 0
	for _, para := range doc.Body.Paragraphs {
		text := p.extractParagraphText(&para)
		wordCount += len(strings.Fields(text))
	}
	metadata["word_count"] = wordCount

	// Estimate page count (250 words per page)
	pageCount := wordCount / 250
	if pageCount < 1 {
		pageCount = 1
	}
	metadata["page_count"] = pageCount
}

// Content location helpers

// createContentLocation creates a content location object
func (p *DocxParser) createContentLocation(source, elementType, selector string) map[string]interface{} {
	return map[string]interface{}{
		"source":   source,
		"type":     elementType,
		"selector": selector,
	}
}

// createContentLocationForParagraph creates content location for a paragraph
func (p *DocxParser) createContentLocationForParagraph(source, elementType string, index, level int) map[string]interface{} {
	location := map[string]interface{}{
		"source": source,
		"type":   elementType,
		"index":  index,
	}
	if elementType == "header" {
		location["level"] = level
	}
	return location
}

// createContentLocationForTable creates content location for a table
func (p *DocxParser) createContentLocationForTable(source string, index int) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   "table",
		"index":  index,
	}
}

// createContentLocationForTableRow creates content location for a table row
func (p *DocxParser) createContentLocationForTableRow(source string, tableIndex, row int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "table_row",
		"table_index": tableIndex,
		"row":         row,
	}
}

// createContentLocationForTableCell creates content location for a table cell
func (p *DocxParser) createContentLocationForTableCell(source string, tableIndex, row, col int, cellType string) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        cellType,
		"table_index": tableIndex,
		"row":         row,
		"col":         col,
	}
}

// Utility methods

// generateID generates a unique ID with the given prefix
func (p *DocxParser) generateID(prefix string) string {
	// Generate UUID-based unique ID like Python does
	// Use random hex string (8 characters from UUID)
	id := make([]byte, 4)
	if _, err := rand.Read(id); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s%x", prefix, id)
}

// generateHash generates an MD5 hash of the content
func (p *DocxParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content to the maximum preview length
func (p *DocxParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// parseInt safely parses an integer string
func parseInt(s string) int {
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			break
		}
	}
	return result
}

// ToJSON converts the parse response to JSON
func (r *DocxParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a DocxParseRequest from JSON
func (r *DocxParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}