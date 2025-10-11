package parser

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PptxElement represents a parsed PPTX element
type PptxElement struct {
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

// PptxRelationship represents a relationship between elements
type PptxRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_id"`
	TargetElementID  string                 `json:"target_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// PptxLink represents an extracted link
type PptxLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// PptxParseRequest represents the input for PPTX parsing
type PptxParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// PptxParseResponse represents the output of PPTX parsing
type PptxParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []PptxElement          `json:"elements"`
	Links         []PptxLink             `json:"links"`
	Relationships []PptxRelationship     `json:"relationships"`
}

// XML structures for PPTX parsing
type Presentation struct {
	XMLName  xml.Name `xml:"presentation"`
	SlideIds SlideIds `xml:"sldIdLst"`
}

type SlideIds struct {
	SlideIds []SlideId `xml:"sldId"`
}

type SlideId struct {
	ID    string `xml:"id,attr"`
	RelID string `xml:"id,attr"`
}

type Slide struct {
	XMLName xml.Name        `xml:"sld"`
	CSld    CommonSlideData `xml:"cSld"`
}

type CommonSlideData struct {
	SpTree ShapeTree `xml:"spTree"`
}

type ShapeTree struct {
	Shapes        []Shape        `xml:"sp"`
	GraphicFrames []GraphicFrame `xml:"graphicFrame"`
	Pictures      []Picture      `xml:"pic"`
	GroupShapes   []GroupShape   `xml:"grpSp"`
}

// GraphicFrame can contain tables
type GraphicFrame struct {
	Graphic          Graphic                    `xml:"graphic"`
	NvGraphicFramePr NonVisualGraphicFrameProps `xml:"nvGraphicFramePr"`
}

type NonVisualGraphicFrameProps struct {
	CNvPr CommonNonVisualProps `xml:"cNvPr"`
}

type Graphic struct {
	GraphicData GraphicData `xml:"graphicData"`
}

type GraphicData struct {
	URI   string    `xml:"uri,attr"`
	Table PptxTable `xml:"tbl"`
	Chart ChartData `xml:"chart"`
}

type ChartData struct {
	// Charts are referenced by relationship ID, not embedded
	// We'll detect charts by URI and frame name
}

type PptxTable struct {
	Rows []PptxTableRow `xml:"tr"`
}

type PptxTableRow struct {
	Cells []PptxTableCell `xml:"tc"`
}

type PptxTableCell struct {
	TxBody TextBody `xml:"txBody"`
	TcPr   PptxTcPr `xml:"tcPr"`
}

type PptxTcPr struct {
	GridSpan int  `xml:"gridSpan,attr"`
	RowSpan  int  `xml:"rowSpan,attr"`
	HMerge   bool `xml:"hMerge,attr"`
	VMerge   bool `xml:"vMerge,attr"`
}

type Shape struct {
	NvSpPr NonVisualShapeProps `xml:"nvSpPr"`
	SpPr   ShapeProperties     `xml:"spPr"`
	TxBody TextBody            `xml:"txBody"`
}

type NonVisualShapeProps struct {
	CNvPr CommonNonVisualProps `xml:"cNvPr"`
}

type CommonNonVisualProps struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type ShapeProperties struct {
	// Shape properties like position, size, etc.
}

type TextBody struct {
	Paragraphs []Paragraph `xml:"p"`
}

type Paragraph struct {
	Runs []Run `xml:"r"`
}

type Run struct {
	Text string `xml:"t"`
}

// Picture represents a picture shape
type Picture struct {
	NvPicPr  NonVisualPictureProps `xml:"nvPicPr"`
	BlipFill BlipFill              `xml:"blipFill"`
}

type NonVisualPictureProps struct {
	CNvPr CommonNonVisualProps `xml:"cNvPr"`
}

type BlipFill struct {
	Blip Blip `xml:"blip"`
}

type Blip struct {
	Embed string `xml:"embed,attr"`
}

// GroupShape represents a group of shapes
type GroupShape struct {
	NvGrpSpPr NonVisualGroupProps `xml:"nvGrpSpPr"`
	GrpSpPr   GroupShapeProps     `xml:"grpSpPr"`
	Shapes    []Shape             `xml:"sp"`
	Pictures  []Picture           `xml:"pic"`
	SubGroups []GroupShape        `xml:"grpSp"`
}

type NonVisualGroupProps struct {
	CNvPr CommonNonVisualProps `xml:"cNvPr"`
}

type GroupShapeProps struct {
	// Group shape properties
}

// Comments structures
type CommentList struct {
	XMLName  xml.Name  `xml:"cmLst"`
	Comments []Comment `xml:"cm"`
}

type Comment struct {
	AuthorID string `xml:"authorId,attr"`
	DateTime string `xml:"dt,attr"`
	Idx      string `xml:"idx,attr"`
	Text     string `xml:"text"`
}

// PptxRelationships structure for parsing relationships
type PptxRelationships struct {
	XMLName       xml.Name           `xml:"Relationships"`
	Relationships []PptxRelationItem `xml:"Relationship"`
}

type PptxRelationItem struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

// PptxParser handles PPTX document parsing
type PptxParser struct {
	MaxContentPreview int
	ExtractNotes      bool
	ExtractComments   bool
	ExtractShapes     bool
	ExtractTables     bool
	ExtractImages     bool
}

// NewPptxParser creates a new PPTX parser instance
func NewPptxParser() *PptxParser {
	return &PptxParser{
		MaxContentPreview: 100,
		ExtractNotes:      true,
		ExtractComments:   true,
		ExtractShapes:     true,
		ExtractTables:     true,
		ExtractImages:     true,
	}
}

// GetName returns the parser name
func (p *PptxParser) GetName() string {
	return "pptx"
}

// GetSupportedFormats returns supported file formats
func (p *PptxParser) GetSupportedFormats() []string {
	return []string{".pptx", "pptx"}
}

// Parse implements the Parser interface for PPTX documents
func (p *PptxParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract content from request - PPTX parser expects file path
	var contentPath string
	switch v := req.Content.(type) {
	case string:
		contentPath = v
	default:
		return nil, fmt.Errorf("unsupported content type for PPTX parser: %T (expected file path string)", req.Content)
	}

	// Create PPTX-specific request
	pptxRequest := PptxParseRequest{
		ID:       req.ID,
		Content:  contentPath,
		Metadata: req.Metadata,
	}

	// Parse using internal implementation
	response, err := p.parsePptx(pptxRequest)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult with promoted fields
	return response.ToParseResult(), nil
}

// SupportsStreaming returns whether the parser supports streaming
func (p *PptxParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *PptxParser) Close() error {
	// PPTX parser has no persistent resources to clean up
	return nil
}

// parsePptx parses a PPTX document into structured elements (internal implementation)
func (p *PptxParser) parsePptx(request PptxParseRequest) (*PptxParseResponse, error) {
	// Open PPTX file as ZIP archive
	reader, err := zip.OpenReader(request.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to open PPTX file: %w", err)
	}
	defer reader.Close()

	// Initialize response
	response := &PptxParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "pptx",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []PptxElement{},
		Links:         []PptxLink{},
		Relationships: []PptxRelationship{},
	}

	// Create root element
	rootElement := PptxElement{
		ElementID:       p.generateID("root_"),
		ElementType:     "presentation_root",
		ContentPreview:  p.truncateContent(fmt.Sprintf("Presentation: %s", request.ID)),
		ContentLocation: p.createContentLocation(request.ID, "root", ""),
		ContentHash:     p.generateHash(request.Content),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, rootElement)

	// Create presentation body element
	bodyElement := PptxElement{
		ElementID:       p.generateID("body_"),
		ElementType:     "presentation_body",
		ParentID:        rootElement.ElementID,
		ContentPreview:  "Presentation body",
		ContentLocation: p.createContentLocation(request.ID, "body", ""),
		ContentHash:     p.generateHash("body"),
		ElementOrder:    1,
		DocumentOrder:   1,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, bodyElement)

	// Create bidirectional relationships from root to body
	rels := p.createBidirectionalRelationship(rootElement.ElementID, bodyElement.ElementID, make(map[string]interface{}))
	response.Relationships = append(response.Relationships, rels...)

	// Parse presentation structure
	elementCounter := 2
	err = p.parsePresentationStructure(reader, &response.Elements, &response.Links,
		&response.Relationships, bodyElement.ElementID, request.ID, &elementCounter)
	if err != nil {
		// Log error but continue with partial results
		fmt.Printf("Warning: Error parsing presentation structure: %v\n", err)
	}

	// Extract presentation metadata
	p.extractPresentationMetadata(reader, response.Document)

	return response, nil
}

// parsePresentationStructure parses the PPTX presentation structure
func (p *PptxParser) parsePresentationStructure(reader *zip.ReadCloser, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, parentID, sourceID string, counter *int) error {

	// Find and parse slides
	slideFiles := make(map[int]string)
	slidePattern := regexp.MustCompile(`ppt/slides/slide(\d+)\.xml`)

	for _, file := range reader.File {
		if matches := slidePattern.FindStringSubmatch(file.Name); matches != nil {
			if slideNum, err := strconv.Atoi(matches[1]); err == nil {
				slideFiles[slideNum] = file.Name
			}
		}
	}

	// Process slides in order
	for slideNum := 1; slideNum <= len(slideFiles); slideNum++ {
		slidePath, exists := slideFiles[slideNum]
		if !exists {
			continue
		}

		// Read slide XML
		slideXML, err := p.readZipFile(reader, slidePath)
		if err != nil {
			continue
		}

		// Parse slide
		slideID := p.parseSlide(slideXML, slideNum, elements, links, relationships, parentID, sourceID, counter)

		// Check for slide notes
		notesPath := strings.Replace(slidePath, "slides/slide", "notesSlides/notesSlide", 1)
		if notesXML, err := p.readZipFile(reader, notesPath); err == nil && p.ExtractNotes {
			p.parseSlideNotes(notesXML, slideNum, elements, links, relationships,
				parentID, sourceID, counter)
		}

		// Check for slide comments
		if p.ExtractComments {
			commentsPath := fmt.Sprintf("ppt/comments/comment%d.xml", slideNum)
			if commentsXML, err := p.readZipFile(reader, commentsPath); err == nil {
				p.parseSlideComments(commentsXML, slideNum, elements, links, relationships,
					slideID, sourceID, counter)
			}
		}
	}

	return nil
}

// parseSlide parses a single slide and returns the slide ID
func (p *PptxParser) parseSlide(slideXML []byte, slideNum int, elements *[]PptxElement, links *[]PptxLink,
	relationships *[]PptxRelationship, parentID, sourceID string, counter *int) string {

	// Create slide element
	slideID := p.generateID(fmt.Sprintf("slide_%d_", slideNum))
	slideElement := PptxElement{
		ElementID:       slideID,
		ElementType:     "slide",
		ParentID:        parentID,
		ContentPreview:  fmt.Sprintf("Slide %d", slideNum),
		ContentLocation: p.createSlideLocation(sourceID, slideNum),
		ContentHash:     p.generateHash(string(slideXML)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index":  slideNum - 1,
			"slide_number": slideNum,
		},
	}
	*elements = append(*elements, slideElement)
	*counter++

	// Create bidirectional relationships from parent to slide
	rels := p.createBidirectionalRelationship(parentID, slideID, map[string]interface{}{"slide_number": slideNum})
	*relationships = append(*relationships, rels...)

	// Parse slide content
	var slide Slide
	err := xml.Unmarshal(slideXML, &slide)
	if err == nil {
		// Process shapes in the slide
		for shapeIdx, shape := range slide.CSld.SpTree.Shapes {
			p.processShape(&shape, shapeIdx, slideNum, elements, links, relationships,
				slideID, sourceID, counter)
		}

		// Process graphic frames (tables and charts)
		for frameIdx, frame := range slide.CSld.SpTree.GraphicFrames {
			p.processGraphicFrame(&frame, frameIdx, slideNum, elements, links, relationships,
				slideID, sourceID, counter)
		}

		// Process pictures
		if p.ExtractImages {
			for picIdx, picture := range slide.CSld.SpTree.Pictures {
				p.processPicture(&picture, picIdx, slideNum, elements, links, relationships,
					slideID, sourceID, counter)
			}
		}

		// Process group shapes
		for grpIdx, group := range slide.CSld.SpTree.GroupShapes {
			p.processGroupShape(&group, grpIdx, slideNum, elements, links, relationships,
				slideID, sourceID, counter, "")
		}
	}

	// Extract text content for preview
	allText := p.extractSlideText(slideXML)
	if allText != "" {
		slideElement.ContentPreview = p.truncateContent(fmt.Sprintf("Slide %d: %s", slideNum, allText))
	}

	return slideID
}

// processShape processes a shape from a slide
func (p *PptxParser) processShape(shape *Shape, shapeIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	if !p.ExtractShapes {
		return
	}

	// Extract text from shape
	text := p.extractShapeText(shape)
	if text == "" {
		return // Skip empty shapes
	}

	// Determine shape type
	shapeType := "text_box"
	if shape.NvSpPr.CNvPr.Name != "" {
		if strings.Contains(strings.ToLower(shape.NvSpPr.CNvPr.Name), "title") {
			shapeType = "title"
		} else if strings.Contains(strings.ToLower(shape.NvSpPr.CNvPr.Name), "subtitle") {
			shapeType = "subtitle"
		}
	}

	// Create shape element
	shapeElementID := p.generateID(fmt.Sprintf("shape_%d_%d_", slideNum, shapeIdx))

	// Create metadata and extract temporal content
	metadata := map[string]interface{}{
		"slide_index": slideNum - 1,
		"shape_index": shapeIdx,
		"text":        text,
	}
	ProcessTemporalContent(text, metadata)

	shapeElement := PptxElement{
		ElementID:       shapeElementID,
		ElementType:     shapeType,
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(text),
		ContentLocation: p.createShapeLocation(sourceID, slideNum, shapeIdx),
		ContentHash:     p.generateHash(text),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata:        metadata,
	}
	*elements = append(*elements, shapeElement)
	*counter++

	// Create bidirectional relationships from slide to shape
	rels := p.createBidirectionalRelationship(slideID, shapeElementID, map[string]interface{}{"shape_index": shapeIdx})
	*relationships = append(*relationships, rels...)

	// Extract individual paragraphs if there are multiple
	if len(shape.TxBody.Paragraphs) > 1 {
		for paraIdx, para := range shape.TxBody.Paragraphs {
			// Extract text from paragraph
			var paraText string
			for _, run := range para.Runs {
				if strings.TrimSpace(run.Text) != "" {
					paraText += strings.TrimSpace(run.Text) + " "
				}
			}
			paraText = strings.TrimSpace(paraText)
			if paraText == "" {
				continue // Skip empty paragraphs
			}

			// Create paragraph element
			paraID := p.generateID(fmt.Sprintf("para_%d_%d_%d_", slideNum, shapeIdx, paraIdx))

			// Create metadata and extract temporal content
			paraMetadata := map[string]interface{}{
				"slide_index":     slideNum - 1,
				"shape_index":     shapeIdx,
				"paragraph_index": paraIdx,
				"text":            paraText,
			}
			ProcessTemporalContent(paraText, paraMetadata)

			paraElement := PptxElement{
				ElementID:       paraID,
				ElementType:     "paragraph",
				ParentID:        shapeElementID,
				ContentPreview:  p.truncateContent(paraText),
				ContentLocation: p.createParagraphLocation(sourceID, slideNum, shapeIdx, paraIdx),
				ContentHash:     p.generateHash(paraText),
				ElementOrder:    *counter,
				DocumentOrder:   *counter,
				Metadata:        paraMetadata,
			}
			*elements = append(*elements, paraElement)
			*counter++

			// Create bidirectional relationships from shape to paragraph
			paraRels := p.createBidirectionalRelationship(shapeElementID, paraID, map[string]interface{}{"paragraph_index": paraIdx})
			*relationships = append(*relationships, paraRels...)

			// Extract links from paragraph text
			p.extractLinksFromText(paraText, paraID, links)
		}
	} else {
		// Single paragraph - extract links from shape text
		p.extractLinksFromText(text, shapeElementID, links)
	}
}

// processGraphicFrame processes a graphic frame (table or chart)
func (p *PptxParser) processGraphicFrame(frame *GraphicFrame, frameIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	// Check if this is a chart based on URI
	uri := frame.Graphic.GraphicData.URI
	isChart := strings.Contains(uri, "chart") || strings.Contains(uri, "Chart")

	// Get frame name to double-check
	frameName := frame.NvGraphicFramePr.CNvPr.Name
	if strings.Contains(strings.ToLower(frameName), "chart") {
		isChart = true
	}

	if isChart {
		p.processChart(frame, frameIdx, slideNum, elements, links, relationships,
			slideID, sourceID, counter)
	} else {
		// Assume it's a table
		p.processTable(frame, frameIdx, slideNum, elements, links, relationships,
			slideID, sourceID, counter)
	}
}

// processTable processes a table from a graphic frame
func (p *PptxParser) processTable(frame *GraphicFrame, tableIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	table := frame.Graphic.GraphicData.Table
	if len(table.Rows) == 0 {
		return // Empty table
	}

	// Create table element
	tableID := p.generateID(fmt.Sprintf("table_%d_%d_", slideNum, tableIdx))

	// Extract all text from table for preview
	var tableText string
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			cellText := p.extractTextFromTextBody(&cell.TxBody)
			if cellText != "" {
				tableText += cellText + " | "
			}
		}
		tableText += "\n"
	}

	tableElement := PptxElement{
		ElementID:       tableID,
		ElementType:     "table",
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Table: %s", tableText)),
		ContentLocation: p.createTableLocation(sourceID, slideNum, tableIdx),
		ContentHash:     p.generateHash(tableText),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"table_index": tableIdx,
			"rows":        len(table.Rows),
			"cols":        len(table.Rows[0].Cells),
		},
	}
	*elements = append(*elements, tableElement)
	*counter++

	// Create bidirectional relationship from slide to table
	tableRels := p.createBidirectionalRelationship(slideID, tableID, map[string]interface{}{"table_index": tableIdx})
	*relationships = append(*relationships, tableRels...)

	// Process rows
	for rowIdx, row := range table.Rows {
		rowID := p.generateID(fmt.Sprintf("row_%d_%d_%d_", slideNum, tableIdx, rowIdx))

		rowElement := PptxElement{
			ElementID:       rowID,
			ElementType:     "table_row",
			ParentID:        tableID,
			ContentPreview:  fmt.Sprintf("Row %d", rowIdx+1),
			ContentLocation: p.createTableRowLocation(sourceID, slideNum, tableIdx, rowIdx),
			ContentHash:     p.generateHash(fmt.Sprintf("row_%d_%d_%d", slideNum, tableIdx, rowIdx)),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata: map[string]interface{}{
				"slide_index": slideNum - 1,
				"table_index": tableIdx,
				"row_index":   rowIdx,
			},
		}
		*elements = append(*elements, rowElement)
		*counter++

		// Create bidirectional relationship from table to row
		rowRels := p.createBidirectionalRelationship(tableID, rowID, map[string]interface{}{"row_index": rowIdx})
		*relationships = append(*relationships, rowRels...)

		// Process cells
		for cellIdx, cell := range row.Cells {
			cellText := p.extractTextFromTextBody(&cell.TxBody)
			if cellText == "" {
				continue // Skip empty cells
			}

			cellID := p.generateID(fmt.Sprintf("cell_%d_%d_%d_%d_", slideNum, tableIdx, rowIdx, cellIdx))

			// Create metadata and extract temporal content
			cellMetadata := map[string]interface{}{
				"slide_index": slideNum - 1,
				"table_index": tableIdx,
				"row_index":   rowIdx,
				"col_index":   cellIdx,
				"text":        cellText,
			}
			ProcessTemporalContent(cellText, cellMetadata)

			cellElement := PptxElement{
				ElementID:       cellID,
				ElementType:     "table_cell",
				ParentID:        rowID,
				ContentPreview:  p.truncateContent(cellText),
				ContentLocation: p.createTableCellLocation(sourceID, slideNum, tableIdx, rowIdx, cellIdx),
				ContentHash:     p.generateHash(cellText),
				ElementOrder:    *counter,
				DocumentOrder:   *counter,
				Metadata:        cellMetadata,
			}
			*elements = append(*elements, cellElement)
			*counter++

			// Create bidirectional relationship from row to cell
			cellRels := p.createBidirectionalRelationship(rowID, cellID, map[string]interface{}{"col_index": cellIdx})
			*relationships = append(*relationships, cellRels...)

			// Extract links from cell text
			p.extractLinksFromText(cellText, cellID, links)
		}
	}
}

// processChart processes a chart from a graphic frame
func (p *PptxParser) processChart(frame *GraphicFrame, chartIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	// Generate chart element ID
	chartID := p.generateID(fmt.Sprintf("chart_%d_%d_", slideNum, chartIdx))

	// Get chart name
	chartName := frame.NvGraphicFramePr.CNvPr.Name
	if chartName == "" {
		chartName = fmt.Sprintf("Chart %d", chartIdx+1)
	}

	// Create chart element
	chartElement := PptxElement{
		ElementID:       chartID,
		ElementType:     "chart",
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Chart: %s", chartName)),
		ContentLocation: p.createChartLocation(sourceID, slideNum, chartIdx),
		ContentHash:     p.generateHash(fmt.Sprintf("chart_%d_%d", slideNum, chartIdx)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"chart_index": chartIdx,
			"chart_name":  chartName,
			"chart_type":  "chart", // Simplified - full chart parsing would require more XML parsing
		},
	}
	*elements = append(*elements, chartElement)
	*counter++

	// Create bidirectional relationship from slide to chart
	chartRels := p.createBidirectionalRelationship(slideID, chartID, map[string]interface{}{"chart_index": chartIdx})
	*relationships = append(*relationships, chartRels...)
}

// parseSlideComments parses comments for a slide
func (p *PptxParser) parseSlideComments(commentsXML []byte, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	// Parse comments XML
	var commentList CommentList
	err := xml.Unmarshal(commentsXML, &commentList)
	if err != nil || len(commentList.Comments) == 0 {
		return
	}

	// Create comments container element
	commentsContainerID := p.generateID(fmt.Sprintf("comments_%d_", slideNum))
	commentsElement := PptxElement{
		ElementID:       commentsContainerID,
		ElementType:     "comments_container",
		ParentID:        slideID,
		ContentPreview:  fmt.Sprintf("Comments for Slide %d (%d comments)", slideNum, len(commentList.Comments)),
		ContentLocation: p.createCommentsLocation(sourceID, slideNum),
		ContentHash:     p.generateHash(fmt.Sprintf("comments_%d", slideNum)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index":   slideNum - 1,
			"comment_count": len(commentList.Comments),
		},
	}
	*elements = append(*elements, commentsElement)
	*counter++

	// Create bidirectional relationship from slide to comments container
	containerRels := p.createBidirectionalRelationship(slideID, commentsContainerID, make(map[string]interface{}))
	*relationships = append(*relationships, containerRels...)

	// Process individual comments
	for commentIdx, comment := range commentList.Comments {
		if comment.Text == "" {
			continue
		}

		// Generate comment element ID
		commentID := p.generateID(fmt.Sprintf("comment_%d_%d_", slideNum, commentIdx))

		// Create metadata and extract temporal content
		commentMetadata := map[string]interface{}{
			"slide_index":   slideNum - 1,
			"comment_index": commentIdx,
			"text":          comment.Text,
			"author_id":     comment.AuthorID,
			"date_time":     comment.DateTime,
		}
		ProcessTemporalContent(comment.Text, commentMetadata)

		// Create comment element
		commentElement := PptxElement{
			ElementID:       commentID,
			ElementType:     "comment",
			ParentID:        commentsContainerID,
			ContentPreview:  p.truncateContent(comment.Text),
			ContentLocation: p.createCommentLocation(sourceID, slideNum, commentIdx),
			ContentHash:     p.generateHash(fmt.Sprintf("comment_%d_%d_%s", slideNum, commentIdx, comment.Text)),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata:        commentMetadata,
		}
		*elements = append(*elements, commentElement)
		*counter++

		// Create bidirectional relationship from container to comment
		commentRels := p.createBidirectionalRelationship(commentsContainerID, commentID, map[string]interface{}{"comment_index": commentIdx})
		*relationships = append(*relationships, commentRels...)
	}
}

// parseSlideNotes parses notes for a slide
func (p *PptxParser) parseSlideNotes(notesXML []byte, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, parentID, sourceID string, counter *int) {

	// Extract text from notes
	notesText := p.extractNotesText(notesXML)
	if notesText == "" {
		return
	}

	// Find the slide element
	var slideID string
	for _, elem := range *elements {
		if elem.ElementType == "slide" {
			if metadata, ok := elem.Metadata["slide_number"].(int); ok && metadata == slideNum {
				slideID = elem.ElementID
				break
			}
		}
	}

	if slideID == "" {
		return // Slide not found
	}

	// Create notes element
	notesElementID := p.generateID(fmt.Sprintf("notes_%d_", slideNum))

	// Create metadata and extract temporal content
	notesMetadata := map[string]interface{}{
		"slide_index": slideNum - 1,
		"text":        notesText,
	}
	ProcessTemporalContent(notesText, notesMetadata)

	notesElement := PptxElement{
		ElementID:       notesElementID,
		ElementType:     "slide_notes",
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(notesText),
		ContentLocation: p.createNotesLocation(sourceID, slideNum),
		ContentHash:     p.generateHash(notesText),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata:        notesMetadata,
	}
	*elements = append(*elements, notesElement)
	*counter++

	// Create bidirectional relationship from slide to notes
	notesRels := p.createBidirectionalRelationship(slideID, notesElementID, make(map[string]interface{}))
	*relationships = append(*relationships, notesRels...)
}

// processPicture processes a picture shape
func (p *PptxParser) processPicture(picture *Picture, picIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, slideID, sourceID string, counter *int) {

	// Generate picture element ID
	pictureID := p.generateID(fmt.Sprintf("image_%d_%d_", slideNum, picIdx))

	// Get picture name
	pictureName := picture.NvPicPr.CNvPr.Name
	if pictureName == "" {
		pictureName = fmt.Sprintf("Image %d", picIdx+1)
	}

	// Create picture element
	pictureElement := PptxElement{
		ElementID:       pictureID,
		ElementType:     "image",
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Image: %s", pictureName)),
		ContentLocation: p.createPictureLocation(sourceID, slideNum, picIdx),
		ContentHash:     p.generateHash(fmt.Sprintf("image_%d_%d", slideNum, picIdx)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"image_index": picIdx,
			"image_name":  pictureName,
		},
	}
	*elements = append(*elements, pictureElement)
	*counter++

	// Create bidirectional relationship from slide to picture
	pictureRels := p.createBidirectionalRelationship(slideID, pictureID, map[string]interface{}{"image_index": picIdx})
	*relationships = append(*relationships, pictureRels...)
}

// processGroupShape processes a group shape and its children
func (p *PptxParser) processGroupShape(group *GroupShape, grpIdx, slideNum int, elements *[]PptxElement,
	links *[]PptxLink, relationships *[]PptxRelationship, parentID, sourceID string, counter *int, shapePath string) {

	// Generate group element ID
	currentPath := fmt.Sprintf("%s/%d", shapePath, grpIdx)
	if shapePath == "" {
		currentPath = fmt.Sprintf("%d", grpIdx)
	}

	groupID := p.generateID(fmt.Sprintf("group_%d_%s_", slideNum, currentPath))

	// Get group name
	groupName := group.NvGrpSpPr.CNvPr.Name
	if groupName == "" {
		groupName = fmt.Sprintf("Group %s", currentPath)
	}

	// Create group element
	groupElement := PptxElement{
		ElementID:       groupID,
		ElementType:     "shape_group",
		ParentID:        parentID,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Shape Group: %s", groupName)),
		ContentLocation: p.createGroupLocation(sourceID, slideNum, currentPath),
		ContentHash:     p.generateHash(fmt.Sprintf("group_%d_%s", slideNum, currentPath)),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"group_path":  currentPath,
			"group_name":  groupName,
			"shape_count": len(group.Shapes) + len(group.Pictures) + len(group.SubGroups),
		},
	}
	*elements = append(*elements, groupElement)
	*counter++

	// Create bidirectional relationship from parent to group
	groupRels := p.createBidirectionalRelationship(parentID, groupID, map[string]interface{}{"group_index": grpIdx})
	*relationships = append(*relationships, groupRels...)

	// Process shapes in the group
	if p.ExtractShapes {
		for shapeIdx, shape := range group.Shapes {
			p.processShape(&shape, shapeIdx, slideNum, elements, links, relationships,
				groupID, sourceID, counter)
		}
	}

	// Process pictures in the group
	if p.ExtractImages {
		for picIdx, picture := range group.Pictures {
			p.processPicture(&picture, picIdx, slideNum, elements, links, relationships,
				groupID, sourceID, counter)
		}
	}

	// Process sub-groups recursively
	for subGrpIdx, subGroup := range group.SubGroups {
		p.processGroupShape(&subGroup, subGrpIdx, slideNum, elements, links, relationships,
			groupID, sourceID, counter, currentPath)
	}
}

// Helper methods

// extractSlideText extracts all text from a slide XML
func (p *PptxParser) extractSlideText(slideXML []byte) string {
	// Simple regex-based text extraction
	textPattern := regexp.MustCompile(`<a:t[^>]*>([^<]+)</a:t>`)
	matches := textPattern.FindAllStringSubmatch(string(slideXML), -1)

	var texts []string
	for _, match := range matches {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			texts = append(texts, strings.TrimSpace(match[1]))
		}
	}

	return strings.Join(texts, " ")
}

// extractShapeText extracts text from a shape
func (p *PptxParser) extractShapeText(shape *Shape) string {
	var texts []string
	for _, para := range shape.TxBody.Paragraphs {
		for _, run := range para.Runs {
			if strings.TrimSpace(run.Text) != "" {
				texts = append(texts, strings.TrimSpace(run.Text))
			}
		}
	}
	return strings.Join(texts, " ")
}

// extractTextFromTextBody extracts text from a TextBody structure
func (p *PptxParser) extractTextFromTextBody(txBody *TextBody) string {
	var texts []string
	for _, para := range txBody.Paragraphs {
		for _, run := range para.Runs {
			if strings.TrimSpace(run.Text) != "" {
				texts = append(texts, strings.TrimSpace(run.Text))
			}
		}
	}
	return strings.Join(texts, " ")
}

// extractNotesText extracts text from notes XML
func (p *PptxParser) extractNotesText(notesXML []byte) string {
	// Simple regex-based text extraction for notes
	textPattern := regexp.MustCompile(`<a:t[^>]*>([^<]+)</a:t>`)
	matches := textPattern.FindAllStringSubmatch(string(notesXML), -1)

	var texts []string
	for _, match := range matches {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			texts = append(texts, strings.TrimSpace(match[1]))
		}
	}

	return strings.Join(texts, " ")
}

// extractLinksFromText extracts links from text
func (p *PptxParser) extractLinksFromText(text, elementID string, links *[]PptxLink) {
	// Simple URL detection in text
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlRegex.FindAllString(text, -1)

	for _, url := range urls {
		link := PptxLink{
			SourceID:   elementID,
			LinkText:   url,
			LinkTarget: url,
			LinkType:   "url",
		}
		*links = append(*links, link)
	}
}

// extractPresentationMetadata extracts metadata from the presentation
func (p *PptxParser) extractPresentationMetadata(reader *zip.ReadCloser, document map[string]interface{}) {
	// Extract basic presentation statistics
	metadata := document["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
		document["metadata"] = metadata
	}

	// Count slides
	slideCount := 0
	slidePattern := regexp.MustCompile(`ppt/slides/slide\d+\.xml`)
	for _, file := range reader.File {
		if slidePattern.MatchString(file.Name) {
			slideCount++
		}
	}
	metadata["slide_count"] = slideCount

	// Extract core properties if available
	if propsXML, err := p.readZipFile(reader, "docProps/core.xml"); err == nil {
		// Parse core properties
		p.parseCoreProperties(propsXML, metadata)
	}

	// Extract app properties if available
	if appPropsXML, err := p.readZipFile(reader, "docProps/app.xml"); err == nil {
		// Parse app properties
		p.parseAppProperties(appPropsXML, metadata)
	}
}

// parseCoreProperties parses core document properties
func (p *PptxParser) parseCoreProperties(propsXML []byte, metadata map[string]interface{}) {
	// Extract title
	if match := regexp.MustCompile(`<dc:title[^>]*>([^<]+)</dc:title>`).FindStringSubmatch(string(propsXML)); len(match) > 1 {
		metadata["title"] = match[1]
	}

	// Extract creator
	if match := regexp.MustCompile(`<dc:creator[^>]*>([^<]+)</dc:creator>`).FindStringSubmatch(string(propsXML)); len(match) > 1 {
		metadata["author"] = match[1]
	}

	// Extract subject
	if match := regexp.MustCompile(`<dc:subject[^>]*>([^<]+)</dc:subject>`).FindStringSubmatch(string(propsXML)); len(match) > 1 {
		metadata["subject"] = match[1]
	}

	// Extract description
	if match := regexp.MustCompile(`<dc:description[^>]*>([^<]+)</dc:description>`).FindStringSubmatch(string(propsXML)); len(match) > 1 {
		metadata["description"] = match[1]
	}
}

// parseAppProperties parses application properties
func (p *PptxParser) parseAppProperties(appPropsXML []byte, metadata map[string]interface{}) {
	// Extract slides count
	if match := regexp.MustCompile(`<Slides[^>]*>(\d+)</Slides>`).FindStringSubmatch(string(appPropsXML)); len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			metadata["slide_count"] = count
		}
	}

	// Extract words count
	if match := regexp.MustCompile(`<Words[^>]*>(\d+)</Words>`).FindStringSubmatch(string(appPropsXML)); len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			metadata["word_count"] = count
		}
	}

	// Extract application
	if match := regexp.MustCompile(`<Application[^>]*>([^<]+)</Application>`).FindStringSubmatch(string(appPropsXML)); len(match) > 1 {
		metadata["application"] = match[1]
	}

	// Extract presentation format
	if match := regexp.MustCompile(`<PresentationFormat[^>]*>([^<]+)</PresentationFormat>`).FindStringSubmatch(string(appPropsXML)); len(match) > 1 {
		metadata["presentation_format"] = match[1]
	}
}

// readZipFile reads a file from the ZIP archive
func (p *PptxParser) readZipFile(reader *zip.ReadCloser, filename string) ([]byte, error) {
	for _, file := range reader.File {
		if file.Name == filename {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file %s not found in archive", filename)
}

// Content location helpers

// createContentLocation creates a content location object
func (p *PptxParser) createContentLocation(source, elementType, selector string) map[string]interface{} {
	return map[string]interface{}{
		"source":   source,
		"type":     elementType,
		"selector": selector,
	}
}

// createSlideLocation creates content location for a slide
func (p *PptxParser) createSlideLocation(source string, slideNum int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "slide",
		"slide_index": slideNum - 1,
	}
}

// createShapeLocation creates content location for a shape
func (p *PptxParser) createShapeLocation(source string, slideNum, shapeIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "shape",
		"slide_index": slideNum - 1,
		"shape_index": shapeIdx,
		"shape_path":  fmt.Sprintf("%d", shapeIdx),
	}
}

// createParagraphLocation creates content location for a paragraph
func (p *PptxParser) createParagraphLocation(source string, slideNum, shapeIdx, paraIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":          source,
		"type":            "paragraph",
		"slide_index":     slideNum - 1,
		"shape_index":     shapeIdx,
		"paragraph_index": paraIdx,
	}
}

// createTableLocation creates content location for a table
func (p *PptxParser) createTableLocation(source string, slideNum, tableIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "table",
		"slide_index": slideNum - 1,
		"table_index": tableIdx,
	}
}

// createTableRowLocation creates content location for a table row
func (p *PptxParser) createTableRowLocation(source string, slideNum, tableIdx, rowIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "table_row",
		"slide_index": slideNum - 1,
		"table_index": tableIdx,
		"row_index":   rowIdx,
	}
}

// createTableCellLocation creates content location for a table cell
func (p *PptxParser) createTableCellLocation(source string, slideNum, tableIdx, rowIdx, cellIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "table_cell",
		"slide_index": slideNum - 1,
		"table_index": tableIdx,
		"row_index":   rowIdx,
		"col_index":   cellIdx,
	}
}

// createNotesLocation creates content location for slide notes
func (p *PptxParser) createNotesLocation(source string, slideNum int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "slide_notes",
		"slide_index": slideNum - 1,
	}
}

// createPictureLocation creates content location for a picture
func (p *PptxParser) createPictureLocation(source string, slideNum, picIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "image",
		"slide_index": slideNum - 1,
		"image_index": picIdx,
	}
}

// createGroupLocation creates content location for a shape group
func (p *PptxParser) createGroupLocation(source string, slideNum int, groupPath string) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "shape_group",
		"slide_index": slideNum - 1,
		"group_path":  groupPath,
	}
}

// createChartLocation creates content location for a chart
func (p *PptxParser) createChartLocation(source string, slideNum, chartIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "chart",
		"slide_index": slideNum - 1,
		"chart_index": chartIdx,
	}
}

// createCommentsLocation creates content location for comments container
func (p *PptxParser) createCommentsLocation(source string, slideNum int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "comments_container",
		"slide_index": slideNum - 1,
	}
}

// createCommentLocation creates content location for a single comment
func (p *PptxParser) createCommentLocation(source string, slideNum, commentIdx int) map[string]interface{} {
	return map[string]interface{}{
		"source":        source,
		"type":          "comment",
		"slide_index":   slideNum - 1,
		"comment_index": commentIdx,
	}
}

// Utility methods

// generateID generates a unique ID with the given prefix
func (p *PptxParser) generateID(prefix string) string {
	id := make([]byte, 4)
	if _, err := rand.Read(id); err != nil {
		return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s%x", prefix, id)
}

// generateHash generates an MD5 hash of the content
func (p *PptxParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content to the maximum preview length
func (p *PptxParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// ToParseResult converts PptxParseResponse to generic ParseResult with promoted fields
func (r *PptxParseResponse) ToParseResult() *ParseResult {
	// Convert document map to Document struct
	doc := Document{
		ID:       "",
		DocType:  "pptx",
		Metadata: make(map[string]interface{}),
	}
	if docID, ok := r.Document["doc_id"].(string); ok {
		doc.ID = docID
	}
	if title, ok := r.Document["title"].(string); ok {
		doc.Title = title
	}
	if metadata, ok := r.Document["metadata"].(map[string]interface{}); ok {
		doc.Metadata = metadata
	}

	result := &ParseResult{
		Document:      doc,
		Elements:      make([]Element, 0, len(r.Elements)),
		Links:         make([]Link, 0, len(r.Links)),
		Relationships: make([]Relationship, 0, len(r.Relationships)),
	}

	// Convert elements with promoted fields
	for _, pptxElem := range r.Elements {
		elem := Element{
			ElementID:       pptxElem.ElementID,
			ElementType:     pptxElem.ElementType,
			ParentID:        pptxElem.ParentID,
			ContentPreview:  pptxElem.ContentPreview,
			ContentLocation: pptxElem.ContentLocation,
			Position:        pptxElem.ElementOrder,
			Depth:           0, // PPTX doesn't track depth currently
			Metadata:        pptxElem.Metadata,
		}

		// UDML Phase 1: Populate promoted fields

		// PageNumber - from slide_number or slide_index + 1 (PPTX slides map to pages)
		if slideNumber, ok := pptxElem.Metadata["slide_number"].(int); ok {
			elem.PageNumber = &slideNumber
		} else if slideIndex, ok := pptxElem.Metadata["slide_index"].(int); ok {
			slideNumber := slideIndex + 1
			elem.PageNumber = &slideNumber
		}

		// RowIndex and ColumnIndex - for table cells
		if pptxElem.ElementType == "table_cell" {
			if row, ok := pptxElem.Metadata["row_index"].(int); ok {
				elem.RowIndex = &row
			}
			if col, ok := pptxElem.Metadata["col_index"].(int); ok {
				elem.ColumnIndex = &col
			}
		}

		// TemporalType - from temporal metadata
		if temporalType, ok := pptxElem.Metadata["temporal_type"].(string); ok && temporalType != "" {
			elem.TemporalType = &temporalType
		}

		// ElementCategory - use taxonomy
		elem.ElementCategory = GetElementCategory(pptxElem.ElementType)

		result.Elements = append(result.Elements, elem)
	}

	// Convert links
	for _, pptxLink := range r.Links {
		link := Link{
			LinkID:          generateID("link"),
			SourceElementID: pptxLink.SourceID,
			LinkTarget:      pptxLink.LinkTarget,
			LinkText:        pptxLink.LinkText,
			LinkType:        pptxLink.LinkType,
		}
		result.Links = append(result.Links, link)
	}

	// Convert relationships
	for _, pptxRel := range r.Relationships {
		rel := Relationship{
			RelationshipID:   pptxRel.RelationshipID,
			SourceElementID:  pptxRel.SourceElementID,
			TargetElementID:  pptxRel.TargetElementID,
			RelationshipType: pptxRel.RelationshipType,
			Confidence:       pptxRel.Confidence,
			Metadata:         pptxRel.Metadata,
		}
		result.Relationships = append(result.Relationships, rel)
	}

	return result
}

// ToJSON converts the parse response to JSON
func (r *PptxParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a PptxParseRequest from JSON
func (r *PptxParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// createBidirectionalRelationship creates both "contains" and "contained_by" relationships
func (p *PptxParser) createBidirectionalRelationship(parentID, childID string, metadata map[string]interface{}) []PptxRelationship {
	containsRel := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  parentID,
		TargetElementID:  childID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         metadata,
	}
	containedByRel := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  childID,
		TargetElementID:  parentID,
		RelationshipType: "contained_by",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	return []PptxRelationship{containsRel, containedByRel}
}
