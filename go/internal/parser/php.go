package parser

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// PHPElementType represents the type of PHP element
type PHPElementType string

const (
	PHPElementTypeRoot         PHPElementType = "root"
	PHPElementTypeClass        PHPElementType = "class"
	PHPElementTypeInterface    PHPElementType = "interface"
	PHPElementTypeTrait        PHPElementType = "trait"
	PHPElementTypeFunction     PHPElementType = "function"
	PHPElementTypeMethod       PHPElementType = "method"
	PHPElementTypeProperty     PHPElementType = "property"
	PHPElementTypeConstant     PHPElementType = "constant"
	PHPElementTypeNamespace    PHPElementType = "namespace"
	PHPElementTypeUseStatement PHPElementType = "use_statement"
	PHPElementTypeComment      PHPElementType = "comment"
	PHPElementTypeDocBlock     PHPElementType = "doc_block"
	PHPElementTypeVariable     PHPElementType = "variable"
	PHPElementTypeCodeBlock    PHPElementType = "code_block"
)

// String returns the string representation of PHPElementType
func (t PHPElementType) String() string {
	return string(t)
}

// PHPElement represents a parsed PHP element
type PHPElement struct {
	ElementID       string                 `json:"element_id"`
	DocumentID      string                 `json:"doc_id"`
	ElementType     PHPElementType         `json:"element_type"`
	Text            string                 `json:"text,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ParentID        string                 `json:"parent_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// PHPParseRequest represents a request to parse PHP content
type PHPParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// PHPParseResponse represents the response from parsing PHP content
type PHPParseResponse struct {
	Document map[string]interface{} `json:"document"`
	Elements []PHPElement           `json:"elements"`
}

// ToJSON converts the response to JSON string
func (r *PHPParseResponse) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response to JSON: %w", err)
	}
	return string(data), nil
}

// PHPParser handles parsing of PHP documents
type PHPParser struct {
	MaxContentPreview     int
	ExtractComments       bool
	ExtractDocBlocks      bool
	MaxElements           int
	StripWhitespace       bool
	ElementIDCounter      int
	RelationshipIDCounter int
}

// NewPHPParser creates a new PHP parser with default settings
func NewPHPParser() *PHPParser {
	return &PHPParser{
		MaxContentPreview:     100,
		ExtractComments:       true,
		ExtractDocBlocks:      true,
		MaxElements:           1000,
		StripWhitespace:       true,
		ElementIDCounter:      0,
		RelationshipIDCounter: 0,
	}
}

// Parser interface implementation

// GetName returns the parser name
func (p *PHPParser) GetName() string {
	return "php"
}

// GetSupportedFormats returns supported file formats
func (p *PHPParser) GetSupportedFormats() []string {
	return []string{".php", "php"}
}

// Parse implements the Parser interface for PHP documents
func (p *PHPParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract content from request
	var phpContent string
	switch v := req.Content.(type) {
	case string:
		phpContent = v
	case []byte:
		phpContent = string(v)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", req.Content)
	}

	// Create PHP-specific request
	phpRequest := PHPParseRequest{
		ID:       req.ID,
		Content:  phpContent,
		Metadata: req.Metadata,
	}

	// Parse using existing implementation
	response, err := p.parsePHP(phpRequest)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult
	return p.convertToParseResult(response), nil
}

// SupportsStreaming returns whether the parser supports streaming
func (p *PHPParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *PHPParser) Close() error {
	return nil
}

// parsePHP parses PHP content and returns structured elements (internal method)
func (p *PHPParser) parsePHP(request PHPParseRequest) (*PHPParseResponse, error) {
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

	response := &PHPParseResponse{
		Document: make(map[string]interface{}),
		Elements: []PHPElement{},
	}

	// Create document metadata
	p.createDocumentMetadata(response, request)

	// Create root element
	rootElement := p.createRootElement(request.ID, content)
	response.Elements = append(response.Elements, rootElement)

	// Parse PHP content into elements
	lines := strings.Split(content, "\n")
	p.parsePHPContent(lines, request.ID, rootElement.ElementID, response)

	return response, nil
}

// createDocumentMetadata creates document-level metadata
func (p *PHPParser) createDocumentMetadata(response *PHPParseResponse, request PHPParseRequest) {
	content := request.Content

	// Calculate basic statistics
	charCount := utf8.RuneCountInString(content)
	wordCount := len(strings.Fields(content))
	lineCount := strings.Count(content, "\n") + 1

	// Count different element types
	classCount := len(regexp.MustCompile(`(?m)^\s*(?:abstract\s+|final\s+)?class\s+\w+`).FindAllString(content, -1))
	interfaceCount := len(regexp.MustCompile(`(?m)^\s*interface\s+\w+`).FindAllString(content, -1))
	traitCount := len(regexp.MustCompile(`(?m)^\s*trait\s+\w+`).FindAllString(content, -1))
	functionCount := len(regexp.MustCompile(`(?m)^\s*function\s+\w+`).FindAllString(content, -1))
	methodCount := len(regexp.MustCompile(`(?m)^\s*(?:public|private|protected)\s+function\s+\w+`).FindAllString(content, -1))

	metadata := map[string]interface{}{
		"character_count": charCount,
		"word_count":      wordCount,
		"line_count":      lineCount,
		"class_count":     classCount,
		"interface_count": interfaceCount,
		"trait_count":     traitCount,
		"function_count":  functionCount,
		"method_count":    methodCount,
		"source":          request.Metadata["source"],
		"filename":        request.Metadata["filename"],
		"content_type":    "text/x-php",
		"parsing_method":  "go_php_parser",
	}

	response.Document = map[string]interface{}{
		"doc_id":   request.ID,
		"doc_type": "php",
		"metadata": metadata,
	}
}

// createRootElement creates the root document element
func (p *PHPParser) createRootElement(docID, content string) PHPElement {
	elementID := p.generateElementID("php_root")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeRoot,
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

// parsePHPContent parses PHP content into elements
func (p *PHPParser) parsePHPContent(lines []string, docID, parentID string, response *PHPParseResponse) {
	i := 0
	currentParent := parentID
	var currentClass string
	var currentInterface string
	var currentTrait string

	for i < len(lines) && len(response.Elements) < p.MaxElements {
		line := lines[i]
		originalLine := line

		if p.StripWhitespace {
			line = strings.TrimSpace(line)
		}

		// Skip empty lines
		if line == "" {
			i++
			continue
		}

		// Check for namespace
		if match := regexp.MustCompile(`^\s*namespace\s+([\w\\]+)\s*;`).FindStringSubmatch(originalLine); match != nil {
			element := p.createNamespaceElement(docID, match[1], i, parentID)
			response.Elements = append(response.Elements, element)
			currentParent = element.ElementID
			i++
			continue
		}

		// Check for use statements
		if match := regexp.MustCompile(`^\s*use\s+([\w\\]+)(?:\s+as\s+(\w+))?\s*;`).FindStringSubmatch(originalLine); match != nil {
			element := p.createUseStatementElement(docID, match[1], match[2], i, currentParent)
			response.Elements = append(response.Elements, element)
			i++
			continue
		}

		// Check for doc blocks
		if p.ExtractDocBlocks && strings.HasPrefix(line, "/**") {
			docContent, newIndex := p.parseDocBlock(lines, i)
			if docContent != "" {
				element := p.createDocBlockElement(docID, docContent, i, currentParent)
				response.Elements = append(response.Elements, element)
				i = newIndex
				continue
			}
		}

		// Check for comments
		if p.ExtractComments {
			if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
				element := p.createCommentElement(docID, line, i, currentParent)
				response.Elements = append(response.Elements, element)
				i++
				continue
			}
			if strings.HasPrefix(line, "/*") && !strings.HasPrefix(line, "/**") {
				commentContent, newIndex := p.parseMultilineComment(lines, i)
				if commentContent != "" {
					element := p.createCommentElement(docID, commentContent, i, currentParent)
					response.Elements = append(response.Elements, element)
					i = newIndex
					continue
				}
			}
		}

		// Check for class
		if match := regexp.MustCompile(`^\s*(?:(abstract|final)\s+)?class\s+(\w+)(?:\s+extends\s+([\w\\]+))?(?:\s+implements\s+([\w\\,\s]+))?\s*\{?`).FindStringSubmatch(originalLine); match != nil {
			modifier := match[1]
			className := match[2]
			extendsClass := match[3]
			implements := match[4]

			element := p.createClassElement(docID, className, modifier, extendsClass, implements, i, currentParent)
			response.Elements = append(response.Elements, element)
			currentClass = element.ElementID
			currentParent = element.ElementID
			i++
			continue
		}

		// Check for interface
		if match := regexp.MustCompile(`^\s*interface\s+(\w+)(?:\s+extends\s+([\w\\,\s]+))?\s*\{?`).FindStringSubmatch(originalLine); match != nil {
			interfaceName := match[1]
			extends := match[2]

			element := p.createInterfaceElement(docID, interfaceName, extends, i, currentParent)
			response.Elements = append(response.Elements, element)
			currentInterface = element.ElementID
			currentParent = element.ElementID
			i++
			continue
		}

		// Check for trait
		if match := regexp.MustCompile(`^\s*trait\s+(\w+)\s*\{?`).FindStringSubmatch(originalLine); match != nil {
			traitName := match[1]

			element := p.createTraitElement(docID, traitName, i, currentParent)
			response.Elements = append(response.Elements, element)
			currentTrait = element.ElementID
			currentParent = element.ElementID
			i++
			continue
		}

		// Check for method (inside class/trait/interface)
		if (currentClass != "" || currentTrait != "" || currentInterface != "") &&
			regexp.MustCompile(`^\s*(?:(public|private|protected)\s+)?(?:(static|abstract|final)\s+)?function\s+\w+`).MatchString(originalLine) {
			if match := regexp.MustCompile(`^\s*(?:(public|private|protected)\s+)?(?:(static|abstract|final)\s+)?function\s+(\w+)\s*\((.*?)\)`).FindStringSubmatch(originalLine); match != nil {
				visibility := match[1]
				modifier := match[2]
				methodName := match[3]
				parameters := match[4]

				element := p.createMethodElement(docID, methodName, visibility, modifier, parameters, i, currentParent)
				response.Elements = append(response.Elements, element)
				i++
				continue
			}
		}

		// Check for standalone function
		if currentClass == "" && currentTrait == "" && currentInterface == "" {
			if match := regexp.MustCompile(`^\s*function\s+(\w+)\s*\((.*?)\)`).FindStringSubmatch(originalLine); match != nil {
				functionName := match[1]
				parameters := match[2]

				element := p.createFunctionElement(docID, functionName, parameters, i, currentParent)
				response.Elements = append(response.Elements, element)
				i++
				continue
			}
		}

		// Check for property (inside class/trait)
		if (currentClass != "" || currentTrait != "") &&
			regexp.MustCompile(`^\s*(?:public|private|protected)\s+(?:static\s+)?\$\w+`).MatchString(originalLine) {
			if match := regexp.MustCompile(`^\s*(public|private|protected)\s+(static\s+)?(\$\w+)(?:\s*=\s*(.+?))?;`).FindStringSubmatch(originalLine); match != nil {
				visibility := match[1]
				static := strings.TrimSpace(match[2])
				propertyName := match[3]
				defaultValue := strings.TrimSpace(match[4])

				element := p.createPropertyElement(docID, propertyName, visibility, static, defaultValue, i, currentParent)
				response.Elements = append(response.Elements, element)
				i++
				continue
			}
		}

		// Check for constant (inside class/interface)
		if (currentClass != "" || currentInterface != "") &&
			regexp.MustCompile(`^\s*const\s+\w+`).MatchString(originalLine) {
			if match := regexp.MustCompile(`^\s*const\s+(\w+)\s*=\s*(.+?);`).FindStringSubmatch(originalLine); match != nil {
				constantName := match[1]
				value := strings.TrimSpace(match[2])

				element := p.createConstantElement(docID, constantName, value, i, currentParent)
				response.Elements = append(response.Elements, element)
				i++
				continue
			}
		}

		// Check for closing brace (reset current context)
		if line == "}" {
			// Reset to parent level
			if currentTrait != "" {
				currentTrait = ""
				currentParent = parentID
			} else if currentInterface != "" {
				currentInterface = ""
				currentParent = parentID
			} else if currentClass != "" {
				currentClass = ""
				currentParent = parentID
			}
		}

		i++
	}
}

// createNamespaceElement creates a namespace element
func (p *PHPParser) createNamespaceElement(docID, namespace string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_ns")
	contentHash := p.calculateContentHash(namespace)
	preview := p.truncateContent(namespace)

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeNamespace,
		Text:            namespace,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "namespace", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"element_type": "namespace",
			"namespace":    namespace,
			"line_number":  lineNumber,
		},
	}
}

// createUseStatementElement creates a use statement element
func (p *PHPParser) createUseStatementElement(docID, usePath, alias string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_use")
	contentHash := p.calculateContentHash(usePath)
	preview := p.truncateContent(usePath)

	metadata := map[string]interface{}{
		"element_type": "use_statement",
		"use_path":     usePath,
		"line_number":  lineNumber,
	}

	if alias != "" {
		metadata["alias"] = alias
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeUseStatement,
		Text:            usePath,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "use_statement", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createClassElement creates a class element
func (p *PHPParser) createClassElement(docID, className, modifier, extendsClass, implements string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_class")
	contentHash := p.calculateContentHash(className)
	preview := p.truncateContent(className)

	metadata := map[string]interface{}{
		"element_type": "class",
		"class_name":   className,
		"line_number":  lineNumber,
	}

	if modifier != "" {
		metadata["modifier"] = modifier
	}
	if extendsClass != "" {
		metadata["extends"] = extendsClass
	}
	if implements != "" {
		metadata["implements"] = strings.Split(implements, ",")
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeClass,
		Text:            className,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "class", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createInterfaceElement creates an interface element
func (p *PHPParser) createInterfaceElement(docID, interfaceName, extends string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_iface")
	contentHash := p.calculateContentHash(interfaceName)
	preview := p.truncateContent(interfaceName)

	metadata := map[string]interface{}{
		"element_type":   "interface",
		"interface_name": interfaceName,
		"line_number":    lineNumber,
	}

	if extends != "" {
		metadata["extends"] = strings.Split(extends, ",")
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeInterface,
		Text:            interfaceName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "interface", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createTraitElement creates a trait element
func (p *PHPParser) createTraitElement(docID, traitName string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_trait")
	contentHash := p.calculateContentHash(traitName)
	preview := p.truncateContent(traitName)

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeTrait,
		Text:            traitName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "trait", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"element_type": "trait",
			"trait_name":   traitName,
			"line_number":  lineNumber,
		},
	}
}

// createFunctionElement creates a function element
func (p *PHPParser) createFunctionElement(docID, functionName, parameters string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_func")
	contentHash := p.calculateContentHash(functionName)
	preview := p.truncateContent(functionName)

	metadata := map[string]interface{}{
		"element_type":  "function",
		"function_name": functionName,
		"line_number":   lineNumber,
	}

	if parameters != "" {
		metadata["parameters"] = parameters
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeFunction,
		Text:            functionName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "function", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createMethodElement creates a method element
func (p *PHPParser) createMethodElement(docID, methodName, visibility, modifier, parameters string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_method")
	contentHash := p.calculateContentHash(methodName)
	preview := p.truncateContent(methodName)

	metadata := map[string]interface{}{
		"element_type": "method",
		"method_name":  methodName,
		"line_number":  lineNumber,
	}

	if visibility != "" {
		metadata["visibility"] = visibility
	}
	if modifier != "" {
		metadata["modifier"] = modifier
	}
	if parameters != "" {
		metadata["parameters"] = parameters
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeMethod,
		Text:            methodName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "method", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createPropertyElement creates a property element
func (p *PHPParser) createPropertyElement(docID, propertyName, visibility, static, defaultValue string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_prop")
	contentHash := p.calculateContentHash(propertyName)
	preview := p.truncateContent(propertyName)

	metadata := map[string]interface{}{
		"element_type":  "property",
		"property_name": propertyName,
		"visibility":    visibility,
		"line_number":   lineNumber,
	}

	if static != "" {
		metadata["static"] = true
	}
	if defaultValue != "" {
		metadata["default_value"] = defaultValue
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeProperty,
		Text:            propertyName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "property", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// createConstantElement creates a constant element
func (p *PHPParser) createConstantElement(docID, constantName, value string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_const")
	contentHash := p.calculateContentHash(constantName)
	preview := p.truncateContent(constantName)

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeConstant,
		Text:            constantName,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "constant", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"element_type":  "constant",
			"constant_name": constantName,
			"value":         value,
			"line_number":   lineNumber,
		},
	}
}

// createCommentElement creates a comment element
func (p *PHPParser) createCommentElement(docID, comment string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_comment")
	contentHash := p.calculateContentHash(comment)
	preview := p.truncateContent(comment)

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeComment,
		Text:            comment,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "comment", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"element_type": "comment",
			"line_number":  lineNumber,
		},
	}
}

// createDocBlockElement creates a doc block element
func (p *PHPParser) createDocBlockElement(docID, content string, lineNumber int, parentID string) PHPElement {
	elementID := p.generateElementID("php_doc")
	contentHash := p.calculateContentHash(content)
	preview := p.truncateContent(content)

	// Extract doc block tags
	tags := p.extractDocBlockTags(content)

	metadata := map[string]interface{}{
		"element_type": "doc_block",
		"line_number":  lineNumber,
	}

	if len(tags) > 0 {
		metadata["tags"] = tags
	}

	return PHPElement{
		ElementID:       elementID,
		DocumentID:      docID,
		ElementType:     PHPElementTypeDocBlock,
		Text:            content,
		ContentPreview:  preview,
		ContentLocation: map[string]interface{}{"type": "doc_block", "line": lineNumber},
		ContentHash:     contentHash,
		ParentID:        parentID,
		Metadata:        metadata,
	}
}

// parseDocBlock parses a doc block starting at the given index
func (p *PHPParser) parseDocBlock(lines []string, startIndex int) (string, int) {
	if startIndex >= len(lines) {
		return "", startIndex + 1
	}

	var docLines []string
	i := startIndex

	// Include opening /**
	docLines = append(docLines, lines[i])
	i++

	// Find closing */
	for i < len(lines) {
		docLines = append(docLines, lines[i])
		if strings.Contains(lines[i], "*/") {
			i++
			break
		}
		i++
	}

	return strings.Join(docLines, "\n"), i
}

// parseMultilineComment parses a multiline comment starting at the given index
func (p *PHPParser) parseMultilineComment(lines []string, startIndex int) (string, int) {
	if startIndex >= len(lines) {
		return "", startIndex + 1
	}

	var commentLines []string
	i := startIndex

	// Include opening /*
	commentLines = append(commentLines, lines[i])

	// If the comment closes on the same line
	if strings.Contains(lines[i], "*/") {
		return lines[i], i + 1
	}

	i++

	// Find closing */
	for i < len(lines) {
		commentLines = append(commentLines, lines[i])
		if strings.Contains(lines[i], "*/") {
			i++
			break
		}
		i++
	}

	return strings.Join(commentLines, "\n"), i
}

// extractDocBlockTags extracts tags from a doc block
func (p *PHPParser) extractDocBlockTags(content string) []map[string]interface{} {
	var tags []map[string]interface{}

	// Match @tagname followed by content
	tagPattern := regexp.MustCompile(`@(\w+)\s+(.+?)(?:\n\s*(?:\*\s*)?@|\n\s*\*/|$)`)
	matches := tagPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tag := map[string]interface{}{
				"name":  match[1],
				"value": strings.TrimSpace(match[2]),
			}
			tags = append(tags, tag)
		}
	}

	return tags
}

// generateElementID generates a unique element ID
func (p *PHPParser) generateElementID(prefix string) string {
	p.ElementIDCounter++
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s_%d_%d", prefix, timestamp, p.ElementIDCounter)
}

// calculateContentHash calculates MD5 hash of content
func (p *PHPParser) calculateContentHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// truncateContent truncates content for preview
func (p *PHPParser) truncateContent(content string) string {
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

// convertToParseResult converts PHPParseResponse to universal ParseResult format
func (p *PHPParser) convertToParseResult(response *PHPParseResponse) *ParseResult {
	result := &ParseResult{
		Document: Document{
			ID:      response.Document["doc_id"].(string),
			DocType: response.Document["doc_type"].(string),
		},
		Elements: make([]Element, 0, len(response.Elements)),
	}

	// Convert metadata if present
	if meta, ok := response.Document["metadata"]; ok {
		result.Document.Metadata = meta.(map[string]interface{})
	}

	// Convert elements
	for i, phpElem := range response.Elements {
		element := Element{
			ElementID:       phpElem.ElementID,
			ElementType:     string(phpElem.ElementType),
			Content:         phpElem.Text,
			ContentPreview:  phpElem.ContentPreview,
			ParentID:        phpElem.ParentID,
			Position:        i,
			Depth:           calculatePHPDepth(phpElem.ElementType),
			ContentLocation: phpElem.ContentLocation,
			Metadata:        phpElem.Metadata,
		}

		// Set element category
		element.ElementCategory = GetElementCategory(element.ElementType)

		result.Elements = append(result.Elements, element)
	}

	return result
}

// calculatePHPDepth calculates element depth based on PHP element type
func calculatePHPDepth(elementType PHPElementType) int {
	switch elementType {
	case PHPElementTypeRoot:
		return 0
	case PHPElementTypeNamespace:
		return 1
	case PHPElementTypeClass, PHPElementTypeInterface, PHPElementTypeTrait, PHPElementTypeFunction:
		return 2
	case PHPElementTypeMethod, PHPElementTypeProperty, PHPElementTypeConstant:
		return 3
	case PHPElementTypeComment, PHPElementTypeDocBlock, PHPElementTypeUseStatement:
		return 2
	default:
		return 2 // Default depth for unknown elements
	}
}