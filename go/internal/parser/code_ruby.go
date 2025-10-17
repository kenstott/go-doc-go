package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ruby"
)

// RubyCodeParser parses Ruby source code using tree-sitter
type RubyCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	sourceCode        []byte
	moduleName        string
	fileExtension     string
}

// NewRubyCodeParser creates a new Ruby parser
func NewRubyCodeParser() *RubyCodeParser {
	return &RubyCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *RubyCodeParser) GetName() string {
	return "ruby_code"
}

// GetSupportedFormats returns supported file extensions
func (p *RubyCodeParser) GetSupportedFormats() []string {
	return []string{".rb", "rb"}
}

// Close releases any resources held by the parser
func (p *RubyCodeParser) Close() error {
	return nil
}

// SupportsStreaming returns whether the parser supports streaming mode
func (p *RubyCodeParser) SupportsStreaming() bool {
	return false
}

// Parse parses Ruby source code
func (p *RubyCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Ruby parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Ruby file: %w", err)
	}

	p.sourceCode = sourceCode
	p.moduleName = getModuleName(filePath)
	p.fileExtension = ".rb"

	// Apply configuration
	if req.Config.MaxContentPreview > 0 {
		p.MaxContentPreview = req.Config.MaxContentPreview
	}

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(ruby.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Ruby: %w", err)
	}
	defer tree.Close()

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "ruby_source",
		Title:   p.moduleName,
		Metadata: map[string]interface{}{
			"module":         p.moduleName,
			"path":           filePath,
			"language":       "ruby",
			"file_extension": p.fileExtension,
		},
	}

	// Create root document element
	rootNode := tree.RootNode()
	docElement := p.createDocumentElement(req.ID, filePath, sourceCode)
	elements := []Element{docElement}

	// Extract elements
	extractedElements := p.extractElements(rootNode, req.ID, docElement.ElementID)
	elements = append(elements, extractedElements...)

	return &ParseResult{
		Document: doc,
		Elements: elements,
	}, nil
}

// createDocumentElement creates the root document element
func (p *RubyCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
	preview := string(sourceCode)
	if len(preview) > p.MaxContentPreview {
		preview = preview[:p.MaxContentPreview] + "..."
	}

	return Element{
		ElementID:       id,
		ElementType:     "document",
		ElementCategory: GetElementCategory("document"),
		Content:         string(sourceCode),
		ContentPreview:  preview,
		ParentID:        "",
		Metadata: map[string]interface{}{
			"doc_type":       "ruby_source",
			"module":         p.moduleName,
			"path":           filePath,
			"language":       "ruby",
			"file_extension": p.fileExtension,
		},
	}
}

// extractElements extracts code elements from the AST
func (p *RubyCodeParser) extractElements(node *sitter.Node, docID, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "call":
			// Check if it's a require/require_relative
			if p.isRequireStatement(child) && p.ExtractImports {
				elem := p.createImportElement(child, docID, parentID)
				elements = append(elements, elem)
			}

		case "class":
			classElems := p.createClassElements(child, docID, parentID)
			elements = append(elements, classElems...)

		case "module":
			moduleElems := p.createModuleElements(child, docID, parentID)
			elements = append(elements, moduleElems...)

		case "method", "singleton_method":
			methodElem := p.createMethodElement(child, docID, parentID, "")
			elements = append(elements, methodElem)

			// Extract function calls within method body
			if p.ExtractImports {
				callElems := p.extractFunctionCalls(child, docID, methodElem.ElementID)
				elements = append(elements, callElems...)
			}

			// Extract inline comments
			if p.ExtractComments {
				commentElems := p.extractInlineComments(child, docID, methodElem.ElementID)
				elements = append(elements, commentElems...)
			}

		case "comment":
			if p.ExtractComments {
				// Top-level comments are attached to the document
				commentElem := p.createCommentElement(child, docID, parentID, "comment")
				elements = append(elements, commentElem)
			}

		case "assignment":
			// Constants and class variables
			assignElem := p.createAssignmentElement(child, docID, parentID)
			if assignElem != nil {
				elements = append(elements, *assignElem)
			}

		case "constant_assignment":
			constElem := p.createConstantElement(child, docID, parentID)
			elements = append(elements, constElem)
		}
	}

	return elements
}

// isRequireStatement checks if a call node is a require/require_relative
func (p *RubyCodeParser) isRequireStatement(node *sitter.Node) bool {
	if node.Type() != "call" {
		return false
	}

	methodNode := node.ChildByFieldName("method")
	if methodNode == nil {
		return false
	}

	methodName := p.getNodeText(methodNode)
	return methodName == "require" || methodName == "require_relative"
}

// createImportElement creates an import element from require/require_relative
func (p *RubyCodeParser) createImportElement(node *sitter.Node, docID, parentID string) Element {
	methodNode := node.ChildByFieldName("method")
	argsNode := node.ChildByFieldName("arguments")

	methodName := p.getNodeText(methodNode)
	importPath := ""

	if argsNode != nil {
		// Get the first argument (the string being required)
		for i := 0; i < int(argsNode.ChildCount()); i++ {
			child := argsNode.Child(i)
			if child.Type() == "string" {
				importPath = p.extractStringContent(child)
				break
			}
		}
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	return Element{
		ElementID:       generateID("ruby_import"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		ParentID:        parentID,
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind":   "import",
			"import_type":       methodName,
			"target_namespace":  importPath,
			"language":          "ruby",
			"line_number":       lineNum,
		},
	}
}

// createClassElements creates elements for a class definition
func (p *RubyCodeParser) createClassElements(node *sitter.Node, docID, parentID string) []Element {
	var elements []Element

	// Get class name
	nameNode := p.findChildByType(node, "constant")
	if nameNode == nil {
		return elements
	}

	className := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Check for preceding comment (class documentation)
	var classDoc string
	if prevSibling := p.getPreviousSibling(node); prevSibling != nil && prevSibling.Type() == "comment" {
		classDoc = p.extractCommentText(prevSibling)
	}

	// Create class element
	classElem := Element{
		ElementID:       generateID("ruby_class"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		ParentID:        parentID,
		Content:         fmt.Sprintf("class %s", className),
		ContentPreview:  truncate(fmt.Sprintf("class %s", className), p.MaxContentPreview),
		ClassName:       &className,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "class",
			"class_name":        className,
			"language":          "ruby",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, classElem)

	// Add class documentation if found
	if classDoc != "" {
		docElem := Element{
			ElementID:       generateID("ruby_doc"),
			ElementType:     "code_documentation",
			ElementCategory: GetElementCategory("code_documentation"),
			ParentID:        classElem.ElementID,
			Content:         classDoc,
			ContentPreview:  truncate(classDoc, p.MaxContentPreview),
			Metadata: map[string]interface{}{
				"doc_kind":    "class_comment",
				"language":    "ruby",
				"line_number": lineNum - 1,
			},
		}
		elements = append(elements, docElem)
	}

	// Extract class body
	bodyNode := p.findChildByType(node, "body_statement")
	if bodyNode != nil {
		bodyElements := p.extractClassBody(bodyNode, docID, classElem.ElementID, className)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// extractClassBody extracts elements from a class body
func (p *RubyCodeParser) extractClassBody(node *sitter.Node, docID, parentID, className string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "method", "singleton_method":
			methodElem := p.createMethodElement(child, docID, parentID, className)
			elements = append(elements, methodElem)

			// Extract function calls within method
			if p.ExtractImports {
				callElems := p.extractFunctionCalls(child, docID, methodElem.ElementID)
				elements = append(elements, callElems...)
			}

			// Extract inline comments
			if p.ExtractComments {
				commentElems := p.extractInlineComments(child, docID, methodElem.ElementID)
				elements = append(elements, commentElems...)
			}

		case "call":
			// attr_reader, attr_writer, attr_accessor
			if p.isAttributeDeclaration(child) {
				attrElems := p.createAttributeElements(child, docID, parentID, className)
				elements = append(elements, attrElems...)
			}

		case "comment":
			// Comments within class body
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, docID, parentID, "comment")
				elements = append(elements, commentElem)
			}

		case "assignment":
			// Class-level constants and variables
			assignElem := p.createAssignmentElement(child, docID, parentID)
			if assignElem != nil {
				elements = append(elements, *assignElem)
			}

		case "constant_assignment":
			// Class constants
			constElem := p.createConstantElement(child, docID, parentID)
			elements = append(elements, constElem)
		}
	}

	return elements
}

// isAttributeDeclaration checks if a call is attr_reader/attr_writer/attr_accessor
func (p *RubyCodeParser) isAttributeDeclaration(node *sitter.Node) bool {
	if node.Type() != "call" {
		return false
	}

	methodNode := node.ChildByFieldName("method")
	if methodNode == nil {
		return false
	}

	methodName := p.getNodeText(methodNode)
	return methodName == "attr_reader" || methodName == "attr_writer" || methodName == "attr_accessor"
}

// createAttributeElements creates elements for attr_reader/attr_writer/attr_accessor
func (p *RubyCodeParser) createAttributeElements(node *sitter.Node, docID, parentID, className string) []Element {
	var elements []Element

	methodNode := node.ChildByFieldName("method")
	argsNode := node.ChildByFieldName("arguments")

	attrType := p.getNodeText(methodNode)
	lineNum := int(node.StartPoint().Row) + 1

	if argsNode != nil {
		// Extract each attribute
		for i := 0; i < int(argsNode.ChildCount()); i++ {
			child := argsNode.Child(i)
			if child.Type() == "simple_symbol" {
				attrName := strings.TrimPrefix(p.getNodeText(child), ":")

				elem := Element{
					ElementID:       generateID("ruby_attr"),
					ElementType:     "code_data",
					ElementCategory: GetElementCategory("code_data"),
					ParentID:        parentID,
					Content:         fmt.Sprintf("%s :%s", attrType, attrName),
					ContentPreview:  truncate(fmt.Sprintf("%s :%s", attrType, attrName), p.MaxContentPreview),
					ClassName:       &className,
					LineNumber:      &lineNum,
					Metadata: map[string]interface{}{
						"code_element_kind": "attribute",
						"attribute_name":    attrName,
						"attribute_type":    attrType,
						"class_name":        className,
						"language":          "ruby",
						"line_number":       lineNum,
					},
				}
				elements = append(elements, elem)
			}
		}
	}

	return elements
}

// createModuleElements creates elements for a module definition
func (p *RubyCodeParser) createModuleElements(node *sitter.Node, docID, parentID string) []Element {
	var elements []Element

	// Get module name
	nameNode := p.findChildByType(node, "constant")
	if nameNode == nil {
		return elements
	}

	moduleName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Check for preceding comment (module documentation)
	var moduleDoc string
	if prevSibling := p.getPreviousSibling(node); prevSibling != nil && prevSibling.Type() == "comment" {
		moduleDoc = p.extractCommentText(prevSibling)
	}

	// Create module element
	moduleElem := Element{
		ElementID:       generateID("ruby_module"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		ParentID:        parentID,
		Content:         fmt.Sprintf("module %s", moduleName),
		ContentPreview:  truncate(fmt.Sprintf("module %s", moduleName), p.MaxContentPreview),
		Namespace:       &moduleName,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "module",
			"module_name":       moduleName,
			"language":          "ruby",
			"line_number":       lineNum,
		},
	}

	elements = append(elements, moduleElem)

	// Add module documentation if found
	if moduleDoc != "" {
		docElem := Element{
			ElementID:       generateID("ruby_doc"),
			ElementType:     "code_documentation",
			ElementCategory: GetElementCategory("code_documentation"),
			ParentID:        moduleElem.ElementID,
			Content:         moduleDoc,
			ContentPreview:  truncate(moduleDoc, p.MaxContentPreview),
			Metadata: map[string]interface{}{
				"doc_kind":    "module_comment",
				"language":    "ruby",
				"line_number": lineNum - 1,
			},
		}
		elements = append(elements, docElem)
	}

	// Extract module body
	bodyNode := p.findChildByType(node, "body_statement")
	if bodyNode != nil {
		bodyElements := p.extractModuleBody(bodyNode, docID, moduleElem.ElementID, moduleName)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// extractModuleBody extracts elements from a module body
func (p *RubyCodeParser) extractModuleBody(node *sitter.Node, docID, parentID, moduleName string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "method", "singleton_method":
			methodElem := p.createMethodElement(child, docID, parentID, "")

			// Update namespace for module methods
			methodElem.Namespace = &moduleName
			if methodElem.Metadata == nil {
				methodElem.Metadata = make(map[string]interface{})
			}
			methodElem.Metadata["module_name"] = moduleName

			elements = append(elements, methodElem)

			// Extract function calls within method
			if p.ExtractImports {
				callElems := p.extractFunctionCalls(child, docID, methodElem.ElementID)
				elements = append(elements, callElems...)
			}

			// Extract inline comments
			if p.ExtractComments {
				commentElems := p.extractInlineComments(child, docID, methodElem.ElementID)
				elements = append(elements, commentElems...)
			}

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, docID, parentID, "comment")
				elements = append(elements, commentElem)
			}
		}
	}

	return elements
}

// createMethodElement creates a method element
func (p *RubyCodeParser) createMethodElement(node *sitter.Node, docID, parentID, className string) Element {
	isSingleton := node.Type() == "singleton_method"

	// Get method name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		// Fallback for operators or special methods
		nameNode = node.Child(1)
	}

	methodName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Build signature
	signature := p.buildMethodSignature(node, methodName, isSingleton)

	// Check for preceding comment (method documentation)
	var methodDoc string
	if prevSibling := p.getPreviousSibling(node); prevSibling != nil && prevSibling.Type() == "comment" {
		methodDoc = p.extractCommentText(prevSibling)
	}

	// Determine method kind
	methodKind := "method"
	if isSingleton {
		methodKind = "class_method"
	} else if className != "" {
		methodKind = "instance_method"
	} else {
		methodKind = "function"
	}

	elem := Element{
		ElementID:       generateID("ruby_func"),
		ElementType:     "code_function",
		ElementCategory: GetElementCategory("code_function"),
		ParentID:        parentID,
		Content:         signature,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		FunctionName:    &methodName,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": methodKind,
			"method_name":       methodName,
			"is_singleton":      isSingleton,
			"language":          "ruby",
			"line_number":       lineNum,
		},
	}

	if className != "" {
		elem.ClassName = &className
		elem.Metadata["class_name"] = className
	}

	// Add method documentation if found
	if methodDoc != "" {
		// Return as a separate element in the calling function
		// For now, just add to metadata
		elem.Metadata["docstring"] = methodDoc
	}

	return elem
}

// buildMethodSignature builds a method signature string
func (p *RubyCodeParser) buildMethodSignature(node *sitter.Node, methodName string, isSingleton bool) string {
	var sig strings.Builder

	sig.WriteString("def ")

	if isSingleton {
		sig.WriteString("self.")
	}

	sig.WriteString(methodName)

	// Get parameters
	paramsNode := p.findChildByType(node, "method_parameters")
	if paramsNode != nil {
		params := p.extractMethodParameters(paramsNode)
		if len(params) > 0 {
			sig.WriteString("(")
			sig.WriteString(strings.Join(params, ", "))
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// extractMethodParameters extracts parameter names from method_parameters node
func (p *RubyCodeParser) extractMethodParameters(node *sitter.Node) []string {
	var params []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "identifier":
			params = append(params, p.getNodeText(child))
		case "optional_parameter":
			// Parameter with default value
			idNode := p.findChildByType(child, "identifier")
			if idNode != nil {
				params = append(params, p.getNodeText(idNode)+" = ...")
			}
		case "splat_parameter":
			// *args
			params = append(params, "*"+p.getNodeText(child.Child(1)))
		case "hash_splat_parameter":
			// **kwargs
			params = append(params, "**"+p.getNodeText(child.Child(1)))
		case "block_parameter":
			// &block
			params = append(params, "&"+p.getNodeText(child.Child(1)))
		}
	}

	return params
}

// createConstantElement creates an element for a constant assignment
func (p *RubyCodeParser) createConstantElement(node *sitter.Node, docID, parentID string) Element {
	// Get constant name
	constNode := p.findChildByType(node, "constant")
	if constNode == nil {
		constNode = node.Child(0)
	}

	constName := p.getNodeText(constNode)
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	return Element{
		ElementID:       generateID("ruby_const"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		ParentID:        parentID,
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "constant",
			"constant_name":     constName,
			"language":          "ruby",
			"line_number":       lineNum,
		},
	}
}

// createAssignmentElement creates an element for variable assignments
func (p *RubyCodeParser) createAssignmentElement(node *sitter.Node, docID, parentID string) *Element {
	// Handle instance variables (@var), class variables (@@var), and constants (CONSTANT)
	leftNode := node.Child(0)
	if leftNode == nil {
		return nil
	}

	leftText := p.getNodeText(leftNode)

	// Skip regular local variables (lowercase)
	if !strings.HasPrefix(leftText, "@") && (len(leftText) == 0 || (leftText[0] >= 'a' && leftText[0] <= 'z')) {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	var varKind string
	var elementID string

	if strings.HasPrefix(leftText, "@@") {
		varKind = "class_variable"
		elementID = "ruby_var"
	} else if strings.HasPrefix(leftText, "@") {
		varKind = "instance_variable"
		elementID = "ruby_var"
	} else if len(leftText) > 0 && leftText[0] >= 'A' && leftText[0] <= 'Z' {
		// Constants in Ruby start with uppercase letter
		varKind = "constant"
		elementID = "ruby_const"
	} else {
		return nil
	}

	metadata := map[string]interface{}{
		"code_element_kind": varKind,
		"language":          "ruby",
		"line_number":       lineNum,
	}

	if varKind == "constant" {
		metadata["constant_name"] = leftText
	} else {
		metadata["variable_name"] = leftText
	}

	elem := Element{
		ElementID:       generateID(elementID),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		ParentID:        parentID,
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		LineNumber:      &lineNum,
		Metadata:        metadata,
	}

	return &elem
}

// extractFunctionCalls extracts function call dependencies from a method
func (p *RubyCodeParser) extractFunctionCalls(node *sitter.Node, docID, parentID string) []Element {
	var elements []Element

	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		if n.Type() == "call" {
			// Extract function call
			methodNode := n.ChildByFieldName("method")
			if methodNode != nil {
				funcName := p.getNodeText(methodNode)
				lineNum := int(n.StartPoint().Row) + 1

				elem := Element{
					ElementID:       generateID("ruby_call"),
					ElementType:     "code_dependency",
					ElementCategory: GetElementCategory("code_dependency"),
					ParentID:        parentID,
					Content:         p.getNodeText(n),
					ContentPreview:  truncate(p.getNodeText(n), p.MaxContentPreview),
					LineNumber:      &lineNum,
					Metadata: map[string]interface{}{
						"dependency_kind": "function_call",
						"target_function": funcName,
						"language":        "ruby",
						"line_number":     lineNum,
					},
				}
				elements = append(elements, elem)
			}
		}

		// Recurse into children
		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(i))
		}
	}

	// Only traverse method body
	bodyNode := p.findChildByType(node, "body_statement")
	if bodyNode != nil {
		traverse(bodyNode)
	}

	return elements
}

// extractInlineComments extracts inline comments from a method body
func (p *RubyCodeParser) extractInlineComments(node *sitter.Node, docID, parentID string) []Element {
	var elements []Element

	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		// Process this node if it's a comment
		if n.Type() == "comment" {
			commentText := p.extractCommentText(n)
			lineNum := int(n.StartPoint().Row) + 1

			// Detect markers
			markers := detectCommentMarkers(commentText)

			elem := Element{
				ElementID:       generateID("ruby_inline_comment"),
				ElementType:     "code_documentation",
				ElementCategory: GetElementCategory("code_documentation"),
				ParentID:        parentID,
				Content:         commentText,
				ContentPreview:  truncate(commentText, p.MaxContentPreview),
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"doc_kind":     "inline_comment",
					"comment_type": "line_comment",
					"language":     "ruby",
					"line_number":  lineNum,
				},
			}

			if len(markers) > 0 {
				elem.Metadata["markers"] = markers
			}

			elements = append(elements, elem)
		}

		// Recurse into children
		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(i))
		}
	}

	// Traverse the entire method node to find all comments
	traverse(node)
	return elements
}

// createCommentElement creates a comment element
func (p *RubyCodeParser) createCommentElement(node *sitter.Node, docID, parentID, commentType string) Element {
	commentText := p.extractCommentText(node)
	lineNum := int(node.StartPoint().Row) + 1

	return Element{
		ElementID:       generateID("ruby_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		ParentID:        parentID,
		Content:         commentText,
		ContentPreview:  truncate(commentText, p.MaxContentPreview),
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":     commentType,
			"comment_type": "line_comment",
			"language":     "ruby",
			"line_number":  lineNum,
		},
	}
}

// Helper functions

func (p *RubyCodeParser) getNodeText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	if start >= uint32(len(p.sourceCode)) || end > uint32(len(p.sourceCode)) {
		return ""
	}
	return string(p.sourceCode[start:end])
}

func (p *RubyCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *RubyCodeParser) getPreviousSibling(node *sitter.Node) *sitter.Node {
	parent := node.Parent()
	if parent == nil {
		return nil
	}

	var prevSibling *sitter.Node
	for i := 0; i < int(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child == node {
			return prevSibling
		}
		prevSibling = child
	}
	return nil
}

func (p *RubyCodeParser) extractCommentText(node *sitter.Node) string {
	text := p.getNodeText(node)
	// Remove leading # and whitespace
	text = strings.TrimPrefix(text, "#")
	text = strings.TrimSpace(text)
	return text
}

func (p *RubyCodeParser) extractStringContent(node *sitter.Node) string {
	// Find string_content child
	contentNode := p.findChildByType(node, "string_content")
	if contentNode != nil {
		return p.getNodeText(contentNode)
	}

	// Fallback: remove quotes
	text := p.getNodeText(node)
	text = strings.Trim(text, "\"'")
	return text
}

// detectCommentMarkers is now imported from code_java.go or another code parser file
// It detects technical debt markers in comments
