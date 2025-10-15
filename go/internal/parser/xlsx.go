package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/temporal"
	"github.com/xuri/excelize/v2"
)

// XLSXParser handles Excel document parsing with table and header detection
type XLSXParser struct {
	MaxContentPreview int
	MaxRows           int
	MaxCols           int
	DetectTables      bool
	MinTableRows      int
	MinTableCols      int
	ExtractComments   bool
	ExtractFormulas   bool
	ExtractLinks      bool
	ExtractDates      bool
}

// NewXLSXParser creates a new XLSX parser
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{
		MaxContentPreview: 100,
		MaxRows:           1000,
		MaxCols:           100,
		DetectTables:      true,
		MinTableRows:      2,
		MinTableCols:      2,
		ExtractComments:   true,
		ExtractFormulas:   true,
		ExtractLinks:      true,
		ExtractDates:      true,
	}
}

// GetName returns the parser name
func (p *XLSXParser) GetName() string {
	return "xlsx"
}

// GetSupportedFormats returns supported file formats
func (p *XLSXParser) GetSupportedFormats() []string {
	return []string{".xlsx", ".xls", "xlsx"}
}

// Parse implements the Parser interface for Excel documents
func (p *XLSXParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// XLSX parser can handle both file paths and byte content
	return p.ParseLegacy(req.ID, req.Content)
}

// SupportsStreaming returns whether the parser supports streaming
func (p *XLSXParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *XLSXParser) Close() error {
	// XLSX parser has no persistent resources to clean up
	return nil
}

// CellData represents information about a cell
type CellData struct {
	Value   string
	IsEmpty bool
	IsBold  bool
	HasFill bool
	Formula string
	Address string
	Row     int
	Col     int
}

// TableRegion represents a detected table area
type TableRegion struct {
	MinRow     int
	MinCol     int
	MaxRow     int
	MaxCol     int
	HasHeader  bool
	Confidence string
}

// ParseLegacy parses Excel content (legacy interface)
// Deprecated: Use Parse(ctx, ParseRequest) instead
func (p *XLSXParser) ParseLegacy(docID string, content interface{}) (*ParseResult, error) {
	var file *excelize.File
	var err error

	// Handle different input types
	switch v := content.(type) {
	case string:
		// File path
		file, err = excelize.OpenFile(v)
		if err != nil {
			return nil, fmt.Errorf("failed to open Excel file: %v", err)
		}
	case []byte:
		// Raw bytes
		file, err = excelize.OpenReader(strings.NewReader(string(v)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse Excel content: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	defer file.Close()

	// Initialize result
	result := &ParseResult{
		Document: Document{
			ID:      docID,
			DocType: "xlsx",
		},
		Elements: []Element{},
	}

	// Create root element
	rootID := generateID("root_")
	rootElement := Element{
		ElementID:       rootID,
		ElementType:     "root",
		ContentPreview:  "Excel Document",
		Position:        0,
		Depth:           0,
		ElementCategory: GetElementCategory("root"),
	}
	result.Elements = append(result.Elements, rootElement)

	// Create workbook element
	workbookID := generateID("workbook_")
	sheetNames := file.GetSheetList()
	workbookElement := Element{
		ElementID:       workbookID,
		ElementType:     "workbook",
		ContentPreview:  fmt.Sprintf("Excel workbook with %d sheets", len(sheetNames)),
		ParentID:        rootID,
		Position:        1,
		Depth:           1,
		ElementCategory: GetElementCategory("workbook"),
		Metadata: map[string]interface{}{
			"sheet_count": len(sheetNames),
			"sheet_names": sheetNames,
		},
	}
	result.Elements = append(result.Elements, workbookElement)

	elementPosition := 2

	// Process each sheet
	for sheetIdx, sheetName := range sheetNames {
		// Process the sheet
		sheetElements, newPosition := p.processSheet(file, sheetName, docID, workbookID, sheetIdx, elementPosition)
		result.Elements = append(result.Elements, sheetElements...)
		elementPosition = newPosition
	}

	return result, nil
}

// processSheet processes a single worksheet
func (p *XLSXParser) processSheet(file *excelize.File, sheetName string, docID, parentID string, sheetIdx, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Create sheet element
	sheetID := generateID("sheet_")

	// Get sheet dimensions
	rows, err := file.GetRows(sheetName)
	if err != nil {
		rows = [][]string{}
	}

	maxRow := len(rows)
	if maxRow > p.MaxRows {
		maxRow = p.MaxRows
	}

	maxCol := 0
	for _, row := range rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}
	if maxCol > p.MaxCols {
		maxCol = p.MaxCols
	}

	// Create sheet preview
	var preview string
	if maxRow > 0 && maxCol > 0 {
		preview = fmt.Sprintf("Sheet '%s' with %d rows and %d columns", sheetName, maxRow, maxCol)
	} else {
		preview = fmt.Sprintf("Empty sheet '%s'", sheetName)
	}

	// Create sheet element
	sheetPageNum := sheetIdx + 1 // Sheets map to pages in XLSX
	sheetElement := Element{
		ElementID:      sheetID,
		ElementType:    "sheet",
		ContentPreview: preview,
		ParentID:       parentID,
		Position:       position,
		Depth:          2,
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
		},
		Metadata: map[string]interface{}{
			"title":      sheetName,
			"max_row":    maxRow,
			"max_column": maxCol,
		},
		// UDML Phase 1: Populate promoted fields
		PageNumber:      &sheetPageNum,
		ElementCategory: GetElementCategory("sheet"),
	}
	elements = append(elements, sheetElement)
	position++

	// Process sheet content if not empty
	if maxRow > 0 && maxCol > 0 {
		// Detect tables if enabled
		if p.DetectTables && maxRow >= p.MinTableRows && maxCol >= p.MinTableCols {
			tableElements, newPos := p.detectAndProcessTables(file, sheetName, docID, sheetID, maxRow, maxCol, position)
			elements = append(elements, tableElements...)
			position = newPos
		} else {
			// Process as regular rows/cells
			rowElements, newPos := p.processRows(file, sheetName, docID, sheetID, maxRow, maxCol, position)
			elements = append(elements, rowElements...)
			position = newPos
		}
	}

	// Extract comments if enabled
	if p.ExtractComments {
		commentElements, newPos := p.processComments(file, sheetName, docID, sheetID, position)
		elements = append(elements, commentElements...)
		position = newPos
	}

	return elements, position
}

// processComments extracts comments from a sheet
func (p *XLSXParser) processComments(file *excelize.File, sheetName, docID, sheetID string, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Get comments for the sheet
	comments, err := file.GetComments(sheetName)
	if err != nil || len(comments) == 0 {
		return elements, position
	}

	// Create comments container element
	commentsID := generateID("comments_")
	commentsElement := Element{
		ElementID:       commentsID,
		ElementType:     "comments",
		ContentPreview:  fmt.Sprintf("Comments in sheet '%s'", sheetName),
		ParentID:        sheetID,
		Position:        position,
		Depth:           3,
		ElementCategory: GetElementCategory("comments"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
			"type":       "comments",
		},
		Metadata: map[string]interface{}{
			"sheet": sheetName,
			"count": len(comments),
		},
	}
	elements = append(elements, commentsElement)
	position++

	// Process individual comments
	for commentIdx, comment := range comments {
		commentID := generateID(fmt.Sprintf("comment_%d_", commentIdx))

		// Extract comment text and author
		text := comment.Text
		author := comment.Author

		// Create content preview
		contentPreview := fmt.Sprintf("Comment by %s: %s", author, text)
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:97] + "..."
		}

		commentElement := Element{
			ElementID:       commentID,
			ElementType:     "comment",
			ContentPreview:  contentPreview,
			ParentID:        commentsID,
			Position:        position,
			Depth:           4,
			ElementCategory: GetElementCategory("comment"),
			ContentLocation: map[string]interface{}{
				"sheet_name": sheetName,
				"cell":       comment.Cell,
				"type":       "comment",
			},
			Metadata: map[string]interface{}{
				"cell":   comment.Cell,
				"author": author,
				"text":   text,
				"sheet":  sheetName,
			},
		}
		elements = append(elements, commentElement)
		position++
	}

	return elements, position
}

// detectAndProcessTables detects tables within the sheet using heuristics
func (p *XLSXParser) detectAndProcessTables(file *excelize.File, sheetName, docID, sheetID string, maxRow, maxCol, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Get cell data for analysis
	cellData := p.getCellData(file, sheetName, maxRow, maxCol)

	// Detect table regions first
	tableRegions := p.detectTableRegions(cellData, maxRow, maxCol)

	// First pass: Raw row-by-row processing (like Python parser)
	// But only for cells NOT covered by structured tables
	rawElements, newPosition := p.processRawRows(file, sheetName, docID, sheetID, maxRow, maxCol, position, tableRegions)
	elements = append(elements, rawElements...)
	position = newPosition

	if len(tableRegions) == 0 {
		// No structured tables detected, skip table processing
		return elements, position
	}

	// Create data tables container
	dataTablesID := generateID("data_tables_")
	dataTablesElement := Element{
		ElementID:       dataTablesID,
		ElementType:     "data_tables",
		ContentPreview:  fmt.Sprintf("Data tables in sheet '%s'", sheetName),
		ParentID:        sheetID,
		Position:        position,
		Depth:           3,
		ElementCategory: GetElementCategory("data_tables"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
		},
		Metadata: map[string]interface{}{
			"sheet": sheetName,
			"count": len(tableRegions),
		},
	}
	elements = append(elements, dataTablesElement)
	position++

	// Process each detected table
	for tableIdx, region := range tableRegions {
		tableElements, newPos := p.processTableRegion(file, sheetName, docID, dataTablesID, region, tableIdx, position)
		elements = append(elements, tableElements...)
		position = newPos
	}

	// Process merged cells (if any)
	if p.ExtractFormulas { // Use existing flag to control merged cell extraction
		mergedElements, newPos := p.processMergedCells(file, sheetName, docID, sheetID, position)
		elements = append(elements, mergedElements...)
		position = newPos
	}

	return elements, position
}

// getCellData extracts cell information for analysis
func (p *XLSXParser) getCellData(file *excelize.File, sheetName string, maxRow, maxCol int) [][]CellData {
	cellData := make([][]CellData, maxRow)

	for row := 1; row <= maxRow; row++ {
		cellData[row-1] = make([]CellData, maxCol)

		for col := 1; col <= maxCol; col++ {
			cellAddr, _ := excelize.CoordinatesToCellName(col, row)
			cellValue, _ := file.GetCellValue(sheetName, cellAddr)

			// Get cell style for formatting detection
			styleID, _ := file.GetCellStyle(sheetName, cellAddr)
			style, _ := file.GetStyle(styleID)

			// Check if cell has bold formatting
			isBold := false
			if style != nil && style.Font != nil {
				isBold = style.Font.Bold
			}

			// Check if cell has fill
			hasFill := false
			if style != nil && style.Fill.Type != "" {
				hasFill = true
			}

			// Get formula if extracting formulas
			var formula string
			if p.ExtractFormulas {
				formula, _ = file.GetCellFormula(sheetName, cellAddr)
			}

			cellData[row-1][col-1] = CellData{
				Value:   cellValue,
				IsEmpty: cellValue == "",
				IsBold:  isBold,
				HasFill: hasFill,
				Formula: formula,
				Address: cellAddr,
				Row:     row,
				Col:     col,
			}
		}
	}

	return cellData
}

// detectTableRegions uses heuristics to detect table areas
func (p *XLSXParser) detectTableRegions(cellData [][]CellData, maxRow, maxCol int) []TableRegion {
	var regions []TableRegion

	// Simple table detection heuristic:
	// 1. Look for header row with special formatting (bold/fill)
	// 2. Check for consistent data below header
	// 3. Stop at empty rows or formatting changes

	if len(cellData) < 2 {
		return regions
	}

	// Check first row for header indicators
	firstRow := cellData[0]
	headerIndicators := 0
	maxColWithData := 0

	for colIdx, cell := range firstRow {
		if !cell.IsEmpty {
			maxColWithData = colIdx + 1
		}
		if cell.IsBold || cell.HasFill {
			headerIndicators++
		}
	}

	// If more than half of non-empty cells in first row have header formatting
	nonEmptyCells := 0
	for _, cell := range firstRow {
		if !cell.IsEmpty {
			nonEmptyCells++
		}
	}

	isLikelyHeader := false
	if nonEmptyCells > 0 && headerIndicators > nonEmptyCells/2 {
		isLikelyHeader = true
	}

	// Find table boundaries
	if maxColWithData >= p.MinTableCols {
		// Determine table height by looking for empty rows
		tableMaxRow := len(cellData)
		for rowIdx := 1; rowIdx < len(cellData); rowIdx++ {
			emptyCount := 0
			for colIdx := 0; colIdx < maxColWithData; colIdx++ {
				if cellData[rowIdx][colIdx].IsEmpty {
					emptyCount++
				}
			}
			// If more than half the cells are empty, consider this the end
			if emptyCount > maxColWithData/2 {
				tableMaxRow = rowIdx
				break
			}
		}

		// If we found a viable table region
		if tableMaxRow >= p.MinTableRows && maxColWithData >= p.MinTableCols {
			confidence := "medium"
			if isLikelyHeader {
				confidence = "high"
			}

			regions = append(regions, TableRegion{
				MinRow:     1,
				MinCol:     1,
				MaxRow:     tableMaxRow,
				MaxCol:     maxColWithData,
				HasHeader:  isLikelyHeader,
				Confidence: confidence,
			})
		}
	}

	return regions
}

// processTableRegion creates elements for a detected table region
func (p *XLSXParser) processTableRegion(file *excelize.File, sheetName, docID, parentID string, region TableRegion, tableIdx, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Create data table element
	tableID := generateID(fmt.Sprintf("data_table_%d_", tableIdx+1))

	// Generate table preview
	preview := fmt.Sprintf("Data table %d (%dx%d)", tableIdx+1, region.MaxRow-region.MinRow+1, region.MaxCol-region.MinCol+1)

	if region.HasHeader {
		// Add header info to preview
		headerValues := make([]string, 0, 3)
		for col := region.MinCol; col <= region.MaxCol && len(headerValues) < 3; col++ {
			cellAddr, _ := excelize.CoordinatesToCellName(col, region.MinRow)
			cellValue, _ := file.GetCellValue(sheetName, cellAddr)
			if cellValue != "" {
				headerValues = append(headerValues, cellValue)
			}
		}
		if len(headerValues) > 0 {
			preview += fmt.Sprintf(" with headers: %s", strings.Join(headerValues, ", "))
			if region.MaxCol-region.MinCol+1 > 3 {
				preview += ", ..."
			}
		}
	}

	// Create table element
	tableElement := Element{
		ElementID:       tableID,
		ElementType:     "data_table",
		ContentPreview:  preview,
		ParentID:        parentID,
		Position:        position,
		Depth:           4,
		ElementCategory: GetElementCategory("data_table"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
			"range":      fmt.Sprintf("%s%d:%s%d", columnLetter(region.MinCol), region.MinRow, columnLetter(region.MaxCol), region.MaxRow),
		},
		Metadata: map[string]interface{}{
			"min_row":              region.MinRow,
			"min_col":              region.MinCol,
			"max_row":              region.MaxRow,
			"max_col":              region.MaxCol,
			"row_count":            region.MaxRow - region.MinRow + 1,
			"column_count":         region.MaxCol - region.MinCol + 1,
			"has_header":           region.HasHeader,
			"detection_confidence": region.Confidence,
			"sheet":                sheetName,
		},
	}
	elements = append(elements, tableElement)
	position++

	// Process header row if present
	if region.HasHeader {
		headerElements, newPos := p.processHeaderRow(file, sheetName, docID, tableID, region, position)
		elements = append(elements, headerElements...)
		position = newPos
	}

	// Process data rows
	startRow := region.MinRow
	if region.HasHeader {
		startRow++ // Skip header row
	}

	for row := startRow; row <= region.MaxRow; row++ {
		rowElements, newPos := p.processTableRow(file, sheetName, docID, tableID, region, row, position)
		elements = append(elements, rowElements...)
		position = newPos
	}

	return elements, position
}

// processHeaderRow processes the header row of a table
func (p *XLSXParser) processHeaderRow(file *excelize.File, sheetName, docID, tableID string, region TableRegion, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Create header row element
	headerRowID := generateID("table_header_row_")
	headerValues := make([]string, 0, region.MaxCol-region.MinCol+1)

	for col := region.MinCol; col <= region.MaxCol; col++ {
		cellAddr, _ := excelize.CoordinatesToCellName(col, region.MinRow)
		cellValue, _ := file.GetCellValue(sheetName, cellAddr)
		headerValues = append(headerValues, cellValue)
	}

	headerText := strings.Join(headerValues, "\t")
	headerPreview := headerText
	if len(headerPreview) > 100 {
		headerPreview = headerPreview[:97] + "..."
	}

	headerRowElement := Element{
		ElementID:       headerRowID,
		ElementType:     "table_header_row",
		ContentPreview:  headerPreview,
		ParentID:        tableID,
		Position:        position,
		Depth:           5,
		ElementCategory: GetElementCategory("table_header_row"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
			"row":        region.MinRow,
		},
		Metadata: map[string]interface{}{
			"row":    region.MinRow,
			"values": headerValues,
			"sheet":  sheetName,
		},
	}
	elements = append(elements, headerRowElement)
	position++

	// Process individual header cells
	for col := region.MinCol; col <= region.MaxCol; col++ {
		cellAddr, _ := excelize.CoordinatesToCellName(col, region.MinRow)
		cellValue, _ := file.GetCellValue(sheetName, cellAddr)

		if cellValue != "" {
			headerCellID := generateID("table_header_cell_")
			cellPreview := cellValue
			if len(cellPreview) > 100 {
				cellPreview = cellPreview[:97] + "..."
			}

			headerCellElement := Element{
				ElementID:      headerCellID,
				ElementType:    "table_header",
				ContentPreview: cellPreview,
				ParentID:       headerRowID,
				Position:       position,
				Depth:          6,
				ContentLocation: map[string]interface{}{
					"sheet_name": sheetName,
					"cell":       cellAddr,
				},
				Metadata: map[string]interface{}{
					"row":    region.MinRow,
					"column": col,
					"value":  cellValue,
					"sheet":  sheetName,
				},
			}

			// UDML Phase 1: Populate promoted fields
			rowIdx := region.MinRow
			colIdx := col
			headerCellElement.RowIndex = &rowIdx
			headerCellElement.ColumnIndex = &colIdx
			headerCellElement.ElementCategory = GetElementCategory("table_header")

			elements = append(elements, headerCellElement)
			position++
		}
	}

	return elements, position
}

// processTableRow processes a data row in a table
func (p *XLSXParser) processTableRow(file *excelize.File, sheetName, docID, tableID string, region TableRegion, row, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Check if row has any data
	hasData := false
	for col := region.MinCol; col <= region.MaxCol; col++ {
		cellAddr, _ := excelize.CoordinatesToCellName(col, row)
		cellValue, _ := file.GetCellValue(sheetName, cellAddr)
		if cellValue != "" {
			hasData = true
			break
		}
	}

	if !hasData {
		return elements, position
	}

	// Create row element
	rowID := generateID("table_row_")
	rowValues := make([]string, 0, region.MaxCol-region.MinCol+1)

	for col := region.MinCol; col <= region.MaxCol; col++ {
		cellAddr, _ := excelize.CoordinatesToCellName(col, row)
		cellValue, _ := file.GetCellValue(sheetName, cellAddr)
		rowValues = append(rowValues, cellValue)
	}

	rowPreview := fmt.Sprintf("Row %d", row)

	rowElement := Element{
		ElementID:       rowID,
		ElementType:     "table_row",
		ContentPreview:  rowPreview,
		ParentID:        tableID,
		Position:        position,
		Depth:           5,
		ElementCategory: GetElementCategory("table_row"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
			"row":        row,
		},
		Metadata: map[string]interface{}{
			"row":   row,
			"sheet": sheetName,
		},
	}
	elements = append(elements, rowElement)
	position++

	// Process individual cells
	for col := region.MinCol; col <= region.MaxCol; col++ {
		cellAddr, _ := excelize.CoordinatesToCellName(col, row)
		cellValue, _ := file.GetCellValue(sheetName, cellAddr)

		if cellValue != "" {
			cellID := generateID("table_cell_")
			cellPreview := cellValue
			if len(cellPreview) > 100 {
				cellPreview = cellPreview[:97] + "..."
			}

			// Normalize temporal patterns in cell value
			normalizedValue := cellValue
			var temporalMeta map[string]interface{}
			if p.ExtractDates && cellValue != "" {
				normalizedValue, temporalMeta = temporal.ProcessFieldValue(cellAddr, cellValue)
			}

			// Use normalized value for preview
			normalizedPreview := normalizedValue
			if len(normalizedPreview) > 100 {
				normalizedPreview = normalizedPreview[:97] + "..."
			}

			cellElement := Element{
				ElementID:      cellID,
				ElementType:    "table_cell",
				Content:        normalizedValue,
				ContentPreview: normalizedPreview,
				ParentID:       rowID,
				Position:       position,
				Depth:          6,
				ContentLocation: map[string]interface{}{
					"sheet_name": sheetName,
					"cell":       cellAddr,
				},
				Metadata: map[string]interface{}{
					"row":    row,
					"column": col,
					"value":  normalizedValue,
					"sheet":  sheetName,
				},
			}

			// Add temporal normalization metadata
			if temporalMeta != nil {
				for k, v := range temporalMeta {
					cellElement.Metadata[k] = v
				}
			}

			// Add formula if present
			if p.ExtractFormulas {
				formula, _ := file.GetCellFormula(sheetName, cellAddr)
				if formula != "" {
					cellElement.Metadata["formula"] = formula
				}
			}

			// Extract hyperlinks if enabled - add to metadata only
			if p.ExtractLinks {
				hasLink, target, err := file.GetCellHyperLink(sheetName, cellAddr)
				if err == nil && hasLink && target != "" {
					// Add link metadata to cell
					cellElement.Metadata["hyperlink"] = target
				}
			}

			// Extract additional temporal metadata if enabled
			if p.ExtractDates && normalizedValue != "" {
				ProcessTemporalContent(normalizedValue, cellElement.Metadata)
			}

			// UDML Phase 1: Populate promoted fields
			rowIdx := row
			colIdx := col
			cellElement.RowIndex = &rowIdx
			cellElement.ColumnIndex = &colIdx

			// TemporalType from temporal metadata
			if temporalType, ok := cellElement.Metadata["temporal_type"].(string); ok && temporalType != "" {
				cellElement.TemporalType = &temporalType
			}

			cellElement.ElementCategory = GetElementCategory("table_cell")

			elements = append(elements, cellElement)
			position++
		}
	}

	return elements, position
}

// processRows processes rows without table detection (fallback)
func (p *XLSXParser) processRows(file *excelize.File, sheetName, docID, sheetID string, maxRow, maxCol, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Simple row/cell processing without table detection
	for row := 1; row <= maxRow; row++ {
		// Check if row has data
		hasData := false
		for col := 1; col <= maxCol; col++ {
			cellAddr, _ := excelize.CoordinatesToCellName(col, row)
			cellValue, _ := file.GetCellValue(sheetName, cellAddr)
			if cellValue != "" {
				hasData = true
				break
			}
		}

		if !hasData {
			continue
		}

		// Create row element
		rowID := generateID("row_")
		rowElement := Element{
			ElementID:       rowID,
			ElementType:     "table_row",
			ContentPreview:  fmt.Sprintf("Row %d", row),
			ParentID:        sheetID,
			Position:        position,
			Depth:           3,
			ElementCategory: GetElementCategory("table_row"),
			ContentLocation: map[string]interface{}{
				"sheet_name": sheetName,
				"row":        row,
			},
			Metadata: map[string]interface{}{
				"row":   row,
				"sheet": sheetName,
			},
		}
		elements = append(elements, rowElement)
		position++

		// Process cells
		for col := 1; col <= maxCol; col++ {
			cellAddr, _ := excelize.CoordinatesToCellName(col, row)
			cellValue, _ := file.GetCellValue(sheetName, cellAddr)

			if cellValue != "" {
				cellID := generateID("cell_")
				cellPreview := cellValue
				if len(cellPreview) > 100 {
					cellPreview = cellPreview[:97] + "..."
				}

				elementType := "table_cell"
				if row == 1 {
					elementType = "table_header"
				}

				// Normalize temporal patterns in cell value
				normalizedValue := cellValue
				var temporalMeta map[string]interface{}
				if p.ExtractDates && cellValue != "" {
					normalizedValue, temporalMeta = temporal.ProcessFieldValue(cellAddr, cellValue)
				}

				// Use normalized value for preview
				normalizedPreview := normalizedValue
				if len(normalizedPreview) > 100 {
					normalizedPreview = normalizedPreview[:97] + "..."
				}

				cellElement := Element{
					ElementID:       cellID,
					ElementType:     elementType,
					Content:         normalizedValue,
					ContentPreview:  normalizedPreview,
					ParentID:        rowID,
					Position:        position,
					Depth:           4,
					ElementCategory: GetElementCategory(elementType),
					ContentLocation: map[string]interface{}{
						"sheet_name": sheetName,
						"cell":       cellAddr,
					},
					Metadata: map[string]interface{}{
						"row":    row,
						"column": col,
						"value":  normalizedValue,
						"sheet":  sheetName,
					},
				}

				// Add temporal normalization metadata
				if temporalMeta != nil {
					for k, v := range temporalMeta {
						cellElement.Metadata[k] = v
					}
				}

				// Extract additional temporal metadata if enabled
				if p.ExtractDates && normalizedValue != "" {
					ProcessTemporalContent(normalizedValue, cellElement.Metadata)
				}

				elements = append(elements, cellElement)
				position++
			}
		}
	}

	return elements, position
}

// processRawRows processes sheet content row-by-row like Python parser (first pass)
// Skips cells that are covered by structured table regions to avoid duplication
func (p *XLSXParser) processRawRows(file *excelize.File, sheetName, docID, sheetID string, maxRow, maxCol, startPosition int, tableRegions []TableRegion) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Process each row (including header row)
	for row := 1; row <= maxRow; row++ {
		// Create row element
		rowID := generateID(fmt.Sprintf("raw_row_%d_", row))
		rowPreview := fmt.Sprintf("Row %d", row)

		rowElement := Element{
			ElementID:       rowID,
			ElementType:     "table_row",
			ContentPreview:  rowPreview,
			ParentID:        sheetID,
			Position:        position,
			Depth:           3,
			ElementCategory: GetElementCategory("table_row"),
			ContentLocation: map[string]interface{}{
				"sheet_name": sheetName,
				"row":        row,
			},
			Metadata: map[string]interface{}{
				"sheet": sheetName,
				"row":   row,
			},
		}
		elements = append(elements, rowElement)
		position++

		// Process each cell in the row
		for col := 1; col <= maxCol; col++ {
			cellAddr, _ := excelize.CoordinatesToCellName(col, row)
			cellValue, _ := file.GetCellValue(sheetName, cellAddr)

			// Skip empty cells in raw processing
			if cellValue == "" {
				continue
			}

			// Skip cells that are covered by table regions (to avoid duplication)
			if p.isCellInTableRegion(row, col, tableRegions) {
				continue
			}

			// Create cell element
			cellID := generateID(fmt.Sprintf("raw_cell_%d_%d_", row, col))

			// Determine element type based on row position
			var elementType string
			if row == 1 {
				// First row is typically headers
				elementType = "table_header"
			} else {
				elementType = "table_cell"
			}

			// Create content preview
			cellPreview := cellValue
			if len(cellPreview) > p.MaxContentPreview {
				cellPreview = cellPreview[:p.MaxContentPreview-3] + "..."
			}

			// Normalize temporal patterns in cell value
			normalizedValue := cellValue
			var temporalMeta map[string]interface{}
			if p.ExtractDates && cellValue != "" {
				normalizedValue, temporalMeta = temporal.ProcessFieldValue(cellAddr, cellValue)
			}

			// Use normalized value for preview
			normalizedPreview := normalizedValue
			if len(normalizedPreview) > p.MaxContentPreview {
				normalizedPreview = normalizedPreview[:p.MaxContentPreview-3] + "..."
			}

			cellElement := Element{
				ElementID:      cellID,
				ElementType:    elementType,
				Content:        normalizedValue,
				ContentPreview: normalizedPreview,
				ParentID:       rowID,
				Position:       position,
				Depth:          4,
				ContentLocation: map[string]interface{}{
					"sheet_name": sheetName,
					"cell":       cellAddr,
					"row":        row,
					"column":     col,
				},
				Metadata: map[string]interface{}{
					"row":    row,
					"column": col,
					"value":  normalizedValue,
					"sheet":  sheetName,
					"cell":   cellAddr,
				},
			}

			// Add temporal normalization metadata
			if temporalMeta != nil {
				for k, v := range temporalMeta {
					cellElement.Metadata[k] = v
				}
			}

			// Add formula if available and enabled
			if p.ExtractFormulas {
				formula, _ := file.GetCellFormula(sheetName, cellAddr)
				if formula != "" {
					cellElement.Metadata["formula"] = formula
					// Also update content if it's a formula
					if cellValue == "" {
						cellElement.Content = formula
						cellElement.ContentPreview = formula
						if len(cellElement.ContentPreview) > p.MaxContentPreview {
							cellElement.ContentPreview = cellElement.ContentPreview[:p.MaxContentPreview-3] + "..."
						}
					}
				}
			}

			// Extract additional temporal metadata if enabled
			if p.ExtractDates && normalizedValue != "" {
				ProcessTemporalContent(normalizedValue, cellElement.Metadata)
			}

			// UDML Phase 1: Populate promoted fields
			rowIdx := row
			colIdx := col
			cellElement.RowIndex = &rowIdx
			cellElement.ColumnIndex = &colIdx

			// TemporalType from temporal metadata
			if temporalType, ok := cellElement.Metadata["temporal_type"].(string); ok && temporalType != "" {
				cellElement.TemporalType = &temporalType
			}

			cellElement.ElementCategory = GetElementCategory(elementType)

			elements = append(elements, cellElement)
			position++
		}
	}

	return elements, position
}

// processMergedCells extracts merged cell information from a worksheet
func (p *XLSXParser) processMergedCells(file *excelize.File, sheetName, docID, sheetID string, startPosition int) ([]Element, int) {
	var elements []Element
	position := startPosition

	// Get merged cells from the sheet
	mergedCells, err := file.GetMergeCells(sheetName)
	if err != nil || len(mergedCells) == 0 {
		return elements, position
	}

	// Create merged cells container element
	mergedCellsID := generateID("merged_cells_")
	mergedCellsElement := Element{
		ElementID:       mergedCellsID,
		ElementType:     "merged_cells",
		ContentPreview:  fmt.Sprintf("Merged cells in sheet '%s'", sheetName),
		ParentID:        sheetID,
		Position:        position,
		Depth:           3,
		ElementCategory: GetElementCategory("merged_cells"),
		ContentLocation: map[string]interface{}{
			"sheet_name": sheetName,
			"type":       "merged_cells",
		},
		Metadata: map[string]interface{}{
			"sheet": sheetName,
			"count": len(mergedCells),
		},
	}
	elements = append(elements, mergedCellsElement)
	position++

	// Process each merged cell range
	for i, mergedCell := range mergedCells {
		mergedID := generateID(fmt.Sprintf("merged_%d_", i))

		// Get the value from the top-left cell (Excel stores merged cell value there)
		cellValue, _ := file.GetCellValue(sheetName, mergedCell.GetStartAxis())

		// Create content preview
		contentPreview := fmt.Sprintf("Merged cell %s", mergedCell.GetCellValue())
		if cellValue != "" {
			contentPreview = fmt.Sprintf("Merged cell: %s", cellValue)
		}

		// Truncate if too long
		if len(contentPreview) > p.MaxContentPreview {
			contentPreview = contentPreview[:p.MaxContentPreview-3] + "..."
		}

		// Create merged cell element
		mergedElement := Element{
			ElementID:       mergedID,
			ElementType:     "merged_cell",
			Content:         cellValue,
			ContentPreview:  contentPreview,
			ParentID:        mergedCellsID,
			Position:        position,
			Depth:           4,
			ElementCategory: GetElementCategory("merged_cell"),
			ContentLocation: map[string]interface{}{
				"sheet_name": sheetName,
				"range":      mergedCell.GetCellValue(),
				"type":       "merged_cell",
			},
			Metadata: map[string]interface{}{
				"sheet":      sheetName,
				"range":      mergedCell.GetCellValue(),
				"start_axis": mergedCell.GetStartAxis(),
				"end_axis":   mergedCell.GetEndAxis(),
				"value":      cellValue,
			},
		}
		elements = append(elements, mergedElement)
		position++
	}

	return elements, position
}

// isCellInTableRegion checks if a cell is covered by any table region
func (p *XLSXParser) isCellInTableRegion(row, col int, tableRegions []TableRegion) bool {
	for _, region := range tableRegions {
		if row >= region.MinRow && row <= region.MaxRow &&
			col >= region.MinCol && col <= region.MaxCol {
			return true
		}
	}
	return false
}

// columnLetter converts column number to Excel letter (1=A, 2=B, etc.)
func columnLetter(col int) string {
	result := ""
	for col > 0 {
		col--
		result = string(rune('A'+col%26)) + result
		col /= 26
	}
	return result
}
