package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/hcl"
)

// HCLCodeParser parses HCL (Terraform) files using tree-sitter
type HCLCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	sourceCode        []byte
}

// NewHCLCodeParser creates a new HCL code parser
func NewHCLCodeParser() *HCLCodeParser {
	return &HCLCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
	}
}

// GetName returns the parser name
func (p *HCLCodeParser) GetName() string {
	return "hcl_code"
}

// GetSupportedFormats returns supported file formats
func (p *HCLCodeParser) GetSupportedFormats() []string {
	return []string{".tf", ".hcl", "tf", "hcl"}
}

// SupportsStreaming returns false as this parser needs complete content
func (p *HCLCodeParser) SupportsStreaming() bool {
	return false
}

// Close performs any cleanup needed
func (p *HCLCodeParser) Close() error {
	return nil
}

// Parse parses HCL/Terraform file and extracts infrastructure elements
func (p *HCLCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from content
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for HCL parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p.sourceCode = sourceCode

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(hcl.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HCL source: %w", err)
	}
	defer tree.Close()

	// Create document element
	content := string(sourceCode)
	docElement := Element{
		ElementID:       generateID("hcl_doc"),
		ElementType:     "document",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "document",
		Metadata: map[string]interface{}{
			"language": "hcl",
			"parser":   "hcl_code",
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
		DocType: "hcl_source",
		Metadata: map[string]interface{}{
			"language": "hcl",
			"parser":   "hcl_code",
		},
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			doc.Metadata[k] = v
		}
	}

	return &ParseResult{
		Document: doc,
		Elements: elements,
	}, nil
}

// extractElements recursively extracts HCL elements from AST
func (p *HCLCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "block":
			blockElements := p.createBlockElements(child, parentID)
			elements = append(elements, blockElements...)

		case "attribute":
			attrElements := p.createAttributeElements(child, parentID)
			elements = append(elements, attrElements...)

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

// createBlockElements extracts HCL blocks (resource, variable, module, etc.)
func (p *HCLCodeParser) createBlockElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract block type (resource, variable, output, etc.)
	var blockType string
	var blockLabels []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "identifier" {
			if blockType == "" {
				blockType = p.getNodeText(child)
			} else {
				blockLabels = append(blockLabels, p.getNodeText(child))
			}
		} else if childType == "string_lit" {
			// String labels (e.g., resource "aws_instance" "example")
			label := p.getNodeText(child)
			label = strings.Trim(label, "\"")
			blockLabels = append(blockLabels, label)
		}
	}

	if blockType == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Determine element type based on block type
	var elementType string
	var category string
	var metadata map[string]interface{}

	switch blockType {
	case "resource", "data":
		elementType = "code_grouping"
		category = "infrastructure"
		resourceType := ""
		resourceName := ""
		if len(blockLabels) >= 2 {
			resourceType = blockLabels[0]
			resourceName = blockLabels[1]
		}
		metadata = map[string]interface{}{
			"code_element_kind": blockType,
			"resource_type":     resourceType,
			"resource_name":     resourceName,
			"language":          "hcl",
			"line_number":       lineNum,
		}

	case "variable":
		elementType = "code_data"
		category = "configuration"
		varName := ""
		if len(blockLabels) > 0 {
			varName = blockLabels[0]
		}
		metadata = map[string]interface{}{
			"data_kind":   "variable",
			"data_name":   varName,
			"language":    "hcl",
			"line_number": lineNum,
		}

	case "output":
		elementType = "code_data"
		category = "configuration"
		outputName := ""
		if len(blockLabels) > 0 {
			outputName = blockLabels[0]
		}
		metadata = map[string]interface{}{
			"data_kind":   "output",
			"data_name":   outputName,
			"language":    "hcl",
			"line_number": lineNum,
		}

	case "module":
		elementType = "code_dependency"
		category = "dependency"
		moduleName := ""
		if len(blockLabels) > 0 {
			moduleName = blockLabels[0]
		}
		metadata = map[string]interface{}{
			"dependency_kind": "module",
			"module_name":     moduleName,
			"language":        "hcl",
			"line_number":     lineNum,
		}

	case "provider":
		elementType = "code_dependency"
		category = "dependency"
		providerName := ""
		if len(blockLabels) > 0 {
			providerName = blockLabels[0]
		}
		metadata = map[string]interface{}{
			"dependency_kind": "provider",
			"provider_name":   providerName,
			"language":        "hcl",
			"line_number":     lineNum,
		}

	case "terraform":
		elementType = "code_grouping"
		category = "configuration"
		metadata = map[string]interface{}{
			"code_element_kind": "terraform_config",
			"language":          "hcl",
			"line_number":       lineNum,
		}

	case "locals":
		elementType = "code_grouping"
		category = "configuration"
		metadata = map[string]interface{}{
			"code_element_kind": "locals",
			"language":          "hcl",
			"line_number":       lineNum,
		}

	default:
		elementType = "code_grouping"
		category = "configuration"
		metadata = map[string]interface{}{
			"code_element_kind": blockType,
			"language":          "hcl",
			"line_number":       lineNum,
		}
	}

	preview := truncate(content, p.MaxContentPreview)

	blockElement := Element{
		ElementID:       generateID("hcl_block"),
		ElementType:     elementType,
		ContentPreview:  preview,
		ElementCategory: category,
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata:        metadata,
	}

	elements = append(elements, blockElement)

	// Extract block body
	bodyNode := p.findChildByType(node, "body")
	if bodyNode != nil {
		bodyElements := p.extractElements(bodyNode, blockElement.ElementID)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// createAttributeElements extracts HCL attributes (key = value)
func (p *HCLCodeParser) createAttributeElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract attribute name
	var attrName string
	nameNode := p.findChildByType(node, "identifier")
	if nameNode != nil {
		attrName = p.getNodeText(nameNode)
	}

	if attrName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract value
	var attrValue string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "identifier" && child.Type() != "=" {
			attrValue = p.getNodeText(child)
			break
		}
	}

	elem := Element{
		ElementID:       generateID("hcl_attr"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":   "attribute",
			"data_name":   attrName,
			"value":       attrValue,
			"language":    "hcl",
			"line_number": lineNum,
		},
	}

	elements = append(elements, elem)

	return elements
}

// createCommentElement creates a comment element
func (p *HCLCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean up comment markers
	if strings.HasPrefix(content, "//") {
		content = strings.TrimPrefix(content, "//")
	} else if strings.HasPrefix(content, "#") {
		content = strings.TrimPrefix(content, "#")
	} else if strings.HasPrefix(content, "/*") {
		content = strings.TrimPrefix(content, "/*")
		content = strings.TrimSuffix(content, "*/")
	}
	content = strings.TrimSpace(content)

	return &Element{
		ElementID:       generateID("hcl_comment"),
		ElementType:     "code_documentation",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "documentation",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    "comment",
			"language":    "hcl",
			"line_number": lineNum,
		},
	}
}

// Helper functions

func (p *HCLCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *HCLCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}
