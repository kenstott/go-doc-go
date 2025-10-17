package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// PythonCodeParser parses Python source code files using tree-sitter
type PythonCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	sourceCode        []byte
	moduleName        string
	classStack        []string // Stack to track nested class names
}

// NewPythonCodeParser creates a new Python code parser with default settings
func NewPythonCodeParser() *PythonCodeParser {
	return &PythonCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *PythonCodeParser) GetName() string {
	return "python_code"
}

// GetSupportedFormats returns supported file formats
func (p *PythonCodeParser) GetSupportedFormats() []string {
	return []string{".py", "py"}
}

// SupportsStreaming indicates if parser can handle streaming content
func (p *PythonCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *PythonCodeParser) Close() error {
	return nil
}

// Parse implements the Parser interface for Python source code
func (p *PythonCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Python parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Python file: %w", err)
	}

	p.sourceCode = sourceCode
	p.moduleName = getModuleName(filePath)

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python source: %w", err)
	}
	defer tree.Close()

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "python_source",
		Title:   p.moduleName,
		Metadata: map[string]interface{}{
			"module":   p.moduleName,
			"path":     filePath,
			"language": "python",
		},
	}

	// Create document element
	docElement := p.createDocumentElement(req.ID, filePath, sourceCode)
	elements := []Element{docElement}

	// Extract elements from AST
	rootNode := tree.RootNode()
	extractedElements := p.extractElements(rootNode, docElement.ElementID)
	elements = append(elements, extractedElements...)

	return &ParseResult{
		Document: doc,
		Elements: elements,
	}, nil
}

func (p *PythonCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
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
			"doc_type": "python_source",
			"module":   p.moduleName,
			"path":     filePath,
			"language": "python",
		},
	}
}

func (p *PythonCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "import_statement", "import_from_statement":
			elem := p.createImportElement(child, parentID)
			if elem != nil {
				elements = append(elements, *elem)
			}

		case "function_definition":
			funcElements := p.createFunctionElements(child, parentID)
			elements = append(elements, funcElements...)

		case "decorated_definition":
			// Handle decorated functions/classes (e.g., @property, @staticmethod)
			for j := 0; j < int(child.ChildCount()); j++ {
				defChild := child.Child(j)
				if defChild.Type() == "function_definition" {
					funcElements := p.createFunctionElements(defChild, parentID)
					elements = append(elements, funcElements...)
					break
				} else if defChild.Type() == "class_definition" {
					classElements := p.createClassElements(defChild, parentID)
					elements = append(elements, classElements...)
					break
				}
			}

		case "class_definition":
			classElements := p.createClassElements(child, parentID)
			elements = append(elements, classElements...)

		case "assignment":
			// Module-level variables (only at module scope, not inside functions/classes)
			if len(p.classStack) == 0 {
				varElem := p.createVariableElement(child, parentID)
				if varElem != nil {
					elements = append(elements, *varElem)
				}
			}

		case "expression_statement":
			// Check for docstrings or assignments
			if child.ChildCount() > 0 {
				firstChild := child.Child(0)
				if firstChild.Type() == "string" {
					docElem := p.createDocstringElement(child, parentID)
					if docElem != nil {
						elements = append(elements, *docElem)
					}
				} else if firstChild.Type() == "assignment" && len(p.classStack) == 0 {
					// Module-level assignment wrapped in expression_statement
					varElem := p.createVariableElement(firstChild, parentID)
					if varElem != nil {
						elements = append(elements, *varElem)
					}
				}
			}

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, parentID)
				if commentElem != nil {
					elements = append(elements, *commentElem)
				}
			}
		}

		// Don't recursively process children - we handle them explicitly above
		// to avoid duplicate elements
	}

	return elements
}

func (p *PythonCodeParser) createImportElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	targetNamespace := ""
	importAlias := ""
	dependencyKind := "import"

	if node.Type() == "import_statement" {
		// import foo
		// import foo as bar
		dotted := p.findChildByType(node, "dotted_name")
		if dotted == nil {
			dotted = p.findChildByType(node, "aliased_import")
			if dotted != nil {
				nameNode := p.findChildByType(dotted, "dotted_name")
				if nameNode != nil {
					targetNamespace = p.getNodeText(nameNode)
				}
				aliasNode := p.findChildByType(dotted, "identifier")
				if aliasNode != nil {
					importAlias = p.getNodeText(aliasNode)
				}
			}
		} else {
			targetNamespace = p.getNodeText(dotted)
		}
	} else {
		// from foo import bar
		dependencyKind = "from_import"
		dotted := p.findChildByType(node, "dotted_name")
		if dotted != nil {
			targetNamespace = p.getNodeText(dotted)
		}
	}

	namespace := p.moduleName

	metadata := map[string]interface{}{
		"dependency_kind":  dependencyKind,
		"target_namespace": targetNamespace,
		"language":         "python",
		"line_number":      lineNum,
	}

	if importAlias != "" {
		metadata["dependency_alias"] = importAlias
	}

	return &Element{
		ElementID:       generateID("py_import"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata:        metadata,
	}
}

func (p *PythonCodeParser) createFunctionElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract function name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	funcName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Build signature
	paramsNode := p.findChildByType(node, "parameters")
	params := ""
	if paramsNode != nil {
		params = p.getNodeText(paramsNode)
	}

	// Check for return type annotation
	returnType := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type" {
			returnType = " -> " + p.getNodeText(child)
			break
		}
	}

	signature := fmt.Sprintf("def %s%s%s", funcName, params, returnType)

	// Determine function kind and visibility
	funcKind := "function"
	visibility := "public"

	// Check if it's a method (has 'self' or 'cls' parameter)
	if paramsNode != nil && paramsNode.ChildCount() > 0 {
		firstParam := paramsNode.Child(1) // Skip opening paren
		if firstParam != nil && firstParam.Type() == "identifier" {
			firstParamName := p.getNodeText(firstParam)
			if firstParamName == "self" {
				funcKind = "method"
			} else if firstParamName == "cls" {
				funcKind = "class_method"
			}
		}
	}

	// Check for decorators
	decorators := p.extractDecorators(node)
	for _, decorator := range decorators {
		if decorator == "@staticmethod" {
			funcKind = "static_method"
		} else if decorator == "@classmethod" {
			funcKind = "class_method"
		} else if decorator == "@property" {
			funcKind = "property"
		}
	}

	// Private function check
	if strings.HasPrefix(funcName, "_") && !strings.HasPrefix(funcName, "__") {
		visibility = "private"
	} else if strings.HasPrefix(funcName, "__") {
		visibility = "private"
	}

	namespace := p.moduleName
	className := p.extractClassName(parentID)
	if className != "" {
		namespace = fmt.Sprintf("%s.%s", p.moduleName, className)
	}

	content := p.getNodeText(node)

	funcElement := Element{
		ElementID:       generateID("py_func"),
		ElementType:     "code_function",
		ElementCategory: GetElementCategory("code_function"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		FunctionName:    &funcName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": funcKind,
			"visibility":        visibility,
			"language":          "python",
			"decorators":        decorators,
			"line_number":       lineNum,
		},
	}

	if className != "" {
		funcElement.ClassName = &className
	}

	elements = append(elements, funcElement)

	// Extract parameters as code_data elements with type usage
	if paramsNode != nil {
		paramElements := p.extractParameters(paramsNode, funcElement.ElementID)
		elements = append(elements, paramElements...)
	}

	// Extract return type as type usage
	if returnType != "" {
		returnTypeElem := p.createTypeUsageElement(strings.TrimPrefix(returnType, " -> "), funcElement.ElementID, "return")
		if returnTypeElem != nil {
			elements = append(elements, *returnTypeElem)
		}
	}

	// Extract function body for function calls
	bodyNode := p.findChildByType(node, "block")
	if bodyNode != nil {
		callElements := p.extractFunctionCalls(bodyNode, funcElement.ElementID)
		elements = append(elements, callElements...)

		// Extract inline comments within function body
		if p.ExtractComments {
			inlineComments := p.extractInlineComments(bodyNode, funcElement.ElementID)
			elements = append(elements, inlineComments...)
		}
	}

	// Extract docstring
	if bodyNode != nil && bodyNode.ChildCount() > 0 {
		firstStatement := bodyNode.Child(0)
		if firstStatement.Type() == "expression_statement" && firstStatement.ChildCount() > 0 {
			stringNode := firstStatement.Child(0)
			if stringNode.Type() == "string" {
				docElem := p.createDocstringElement(firstStatement, funcElement.ElementID)
				if docElem != nil {
					elements = append(elements, *docElem)
				}
			}
		}
	}

	return elements
}

func (p *PythonCodeParser) createClassElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract class name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	className := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract base classes
	baseClasses := []string{}
	argListNode := p.findChildByType(node, "argument_list")
	if argListNode != nil {
		for i := 0; i < int(argListNode.ChildCount()); i++ {
			child := argListNode.Child(i)
			if child.Type() == "identifier" {
				baseClasses = append(baseClasses, p.getNodeText(child))
			}
		}
	}

	signature := "class " + className
	if len(baseClasses) > 0 {
		signature += "(" + strings.Join(baseClasses, ", ") + ")"
	}

	namespace := p.moduleName
	content := p.getNodeText(node)

	// Determine visibility
	visibility := "public"
	if strings.HasPrefix(className, "_") {
		visibility = "private"
	}

	classElement := Element{
		ElementID:       generateID("py_class"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &className,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "class",
			"visibility":        visibility,
			"language":          "python",
			"base_classes":      baseClasses,
			"line_number":       lineNum,
		},
	}

	elements = append(elements, classElement)

	// Extract class body (methods, fields, nested classes)
	bodyNode := p.findChildByType(node, "block")
	if bodyNode != nil {
		// Push class name onto stack
		p.classStack = append(p.classStack, className)
		bodyElements := p.extractClassBody(bodyNode, classElement.ElementID, className)
		elements = append(elements, bodyElements...)
		// Pop class name from stack
		if len(p.classStack) > 0 {
			p.classStack = p.classStack[:len(p.classStack)-1]
		}
	}

	// Extract docstring
	if bodyNode != nil && bodyNode.ChildCount() > 0 {
		firstStatement := bodyNode.Child(0)
		if firstStatement.Type() == "expression_statement" && firstStatement.ChildCount() > 0 {
			stringNode := firstStatement.Child(0)
			if stringNode.Type() == "string" {
				docElem := p.createDocstringElement(firstStatement, classElement.ElementID)
				if docElem != nil {
					elements = append(elements, *docElem)
				}
			}
		}
	}

	return elements
}

func (p *PythonCodeParser) extractClassBody(bodyNode *sitter.Node, parentID, className string) []Element {
	var elements []Element

	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)

		switch child.Type() {
		case "function_definition":
			funcElements := p.createFunctionElements(child, parentID)
			elements = append(elements, funcElements...)

		case "decorated_definition":
			// Handle decorated methods (e.g., @property, @staticmethod)
			for j := 0; j < int(child.ChildCount()); j++ {
				defChild := child.Child(j)
				if defChild.Type() == "function_definition" {
					funcElements := p.createFunctionElements(defChild, parentID)
					elements = append(elements, funcElements...)
					break
				} else if defChild.Type() == "class_definition" {
					nestedClassElements := p.createClassElements(defChild, parentID)
					elements = append(elements, nestedClassElements...)
					break
				}
			}

		case "class_definition":
			nestedClassElements := p.createClassElements(child, parentID)
			elements = append(elements, nestedClassElements...)

		case "expression_statement":
			// Assignment statements (class variables)
			if child.ChildCount() > 0 {
				assignNode := child.Child(0)
				if assignNode.Type() == "assignment" {
					varElem := p.createClassVariableElement(assignNode, parentID, className)
					if varElem != nil {
						elements = append(elements, *varElem)
					}
				}
			}
		}
	}

	return elements
}

func (p *PythonCodeParser) createClassVariableElement(node *sitter.Node, parentID, className string) *Element {
	// Extract left side (variable name)
	var varName string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			varName = p.getNodeText(child)
			break
		}
	}

	if varName == "" {
		return nil
	}

	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := fmt.Sprintf("%s.%s", p.moduleName, className)

	visibility := "public"
	if strings.HasPrefix(varName, "_") {
		visibility = "private"
	}

	return &Element{
		ElementID:       generateID("py_field"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		ClassName:       &className,
		Metadata: map[string]interface{}{
			"data_kind":   "field",
			"data_name":   varName,
			"scope":       "class",
			"mutability":  "mutable",
			"visibility":  visibility,
			"language":    "python",
			"line_number": lineNum,
		},
	}
}

func (p *PythonCodeParser) createVariableElement(node *sitter.Node, parentID string) *Element {
	// Extract variable name
	var varName string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			varName = p.getNodeText(child)
			break
		}
	}

	if varName == "" {
		return nil
	}

	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := p.moduleName

	visibility := "public"
	if strings.HasPrefix(varName, "_") {
		visibility = "private"
	}

	return &Element{
		ElementID:       generateID("py_var"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":   "variable",
			"data_name":   varName,
			"scope":       "module",
			"mutability":  "mutable",
			"visibility":  visibility,
			"language":    "python",
			"line_number": lineNum,
		},
	}
}

func (p *PythonCodeParser) extractParameters(paramsNode *sitter.Node, parentFuncID string) []Element {
	var elements []Element

	for i := 0; i < int(paramsNode.ChildCount()); i++ {
		child := paramsNode.Child(i)

		var paramName string
		var paramType string

		switch child.Type() {
		case "identifier":
			paramName = p.getNodeText(child)

		case "typed_parameter":
			// name: type
			nameNode := p.findChildByType(child, "identifier")
			if nameNode != nil {
				paramName = p.getNodeText(nameNode)
			}
			typeNode := p.findChildByType(child, "type")
			if typeNode != nil {
				paramType = p.getNodeText(typeNode)
			}

		case "default_parameter":
			// name = default or name: type = default
			nameNode := p.findChildByType(child, "identifier")
			if nameNode != nil {
				paramName = p.getNodeText(nameNode)
			}
			typeNode := p.findChildByType(child, "type")
			if typeNode != nil {
				paramType = p.getNodeText(typeNode)
			}
		}

		if paramName == "" || paramName == "self" || paramName == "cls" {
			continue
		}

		lineNum := int(child.StartPoint().Row) + 1
		content := p.getNodeText(child)

		paramElement := Element{
			ElementID:       generateID("py_param"),
			ElementType:     "code_data",
			ElementCategory: GetElementCategory("code_data"),
			Content:         content,
			ContentPreview:  truncate(content, p.MaxContentPreview),
			ParentID:        parentFuncID,
			LineNumber:      &lineNum,
			Metadata: map[string]interface{}{
				"data_kind":   "parameter",
				"data_name":   paramName,
				"scope":       "local",
				"mutability":  "mutable",
				"language":    "python",
				"line_number": lineNum,
			},
		}

		if paramType != "" {
			paramElement.Metadata["data_type"] = paramType
		}

		elements = append(elements, paramElement)

		// Create type usage dependency if type is specified
		if paramType != "" {
			typeUsageElem := p.createTypeUsageElement(paramType, parentFuncID, "parameter")
			if typeUsageElem != nil {
				elements = append(elements, *typeUsageElem)
			}
		}
	}

	return elements
}

func (p *PythonCodeParser) extractFunctionCalls(bodyNode *sitter.Node, parentFuncID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "call" {
			callElem := p.createFunctionCallElement(node, parentFuncID)
			if callElem != nil {
				elements = append(elements, *callElem)
			}
		}
	})

	return elements
}

func (p *PythonCodeParser) extractInlineComments(bodyNode *sitter.Node, parentFunctionID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		// Python only has line comments (# ...)
		if node.Type() == "comment" {
			elem := p.createInlineCommentElement(node, parentFunctionID)
			if elem != nil {
				elements = append(elements, *elem)
			}
		}
	})

	return elements
}

func (p *PythonCodeParser) createInlineCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean up comment marker
	cleanContent := strings.TrimPrefix(content, "#")
	cleanContent = strings.TrimSpace(cleanContent)

	// Detect markers (TODO, FIXME, HACK, XXX, NOTE, BUG, OPTIMIZE)
	markers := detectCommentMarkers(cleanContent)

	return &Element{
		ElementID:       generateID("py_inline_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":      "inline_comment",
			"comment_type":  "line_comment",
			"language":      "python",
			"line_number":   lineNum,
			"markers":       markers,
		},
	}
}

func (p *PythonCodeParser) createFunctionCallElement(node *sitter.Node, parentID string) *Element {
	// Extract function name (first child)
	if node.ChildCount() == 0 {
		return nil
	}

	funcNode := node.Child(0)
	funcName := p.extractCallTarget(funcNode)

	if funcName == "" {
		return nil
	}

	// Skip empty function names
	if strings.TrimSpace(funcName) == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1

	callType := "direct"
	if strings.Contains(funcName, ".") {
		callType = "method_call"
	}

	content := funcName + "()"

	return &Element{
		ElementID:       generateID("py_call"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind": "function_call",
			"target_function": funcName,
			"call_type":       callType,
			"line_number":     lineNum,
			"language":        "python",
		},
	}
}

func (p *PythonCodeParser) extractCallTarget(node *sitter.Node) string {
	switch node.Type() {
	case "identifier":
		return p.getNodeText(node)

	case "attribute":
		// obj.method
		parts := []string{}
		p.walkNode(node, func(n *sitter.Node) {
			if n.Type() == "identifier" {
				parts = append(parts, p.getNodeText(n))
			}
		})
		return strings.Join(parts, ".")

	default:
		return ""
	}
}

func (p *PythonCodeParser) createTypeUsageElement(typeStr string, parentID string, context string) *Element {
	if typeStr == "" {
		return nil
	}

	// Skip basic built-in types
	basicTypes := map[string]bool{
		"int": true, "str": true, "float": true, "bool": true,
		"list": true, "dict": true, "tuple": true, "set": true,
		"None": true, "any": true, "Any": true,
	}

	cleanType := strings.TrimSpace(typeStr)
	if basicTypes[cleanType] {
		return nil
	}

	return &Element{
		ElementID:       generateID("py_type_usage"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         cleanType,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"dependency_kind": "type_usage",
			"usage_context":   context,
			"target_type":     cleanType,
			"language":        "python",
		},
	}
}

func (p *PythonCodeParser) createDocstringElement(node *sitter.Node, parentID string) *Element {
	if node.ChildCount() == 0 {
		return nil
	}

	stringNode := node.Child(0)
	if stringNode.Type() != "string" {
		return nil
	}

	content := p.getNodeText(stringNode)
	// Remove quotes
	content = strings.Trim(content, `"'`)
	lineNum := int(node.StartPoint().Row) + 1

	return &Element{
		ElementID:       generateID("py_doc"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    "docstring",
			"language":    "python",
			"line_number": lineNum,
		},
	}
}

func (p *PythonCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	content = strings.TrimPrefix(content, "#")
	content = strings.TrimSpace(content)
	lineNum := int(node.StartPoint().Row) + 1

	return &Element{
		ElementID:       generateID("py_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    "comment",
			"language":    "python",
			"line_number": lineNum,
		},
	}
}

// Helper functions

func (p *PythonCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *PythonCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *PythonCodeParser) walkNode(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNode(node.Child(i), fn)
	}
}

func (p *PythonCodeParser) extractDecorators(node *sitter.Node) []string {
	var decorators []string

	// Look for decorator nodes - they can be multiple siblings before the function
	current := node.PrevSibling()
	for current != nil {
		if current.Type() == "decorator" {
			decorators = append([]string{p.getNodeText(current)}, decorators...) // Prepend to maintain order
		} else if current.Type() != "comment" {
			// Stop if we hit a non-decorator, non-comment node
			break
		}
		current = current.PrevSibling()
	}

	return decorators
}

func (p *PythonCodeParser) extractClassName(parentID string) string {
	// Return the current class name from the stack
	if len(p.classStack) > 0 {
		return p.classStack[len(p.classStack)-1]
	}
	return ""
}

func (p *PythonCodeParser) isClassParent(parentID string) bool {
	// Check if parent ID starts with "py_class" prefix
	return strings.HasPrefix(parentID, "py_class")
}

func getModuleName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
