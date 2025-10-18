package parser

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// DockerfileParser parses Dockerfile using regex patterns for Dockerfile instructions
type DockerfileParser struct {
	MaxContentPreview int
	ExtractComments   bool
}

// NewDockerfileParser creates a new Dockerfile parser
func NewDockerfileParser() *DockerfileParser {
	return &DockerfileParser{
		MaxContentPreview: 100,
		ExtractComments:   true,
	}
}

// GetName returns the parser name
func (p *DockerfileParser) GetName() string {
	return "dockerfile_code"
}

// GetSupportedFormats returns supported file formats
func (p *DockerfileParser) GetSupportedFormats() []string {
	return []string{"Dockerfile", ".dockerfile", "dockerfile"}
}

// SupportsStreaming returns false as this parser needs complete content
func (p *DockerfileParser) SupportsStreaming() bool {
	return false
}

// Close performs any cleanup needed
func (p *DockerfileParser) Close() error {
	return nil
}

// Parse parses Dockerfile and extracts container definition elements
func (p *DockerfileParser) Parse(ctx context.Context, req ParseRequest) (*ParseResult, error) {
	// Extract file path from content
	var filePath string
	switch v := req.Content.(type) {
	case string:
		filePath = v
	default:
		return nil, fmt.Errorf("unsupported content type for Dockerfile parser: %T (expected file path string)", req.Content)
	}

	// Read file content
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(sourceCode)

	// Create document element
	docElement := Element{
		ElementID:       generateID("dockerfile_doc"),
		ElementType:     "document",
		ContentPreview:  truncate(content, p.MaxContentPreview),
		ElementCategory: "document",
		Metadata: map[string]interface{}{
			"language": "dockerfile",
			"parser":   "dockerfile_code",
		},
	}

	// Extract all elements
	elements := []Element{docElement}

	// Extract comments
	if p.ExtractComments {
		commentElements := p.extractComments(content, docElement.ElementID)
		elements = append(elements, commentElements...)
	}

	// Extract FROM instructions (base images)
	fromElements := p.extractFromInstructions(content, docElement.ElementID)
	elements = append(elements, fromElements...)

	// Extract RUN instructions
	runElements := p.extractRunInstructions(content, docElement.ElementID)
	elements = append(elements, runElements...)

	// Extract COPY instructions
	copyElements := p.extractCopyInstructions(content, docElement.ElementID)
	elements = append(elements, copyElements...)

	// Extract ADD instructions
	addElements := p.extractAddInstructions(content, docElement.ElementID)
	elements = append(elements, addElements...)

	// Extract ENV instructions
	envElements := p.extractEnvInstructions(content, docElement.ElementID)
	elements = append(elements, envElements...)

	// Extract WORKDIR instructions
	workdirElements := p.extractWorkdirInstructions(content, docElement.ElementID)
	elements = append(elements, workdirElements...)

	// Extract EXPOSE instructions
	exposeElements := p.extractExposeInstructions(content, docElement.ElementID)
	elements = append(elements, exposeElements...)

	// Extract CMD instructions
	cmdElements := p.extractCmdInstructions(content, docElement.ElementID)
	elements = append(elements, cmdElements...)

	// Extract ENTRYPOINT instructions
	entrypointElements := p.extractEntrypointInstructions(content, docElement.ElementID)
	elements = append(elements, entrypointElements...)

	// Extract ARG instructions
	argElements := p.extractArgInstructions(content, docElement.ElementID)
	elements = append(elements, argElements...)

	// Extract LABEL instructions
	labelElements := p.extractLabelInstructions(content, docElement.ElementID)
	elements = append(elements, labelElements...)

	// Create document
	doc := Document{
		ID:      req.ID,
		DocType: "dockerfile_source",
		Metadata: map[string]interface{}{
			"language": "dockerfile",
			"parser":   "dockerfile_code",
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

// extractComments extracts Dockerfile comments
func (p *DockerfileParser) extractComments(content string, parentID string) []Element {
	var elements []Element

	// Comments start with # (but not in the middle of lines)
	commentRegex := regexp.MustCompile(`(?m)^\s*#\s*(.+)`)
	matches := commentRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			commentText := strings.TrimSpace(match[1])
			elements = append(elements, Element{
				ElementID:       generateID("dockerfile_comment"),
				ElementType:     "code_documentation",
				ContentPreview:  truncate(commentText, p.MaxContentPreview),
				ElementCategory: "documentation",
				ParentID:        parentID,
				Metadata: map[string]interface{}{
					"doc_kind": "comment",
					"language": "dockerfile",
				},
			})
		}
	}

	return elements
}

// extractFromInstructions extracts FROM instructions (base images)
func (p *DockerfileParser) extractFromInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match FROM instructions with optional platform and AS alias
	fromRegex := regexp.MustCompile(`(?mi)^\s*FROM\s+(?:--platform=(\S+)\s+)?([^\s]+)(?:\s+AS\s+([^\s]+))?`)
	matches := fromRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		platform := ""
		if match[1] != "" {
			platform = match[1]
		}

		baseImage := match[2]
		alias := ""
		if len(match) > 3 && match[3] != "" {
			alias = match[3]
		}

		preview := fmt.Sprintf("FROM %s", baseImage)
		if alias != "" {
			preview = fmt.Sprintf("FROM %s AS %s", baseImage, alias)
		}

		metadata := map[string]interface{}{
			"dependency_kind": "base_image",
			"base_image":      baseImage,
			"language":        "dockerfile",
		}

		if platform != "" {
			metadata["platform"] = platform
		}
		if alias != "" {
			metadata["stage_alias"] = alias
		}

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_from"),
			ElementType:     "code_dependency",
			ContentPreview:  truncate(preview, p.MaxContentPreview),
			ElementCategory: "dependency",
			ParentID:        parentID,
			Metadata:        metadata,
		})
	}

	return elements
}

// extractRunInstructions extracts RUN instructions
func (p *DockerfileParser) extractRunInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match RUN instructions - simpler approach without negative lookahead
	runRegex := regexp.MustCompile(`(?mi)^\s*RUN\s+(.+?)$`)
	matches := runRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		command := strings.TrimSpace(match[1])
		// Clean up line continuations
		command = strings.ReplaceAll(command, "\\\n", " ")
		command = strings.ReplaceAll(command, "\\", " ")
		command = strings.TrimSpace(command)

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_run"),
			ElementType:     "code_function",
			ContentPreview:  truncate(fmt.Sprintf("RUN %s", command), p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"code_element_kind": "run_command",
				"command":           command,
				"language":          "dockerfile",
			},
		})
	}

	return elements
}

// extractCopyInstructions extracts COPY instructions
func (p *DockerfileParser) extractCopyInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match COPY instructions with optional --from flag
	copyRegex := regexp.MustCompile(`(?mi)^\s*COPY\s+(?:--from=([^\s]+)\s+)?(.+?)\s+(.+)`)
	matches := copyRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		fromStage := ""
		if match[1] != "" {
			fromStage = match[1]
		}

		source := strings.TrimSpace(match[2])
		dest := strings.TrimSpace(match[3])

		preview := fmt.Sprintf("COPY %s %s", source, dest)
		if fromStage != "" {
			preview = fmt.Sprintf("COPY --from=%s %s %s", fromStage, source, dest)
		}

		metadata := map[string]interface{}{
			"code_element_kind": "copy_instruction",
			"source":            source,
			"destination":       dest,
			"language":          "dockerfile",
		}

		if fromStage != "" {
			metadata["from_stage"] = fromStage
		}

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_copy"),
			ElementType:     "code_function",
			ContentPreview:  truncate(preview, p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			Metadata:        metadata,
		})
	}

	return elements
}

// extractAddInstructions extracts ADD instructions
func (p *DockerfileParser) extractAddInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match ADD instructions
	addRegex := regexp.MustCompile(`(?mi)^\s*ADD\s+(.+?)\s+(.+)`)
	matches := addRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		source := strings.TrimSpace(match[1])
		dest := strings.TrimSpace(match[2])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_add"),
			ElementType:     "code_function",
			ContentPreview:  truncate(fmt.Sprintf("ADD %s %s", source, dest), p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"code_element_kind": "add_instruction",
				"source":            source,
				"destination":       dest,
				"language":          "dockerfile",
			},
		})
	}

	return elements
}

// extractEnvInstructions extracts ENV instructions
func (p *DockerfileParser) extractEnvInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match ENV instructions (key=value or key value format)
	envRegex := regexp.MustCompile(`(?mi)^\s*ENV\s+([^\s=]+)(?:=|\s+)(.+)`)
	matches := envRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		varName := strings.TrimSpace(match[1])
		varValue := strings.TrimSpace(match[2])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_env"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("ENV %s=%s", varName, varValue), p.MaxContentPreview),
			ElementCategory: "data_field",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"data_kind": "environment_variable",
				"data_name": varName,
				"value":     varValue,
				"language":  "dockerfile",
			},
		})
	}

	return elements
}

// extractWorkdirInstructions extracts WORKDIR instructions
func (p *DockerfileParser) extractWorkdirInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match WORKDIR instructions
	workdirRegex := regexp.MustCompile(`(?mi)^\s*WORKDIR\s+(.+)`)
	matches := workdirRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		path := strings.TrimSpace(match[1])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_workdir"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("WORKDIR %s", path), p.MaxContentPreview),
			ElementCategory: "configuration",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"data_kind": "working_directory",
				"path":      path,
				"language":  "dockerfile",
			},
		})
	}

	return elements
}

// extractExposeInstructions extracts EXPOSE instructions
func (p *DockerfileParser) extractExposeInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match EXPOSE instructions
	exposeRegex := regexp.MustCompile(`(?mi)^\s*EXPOSE\s+(.+)`)
	matches := exposeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		ports := strings.TrimSpace(match[1])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_expose"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("EXPOSE %s", ports), p.MaxContentPreview),
			ElementCategory: "configuration",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"data_kind": "port_exposure",
				"ports":     ports,
				"language":  "dockerfile",
			},
		})
	}

	return elements
}

// extractCmdInstructions extracts CMD instructions
func (p *DockerfileParser) extractCmdInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match CMD instructions (JSON array or shell form)
	cmdRegex := regexp.MustCompile(`(?mi)^\s*CMD\s+(.+)`)
	matches := cmdRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		command := strings.TrimSpace(match[1])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_cmd"),
			ElementType:     "code_function",
			ContentPreview:  truncate(fmt.Sprintf("CMD %s", command), p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"code_element_kind": "container_command",
				"command":           command,
				"language":          "dockerfile",
			},
		})
	}

	return elements
}

// extractEntrypointInstructions extracts ENTRYPOINT instructions
func (p *DockerfileParser) extractEntrypointInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match ENTRYPOINT instructions (JSON array or shell form)
	entrypointRegex := regexp.MustCompile(`(?mi)^\s*ENTRYPOINT\s+(.+)`)
	matches := entrypointRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		command := strings.TrimSpace(match[1])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_entrypoint"),
			ElementType:     "code_function",
			ContentPreview:  truncate(fmt.Sprintf("ENTRYPOINT %s", command), p.MaxContentPreview),
			ElementCategory: "code_logic",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"code_element_kind": "container_entrypoint",
				"command":           command,
				"language":          "dockerfile",
			},
		})
	}

	return elements
}

// extractArgInstructions extracts ARG instructions
func (p *DockerfileParser) extractArgInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match ARG instructions
	argRegex := regexp.MustCompile(`(?mi)^\s*ARG\s+([^\s=]+)(?:=(.+))?`)
	matches := argRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		argName := strings.TrimSpace(match[1])
		defaultValue := ""
		if len(match) > 2 && match[2] != "" {
			defaultValue = strings.TrimSpace(match[2])
		}

		preview := fmt.Sprintf("ARG %s", argName)
		if defaultValue != "" {
			preview = fmt.Sprintf("ARG %s=%s", argName, defaultValue)
		}

		metadata := map[string]interface{}{
			"data_kind": "build_argument",
			"data_name": argName,
			"language":  "dockerfile",
		}

		if defaultValue != "" {
			metadata["default_value"] = defaultValue
		}

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_arg"),
			ElementType:     "code_data",
			ContentPreview:  truncate(preview, p.MaxContentPreview),
			ElementCategory: "data_field",
			ParentID:        parentID,
			Metadata:        metadata,
		})
	}

	return elements
}

// extractLabelInstructions extracts LABEL instructions
func (p *DockerfileParser) extractLabelInstructions(content string, parentID string) []Element {
	var elements []Element

	// Match LABEL instructions
	labelRegex := regexp.MustCompile(`(?mi)^\s*LABEL\s+(.+)`)
	matches := labelRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		labelDef := strings.TrimSpace(match[1])

		elements = append(elements, Element{
			ElementID:       generateID("dockerfile_label"),
			ElementType:     "code_data",
			ContentPreview:  truncate(fmt.Sprintf("LABEL %s", labelDef), p.MaxContentPreview),
			ElementCategory: "metadata",
			ParentID:        parentID,
			Metadata: map[string]interface{}{
				"data_kind": "label",
				"label_def": labelDef,
				"language":  "dockerfile",
			},
		})
	}

	return elements
}
