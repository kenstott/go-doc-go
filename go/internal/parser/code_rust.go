package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

// RustCodeParser parses Rust source code files using tree-sitter
type RustCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	sourceCode        []byte
	moduleName        string
	fileExtension     string
}

// NewRustCodeParser creates a new Rust code parser with default settings
func NewRustCodeParser() *RustCodeParser {
	return &RustCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *RustCodeParser) GetName() string {
	return "rust_code"
}

// GetSupportedFormats returns supported file formats
func (p *RustCodeParser) GetSupportedFormats() []string {
	return []string{".rs", "rs"}
}

// SupportsStreaming indicates if parser can handle streaming content
func (p *RustCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *RustCodeParser) Close() error {
	return nil
}

// Parse implements the Parser interface for Rust source code
func (p *RustCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Rust parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Rust file: %w", err)
	}

	p.sourceCode = sourceCode
	p.moduleName = getModuleName(filePath)
	p.fileExtension = filepath.Ext(filePath)

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(rust.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Rust source: %w", err)
	}
	defer tree.Close()

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "rust_source",
		Title:   p.moduleName,
		Metadata: map[string]interface{}{
			"module":         p.moduleName,
			"path":           filePath,
			"language":       "rust",
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

func (p *RustCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
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
			"doc_type":       "rust_source",
			"module":         p.moduleName,
			"path":           filePath,
			"language":       "rust",
			"file_extension": p.fileExtension,
		},
	}
}

func (p *RustCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "use_declaration":
			if p.ExtractImports {
				elem := p.createImportElement(child, parentID)
				if elem != nil {
					elements = append(elements, *elem)
				}
			}

		case "function_item":
			funcElements := p.createFunctionElements(child, parentID, "", false)
			elements = append(elements, funcElements...)

		case "struct_item":
			structElem := p.createStructElement(child, parentID)
			if structElem != nil {
				elements = append(elements, *structElem)
			}

		case "enum_item":
			enumElem := p.createEnumElement(child, parentID)
			if enumElem != nil {
				elements = append(elements, *enumElem)
			}

		case "trait_item":
			traitElements := p.createTraitElements(child, parentID)
			elements = append(elements, traitElements...)

		case "impl_item":
			implElements := p.extractImplElements(child, parentID)
			elements = append(elements, implElements...)

		case "mod_item":
			modElements := p.extractModuleElements(child, parentID)
			elements = append(elements, modElements...)

		case "line_comment", "block_comment":
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

// createImportElement creates an import/use element
func (p *RustCodeParser) createImportElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract the target path
	var targetPath string
	scopedIdNode := p.findChildByType(node, "scoped_identifier")
	if scopedIdNode != nil {
		targetPath = p.getNodeText(scopedIdNode)
	} else {
		idNode := p.findChildByType(node, "identifier")
		if idNode != nil {
			targetPath = p.getNodeText(idNode)
		}
	}

	namespace := p.moduleName

	return &Element{
		ElementID:       generateID("rust_import"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind":   "import",
			"target_namespace":  targetPath,
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createFunctionElements creates function element(s)
func (p *RustCodeParser) createFunctionElements(node *sitter.Node, parentID string, structName string, isMethod bool) []Element {
	var elements []Element

	// Extract function name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	funcName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Build signature
	signature := p.buildFunctionSignature(node, funcName)

	// Determine visibility
	visibility := "private"
	visNode := p.findChildByType(node, "visibility_modifier")
	if visNode != nil {
		visibility = "public"
	}

	// Determine function kind
	functionKind := "function"
	if isMethod {
		functionKind = "method"
	}

	namespace := p.moduleName
	if structName != "" {
		namespace = fmt.Sprintf("%s::%s", p.moduleName, structName)
	}

	content := p.getNodeText(node)

	funcElement := Element{
		ElementID:       generateID("rust_func"),
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
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	if structName != "" {
		funcElement.ClassName = &structName
	}

	elements = append(elements, funcElement)

	// Extract function body for function calls and inline comments
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

	return elements
}

// createStructElement creates a struct element
func (p *RustCodeParser) createStructElement(node *sitter.Node, parentID string) *Element {
	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode == nil {
		return nil
	}

	structName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName
	signature := "struct " + structName

	// Determine visibility
	visibility := "private"
	visNode := p.findChildByType(node, "visibility_modifier")
	if visNode != nil {
		visibility = "public"
	}

	return &Element{
		ElementID:       generateID("rust_struct"),
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
			"code_element_kind": "struct",
			"visibility":        visibility,
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createEnumElement creates an enum element
func (p *RustCodeParser) createEnumElement(node *sitter.Node, parentID string) *Element {
	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode == nil {
		return nil
	}

	enumName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)
	namespace := p.moduleName
	signature := "enum " + enumName

	// Determine visibility
	visibility := "private"
	visNode := p.findChildByType(node, "visibility_modifier")
	if visNode != nil {
		visibility = "public"
	}

	return &Element{
		ElementID:       generateID("rust_enum"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &enumName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "enum",
			"visibility":        visibility,
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}
}

// createTraitElements creates trait element(s)
func (p *RustCodeParser) createTraitElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode == nil {
		return elements
	}

	traitName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	signature := "trait " + traitName
	namespace := p.moduleName
	content := p.getNodeText(node)

	// Determine visibility
	visibility := "private"
	visNode := p.findChildByType(node, "visibility_modifier")
	if visNode != nil {
		visibility = "public"
	}

	traitElement := Element{
		ElementID:       generateID("rust_trait"),
		ElementType:     "code_type",
		ElementCategory: GetElementCategory("code_type"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &traitName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "trait",
			"visibility":        visibility,
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	elements = append(elements, traitElement)

	// Extract trait methods
	declListNode := p.findChildByType(node, "declaration_list")
	if declListNode != nil {
		for i := 0; i < int(declListNode.ChildCount()); i++ {
			child := declListNode.Child(i)
			if child.Type() == "function_item" || child.Type() == "function_signature_item" {
				methodElements := p.createFunctionElements(child, traitElement.ElementID, traitName, true)
				elements = append(elements, methodElements...)
			}
		}
	}

	return elements
}

// extractImplElements extracts impl block elements
func (p *RustCodeParser) extractImplElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Get the type being implemented
	typeNode := p.findChildByType(node, "type_identifier")
	if typeNode == nil {
		return elements
	}

	typeName := p.getNodeText(typeNode)

	// Extract methods from impl block
	declListNode := p.findChildByType(node, "declaration_list")
	if declListNode != nil {
		for i := 0; i < int(declListNode.ChildCount()); i++ {
			child := declListNode.Child(i)
			if child.Type() == "function_item" {
				methodElements := p.createFunctionElements(child, parentID, typeName, true)
				elements = append(elements, methodElements...)
			}
		}
	}

	return elements
}

// extractModuleElements extracts module elements
func (p *RustCodeParser) extractModuleElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	modName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1
	signature := "mod " + modName
	namespace := p.moduleName

	// Determine visibility
	visibility := "private"
	visNode := p.findChildByType(node, "visibility_modifier")
	if visNode != nil {
		visibility = "public"
	}

	modElement := Element{
		ElementID:       generateID("rust_module"),
		ElementType:     "code_grouping",
		ElementCategory: GetElementCategory("code_grouping"),
		Content:         p.getNodeText(node),
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "module",
			"module_name":       modName,
			"visibility":        visibility,
			"language":          "rust",
			"line_number":       lineNum,
			"file_extension":    p.fileExtension,
		},
	}

	elements = append(elements, modElement)

	// Extract elements inside the module
	declListNode := p.findChildByType(node, "declaration_list")
	if declListNode != nil {
		bodyElements := p.extractElements(declListNode, parentID)
		elements = append(elements, bodyElements...)
	}

	return elements
}

// Inline comments extraction
func (p *RustCodeParser) extractInlineComments(bodyNode *sitter.Node, parentFunctionID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "line_comment" || node.Type() == "block_comment" {
			elem := p.createInlineCommentElement(node, parentFunctionID)
			if elem != nil {
				elements = append(elements, *elem)
			}
		}
	})

	return elements
}

func (p *RustCodeParser) createInlineCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Determine comment type
	commentType := "line_comment"
	if node.Type() == "block_comment" {
		commentType = "block_comment"
	}

	// Clean up comment markers
	cleanContent := content
	if commentType == "line_comment" {
		cleanContent = strings.TrimPrefix(cleanContent, "//")
		cleanContent = strings.TrimPrefix(cleanContent, "/") // For doc comments
	} else {
		cleanContent = strings.TrimPrefix(cleanContent, "/*")
		cleanContent = strings.TrimSuffix(cleanContent, "*/")
	}
	cleanContent = strings.TrimSpace(cleanContent)

	// Detect markers (TODO, FIXME, HACK, XXX, NOTE, BUG, OPTIMIZE, TEMP)
	markers := detectCommentMarkers(cleanContent)

	return &Element{
		ElementID:       generateID("rust_inline_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":       "inline_comment",
			"comment_type":   commentType,
			"language":       "rust",
			"line_number":    lineNum,
			"markers":        markers,
			"file_extension": p.fileExtension,
		},
	}
}

// Function calls extraction
func (p *RustCodeParser) extractFunctionCalls(bodyNode *sitter.Node, parentFuncID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "call_expression" || node.Type() == "macro_invocation" {
			callElem := p.createFunctionCallElement(node, parentFuncID)
			if callElem != nil {
				elements = append(elements, *callElem)
			}
		}
	})

	return elements
}

func (p *RustCodeParser) createFunctionCallElement(node *sitter.Node, parentID string) *Element {
	if node.ChildCount() == 0 {
		return nil
	}

	var funcName string
	var isMacro bool

	if node.Type() == "macro_invocation" {
		// For macros like println!
		macroNode := node.Child(0)
		if macroNode != nil {
			funcName = p.getNodeText(macroNode)
			isMacro = true
		}
	} else {
		// For regular function calls
		funcNode := node.Child(0)
		funcName = p.extractCallTarget(funcNode)
	}

	if funcName == "" || strings.TrimSpace(funcName) == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1

	callType := "direct"
	if strings.Contains(funcName, "::") {
		callType = "qualified"
	} else if strings.Contains(funcName, ".") {
		callType = "method_call"
	}

	if isMacro {
		callType = "macro"
	}

	content := funcName
	if isMacro {
		content += "!"
	} else {
		content += "()"
	}

	return &Element{
		ElementID:       generateID("rust_call"),
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
			"is_macro":        isMacro,
			"line_number":     lineNum,
			"language":        "rust",
			"file_extension":  p.fileExtension,
		},
	}
}

func (p *RustCodeParser) extractCallTarget(node *sitter.Node) string {
	if node == nil {
		return ""
	}

	switch node.Type() {
	case "identifier":
		return p.getNodeText(node)

	case "scoped_identifier":
		return p.getNodeText(node)

	case "field_expression":
		// obj.method
		return p.getNodeText(node)

	default:
		// Try to get text for other node types
		text := p.getNodeText(node)
		if len(text) < 100 { // Avoid huge expressions
			return text
		}
		return ""
	}
}

func (p *RustCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Clean comment markers
	cleanContent := content
	commentType := "line_comment"

	if strings.HasPrefix(content, "//") {
		cleanContent = strings.TrimPrefix(content, "//")
		cleanContent = strings.TrimPrefix(cleanContent, "/") // For doc comments
	} else if strings.HasPrefix(content, "/*") {
		cleanContent = strings.TrimPrefix(content, "/*")
		cleanContent = strings.TrimSuffix(cleanContent, "*/")
		commentType = "block_comment"
	}
	cleanContent = strings.TrimSpace(cleanContent)

	return &Element{
		ElementID:       generateID("rust_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":       "comment",
			"comment_type":   commentType,
			"language":       "rust",
			"line_number":    lineNum,
			"file_extension": p.fileExtension,
		},
	}
}

// Helper functions
func (p *RustCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *RustCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *RustCodeParser) walkNode(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNode(node.Child(i), fn)
	}
}

func (p *RustCodeParser) buildFunctionSignature(node *sitter.Node, funcName string) string {
	// Get parameters
	params := ""
	paramNode := p.findChildByType(node, "parameters")
	if paramNode != nil {
		params = p.getNodeText(paramNode)
	}

	// Get return type
	returnType := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "->" {
			// Next node should be the return type
			if i+1 < int(node.ChildCount()) {
				returnTypeNode := node.Child(i + 1)
				returnType = " -> " + p.getNodeText(returnTypeNode)
				break
			}
		}
	}

	return "fn " + funcName + params + returnType
}
