package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// JavaScriptTypeScriptCodeParser parses JavaScript and TypeScript source code files using tree-sitter
type JavaScriptTypeScriptCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	sourceCode        []byte
	moduleName        string
	language          string // "javascript" or "typescript"
	fileExtension     string
}

// NewJavaScriptTypeScriptCodeParser creates a new JavaScript/TypeScript code parser with default settings
func NewJavaScriptTypeScriptCodeParser() *JavaScriptTypeScriptCodeParser {
	return &JavaScriptTypeScriptCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *JavaScriptTypeScriptCodeParser) GetName() string {
	return "javascript_typescript_code"
}

// GetSupportedFormats returns supported file formats
func (p *JavaScriptTypeScriptCodeParser) GetSupportedFormats() []string {
	return []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", "js", "jsx", "ts", "tsx", "mjs", "cjs"}
}

// SupportsStreaming indicates if parser can handle streaming content
func (p *JavaScriptTypeScriptCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *JavaScriptTypeScriptCodeParser) Close() error {
	return nil
}

// Parse implements the Parser interface for JavaScript/TypeScript source code
func (p *JavaScriptTypeScriptCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for JS/TS parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JS/TS file: %w", err)
	}

	p.sourceCode = sourceCode
	p.moduleName = getModuleName(filePath)
	p.fileExtension = filepath.Ext(filePath)

	// Detect language based on file extension
	var grammarLang *sitter.Language
	switch p.fileExtension {
	case ".ts", ".tsx":
		grammarLang = typescript.GetLanguage()
		p.language = "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		grammarLang = javascript.GetLanguage()
		p.language = "javascript"
	default:
		// Default to JavaScript
		grammarLang = javascript.GetLanguage()
		p.language = "javascript"
	}

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(grammarLang)

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s source: %w", p.language, err)
	}
	defer tree.Close()

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: p.language + "_source",
		Title:   p.moduleName,
		Metadata: map[string]interface{}{
			"module":         p.moduleName,
			"path":           filePath,
			"language":       p.language,
			"file_extension": p.fileExtension,
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

func (p *JavaScriptTypeScriptCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
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
			"doc_type":       p.language + "_source",
			"module":         p.moduleName,
			"path":           filePath,
			"language":       p.language,
			"file_extension": p.fileExtension,
		},
	}
}

func (p *JavaScriptTypeScriptCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "import_statement":
			elem := p.createImportElement(child, parentID)
			if elem != nil {
				elements = append(elements, *elem)
			}

		case "export_statement":
			exportElems := p.createExportElements(child, parentID)
			elements = append(elements, exportElems...)

		case "function_declaration", "arrow_function", "function":
			funcElements := p.createFunctionElements(child, parentID, "")
			elements = append(elements, funcElements...)

		case "class_declaration":
			classElements := p.createClassElements(child, parentID)
			elements = append(elements, classElements...)

		case "lexical_declaration", "variable_declaration":
			// Module-level variables/constants
			varElements := p.createVariableElements(child, parentID)
			elements = append(elements, varElements...)

		case "interface_declaration", "type_alias_declaration":
			// TypeScript only
			typeElem := p.createTypeElement(child, parentID)
			if typeElem != nil {
				elements = append(elements, *typeElem)
			}

		case "enum_declaration":
			// TypeScript only
			enumElem := p.createEnumElement(child, parentID)
			if enumElem != nil {
				elements = append(elements, *enumElem)
			}

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, parentID)
				if commentElem != nil {
					elements = append(elements, *commentElem)
				}
			}
		}

		// Don't recursively process children - we handle them explicitly
	}

	return elements
}

// createImportElement creates an import element
func (p *JavaScriptTypeScriptCodeParser) createImportElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract import source
	sourceNode := p.findChildByType(node, "string")
	var targetNamespace string
	if sourceNode != nil {
		targetNamespace = strings.Trim(p.getNodeText(sourceNode), `"'`)
	}

	namespace := p.moduleName

	return &Element{
		ElementID:       generateID(p.language + "_import"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind":   "import",
			"target_namespace":  targetNamespace,
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createExportElements creates export element(s)
func (p *JavaScriptTypeScriptCodeParser) createExportElements(node *sitter.Node, parentID string) []Element {
	var elements []Element
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := p.moduleName

	// Extract exported names
	var exportedNames []string
	p.walkNode(node, func(n *sitter.Node) {
		if n.Type() == "identifier" || n.Type() == "property_identifier" {
			name := p.getNodeText(n)
			if name != "" && name != "export" && name != "default" {
				exportedNames = append(exportedNames, name)
			}
		}
	})

	elem := Element{
		ElementID:       generateID(p.language + "_export"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind":   "export",
			"exported_names":    exportedNames,
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	elements = append(elements, elem)
	return elements
}

// createFunctionElements creates function element(s)
func (p *JavaScriptTypeScriptCodeParser) createFunctionElements(node *sitter.Node, parentID string, className string) []Element {
	var elements []Element

	// Extract function name
	var funcName string
	var functionKind string

	switch node.Type() {
	case "function_declaration":
		nameNode := p.findChildByType(node, "identifier")
		if nameNode != nil {
			funcName = p.getNodeText(nameNode)
		}
		functionKind = "function"

	case "arrow_function":
		funcName = "<anonymous>"
		functionKind = "arrow_function"

	case "method_definition":
		nameNode := p.findChildByType(node, "property_identifier")
		if nameNode != nil {
			funcName = p.getNodeText(nameNode)
		}
		// Check if this is a constructor
		if funcName == "constructor" {
			functionKind = "constructor"
		} else {
			functionKind = "method"
		}
	}

	if funcName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1

	// Build signature
	signature := p.buildFunctionSignature(node, funcName, functionKind)

	// Determine visibility
	visibility := "public"
	if strings.HasPrefix(funcName, "_") || strings.HasPrefix(funcName, "#") {
		visibility = "private"
	}

	// Check for async
	isAsync := p.hasChildWithType(node, "async")

	namespace := p.moduleName
	if className != "" {
		namespace = fmt.Sprintf("%s.%s", p.moduleName, className)
	}

	content := p.getNodeText(node)

	funcElement := Element{
		ElementID:       generateID(p.language + "_func"),
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
			"code_element_kind": functionKind,
			"visibility":        visibility,
			"language":          p.language,
			"is_async":          isAsync,
			"is_arrow_function": functionKind == "arrow_function",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	if className != "" {
		funcElement.ClassName = &className
	}

	elements = append(elements, funcElement)

	// Extract function body for function calls and inline comments
	bodyNode := p.findChildByType(node, "statement_block")
	if bodyNode != nil {
		callElements := p.extractFunctionCalls(bodyNode, funcElement.ElementID)
		elements = append(elements, callElements...)

		// Extract inline comments within function body
		if p.ExtractComments {
			inlineComments := p.extractInlineComments(bodyNode, funcElement.ElementID)
			elements = append(elements, inlineComments...)
		}
	}

	return elements
}

// createClassElements creates class element(s)
func (p *JavaScriptTypeScriptCodeParser) createClassElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract class name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		nameNode = p.findChildByType(node, "type_identifier")
	}
	if nameNode == nil {
		return elements
	}

	className := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	signature := "class " + className
	namespace := p.moduleName
	content := p.getNodeText(node)

	// Determine visibility
	visibility := "public"
	if strings.HasPrefix(className, "_") {
		visibility = "private"
	}

	classElement := Element{
		ElementID:       generateID(p.language + "_class"),
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
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	elements = append(elements, classElement)

	// Extract class body (methods, fields)
	bodyNode := p.findChildByType(node, "class_body")
	if bodyNode != nil {
		bodyElements := p.extractClassBody(bodyNode, classElement.ElementID, className)
		elements = append(elements, bodyElements...)
	}

	return elements
}

func (p *JavaScriptTypeScriptCodeParser) extractClassBody(bodyNode *sitter.Node, parentID, className string) []Element {
	var elements []Element

	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)

		switch child.Type() {
		case "method_definition":
			methodElements := p.createFunctionElements(child, parentID, className)
			elements = append(elements, methodElements...)

		case "field_definition", "public_field_definition":
			fieldElem := p.createClassFieldElement(child, parentID, className)
			if fieldElem != nil {
				elements = append(elements, *fieldElem)
			}
		}
	}

	return elements
}

func (p *JavaScriptTypeScriptCodeParser) createClassFieldElement(node *sitter.Node, parentID, className string) *Element {
	nameNode := p.findChildByType(node, "property_identifier")
	if nameNode == nil {
		return nil
	}

	fieldName := p.getNodeText(nameNode)
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := fmt.Sprintf("%s.%s", p.moduleName, className)

	visibility := "public"
	if strings.HasPrefix(fieldName, "_") || strings.HasPrefix(fieldName, "#") {
		visibility = "private"
	}

	return &Element{
		ElementID:       generateID(p.language + "_field"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		ClassName:       &className,
		Metadata: map[string]interface{}{
			"data_kind":      "field",
			"data_name":      fieldName,
			"scope":          "class",
			"mutability":     "mutable",
			"visibility":     visibility,
			"language":       p.language,
			"line_number":    lineNum,
			"file_extension": p.fileExtension,
		},
	}
}

func (p *JavaScriptTypeScriptCodeParser) createVariableElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Find variable_declarator nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declarator" {
			varElem := p.createVariableElement(child, parentID)
			if varElem != nil {
				elements = append(elements, *varElem)
			}
		}
	}

	return elements
}

func (p *JavaScriptTypeScriptCodeParser) createVariableElement(node *sitter.Node, parentID string) *Element {
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	varName := p.getNodeText(nameNode)

	// Check if this is an arrow function assigned to a variable
	arrowFuncNode := p.findChildByType(node, "arrow_function")
	if arrowFuncNode != nil {
		// This is actually a function definition, not a variable
		// Extract as a function element instead
		return p.createArrowFunctionFromVariable(node, parentID, varName)
	}

	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := p.moduleName

	visibility := "public"
	if strings.HasPrefix(varName, "_") {
		visibility = "private"
	}

	// Determine if this is a constant
	parent := node.Parent()
	dataKind := "variable"
	mutability := "mutable"
	if parent != nil && parent.Type() == "lexical_declaration" {
		// Check if it's const or let
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			text := p.getNodeText(child)
			if text == "const" {
				dataKind = "constant"
				mutability = "immutable"
				break
			}
		}
	}

	return &Element{
		ElementID:       generateID(p.language + "_var"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":      dataKind,
			"data_name":      varName,
			"scope":          "module",
			"mutability":     mutability,
			"visibility":     visibility,
			"language":       p.language,
			"line_number":    lineNum,
			"file_extension": p.fileExtension,
		},
	}
}

// createArrowFunctionFromVariable creates a function element from an arrow function in a variable declaration
func (p *JavaScriptTypeScriptCodeParser) createArrowFunctionFromVariable(node *sitter.Node, parentID string, funcName string) *Element {
	arrowFuncNode := p.findChildByType(node, "arrow_function")
	if arrowFuncNode == nil {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName

	// Build signature
	paramsNode := p.findChildByType(arrowFuncNode, "formal_parameters")
	params := "()"
	if paramsNode != nil {
		params = p.getNodeText(paramsNode)
	}

	signature := params + " => {...}"

	// Check for async - look at the whole variable declarator
	isAsync := false
	// Check if arrow function itself has "async" keyword
	arrowText := p.getNodeText(node)
	if strings.Contains(arrowText, "async") {
		// Verify it's actually before the arrow function
		parts := strings.Split(arrowText, "=>")
		if len(parts) > 0 && strings.Contains(parts[0], "async") {
			isAsync = true
		}
	}

	if isAsync {
		signature = "async " + signature
	}

	visibility := "public"
	if strings.HasPrefix(funcName, "_") {
		visibility = "private"
	}

	return &Element{
		ElementID:       generateID(p.language + "_func"),
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
			"code_element_kind": "arrow_function",
			"visibility":        visibility,
			"language":          p.language,
			"is_async":          isAsync,
			"is_arrow_function": true,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// TypeScript-specific elements

func (p *JavaScriptTypeScriptCodeParser) createTypeElement(node *sitter.Node, parentID string) *Element {
	var typeName string
	var codeElementKind string

	switch node.Type() {
	case "interface_declaration":
		nameNode := p.findChildByType(node, "type_identifier")
		if nameNode != nil {
			typeName = p.getNodeText(nameNode)
		}
		codeElementKind = "interface"
	case "type_alias_declaration":
		nameNode := p.findChildByType(node, "type_identifier")
		if nameNode != nil {
			typeName = p.getNodeText(nameNode)
		}
		codeElementKind = "type_alias"
	}

	if typeName == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName

	// Check if the type is generic (contains <T> or similar)
	isGeneric := strings.Contains(content, "<") && strings.Contains(content, ">")

	return &Element{
		ElementID:       generateID("ts_type"),
		ElementType:     "code_type",
		ElementCategory: GetElementCategory("code_type"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &typeName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": codeElementKind,
			"type_name":         typeName,
			"is_generic":        isGeneric,
			"language":          "typescript",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

func (p *JavaScriptTypeScriptCodeParser) createEnumElement(node *sitter.Node, parentID string) *Element {
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	enumName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName

	return &Element{
		ElementID:       generateID("ts_enum"),
		ElementType:     "code_type",
		ElementCategory: GetElementCategory("code_type"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &enumName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"code_element_kind": "enum",
			"type_name":         enumName,
			"language":          "typescript",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// Inline comments extraction

func (p *JavaScriptTypeScriptCodeParser) extractInlineComments(bodyNode *sitter.Node, parentFunctionID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "comment" {
			elem := p.createInlineCommentElement(node, parentFunctionID)
			if elem != nil {
				elements = append(elements, *elem)
			}
		}
	})

	return elements
}

func (p *JavaScriptTypeScriptCodeParser) createInlineCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Determine comment type
	commentType := "line_comment"
	if strings.HasPrefix(content, "/*") {
		commentType = "block_comment"
	}

	// Clean up comment markers
	cleanContent := content
	if commentType == "line_comment" {
		cleanContent = strings.TrimPrefix(cleanContent, "//")
	} else {
		cleanContent = strings.TrimPrefix(cleanContent, "/*")
		cleanContent = strings.TrimSuffix(cleanContent, "*/")
	}
	cleanContent = strings.TrimSpace(cleanContent)

	// Detect markers (TODO, FIXME, HACK, XXX, NOTE, BUG, OPTIMIZE, TEMP)
	markers := detectCommentMarkers(cleanContent)

	return &Element{
		ElementID:       generateID(p.language + "_inline_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":       "inline_comment",
			"comment_type":   commentType,
			"language":       p.language,
			"line_number":    lineNum,
			"markers":        markers,
			"file_extension": p.fileExtension,
		},
	}
}

// Function calls extraction

func (p *JavaScriptTypeScriptCodeParser) extractFunctionCalls(bodyNode *sitter.Node, parentFuncID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			callElem := p.createFunctionCallElement(node, parentFuncID)
			if callElem != nil {
				elements = append(elements, *callElem)
			}
		}
	})

	return elements
}

func (p *JavaScriptTypeScriptCodeParser) createFunctionCallElement(node *sitter.Node, parentID string) *Element {
	if node.ChildCount() == 0 {
		return nil
	}

	funcNode := node.Child(0)
	funcName := p.extractCallTarget(funcNode)

	if funcName == "" || strings.TrimSpace(funcName) == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1

	callType := "direct"
	if strings.Contains(funcName, ".") {
		callType = "method_call"
	}

	content := funcName + "()"

	return &Element{
		ElementID:       generateID(p.language + "_call"),
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
			"language":        p.language,
			"file_extension":  p.fileExtension,
		},
	}
}

func (p *JavaScriptTypeScriptCodeParser) extractCallTarget(node *sitter.Node) string {
	switch node.Type() {
	case "identifier":
		return p.getNodeText(node)

	case "member_expression":
		// obj.method
		parts := []string{}
		p.walkNode(node, func(n *sitter.Node) {
			if n.Type() == "identifier" || n.Type() == "property_identifier" {
				parts = append(parts, p.getNodeText(n))
			}
		})
		return strings.Join(parts, ".")

	default:
		return ""
	}
}

func (p *JavaScriptTypeScriptCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean comment markers
	cleanContent := content
	if strings.HasPrefix(content, "//") {
		cleanContent = strings.TrimPrefix(content, "//")
	} else if strings.HasPrefix(content, "/*") {
		cleanContent = strings.TrimPrefix(content, "/*")
		cleanContent = strings.TrimSuffix(cleanContent, "*/")
	}
	cleanContent = strings.TrimSpace(cleanContent)

	return &Element{
		ElementID:       generateID(p.language + "_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":       "comment",
			"language":       p.language,
			"line_number":    lineNum,
			"file_extension": p.fileExtension,
		},
	}
}

// Helper functions

func (p *JavaScriptTypeScriptCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *JavaScriptTypeScriptCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *JavaScriptTypeScriptCodeParser) hasChildWithType(node *sitter.Node, nodeType string) bool {
	return p.findChildByType(node, nodeType) != nil
}

func (p *JavaScriptTypeScriptCodeParser) walkNode(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNode(node.Child(i), fn)
	}
}

func (p *JavaScriptTypeScriptCodeParser) buildFunctionSignature(node *sitter.Node, funcName string, functionKind string) string {
	// Build basic signature
	paramsNode := p.findChildByType(node, "formal_parameters")
	params := "()"
	if paramsNode != nil {
		params = p.getNodeText(paramsNode)
	}

	signature := funcName + params

	// Add function keyword for regular functions
	if functionKind == "function" {
		signature = "function " + signature
	} else if functionKind == "arrow_function" {
		signature = params + " => {...}"
	}

	// Check for async
	if p.hasChildWithType(node, "async") {
		signature = "async " + signature
	}

	return signature
}
