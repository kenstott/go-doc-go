package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/protobuf"
)

// ProtobufCodeParser parses Protocol Buffer files using tree-sitter
type ProtobufCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	sourceCode        []byte
	packageName       string
}

// NewProtobufCodeParser creates a new Protobuf code parser
func NewProtobufCodeParser() *ProtobufCodeParser {
	return &ProtobufCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
	}
}

// GetName returns the parser name
func (p *ProtobufCodeParser) GetName() string {
	return "protobuf_code"
}

// GetSupportedFormats returns supported file formats
func (p *ProtobufCodeParser) GetSupportedFormats() []string {
	return []string{".proto", "proto"}
}

// SupportsStreaming returns false as this parser needs complete content
func (p *ProtobufCodeParser) SupportsStreaming() bool {
	return false
}

// Close performs any cleanup needed
func (p *ProtobufCodeParser) Close() error {
	return nil
}

// Parse parses Protobuf file and extracts schema elements
func (p *ProtobufCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from content
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Protobuf parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p.sourceCode = sourceCode
	p.packageName = ""

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(protobuf.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Protobuf source: %w", err)
	}
	defer tree.Close()

	// Create document element
	content := string(sourceCode)
	docElement := Element{
		ElementID:       generateID("proto_doc"),
		ElementType:     "document",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "document",
		Metadata: map[string]interface{}{
			"language": "protobuf",
			"parser":   "protobuf_code",
		},
	}

	// Extract all elements
	elements := []Element{docElement}

	// Extract elements from AST
	rootNode := tree.RootNode()
	extractedElements := p.extractElements(rootNode, docElement.ElementID)
	elements = append(elements, extractedElements...)

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "protobuf_source",
		Metadata: map[string]interface{}{
			"language": "protobuf",
			"parser":   "protobuf_code",
		},
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			doc.Metadata[k] = v
		}
	}

	// Add package name if found
	if p.packageName != "" {
		doc.Metadata["package"] = p.packageName
	}

	return &ParseResult{
		Document: doc,
		Elements: elements,
	}, nil
}

// extractElements recursively extracts Protobuf elements from AST
func (p *ProtobufCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "package":
			p.extractPackageName(child)

		case "import":
			importElements := p.createImportElements(child, parentID)
			elements = append(elements, importElements...)

		case "message":
			messageElements := p.createMessageElements(child, parentID)
			elements = append(elements, messageElements...)

		case "service":
			serviceElements := p.createServiceElements(child, parentID)
			elements = append(elements, serviceElements...)

		case "enum":
			enumElements := p.createEnumElements(child, parentID)
			elements = append(elements, enumElements...)

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, parentID)
				if commentElem != nil {
					elements = append(elements, *commentElem)
				}
			}

		default:
			// Recursively process other nodes
			childElements := p.extractElements(child, parentID)
			elements = append(elements, childElements...)
		}
	}

	return elements
}

// extractPackageName extracts the package declaration
func (p *ProtobufCodeParser) extractPackageName(node *sitter.Node) {
	// Find identifier for package name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "full_ident" || child.Type() == "identifier" {
			p.packageName = p.getNodeText(child)
			break
		}
	}
}

// createImportElements extracts import statements
func (p *ProtobufCodeParser) createImportElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract import path
	var importPath string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "string" {
			importPath = p.getNodeText(child)
			importPath = strings.Trim(importPath, "\"")
			break
		}
	}

	elem := Element{
		ElementID:       generateID("proto_import"),
		ElementType:     "code_dependency",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "dependency",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind": "import",
			"import_path":     importPath,
			"language":        "protobuf",
			"line_number":     lineNum,
		},
	}

	elements = append(elements, elem)

	return elements
}

// createMessageElements extracts message definitions
func (p *ProtobufCodeParser) createMessageElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract message name
	var messageName string
	nameNode := p.findChildByType(node, "message_name")
	if nameNode == nil {
		nameNode = p.findChildByType(node, "identifier")
	}
	if nameNode != nil {
		messageName = p.getNodeText(nameNode)
	}

	if messageName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1

	namespace := p.packageName

	messageElement := Element{
		ElementID:       generateID("proto_message"),
		ElementType:     "code_grouping",
		ContentPreview:  truncate(fmt.Sprintf("message %s", messageName), p.MaxContentPreview),
		ElementCategory: "data_structure",
		ParentID:        parentID,
		ClassName:       &messageName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "message",
			"message_name":      messageName,
			"language":          "protobuf",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, messageElement)

	// Extract message body (fields)
	bodyNode := p.findChildByType(node, "message_body")
	if bodyNode != nil {
		fieldElements := p.extractMessageFields(bodyNode, messageElement.ElementID, messageName)
		elements = append(elements, fieldElements...)
	}

	return elements
}

// extractMessageFields extracts fields from message body
func (p *ProtobufCodeParser) extractMessageFields(node *sitter.Node, parentID string, messageName string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		if nodeType == "field" {
			fieldElem := p.createFieldElement(child, parentID, messageName)
			if fieldElem != nil {
				elements = append(elements, *fieldElem)
			}
		} else if nodeType == "message" {
			// Nested message
			nestedElements := p.createMessageElements(child, parentID)
			elements = append(elements, nestedElements...)
		} else if nodeType == "enum" {
			// Nested enum
			enumElements := p.createEnumElements(child, parentID)
			elements = append(elements, enumElements...)
		}
	}

	return elements
}

// createFieldElement creates a field element
func (p *ProtobufCodeParser) createFieldElement(node *sitter.Node, parentID string, messageName string) *Element {
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract field type and name
	var fieldType string
	var fieldName string
	var fieldNumber string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "type" || childType == "message_or_enum_type" {
			fieldType = p.getNodeText(child)
		} else if childType == "field_name" || (childType == "identifier" && fieldName == "") {
			fieldName = p.getNodeText(child)
		} else if childType == "field_number" {
			fieldNumber = p.getNodeText(child)
		}
	}

	if fieldName == "" {
		return nil
	}

	return &Element{
		ElementID:       generateID("proto_field"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":    "field",
			"data_name":    fieldName,
			"data_type":    fieldType,
			"field_number": fieldNumber,
			"message_name": messageName,
			"language":     "protobuf",
			"line_number":  lineNum,
		},
	}
}

// createServiceElements extracts service definitions
func (p *ProtobufCodeParser) createServiceElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract service name
	var serviceName string
	nameNode := p.findChildByType(node, "service_name")
	if nameNode == nil {
		nameNode = p.findChildByType(node, "identifier")
	}
	if nameNode != nil {
		serviceName = p.getNodeText(nameNode)
	}

	if serviceName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1

	namespace := p.packageName

	serviceElement := Element{
		ElementID:       generateID("proto_service"),
		ElementType:     "code_grouping",
		ContentPreview:  truncate(fmt.Sprintf("service %s", serviceName), p.MaxContentPreview),
		ElementCategory: "service_definition",
		ParentID:        parentID,
		ClassName:       &serviceName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "service",
			"service_name":      serviceName,
			"language":          "protobuf",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, serviceElement)

	// Extract RPCs directly from service node
	rpcElements := p.extractRPCs(node, serviceElement.ElementID, serviceName)
	elements = append(elements, rpcElements...)

	return elements
}

// extractRPCs extracts RPC methods from service body
func (p *ProtobufCodeParser) extractRPCs(node *sitter.Node, parentID string, serviceName string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "rpc" {
			rpcElem := p.createRPCElement(child, parentID, serviceName)
			if rpcElem != nil {
				elements = append(elements, *rpcElem)
			}
		}
	}

	return elements
}

// createRPCElement creates an RPC element
func (p *ProtobufCodeParser) createRPCElement(node *sitter.Node, parentID string, serviceName string) *Element {
	lineNum := int(node.StartPoint().Row) + 1

	// Extract RPC name, request type, and response type
	var rpcName string
	var requestType string
	var responseType string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "rpc_name" || (childType == "identifier" && rpcName == "") {
			rpcName = p.getNodeText(child)
		} else if childType == "message_or_enum_type" {
			if requestType == "" {
				requestType = p.getNodeText(child)
			} else if responseType == "" {
				responseType = p.getNodeText(child)
			}
		}
	}

	if rpcName == "" {
		return nil
	}

	signature := fmt.Sprintf("rpc %s(%s) returns (%s)", rpcName, requestType, responseType)

	return &Element{
		ElementID:       generateID("proto_rpc"),
		ElementType:     "code_function",
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ElementCategory: "service_method",
		ParentID:        parentID,
		FunctionName:    &rpcName,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "rpc",
			"rpc_name":          rpcName,
			"request_type":      requestType,
			"response_type":     responseType,
			"service_name":      serviceName,
			"language":          "protobuf",
			"line_number":       lineNum,
		},
	}
}

// createEnumElements extracts enum definitions
func (p *ProtobufCodeParser) createEnumElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract enum name
	var enumName string
	nameNode := p.findChildByType(node, "enum_name")
	if nameNode == nil {
		nameNode = p.findChildByType(node, "identifier")
	}
	if nameNode != nil {
		enumName = p.getNodeText(nameNode)
	}

	if enumName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1

	namespace := p.packageName

	enumElement := Element{
		ElementID:       generateID("proto_enum"),
		ElementType:     "code_type",
		ContentPreview:  truncate(fmt.Sprintf("enum %s", enumName), p.MaxContentPreview),
		ElementCategory: "type_definition",
		ParentID:        parentID,
		ClassName:       &enumName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "enum",
			"type_kind":         "enum",
			"enum_name":         enumName,
			"language":          "protobuf",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, enumElement)

	// Extract enum values
	bodyNode := p.findChildByType(node, "enum_body")
	if bodyNode != nil {
		valueElements := p.extractEnumValues(bodyNode, enumElement.ElementID, enumName)
		elements = append(elements, valueElements...)
	}

	return elements
}

// extractEnumValues extracts enum value fields
func (p *ProtobufCodeParser) extractEnumValues(node *sitter.Node, parentID string, enumName string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "enum_field" {
			valueElem := p.createEnumValueElement(child, parentID, enumName)
			if valueElem != nil {
				elements = append(elements, *valueElem)
			}
		}
	}

	return elements
}

// createEnumValueElement creates an enum value element
func (p *ProtobufCodeParser) createEnumValueElement(node *sitter.Node, parentID string, enumName string) *Element {
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract enum value name and number
	var valueName string
	var valueNumber string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "identifier" && valueName == "" {
			valueName = p.getNodeText(child)
		} else if childType == "int_lit" {
			valueNumber = p.getNodeText(child)
		}
	}

	if valueName == "" {
		return nil
	}

	return &Element{
		ElementID:       generateID("proto_enum_val"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":   "enum_value",
			"data_name":   valueName,
			"value":       valueNumber,
			"enum_name":   enumName,
			"language":    "protobuf",
			"line_number": lineNum,
		},
	}
}

// createCommentElement creates a comment element
func (p *ProtobufCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean up comment markers
	if strings.HasPrefix(content, "//") {
		content = strings.TrimPrefix(content, "//")
	} else if strings.HasPrefix(content, "/*") {
		content = strings.TrimPrefix(content, "/*")
		content = strings.TrimSuffix(content, "*/")
	}
	content = strings.TrimSpace(content)

	return &Element{
		ElementID:       generateID("proto_comment"),
		ElementType:     "code_documentation",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "documentation",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    "comment",
			"language":    "protobuf",
			"line_number": lineNum,
		},
	}
}

// Helper functions

func (p *ProtobufCodeParser) getNodeText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	if end > uint32(len(p.sourceCode)) {
		end = uint32(len(p.sourceCode))
	}
	return string(p.sourceCode[start:end])
}

func (p *ProtobufCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}
