package parser

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// CSVElementType represents the type of CSV element
type CSVElementType string

const (
	CSVElementTypeRoot           CSVElementType = "root"
	CSVElementTypeTable          CSVElementType = "table"
	CSVElementTypeTableRow       CSVElementType = "table_row"
	CSVElementTypeTableHeaderRow CSVElementType = "table_header_row"
	CSVElementTypeTableCell      CSVElementType = "table_cell"
)

// CSVElement represents a parsed CSV element
type CSVElement struct {
	ElementID       string                 `json:"element_id"`
	DocID           string                 `json:"doc_id"`
	ElementType     CSVElementType         `json:"element_type"`
	ParentID        string                 `json:"parent_id,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ElementOrder    int                    `json:"element_order"`
	DocumentOrder   int                    `json:"document_position"`
	Metadata        map[string]interface{} `json:"metadata"`
	Text            string                 `json:"text,omitempty"`
	Content         string                 `json:"content,omitempty"`
	TemporalValue   interface{}            `json:"temporal_value,omitempty"`
}

// CSVLink represents an extracted link
type CSVLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// CSVRelationship represents a relationship between elements
type CSVRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// CSVParseRequest represents the input for CSV parsing
type CSVParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// CSVParseResponse represents the output of CSV parsing
type CSVParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []CSVElement           `json:"elements"`
	Links         []CSVLink              `json:"links"`
	Relationships []CSVRelationship      `json:"relationships"`
	Dates         map[string]interface{} `json:"dates,omitempty"`
}

// CSVParser handles CSV document parsing
type CSVParser struct {
	MaxContentPreview    int
	ExtractHeader        bool
	Delimiter            rune
	MaxRows              int
	MaxPreviewColumns    int
	StripWhitespace      bool
	EnableLinkExtraction bool
	ExtractDates         bool
}

// NewCSVParser creates a new CSV parser instance
func NewCSVParser() *CSVParser {
	return &CSVParser{
		MaxContentPreview:    100,
		ExtractHeader:        true,
		Delimiter:            ',',
		MaxRows:              1000,
		MaxPreviewColumns:    5,
		StripWhitespace:      true,
		EnableLinkExtraction: true,
	}
}

// Parse is the universal interface that converts CSV content to ParseResult
func (p *CSVParser) Parse(docID string, content interface{}) (*ParseResult, error) {
	// Handle different input types
	var csvContent string
	switch v := content.(type) {
	case string:
		csvContent = v
	case []byte:
		csvContent = string(v)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Create request structure for compatibility
	request := CSVParseRequest{
		ID:       docID,
		Content:  csvContent,
		Metadata: make(map[string]interface{}),
	}

	// Parse using existing implementation
	response, err := p.parseCSV(request)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult
	return p.convertToParseResult(response), nil
}

// parseCSV parses a CSV document into structured elements (internal implementation)
func (p *CSVParser) parseCSV(request CSVParseRequest) (*CSVParseResponse, error) {
	// Parse CSV content
	csvData, err := p.parseCSVContent(request.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Handle empty CSV
	if len(csvData) == 0 {
		return p.createEmptyResponse(request), nil
	}

	// Initialize response
	response := &CSVParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "csv",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []CSVElement{},
		Links:         []CSVLink{},
		Relationships: []CSVRelationship{},
	}

	// Create root element
	rootElement := CSVElement{
		ElementID:       p.generateID("root_"),
		DocID:           request.ID,
		ElementType:     CSVElementTypeRoot,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Document: %s", request.ID)),
		ContentLocation: p.createContentLocation(request.ID, CSVElementTypeRoot, "/"),
		ContentHash:     p.generateHash(request.ID),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata: map[string]interface{}{
			"source_id": request.ID,
			"path":      "/",
		},
	}
	response.Elements = append(response.Elements, rootElement)

	// Create table element
	tableElement := CSVElement{
		ElementID:       p.generateID("table_"),
		DocID:           request.ID,
		ElementType:     CSVElementTypeTable,
		ParentID:        rootElement.ElementID,
		ContentPreview:  p.truncateContent(fmt.Sprintf("CSV table with %d rows", len(csvData))),
		ContentLocation: p.createContentLocation(request.ID, CSVElementTypeTable, "/table"),
		ContentHash:     p.generateHash("csv_table"),
		ElementOrder:    1,
		DocumentOrder:   1,
		Metadata: map[string]interface{}{
			"rows":       len(csvData),
			"columns":    p.getColumnCount(csvData),
			"has_header": p.ExtractHeader,
		},
	}
	response.Elements = append(response.Elements, tableElement)

	// Create relationship from root to table
	rootToTableRel := CSVRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  rootElement.ElementID,
		TargetElementID:  tableElement.ElementID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	response.Relationships = append(response.Relationships, rootToTableRel)

	elementCounter := 2

	// Process header row if enabled
	var headerRow []string
	if p.ExtractHeader && len(csvData) > 0 {
		headerRow = csvData[0]
		headerElement := p.createHeaderRowElement(request.ID, tableElement.ElementID, headerRow, elementCounter)
		response.Elements = append(response.Elements, headerElement)
		elementCounter++

		// Create relationship from table to header
		tableToHeaderRel := CSVRelationship{
			RelationshipID:   p.generateID("rel_"),
			SourceElementID:  tableElement.ElementID,
			TargetElementID:  headerElement.ElementID,
			RelationshipType: "contains",
			Confidence:       1.0,
			Metadata:         map[string]interface{}{"row": 0},
		}
		response.Relationships = append(response.Relationships, tableToHeaderRel)

		// Create header cells
		for colIdx, cellValue := range headerRow {
			cellElement := p.createCellElement(request.ID, headerElement.ElementID, cellValue, 0, colIdx, nil, elementCounter)
			response.Elements = append(response.Elements, cellElement)
			elementCounter++

			// Create relationship from header row to cell
			headerToCellRel := CSVRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  headerElement.ElementID,
				TargetElementID:  cellElement.ElementID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         map[string]interface{}{"col_index": colIdx, "is_header": true},
			}
			response.Relationships = append(response.Relationships, headerToCellRel)
		}
	}

	// Process data rows
	dataStartIdx := 0
	if p.ExtractHeader && len(csvData) > 0 {
		dataStartIdx = 1
	}

	maxRowsToProcess := min(len(csvData), dataStartIdx+p.MaxRows)
	for rowIdx := dataStartIdx; rowIdx < maxRowsToProcess; rowIdx++ {
		row := csvData[rowIdx]

		// Create row element
		rowElement := p.createRowElement(request.ID, tableElement.ElementID, row, rowIdx, elementCounter)
		response.Elements = append(response.Elements, rowElement)
		elementCounter++

		// Create relationship from table to row
		tableToRowRel := CSVRelationship{
			RelationshipID:   p.generateID("rel_"),
			SourceElementID:  tableElement.ElementID,
			TargetElementID:  rowElement.ElementID,
			RelationshipType: "contains",
			Confidence:       1.0,
			Metadata:         map[string]interface{}{"row_index": rowIdx},
		}
		response.Relationships = append(response.Relationships, tableToRowRel)

		// Process cells in this row
		for colIdx, cellValue := range row {
			cellElement := p.createCellElement(request.ID, rowElement.ElementID, cellValue, rowIdx, colIdx, headerRow, elementCounter)
			response.Elements = append(response.Elements, cellElement)
			elementCounter++

			// Create relationship from row to cell
			rowToCellRel := CSVRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  rowElement.ElementID,
				TargetElementID:  cellElement.ElementID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         map[string]interface{}{"col_index": colIdx},
			}
			response.Relationships = append(response.Relationships, rowToCellRel)

			// Extract links from cell if enabled
			if p.EnableLinkExtraction {
				cellLinks := p.extractLinksFromText(cellValue, cellElement.ElementID)
				response.Links = append(response.Links, cellLinks...)
			}
		}
	}

	// Add document metadata
	response.Document["metadata"] = p.createDocumentMetadata(csvData, headerRow, request.Metadata)

	return response, nil
}

// Helper functions

func (p *CSVParser) parseCSVContent(content string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = p.Delimiter
	reader.TrimLeadingSpace = p.StripWhitespace
	reader.FieldsPerRecord = -1

	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if p.StripWhitespace {
			for i, field := range record {
				record[i] = strings.TrimSpace(field)
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func (p *CSVParser) createEmptyResponse(request CSVParseRequest) *CSVParseResponse {
	rootElement := CSVElement{
		ElementID:       p.generateID("root_"),
		DocID:           request.ID,
		ElementType:     CSVElementTypeRoot,
		ContentPreview:  "Empty CSV document",
		ContentLocation: p.createContentLocation(request.ID, CSVElementTypeRoot, "/"),
		ContentHash:     p.generateHash(request.ID),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata: map[string]interface{}{
			"source_id": request.ID,
			"path":      "/",
		},
	}

	return &CSVParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "csv",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []CSVElement{rootElement},
		Links:         []CSVLink{},
		Relationships: []CSVRelationship{},
	}
}

func (p *CSVParser) createHeaderRowElement(docID, parentID string, headerRow []string, order int) CSVElement {
	preview := p.createRowPreview(headerRow)

	return CSVElement{
		ElementID:       p.generateID("header_"),
		DocID:           docID,
		ElementType:     CSVElementTypeTableHeaderRow,
		ParentID:        parentID,
		ContentPreview:  preview,
		ContentLocation: p.createRowContentLocation(docID, CSVElementTypeTableHeaderRow, 0),
		ContentHash:     p.generateHash(strings.Join(headerRow, ",")),
		ElementOrder:    order,
		DocumentOrder:   order,
		Metadata: map[string]interface{}{
			"row":          0,
			"values":       headerRow,
			"column_count": len(headerRow),
		},
	}
}

func (p *CSVParser) createRowElement(docID, parentID string, row []string, rowIdx, order int) CSVElement {
	preview := p.createRowPreview(row)

	return CSVElement{
		ElementID:       p.generateID(fmt.Sprintf("row_%d_", rowIdx)),
		DocID:           docID,
		ElementType:     CSVElementTypeTableRow,
		ParentID:        parentID,
		ContentPreview:  preview,
		ContentLocation: p.createRowContentLocation(docID, CSVElementTypeTableRow, rowIdx),
		ContentHash:     p.generateHash(strings.Join(row, ",")),
		ElementOrder:    order,
		DocumentOrder:   order,
		Metadata: map[string]interface{}{
			"row":          rowIdx,
			"values":       row,
			"column_count": len(row),
		},
	}
}

func (p *CSVParser) createCellElement(docID, parentID, cellValue string, rowIdx, colIdx int, headerRow []string, order int) CSVElement {
	preview := p.truncateContent(cellValue)

	// Get header name if available
	headerName := ""
	if headerRow != nil && colIdx < len(headerRow) {
		headerName = headerRow[colIdx]
	}

	return CSVElement{
		ElementID:       p.generateID(fmt.Sprintf("cell_%d_%d_", rowIdx, colIdx)),
		DocID:           docID,
		ElementType:     CSVElementTypeTableCell,
		ParentID:        parentID,
		ContentPreview:  preview,
		ContentLocation: p.createCellContentLocation(docID, rowIdx, colIdx),
		ContentHash:     p.generateHash(cellValue),
		ElementOrder:    order,
		DocumentOrder:   order,
		Text:            cellValue,
		Content:         cellValue,
		Metadata: map[string]interface{}{
			"row":    rowIdx,
			"col":    colIdx,
			"header": headerName,
			"value":  cellValue,
		},
	}
}

func (p *CSVParser) createRowPreview(row []string) string {
	if len(row) == 0 {
		return ""
	}

	maxCols := min(len(row), p.MaxPreviewColumns)
	preview := strings.Join(row[:maxCols], ", ")

	if len(row) > p.MaxPreviewColumns {
		preview += "..."
	}

	return p.truncateContent(preview)
}

func (p *CSVParser) createContentLocation(source string, elementType CSVElementType, path string) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   string(elementType),
		"path":   path,
	}
}

func (p *CSVParser) createRowContentLocation(source string, elementType CSVElementType, row int) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   string(elementType),
		"row":    row,
	}
}

func (p *CSVParser) createCellContentLocation(source string, row, col int) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   string(CSVElementTypeTableCell),
		"row":    row,
		"col":    col,
	}
}

func (p *CSVParser) createDocumentMetadata(csvData [][]string, headerRow []string, baseMetadata map[string]interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Copy base metadata
	for k, v := range baseMetadata {
		metadata[k] = v
	}

	// Add CSV-specific metadata
	metadata["row_count"] = len(csvData)
	metadata["column_count"] = p.getColumnCount(csvData)
	metadata["has_header"] = p.ExtractHeader

	if headerRow != nil {
		metadata["headers"] = headerRow
		metadata["column_types"] = p.detectColumnTypes(csvData, headerRow)
	}

	return metadata
}

func (p *CSVParser) detectColumnTypes(csvData [][]string, headerRow []string) []string {
	if len(csvData) <= 1 || headerRow == nil {
		return []string{}
	}

	columnTypes := make([]string, len(headerRow))

	for colIdx := range headerRow {
		// Sample values from this column
		var values []string
		for rowIdx := 1; rowIdx < len(csvData) && rowIdx <= 10; rowIdx++ { // Sample first 10 rows
			if colIdx < len(csvData[rowIdx]) {
				value := csvData[rowIdx][colIdx]
				if strings.TrimSpace(value) != "" {
					values = append(values, value)
				}
			}
		}

		columnTypes[colIdx] = p.detectColumnType(values)
	}

	return columnTypes
}

func (p *CSVParser) detectColumnType(values []string) string {
	if len(values) == 0 {
		return "string"
	}

	// Check if all values are integers
	intCount := 0
	floatCount := 0
	dateCount := 0
	boolCount := 0

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		// Check integer
		if _, err := strconv.Atoi(value); err == nil {
			intCount++
			continue
		}

		// Check float
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			floatCount++
			continue
		}

		// Check boolean
		lowerValue := strings.ToLower(value)
		if lowerValue == "true" || lowerValue == "false" ||
			lowerValue == "yes" || lowerValue == "no" ||
			lowerValue == "1" || lowerValue == "0" ||
			lowerValue == "y" || lowerValue == "n" {
			boolCount++
			continue
		}

		// Check date pattern
		if matched, _ := regexp.MatchString(`^\d{1,4}[-/\.]\d{1,2}[-/\.]\d{1,4}$`, value); matched {
			dateCount++
		}
	}

	totalNonEmpty := len(values)

	// Determine type based on majority
	if intCount == totalNonEmpty {
		return "integer"
	}
	if floatCount == totalNonEmpty || (intCount+floatCount) == totalNonEmpty {
		return "float"
	}
	if boolCount == totalNonEmpty {
		return "boolean"
	}
	if dateCount >= totalNonEmpty/2 {
		return "date"
	}

	return "string"
}

func (p *CSVParser) extractLinksFromText(text, sourceID string) []CSVLink {
	var links []CSVLink

	// URL regex pattern
	urlRegex := regexp.MustCompile(`https?://[^\s,\]"']+`)
	urlMatches := urlRegex.FindAllString(text, -1)
	for _, match := range urlMatches {
		link := CSVLink{
			SourceID:   sourceID,
			LinkText:   match,
			LinkTarget: match,
			LinkType:   "url",
		}
		links = append(links, link)
	}

	// Email regex pattern
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emailMatches := emailRegex.FindAllString(text, -1)
	for _, match := range emailMatches {
		link := CSVLink{
			SourceID:   sourceID,
			LinkText:   match,
			LinkTarget: "mailto:" + match,
			LinkType:   "email",
		}
		links = append(links, link)
	}

	return links
}

func (p *CSVParser) getColumnCount(csvData [][]string) int {
	if len(csvData) == 0 {
		return 0
	}

	maxCols := 0
	for _, row := range csvData {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	return maxCols
}

func (p *CSVParser) generateID(prefix string) string {
	return generateID(prefix)
}

func (p *CSVParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (p *CSVParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// ToJSON converts the parse response to JSON
func (r *CSVParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a ParseRequest from JSON
func (r *CSVParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// convertToParseResult converts CSVParseResponse to universal ParseResult format
func (p *CSVParser) convertToParseResult(response *CSVParseResponse) *ParseResult {
	result := &ParseResult{
		Document: Document{
			ID:      response.Document["doc_id"].(string),
			DocType: response.Document["doc_type"].(string),
		},
		Elements:      make([]Element, 0, len(response.Elements)),
		Relationships: make([]Relationship, 0, len(response.Relationships)),
		Links:         make([]Link, 0, len(response.Links)),
	}

	// Convert metadata if present
	if meta, ok := response.Document["metadata"]; ok {
		result.Document.Metadata = meta.(map[string]interface{})
	}

	// Convert elements
	for i, csvElem := range response.Elements {
		element := Element{
			ElementID:       csvElem.ElementID,
			ElementType:     string(csvElem.ElementType),
			Content:         csvElem.Content,
			ContentPreview:  csvElem.ContentPreview,
			ParentID:        csvElem.ParentID,
			Position:        i,
			Depth:           0, // Will be calculated based on hierarchy
			ContentLocation: csvElem.ContentLocation,
			Metadata:        csvElem.Metadata,
		}

		// Calculate depth based on parent relationships
		if csvElem.ElementType == CSVElementTypeRoot {
			element.Depth = 0
		} else if csvElem.ElementType == CSVElementTypeTable {
			element.Depth = 1
		} else if csvElem.ElementType == CSVElementTypeTableHeaderRow || csvElem.ElementType == CSVElementTypeTableRow {
			element.Depth = 2
		} else if csvElem.ElementType == CSVElementTypeTableCell {
			element.Depth = 3
		}

		result.Elements = append(result.Elements, element)
	}

	// Convert relationships
	for _, csvRel := range response.Relationships {
		relationship := Relationship{
			RelationshipID:   csvRel.RelationshipID,
			RelationshipType: csvRel.RelationshipType,
			SourceElementID:  csvRel.SourceElementID,
			TargetElementID:  csvRel.TargetElementID,
			Confidence:       csvRel.Confidence,
			Metadata:         csvRel.Metadata,
		}
		result.Relationships = append(result.Relationships, relationship)
	}

	// Convert links
	for _, csvLink := range response.Links {
		link := Link{
			LinkID:          generateID("link"),
			SourceElementID: csvLink.SourceID,
			LinkType:        csvLink.LinkType,
			LinkTarget:      csvLink.LinkTarget,
			LinkText:        csvLink.LinkText,
		}
		result.Links = append(result.Links, link)
	}

	return result
}

// Utility function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
