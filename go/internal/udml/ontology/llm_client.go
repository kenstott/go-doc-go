package ontology

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicClient implements LLMClient and MCPCapableLLMClient for Anthropic's Claude API
type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	apiURL     string
	mcpServer  MCPToolExecutor // Optional: MCP server for tool calling
}

// MCPToolExecutor is the interface for executing MCP tools
// This allows the LLM client to execute tool calls without direct coupling to mcp package
type MCPToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error)
}

// NewAnthropicClient creates a new Anthropic API client
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Latest Sonnet model
	}

	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiURL: "https://api.anthropic.com/v1/messages",
	}
}

// SetMCPServer sets the MCP server for tool calling support
func (c *AnthropicClient) SetMCPServer(server MCPToolExecutor) {
	c.mcpServer = server
}

// Complete sends a prompt to Claude and returns the response
func (c *AnthropicClient) Complete(ctx context.Context, prompt string, options LLMOptions) (string, error) {
	// Build request payload
	messages := []anthropicMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	payload := anthropicRequest{
		Model:       c.model,
		MaxTokens:   options.MaxTokens,
		Messages:    messages,
		Temperature: options.Temperature,
	}

	if options.SystemPrompt != "" {
		payload.System = options.SystemPrompt
	}

	// Serialize request
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Content[0].Text, nil
}

// GetProvider returns the provider name
func (c *AnthropicClient) GetProvider() string {
	return "anthropic"
}

// GetModel returns the model name
func (c *AnthropicClient) GetModel() string {
	return c.model
}

// CompleteWithTools sends a prompt to Claude with MCP tools and handles tool calling loop
func (c *AnthropicClient) CompleteWithTools(ctx context.Context, prompt string, tools []MCPToolDefinition, options LLMOptions) (string, error) {
	if c.mcpServer == nil {
		return "", fmt.Errorf("MCP server not configured - call SetMCPServer() first")
	}

	// Convert MCPToolDefinition to Anthropic format
	anthropicTools := c.convertToAnthropicTools(tools)

	// Build initial message
	messages := []anthropicMessageWithContent{
		{
			Role: "user",
			Content: []anthropicContentBlock{
				{Type: "text", Text: prompt},
			},
		},
	}

	// Tool calling loop
	maxIterations := 10 // Prevent infinite loops
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Build request with tools
		payload := anthropicRequestWithTools{
			Model:       c.model,
			MaxTokens:   options.MaxTokens,
			Messages:    messages,
			Temperature: options.Temperature,
			Tools:       anthropicTools,
		}

		if options.SystemPrompt != "" {
			payload.System = options.SystemPrompt
		}

		// Send request
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("API request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		// Parse response
		var apiResp anthropicResponseWithTools
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		// Check stop reason
		if apiResp.StopReason == "end_turn" || apiResp.StopReason == "max_tokens" {
			// LLM finished - extract text response
			return c.extractTextFromResponse(apiResp), nil
		}

		if apiResp.StopReason == "tool_use" {
			// LLM wants to call tools - execute them
			toolResults, err := c.executeToolCalls(ctx, apiResp.Content)
			if err != nil {
				return "", fmt.Errorf("tool execution failed: %w", err)
			}

			// Add assistant message with tool calls
			messages = append(messages, anthropicMessageWithContent{
				Role:    "assistant",
				Content: apiResp.Content,
			})

			// Add user message with tool results
			messages = append(messages, anthropicMessageWithContent{
				Role:    "user",
				Content: toolResults,
			})

			// Continue loop to get next response
			continue
		}

		// Unknown stop reason
		return "", fmt.Errorf("unexpected stop reason: %s", apiResp.StopReason)
	}

	return "", fmt.Errorf("exceeded maximum tool calling iterations (%d)", maxIterations)
}

// convertToAnthropicTools converts MCPToolDefinition to Anthropic's tool format
func (c *AnthropicClient) convertToAnthropicTools(tools []MCPToolDefinition) []anthropicTool {
	result := make([]anthropicTool, len(tools))
	for i, tool := range tools {
		// Build input schema
		properties := make(map[string]interface{})
		required := []string{}

		for paramName, paramDef := range tool.Parameters {
			paramSchema := map[string]interface{}{
				"type":        paramDef.Type,
				"description": paramDef.Description,
			}

			if len(paramDef.Enum) > 0 {
				paramSchema["enum"] = paramDef.Enum
			}

			if paramDef.Items != nil {
				paramSchema["items"] = map[string]interface{}{
					"type": paramDef.Items.Type,
				}
			}

			properties[paramName] = paramSchema

			if paramDef.Required {
				required = append(required, paramName)
			}
		}

		inputSchema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			inputSchema["required"] = required
		}

		result[i] = anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: inputSchema,
		}
	}
	return result
}

// executeToolCalls executes all tool_use blocks in the response content
func (c *AnthropicClient) executeToolCalls(ctx context.Context, content []anthropicContentBlock) ([]anthropicContentBlock, error) {
	results := []anthropicContentBlock{}

	for _, block := range content {
		if block.Type == "tool_use" {
			// Execute the tool via MCP server
			result, err := c.mcpServer.ExecuteTool(ctx, block.Name, block.Input)
			if err != nil {
				// Return error as tool result
				results = append(results, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   fmt.Sprintf("Error: %v", err),
					IsError:   true,
				})
			} else {
				// Return successful result
				results = append(results, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   result,
				})
			}
		}
	}

	return results, nil
}

// extractTextFromResponse extracts text content from response
func (c *AnthropicClient) extractTextFromResponse(resp anthropicResponseWithTools) string {
	var texts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			texts = append(texts, block.Text)
		}
	}
	if len(texts) > 0 {
		return texts[0] // Return first text block
	}
	return ""
}

// Anthropic API types

type anthropicRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string              `json:"id"`
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []anthropicContent  `json:"content"`
	Model   string              `json:"model"`
	Usage   anthropicUsage      `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Tool calling API types

type anthropicRequestWithTools struct {
	Model       string                        `json:"model"`
	MaxTokens   int                           `json:"max_tokens"`
	Messages    []anthropicMessageWithContent `json:"messages"`
	System      string                        `json:"system,omitempty"`
	Temperature float64                       `json:"temperature,omitempty"`
	Tools       []anthropicTool               `json:"tools,omitempty"`
}

type anthropicMessageWithContent struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string                 `json:"type"`                 // "text", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`       // For text blocks
	ID        string                 `json:"id,omitempty"`         // For tool_use blocks
	Name      string                 `json:"name,omitempty"`       // For tool_use blocks
	Input     map[string]interface{} `json:"input,omitempty"`      // For tool_use blocks
	ToolUseID string                 `json:"tool_use_id,omitempty"` // For tool_result blocks
	Content   string                 `json:"content,omitempty"`    // For tool_result blocks
	IsError   bool                   `json:"is_error,omitempty"`   // For tool_result blocks
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponseWithTools struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}
