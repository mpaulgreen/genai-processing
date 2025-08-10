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

// OllamaProvider implements a client for Ollama's API format
type OllamaProvider struct {
	APIKey     string
	Endpoint   string
	ModelName  string
	Parameters map[string]interface{}
	Headers    map[string]string
	client     *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(apiKey, endpoint, modelName string, headers map[string]string, parameters map[string]interface{}) *OllamaProvider {
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/generate"
	}
	if modelName == "" {
		modelName = "llama3.1:8b"
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if parameters == nil {
		parameters = map[string]interface{}{}
	}

	// Set default content-type for Ollama
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	return &OllamaProvider{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		ModelName:  modelName,
		Parameters: parameters,
		Headers:    headers,
		client:     &http.Client{Timeout: 60 * time.Second}, // Longer timeout for local models
	}
}

// GenerateResponse sends a request to Ollama and returns a RawResponse
func (o *OllamaProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	if o.Endpoint == "" {
		return nil, fmt.Errorf("ollama provider endpoint is required")
	}

	// Determine model
	modelName := request.Model
	if modelName == "" {
		modelName = o.ModelName
	}

	// Extract Ollama request from the messages
	var ollamaReq types.OllamaRequest
	if len(request.Messages) > 0 {
		// The first message should contain the Ollama request
		if msg, ok := request.Messages[0].(types.OllamaRequest); ok {
			ollamaReq = msg
		} else {
			// Fallback: try to extract prompt from generic message format
			if msgMap, ok := request.Messages[0].(map[string]interface{}); ok {
				if content, ok := msgMap["content"].(string); ok {
					ollamaReq = types.OllamaRequest{
						Model:  modelName,
						Prompt: content,
						Stream: false,
						Options: map[string]interface{}{
							"temperature": 0.1,
							"num_predict": 4000,
						},
						Format: "json",
					}
				}
			}
		}
	}

	// If we still don't have a valid request, create one with defaults
	if ollamaReq.Model == "" {
		ollamaReq = types.OllamaRequest{
			Model:  modelName,
			Prompt: "Hello", // Fallback prompt
			Stream: false,
			Options: map[string]interface{}{
				"temperature": 0.1,
				"num_predict": 4000,
			},
			Format: "json",
		}
	}

	// Override with parameters from request
	if request.Parameters != nil {
		if v, ok := request.Parameters["temperature"].(float64); ok {
			if ollamaReq.Options == nil {
				ollamaReq.Options = make(map[string]interface{})
			}
			ollamaReq.Options["temperature"] = v
		}
		if v, ok := request.Parameters["max_tokens"].(int); ok {
			if ollamaReq.Options == nil {
				ollamaReq.Options = make(map[string]interface{})
			}
			ollamaReq.Options["num_predict"] = v
		}
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range o.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse Ollama error envelope
		var apiErr types.OllamaAPIError
		if err := json.Unmarshal(respBytes, &apiErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
		}
		return nil, fmt.Errorf("ollama API error: %s", apiErr.Error)
	}

	var ollamaResp types.OllamaResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	processingTime := time.Since(start)

	// Calculate tokens per second (approximate)
	tokensPerSecond := 0.0
	if processingTime > 0 && ollamaResp.EvalCount > 0 {
		tokensPerSecond = float64(ollamaResp.EvalCount) / processingTime.Seconds()
	}

	return &types.RawResponse{
		Content: ollamaResp.Response,
		ModelInfo: map[string]interface{}{
			"model":         ollamaResp.Model,
			"done":          ollamaResp.Done,
			"finish_reason": "stop", // Ollama doesn't provide this, assume "stop"
		},
		Metadata: map[string]interface{}{
			"provider":        "ollama",
			"api_version":     "v1",
			"processing_time": processingTime.String(),
			"token_usage": map[string]interface{}{
				"prompt_tokens":     ollamaResp.PromptEvalCount,
				"completion_tokens": ollamaResp.EvalCount,
				"total_tokens":      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
				"tokens_per_second": tokensPerSecond,
				"timestamp":         time.Now(),
			},
			"ollama_metadata": map[string]interface{}{
				"total_duration":       ollamaResp.TotalDuration,
				"load_duration":        ollamaResp.LoadDuration,
				"prompt_eval_duration": ollamaResp.PromptEvalDuration,
				"eval_duration":        ollamaResp.EvalDuration,
			},
		},
	}, nil
}

// GetModelInfo returns basic model information for Ollama
func (o *OllamaProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:            o.ModelName,
		Provider:        "ollama",
		Version:         o.ModelName,
		Description:     "Local Ollama model",
		ModelType:       "completion",
		ContextWindow:   8192,
		MaxOutputTokens: 8192,
	}
}

// SupportsStreaming indicates whether streaming is supported
func (o *OllamaProvider) SupportsStreaming() bool { return true }

// ValidateConnection makes a test request to validate the Ollama endpoint
func (o *OllamaProvider) ValidateConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := types.OllamaRequest{
		Model:  o.ModelName,
		Prompt: "Hello",
		Stream: false,
		Options: map[string]interface{}{
			"num_predict": 1,
		},
	}
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	for k, v := range o.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama connection test failed with status: %d", resp.StatusCode)
	}
	return nil
}

// Ensure interface implementation
var _ interfaces.LLMProvider = (*OllamaProvider)(nil)
