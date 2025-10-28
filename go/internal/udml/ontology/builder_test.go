package ontology

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
)

// MockLLMClient implements LLMClient for testing
type MockLLMClient struct {
	responses []string
	callCount int
}

func NewMockLLMClient(responses ...string) *MockLLMClient {
	return &MockLLMClient{
		responses: responses,
		callCount: 0,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string, options LLMOptions) (string, error) {
	if m.callCount >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}

	response := m.responses[m.callCount]
	m.callCount++
	return response, nil
}

func (m *MockLLMClient) GetProvider() string {
	return "mock"
}

func (m *MockLLMClient) GetModel() string {
	return "mock-model"
}

func TestBuilderConfig_Defaults(t *testing.T) {
	config := BuilderConfig{
		LLMProvider: "anthropic",
		LLMAPIKey:   "test-key",
	}

	// These would be set in NewOntologyBuilder
	if config.TopEntityCount == 0 {
		t.Log("TopEntityCount defaults to 0 before NewOntologyBuilder")
	}
}

func TestAnthropicClient_Creation(t *testing.T) {
	client := NewAnthropicClient("test-key", "")

	if client.GetProvider() != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", client.GetProvider())
	}

	if client.model == "" {
		t.Error("Expected default model to be set")
	}

	t.Logf("Default model: %s", client.GetModel())
}

func TestExtractJSON(t *testing.T) {
	builder := &OntologyBuilder{}

	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "plain JSON",
			response: `{"domain": "financial", "key_concepts": ["revenue", "profit"]}`,
			wantErr:  false,
		},
		{
			name: "JSON in code block",
			response: "```json\n" +
				`{"domain": "technical", "key_concepts": ["API", "database"]}` +
				"\n```",
			wantErr: false,
		},
		{
			name: "JSON in generic code block",
			response: "```\n" +
				`{"domain": "medical", "key_concepts": ["diagnosis", "treatment"]}` +
				"\n```",
			wantErr: false,
		},
		{
			name:     "invalid JSON",
			response: `{invalid json}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Domain      string   `json:"domain"`
				KeyConcepts []string `json:"key_concepts"`
			}

			err := builder.extractJSON(tt.response, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.Domain == "" {
					t.Error("Expected domain to be extracted")
				}
				t.Logf("Extracted: domain=%s, concepts=%v", result.Domain, result.KeyConcepts)
			}
		})
	}
}

func TestPrepareSampleTexts(t *testing.T) {
	builder := &OntologyBuilder{}

	samples := []sampler.Sample{
		{
			ElementType: "paragraph",
			Content:     "This is the first sample.",
		},
		{
			ElementType: "heading",
			Content:     "This is the second sample.",
		},
		{
			ElementType: "paragraph",
			Content:     "This is the third sample.",
		},
	}

	result := builder.prepareSampleTexts(samples, 2)

	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Should only include first 2 samples
	if !strings.Contains(result, "first sample") {
		t.Error("Expected first sample to be included")
	}

	if !strings.Contains(result, "second sample") {
		t.Error("Expected second sample to be included")
	}

	if strings.Contains(result, "third sample") {
		t.Error("Expected third sample to be excluded (maxSamples=2)")
	}

	t.Logf("Prepared text:\n%s", result)
}

func TestBuildResult_Log(t *testing.T) {
	builder := &OntologyBuilder{}
	result := &BuildResult{
		AnalysisLog: []string{},
	}

	builder.log(result, "Test message 1")
	builder.log(result, "Test message 2")

	if len(result.AnalysisLog) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(result.AnalysisLog))
	}

	for _, entry := range result.AnalysisLog {
		if !strings.Contains(entry, "Test message") {
			t.Errorf("Log entry doesn't contain expected message: %s", entry)
		}
	}

	t.Logf("Log entries: %v", result.AnalysisLog)
}

func TestElementEntityMapping_Structure(t *testing.T) {
	mapping := ElementEntityMapping{
		EntityType:   "organization",
		Domain:       "test",
		Confidence:   0.9,
		Description:  "Companies and organizations",
		ElementTypes: []string{"paragraph"},
		ExtractionRules: []ExtractionRule{
			{
				PhraseList: []string{"Microsoft", "MSFT"},
			},
		},
	}

	if mapping.EntityType != "organization" {
		t.Errorf("Expected entity_type 'organization', got '%s'", mapping.EntityType)
	}

	if mapping.Domain != "test" {
		t.Errorf("Expected domain 'test', got '%s'", mapping.Domain)
	}

	if mapping.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9 (context quality), got %.2f", mapping.Confidence)
	}

	if len(mapping.ExtractionRules) != 1 {
		t.Errorf("Expected 1 extraction rule, got %d", len(mapping.ExtractionRules))
	}

	rule := mapping.ExtractionRules[0]
	if len(rule.PhraseList) != 2 {
		t.Errorf("Expected 2 phrases, got %d", len(rule.PhraseList))
	}
}

func TestEntityRelationshipRule_Structure(t *testing.T) {
	rule := EntityRelationshipRule{
		Name:             "person_works_at_org",
		SourceEntityType: "person",
		TargetEntityType: "organization",
		RelationshipType: RelationshipPartOf,
		Description:      "Person employed by organization",
		Confidence:       0.7,
	}

	if rule.Name != "person_works_at_org" {
		t.Errorf("Expected name 'person_works_at_org', got '%s'", rule.Name)
	}

	if rule.RelationshipType != RelationshipPartOf {
		t.Errorf("Expected relationship type 'part_of', got '%s'", rule.RelationshipType)
	}

	if rule.Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7 (pattern reliability), got %.2f", rule.Confidence)
	}
}

// MockMCPToolExecutor implements MCPToolExecutor interface for testing
type MockMCPToolExecutor struct {
	toolCalls []string // Track which tools were called
}

func (m *MockMCPToolExecutor) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	m.toolCalls = append(m.toolCalls, toolName)
	return fmt.Sprintf(`{"results": ["mock result for %s"]}`, toolName), nil
}

// TestGetMCPToolDefinitions_Disabled tests that getMCPToolDefinitions returns nil when MCP is disabled
func TestGetMCPToolDefinitions_Disabled(t *testing.T) {
	builder := &OntologyBuilder{
		config: BuilderConfig{
			EnableMCP: false,
		},
	}

	tools := builder.getMCPToolDefinitions()

	if tools != nil {
		t.Errorf("Expected nil tools when MCP disabled, got %d tools", len(tools))
	}
}

// TestGetMCPToolDefinitions_NoServer tests that getMCPToolDefinitions returns nil when server is nil
func TestGetMCPToolDefinitions_NoServer(t *testing.T) {
	builder := &OntologyBuilder{
		config: BuilderConfig{
			EnableMCP: true,
		},
		mcpServer: nil,
	}

	tools := builder.getMCPToolDefinitions()

	if tools != nil {
		t.Errorf("Expected nil tools when MCP server is nil, got %d tools", len(tools))
	}
}

// TestAnthropicClient_SetMCPServer tests MCP server attachment
func TestAnthropicClient_SetMCPServer(t *testing.T) {
	client := NewAnthropicClient("test-key", "test-model")

	if client.mcpServer != nil {
		t.Error("Expected mcpServer to be nil initially")
	}

	mockServer := &MockMCPToolExecutor{}
	client.SetMCPServer(mockServer)

	if client.mcpServer == nil {
		t.Error("Expected mcpServer to be set")
	}

	t.Log("MCP server successfully attached to Anthropic client")
}

// TestMCPCapableLLMClient_Interface tests that AnthropicClient implements MCPCapableLLMClient
func TestMCPCapableLLMClient_Interface(t *testing.T) {
	client := NewAnthropicClient("test-key", "test-model")

	// Type assert to MCPCapableLLMClient
	_, ok := interface{}(client).(MCPCapableLLMClient)
	if !ok {
		t.Error("AnthropicClient does not implement MCPCapableLLMClient interface")
	}

	// Type assert to LLMClient (base interface)
	_, ok = interface{}(client).(LLMClient)
	if !ok {
		t.Error("AnthropicClient does not implement LLMClient interface")
	}

	t.Log("AnthropicClient correctly implements MCPCapableLLMClient interface")
}

// TestMCPToolDefinitionTypes tests MCPToolDefinition type structures
func TestMCPToolDefinitionTypes(t *testing.T) {
	// Test MCPToolDefinition structure
	tool := MCPToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]ParameterDef{
			"param1": {
				Type:        "string",
				Description: "First parameter",
				Required:    true,
			},
			"param2": {
				Type:        "array",
				Description: "Second parameter",
				Required:    false,
				Enum:        []string{"option1", "option2"},
				Items: &ItemDef{
					Type: "string",
				},
			},
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Name)
	}

	if len(tool.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(tool.Parameters))
	}

	param1 := tool.Parameters["param1"]
	if !param1.Required {
		t.Error("Expected param1 to be required")
	}

	param2 := tool.Parameters["param2"]
	if param2.Items == nil {
		t.Error("Expected param2 to have Items")
	}
	if len(param2.Enum) != 2 {
		t.Errorf("Expected 2 enum values, got %d", len(param2.Enum))
	}

	t.Log("MCPToolDefinition type structure validated")
}
