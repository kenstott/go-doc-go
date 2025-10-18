package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
)

// BashCodeParser parses Bash shell scripts using tree-sitter
type BashCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	sourceCode        []byte
}

// NewBashCodeParser creates a new Bash code parser
func NewBashCodeParser() *BashCodeParser {
	return &BashCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
	}
}

// GetName returns the parser name
func (p *BashCodeParser) GetName() string {
	return "bash_code"
}

// GetSupportedFormats returns supported file formats
func (p *BashCodeParser) GetSupportedFormats() []string {
	return []string{".sh", ".bash", "sh", "bash"}
}

// SupportsStreaming returns false as this parser needs complete content
func (p *BashCodeParser) SupportsStreaming() bool {
	return false
}

// Close performs any cleanup needed
func (p *BashCodeParser) Close() error {
	return nil
}

// Parse parses Bash script and extracts code elements
func (p *BashCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from content
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Bash parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p.sourceCode = sourceCode

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(bash.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Bash source: %w", err)
	}
	defer tree.Close()

	// Create document element
	content := string(sourceCode)
	docElement := Element{
		ElementID:       generateID("bash_doc"),
		ElementType:     "document",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "document",
		Metadata: map[string]interface{}{
			"language": "bash",
			"parser":   "bash_code",
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
		DocType: "bash_source",
		Metadata: map[string]interface{}{
			"language": "bash",
			"parser":   "bash_code",
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

// extractElements recursively extracts Bash elements from AST
func (p *BashCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "function_definition":
			funcElements := p.createFunctionElements(child, parentID)
			elements = append(elements, funcElements...)

		case "variable_assignment":
			varElements := p.createVariableElements(child, parentID)
			elements = append(elements, varElements...)

		case "for_statement":
			loopElements := p.createLoopElements(child, parentID, "for")
			elements = append(elements, loopElements...)

		case "while_statement":
			loopElements := p.createLoopElements(child, parentID, "while")
			elements = append(elements, loopElements...)

		case "if_statement":
			condElements := p.createConditionalElements(child, parentID)
			elements = append(elements, condElements...)

		case "case_statement":
			caseElements := p.createCaseElements(child, parentID)
			elements = append(elements, caseElements...)

		case "pipeline":
			pipeElements := p.createPipelineElements(child, parentID)
			elements = append(elements, pipeElements...)

		case "command":
			cmdElements := p.createCommandElements(child, parentID)
			elements = append(elements, cmdElements...)

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

// createFunctionElements extracts function definitions
func (p *BashCodeParser) createFunctionElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract function name
	nameNode := p.findChildByType(node, "word")
	if nameNode == nil {
		return elements
	}

	functionName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	signature := fmt.Sprintf("function %s()", functionName)

	functionElement := Element{
		ElementID:       generateID("bash_function"),
		ElementType:     "code_function",
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ElementCategory: "code_logic",
		ParentID:        parentID,
		FunctionName:    &functionName,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "function",
			"language":          "bash",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, functionElement)

	// Extract function body
	bodyNode := p.findChildByType(node, "compound_statement")
	if bodyNode != nil {
		bodyElements := p.extractElements(bodyNode, functionElement.ElementID)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// createVariableElements extracts variable assignments
func (p *BashCodeParser) createVariableElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract variable name
	nameNode := p.findChildByType(node, "variable_name")
	if nameNode == nil {
		return elements
	}

	varName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract value if present
	var varValue string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "variable_name" && child.Type() != "=" {
			varValue = p.getNodeText(child)
			break
		}
	}

	elem := Element{
		ElementID:       generateID("bash_var"),
		ElementType:     "code_data",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "data_field",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":   "variable",
			"data_name":   varName,
			"value":       varValue,
			"language":    "bash",
			"line_number": lineNum,
		},
	}

	elements = append(elements, elem)

	return elements
}

// createLoopElements extracts loop statements
func (p *BashCodeParser) createLoopElements(node *sitter.Node, parentID string, loopType string) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	preview := truncate(content, p.MaxContentPreview)

	loopElement := Element{
		ElementID:       generateID("bash_loop"),
		ElementType:     "code_grouping",
		ContentPreview:  preview,
		ElementCategory: "code_logic",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": loopType + "_loop",
			"language":          "bash",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, loopElement)

	// Extract loop body
	bodyNode := p.findChildByType(node, "do_group")
	if bodyNode != nil {
		bodyElements := p.extractElements(bodyNode, loopElement.ElementID)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// createConditionalElements extracts if statements
func (p *BashCodeParser) createConditionalElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	preview := truncate(content, p.MaxContentPreview)

	condElement := Element{
		ElementID:       generateID("bash_if"),
		ElementType:     "code_grouping",
		ContentPreview:  preview,
		ElementCategory: "code_logic",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "conditional",
			"language":          "bash",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, condElement)

	// Extract if body and branches
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "compound_statement" || child.Type() == "elif_clause" || child.Type() == "else_clause" {
			branchElements := p.extractElements(child, condElement.ElementID)
			elements = append(elements, branchElements...)
		}
	}

	return elements
}

// createCaseElements extracts case statements
func (p *BashCodeParser) createCaseElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	preview := truncate(content, p.MaxContentPreview)

	caseElement := Element{
		ElementID:       generateID("bash_case"),
		ElementType:     "code_grouping",
		ContentPreview:  preview,
		ElementCategory: "code_logic",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "case_statement",
			"language":          "bash",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, caseElement)

	// Extract case items
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "case_item" {
			itemElements := p.extractElements(child, caseElement.ElementID)
			elements = append(elements, itemElements...)
		}
	}

	return elements
}

// createPipelineElements extracts pipeline commands
func (p *BashCodeParser) createPipelineElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	// Extract individual commands in the pipeline
	var commands []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "command" {
			cmdText := p.getNodeText(child)
			commands = append(commands, strings.TrimSpace(cmdText))
		}
	}

	if len(commands) > 0 {
		pipelinePreview := strings.Join(commands, " | ")

		elem := Element{
			ElementID:       generateID("bash_pipeline"),
			ElementType:     "code_function",
			ContentPreview:  truncate(pipelinePreview, p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"code_element_kind": "pipeline",
				"commands":          commands,
				"language":          "bash",
				"line_number":       lineNum,
			},
		}

		elements = append(elements, elem)
	} else {
		// Fallback: treat as single element
		elem := Element{
			ElementID:       generateID("bash_pipeline"),
			ElementType:     "code_function",
			ContentPreview:  truncate(content, p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"code_element_kind": "pipeline",
				"language":          "bash",
				"line_number":       lineNum,
			},
		}

		elements = append(elements, elem)
	}

	return elements
}

// createCommandElements extracts command invocations
func (p *BashCodeParser) createCommandElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract command name
	var commandName string
	nameNode := p.findChildByType(node, "command_name")
	if nameNode != nil {
		// Get the actual command from the word node
		wordNode := p.findChildByType(nameNode, "word")
		if wordNode != nil {
			commandName = p.getNodeText(wordNode)
		} else {
			commandName = p.getNodeText(nameNode)
		}
	}

	if commandName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	elem := Element{
		ElementID:       generateID("bash_cmd"),
		ElementType:     "code_function",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "code_logic",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "command",
			"command_name":      commandName,
			"language":          "bash",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, elem)

	return elements
}

// createCommentElement creates a comment element
func (p *BashCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean up comment marker
	content = strings.TrimPrefix(content, "#")
	content = strings.TrimSpace(content)

	return &Element{
		ElementID:       generateID("bash_comment"),
		ElementType:     "code_documentation",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "documentation",
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    "comment",
			"language":    "bash",
			"line_number": lineNum,
		},
	}
}

// Helper functions

func (p *BashCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *BashCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}
