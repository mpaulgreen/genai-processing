package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"genai-processing/pkg/types"
)

// OpenAIProvider implements the LLMProvider interface for OpenAI's API
type OpenAIProvider struct {
	APIKey     string
	Endpoint   string
	ModelName  string
	Parameters map[string]interface{}
	client     *http.Client
}

// OpenAIMessage represents a message in the OpenAI API format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents the request payload for OpenAI API
type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	ResponseFormat   *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIError represents an error response from OpenAI API
type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// NewOpenAIProvider creates a new OpenAIProvider instance
func NewOpenAIProvider(apiKey, endpoint string) *OpenAIProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}

	return &OpenAIProvider{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		ModelName:  "gpt-4", // Default model name
		Parameters: make(map[string]interface{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewOpenAIProviderWithConfig creates a new OpenAIProvider instance with full configuration
func NewOpenAIProviderWithConfig(apiKey, endpoint, modelName string, parameters map[string]interface{}) *OpenAIProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if modelName == "" {
		modelName = "gpt-4"
	}
	if parameters == nil {
		parameters = make(map[string]interface{})
	}

	return &OpenAIProvider{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		ModelName:  modelName,
		Parameters: parameters,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateResponse implements the LLMProvider interface
func (o *OpenAIProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	// Validate API key
	if o.APIKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}

	// Use stored model name if not provided in request
	modelName := request.Model
	if modelName == "" {
		modelName = o.ModelName
	}

	// Convert ModelRequest to OpenAIRequest with stored defaults
	openaiReq := OpenAIRequest{
		Model:       modelName,
		MaxTokens:   4000, // Default max tokens
		Temperature: 0.1,  // Default temperature
		TopP:        1.0,  // Default top_p
		Stream:      false,
	}

	// Apply stored configuration as defaults
	if o.Parameters != nil {
		if maxTokens, ok := o.Parameters["max_tokens"].(int); ok {
			openaiReq.MaxTokens = maxTokens
		}
		if temp, ok := o.Parameters["temperature"].(float64); ok {
			openaiReq.Temperature = temp
		}
		if topP, ok := o.Parameters["top_p"].(float64); ok {
			openaiReq.TopP = topP
		}
		if freqPenalty, ok := o.Parameters["frequency_penalty"].(float64); ok {
			openaiReq.FrequencyPenalty = freqPenalty
		}
		if presPenalty, ok := o.Parameters["presence_penalty"].(float64); ok {
			openaiReq.PresencePenalty = presPenalty
		}
		if stream, ok := o.Parameters["stream"].(bool); ok {
			openaiReq.Stream = stream
		}
		// response_format may be provided as string ("json_object") or map with {type}
		if rf, ok := o.Parameters["response_format"]; ok {
			switch v := rf.(type) {
			case string:
				if v != "" {
					openaiReq.ResponseFormat = &struct {
						Type string `json:"type"`
					}{Type: v}
				}
			case map[string]interface{}:
				if t, ok := v["type"].(string); ok && t != "" {
					openaiReq.ResponseFormat = &struct {
						Type string `json:"type"`
					}{Type: t}
				}
			}
		}
	}

	// Override with request-specific parameters
	if request.Parameters != nil {
		if maxTokens, ok := request.Parameters["max_tokens"].(int); ok {
			openaiReq.MaxTokens = maxTokens
		}
		if temp, ok := request.Parameters["temperature"].(float64); ok {
			openaiReq.Temperature = temp
		}
		if topP, ok := request.Parameters["top_p"].(float64); ok {
			openaiReq.TopP = topP
		}
		if freqPenalty, ok := request.Parameters["frequency_penalty"].(float64); ok {
			openaiReq.FrequencyPenalty = freqPenalty
		}
		if presPenalty, ok := request.Parameters["presence_penalty"].(float64); ok {
			openaiReq.PresencePenalty = presPenalty
		}
		if stream, ok := request.Parameters["stream"].(bool); ok {
			openaiReq.Stream = stream
		}
		if rf, ok := request.Parameters["response_format"]; ok {
			switch v := rf.(type) {
			case string:
				if v != "" {
					openaiReq.ResponseFormat = &struct {
						Type string `json:"type"`
					}{Type: v}
				}
			case map[string]interface{}:
				if t, ok := v["type"].(string); ok && t != "" {
					openaiReq.ResponseFormat = &struct {
						Type string `json:"type"`
					}{Type: t}
				}
			}
		}
	}

	// Convert messages to OpenAI format
	for _, msg := range request.Messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			role, _ := msgMap["role"].(string)
			content, _ := msgMap["content"].(string)
			openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	// If model does not support response_format json_object, drop it to avoid API errors
	if openaiReq.ResponseFormat != nil && !supportsJSONMode(modelName) {
		openaiReq.ResponseFormat = nil
	}

	// Prepare the HTTP request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	// Make the request
	startTime := time.Now()
	resp, err := o.client.Do(req)
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
		var openaiErr OpenAIError
		if err := json.Unmarshal(body, &openaiErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: failed to parse error response: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("openai API error: %s - %s", openaiErr.Error.Type, openaiErr.Error.Message)
	}

	// Parse successful response
	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	// Extract content from response
	var content string
	var finishReason string
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
		finishReason = openaiResp.Choices[0].FinishReason
	}

	// Calculate token usage
	totalTokens := openaiResp.Usage.TotalTokens
	tokensPerSecond := 0.0
	if processingTime > 0 {
		tokensPerSecond = float64(totalTokens) / processingTime.Seconds()
	}

	// Calculate estimated cost (OpenAI pricing as of 2024)
	estimatedCost := o.calculateCost(openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens, openaiResp.Model)

	return &types.RawResponse{
		Content: content,
		ModelInfo: map[string]interface{}{
			"model":         openaiResp.Model,
			"id":            openaiResp.ID,
			"object":        openaiResp.Object,
			"created":       openaiResp.Created,
			"finish_reason": finishReason,
		},
		Metadata: map[string]interface{}{
			"provider":        "openai",
			"api_version":     "v1",
			"processing_time": processingTime.String(),
			"token_usage": map[string]interface{}{
				"prompt_tokens":     openaiResp.Usage.PromptTokens,
				"completion_tokens": openaiResp.Usage.CompletionTokens,
				"total_tokens":      totalTokens,
				"tokens_per_second": tokensPerSecond,
				"model_name":        openaiResp.Model,
				"estimated_cost":    estimatedCost,
				"currency":          "USD",
				"timestamp":         time.Now(),
			},
		},
	}, nil
}

// supportsJSONMode returns true if the model supports response_format {type:"json_object"}
// based on OpenAI's documented JSON mode support.
func supportsJSONMode(model string) bool {
	m := model
	// normalize to lowercase for contains checks
	// note: model names can include versions/suffixes
	mLower := strings.ToLower(m)
	switch {
	case strings.Contains(mLower, "gpt-4o"),
		strings.Contains(mLower, "gpt-4.1"),
		strings.Contains(mLower, "gpt-4o-mini"),
		strings.Contains(mLower, "4o-mini"),
		strings.Contains(mLower, "o-mini"):
		return true
	default:
		return false
	}
}

// GetModelInfo implements the LLMProvider interface
func (o *OpenAIProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:               o.ModelName,
		Provider:           "openai",
		Version:            o.ModelName,
		Description:        "GPT-4 - Advanced language model for complex reasoning and analysis",
		ModelType:          "chat",
		ContextWindow:      8192,
		MaxOutputTokens:    4096,
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ja", "ko", "zh"},
		PricingInfo: map[string]interface{}{
			"input_cost_per_1k_tokens":  0.03,
			"output_cost_per_1k_tokens": 0.06,
			"currency":                  "USD",
		},
		PerformanceMetrics: map[string]interface{}{
			"reasoning_capability": "excellent",
			"coding_capability":    "excellent",
			"analysis_capability":  "excellent",
		},
	}
}

// SupportsStreaming implements the LLMProvider interface
func (o *OpenAIProvider) SupportsStreaming() bool {
	// OpenAI supports streaming, but not implemented in this version
	return false
}

// ValidateConnection checks if the OpenAI API connection is working
func (o *OpenAIProvider) ValidateConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := OpenAIRequest{
		Model: "gpt-4",
		Messages: []OpenAIMessage{
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

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}

	return nil
}

// calculateCost estimates the cost of the API call based on OpenAI pricing
func (o *OpenAIProvider) calculateCost(promptTokens, completionTokens int, model string) float64 {
	var inputCostPer1k, outputCostPer1k float64

	// OpenAI pricing as of 2024 (approximate)
	switch model {
	case "gpt-4":
		inputCostPer1k = 0.03
		outputCostPer1k = 0.06
	case "gpt-4-turbo":
		inputCostPer1k = 0.01
		outputCostPer1k = 0.03
	case "gpt-3.5-turbo":
		inputCostPer1k = 0.0015
		outputCostPer1k = 0.002
	default:
		// Default to GPT-4 pricing for unknown models
		inputCostPer1k = 0.03
		outputCostPer1k = 0.06
	}

	inputCost := float64(promptTokens) / 1000.0 * inputCostPer1k
	outputCost := float64(completionTokens) / 1000.0 * outputCostPer1k

	return inputCost + outputCost
}
