package parser

import (
	"archive/zip"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
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
	ID   string `xml:"id,attr"`
	RelID string `xml:"id,attr"`
}

type Slide struct {
	XMLName xml.Name         `xml:"sld"`
	CSld    CommonSlideData  `xml:"cSld"`
}

type CommonSlideData struct {
	SpTree ShapeTree `xml:"spTree"`
}

type ShapeTree struct {
	Shapes      []Shape      `xml:"sp"`
	GraphicFrames []GraphicFrame `xml:"graphicFrame"`
}

// GraphicFrame can contain tables
type GraphicFrame struct {
	Graphic Graphic `xml:"graphic"`
}

type Graphic struct {
	GraphicData GraphicData `xml:"graphicData"`
}

type GraphicData struct {
	Table PptxTable `xml:"tbl"`
}

type PptxTable struct {
	Rows []PptxTableRow `xml:"tr"`
}

type PptxTableRow struct {
	Cells []PptxTableCell `xml:"tc"`
}

type PptxTableCell struct {
	TxBody TextBody    `xml:"txBody"`
	TcPr   PptxTcPr    `xml:"tcPr"`
}

type PptxTcPr struct {
	GridSpan   int  `xml:"gridSpan,attr"`
	RowSpan    int  `xml:"rowSpan,attr"`
	HMerge     bool `xml:"hMerge,attr"`
	VMerge     bool `xml:"vMerge,attr"`
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

// PptxRelationships structure for parsing relationships
type PptxRelationships struct {
	XMLName       xml.Name         `xml:"Relationships"`
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

// Parse parses a PPTX document into structured elements
func (p *PptxParser) Parse(request PptxParseRequest) (*PptxParseResponse, error) {
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

	// Create relationship from root to body
	relationship := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  rootElement.ElementID,
		TargetElementID:  bodyElement.ElementID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	response.Relationships = append(response.Relationships, relationship)

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
		p.parseSlide(slideXML, slideNum, elements, links, relationships, parentID, sourceID, counter)

		// Check for slide notes
		notesPath := strings.Replace(slidePath, "slides/slide", "notesSlides/notesSlide", 1)
		if notesXML, err := p.readZipFile(reader, notesPath); err == nil && p.ExtractNotes {
			p.parseSlideNotes(notesXML, slideNum, elements, links, relationships,
				parentID, sourceID, counter)
		}
	}

	return nil
}

// parseSlide parses a single slide
func (p *PptxParser) parseSlide(slideXML []byte, slideNum int, elements *[]PptxElement, links *[]PptxLink,
	relationships *[]PptxRelationship, parentID, sourceID string, counter *int) {

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
			"slide_index": slideNum - 1,
			"slide_number": slideNum,
		},
	}
	*elements = append(*elements, slideElement)
	*counter++

	// Create relationship from parent to slide
	relationship := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  parentID,
		TargetElementID:  slideID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"slide_number": slideNum},
	}
	*relationships = append(*relationships, relationship)

	// Parse slide content
	var slide Slide
	err := xml.Unmarshal(slideXML, &slide)
	if err == nil {
		// Process shapes in the slide
		for shapeIdx, shape := range slide.CSld.SpTree.Shapes {
			p.processShape(&shape, shapeIdx, slideNum, elements, links, relationships,
				slideID, sourceID, counter)
		}
	}

	// Extract text content for preview
	allText := p.extractSlideText(slideXML)
	if allText != "" {
		slideElement.ContentPreview = p.truncateContent(fmt.Sprintf("Slide %d: %s", slideNum, allText))
	}
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
	shapeElement := PptxElement{
		ElementID:       shapeElementID,
		ElementType:     shapeType,
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(text),
		ContentLocation: p.createShapeLocation(sourceID, slideNum, shapeIdx),
		ContentHash:     p.generateHash(text),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"shape_index": shapeIdx,
			"text":        text,
		},
	}
	*elements = append(*elements, shapeElement)
	*counter++

	// Create relationship from slide to shape
	relationship := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  slideID,
		TargetElementID:  shapeElementID,
		RelationshipType: "contains",
		Confidence:       1.0,
		Metadata:         map[string]interface{}{"shape_index": shapeIdx},
	}
	*relationships = append(*relationships, relationship)

	// Extract links from text
	p.extractLinksFromText(text, shapeElementID, links)
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
	notesElement := PptxElement{
		ElementID:       notesElementID,
		ElementType:     "slide_notes",
		ParentID:        slideID,
		ContentPreview:  p.truncateContent(notesText),
		ContentLocation: p.createNotesLocation(sourceID, slideNum),
		ContentHash:     p.generateHash(notesText),
		ElementOrder:    *counter,
		DocumentOrder:   *counter,
		Metadata: map[string]interface{}{
			"slide_index": slideNum - 1,
			"text":        notesText,
		},
	}
	*elements = append(*elements, notesElement)
	*counter++

	// Create relationship from slide to notes
	relationship := PptxRelationship{
		RelationshipID:   p.generateID("rel_"),
		SourceElementID:  slideID,
		TargetElementID:  notesElementID,
		RelationshipType: "has_notes",
		Confidence:       1.0,
		Metadata:         make(map[string]interface{}),
	}
	*relationships = append(*relationships, relationship)
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

// createNotesLocation creates content location for slide notes
func (p *PptxParser) createNotesLocation(source string, slideNum int) map[string]interface{} {
	return map[string]interface{}{
		"source":      source,
		"type":        "slide_notes",
		"slide_index": slideNum - 1,
	}
}

// Utility methods

// generateID generates a unique ID with the given prefix
func (p *PptxParser) generateID(prefix string) string {
	// Simple counter-based approach for now
	return fmt.Sprintf("%s%d", prefix, len(prefix)*1000+strings.Count(prefix, "_"))
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