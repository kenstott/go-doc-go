package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/yaml"
)

// YAMLParser parses YAML configuration files using tree-sitter
type YAMLParser struct {
	MaxContentPreview int
	ExtractComments   bool
	sourceCode        []byte
}

// NewYAMLParser creates a new YAML parser
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
	}
}

// GetName returns the parser name
func (p *YAMLParser) GetName() string {
	return "yaml_data"
}

// GetSupportedFormats returns supported file formats
func (p *YAMLParser) GetSupportedFormats() []string {
	return []string{".yaml", ".yml", "yaml", "yml"}
}

// SupportsStreaming returns false as this parser needs complete content
func (p *YAMLParser) SupportsStreaming() bool {
	return false
}

// Close performs any cleanup needed
func (p *YAMLParser) Close() error {
	return nil
}

// Parse parses YAML file and extracts configuration elements
func (p *YAMLParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from content
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for YAML parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p.sourceCode = sourceCode
	content := string(sourceCode)

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(yaml.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML source: %w", err)
	}
	defer tree.Close()

	// Create document element
	docElement := Element{
		ElementID:       generateID("yaml_doc"),
		ElementType:     "document",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "document",
		Metadata: map[string]interface{}{
			"format": "yaml",
			"parser": "yaml_data",
		},
	}

	// Extract all elements
	elements := []Element{docElement}

	// Extract elements from AST
	rootNode := tree.RootNode()
	extractedElements := p.extractElements(rootNode, docElement.ElementID, []string{})
	elements = append(elements, extractedElements...)

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "yaml_config",
		Metadata: map[string]interface{}{
			"format": "yaml",
			"parser": "yaml_data",
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

// extractElements recursively extracts YAML elements from AST
func (p *YAMLParser) extractElements(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "stream":
			// Top-level stream node, recurse into it
			streamElements := p.extractElements(child, parentID, path)
			elements = append(elements, streamElements...)

		case "document":
			// YAML document, recurse into it
			docElements := p.extractElements(child, parentID, path)
			elements = append(elements, docElements...)

		case "block_node":
			// Block node container, recurse into it
			blockElements := p.extractElements(child, parentID, path)
			elements = append(elements, blockElements...)

		case "block_mapping":
			// Mapping (key-value pairs)
			mappingElements := p.extractMappingElements(child, parentID, path)
			elements = append(elements, mappingElements...)

		case "block_mapping_pair":
			// Individual mapping pair
			pairElements := p.extractMappingPair(child, parentID, path)
			elements = append(elements, pairElements...)

		case "block_sequence":
			// Sequence (array)
			sequenceElements := p.extractSequenceElements(child, parentID, path)
			elements = append(elements, sequenceElements...)

		case "flow_node":
			// Flow-style node (inline), recurse into it
			flowElements := p.extractElements(child, parentID, path)
			elements = append(elements, flowElements...)

		case "flow_sequence":
			// Flow sequence [a, b, c]
			flowSeqElements := p.extractFlowSequence(child, parentID, path)
			elements = append(elements, flowSeqElements...)

		case "flow_mapping":
			// Flow mapping {key: value}
			flowMapElements := p.extractFlowMapping(child, parentID, path)
			elements = append(elements, flowMapElements...)

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, parentID)
				if commentElem != nil {
					elements = append(elements, *commentElem)
				}
			}

		default:
			// For other node types, recurse
			childElements := p.extractElements(child, parentID, path)
			elements = append(elements, childElements...)
		}
	}

	return elements
}

// extractMappingElements extracts elements from a block mapping
func (p *YAMLParser) extractMappingElements(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "block_mapping_pair" {
			pairElements := p.extractMappingPair(child, parentID, path)
			elements = append(elements, pairElements...)
		}
	}

	return elements
}

// extractMappingPair extracts a key-value pair
func (p *YAMLParser) extractMappingPair(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	// Find key node
	keyNode := p.findChildByType(node, "flow_node")
	if keyNode == nil {
		// Try other key types
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if strings.Contains(child.Type(), "scalar") || child.Type() == "plain_scalar" {
				keyNode = child
				break
			}
		}
	}

	if keyNode == nil {
		return elements
	}

	keyText := strings.TrimSpace(p.getNodeText(keyNode))
	if keyText == "" || keyText == ":" {
		return elements
	}

	// Build path for this key
	currentPath := append(path, keyText)
	pathStr := strings.Join(currentPath, ".")

	// Find value node (look for block_node, flow_node, scalar, sequence, mapping)
	var valueNode *sitter.Node
	var valueType string
	foundKey := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		// Skip until we find the key
		if !foundKey {
			if strings.Contains(p.getNodeText(child), keyText) {
				foundKey = true
			}
			continue
		}

		// After key, look for value
		if childType == "block_node" || childType == "flow_node" ||
			strings.Contains(childType, "scalar") ||
			childType == "block_sequence" || childType == "block_mapping" ||
			childType == "flow_sequence" || childType == "flow_mapping" {
			valueNode = child
			valueType = childType
			break
		}
	}

	lineNum := int(node.StartPoint().Row) + 1

	// Determine element type based on value
	if valueNode != nil {
		valueText := strings.TrimSpace(p.getNodeText(valueNode))

		if valueType == "block_mapping" || valueType == "flow_mapping" {
			// Nested mapping (object)
			elem := Element{
				ElementID:       generateID("yaml_mapping"),
				ElementType:     "code_grouping",
				ContentPreview:  truncate(fmt.Sprintf("%s: {...}", keyText), p.MaxContentPreview),
				ElementCategory: "data_structure",
				ParentID:        parentID,
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"data_kind": "mapping",
					"key":       keyText,
					"path":      pathStr,
					"format":    "yaml",
				},
			}
			elements = append(elements, elem)

			// Recurse into nested mapping
			nestedElements := p.extractElements(valueNode, elem.ElementID, currentPath)
			elements = append(elements, nestedElements...)

		} else if valueType == "block_sequence" || valueType == "flow_sequence" {
			// Array/list
			elem := Element{
				ElementID:       generateID("yaml_sequence"),
				ElementType:     "code_data",
				ContentPreview:  truncate(fmt.Sprintf("%s: [...]", keyText), p.MaxContentPreview),
				ElementCategory: "data_field",
				ParentID:        parentID,
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"data_kind": "sequence",
					"data_name": keyText,
					"path":      pathStr,
					"format":    "yaml",
				},
			}
			elements = append(elements, elem)

			// Extract sequence items
			seqElements := p.extractElements(valueNode, elem.ElementID, currentPath)
			elements = append(elements, seqElements...)

		} else {
			// Scalar value (string, number, boolean)
			preview := fmt.Sprintf("%s: %s", keyText, valueText)
			if len(preview) > p.MaxContentPreview {
				preview = truncate(preview, p.MaxContentPreview)
			}

			elem := Element{
				ElementID:       generateID("yaml_scalar"),
				ElementType:     "code_data",
				ContentPreview:  preview,
				ElementCategory: "data_field",
				ParentID:        parentID,
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"data_kind": "scalar",
					"data_name": keyText,
					"value":     valueText,
					"path":      pathStr,
					"format":    "yaml",
				},
			}
			elements = append(elements, elem)
		}
	} else {
		// No value found, might be null or empty
		elem := Element{
			ElementID:       generateID("yaml_scalar"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("%s: null", keyText), p.MaxContentPreview),
			ElementCategory: "data_field",
			ParentID:        parentID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"data_kind": "scalar",
				"data_name": keyText,
				"value":     nil,
				"path":      pathStr,
				"format":    "yaml",
			},
		}
		elements = append(elements, elem)
	}

	return elements
}

// extractSequenceElements extracts array items
func (p *YAMLParser) extractSequenceElements(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	itemIndex := 0
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "block_sequence_item" {
			// Extract item
			itemPath := append(path, fmt.Sprintf("[%d]", itemIndex))
			itemElements := p.extractSequenceItem(child, parentID, itemPath, itemIndex)
			elements = append(elements, itemElements...)
			itemIndex++
		}
	}

	return elements
}

// extractSequenceItem extracts a single sequence item
func (p *YAMLParser) extractSequenceItem(node *sitter.Node, parentID string, path []string, index int) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := strings.TrimSpace(p.getNodeText(node))
	// Remove leading "- "
	content = strings.TrimPrefix(content, "-")
	content = strings.TrimSpace(content)

	// Check if item is a mapping or scalar
	hasMapping := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "block_mapping" || child.Type() == "block_node" {
			hasMapping = true
			break
		}
	}

	pathStr := strings.Join(path, ".")

	if hasMapping {
		// Nested structure
		elem := Element{
			ElementID:       generateID("yaml_seq_item"),
			ElementType:     "code_grouping",
			ContentPreview:  truncate(fmt.Sprintf("[%d]: {...}", index), p.MaxContentPreview),
			ElementCategory: "data_structure",
			ParentID:        parentID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"data_kind": "sequence_item",
				"index":     index,
				"path":      pathStr,
				"format":    "yaml",
			},
		}
		elements = append(elements, elem)

		// Recurse into nested structure
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			nestedElements := p.extractElements(child, elem.ElementID, path)
			elements = append(elements, nestedElements...)
		}
	} else {
		// Scalar item
		elem := Element{
			ElementID:       generateID("yaml_seq_item"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("[%d]: %s", index, content), p.MaxContentPreview),
			ElementCategory: "data_field",
			ParentID:        parentID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"data_kind": "sequence_item",
				"index":     index,
				"value":     content,
				"path":      pathStr,
				"format":    "yaml",
			},
		}
		elements = append(elements, elem)
	}

	return elements
}

// extractFlowSequence extracts flow-style array [a, b, c]
func (p *YAMLParser) extractFlowSequence(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	pathStr := strings.Join(path, ".")

	elem := Element{
		ElementID:       generateID("yaml_flow_seq"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind": "flow_sequence",
			"value":     content,
			"path":      pathStr,
			"format":    "yaml",
		},
	}
	elements = append(elements, elem)

	return elements
}

// extractFlowMapping extracts flow-style mapping {key: value}
func (p *YAMLParser) extractFlowMapping(node *sitter.Node, parentID string, path []string) []Element {
	var elements []Element

	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	pathStr := strings.Join(path, ".")

	elem := Element{
		ElementID:       generateID("yaml_flow_map"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind": "flow_mapping",
			"value":     content,
			"path":      pathStr,
			"format":    "yaml",
		},
	}
	elements = append(elements, elem)

	return elements
}

// createCommentElement creates a comment element
func (p *YAMLParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean up comment marker
	content = strings.TrimPrefix(content, "#")
	content = strings.TrimSpace(content)

	return &Element{
		ElementID:       generateID("yaml_comment"),
		ElementType:     "code_documentation",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "documentation",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind": "comment",
			"format":   "yaml",
		},
	}
}

// Helper functions

func (p *YAMLParser) getNodeText(node *sitter.Node) string {
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

func (p *YAMLParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}
