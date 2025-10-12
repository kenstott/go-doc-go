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

// AnthropicClient implements LLMClient for Anthropic's Claude API
type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	apiURL     string
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
