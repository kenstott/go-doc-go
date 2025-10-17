package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
)

// CCppCodeParser parses C and C++ source code files using tree-sitter
type CCppCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractIncludes   bool
	sourceCode        []byte
	moduleName        string
	language          string // "c" or "cpp"
	fileExtension     string
}

// NewCCppCodeParser creates a new C/C++ code parser with default settings
func NewCCppCodeParser() *CCppCodeParser {
	return &CCppCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractIncludes:   true,
	}
}

// GetName returns the parser name
func (p *CCppCodeParser) GetName() string {
	return "c_cpp_code"
}

// GetSupportedFormats returns supported file formats
func (p *CCppCodeParser) GetSupportedFormats() []string {
	return []string{".c", ".cpp", ".cc", ".cxx", ".c++", ".h", ".hpp", ".hxx", ".h++", "c", "cpp", "cc", "cxx", "h", "hpp", "hxx"}
}

// SupportsStreaming indicates if parser can handle streaming content
func (p *CCppCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *CCppCodeParser) Close() error {
	return nil
}

// Parse implements the Parser interface for C/C++ source code
func (p *CCppCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for C/C++ parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read C/C++ file: %w", err)
	}

	p.sourceCode = sourceCode
	p.moduleName = getModuleName(filePath)
	p.fileExtension = filepath.Ext(filePath)

	// Detect language based on file extension
	var grammarLang *sitter.Language
	switch p.fileExtension {
	case ".cpp", ".cc", ".cxx", ".c++", ".hpp", ".hxx", ".h++":
		grammarLang = cpp.GetLanguage()
		p.language = "cpp"
	case ".c", ".h":
		// .h could be either C or C++, default to C for .h
		grammarLang = c.GetLanguage()
		p.language = "c"
	default:
		// Default to C
		grammarLang = c.GetLanguage()
		p.language = "c"
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

func (p *CCppCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
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

func (p *CCppCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "preproc_include":
			if p.ExtractIncludes {
				elem := p.createIncludeElement(child, parentID)
				if elem != nil {
					elements = append(elements, *elem)
				}
			}

		case "function_definition":
			funcElements := p.createFunctionElements(child, parentID, "", false)
			elements = append(elements, funcElements...)

		case "declaration":
			// Handle function declarations, variable declarations
			declElements := p.handleDeclaration(child, parentID, "")
			elements = append(elements, declElements...)

		case "struct_specifier", "union_specifier", "enum_specifier":
			structElem := p.createStructElement(child, parentID)
			if structElem != nil {
				elements = append(elements, *structElem)
			}

		case "class_specifier": // C++ only
			classElements := p.createClassElements(child, parentID)
			elements = append(elements, classElements...)

		case "namespace_definition": // C++ only
			nsElements := p.extractNamespaceElements(child, parentID)
			elements = append(elements, nsElements...)

		case "template_declaration": // C++ templates
			// Extract the actual declaration from inside the template
			templateElements := p.extractTemplateElements(child, parentID)
			elements = append(elements, templateElements...)

		case "comment":
			if p.ExtractComments {
				commentElem := p.createCommentElement(child, parentID)
				if commentElem != nil {
					elements = append(elements, *commentElem)
				}
			}
		}
	}

	return elements
}

// createIncludeElement creates an include directive element
func (p *CCppCodeParser) createIncludeElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract included file
	var includedFile string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "string_literal" || child.Type() == "system_lib_string" {
			includedFile = strings.Trim(p.getNodeText(child), `"<>`)
			break
		}
	}

	namespace := p.moduleName

	return &Element{
		ElementID:       generateID(p.language + "_include"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind":   "include",
			"target_file":       includedFile,
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createFunctionElements creates function element(s)
func (p *CCppCodeParser) createFunctionElements(node *sitter.Node, parentID string, className string, isMethod bool) []Element {
	var elements []Element

	// Extract function name
	var funcName string
	var declarator *sitter.Node

	// Find the function_declarator
	declarator = p.findChildByType(node, "function_declarator")
	if declarator == nil {
		declarator = p.findChildByType(node, "pointer_declarator")
	}

	if declarator != nil {
		// Extract function name from declarator
		nameNode := p.findIdentifierInDeclarator(declarator)
		if nameNode != nil {
			funcName = p.getNodeText(nameNode)
		}
	}

	if funcName == "" {
		return elements
	}

	lineNum := int(node.StartPoint().Row) + 1

	// Build signature
	signature := p.buildFunctionSignature(node, funcName)

	// Determine visibility (C++ only)
	visibility := "public"
	if className != "" && strings.HasPrefix(funcName, "_") {
		visibility = "private"
	}

	// Determine function kind
	functionKind := "function"
	if isMethod {
		functionKind = "method"
	}

	namespace := p.moduleName
	if className != "" {
		namespace = fmt.Sprintf("%s::%s", p.moduleName, className)
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
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	if className != "" {
		funcElement.ClassName = &className
	}

	elements = append(elements, funcElement)

	// Extract function body for function calls and inline comments
	bodyNode := p.findChildByType(node, "compound_statement")
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

// handleDeclaration processes declaration nodes
func (p *CCppCodeParser) handleDeclaration(node *sitter.Node, parentID string, className string) []Element {
	var elements []Element

	// Check if this is a function declaration
	funcDeclarator := p.findChildByType(node, "function_declarator")
	if funcDeclarator != nil {
		// This is a function declaration (prototype)
		nameNode := p.findIdentifierInDeclarator(funcDeclarator)
		if nameNode != nil {
			funcName := p.getNodeText(nameNode)
			lineNum := int(node.StartPoint().Row) + 1
			signature := p.getNodeText(node)
			namespace := p.moduleName

			funcElement := Element{
				ElementID:       generateID(p.language + "_func_decl"),
				ElementType:     "code_function",
				ElementCategory: GetElementCategory("code_function"),
				Content:         signature,
				ContentPreview:  truncate(signature, p.MaxContentPreview),
				ParentID:        parentID,
				FunctionName:    &funcName,
				Namespace:       &namespace,
				LineNumber:      &lineNum,
				Signature:       &signature,
				Metadata: map[string]interface{}{
					"code_element_kind": "function_declaration",
					"visibility":        "public",
					"language":          p.language,
					"line_number":       lineNum,
					"file_extension":    p.fileExtension,
				},
			}
			elements = append(elements, funcElement)
		}
	}

	return elements
}

// createStructElement creates a struct/union/enum element
func (p *CCppCodeParser) createStructElement(node *sitter.Node, parentID string) *Element {
	var structName string
	var structKind string

	switch node.Type() {
	case "struct_specifier":
		structKind = "struct"
	case "union_specifier":
		structKind = "union"
	case "enum_specifier":
		structKind = "enum"
	}

	// Find the name
	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode != nil {
		structName = p.getNodeText(nameNode)
	}

	if structName == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName
	signature := structKind + " " + structName

	return &Element{
		ElementID:       generateID(p.language + "_struct"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &structName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": structKind,
			"visibility":        "public",
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createClassElements creates class element(s) (C++ only)
func (p *CCppCodeParser) createClassElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract class name
	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode == nil {
		return elements
	}

	className := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	signature := "class " + className
	namespace := p.moduleName
	content := p.getNodeText(node)

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
			"visibility":        "public",
			"language":          p.language,
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	elements = append(elements, classElement)

	// Extract class body (methods, fields)
	bodyNode := p.findChildByType(node, "field_declaration_list")
	if bodyNode != nil {
		bodyElements := p.extractClassBody(bodyNode, classElement.ElementID, className)
		elements = append(elements, bodyElements...)
	}

	return elements
}

func (p *CCppCodeParser) extractClassBody(bodyNode *sitter.Node, parentID, className string) []Element {
	var elements []Element

	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)

		switch child.Type() {
		case "function_definition":
			methodElements := p.createFunctionElements(child, parentID, className, true)
			elements = append(elements, methodElements...)

		case "field_declaration":
			fieldElem := p.createFieldElement(child, parentID, className)
			if fieldElem != nil {
				elements = append(elements, *fieldElem)
			}
		}
	}

	return elements
}

func (p *CCppCodeParser) createFieldElement(node *sitter.Node, parentID, className string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1
	namespace := fmt.Sprintf("%s::%s", p.moduleName, className)

	// Try to extract field name
	var fieldName string
	declarator := p.findChildByType(node, "field_identifier")
	if declarator != nil {
		fieldName = p.getNodeText(declarator)
	}

	visibility := "public"
	if strings.HasPrefix(fieldName, "m_") || strings.HasPrefix(fieldName, "_") {
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

// extractTemplateElements extracts the class or function inside a template declaration
func (p *CCppCodeParser) extractTemplateElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Look for class_specifier or function_definition inside template
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "class_specifier":
			classElements := p.createClassElements(child, parentID)
			elements = append(elements, classElements...)
		case "function_definition":
			funcElements := p.createFunctionElements(child, parentID, "", false)
			elements = append(elements, funcElements...)
		}
	}

	return elements
}

func (p *CCppCodeParser) extractNamespaceElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract namespace name - use namespace_identifier for C++
	nameNode := p.findChildByType(node, "namespace_identifier")
	if nameNode == nil {
		nameNode = p.findChildByType(node, "identifier")
	}

	var namespaceName string
	if nameNode != nil {
		namespaceName = p.getNodeText(nameNode)
	}

	// Create namespace element
	if namespaceName != "" {
		lineNum := int(node.StartPoint().Row) + 1
		signature := "namespace " + namespaceName
		namespace := p.moduleName

		nsElement := Element{
			ElementID:       generateID(p.language + "_namespace"),
			ElementType:     "code_grouping",
			ElementCategory: GetElementCategory("code_grouping"),
			Content:         p.getNodeText(node),
			ContentPreview:  truncate(signature, p.MaxContentPreview),
			ParentID:        parentID,
			Namespace:       &namespace,
			LineNumber:      &lineNum,
			Signature:       &signature,
			Metadata: map[string]interface{}{
				"code_element_kind": "namespace",
				"namespace_name":    namespaceName,
				"language":          p.language,
				"line_number":       lineNum,
				"file_extension":    p.fileExtension,
			},
		}
		elements = append(elements, nsElement)

		// Extract elements inside the namespace body
		bodyNode := p.findChildByType(node, "declaration_list")
		if bodyNode != nil {
			bodyElements := p.extractElements(bodyNode, parentID)
			elements = append(elements, bodyElements...)
		}
	}

	return elements
}

// Inline comments extraction

func (p *CCppCodeParser) extractInlineComments(bodyNode *sitter.Node, parentFunctionID string) []Element {
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

func (p *CCppCodeParser) createInlineCommentElement(node *sitter.Node, parentID string) *Element {
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

func (p *CCppCodeParser) extractFunctionCalls(bodyNode *sitter.Node, parentFuncID string) []Element {
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

func (p *CCppCodeParser) createFunctionCallElement(node *sitter.Node, parentID string) *Element {
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
	if strings.Contains(funcName, ".") || strings.Contains(funcName, "->") || strings.Contains(funcName, "::") {
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

func (p *CCppCodeParser) extractCallTarget(node *sitter.Node) string {
	switch node.Type() {
	case "identifier":
		return p.getNodeText(node)

	case "field_expression":
		// obj.method or obj->method
		parts := []string{}
		p.walkNode(node, func(n *sitter.Node) {
			if n.Type() == "identifier" || n.Type() == "field_identifier" {
				parts = append(parts, p.getNodeText(n))
			}
		})
		return strings.Join(parts, ".")

	case "qualified_identifier": // C++ namespace::function
		return p.getNodeText(node)

	default:
		return ""
	}
}

func (p *CCppCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
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

func (p *CCppCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *CCppCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *CCppCodeParser) findIdentifierInDeclarator(node *sitter.Node) *sitter.Node {
	// Recursively search for identifier or field_identifier in declarator
	if node.Type() == "identifier" || node.Type() == "field_identifier" {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "field_identifier" {
			return child
		}
		if child.Type() == "function_declarator" || child.Type() == "pointer_declarator" {
			result := p.findIdentifierInDeclarator(child)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func (p *CCppCodeParser) walkNode(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNode(node.Child(i), fn)
	}
}

func (p *CCppCodeParser) buildFunctionSignature(node *sitter.Node, funcName string) string {
	// Get the declaration part (return type + name + parameters)
	declaratorNode := p.findChildByType(node, "function_declarator")
	if declaratorNode != nil {
		// Get parameters
		params := ""
		paramListNode := p.findChildByType(declaratorNode, "parameter_list")
		if paramListNode != nil {
			params = p.getNodeText(paramListNode)
		}

		// Try to get return type
		returnType := ""
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "primitive_type" || child.Type() == "type_identifier" {
				returnType = p.getNodeText(child) + " "
				break
			}
		}

		return returnType + funcName + params
	}

	return funcName + "()"
}
