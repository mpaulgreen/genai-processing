package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"genai-processing/pkg/types"
)

// ClaudeProvider implements the LLMProvider interface for Anthropic's Claude API
type ClaudeProvider struct {
	APIKey   string
	Endpoint string
	client   *http.Client
}

// ClaudeMessage represents a message in the Claude API format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest represents the request payload for Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	System      string          `json:"system,omitempty"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeError represents an error response from Claude API
type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewClaudeProvider creates a new ClaudeProvider instance
func NewClaudeProvider(apiKey, endpoint string) *ClaudeProvider {
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}

	return &ClaudeProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateResponse implements the LLMProvider interface
func (c *ClaudeProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	// TODO: Enhance authentication with proper API key validation and rotation
	if c.APIKey == "" {
		return nil, fmt.Errorf("claude API key is required")
	}

	// Convert ModelRequest to ClaudeRequest
	claudeReq := ClaudeRequest{
		Model:       request.Model,
		MaxTokens:   4000, // Default max tokens
		Temperature: 0.1,  // Default temperature
	}

	// Extract parameters
	if request.Parameters != nil {
		if maxTokens, ok := request.Parameters["max_tokens"].(int); ok {
			claudeReq.MaxTokens = maxTokens
		}
		if temp, ok := request.Parameters["temperature"].(float64); ok {
			claudeReq.Temperature = temp
		}
		if system, ok := request.Parameters["system"].(string); ok {
			claudeReq.System = system
		}
	}

	// Convert messages to Claude format
	for _, msg := range request.Messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			role, _ := msgMap["role"].(string)
			content, _ := msgMap["content"].(string)
			claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	// Prepare the HTTP request
	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Claude request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make the request
	startTime := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	processingTime := time.Since(startTime)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var claudeErr ClaudeError
		if err := json.Unmarshal(body, &claudeErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: failed to parse error response: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("claude API error: %s - %s", claudeErr.Type, claudeErr.Message)
	}

	// Parse successful response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Extract content from response
	var content string
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	// Calculate token usage
	totalTokens := claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens
	tokensPerSecond := 0.0
	if processingTime > 0 {
		tokensPerSecond = float64(totalTokens) / processingTime.Seconds()
	}

	// TODO: Implement cost calculation based on Claude pricing
	// estimatedCost := calculateClaudeCost(totalTokens, claudeResp.Model)

	return &types.RawResponse{
		Content: content,
		ModelInfo: map[string]interface{}{
			"model":       claudeResp.Model,
			"id":          claudeResp.ID,
			"stop_reason": claudeResp.StopReason,
			"type":        claudeResp.Type,
		},
		Metadata: map[string]interface{}{
			"provider":        "anthropic",
			"api_version":     "2023-06-01",
			"processing_time": processingTime.String(),
			"token_usage": map[string]interface{}{
				"prompt_tokens":     claudeResp.Usage.InputTokens,
				"completion_tokens": claudeResp.Usage.OutputTokens,
				"total_tokens":      totalTokens,
				"tokens_per_second": tokensPerSecond,
				"model_name":        claudeResp.Model,
				"timestamp":         time.Now(),
			},
		},
	}, nil
}

// GetModelInfo implements the LLMProvider interface
func (c *ClaudeProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:               "claude-3-5-sonnet-20241022",
		Provider:           "anthropic",
		Version:            "20241022",
		Description:        "Claude 3.5 Sonnet - Advanced language model for complex reasoning and analysis",
		ModelType:          "chat",
		ContextWindow:      200000,
		MaxOutputTokens:    4096,
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ja", "ko", "zh"},
		PricingInfo: map[string]interface{}{
			"input_cost_per_1k_tokens":  0.003,
			"output_cost_per_1k_tokens": 0.015,
			"currency":                  "USD",
		},
		PerformanceMetrics: map[string]interface{}{
			"reasoning_capability": "high",
			"coding_capability":    "excellent",
			"analysis_capability":  "excellent",
		},
	}
}

// SupportsStreaming implements the LLMProvider interface
func (c *ClaudeProvider) SupportsStreaming() bool {
	// TODO: Implement streaming support for Claude API
	return false
}

// ValidateConnection checks if the Claude API connection is working
func (c *ClaudeProvider) ValidateConnection() error {
	// TODO: Implement connection validation with a simple test request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := ClaudeRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		MaxTokens: 10,
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}

	return nil
}
