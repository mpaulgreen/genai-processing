package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// GenericProvider implements a simple OpenAI-compatible Chat Completions client
// with configurable endpoint and headers.
type GenericProvider struct {
	APIKey     string
	Endpoint   string
	ModelName  string
	Parameters map[string]interface{}
	Headers    map[string]string
	client     *http.Client
}

// GenericChatMessage minimal role/content message
type GenericChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenericChatRequest is compatible with OpenAI-style chat endpoints
type GenericChatRequest struct {
	Model       string               `json:"model"`
	Messages    []GenericChatMessage `json:"messages"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	Temperature float64              `json:"temperature,omitempty"`
	TopP        float64              `json:"top_p,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
}

// GenericChatResponse models a minimal OpenAI-like response
type GenericChatResponse struct {
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GenericAPIError minimal error envelope
type GenericAPIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// NewGenericProvider creates a generic OpenAI-compatible provider
func NewGenericProvider(apiKey, endpoint, modelName string, headers map[string]string, parameters map[string]interface{}) *GenericProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if modelName == "" {
		modelName = "generic-model"
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if parameters == nil {
		parameters = map[string]interface{}{}
	}

	// Default auth/content-type if not provided
	if _, ok := headers["Authorization"]; !ok && apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	return &GenericProvider{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		ModelName:  modelName,
		Parameters: parameters,
		Headers:    headers,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateResponse sends a request to the configured endpoint and returns a RawResponse
func (g *GenericProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	if g.Endpoint == "" {
		return nil, fmt.Errorf("generic provider endpoint is required")
	}

	// Determine model
	modelName := request.Model
	if modelName == "" {
		modelName = g.ModelName
	}

	// Build request body with defaults then override from request.Parameters
	chatReq := GenericChatRequest{
		Model:       modelName,
		MaxTokens:   4000,
		Temperature: 0.1,
		TopP:        1.0,
		Stream:      false,
	}

	if g.Parameters != nil {
		if v, ok := g.Parameters["max_tokens"].(int); ok {
			chatReq.MaxTokens = v
		}
		if v, ok := g.Parameters["temperature"].(float64); ok {
			chatReq.Temperature = v
		}
		if v, ok := g.Parameters["top_p"].(float64); ok {
			chatReq.TopP = v
		}
		if v, ok := g.Parameters["stream"].(bool); ok {
			chatReq.Stream = v
		}
	}

	if request.Parameters != nil {
		if v, ok := request.Parameters["max_tokens"].(int); ok {
			chatReq.MaxTokens = v
		}
		if v, ok := request.Parameters["temperature"].(float64); ok {
			chatReq.Temperature = v
		}
		if v, ok := request.Parameters["top_p"].(float64); ok {
			chatReq.TopP = v
		}
		if v, ok := request.Parameters["stream"].(bool); ok {
			chatReq.Stream = v
		}
	}

	// Convert generic messages
	for _, msg := range request.Messages {
		if m, ok := msg.(map[string]interface{}); ok {
			role, _ := m["role"].(string)
			content, _ := m["content"].(string)
			chatReq.Messages = append(chatReq.Messages, GenericChatMessage{Role: role, Content: content})
		}
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range g.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error envelope
		var apiErr GenericAPIError
		if err := json.Unmarshal(respBytes, &apiErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
		}
		return nil, fmt.Errorf("generic API error: %s - %s", apiErr.Error.Type, apiErr.Error.Message)
	}

	var chatResp GenericChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse generic chat response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(chatResp.Choices) > 0 {
		content = chatResp.Choices[0].Message.Content
		finishReason = chatResp.Choices[0].FinishReason
	}

	processingTime := time.Since(start)

	totalTokens := chatResp.Usage.TotalTokens
	tokensPerSecond := 0.0
	if processingTime > 0 {
		tokensPerSecond = float64(totalTokens) / processingTime.Seconds()
	}

	return &types.RawResponse{
		Content: content,
		ModelInfo: map[string]interface{}{
			"model":         chatResp.Model,
			"finish_reason": finishReason,
		},
		Metadata: map[string]interface{}{
			"provider":        "generic",
			"api_version":     "v1",
			"processing_time": processingTime.String(),
			"token_usage": map[string]interface{}{
				"prompt_tokens":     chatResp.Usage.PromptTokens,
				"completion_tokens": chatResp.Usage.CompletionTokens,
				"total_tokens":      totalTokens,
				"tokens_per_second": tokensPerSecond,
				"timestamp":         time.Now(),
			},
		},
	}, nil
}

// GetModelInfo returns basic model information
func (g *GenericProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:            g.ModelName,
		Provider:        "generic",
		Version:         g.ModelName,
		Description:     "Generic OpenAI-compatible chat model",
		ModelType:       "chat",
		ContextWindow:   8192,
		MaxOutputTokens: 4096,
	}
}

// SupportsStreaming indicates whether streaming is supported
func (g *GenericProvider) SupportsStreaming() bool { return false }

// ValidateConnection makes a tiny test request to validate the endpoint
func (g *GenericProvider) ValidateConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := GenericChatRequest{
		Model:     g.ModelName,
		Messages:  []GenericChatMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 1,
	}
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	for k, v := range g.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}
	return nil
}

// Ensure interface implementation
var _ interfaces.LLMProvider = (*GenericProvider)(nil)
