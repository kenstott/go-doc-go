package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// JavaCodeParser parses Java source code files using tree-sitter
type JavaCodeParser struct {
	MaxContentPreview int
	ExtractComments   bool
	ExtractImports    bool
	sourceCode        []byte
	packageName       string
	classStack        []string // Stack to track nested class names
}

// NewJavaCodeParser creates a new Java code parser with default settings
func NewJavaCodeParser() *JavaCodeParser {
	return &JavaCodeParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
		ExtractImports:    true,
	}
}

// GetName returns the parser name
func (p *JavaCodeParser) GetName() string {
	return "java_code"
}

// GetSupportedFormats returns supported file formats
func (p *JavaCodeParser) GetSupportedFormats() []string {
	return []string{".java", "java"}
}

// SupportsStreaming indicates if parser can handle streaming content
func (p *JavaCodeParser) SupportsStreaming() bool {
	return false
}

// Close releases any resources held by the parser
func (p *JavaCodeParser) Close() error {
	return nil
}

// Parse implements the Parser interface for Java source code
func (p *JavaCodeParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from request
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Java parser: %T (expected file path string)", req.Content)
	}

	// Read file
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Java file: %w", err)
	}

	p.sourceCode = sourceCode
	p.packageName = ""

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Java source: %w", err)
	}
	defer tree.Close()

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "java_source",
		Title:   filepath.Base(filePath),
		Metadata: map[string]interface{}{
			"path":     filePath,
			"language": "java",
		},
	}

	// Create document element
	docElement := p.createDocumentElement(req.ID, filePath, sourceCode)
	elements := []Element{docElement}

	// Extract elements from AST
	rootNode := tree.RootNode()
	extractedElements := p.extractElements(rootNode, docElement.ElementID)
	elements = append(elements, extractedElements...)

	// Update document with package name
	if p.packageName != "" {
		doc.Metadata["package"] = p.packageName
	}

	return &ParseResult{
		Document: doc,
		Elements: elements,
	}, nil
}

func (p *JavaCodeParser) createDocumentElement(id, filePath string, sourceCode []byte) Element {
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
			"doc_type": "java_source",
			"path":     filePath,
			"language": "java",
		},
	}
}

func (p *JavaCodeParser) extractElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		switch nodeType {
		case "package_declaration":
			p.extractPackageName(child)

		case "import_declaration":
			if p.ExtractImports {
				elem := p.createImportElement(child, parentID)
				if elem != nil {
					elements = append(elements, *elem)
				}
			}

		case "class_declaration":
			classElements := p.createClassElements(child, parentID)
			elements = append(elements, classElements...)

		case "interface_declaration":
			interfaceElements := p.createInterfaceElements(child, parentID)
			elements = append(elements, interfaceElements...)

		case "enum_declaration":
			enumElements := p.createEnumElements(child, parentID)
			elements = append(elements, enumElements...)

		case "method_declaration":
			methodElements := p.createMethodElements(child, parentID)
			elements = append(elements, methodElements...)

		case "constructor_declaration":
			constructorElements := p.createConstructorElements(child, parentID)
			elements = append(elements, constructorElements...)

		case "field_declaration":
			fieldElements := p.createFieldElements(child, parentID)
			elements = append(elements, fieldElements...)

		case "block_comment", "line_comment":
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

func (p *JavaCodeParser) extractPackageName(node *sitter.Node) {
	// Find scoped_identifier in package declaration
	scopedId := p.findChildByType(node, "scoped_identifier")
	if scopedId != nil {
		p.packageName = p.getNodeText(scopedId)
	} else {
		// Try identifier
		identifier := p.findChildByType(node, "identifier")
		if identifier != nil {
			p.packageName = p.getNodeText(identifier)
		}
	}
}

func (p *JavaCodeParser) createImportElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract import path
	scopedId := p.findChildByType(node, "scoped_identifier")
	var targetNamespace string
	if scopedId != nil {
		targetNamespace = p.getNodeText(scopedId)
	} else {
		identifier := p.findChildByType(node, "identifier")
		if identifier != nil {
			targetNamespace = p.getNodeText(identifier)
		}
	}

	// Check for static import
	dependencyKind := "import"
	isStatic := strings.Contains(content, "import static")
	if isStatic {
		dependencyKind = "static_import"
	}

	// Check for wildcard import
	isWildcard := strings.Contains(content, "*")

	namespace := p.packageName

	metadata := map[string]interface{}{
		"dependency_kind":  dependencyKind,
		"target_namespace": targetNamespace,
		"language":         "java",
		"line_number":      lineNum,
		"is_static":        isStatic,
		"is_wildcard":      isWildcard,
	}

	return &Element{
		ElementID:       generateID("java_import"),
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

func (p *JavaCodeParser) createClassElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract class name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	className := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Extract superclass and interfaces
	var superclass string
	var interfaces []string

	superclassNode := p.findChildByType(node, "superclass")
	if superclassNode != nil {
		typeNode := p.findChildByType(superclassNode, "type_identifier")
		if typeNode != nil {
			superclass = p.getNodeText(typeNode)
		}
	}

	interfacesNode := p.findChildByType(node, "super_interfaces")
	if interfacesNode != nil {
		interfaces = p.extractTypeList(interfacesNode)
	}

	// Build signature
	signature := strings.Join(modifiers, " ")
	if signature != "" {
		signature += " "
	}
	signature += "class " + className
	if superclass != "" {
		signature += " extends " + superclass
	}
	if len(interfaces) > 0 {
		signature += " implements " + strings.Join(interfaces, ", ")
	}

	namespace := p.packageName
	content := p.getNodeText(node)

	classElement := Element{
		ElementID:       generateID("java_class"),
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
			"language":          "java",
			"modifiers":         modifiers,
			"superclass":        superclass,
			"interfaces":        interfaces,
			"line_number":       lineNum,
		},
	}

	elements = append(elements, classElement)

	// Extract class body
	bodyNode := p.findChildByType(node, "class_body")
	if bodyNode != nil {
		p.classStack = append(p.classStack, className)
		bodyElements := p.extractElements(bodyNode, classElement.ElementID)
		elements = append(elements, bodyElements...)
		if len(p.classStack) > 0 {
			p.classStack = p.classStack[:len(p.classStack)-1]
		}
	}

	// Extract Javadoc
	javadocElem := p.extractJavadoc(node, classElement.ElementID)
	if javadocElem != nil {
		elements = append(elements, *javadocElem)
	}

	return elements
}

func (p *JavaCodeParser) createInterfaceElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract interface name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	interfaceName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Extract extends interfaces
	var extendsInterfaces []string
	extendsNode := p.findChildByType(node, "extends_interfaces")
	if extendsNode != nil {
		extendsInterfaces = p.extractTypeList(extendsNode)
	}

	// Build signature
	signature := strings.Join(modifiers, " ")
	if signature != "" {
		signature += " "
	}
	signature += "interface " + interfaceName
	if len(extendsInterfaces) > 0 {
		signature += " extends " + strings.Join(extendsInterfaces, ", ")
	}

	namespace := p.packageName
	content := p.getNodeText(node)

	interfaceElement := Element{
		ElementID:       generateID("java_interface"),
		ElementType:     "code_type",
		ElementCategory: GetElementCategory("code_type"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &interfaceName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "interface",
			"type_kind":         "interface",
			"visibility":        visibility,
			"language":          "java",
			"modifiers":         modifiers,
			"extends":           extendsInterfaces,
			"line_number":       lineNum,
		},
	}

	elements = append(elements, interfaceElement)

	// Extract interface body
	bodyNode := p.findChildByType(node, "interface_body")
	if bodyNode != nil {
		p.classStack = append(p.classStack, interfaceName)
		bodyElements := p.extractElements(bodyNode, interfaceElement.ElementID)
		elements = append(elements, bodyElements...)
		if len(p.classStack) > 0 {
			p.classStack = p.classStack[:len(p.classStack)-1]
		}
	}

	// Extract Javadoc
	javadocElem := p.extractJavadoc(node, interfaceElement.ElementID)
	if javadocElem != nil {
		elements = append(elements, *javadocElem)
	}

	return elements
}

func (p *JavaCodeParser) createEnumElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract enum name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	enumName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Build signature
	signature := strings.Join(modifiers, " ")
	if signature != "" {
		signature += " "
	}
	signature += "enum " + enumName

	namespace := p.packageName
	content := p.getNodeText(node)

	enumElement := Element{
		ElementID:       generateID("java_enum"),
		ElementType:     "code_type",
		ElementCategory: GetElementCategory("code_type"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		ClassName:       &enumName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "enum",
			"type_kind":         "enum",
			"visibility":        visibility,
			"language":          "java",
			"modifiers":         modifiers,
			"line_number":       lineNum,
		},
	}

	elements = append(elements, enumElement)

	// Extract enum body
	bodyNode := p.findChildByType(node, "enum_body")
	if bodyNode != nil {
		p.classStack = append(p.classStack, enumName)
		bodyElements := p.extractElements(bodyNode, enumElement.ElementID)
		elements = append(elements, bodyElements...)
		if len(p.classStack) > 0 {
			p.classStack = p.classStack[:len(p.classStack)-1]
		}
	}

	// Extract Javadoc
	javadocElem := p.extractJavadoc(node, enumElement.ElementID)
	if javadocElem != nil {
		elements = append(elements, *javadocElem)
	}

	return elements
}

func (p *JavaCodeParser) createMethodElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract method name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	methodName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Determine method kind
	methodKind := "method"
	if contains(modifiers, "static") {
		methodKind = "static_method"
	}
	if contains(modifiers, "abstract") {
		methodKind = "abstract_method"
	}

	// Extract return type
	var returnType string
	typeNode := p.findChildByType(node, "type_identifier")
	if typeNode == nil {
		typeNode = p.findChildByType(node, "void_type")
		if typeNode == nil {
			// Try generic_type or other type nodes
			for j := 0; j < int(node.ChildCount()); j++ {
				child := node.Child(j)
				if strings.Contains(child.Type(), "type") {
					typeNode = child
					break
				}
			}
		}
	}
	if typeNode != nil {
		returnType = p.getNodeText(typeNode)
	}

	// Extract parameters
	paramsNode := p.findChildByType(node, "formal_parameters")
	var paramTypes []string
	if paramsNode != nil {
		paramTypes = p.extractParameterTypes(paramsNode)
	}

	// Extract throws clause
	var throwsTypes []string
	throwsNode := p.findChildByType(node, "throws")
	if throwsNode != nil {
		throwsTypes = p.extractTypeList(throwsNode)
	}

	// Extract annotations
	annotations := p.extractAnnotations(node)

	// Build signature
	signature := p.buildMethodSignature(modifiers, returnType, methodName, paramTypes)

	namespace := p.packageName
	className := p.getCurrentClassName()
	if className != "" {
		namespace = fmt.Sprintf("%s.%s", p.packageName, className)
	}

	content := p.getNodeText(node)

	methodElement := Element{
		ElementID:       generateID("java_method"),
		ElementType:     "code_function",
		ElementCategory: GetElementCategory("code_function"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		FunctionName:    &methodName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": methodKind,
			"visibility":        visibility,
			"language":          "java",
			"modifiers":         modifiers,
			"return_type":       returnType,
			"annotations":       annotations,
			"throws":            throwsTypes,
			"line_number":       lineNum,
		},
	}

	if className != "" {
		methodElement.ClassName = &className
	}

	elements = append(elements, methodElement)

	// Extract parameters as code_data elements
	if paramsNode != nil {
		paramElements := p.createParameterElements(paramsNode, methodElement.ElementID)
		elements = append(elements, paramElements...)
	}

	// Extract return type as type usage
	if returnType != "" && returnType != "void" {
		returnTypeElem := p.createTypeUsageElement(returnType, methodElement.ElementID, "return")
		if returnTypeElem != nil {
			elements = append(elements, *returnTypeElem)
		}
	}

	// Extract method body for function calls
	bodyNode := p.findChildByType(node, "block")
	if bodyNode != nil {
		callElements := p.extractFunctionCalls(bodyNode, methodElement.ElementID)
		elements = append(elements, callElements...)

		// Extract inline comments within method body
		if p.ExtractComments {
			inlineComments := p.extractInlineComments(bodyNode, methodElement.ElementID)
			elements = append(elements, inlineComments...)
		}
	}

	// Extract Javadoc
	javadocElem := p.extractJavadoc(node, methodElement.ElementID)
	if javadocElem != nil {
		elements = append(elements, *javadocElem)
	}

	return elements
}

func (p *JavaCodeParser) createConstructorElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract constructor name
	nameNode := p.findChildByType(node, "identifier")
	if nameNode == nil {
		return elements
	}

	constructorName := p.getNodeText(nameNode)
	lineNum := int(node.StartPoint().Row) + 1

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Extract parameters
	paramsNode := p.findChildByType(node, "formal_parameters")
	var paramTypes []string
	if paramsNode != nil {
		paramTypes = p.extractParameterTypes(paramsNode)
	}

	// Extract throws clause
	var throwsTypes []string
	throwsNode := p.findChildByType(node, "throws")
	if throwsNode != nil {
		throwsTypes = p.extractTypeList(throwsNode)
	}

	// Extract annotations
	annotations := p.extractAnnotations(node)

	// Build signature
	signature := strings.Join(modifiers, " ")
	if signature != "" {
		signature += " "
	}
	signature += constructorName + "(" + strings.Join(paramTypes, ", ") + ")"

	namespace := p.packageName
	className := p.getCurrentClassName()
	if className != "" {
		namespace = fmt.Sprintf("%s.%s", p.packageName, className)
	}

	content := p.getNodeText(node)

	constructorElement := Element{
		ElementID:       generateID("java_constructor"),
		ElementType:     "code_function",
		ElementCategory: GetElementCategory("code_function"),
		Content:         content,
		ContentPreview:  truncate(signature, p.MaxContentPreview),
		ParentID:        parentID,
		FunctionName:    &constructorName,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Signature:       &signature,
		Metadata: map[string]interface{}{
			"code_element_kind": "constructor",
			"visibility":        visibility,
			"language":          "java",
			"modifiers":         modifiers,
			"annotations":       annotations,
			"throws":            throwsTypes,
			"line_number":       lineNum,
		},
	}

	if className != "" {
		constructorElement.ClassName = &className
	}

	elements = append(elements, constructorElement)

	// Extract parameters as code_data elements
	if paramsNode != nil {
		paramElements := p.createParameterElements(paramsNode, constructorElement.ElementID)
		elements = append(elements, paramElements...)
	}

	// Extract constructor body for function calls
	bodyNode := p.findChildByType(node, "constructor_body")
	if bodyNode != nil {
		callElements := p.extractFunctionCalls(bodyNode, constructorElement.ElementID)
		elements = append(elements, callElements...)

		// Extract inline comments within constructor body
		if p.ExtractComments {
			inlineComments := p.extractInlineComments(bodyNode, constructorElement.ElementID)
			elements = append(elements, inlineComments...)
		}
	}

	// Extract Javadoc
	javadocElem := p.extractJavadoc(node, constructorElement.ElementID)
	if javadocElem != nil {
		elements = append(elements, *javadocElem)
	}

	return elements
}

func (p *JavaCodeParser) createFieldElements(node *sitter.Node, parentID string) []Element {
	var elements []Element

	// Extract modifiers
	modifiers := p.extractModifiers(node)
	visibility := p.determineVisibility(modifiers)

	// Extract field type
	var fieldType string
	typeNode := p.findChildByType(node, "type_identifier")
	if typeNode == nil {
		// Try generic_type or other type constructs
		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			if strings.Contains(child.Type(), "type") {
				typeNode = child
				break
			}
		}
	}
	if typeNode != nil {
		fieldType = p.getNodeText(typeNode)
	}

	// Extract variable declarators
	declaratorNode := p.findChildByType(node, "variable_declarator")
	if declaratorNode == nil {
		return elements
	}

	fieldName := ""
	nameNode := p.findChildByType(declaratorNode, "identifier")
	if nameNode != nil {
		fieldName = p.getNodeText(nameNode)
	}

	lineNum := int(node.StartPoint().Row) + 1
	content := p.getNodeText(node)

	namespace := p.packageName
	className := p.getCurrentClassName()
	if className != "" {
		namespace = fmt.Sprintf("%s.%s", p.packageName, className)
	}

	scope := "instance"
	if contains(modifiers, "static") {
		scope = "static"
	}

	mutability := "mutable"
	if contains(modifiers, "final") {
		mutability = "immutable"
	}

	fieldElement := Element{
		ElementID:       generateID("java_field"),
		ElementType:     "code_data",
		ElementCategory: GetElementCategory("code_data"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		Namespace:       &namespace,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"data_kind":   "field",
			"data_name":   fieldName,
			"data_type":   fieldType,
			"scope":       scope,
			"mutability":  mutability,
			"visibility":  visibility,
			"modifiers":   modifiers,
			"language":    "java",
			"line_number": lineNum,
		},
	}

	if className != "" {
		fieldElement.ClassName = &className
	}

	elements = append(elements, fieldElement)

	// Create type usage dependency
	if fieldType != "" {
		typeUsageElem := p.createTypeUsageElement(fieldType, parentID, "field")
		if typeUsageElem != nil {
			elements = append(elements, *typeUsageElem)
		}
	}

	return elements
}

func (p *JavaCodeParser) createParameterElements(paramsNode *sitter.Node, parentMethodID string) []Element {
	var elements []Element

	for i := 0; i < int(paramsNode.ChildCount()); i++ {
		child := paramsNode.Child(i)

		if child.Type() == "formal_parameter" {
			// Extract parameter type
			var paramType string
			typeNode := p.findChildByType(child, "type_identifier")
			if typeNode == nil {
				// Try other type nodes
				for j := 0; j < int(child.ChildCount()); j++ {
					typeChild := child.Child(j)
					if strings.Contains(typeChild.Type(), "type") {
						typeNode = typeChild
						break
					}
				}
			}
			if typeNode != nil {
				paramType = p.getNodeText(typeNode)
			}

			// Extract parameter name
			var paramName string
			nameNode := p.findChildByType(child, "identifier")
			if nameNode != nil {
				paramName = p.getNodeText(nameNode)
			}

			if paramName == "" {
				continue
			}

			lineNum := int(child.StartPoint().Row) + 1
			content := p.getNodeText(child)

			paramElement := Element{
				ElementID:       generateID("java_param"),
				ElementType:     "code_data",
				ElementCategory: GetElementCategory("code_data"),
				Content:         content,
				ContentPreview:  truncate(content, p.MaxContentPreview),
				ParentID:        parentMethodID,
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"data_kind":   "parameter",
					"data_name":   paramName,
					"data_type":   paramType,
					"scope":       "local",
					"mutability":  "mutable",
					"language":    "java",
					"line_number": lineNum,
				},
			}

			elements = append(elements, paramElement)

			// Create type usage dependency
			if paramType != "" {
				typeUsageElem := p.createTypeUsageElement(paramType, parentMethodID, "parameter")
				if typeUsageElem != nil {
					elements = append(elements, *typeUsageElem)
				}
			}
		}
	}

	return elements
}

func (p *JavaCodeParser) extractFunctionCalls(bodyNode *sitter.Node, parentMethodID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			callElem := p.createFunctionCallElement(node, parentMethodID)
			if callElem != nil {
				elements = append(elements, *callElem)
			}
		}
	})

	return elements
}

func (p *JavaCodeParser) extractInlineComments(bodyNode *sitter.Node, parentFunctionID string) []Element {
	var elements []Element

	p.walkNode(bodyNode, func(node *sitter.Node) {
		nodeType := node.Type()

		// Only process inline comments (not Javadoc which is handled separately)
		if nodeType == "line_comment" || nodeType == "block_comment" {
			// Skip Javadoc comments (/** ... */)
			if nodeType == "block_comment" {
				content := p.getNodeText(node)
				if strings.HasPrefix(content, "/**") {
					return // Skip Javadoc
				}
			}

			elem := p.createInlineCommentElement(node, parentFunctionID)
			if elem != nil {
				elements = append(elements, *elem)
			}
		}
	})

	return elements
}

func (p *JavaCodeParser) createInlineCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	commentType := node.Type() // "line_comment" or "block_comment"

	// Clean up comment markers
	cleanContent := content
	if commentType == "line_comment" {
		cleanContent = strings.TrimPrefix(cleanContent, "//")
	} else {
		cleanContent = strings.TrimPrefix(cleanContent, "/*")
		cleanContent = strings.TrimSuffix(cleanContent, "*/")
	}
	cleanContent = strings.TrimSpace(cleanContent)

	// Detect markers (TODO, FIXME, HACK, XXX, NOTE, BUG, OPTIMIZE)
	markers := detectCommentMarkers(cleanContent)

	return &Element{
		ElementID:       generateID("java_inline_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         cleanContent,
		ContentPreview:  truncate(cleanContent, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":      "inline_comment",
			"comment_type":  commentType,
			"language":      "java",
			"line_number":   lineNum,
			"markers":       markers,
		},
	}
}

func detectCommentMarkers(content string) []string {
	var markers []string
	upperContent := strings.ToUpper(content)

	markerKeywords := []string{"TODO", "FIXME", "HACK", "XXX", "NOTE", "BUG", "OPTIMIZE", "TEMP"}
	for _, marker := range markerKeywords {
		if strings.Contains(upperContent, marker) {
			markers = append(markers, marker)
		}
	}

	return markers
}

func (p *JavaCodeParser) createFunctionCallElement(node *sitter.Node, parentID string) *Element {
	// Extract method name
	var methodName string
	nameNode := p.findChildByType(node, "identifier")
	if nameNode != nil {
		methodName = p.getNodeText(nameNode)
	}

	// Check for object.method() pattern
	objectNode := p.findChildByType(node, "field_access")
	if objectNode != nil {
		// This is object.method()
		methodName = p.getNodeText(objectNode) + "." + methodName
	}

	if methodName == "" {
		return nil
	}

	lineNum := int(node.StartPoint().Row) + 1

	callType := "direct"
	if strings.Contains(methodName, ".") {
		callType = "method_call"
	}

	content := methodName + "()"

	return &Element{
		ElementID:       generateID("java_call"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"dependency_kind": "function_call",
			"target_function": methodName,
			"call_type":       callType,
			"line_number":     lineNum,
			"language":        "java",
		},
	}
}

func (p *JavaCodeParser) createTypeUsageElement(typeStr string, parentID string, context string) *Element {
	if typeStr == "" {
		return nil
	}

	// Skip primitive types
	primitiveTypes := map[string]bool{
		"int": true, "long": true, "short": true, "byte": true,
		"float": true, "double": true, "char": true, "boolean": true,
		"void": true,
	}

	cleanType := strings.TrimSpace(typeStr)
	// Remove generic parameters for checking
	baseType := strings.Split(cleanType, "<")[0]

	if primitiveTypes[baseType] {
		return nil
	}

	return &Element{
		ElementID:       generateID("java_type_usage"),
		ElementType:     "code_dependency",
		ElementCategory: GetElementCategory("code_dependency"),
		Content:         cleanType,
		ParentID:        parentID,
		Metadata: map[string]interface{}{
			"dependency_kind": "type_usage",
			"usage_context":   context,
			"target_type":     cleanType,
			"language":        "java",
		},
	}
}

func (p *JavaCodeParser) extractJavadoc(node *sitter.Node, parentID string) *Element {
	// Look for block_comment before the node
	prev := node.PrevSibling()
	if prev != nil && prev.Type() == "block_comment" {
		content := p.getNodeText(prev)
		// Check if it's a Javadoc comment (starts with /**)
		if strings.HasPrefix(content, "/**") {
			lineNum := int(prev.StartPoint().Row) + 1
			// Remove comment markers
			content = strings.TrimPrefix(content, "/**")
			content = strings.TrimSuffix(content, "*/")
			content = strings.TrimSpace(content)

			return &Element{
				ElementID:       generateID("java_javadoc"),
				ElementType:     "code_documentation",
				ElementCategory: GetElementCategory("code_documentation"),
				Content:         content,
				ContentPreview:  truncate(content, p.MaxContentPreview),
				ParentID:        parentID,
				LineNumber:      &lineNum,
				Metadata: map[string]interface{}{
					"doc_kind":    "javadoc",
					"language":    "java",
					"line_number": lineNum,
				},
			}
		}
	}

	return nil
}

func (p *JavaCodeParser) createCommentElement(node *sitter.Node, parentID string) *Element {
	content := p.getNodeText(node)
	lineNum := int(node.StartPoint().Row) + 1

	docKind := "comment"
	if node.Type() == "block_comment" {
		docKind = "block_comment"
		// Remove /* and */
		content = strings.TrimPrefix(content, "/*")
		content = strings.TrimSuffix(content, "*/")
	} else {
		// line_comment
		content = strings.TrimPrefix(content, "//")
	}
	content = strings.TrimSpace(content)

	return &Element{
		ElementID:       generateID("java_comment"),
		ElementType:     "code_documentation",
		ElementCategory: GetElementCategory("code_documentation"),
		Content:         content,
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ParentID:        parentID,
		LineNumber:      &lineNum,
		Metadata: map[string]interface{}{
			"doc_kind":    docKind,
			"language":    "java",
			"line_number": lineNum,
		},
	}
}

// Helper functions

func (p *JavaCodeParser) extractModifiers(node *sitter.Node) []string {
	var modifiers []string

	modifiersNode := p.findChildByType(node, "modifiers")
	if modifiersNode != nil {
		for i := 0; i < int(modifiersNode.ChildCount()); i++ {
			child := modifiersNode.Child(i)
			modifier := p.getNodeText(child)
			if modifier != "" && modifier != " " {
				modifiers = append(modifiers, modifier)
			}
		}
	}

	return modifiers
}

func (p *JavaCodeParser) extractAnnotations(node *sitter.Node) []string {
	var annotations []string

	// First, check if the node itself has a modifiers child with annotations
	modifiersNode := p.findChildByType(node, "modifiers")
	if modifiersNode != nil {
		// Extract annotations from modifiers children
		for i := 0; i < int(modifiersNode.ChildCount()); i++ {
			child := modifiersNode.Child(i)
			childType := child.Type()
			if childType == "marker_annotation" || childType == "annotation" ||
			   strings.Contains(childType, "annotation") {
				annotations = append(annotations, p.getNodeText(child))
			}
		}
	}

	// Also look for annotations as previous siblings (for cases where they're separate)
	current := node.PrevSibling()
	for current != nil {
		nodeType := current.Type()

		// Check if this is an annotation node
		if nodeType == "marker_annotation" || nodeType == "annotation" ||
		   strings.Contains(nodeType, "annotation") {
			// Prepend to maintain order (we're walking backwards)
			annotations = append([]string{p.getNodeText(current)}, annotations...)
		} else if nodeType == "modifiers" {
			// Modifiers node might contain annotations as children
			// Check children of modifiers for annotations
			for i := 0; i < int(current.ChildCount()); i++ {
				child := current.Child(i)
				childType := child.Type()
				if childType == "marker_annotation" || childType == "annotation" ||
				   strings.Contains(childType, "annotation") {
					annotations = append([]string{p.getNodeText(child)}, annotations...)
				}
			}
		} else if nodeType != "block_comment" && nodeType != "line_comment" {
			// Stop when we hit something that's not an annotation, comment, or modifiers
			break
		}

		current = current.PrevSibling()
	}

	return annotations
}

func (p *JavaCodeParser) extractTypeList(node *sitter.Node) []string {
	var types []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_identifier" || strings.Contains(child.Type(), "type") {
			types = append(types, p.getNodeText(child))
		}
	}

	return types
}

func (p *JavaCodeParser) extractParameterTypes(paramsNode *sitter.Node) []string {
	var types []string

	for i := 0; i < int(paramsNode.ChildCount()); i++ {
		child := paramsNode.Child(i)
		if child.Type() == "formal_parameter" {
			typeNode := p.findChildByType(child, "type_identifier")
			if typeNode == nil {
				// Try other type nodes
				for j := 0; j < int(child.ChildCount()); j++ {
					typeChild := child.Child(j)
					if strings.Contains(typeChild.Type(), "type") {
						typeNode = typeChild
						break
					}
				}
			}
			if typeNode != nil {
				types = append(types, p.getNodeText(typeNode))
			}
		}
	}

	return types
}

func (p *JavaCodeParser) determineVisibility(modifiers []string) string {
	for _, mod := range modifiers {
		switch mod {
		case "public":
			return "public"
		case "private":
			return "private"
		case "protected":
			return "protected"
		}
	}
	return "package" // default visibility in Java
}

func (p *JavaCodeParser) buildMethodSignature(modifiers []string, returnType, methodName string, paramTypes []string) string {
	var parts []string

	if len(modifiers) > 0 {
		parts = append(parts, strings.Join(modifiers, " "))
	}

	if returnType != "" {
		parts = append(parts, returnType)
	}

	methodSig := methodName + "(" + strings.Join(paramTypes, ", ") + ")"
	parts = append(parts, methodSig)

	return strings.Join(parts, " ")
}

func (p *JavaCodeParser) getCurrentClassName() string {
	if len(p.classStack) > 0 {
		return p.classStack[len(p.classStack)-1]
	}
	return ""
}

func (p *JavaCodeParser) getNodeText(node *sitter.Node) string {
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

func (p *JavaCodeParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func (p *JavaCodeParser) walkNode(node *sitter.Node, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNode(node.Child(i), fn)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
