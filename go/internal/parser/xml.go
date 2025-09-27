package parser

import (
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// XMLElementType represents the type of XML element
type XMLElementType string

const (
	XMLElementTypeRoot    XMLElementType = "document_root"
	XMLElementTypeElement XMLElementType = "xml_element"
	XMLElementTypeText    XMLElementType = "xml_text"
	XMLElementTypeList    XMLElementType = "xml_list"
	XMLElementTypeObject  XMLElementType = "xml_object"
)

// XMLElement represents a parsed XML element
type XMLElement struct {
	ElementID       string                 `json:"element_id"`
	DocID          string                 `json:"doc_id"`
	ElementType     XMLElementType         `json:"element_type"`
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

// XMLLink represents an extracted link
type XMLLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// XMLRelationship represents a relationship between elements
type XMLRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// XMLParseRequest represents the input for XML parsing
type XMLParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// XMLParseResponse represents the output of XML parsing
type XMLParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []XMLElement           `json:"elements"`
	Links         []XMLLink              `json:"links"`
	Relationships []XMLRelationship      `json:"relationships"`
	Dates         map[string]interface{} `json:"dates,omitempty"`
}

// XMLParser handles XML document parsing
type XMLParser struct {
	MaxContentPreview      int
	ExtractAttributes      bool
	FlattenNamespaces      bool
	ExtractNamespaces      bool
	MaxDepth               int
	EnableCaching          bool
}

// NewXMLParser creates a new XML parser instance
func NewXMLParser() *XMLParser {
	return &XMLParser{
		MaxContentPreview:      100,
		ExtractAttributes:      true,
		FlattenNamespaces:      true,
		ExtractNamespaces:      true,
		MaxDepth:               20,
		EnableCaching:          true,
	}
}

// Parse parses an XML document into structured elements
func (p *XMLParser) Parse(request XMLParseRequest) (*XMLParseResponse, error) {
	// Initialize response
	response := &XMLParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "xml",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []XMLElement{},
		Links:         []XMLLink{},
		Relationships: []XMLRelationship{},
	}

	// Create root element
	rootElement := XMLElement{
		ElementID:       p.generateID("root_"),
		DocID:          request.ID,
		ElementType:     XMLElementTypeRoot,
		ContentPreview:  p.truncateContent(fmt.Sprintf("Document: %s", request.ID)),
		ContentLocation: p.createContentLocation(request.ID, XMLElementTypeRoot, "/"),
		ContentHash:     p.generateHash(request.ID),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata: map[string]interface{}{
			"source_id": request.ID,
			"path":      "/",
		},
	}
	response.Elements = append(response.Elements, rootElement)

	// Parse XML content
	decoder := xml.NewDecoder(strings.NewReader(request.Content))
	elementCounter := 1
	elementStack := []string{rootElement.ElementID}
	pathStack := []string{"/"}

	// Track namespaces
	namespaces := make(map[string]string)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Process start element
			elementID := p.generateID("elem_")
			path := p.buildPath(pathStack, t.Name.Local)
			parentID := ""
			if len(elementStack) > 0 {
				parentID = elementStack[len(elementStack)-1]
			}

			// Determine element type based on structure
			elementType := p.determineElementType(t)

			// Create element
			element := XMLElement{
				ElementID:       elementID,
				DocID:          request.ID,
				ElementType:     elementType,
				ParentID:        parentID,
				ContentPreview:  p.formatElementPreview(t),
				ContentLocation: p.createXMLContentLocation(request.ID, elementType, path, t.Name.Space),
				ContentHash:     p.generateHash(fmt.Sprintf("%s_%s", path, t.Name.Local)),
				ElementOrder:    elementCounter,
				DocumentOrder:   elementCounter,
				Metadata: map[string]interface{}{
					"tag_name":  t.Name.Local,
					"xml_path":  path,
					"namespace": t.Name.Space,
				},
			}

			// Extract attributes
			if p.ExtractAttributes && len(t.Attr) > 0 {
				attributes := make(map[string]string)
				for _, attr := range t.Attr {
					attributes[attr.Name.Local] = attr.Value

					// Extract namespace declarations
					if p.ExtractNamespaces && strings.HasPrefix(attr.Name.Local, "xmlns") {
						if attr.Name.Local == "xmlns" {
							namespaces[""] = attr.Value
						} else {
							prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
							namespaces[prefix] = attr.Value
						}
					}
				}
				element.Metadata["attributes"] = attributes
			}

			response.Elements = append(response.Elements, element)
			elementCounter++

			// Create parent-child relationship
			if parentID != "" {
				relationship := XMLRelationship{
					RelationshipID:   p.generateID("rel_"),
					SourceElementID:  parentID,
					TargetElementID:  elementID,
					RelationshipType: "contains",
					Confidence:       1.0,
					Metadata:         make(map[string]interface{}),
				}
				response.Relationships = append(response.Relationships, relationship)
			}

			// Update stacks
			elementStack = append(elementStack, elementID)
			pathStack = append(pathStack, path)

		case xml.EndElement:
			// Pop from stacks
			if len(elementStack) > 0 {
				elementStack = elementStack[:len(elementStack)-1]
			}
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}

		case xml.CharData:
			// Process text content
			text := strings.TrimSpace(string(t))
			if text != "" && len(elementStack) > 0 {
				textID := p.generateID("text_")
				parentID := elementStack[len(elementStack)-1]
				path := ""
				if len(pathStack) > 0 {
					path = pathStack[len(pathStack)-1]
				}

				textElement := XMLElement{
					ElementID:       textID,
					DocID:          request.ID,
					ElementType:     XMLElementTypeText,
					ParentID:        parentID,
					ContentPreview:  p.truncateContent(text),
					ContentLocation: p.createContentLocation(request.ID, XMLElementTypeText, path),
					ContentHash:     p.generateHash(text),
					ElementOrder:    elementCounter,
					DocumentOrder:   elementCounter,
					Text:            text,
					Content:         text,
					Metadata: map[string]interface{}{
						"xml_path": path,
						"is_text":  true,
					},
				}

				response.Elements = append(response.Elements, textElement)
				elementCounter++

				// Create relationship
				relationship := XMLRelationship{
					RelationshipID:   p.generateID("rel_"),
					SourceElementID:  parentID,
					TargetElementID:  textID,
					RelationshipType: "contains",
					Confidence:       1.0,
					Metadata:         make(map[string]interface{}),
				}
				response.Relationships = append(response.Relationships, relationship)

				// Extract links from text
				p.extractLinksFromText(text, textID, &response.Links)
			}
		}
	}

	// Store namespaces in document metadata
	if len(namespaces) > 0 {
		response.Document["namespaces"] = namespaces
	}

	return response, nil
}

// Helper functions

func (p *XMLParser) determineElementType(element xml.StartElement) XMLElementType {
	// Simple heuristic: elements with many attributes might be objects
	// Elements with no attributes are likely simple elements
	// This can be enhanced with more sophisticated logic
	if len(element.Attr) > 2 {
		return XMLElementTypeObject
	}
	return XMLElementTypeElement
}

func (p *XMLParser) buildPath(pathStack []string, elementName string) string {
	if len(pathStack) == 0 {
		return "/" + elementName
	}
	lastPath := pathStack[len(pathStack)-1]
	if lastPath == "/" {
		return "/" + elementName
	}
	return lastPath + "/" + elementName
}

func (p *XMLParser) formatElementPreview(element xml.StartElement) string {
	preview := fmt.Sprintf("<%s", element.Name.Local)

	// Add first few attributes to preview
	attrCount := 0
	for _, attr := range element.Attr {
		if attrCount >= 2 {
			preview += " ..."
			break
		}
		preview += fmt.Sprintf(" %s=\"%s\"", attr.Name.Local, p.truncateContent(attr.Value))
		attrCount++
	}

	preview += ">"
	return p.truncateContent(preview)
}

func (p *XMLParser) createXMLContentLocation(source string, elementType XMLElementType, path string, namespace string) map[string]interface{} {
	location := map[string]interface{}{
		"source": source,
		"type":   string(elementType),
		"path":   path,
	}
	if namespace != "" {
		location["namespace"] = namespace
	}
	return location
}

func (p *XMLParser) createContentLocation(source string, elementType XMLElementType, path string) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   string(elementType),
		"path":   path,
	}
}

func (p *XMLParser) extractLinksFromText(text string, sourceID string, links *[]XMLLink) {
	// Extract URLs
	urlRegex := regexp.MustCompile(`https?://[^\s"'<>]+`)
	urlMatches := urlRegex.FindAllString(text, -1)
	for _, match := range urlMatches {
		link := XMLLink{
			SourceID:   sourceID,
			LinkText:   match,
			LinkTarget: match,
			LinkType:   "url",
		}
		*links = append(*links, link)
	}

	// Extract email addresses
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emailMatches := emailRegex.FindAllString(text, -1)
	for _, match := range emailMatches {
		link := XMLLink{
			SourceID:   sourceID,
			LinkText:   match,
			LinkTarget: "mailto:" + match,
			LinkType:   "email",
		}
		*links = append(*links, link)
	}
}

func (p *XMLParser) generateID(prefix string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s%d", prefix, timestamp%1000000)
}

func (p *XMLParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (p *XMLParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// ToJSON converts the parse response to JSON
func (r *XMLParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a ParseRequest from JSON
func (r *XMLParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}