package parser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// Use string literals for element and relationship types to match universal model

// HTMLElement represents a parsed HTML element
type HTMLElement struct {
	ElementID       string                 `json:"element_id"`
	ElementType     string                 `json:"element_type"`
	ParentID        string                 `json:"parent_id,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ElementOrder    int                    `json:"element_order"`
	DocumentOrder   int                    `json:"document_position"`
	Metadata        map[string]interface{} `json:"metadata"`
	Text            string                 `json:"text,omitempty"`
	Content         string                 `json:"content,omitempty"`
}

// HTMLLink represents an extracted link
type HTMLLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// HTMLRelationship represents a relationship between elements
type HTMLRelationship struct {
	RelationshipID   string `json:"relationship_id"`
	SourceElementID  string `json:"source_element_id"`
	TargetElementID  string `json:"target_element_id"`
	RelationshipType string `json:"relationship_type"`
	Confidence       float64 `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ParseRequest represents the input for HTML parsing
type ParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ParseResponse represents the output of HTML parsing
type ParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []HTMLElement          `json:"elements"`
	Links         []HTMLLink             `json:"links"`
	Relationships []HTMLRelationship     `json:"relationships"`
	Dates         map[string]interface{} `json:"dates,omitempty"`
}

// HTMLParser handles HTML document parsing
type HTMLParser struct {
	MaxContentPreview int
	ExtractDates      bool
	EnableCaching     bool
}

// NewHTMLParser creates a new HTML parser instance
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{
		MaxContentPreview: 100,
		ExtractDates:      true,
		EnableCaching:     true,
	}
}

// Parse parses an HTML document into structured elements
func (p *HTMLParser) Parse(request ParseRequest) (*ParseResponse, error) {
	// Parse HTML content
	doc, err := html.Parse(strings.NewReader(request.Content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Initialize response
	response := &ParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "html",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []HTMLElement{},
		Links:         []HTMLLink{},
		Relationships: []HTMLRelationship{},
	}

	// Create root element
	rootElement := HTMLElement{
		ElementID:       p.generateID("root_"),
		ElementType:     "root",
		ContentPreview:  p.truncateContent(request.Content),
		ContentLocation: p.createContentLocation(request.ID, "root", ""),
		ContentHash:     p.generateHash(request.Content),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, rootElement)

	// Parse document elements
	elementCounter := 1
	p.parseNode(doc, &response.Elements, &response.Links, &response.Relationships,
		rootElement.ElementID, request.ID, &elementCounter)

	// Extract links from anchor tags
	p.extractLinks(doc, &response.Links, response.Elements)

	return response, nil
}

// parseNode recursively parses HTML nodes
func (p *HTMLParser) parseNode(node *html.Node, elements *[]HTMLElement, links *[]HTMLLink,
	relationships *[]HTMLRelationship, parentID, sourceID string, counter *int) {

	if node.Type == html.DocumentNode {
		// Document node - recurse into children without creating element
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			p.parseNode(child, elements, links, relationships, parentID, sourceID, counter)
		}
	} else if node.Type == html.ElementNode {
		elementType := p.getElementType(node.Data)

		// Skip script, style, and other non-content elements
		if p.shouldSkipElement(node.Data) {
			return
		}

		// Create element
		element := HTMLElement{
			ElementID:       p.generateID(fmt.Sprintf("%s_", strings.ToLower(string(elementType)))),
			ElementType:     elementType,
			ParentID:        parentID,
			ContentPreview:  p.extractTextContent(node),
			ContentLocation: p.createContentLocationForNode(sourceID, elementType, node),
			ContentHash:     p.generateHash(p.extractTextContent(node)),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata:        p.extractMetadata(node),
		}

		*elements = append(*elements, element)
		*counter++

		// Create parent-child relationship
		if parentID != "" {
			relationship := HTMLRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  parentID,
				TargetElementID:  element.ElementID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			*relationships = append(*relationships, relationship)
		}

		// Recursively parse children
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			p.parseNode(child, elements, links, relationships, element.ElementID, sourceID, counter)
		}
	} else if node.Type == html.TextNode {
		// Handle text nodes with meaningful content
		text := strings.TrimSpace(node.Data)
		if text != "" && len(text) > 1 {
			element := HTMLElement{
				ElementID:       p.generateID("text_"),
				ElementType:     "text",
				ParentID:        parentID,
				ContentPreview:  p.truncateContent(text),
				ContentLocation: p.createContentLocation(sourceID, "text", ""),
				ContentHash:     p.generateHash(text),
				ElementOrder:    *counter,
				DocumentOrder:   *counter,
				Metadata:        make(map[string]interface{}),
				Text:            text,
			}

			*elements = append(*elements, element)
			*counter++

			// Create parent-child relationship
			if parentID != "" {
				relationship := HTMLRelationship{
					RelationshipID:   p.generateID("rel_"),
					SourceElementID:  parentID,
					TargetElementID:  element.ElementID,
					RelationshipType: "contains",
					Confidence:       1.0,
					Metadata:         make(map[string]interface{}),
				}
				*relationships = append(*relationships, relationship)
			}
		}
	}
}

// getElementType maps HTML tag names to element types
func (p *HTMLParser) getElementType(tagName string) string {
	switch strings.ToLower(tagName) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return "header"
	case "p":
		return "paragraph"
	case "ul", "ol":
		return "list"
	case "li":
		return "list_item"
	case "table":
		return "table"
	case "tr":
		return "table_row"
	case "th":
		return "table_header"
	case "td":
		return "table_cell"
	case "img":
		return "image"
	case "pre", "code":
		return "code_block"
	case "blockquote":
		return "blockquote"
	case "div":
		return "div"
	case "article":
		return "article"
	case "section":
		return "section"
	case "span":
		return "span"
	case "body":
		return "body"
	default:
		// Handle namespace-prefixed elements (XBRL, iXBRL, etc.)
		if strings.Contains(tagName, ":") {
			return strings.ReplaceAll(tagName, ":", "_")
		}
		return tagName
	}
}

// shouldSkipElement determines if an element should be skipped during parsing
func (p *HTMLParser) shouldSkipElement(tagName string) bool {
	skipTags := map[string]bool{
		"script": true,
		"style":  true,
		"noscript": true,
		"meta":   true,
		"link":   true,
		"title":  true,
	}
	return skipTags[strings.ToLower(tagName)]
}

// extractTextContent extracts text content from an HTML node
func (p *HTMLParser) extractTextContent(node *html.Node) string {
	var text strings.Builder
	p.extractTextRecursive(node, &text)
	content := strings.TrimSpace(text.String())
	return p.truncateContent(content)
}

// extractTextRecursive recursively extracts text from HTML nodes
func (p *HTMLParser) extractTextRecursive(node *html.Node, text *strings.Builder) {
	if node.Type == html.TextNode {
		textContent := strings.TrimSpace(node.Data)
		if textContent != "" {
			if text.Len() > 0 {
				text.WriteString(" ")
			}
			text.WriteString(textContent)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.extractTextRecursive(child, text)
	}
}

// extractLinks extracts anchor links from the HTML document
func (p *HTMLParser) extractLinks(doc *html.Node, links *[]HTMLLink, elements []HTMLElement) {
	p.extractLinksRecursive(doc, links, elements)
}

// extractLinksRecursive recursively extracts links from HTML nodes
func (p *HTMLParser) extractLinksRecursive(node *html.Node, links *[]HTMLLink, elements []HTMLElement) {
	if node.Type == html.ElementNode && node.Data == "a" {
		// Find href attribute
		var href string
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				href = attr.Val
				break
			}
		}

		if href != "" {
			linkText := p.extractTextContent(node)
			link := HTMLLink{
				SourceID:   elements[0].ElementID, // Will be updated later to correct element
				LinkText:   linkText,
				LinkTarget: href,
				LinkType:   "html",
			}
			*links = append(*links, link)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.extractLinksRecursive(child, links, elements)
	}
}

// extractMetadata extracts metadata from HTML attributes
func (p *HTMLParser) extractMetadata(node *html.Node) map[string]interface{} {
	metadata := make(map[string]interface{})

	for _, attr := range node.Attr {
		// Store important attributes
		switch attr.Key {
		case "id", "class", "colspan", "rowspan":
			metadata[attr.Key] = attr.Val
		default:
			if strings.HasPrefix(attr.Key, "data-") {
				metadata[attr.Key] = attr.Val
			}
		}
	}

	// Set default values for table cells if not present
	if node.Data == "td" || node.Data == "th" {
		if _, exists := metadata["colspan"]; !exists {
			metadata["colspan"] = "1"
		}
		if _, exists := metadata["rowspan"]; !exists {
			metadata["rowspan"] = "1"
		}
	}

	return metadata
}

// createContentLocation creates a content location object
func (p *HTMLParser) createContentLocation(source string, elementType string, selector string) map[string]interface{} {
	return map[string]interface{}{
		"source":   source,
		"type":     elementType,
		"selector": selector,
	}
}

// createContentLocationForNode creates a content location for a specific node
func (p *HTMLParser) createContentLocationForNode(source string, elementType string, node *html.Node) map[string]interface{} {
	selector := p.generateCSSSelector(node)
	return p.createContentLocation(source, elementType, selector)
}

// generateCSSSelector generates a CSS selector for an HTML node
func (p *HTMLParser) generateCSSSelector(node *html.Node) string {
	if node == nil || node.Type != html.ElementNode {
		return ""
	}

	var parts []string
	current := node

	for current != nil && current.Type == html.ElementNode {
		part := current.Data

		// Add ID if present
		for _, attr := range current.Attr {
			if attr.Key == "id" && attr.Val != "" {
				part = fmt.Sprintf("%s#%s", part, attr.Val)
				parts = append([]string{part}, parts...)
				// ID is unique, so we can stop here
				return strings.Join(parts, " > ")
			}
		}

		// Add classes if present
		var classes []string
		for _, attr := range current.Attr {
			if attr.Key == "class" && attr.Val != "" {
				classNames := strings.Fields(attr.Val)
				for _, className := range classNames {
					if className != "" {
						classes = append(classes, "."+className)
					}
				}
			}
		}
		if len(classes) > 0 {
			part = part + strings.Join(classes, "")
		}

		// Add nth-of-type if needed
		if current.Parent != nil {
			position := p.getNthOfType(current)
			if position > 1 {
				part = fmt.Sprintf("%s:nth-of-type(%d)", part, position)
			}
		}

		parts = append([]string{part}, parts...)
		current = current.Parent
	}

	return strings.Join(parts, " > ")
}

// getNthOfType calculates the nth-of-type position for an element
func (p *HTMLParser) getNthOfType(node *html.Node) int {
	if node.Parent == nil {
		return 1
	}

	position := 1
	for sibling := node.Parent.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		if sibling == node {
			break
		}
		if sibling.Type == html.ElementNode && sibling.Data == node.Data {
			position++
		}
	}

	return position
}

// Helper functions

// generateID generates a unique ID with the given prefix
func (p *HTMLParser) generateID(prefix string) string {
	// Use a simple counter-based approach for now
	// In production, you might want to use UUID or similar
	return fmt.Sprintf("%s%d", prefix, len(prefix)*1000+int(strings.Count(prefix, "_")))
}

// generateHash generates an MD5 hash of the content
func (p *HTMLParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content to the maximum preview length
func (p *HTMLParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

// ToJSON converts the parse response to JSON
func (r *ParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a ParseRequest from JSON
func (r *ParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}