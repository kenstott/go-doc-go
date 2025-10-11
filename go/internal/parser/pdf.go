package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/gen2brain/go-fitz"
	"golang.org/x/net/html"
)

// PDFParser handles PDF document parsing with text and table extraction using go-fitz
type PDFParser struct {
	MaxContentPreview int
	MaxPages          int
	ExtractLinks      bool
	ExtractTables     bool
	DetectHeaders     bool
	MinHeaderFontSize float64
}

// NewPDFParser creates a new PDF parser with go-fitz
func NewPDFParser() *PDFParser {
	return &PDFParser{
		MaxContentPreview: 100,
		MaxPages:          1000,
		ExtractLinks:      true,
		ExtractTables:     true,
		DetectHeaders:     true,
		MinHeaderFontSize: 12.0,
	}
}

// GetName returns the parser name
func (p *PDFParser) GetName() string {
	return "pdf"
}

// GetSupportedFormats returns supported file formats
func (p *PDFParser) GetSupportedFormats() []string {
	return []string{".pdf", "pdf"}
}

// Parse implements the Parser interface for PDF documents
func (p *PDFParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract content from request - PDF parser expects file path or []byte
	return p.ParseLegacy(req.ID, req.Content)
}

// SupportsStreaming returns whether the parser supports streaming
func (p *PDFParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *PDFParser) Close() error {
	// PDF parser has no persistent resources to clean up
	return nil
}

// ParseLegacy parses PDF content using go-fitz (MuPDF wrapper)
// Deprecated: Use Parse(ctx, ParseRequest) instead
func (p *PDFParser) ParseLegacy(docID string, content interface{}) (*ParseResult, error) {
	var data []byte
	var err error

	// Handle different input types
	switch v := content.(type) {
	case string:
		// File path
		data, err = os.ReadFile(v)
		if err != nil {
			return nil, fmt.Errorf("failed to read PDF file: %v", err)
		}

	case []byte:
		// Raw bytes
		data = v

	case io.Reader:
		// Reader
		data, err = io.ReadAll(v)
		if err != nil {
			return nil, fmt.Errorf("failed to read PDF data: %v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Open PDF with go-fitz
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PDF with go-fitz: %v", err)
	}
	defer doc.Close()

	// Initialize result
	result := &ParseResult{
		Document: Document{
			ID:      docID,
			DocType: "pdf",
		},
		Elements:      []Element{},
		Relationships: []Relationship{},
		Links:         []Link{},
	}

	// Create root element
	rootID := generateID("root_")
	rootElement := Element{
		ElementID:       rootID,
		ElementType:     "root",
		ContentPreview:  "PDF Document",
		Position:        0,
		Depth:           0,
		ElementCategory: GetElementCategory("root"),
	}
	result.Elements = append(result.Elements, rootElement)

	// Get number of pages
	numPages := doc.NumPage()

	// Create body element
	bodyID := generateID("body_")
	bodyElement := Element{
		ElementID:       bodyID,
		ElementType:     "body",
		ContentPreview:  fmt.Sprintf("PDF content with %d pages", numPages),
		ParentID:        rootID,
		Position:        1,
		Depth:           1,
		ElementCategory: GetElementCategory("body"),
	}
	result.Elements = append(result.Elements, bodyElement)

	// Add root->body relationships
	result.Relationships = append(result.Relationships,
		Relationship{
			RelationshipID:   generateID("rel_"),
			RelationshipType: "contains",
			SourceElementID:  rootID,
			TargetElementID:  bodyID,
		},
		Relationship{
			RelationshipID:   generateID("rel_"),
			RelationshipType: "contained_by",
			SourceElementID:  bodyID,
			TargetElementID:  rootID,
		},
	)

	// Process pages
	pageCount := numPages
	if p.MaxPages > 0 && pageCount > p.MaxPages {
		pageCount = p.MaxPages
	}

	elementPosition := 2
	for pageNum := 0; pageNum < pageCount; pageNum++ {
		pageElements, pageRels, pos := p.processPage(doc, pageNum, bodyID, &elementPosition)
		result.Elements = append(result.Elements, pageElements...)
		result.Relationships = append(result.Relationships, pageRels...)
		elementPosition = pos
	}

	return result, nil
}

// processPage processes a single PDF page
func (p *PDFParser) processPage(doc *fitz.Document, pageNum int, bodyID string, elementPosition *int) ([]Element, []Relationship, int) {
	elements := []Element{}
	relationships := []Relationship{}

	// Create page element
	pageID := generateID(fmt.Sprintf("page_%d_", pageNum+1))
	pageNumber := pageNum + 1
	pageElement := Element{
		ElementID:      pageID,
		ElementType:    "page",
		ContentPreview: fmt.Sprintf("Page %d", pageNumber),
		ParentID:       bodyID,
		Position:       *elementPosition,
		Depth:          2,
		ContentLocation: map[string]interface{}{
			"page_number": pageNumber,
		},
		// UDML Phase 1: Populate promoted fields
		PageNumber:      &pageNumber,
		ElementCategory: GetElementCategory("page"),
	}
	elements = append(elements, pageElement)
	*elementPosition++

	// Add body->page relationships
	relationships = append(relationships,
		Relationship{
			RelationshipID:   generateID("rel_"),
			RelationshipType: "contains",
			SourceElementID:  bodyID,
			TargetElementID:  pageID,
		},
		Relationship{
			RelationshipID:   generateID("rel_"),
			RelationshipType: "contained_by",
			SourceElementID:  pageID,
			TargetElementID:  bodyID,
		},
	)

	// Extract text from page
	pageText, err := doc.Text(pageNum)
	if err != nil || pageText == "" {
		return elements, relationships, *elementPosition
	}

	// Get text blocks with positioning information
	textBlocks := p.extractTextBlocks(doc, pageNum, pageText)

	// Process text blocks
	var prevBlockID string
	for _, block := range textBlocks {
		blockID := generateID("block_")
		blockType := "paragraph"

		// Detect if it's a header based on font size
		if p.DetectHeaders && block.IsLargeFont {
			blockType = "header"
		}

		metadata := map[string]interface{}{
			"page_number": pageNum + 1,
		}

		// Extract temporal metadata from block text
		ProcessTemporalContent(block.Text, metadata)

		blockElement := Element{
			ElementID:       blockID,
			ElementType:     blockType,
			Content:         block.Text,
			ContentPreview:  p.truncateText(block.Text, p.MaxContentPreview),
			ParentID:        pageID,
			Position:        *elementPosition,
			Depth:           3,
			ContentLocation: metadata,
			Metadata:        metadata,
		}

		// UDML Phase 1: Populate promoted fields
		pageNum := pageNum + 1
		blockElement.PageNumber = &pageNum

		// TemporalType from temporal metadata
		if temporalType, ok := metadata["temporal_type"].(string); ok && temporalType != "" {
			blockElement.TemporalType = &temporalType
		}

		// ElementCategory
		blockElement.ElementCategory = GetElementCategory(blockType)

		elements = append(elements, blockElement)
		*elementPosition++

		// Add page->block relationships
		relationships = append(relationships,
			Relationship{
				RelationshipID:   generateID("rel_"),
				RelationshipType: "contains",
				SourceElementID:  pageID,
				TargetElementID:  blockID,
			},
			Relationship{
				RelationshipID:   generateID("rel_"),
				RelationshipType: "contained_by",
				SourceElementID:  blockID,
				TargetElementID:  pageID,
			},
		)

		// Add sibling relationships (previous/next)
		if prevBlockID != "" {
			relationships = append(relationships,
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "next_sibling",
					SourceElementID:  prevBlockID,
					TargetElementID:  blockID,
				},
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "previous_sibling",
					SourceElementID:  blockID,
					TargetElementID:  prevBlockID,
				},
			)
		}
		prevBlockID = blockID

		// Extract links if enabled
		if p.ExtractLinks {
			links := p.extractLinks(block.Text, blockID)
			// Note: Links would be added to result.Links in the caller
			// For now we skip them as we focus on table extraction parity
			_ = links
		}
	}

	// Extract tables if enabled
	if p.ExtractTables {
		tableElements, tableRels, pos := p.extractTables(doc, pageNum, pageID, elementPosition)
		elements = append(elements, tableElements...)
		relationships = append(relationships, tableRels...)
		*elementPosition = pos
	}

	return elements, relationships, *elementPosition
}

// TextBlock represents a block of text with metadata
type TextBlock struct {
	Text        string
	IsLargeFont bool
	BBox        [4]float64 // x0, y0, x1, y1
}

// extractTextBlocks extracts text blocks with positioning from a PDF page
func (p *PDFParser) extractTextBlocks(doc *fitz.Document, pageNum int, pageText string) []TextBlock {
	blocks := []TextBlock{}

	// Split text into paragraphs (simple approach)
	paragraphs := p.splitIntoParagraphs(pageText)

	// For each paragraph, create a text block
	// Note: go-fitz doesn't provide detailed font info easily, so we use simpler heuristics
	for _, para := range paragraphs {
		if strings.TrimSpace(para) == "" {
			continue
		}

		// Simple heuristic: if paragraph is short and ends with no punctuation, likely a header
		isLargeFont := len(para) < 100 && !strings.HasSuffix(strings.TrimSpace(para), ".")

		blocks = append(blocks, TextBlock{
			Text:        para,
			IsLargeFont: isLargeFont,
		})
	}

	return blocks
}

// splitIntoParagraphs splits text into paragraphs
func (p *PDFParser) splitIntoParagraphs(text string) []string {
	// Split by double newlines
	parts := strings.Split(text, "\n\n")
	var paragraphs []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Also handle single newlines as paragraph breaks if they look like separate thoughts
			lines := strings.Split(trimmed, "\n")
			currentPara := ""

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Check if line looks like a new paragraph (starts with capital, previous ended with period)
				if currentPara != "" &&
					strings.HasSuffix(currentPara, ".") &&
					len(line) > 0 &&
					strings.ToUpper(line[:1]) == line[:1] {
					// Save current paragraph and start new one
					paragraphs = append(paragraphs, currentPara)
					currentPara = line
				} else {
					// Continue building current paragraph
					if currentPara != "" {
						currentPara += " "
					}
					currentPara += line
				}
			}

			if currentPara != "" {
				paragraphs = append(paragraphs, currentPara)
			}
		}
	}

	return paragraphs
}

// Table represents a detected table
type Table struct {
	Rows  int
	Cols  int
	Cells map[int]map[int]string // [row][col]text
	BBox  [4]float64             // x0, y0, x1, y1
}

// extractTables extracts tables from a PDF page using heuristic detection
func (p *PDFParser) extractTables(doc *fitz.Document, pageNum int, pageID string, elementPosition *int) ([]Element, []Relationship, int) {
	elements := []Element{}
	relationships := []Relationship{}

	// Detect tables using heuristics
	tables := p.detectTablesHeuristic(doc, pageNum)

	for tableIdx, table := range tables {
		tableID := generateID(fmt.Sprintf("table_%d_", tableIdx))

		// Create table content preview
		contentPreview := fmt.Sprintf("Table with %d rows and %d columns", table.Rows, table.Cols)

		// Create table element
		tablePageNum := pageNum + 1
		tableElement := Element{
			ElementID:      tableID,
			ElementType:    "table",
			ContentPreview: contentPreview,
			ParentID:       pageID,
			Position:       *elementPosition,
			Depth:          3,
			ContentLocation: map[string]interface{}{
				"page_number": tablePageNum,
				"table_index": tableIdx,
			},
			// UDML Phase 1: Populate promoted fields
			PageNumber:      &tablePageNum,
			ElementCategory: GetElementCategory("table"),
		}
		elements = append(elements, tableElement)
		*elementPosition++

		// Add page->table relationships
		relationships = append(relationships,
			Relationship{
				RelationshipID:   generateID("rel_"),
				RelationshipType: "contains",
				SourceElementID:  pageID,
				TargetElementID:  tableID,
			},
			Relationship{
				RelationshipID:   generateID("rel_"),
				RelationshipType: "contained_by",
				SourceElementID:  tableID,
				TargetElementID:  pageID,
			},
		)

		// Add header row if exists
		if table.Rows > 0 {
			headerRowID := generateID(fmt.Sprintf("table_header_row_%d_", tableIdx))
			headerTexts := []string{}

			// Get sorted column keys
			colKeys := []int{}
			if row0, exists := table.Cells[0]; exists {
				for col := range row0 {
					colKeys = append(colKeys, col)
				}
				sort.Ints(colKeys)

				for _, col := range colKeys {
					cellText := table.Cells[0][col]
					headerTexts = append(headerTexts, strings.TrimSpace(cellText))
				}
			}

			headerText := strings.Join(headerTexts, "\t")

			// Create header row element
			headerRowPageNum := pageNum + 1
			headerRowElement := Element{
				ElementID:      headerRowID,
				ElementType:    "table_header_row",
				ContentPreview: p.truncateText(headerText, p.MaxContentPreview),
				ParentID:       tableID,
				Position:       *elementPosition,
				Depth:          4,
				ContentLocation: map[string]interface{}{
					"page_number": headerRowPageNum,
					"table_index": tableIdx,
					"row":         0,
				},
				// UDML Phase 1: Populate promoted fields
				PageNumber:      &headerRowPageNum,
				ElementCategory: GetElementCategory("table_header_row"),
			}
			elements = append(elements, headerRowElement)
			*elementPosition++

			// Add table->header_row relationships
			relationships = append(relationships,
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "contains_table_row",
					SourceElementID:  tableID,
					TargetElementID:  headerRowID,
				},
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "contained_by",
					SourceElementID:  headerRowID,
					TargetElementID:  tableID,
				},
			)

			// Add header cells
			for colIdx, colKey := range colKeys {
				headerCellID := generateID(fmt.Sprintf("table_header_%d_%d_", tableIdx, colIdx))
				cellText := table.Cells[0][colKey]

				headerMetadata := map[string]interface{}{
					"page_number": pageNum + 1,
					"table_index": tableIdx,
					"row":         0,
					"col":         colKey,
				}

				// Extract temporal metadata from header cell text
				ProcessTemporalContent(cellText, headerMetadata)

				headerCellElement := Element{
					ElementID:       headerCellID,
					ElementType:     "table_header",
					Content:         cellText,
					ContentPreview:  p.truncateText(cellText, p.MaxContentPreview),
					ParentID:        headerRowID,
					Position:        *elementPosition,
					Depth:           5,
					ContentLocation: headerMetadata,
					Metadata:        headerMetadata,
				}

				// UDML Phase 1: Populate promoted fields
				headerCellPageNum := pageNum + 1
				headerCellRowIdx := 0
				headerCellColIdx := colKey
				headerCellElement.PageNumber = &headerCellPageNum
				headerCellElement.RowIndex = &headerCellRowIdx
				headerCellElement.ColumnIndex = &headerCellColIdx

				// TemporalType from temporal metadata
				if temporalType, ok := headerMetadata["temporal_type"].(string); ok && temporalType != "" {
					headerCellElement.TemporalType = &temporalType
				}

				// ElementCategory
				headerCellElement.ElementCategory = GetElementCategory("table_header")

				elements = append(elements, headerCellElement)
				*elementPosition++

				// Add header_row->header_cell relationships
				relationships = append(relationships,
					Relationship{
						RelationshipID:   generateID("rel_"),
						RelationshipType: "contains_table_header",
						SourceElementID:  headerRowID,
						TargetElementID:  headerCellID,
					},
					Relationship{
						RelationshipID:   generateID("rel_"),
						RelationshipType: "contained_by",
						SourceElementID:  headerCellID,
						TargetElementID:  headerRowID,
					},
				)
			}
		}

		// Add data rows (skip row 0 which is header)
		rowKeys := []int{}
		for row := range table.Cells {
			if row > 0 {
				rowKeys = append(rowKeys, row)
			}
		}
		sort.Ints(rowKeys)

		for _, row := range rowKeys {
			rowID := generateID(fmt.Sprintf("table_row_%d_%d_", tableIdx, row))

			// Get cells for this row
			colKeys := []int{}
			for col := range table.Cells[row] {
				colKeys = append(colKeys, col)
			}
			sort.Ints(colKeys)

			rowTexts := []string{}
			for _, col := range colKeys {
				cellText := table.Cells[row][col]
				rowTexts = append(rowTexts, strings.TrimSpace(cellText))
			}
			rowText := strings.Join(rowTexts, "\t")

			// Create row element
			rowPageNum := pageNum + 1
			rowElement := Element{
				ElementID:      rowID,
				ElementType:    "table_row",
				ContentPreview: p.truncateText(rowText, p.MaxContentPreview),
				ParentID:       tableID,
				Position:       *elementPosition,
				Depth:          4,
				ContentLocation: map[string]interface{}{
					"page_number": rowPageNum,
					"table_index": tableIdx,
					"row":         row,
				},
				// UDML Phase 1: Populate promoted fields
				PageNumber:      &rowPageNum,
				ElementCategory: GetElementCategory("table_row"),
			}
			elements = append(elements, rowElement)
			*elementPosition++

			// Add table->row relationships
			relationships = append(relationships,
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "contains_table_row",
					SourceElementID:  tableID,
					TargetElementID:  rowID,
				},
				Relationship{
					RelationshipID:   generateID("rel_"),
					RelationshipType: "contained_by",
					SourceElementID:  rowID,
					TargetElementID:  tableID,
				},
			)

			// Add cells
			for colIdx, col := range colKeys {
				cellID := generateID(fmt.Sprintf("table_cell_%d_%d_%d_", tableIdx, row, colIdx))
				cellText := table.Cells[row][col]

				cellMetadata := map[string]interface{}{
					"page_number": pageNum + 1,
					"table_index": tableIdx,
					"row":         row,
					"col":         col,
				}

				// Extract temporal metadata from cell text
				ProcessTemporalContent(cellText, cellMetadata)

				cellElement := Element{
					ElementID:       cellID,
					ElementType:     "table_cell",
					Content:         cellText,
					ContentPreview:  p.truncateText(cellText, p.MaxContentPreview),
					ParentID:        rowID,
					Position:        *elementPosition,
					Depth:           5,
					ContentLocation: cellMetadata,
					Metadata:        cellMetadata,
				}

				// UDML Phase 1: Populate promoted fields
				cellPageNum := pageNum + 1
				cellRowIdx := row
				cellColIdx := col
				cellElement.PageNumber = &cellPageNum
				cellElement.RowIndex = &cellRowIdx
				cellElement.ColumnIndex = &cellColIdx

				// TemporalType from temporal metadata
				if temporalType, ok := cellMetadata["temporal_type"].(string); ok && temporalType != "" {
					cellElement.TemporalType = &temporalType
				}

				// ElementCategory
				cellElement.ElementCategory = GetElementCategory("table_cell")

				elements = append(elements, cellElement)
				*elementPosition++

				// Add row->cell relationships
				relationships = append(relationships,
					Relationship{
						RelationshipID:   generateID("rel_"),
						RelationshipType: "contains_table_cell",
						SourceElementID:  rowID,
						TargetElementID:  cellID,
					},
					Relationship{
						RelationshipID:   generateID("rel_"),
						RelationshipType: "contained_by",
						SourceElementID:  cellID,
						TargetElementID:  rowID,
					},
				)
			}
		}
	}

	return elements, relationships, *elementPosition
}

// detectTablesHeuristic detects tables using positioned text from HTML output
func (p *PDFParser) detectTablesHeuristic(doc *fitz.Document, pageNum int) []Table {
	tables := []Table{}

	// Get page HTML which has positioning information
	pageHTML, err := doc.HTML(pageNum, false)
	if err != nil || pageHTML == "" {
		return tables
	}

	// Extract positioned text blocks from HTML
	textBlocks := p.extractPositionedTextFromHTML(pageHTML)

	// Detect tables based on position alignment
	htmlTables := p.detectTablesFromPositionedText(textBlocks)
	tables = append(tables, htmlTables...)

	return tables
}

// TextWithPosition represents text with its position on the page
type TextWithPosition struct {
	Text   string
	Top    float64
	Left   float64
	Width  float64
	Height float64
}

// extractPositionedTextFromHTML extracts text with positioning from HTML
func (p *PDFParser) extractPositionedTextFromHTML(htmlContent string) []TextWithPosition {
	var blocks []TextWithPosition

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return blocks
	}

	// Find all <p> elements with style attributes
	var findParagraphs func(*html.Node)
	findParagraphs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			// Extract style attributes
			var top, left float64
			for _, attr := range n.Attr {
				if attr.Key == "style" {
					top, left = p.parseStylePosition(attr.Val)
					break
				}
			}

			// Extract text
			text := strings.TrimSpace(p.extractTextFromNode(n))
			if text != "" {
				blocks = append(blocks, TextWithPosition{
					Text: text,
					Top:  top,
					Left: left,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findParagraphs(c)
		}
	}
	findParagraphs(doc)

	return blocks
}

// parseStylePosition extracts top and left from style attribute
func (p *PDFParser) parseStylePosition(style string) (float64, float64) {
	var top, left float64

	// Parse style="top:518.1pt;left:532.1pt;..."
	parts := strings.Split(style, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "top:") {
			// Extract just the number, ignoring the 'pt' suffix
			val := strings.TrimPrefix(part, "top:")
			val = strings.TrimSuffix(val, "pt")
			fmt.Sscanf(val, "%f", &top)
		} else if strings.HasPrefix(part, "left:") {
			// Extract just the number, ignoring the 'pt' suffix
			val := strings.TrimPrefix(part, "left:")
			val = strings.TrimSuffix(val, "pt")
			fmt.Sscanf(val, "%f", &left)
		}
	}

	return top, left
}

// detectTablesFromPositionedText detects tables from positioned text blocks
func (p *PDFParser) detectTablesFromPositionedText(blocks []TextWithPosition) []Table {
	var tables []Table

	if len(blocks) < 4 {
		return tables
	}

	// Sort by top position to group into rows
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Top < blocks[j].Top
	})

	// Group blocks into rows based on vertical proximity (within 5pt)
	rowTolerance := 5.0
	var rows [][]TextWithPosition
	var currentRow []TextWithPosition
	var lastTop float64 = -1

	for _, block := range blocks {
		if lastTop < 0 {
			currentRow = append(currentRow, block)
			lastTop = block.Top
		} else {
			diff := block.Top - lastTop
			if diff <= rowTolerance {
				// Same row - within tolerance
				currentRow = append(currentRow, block)
			} else {
				// New row - save current row
				if len(currentRow) > 0 {
					// Sort row by left position
					sort.Slice(currentRow, func(i, j int) bool {
						return currentRow[i].Left < currentRow[j].Left
					})
					rows = append(rows, currentRow)
				}
				currentRow = []TextWithPosition{block}
				lastTop = block.Top
			}
		}
	}
	if len(currentRow) > 0 {
		sort.Slice(currentRow, func(i, j int) bool {
			return currentRow[i].Left < currentRow[j].Left
		})
		rows = append(rows, currentRow)
	}

	// Detect tables: consecutive rows with similar column alignment
	minRows := 2
	minCols := 2

	i := 0
	for i < len(rows) {
		// Try to build a table starting from row i
		if len(rows[i]) < minCols {
			i++
			continue
		}

		// Get column positions from first row
		colPositions := make([]float64, len(rows[i]))
		for j, block := range rows[i] {
			colPositions[j] = block.Left
		}

		// Find consecutive rows that align with these columns
		tableRows := [][]TextWithPosition{rows[i]}
		j := i + 1

		for j < len(rows) {
			if len(rows[j]) < minCols {
				break
			}

			// Check if this row aligns with column positions
			if p.rowAlignsWithColumns(rows[j], colPositions, 20.0) {
				tableRows = append(tableRows, rows[j])
				j++
			} else {
				break
			}
		}

		// If we found enough aligned rows, create a table
		if len(tableRows) >= minRows {
			table := p.createTableFromRows(tableRows, colPositions)
			if table != nil {
				tables = append(tables, *table)
			}
			i = j
		} else {
			i++
		}
	}

	return tables
}

// rowAlignsWithColumns checks if a row's cells align with column positions
func (p *PDFParser) rowAlignsWithColumns(row []TextWithPosition, colPositions []float64, tolerance float64) bool {
	if len(row) != len(colPositions) {
		return false
	}

	for i, block := range row {
		if i >= len(colPositions) {
			return false
		}
		if (block.Left-colPositions[i]) > tolerance && (block.Left-colPositions[i]) < -tolerance {
			return false
		}
	}

	return true
}

// createTableFromRows creates a Table structure from aligned rows
func (p *PDFParser) createTableFromRows(rows [][]TextWithPosition, colPositions []float64) *Table {
	table := &Table{
		Cells: make(map[int]map[int]string),
		Rows:  len(rows),
		Cols:  len(colPositions),
	}

	for rowIdx, row := range rows {
		table.Cells[rowIdx] = make(map[int]string)
		for colIdx, block := range row {
			if colIdx < len(colPositions) {
				table.Cells[rowIdx][colIdx] = block.Text
			}
		}
	}

	return table
}

// extractTablesFromHTML parses HTML and extracts table structures
func (p *PDFParser) extractTablesFromHTML(htmlContent string) []Table {
	tables := []Table{}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return tables
	}

	// Find all <table> elements
	var findTables func(*html.Node)
	findTables = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			table := p.parseHTMLTable(n)
			if table != nil && table.Rows >= 2 && table.Cols >= 2 {
				tables = append(tables, *table)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTables(c)
		}
	}
	findTables(doc)

	return tables
}

// parseHTMLTable parses an HTML table node into a Table structure
func (p *PDFParser) parseHTMLTable(tableNode *html.Node) *Table {
	table := &Table{
		Cells: make(map[int]map[int]string),
	}

	rowIdx := 0

	// Process table rows
	var processRows func(*html.Node)
	processRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			colIdx := 0
			table.Cells[rowIdx] = make(map[int]string)

			// Process cells in this row
			var processCells func(*html.Node)
			processCells = func(c *html.Node) {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cellText := p.extractTextFromNode(c)
					table.Cells[rowIdx][colIdx] = strings.TrimSpace(cellText)
					colIdx++
				}
				for child := c.FirstChild; child != nil; child = child.NextSibling {
					processCells(child)
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				processCells(c)
			}

			if colIdx > table.Cols {
				table.Cols = colIdx
			}
			rowIdx++
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processRows(c)
		}
	}

	processRows(tableNode)
	table.Rows = rowIdx

	if table.Rows == 0 || table.Cols == 0 {
		return nil
	}

	return table
}

// extractTextFromNode extracts text content from an HTML node
func (p *PDFParser) extractTextFromNode(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(p.extractTextFromNode(c))
	}
	return text.String()
}

// parseTableLines parses a sequence of lines into a Table
func (p *PDFParser) parseTableLines(lines []string) *Table {
	if len(lines) < 2 {
		return nil
	}

	table := &Table{
		Cells: make(map[int]map[int]string),
	}

	// Parse each line into cells
	for rowIdx, line := range lines {
		// Split by tab or multiple spaces
		var cells []string
		if strings.Contains(line, "\t") {
			cells = strings.Split(line, "\t")
		} else {
			cells = regexp.MustCompile(`\s{3,}`).Split(line, -1)
		}

		// Store cells
		table.Cells[rowIdx] = make(map[int]string)
		for colIdx, cell := range cells {
			table.Cells[rowIdx][colIdx] = strings.TrimSpace(cell)
		}

		if len(cells) > table.Cols {
			table.Cols = len(cells)
		}
	}

	table.Rows = len(lines)

	return table
}

// formatTableText formats table cells into text
func (p *PDFParser) formatTableText(table Table) string {
	var lines []string

	// Get sorted row keys
	rowKeys := []int{}
	for row := range table.Cells {
		rowKeys = append(rowKeys, row)
	}
	sort.Ints(rowKeys)

	for _, row := range rowKeys {
		// Get sorted column keys
		colKeys := []int{}
		for col := range table.Cells[row] {
			colKeys = append(colKeys, col)
		}
		sort.Ints(colKeys)

		rowTexts := []string{}
		for _, col := range colKeys {
			cellText := table.Cells[row][col]
			if cellText == "" {
				cellText = ""
			}
			rowTexts = append(rowTexts, cellText)
		}
		lines = append(lines, strings.Join(rowTexts, "\t"))
	}

	return strings.Join(lines, "\n")
}

// extractLinks extracts URLs and emails from text
func (p *PDFParser) extractLinks(text string, sourceID string) []Link {
	var links []Link

	// URL pattern
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlPattern.FindAllString(text, -1)
	for _, url := range urls {
		links = append(links, Link{
			LinkID:          generateID("link_"),
			SourceElementID: sourceID,
			LinkType:        "url",
			LinkTarget:      url,
		})
	}

	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailPattern.FindAllString(text, -1)
	for _, email := range emails {
		links = append(links, Link{
			LinkID:          generateID("link_"),
			SourceElementID: sourceID,
			LinkType:        "email",
			LinkTarget:      email,
		})
	}

	return links
}

// truncateText truncates text to specified length
func (p *PDFParser) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
