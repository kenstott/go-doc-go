package parser

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDFParser handles PDF document parsing
type PDFParser struct {
	MaxContentPreview    int
	ExtractMetadata      bool
	ExtractLinks         bool
	DetectHeaders        bool
	MinHeaderFontSize    float64
	MaxPages             int
	ExtractTables        bool
	MinTableRows         int
	MinTableCols         int
	PreserveLayout       bool
}

// TextBlock represents a block of text with position information
type TextBlock struct {
	Text     string
	X        float64
	Y        float64
	Width    float64
	Height   float64
	FontSize float64
	FontName string
	PageNum  int
}

// StructuredPage represents a page with structured content
type StructuredPage struct {
	PageNum     int
	TextBlocks  []TextBlock
	Width       float64
	Height      float64
}

// NewPDFParser creates a new PDF parser instance
func NewPDFParser() *PDFParser {
	return &PDFParser{
		MaxContentPreview: 100,
		ExtractMetadata:   true,
		ExtractLinks:      true,
		DetectHeaders:     true,
		MinHeaderFontSize: 12.0,
		MaxPages:          1000,
		ExtractTables:     true,
		MinTableRows:      2,
		MinTableCols:      2,
		PreserveLayout:    true,
	}
}

// Parse extracts text and structure from a PDF file
func (p *PDFParser) Parse(docID string, content interface{}) (*ParseResult, error) {
	// Handle different input types
	var reader io.ReadSeeker
	var contentBytes []byte

	switch v := content.(type) {
	case string:
		// If it's a file path
		if _, err := os.Stat(v); err == nil {
			file, err := os.Open(v)
			if err != nil {
				return nil, fmt.Errorf("failed to open PDF file: %v", err)
			}
			defer file.Close()
			reader = file

			// Read for hash calculation
			data, err := os.ReadFile(v)
			if err != nil {
				return nil, fmt.Errorf("failed to read PDF file: %v", err)
			}
			contentBytes = data
		} else {
			// Treat as PDF content string
			contentBytes = []byte(v)
			reader = bytes.NewReader(contentBytes)
		}
	case []byte:
		contentBytes = v
		reader = bytes.NewReader(v)
	case io.ReadSeeker:
		reader = v
		// Read content for hash
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, reader); err != nil {
			return nil, fmt.Errorf("failed to read PDF content: %v", err)
		}
		contentBytes = buf.Bytes()
		reader.Seek(0, io.SeekStart)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Create result structure
	result := &ParseResult{
		Document: Document{
			ID:      docID,
			DocType: "pdf",
			Title:   "",
		},
		Elements:      []Element{},
		Relationships: []Relationship{},
		Links:         []Link{},
	}

	// Create root element
	rootID := generateID("pdf_root")
	contentHash := fmt.Sprintf("%x", md5.Sum(contentBytes))

	rootElement := Element{
		ElementID:      rootID,
		ElementType:    "root",
		Content:        "",
		ContentPreview: p.truncateContent(fmt.Sprintf("PDF Document: %s", docID)),
		ParentID:       "",
		Position:       0,
		Depth:          0,
		ContentLocation: map[string]interface{}{
			"type":         "pdf_document",
			"content_hash": contentHash,
		},
	}
	result.Elements = append(result.Elements, rootElement)

	// Create body element
	bodyID := generateID("pdf_body")
	bodyElement := Element{
		ElementID:      bodyID,
		ElementType:    "body",
		Content:        "",
		ContentPreview: "Document Body",
		ParentID:       rootID,
		Position:       1,
		Depth:          1,
		ContentLocation: map[string]interface{}{
			"type": "pdf_body",
		},
	}
	result.Elements = append(result.Elements, bodyElement)

	// Create root-body relationship
	result.Relationships = append(result.Relationships, Relationship{
		RelationshipID:   generateID("rel"),
		RelationshipType: "contains",
		SourceElementID:  rootID,
		TargetElementID:  bodyID,
	})

	// Create configuration for pdfcpu
	conf := model.NewDefaultConfiguration()

	// Read and validate PDF
	ctx, err := api.ReadContext(reader, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PDF: %v", err)
	}

	// Extract metadata if available
	// Note: ctx.Info is an IndirectRef in pdfcpu, not directly accessible
	// We'll skip metadata extraction for now until we understand the API better
	if p.ExtractMetadata {
		// TODO: Figure out how to extract metadata with new pdfcpu API
		result.Document.Metadata = map[string]interface{}{
			"note": "metadata extraction pending API investigation",
		}
	}

	// Extract text content page by page
	pageCount := ctx.PageCount
	if p.MaxPages > 0 && pageCount > p.MaxPages {
		pageCount = p.MaxPages
	}

	elementPosition := 2
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		// Create page element
		pageID := generateID(fmt.Sprintf("page_%d", pageNum))
		pageElement := Element{
			ElementID:      pageID,
			ElementType:    "page",
			Content:        "",
			ContentPreview: fmt.Sprintf("Page %d", pageNum),
			ParentID:       bodyID,
			Position:       elementPosition,
			Depth:          2,
			ContentLocation: map[string]interface{}{
				"type":        "pdf_page",
				"page_number": pageNum,
			},
		}
		result.Elements = append(result.Elements, pageElement)
		elementPosition++

		// Create body-page relationship
		result.Relationships = append(result.Relationships, Relationship{
			RelationshipID:   generateID("rel"),
			RelationshipType: "contains",
			SourceElementID:  bodyID,
			TargetElementID:  pageID,
		})

		// Extract structured content from page
		structuredPage, err := p.extractStructuredPage(ctx, pageNum)
		if err != nil {
			// Fallback to simple text extraction
			pageText, _ := p.extractPageText(ctx, pageNum)
			if pageText != "" {
				// Create a simple paragraph element
				elemID := generateID(fmt.Sprintf("para_p%d", pageNum))
				elem := Element{
					ElementID:      elemID,
					ElementType:    "paragraph",
					Content:        pageText,
					ContentPreview: p.truncateContent(pageText),
					ParentID:       pageID,
					Position:       elementPosition,
					Depth:          3,
					ContentLocation: map[string]interface{}{
						"type":        "pdf_paragraph",
						"page_number": pageNum,
					},
				}
				result.Elements = append(result.Elements, elem)
				elementPosition++

				result.Relationships = append(result.Relationships, Relationship{
					RelationshipID:   generateID("rel"),
					RelationshipType: "contains",
					SourceElementID:  pageID,
					TargetElementID:  elemID,
				})
			}
			continue
		}

		// Process structured content
		elements := p.processStructuredPage(structuredPage, pageID)

		for _, elem := range elements {
			elem.Position = elementPosition
			elem.ParentID = pageID
			elem.Depth = 3
			result.Elements = append(result.Elements, elem)
			elementPosition++

			// Create page-element relationship
			result.Relationships = append(result.Relationships, Relationship{
				RelationshipID:   generateID("rel"),
				RelationshipType: "contains",
				SourceElementID:  pageID,
				TargetElementID:  elem.ElementID,
			})

			// Extract links from text
			if p.ExtractLinks {
				links := p.extractLinksFromText(elem.Content, elem.ElementID)
				result.Links = append(result.Links, links...)
			}
		}
	}

	return result, nil
}

// extractStructuredPage extracts structured content from a page
func (p *PDFParser) extractStructuredPage(ctx *model.Context, pageNum int) (*StructuredPage, error) {
	// For now, we'll use text extraction and infer structure
	// In a production system, you'd parse the content streams to get exact positions

	page := &StructuredPage{
		PageNum: pageNum,
		Width:   612.0,  // Default US Letter width in points
		Height:  792.0,  // Default US Letter height in points
	}

	// Extract text
	text, err := p.extractPageText(ctx, pageNum)
	if err != nil {
		return nil, err
	}

	// Split into lines and create text blocks
	lines := strings.Split(text, "\n")
	yPosition := page.Height - 50.0 // Start from top of page with margin

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			yPosition -= 20 // Empty line spacing
			continue
		}

		// Create text block for each line
		// In production, you'd extract actual font metrics
		fontSize := 12.0
		if p.isLikelyHeader(line) {
			fontSize = 16.0
		}

		block := TextBlock{
			Text:     line,
			X:        50.0, // Left margin
			Y:        yPosition,
			Width:    page.Width - 100.0, // Account for margins
			Height:   fontSize * 1.2,
			FontSize: fontSize,
			FontName: "Default",
			PageNum:  pageNum,
		}

		page.TextBlocks = append(page.TextBlocks, block)
		yPosition -= block.Height + 5 // Move down for next line
	}

	return page, nil
}

// extractPageText extracts plain text from a page
func (p *PDFParser) extractPageText(ctx *model.Context, pageNum int) (string, error) {
	// TODO: Implement proper text extraction
	// The pdfcpu library doesn't have a direct text extraction API
	// For now, return a placeholder
	return fmt.Sprintf("Page %d content (text extraction pending)", pageNum), nil
}

// processStructuredPage converts structured page content into elements
func (p *PDFParser) processStructuredPage(page *StructuredPage, pageID string) []Element {
	var elements []Element

	if len(page.TextBlocks) == 0 {
		return elements
	}

	// Detect columns and reading order
	columns := p.detectColumns(page.TextBlocks, page.Width)

	// Sort blocks by reading order (column-aware)
	sort.Slice(page.TextBlocks, func(i, j int) bool {
		blockI := page.TextBlocks[i]
		blockJ := page.TextBlocks[j]

		// First, check if blocks are in different columns
		colI := p.getColumnIndex(blockI.X, columns)
		colJ := p.getColumnIndex(blockJ.X, columns)

		if colI != colJ {
			// If multi-column, read left column first (for LTR languages)
			return colI < colJ
		}

		// Within same column, sort by Y position (top to bottom)
		if abs(blockI.Y-blockJ.Y) > 5 {
			return blockI.Y > blockJ.Y // Inverted because PDF Y starts from bottom
		}

		// If on same line, sort by X position
		return blockI.X < blockJ.X
	})

	// Group blocks into logical elements
	currentGroup := []TextBlock{}
	groupType := ""

	for i, block := range page.TextBlocks {
		// Determine block type
		blockType := p.classifyTextBlock(block)

		// Check if we should start a new group
		shouldStartNewGroup := false
		if i == 0 {
			shouldStartNewGroup = true
			groupType = blockType
		} else {
			prevBlock := page.TextBlocks[i-1]
			// Start new group if type changes or there's a significant gap
			if blockType != groupType || p.hasSignificantGap(prevBlock, block) {
				shouldStartNewGroup = true
			}
		}

		if shouldStartNewGroup && len(currentGroup) > 0 {
			// Create element from current group
			elem := p.createElementFromBlocks(currentGroup, groupType, page.PageNum)
			elements = append(elements, elem)
			currentGroup = []TextBlock{}
		}

		currentGroup = append(currentGroup, block)
		groupType = blockType
	}

	// Don't forget the last group
	if len(currentGroup) > 0 {
		elem := p.createElementFromBlocks(currentGroup, groupType, page.PageNum)
		elements = append(elements, elem)
	}

	// Detect and mark tables
	elements = p.detectTables(elements, page)

	// Detect lists
	elements = p.detectLists(elements)

	return elements
}

// classifyTextBlock determines the type of a text block
func (p *PDFParser) classifyTextBlock(block TextBlock) string {
	text := block.Text

	// Check for footnote/endnote characteristics
	if p.isLikelyFootnote(block) {
		return "footnote"
	}

	// Check for header characteristics
	if block.FontSize >= p.MinHeaderFontSize && p.isLikelyHeader(text) {
		return "header"
	}

	// Check for list items
	if p.isListItem(text) {
		return "list_item"
	}

	// Default to paragraph
	return "paragraph"
}

// hasSignificantGap checks if there's a significant gap between blocks
func (p *PDFParser) hasSignificantGap(block1, block2 TextBlock) bool {
	// Vertical gap
	yGap := abs(block1.Y - block2.Y)
	if yGap > block1.Height*2 {
		return true
	}

	// Check for column break (significant X difference on similar Y)
	if abs(block1.Y-block2.Y) < block1.Height {
		xGap := abs(block1.X - block2.X)
		if xGap > 100 { // More than 100 points suggests column break
			return true
		}
	}

	return false
}

// createElementFromBlocks creates an element from grouped text blocks
func (p *PDFParser) createElementFromBlocks(blocks []TextBlock, elemType string, pageNum int) Element {
	// Combine text from blocks
	var texts []string
	for _, block := range blocks {
		texts = append(texts, block.Text)
	}
	content := strings.Join(texts, " ")

	// Create element
	elem := Element{
		ElementID:      generateID(fmt.Sprintf("%s_p%d", elemType, pageNum)),
		ElementType:    elemType,
		Content:        content,
		ContentPreview: p.truncateContent(content),
		ContentLocation: map[string]interface{}{
			"type":        fmt.Sprintf("pdf_%s", elemType),
			"page_number": pageNum,
		},
	}

	// Add position information
	if len(blocks) > 0 {
		elem.ContentLocation["x"] = blocks[0].X
		elem.ContentLocation["y"] = blocks[0].Y
		elem.ContentLocation["width"] = blocks[0].Width
		elem.ContentLocation["height"] = blocks[0].Height
	}

	return elem
}

// detectTables identifies table structures in elements
func (p *PDFParser) detectTables(elements []Element, page *StructuredPage) []Element {
	// Simple heuristic: look for elements with tab-separated or aligned content
	var result []Element

	for i, elem := range elements {
		if elem.ElementType != "paragraph" {
			result = append(result, elem)
			continue
		}

		// Check if content looks like a table row
		if p.looksLikeTableRow(elem.Content) {
			// Look ahead and behind for more table rows
			tableElements := []Element{elem}

			// Look ahead
			for j := i + 1; j < len(elements) && j < i+20; j++ {
				if elements[j].ElementType == "paragraph" && p.looksLikeTableRow(elements[j].Content) {
					tableElements = append(tableElements, elements[j])
				} else {
					break
				}
			}

			// If we found multiple rows, create a table
			if len(tableElements) >= p.MinTableRows {
				tableElem := p.createTableElement(tableElements, page.PageNum)
				result = append(result, tableElem)

				// Skip the elements we've consumed
				i += len(tableElements) - 1
			} else {
				result = append(result, elem)
			}
		} else {
			result = append(result, elem)
		}
	}

	return result
}

// looksLikeTableRow checks if text looks like a table row
func (p *PDFParser) looksLikeTableRow(text string) bool {
	// Check for multiple tab-separated or pipe-separated values
	tabs := strings.Count(text, "\t")
	pipes := strings.Count(text, "|")

	if tabs >= p.MinTableCols-1 || pipes >= p.MinTableCols {
		return true
	}

	// Check for multiple numbers or dates in a row (common in tables)
	numberPattern := regexp.MustCompile(`\d+[\.\,]?\d*`)
	matches := numberPattern.FindAllString(text, -1)
	if len(matches) >= p.MinTableCols {
		return true
	}

	return false
}

// createTableElement creates a table element from row elements
func (p *PDFParser) createTableElement(rows []Element, pageNum int) Element {
	var content strings.Builder
	for i, row := range rows {
		content.WriteString(row.Content)
		if i < len(rows)-1 {
			content.WriteString("\n")
		}
	}

	return Element{
		ElementID:      generateID(fmt.Sprintf("table_p%d", pageNum)),
		ElementType:    "table",
		Content:        content.String(),
		ContentPreview: p.truncateContent(content.String()),
		ContentLocation: map[string]interface{}{
			"type":        "pdf_table",
			"page_number": pageNum,
			"row_count":   len(rows),
		},
	}
}

// detectLists identifies list structures in elements
func (p *PDFParser) detectLists(elements []Element) []Element {
	var result []Element

	for _, elem := range elements {
		if elem.ElementType == "list_item" {
			elem.ElementType = "list_item"
			// Could group consecutive list items into a list element
		}
		result = append(result, elem)
	}

	return result
}

// isListItem checks if text is a list item
func (p *PDFParser) isListItem(text string) bool {
	// Numbered lists
	if matched, _ := regexp.MatchString(`^\d+[\.\)]\s+`, text); matched {
		return true
	}

	// Lettered lists
	if matched, _ := regexp.MatchString(`^[a-zA-Z][\.\)]\s+`, text); matched {
		return true
	}

	// Bullet points
	if matched, _ := regexp.MatchString(`^[•·▪▫◦‣⁃\-\*]\s+`, text); matched {
		return true
	}

	return false
}

// abs returns absolute value for float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// detectColumns detects column boundaries in text blocks
func (p *PDFParser) detectColumns(blocks []TextBlock, pageWidth float64) []float64 {
	// Simple column detection: look for vertical gaps in X positions
	var xPositions []float64
	for _, block := range blocks {
		xPositions = append(xPositions, block.X)
	}

	sort.Float64s(xPositions)

	// Find significant gaps that might indicate column boundaries
	var columns []float64
	columns = append(columns, 0) // Start of page

	for i := 1; i < len(xPositions); i++ {
		gap := xPositions[i] - xPositions[i-1]
		// If gap is more than 50 points, might be a column boundary
		if gap > 50 {
			columns = append(columns, xPositions[i])
		}
	}

	columns = append(columns, pageWidth) // End of page

	// If we only have start and end, it's single column
	if len(columns) == 2 {
		return columns
	}

	// Merge columns that are too close
	var merged []float64
	merged = append(merged, columns[0])
	for i := 1; i < len(columns)-1; i++ {
		if columns[i]-merged[len(merged)-1] > 100 { // Minimum column width
			merged = append(merged, columns[i])
		}
	}
	merged = append(merged, columns[len(columns)-1])

	return merged
}

// getColumnIndex returns which column a given X position falls into
func (p *PDFParser) getColumnIndex(x float64, columns []float64) int {
	for i := 0; i < len(columns)-1; i++ {
		if x >= columns[i] && x < columns[i+1] {
			return i
		}
	}
	return len(columns) - 2 // Last column
}

// isLikelyFootnote checks if a text block is likely a footnote
func (p *PDFParser) isLikelyFootnote(block TextBlock) bool {
	// Footnotes are typically:
	// 1. At the bottom of the page (Y < 150 points from bottom)
	// 2. Smaller font size (< 10 points)
	// 3. Start with a superscript number or special character

	// Check position (near bottom of page)
	if block.Y > 150 { // More than 150 points from bottom
		return false
	}

	// Check font size
	if block.FontSize >= 10 {
		return false
	}

	// Check if starts with footnote marker
	text := strings.TrimSpace(block.Text)

	// Common footnote patterns
	footnotePatterns := []string{
		`^\d+\.?\s`,           // Starts with number
		`^\[\d+\]`,            // [1] style
		`^\*+\s`,              // *, **, *** style
		`^[†‡§¶]\s`,          // Special footnote symbols
		`^\(\d+\)`,            // (1) style
		`^[a-z]\.\s`,          // a. b. c. style
	}

	for _, pattern := range footnotePatterns {
		if matched, _ := regexp.MatchString(pattern, text); matched {
			return true
		}
	}

	return false
}

// splitIntoParagraphs splits text into paragraphs
func (p *PDFParser) splitIntoParagraphs(text string) []string {
	// Split by double newlines or significant whitespace
	paragraphs := []string{}

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Split by double newlines
	blocks := strings.Split(text, "\n\n")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block != "" {
			// Also split single newlines if they look like separate paragraphs
			lines := strings.Split(block, "\n")
			currentPara := ""

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					if currentPara != "" {
						paragraphs = append(paragraphs, currentPara)
						currentPara = ""
					}
				} else {
					if currentPara != "" {
						// Check if this looks like a continuation or new paragraph
						if p.looksLikeNewParagraph(line) {
							paragraphs = append(paragraphs, currentPara)
							currentPara = line
						} else {
							currentPara += " " + line
						}
					} else {
						currentPara = line
					}
				}
			}

			if currentPara != "" {
				paragraphs = append(paragraphs, currentPara)
			}
		}
	}

	return paragraphs
}

// looksLikeNewParagraph checks if a line looks like it starts a new paragraph
func (p *PDFParser) looksLikeNewParagraph(line string) bool {
	// Simple heuristics
	if len(line) == 0 {
		return false
	}

	// Starts with number (numbered list)
	if matched, _ := regexp.MatchString(`^\d+\.?\s+`, line); matched {
		return true
	}

	// Starts with bullet
	if matched, _ := regexp.MatchString(`^[•·▪▫◦‣⁃]\s+`, line); matched {
		return true
	}

	// Starts with uppercase and previous didn't end with period
	// (would need context from previous line for this)

	// All caps (likely header)
	if strings.ToUpper(line) == line && len(line) > 3 {
		return true
	}

	return false
}

// isLikelyHeader determines if text is likely a header
func (p *PDFParser) isLikelyHeader(text string) bool {
	// Simple heuristics for header detection

	// Short text (less than 100 chars) might be header
	if len(text) > 100 {
		return false
	}

	// All uppercase
	if strings.ToUpper(text) == text && len(text) > 3 {
		return true
	}

	// Numbered sections (1., 1.1, etc.)
	if matched, _ := regexp.MatchString(`^\d+(\.\d+)*\.?\s+\w+`, text); matched {
		return true
	}

	// Common header patterns
	headerPatterns := []string{
		`^(Chapter|Section|Part)\s+\d+`,
		`^(Introduction|Conclusion|Abstract|Summary|References)$`,
		`^(Table of Contents|List of Figures|List of Tables)`,
	}

	for _, pattern := range headerPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, text); matched {
			return true
		}
	}

	// Title case (most words capitalized)
	words := strings.Fields(text)
	if len(words) > 0 {
		capitalizedCount := 0
		for _, word := range words {
			if len(word) > 0 && unicode.IsUpper(rune(word[0])) {
				capitalizedCount++
			}
		}
		if float64(capitalizedCount)/float64(len(words)) > 0.7 {
			return true
		}
	}

	return false
}

// extractLinksFromText finds URLs and email addresses in text
func (p *PDFParser) extractLinksFromText(text string, sourceID string) []Link {
	var links []Link

	// URL pattern
	urlPattern := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	urls := urlPattern.FindAllString(text, -1)
	for _, url := range urls {
		links = append(links, Link{
			LinkID:          generateID("link"),
			SourceElementID: sourceID,
			LinkType:        "url",
			LinkTarget:      url,
			Context:         p.truncateContent(text),
		})
	}

	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailPattern.FindAllString(text, -1)
	for _, email := range emails {
		links = append(links, Link{
			LinkID:          generateID("link"),
			SourceElementID: sourceID,
			LinkType:        "email",
			LinkTarget:      email,
			Context:         p.truncateContent(text),
		})
	}

	// File path pattern (PDFs often reference other documents)
	filePattern := regexp.MustCompile(`[A-Za-z]:[\\\/](?:[^\\\/\s:*?"<>|]+[\\\/])*[^\\\/\s:*?"<>|]+\.\w+`)
	files := filePattern.FindAllString(text, -1)
	for _, file := range files {
		links = append(links, Link{
			LinkID:          generateID("link"),
			SourceElementID: sourceID,
			LinkType:        LinkTypeFile,
			LinkTarget:      file,
			Context:         p.truncateContent(text),
		})
	}

	return links
}

// truncateContent truncates content for preview
func (p *PDFParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}