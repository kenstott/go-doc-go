package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// GoCodeParser parses Go source code files using go/ast
type GoCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	fset              *token.FileSet // File set for position tracking
	packageName       string         // Current package name
}

// NewGoCodeParser creates a new Go code parser with default settings
func NewGoCodeParser() *GoCodeParser {
	return &GoCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *GoCodeParser) GetName() string {
	return "go_code"
}

// GetSupportedFormats returns supported file formats
func (p *GoCodeParser) GetSupportedFormats() []string {
	return []string{".go", "go"}
}

// Parse implements the Parser interface for Go source code
func (p *GoCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Go parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Go file: %w", err)
	}

	// Parse Go source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %w", err)
	}

	// Store fset and package name in parser for reference extraction
	p.fset = fset
	if file.Name != nil {
		p.packageName = file.Name.Name
	}

	// Create parse result
	result := &ParseResult{
		Document: Document{
			ID:      req.ID,
			DocType: "go_source",
			Metadata: map[string]interface{}{
				"filename":       filepath.Base(filePath),
				"file_path":      filePath,
				"content_type":   "text/x-go",
				"parsing_method": "go/ast",
			},
		},
		Elements: []Element{},
	}

	// Extract package name
	if file.Name != nil {
		result.Document.Metadata["package"] = p.packageName
	}

	// Create root element
	rootElement := p.createRootElement(req.ID, string(content), filePath)
	result.Elements = append(result.Elements, rootElement)

	// Extract imports
	if p.ExtractImports {
		for _, imp := range file.Imports {
			element := p.createImportElement(req.ID, imp, fset, rootElement.ElementID)
			result.Elements = append(result.Elements, element)
		}
	}

	// Extract top-level declarations
	for _, decl := range file.Decls {
		elements := p.extractDeclaration(req.ID, decl, fset, rootElement.ElementID, file.Name.Name, "")
		result.Elements = append(result.Elements, elements...)
	}

	// Extract comments/documentation
	if p.ExtractComments && file.Comments != nil {
		for _, commentGroup := range file.Comments {
			element := p.createCommentElement(req.ID, commentGroup, fset, rootElement.ElementID)
			result.Elements = append(result.Elements, element)
		}
	}

	return result, nil
}

// SupportsStreaming returns whether the parser supports streaming
func (p *GoCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *GoCodeParser) Close() error {
	return nil
}

// createRootElement creates the root document element
func (p *GoCodeParser) createRootElement(docID, content, filePath string) Element {
	preview := p.truncateContent(content)
	lineCount := strings.Count(content, "\n") + 1

	return Element{
		ElementID:       generateID("go_root"),
		ElementType:     "document",
		ElementCategory: GetElementCategory("document"),
		Content:         content,
		ContentPreview:  preview,
		Position:        0,
		Depth:           0,
		ContentLocation: map[string]interface{}{
			"type":      "go_source_file",
			"file_path": filePath,
		},
		Metadata: map[string]interface{}{
			"element_type": "root",
			"line_count":   lineCount,
			"char_count":   len(content),
		},
	}
}

// createImportElement creates an import element
func (p *GoCodeParser) createImportElement(docID string, imp *ast.ImportSpec, fset *token.FileSet, parentID string) Element {
	importPath := strings.Trim(imp.Path.Value, "\"")
	pos := fset.Position(imp.Pos())
	lineNum := pos.Line

	var importAlias string
	if imp.Name != nil {
		importAlias = imp.Name.Name
	}

	namespace := importPath
	return Element{
		ElementID:       generateID("go_import"),
		ElementType:     "code_dependency", // Universal type
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         importPath,
		ContentPreview:  p.truncateContent(importPath),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           1,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		ContentLocation: map[string]interface{}{
			"type": "import",
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"dependency_kind":   "import",
			"target_namespace":  importPath,
			"dependency_alias":  importAlias,
			"code_element_kind": "import",
			"line_number":       lineNum,
			"language":          "go",

			// Go-specific metadata
			"import_path":       importPath,
			"import_alias":      importAlias,
		},
	}
}

// extractDeclaration extracts elements from a declaration
func (p *GoCodeParser) extractDeclaration(docID string, decl ast.Decl, fset *token.FileSet, parentID string, packageName string, className string) []Element {
	var elements []Element

	switch d := decl.(type) {
	case *ast.FuncDecl:
		elements = append(elements, p.createFunctionElement(docID, d, fset, parentID, packageName, className)...)
	case *ast.GenDecl:
		elements = append(elements, p.extractGenDecl(docID, d, fset, parentID, packageName)...)
	}

	return elements
}

// createFunctionElement creates function/method elements
func (p *GoCodeParser) createFunctionElement(docID string, fn *ast.FuncDecl, fset *token.FileSet, parentID string, packageName string, className string) []Element {
	var elements []Element

	pos := fset.Position(fn.Pos())
	lineNum := pos.Line
	funcName := fn.Name.Name

	// Universal element type: code_function (maps to both functions and methods)
	elementType := "code_function"
	var currentClassName string
	var functionKind string
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		functionKind = "method"
		// Extract receiver type (class name)
		currentClassName = p.extractReceiverType(fn.Recv.List[0].Type)
	} else {
		functionKind = "function"
		currentClassName = className
	}

	// Build function signature
	signature := p.buildFunctionSignature(fn)

	// Extract doc comment if present
	var docComment string
	if fn.Doc != nil {
		docComment = fn.Doc.Text()
	}

	// Create namespace (package.ClassName or just package)
	namespace := packageName
	if currentClassName != "" {
		namespace = fmt.Sprintf("%s.%s", packageName, currentClassName)
	}

	// Create function element using universal code model
	funcElement := Element{
		ElementID:       generateID("go_func"),
		ElementType:     elementType, // "code_function"
		ElementCategory: GetElementCategory(elementType),
		Content:         signature,
		ContentPreview:  p.truncateContent(signature),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           2,
		FunctionName:    &funcName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		ContentLocation: map[string]interface{}{
			"type": functionKind, // "function" or "method"
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"function_name":        funcName,
			"code_element_kind":    functionKind, // "function" or "method"
			"signature":            signature,
			"line_number":          lineNum,
			"doc_comment":          docComment,
			"visibility":           visibilityFromExported(ast.IsExported(funcName)),
			"language":             "go",

			// Go-specific metadata
			"is_exported":          ast.IsExported(funcName),
			"package":              packageName,
		},
	}

	if currentClassName != "" {
		funcElement.ClassName = &currentClassName
		funcElement.Metadata["class_name"] = currentClassName
		funcElement.Metadata["receiver_type"] = currentClassName
	}

	elements = append(elements, funcElement)

	// Extract parameters as child elements
	if fn.Type.Params != nil {
		for i, field := range fn.Type.Params.List {
			paramElements := p.createParameterElements(docID, field, i, fset, funcElement.ElementID, funcName)
			elements = append(elements, paramElements...)
		}
	}

	// Extract return types as child elements
	if fn.Type.Results != nil {
		for i, field := range fn.Type.Results.List {
			returnElement := p.createReturnTypeElement(docID, field, i, fset, funcElement.ElementID, funcName)
			elements = append(elements, returnElement)
		}
	}

	// NEW: Extract type usage from parameters
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			typeDepElem := p.createTypeUsageElement(field.Type, funcElement.ElementID, "parameter")
			if typeDepElem != nil {
				elements = append(elements, *typeDepElem)
			}
		}
	}

	// NEW: Extract type usage from return types
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			typeDepElem := p.createTypeUsageElement(field.Type, funcElement.ElementID, "return")
			if typeDepElem != nil {
				elements = append(elements, *typeDepElem)
			}
		}
	}

	// NEW: Parse function body for function calls
	if fn.Body != nil {
		callElems := p.parseFunctionCalls(fn.Body, funcElement.ElementID)
		elements = append(elements, callElems...)
	}

	return elements
}

// extractGenDecl extracts elements from general declarations (type, const, var)
func (p *GoCodeParser) extractGenDecl(docID string, decl *ast.GenDecl, fset *token.FileSet, parentID string, packageName string) []Element {
	var elements []Element

	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			elements = append(elements, p.createTypeElement(docID, s, decl.Doc, fset, parentID, packageName)...)
		case *ast.ValueSpec:
			elements = append(elements, p.createValueElement(docID, s, decl.Tok, fset, parentID, packageName)...)
		}
	}

	return elements
}

// createTypeElement creates type definition elements (struct, interface, etc.)
func (p *GoCodeParser) createTypeElement(docID string, spec *ast.TypeSpec, doc *ast.CommentGroup, fset *token.FileSet, parentID string, packageName string) []Element {
	var elements []Element

	pos := fset.Position(spec.Pos())
	lineNum := pos.Line
	typeName := spec.Name.Name

	// Extract doc comment
	var docComment string
	if doc != nil {
		docComment = doc.Text()
	}

	namespace := packageName
	typeSignature := fmt.Sprintf("type %s", typeName)

	// Determine type kind (language-specific detail)
	var typeKind string
	var isGrouping bool // Grouping vs Type
	switch t := spec.Type.(type) {
	case *ast.StructType:
		typeKind = "struct"
		isGrouping = true // Structs are groupings (containers)
		typeSignature = p.buildStructSignature(typeName, t)
	case *ast.InterfaceType:
		typeKind = "interface"
		isGrouping = true // Interfaces are groupings (containers)
		typeSignature = p.buildInterfaceSignature(typeName, t)
	default:
		typeKind = "type_alias"
		isGrouping = false // Type aliases are type definitions
	}

	// Universal element type
	var elementType string
	if isGrouping {
		elementType = "code_grouping" // For structs/interfaces (containers)
	} else {
		elementType = "code_type" // For type aliases
	}

	// Create type element using universal code model
	typeElement := Element{
		ElementID:       generateID("go_type"),
		ElementType:     elementType,
		ElementCategory: GetElementCategory(elementType),
		Content:         typeSignature,
		ContentPreview:  p.truncateContent(typeSignature),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           2,
		ClassName:       &typeName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &typeSignature,
		ContentLocation: map[string]interface{}{
			"type": typeKind, // "struct", "interface", or "type_alias"
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"type_name":         typeName,
			"code_element_kind": typeKind, // "struct", "interface", "type_alias"
			"grouping_kind":     typeKind, // Also store as grouping_kind for consistency
			"signature":         typeSignature,
			"line_number":       lineNum,
			"doc_comment":       docComment,
			"visibility":        visibilityFromExported(ast.IsExported(typeName)),
			"language":          "go",

			// Go-specific metadata
			"is_exported":       ast.IsExported(typeName),
			"package":           packageName,
		},
	}

	elements = append(elements, typeElement)

	// Extract fields for structs
	if structType, ok := spec.Type.(*ast.StructType); ok && structType.Fields != nil {
		for i, field := range structType.Fields.List {
			fieldElements := p.createFieldElements(docID, field, i, fset, typeElement.ElementID, typeName)
			elements = append(elements, fieldElements...)
		}
	}

	// Extract methods for interfaces
	if interfaceType, ok := spec.Type.(*ast.InterfaceType); ok && interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			methodElement := p.createInterfaceMethodElement(docID, method, fset, typeElement.ElementID, typeName)
			elements = append(elements, methodElement)
		}
	}

	return elements
}

// createFieldElements creates field elements for struct fields
func (p *GoCodeParser) createFieldElements(docID string, field *ast.Field, index int, fset *token.FileSet, parentID string, structName string) []Element {
	var elements []Element

	pos := fset.Position(field.Pos())
	lineNum := pos.Line

	// A field can have multiple names (e.g., x, y int)
	if len(field.Names) == 0 {
		// Anonymous field (embedded type)
		typeStr := p.exprToString(field.Type)
		fieldElement := Element{
			ElementID:       generateID("go_field"),
			ElementType:     "code_data", // Universal type
			ElementCategory: GetElementCategory("code_data"),
			Content:         typeStr,
			ContentPreview:  p.truncateContent(typeStr),
			ParentID:        parentID,
			Position:        lineNum,
			Depth:           3,
			ClassName:       &structName,
			LineNumber:      &lineNum,
			ContentLocation: map[string]interface{}{
				"type": "field",
				"line": lineNum,
			},
			Metadata: map[string]interface{}{
				// Universal metadata
				"data_name":         typeStr,
				"data_kind":         "field",
				"data_type":         typeStr,
				"code_element_kind": "field",
				"line_number":       lineNum,
				"language":          "go",

				// Go-specific metadata
				"is_embedded":       true,
				"struct_name":       structName,
			},
		}
		elements = append(elements, fieldElement)
	} else {
		for _, name := range field.Names {
			fieldName := name.Name
			fieldType := p.exprToString(field.Type)

			fieldElement := Element{
				ElementID:       generateID("go_field"),
				ElementType:     "code_data", // Universal type
				ElementCategory: GetElementCategory("code_data"),
				Content:         fmt.Sprintf("%s %s", fieldName, fieldType),
				ContentPreview:  p.truncateContent(fmt.Sprintf("%s %s", fieldName, fieldType)),
				ParentID:        parentID,
				Position:        lineNum,
				Depth:           3,
				ClassName:       &structName,
				LineNumber:      &lineNum,
				ContentLocation: map[string]interface{}{
					"type": "field",
					"line": lineNum,
				},
				Metadata: map[string]interface{}{
					// Universal metadata
					"data_name":         fieldName,
					"data_kind":         "field",
					"data_type":         fieldType,
					"code_element_kind": "field",
					"line_number":       lineNum,
					"visibility":        visibilityFromExported(ast.IsExported(fieldName)),
					"language":          "go",

					// Go-specific metadata
					"is_exported":       ast.IsExported(fieldName),
					"struct_name":       structName,
				},
			}

			// Extract field tag if present
			if field.Tag != nil {
				fieldElement.Metadata["field_tag"] = field.Tag.Value
			}

			elements = append(elements, fieldElement)

			// NEW: Add type usage for field
			typeDepElem := p.createTypeUsageElement(field.Type, parentID, "field")
			if typeDepElem != nil {
				elements = append(elements, *typeDepElem)
			}
		}
	}

	return elements
}

// createParameterElements creates parameter elements
func (p *GoCodeParser) createParameterElements(docID string, field *ast.Field, index int, fset *token.FileSet, parentID string, funcName string) []Element {
	var elements []Element

	pos := fset.Position(field.Pos())
	lineNum := pos.Line
	paramType := p.exprToString(field.Type)

	// Parameters can have multiple names or be unnamed
	if len(field.Names) == 0 {
		// Unnamed parameter
		paramElement := Element{
			ElementID:       generateID("go_param"),
			ElementType:     "code_data", // Universal type
			ElementCategory: GetElementCategory("code_data"),
			Content:         paramType,
			ContentPreview:  p.truncateContent(paramType),
			ParentID:        parentID,
			Position:        index,
			Depth:           3,
			FunctionName:    &funcName,
			LineNumber:      &lineNum,
			ContentLocation: map[string]interface{}{
				"type":  "parameter",
				"index": index,
				"line":  lineNum,
			},
			Metadata: map[string]interface{}{
				// Universal metadata
				"data_kind":         "parameter",
				"data_type":         paramType,
				"code_element_kind": "parameter",
				"line_number":       lineNum,
				"language":          "go",

				// Go-specific metadata
				"param_type":        paramType,
				"param_index":       index,
				"function_name":     funcName,
			},
		}
		elements = append(elements, paramElement)
	} else {
		for _, name := range field.Names {
			paramName := name.Name
			paramElement := Element{
				ElementID:       generateID("go_param"),
				ElementType:     "code_data", // Universal type
				ElementCategory: GetElementCategory("code_data"),
				Content:         fmt.Sprintf("%s %s", paramName, paramType),
				ContentPreview:  p.truncateContent(fmt.Sprintf("%s %s", paramName, paramType)),
				ParentID:        parentID,
				Position:        index,
				Depth:           3,
				FunctionName:    &funcName,
				LineNumber:      &lineNum,
				ContentLocation: map[string]interface{}{
					"type":  "parameter",
					"index": index,
					"line":  lineNum,
				},
				Metadata: map[string]interface{}{
					// Universal metadata
					"data_name":         paramName,
					"data_kind":         "parameter",
					"data_type":         paramType,
					"code_element_kind": "parameter",
					"line_number":       lineNum,
					"language":          "go",

					// Go-specific metadata
					"param_name":        paramName,
					"param_type":        paramType,
					"param_index":       index,
					"function_name":     funcName,
				},
			}
			elements = append(elements, paramElement)
		}
	}

	return elements
}

// createReturnTypeElement creates return type element
func (p *GoCodeParser) createReturnTypeElement(docID string, field *ast.Field, index int, fset *token.FileSet, parentID string, funcName string) Element {
	pos := fset.Position(field.Pos())
	lineNum := pos.Line
	returnType := p.exprToString(field.Type)

	var returnName string
	if len(field.Names) > 0 {
		returnName = field.Names[0].Name
	}

	content := returnType
	if returnName != "" {
		content = fmt.Sprintf("%s %s", returnName, returnType)
	}

	return Element{
		ElementID:       generateID("go_return"),
		ElementType:     "return_type",
		ElementCategory: GetElementCategory("return_type"),
		Content:         content,
		ContentPreview:  p.truncateContent(content),
		ParentID:        parentID,
		Position:        index,
		Depth:           3,
		FunctionName:    &funcName,
		LineNumber:      &lineNum,
		ContentLocation: map[string]interface{}{
			"type":  "return_type",
			"index": index,
			"line":  lineNum,
		},
		Metadata: map[string]interface{}{
			"return_type":   returnType,
			"return_name":   returnName,
			"return_index":  index,
			"line_number":   lineNum,
			"function_name": funcName,
		},
	}
}

// createValueElement creates const/var elements
func (p *GoCodeParser) createValueElement(docID string, spec *ast.ValueSpec, tok token.Token, fset *token.FileSet, parentID string, packageName string) []Element {
	var elements []Element

	pos := fset.Position(spec.Pos())
	lineNum := pos.Line

	for i, name := range spec.Names {
		varName := name.Name
		var varType string
		if spec.Type != nil {
			varType = p.exprToString(spec.Type)
		}

		var varValue string
		if i < len(spec.Values) {
			varValue = p.exprToString(spec.Values[i])
		}

		content := varName
		if varType != "" {
			content = fmt.Sprintf("%s %s", varName, varType)
		}
		if varValue != "" {
			content = fmt.Sprintf("%s = %s", content, varValue)
		}

		namespace := packageName
		dataKind := "variable"
		mutability := "mutable"
		if tok == token.CONST {
			dataKind = "constant"
			mutability = "immutable"
		}

		varElement := Element{
			ElementID:       generateID("go_var"),
			ElementType:     "code_data", // Universal type
			ElementCategory: GetElementCategory("code_data"),
			Content:         content,
			ContentPreview:  p.truncateContent(content),
			ParentID:        parentID,
			Position:        lineNum,
			Depth:           2,
			Namespace:       &namespace,
			LineNumber:      &lineNum,
			ContentLocation: map[string]interface{}{
				"type": dataKind,
				"line": lineNum,
			},
			Metadata: map[string]interface{}{
				// Universal metadata
				"data_name":         varName,
				"data_kind":         dataKind,
				"data_type":         varType,
				"code_element_kind": dataKind,
				"mutability":        mutability,
				"scope":             "global", // Package-level in Go
				"visibility":        visibilityFromExported(ast.IsExported(varName)),
				"line_number":       lineNum,
				"language":          "go",

				// Go-specific metadata
				"var_name":          varName,
				"var_type":          varType,
				"var_value":         varValue,
				"is_exported":       ast.IsExported(varName),
				"package":           packageName,
			},
		}

		elements = append(elements, varElement)
	}

	return elements
}

// createCommentElement creates documentation comment elements
func (p *GoCodeParser) createCommentElement(docID string, cg *ast.CommentGroup, fset *token.FileSet, parentID string) Element {
	pos := fset.Position(cg.Pos())
	lineNum := pos.Line
	commentText := cg.Text()

	return Element{
		ElementID:       generateID("go_comment"),
		ElementType:     "code_documentation", // Universal type
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         commentText,
		ContentPreview:  p.truncateContent(commentText),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           2,
		LineNumber:      &lineNum,
		ContentLocation: map[string]interface{}{
			"type": "doc_comment",
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"doc_kind":          "doc_comment",
			"doc_content":       commentText,
			"code_element_kind": "doc_comment",
			"line_number":       lineNum,
			"language":          "go",

			// Go-specific metadata
			"comment_text":      commentText,
		},
	}
}

// createInterfaceMethodElement creates interface method signature elements
func (p *GoCodeParser) createInterfaceMethodElement(docID string, field *ast.Field, fset *token.FileSet, parentID string, interfaceName string) Element {
	pos := fset.Position(field.Pos())
	lineNum := pos.Line

	var methodName string
	if len(field.Names) > 0 {
		methodName = field.Names[0].Name
	}

	methodSignature := p.exprToString(field.Type)

	return Element{
		ElementID:       generateID("go_method_sig"),
		ElementType:     "code_function", // Universal type
		ElementCategory: GetElementCategory("code_function"),
		Content:         fmt.Sprintf("%s %s", methodName, methodSignature),
		ContentPreview:  p.truncateContent(fmt.Sprintf("%s %s", methodName, methodSignature)),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           3,
		FunctionName:    &methodName,
		ClassName:       &interfaceName,
		LineNumber:      &lineNum,
		Signature:       &methodSignature,
		ContentLocation: map[string]interface{}{
			"type": "interface_method",
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"function_name":     methodName,
			"code_element_kind": "method",
			"function_kind":     "method",
			"signature":         methodSignature,
			"line_number":       lineNum,
			"visibility":        "public", // Interface methods are always public
			"language":          "go",

			// Go-specific metadata
			"method_name":       methodName,
			"interface_name":    interfaceName,
			"is_interface_method": true,
		},
	}
}

// Helper functions

// visibilityFromExported converts Go's exported concept to universal visibility
func visibilityFromExported(isExported bool) string {
	if isExported {
		return "public"
	}
	return "private"
}

// extractReceiverType extracts the receiver type from a method
func (p *GoCodeParser) extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return p.extractReceiverType(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// buildFunctionSignature builds a function signature string
func (p *GoCodeParser) buildFunctionSignature(fn *ast.FuncDecl) string {
	var parts []string

	parts = append(parts, "func")

	// Add receiver for methods
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		receiver := p.exprToString(fn.Recv.List[0].Type)
		parts = append(parts, fmt.Sprintf("(%s)", receiver))
	}

	// Add function name
	parts = append(parts, fn.Name.Name)

	// Add parameters
	params := p.fieldListToString(fn.Type.Params)
	parts = append(parts, fmt.Sprintf("(%s)", params))

	// Add return types
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		results := p.fieldListToString(fn.Type.Results)
		if len(fn.Type.Results.List) > 1 || (len(fn.Type.Results.List) == 1 && len(fn.Type.Results.List[0].Names) > 0) {
			parts = append(parts, fmt.Sprintf("(%s)", results))
		} else {
			parts = append(parts, results)
		}
	}

	return strings.Join(parts, " ")
}

// buildStructSignature builds a struct signature
func (p *GoCodeParser) buildStructSignature(name string, structType *ast.StructType) string {
	if structType.Fields == nil || len(structType.Fields.List) == 0 {
		return fmt.Sprintf("type %s struct{}", name)
	}
	return fmt.Sprintf("type %s struct {...}", name)
}

// buildInterfaceSignature builds an interface signature
func (p *GoCodeParser) buildInterfaceSignature(name string, interfaceType *ast.InterfaceType) string {
	if interfaceType.Methods == nil || len(interfaceType.Methods.List) == 0 {
		return fmt.Sprintf("type %s interface{}", name)
	}
	return fmt.Sprintf("type %s interface {...}", name)
}

// fieldListToString converts a field list to a string
func (p *GoCodeParser) fieldListToString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}

	var parts []string
	for _, field := range fl.List {
		typeStr := p.exprToString(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			for _, name := range field.Names {
				parts = append(parts, fmt.Sprintf("%s %s", name.Name, typeStr))
			}
		}
	}
	return strings.Join(parts, ", ")
}

// exprToString converts an expression to a string
func (p *GoCodeParser) exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + p.exprToString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", p.exprToString(t.Key), p.exprToString(t.Value))
	case *ast.FuncType:
		return "func"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", p.exprToString(t.X), t.Sel.Name)
	case *ast.ChanType:
		return "chan"
	case *ast.Ellipsis:
		return "..." + p.exprToString(t.Elt)
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("%T", t)
	}
}

// truncateContent truncates content for preview
func (p *GoCodeParser) truncateContent(content string) string {
	// Remove extra whitespace and normalize
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	if len(content) <= p.MaxContentPreview {
		return content
	}

	// Truncate by runes, not bytes
	runes := []rune(content)
	if len(runes) <= p.MaxContentPreview {
		return content
	}

	truncated := string(runes[:p.MaxContentPreview-3])

	// Try to break at word boundary
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > p.MaxContentPreview/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// parseFunctionCalls extracts function calls from a function body
func (p *GoCodeParser) parseFunctionCalls(body *ast.BlockStmt, parentFuncID string) []Element {
	var calls []Element

	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			callElem := p.createFunctionCallElement(call, parentFuncID)
			if callElem != nil {
				calls = append(calls, *callElem)
			}
		}
		return true
	})

	return calls
}

// createFunctionCallElement creates a code_dependency element for a function call
func (p *GoCodeParser) createFunctionCallElement(call *ast.CallExpr, parentID string) *Element {
	// Extract function name from call expression
	funcName := p.extractFunctionName(call.Fun)
	if funcName == "" {
		return nil
	}

	pos := p.fset.Position(call.Pos())
	lineNum := pos.Line

	// Determine call type (direct vs indirect)
	callType := "direct"
	if _, ok := call.Fun.(*ast.Ident); !ok {
		// If not a simple identifier, might be indirect (method, package func, etc.)
		if _, ok := call.Fun.(*ast.SelectorExpr); ok {
			callType = "direct" // Method calls and package functions are still direct
		} else {
			callType = "indirect" // Function pointers, etc.
		}
	}

	content := funcName + "()"
	namespace := p.packageName

	return &Element{
		ElementID:       generateID("go_call"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  p.truncateContent(content),
		ParentID:        parentID,
		Position:        lineNum,
		Depth:           4,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		ContentLocation: map[string]interface{}{
			"type": "function_call",
			"line": lineNum,
		},
		Metadata: map[string]interface{}{
			// Universal metadata
			"dependency_kind":  "function_call",
			"target_function":  funcName,
			"call_type":        callType,
			"line_number":      lineNum,
			"language":         "go",

			// Go-specific metadata
			"call_expression":  content,
		},
	}
}

// extractFunctionName extracts the function name from a call expression
func (p *GoCodeParser) extractFunctionName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// Simple function call: foo()
		return e.Name
	case *ast.SelectorExpr:
		// Method call or package function: obj.Method() or pkg.Func()
		prefix := p.extractFunctionName(e.X)
		if prefix != "" {
			return fmt.Sprintf("%s.%s", prefix, e.Sel.Name)
		}
		return e.Sel.Name
	default:
		// Skip complex expressions (function pointers, etc.)
		return ""
	}
}

// createTypeUsageElement creates a code_dependency element for type usage
func (p *GoCodeParser) createTypeUsageElement(
	typeExpr ast.Expr,
	parentID string,
	context string, // "parameter", "return", "field"
) *Element {
	if typeExpr == nil {
		return nil
	}

	typeName := p.exprToString(typeExpr)
	if typeName == "" {
		return nil
	}

	// Skip basic types (built-in types are not interesting dependencies)
	basicTypes := map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"string": true, "bool": true, "byte": true, "rune": true,
		"error": true, // error is a built-in interface
	}

	// Extract base type name (remove *, [], map[], etc.)
	baseType := typeName
	baseType = strings.TrimPrefix(baseType, "*")
	baseType = strings.TrimPrefix(baseType, "[]")
	if strings.HasPrefix(baseType, "map[") {
		return nil // Skip map types for now
	}

	// Extract package if qualified type
	var packageName string
	if strings.Contains(baseType, ".") {
		parts := strings.Split(baseType, ".")
		if len(parts) == 2 {
			packageName = parts[0]
			baseType = parts[1]
		}
	}

	// Skip basic types
	if basicTypes[baseType] {
		return nil
	}

	namespace := p.packageName
	content := typeName

	return &Element{
		ElementID:       generateID("go_type_usage"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  p.truncateContent(content),
		ParentID:        parentID,
		Namespace:       &namespace,
		Metadata: map[string]interface{}{
			// Universal metadata
			"dependency_kind":  "type_usage",
			"usage_context":    context,
			"target_type":      typeName,
			"target_package":   packageName,
			"language":         "go",

			// Go-specific metadata
			"type_expression":  content,
		},
	}
}
