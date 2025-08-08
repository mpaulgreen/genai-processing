package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "valid provider with custom endpoint",
			apiKey:   "test-key",
			endpoint: "https://custom.openai.com/v1/chat/completions",
			wantErr:  false,
		},
		{
			name:     "valid provider with default endpoint",
			apiKey:   "test-key",
			endpoint: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProvider(tt.apiKey, tt.endpoint)

			if provider.APIKey != tt.apiKey {
				t.Errorf("NewOpenAIProvider() APIKey = %v, want %v", provider.APIKey, tt.apiKey)
			}

			if tt.endpoint == "" {
				if provider.Endpoint != "https://api.openai.com/v1/chat/completions" {
					t.Errorf("NewOpenAIProvider() Endpoint = %v, want %v", provider.Endpoint, "https://api.openai.com/v1/chat/completions")
				}
			} else {
				if provider.Endpoint != tt.endpoint {
					t.Errorf("NewOpenAIProvider() Endpoint = %v, want %v", provider.Endpoint, tt.endpoint)
				}
			}

			// Verify new fields have default values
			if provider.ModelName != "gpt-4" {
				t.Errorf("NewOpenAIProvider() ModelName = %v, want %v", provider.ModelName, "gpt-4")
			}

			if provider.Parameters == nil {
				t.Error("NewOpenAIProvider() Parameters is nil")
			}

			if len(provider.Parameters) != 0 {
				t.Errorf("NewOpenAIProvider() Parameters should be empty, got %v", provider.Parameters)
			}

			if provider.client == nil {
				t.Error("NewOpenAIProvider() client is nil")
			}

			if provider.client.Timeout != 30*time.Second {
				t.Errorf("NewOpenAIProvider() client timeout = %v, want %v", provider.client.Timeout, 30*time.Second)
			}
		})
	}
}

func TestNewOpenAIProviderWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		endpoint   string
		modelName  string
		parameters map[string]interface{}
		wantErr    bool
	}{
		{
			name:      "valid provider with full config",
			apiKey:    "test-key",
			endpoint:  "https://custom.openai.com/v1/chat/completions",
			modelName: "gpt-4-turbo",
			parameters: map[string]interface{}{
				"max_tokens":  2000,
				"temperature": 0.2,
				"top_p":       0.9,
			},
			wantErr: false,
		},
		{
			name:      "valid provider with empty endpoint (should use default)",
			apiKey:    "test-key",
			endpoint:  "",
			modelName: "gpt-3.5-turbo",
			parameters: map[string]interface{}{
				"max_tokens": 1000,
			},
			wantErr: false,
		},
		{
			name:      "valid provider with empty model name (should use default)",
			apiKey:    "test-key",
			endpoint:  "https://api.openai.com/v1/chat/completions",
			modelName: "",
			parameters: map[string]interface{}{
				"temperature": 0.5,
			},
			wantErr: false,
		},
		{
			name:       "valid provider with nil parameters (should use empty map)",
			apiKey:     "test-key",
			endpoint:   "https://api.openai.com/v1/chat/completions",
			modelName:  "gpt-4",
			parameters: nil,
			wantErr:    false,
		},
		{
			name:       "valid provider with empty parameters map",
			apiKey:     "test-key",
			endpoint:   "https://api.openai.com/v1/chat/completions",
			modelName:  "gpt-4",
			parameters: map[string]interface{}{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProviderWithConfig(tt.apiKey, tt.endpoint, tt.modelName, tt.parameters)

			// Verify basic fields
			if provider.APIKey != tt.apiKey {
				t.Errorf("NewOpenAIProviderWithConfig() APIKey = %v, want %v", provider.APIKey, tt.apiKey)
			}

			// Verify endpoint
			expectedEndpoint := tt.endpoint
			if expectedEndpoint == "" {
				expectedEndpoint = "https://api.openai.com/v1/chat/completions"
			}
			if provider.Endpoint != expectedEndpoint {
				t.Errorf("NewOpenAIProviderWithConfig() Endpoint = %v, want %v", provider.Endpoint, expectedEndpoint)
			}

			// Verify model name
			expectedModelName := tt.modelName
			if expectedModelName == "" {
				expectedModelName = "gpt-4"
			}
			if provider.ModelName != expectedModelName {
				t.Errorf("NewOpenAIProviderWithConfig() ModelName = %v, want %v", provider.ModelName, expectedModelName)
			}

			// Verify parameters
			if provider.Parameters == nil {
				t.Error("NewOpenAIProviderWithConfig() Parameters is nil")
			} else {
				if tt.parameters == nil {
					// If nil was passed, should be empty map
					if len(provider.Parameters) != 0 {
						t.Errorf("NewOpenAIProviderWithConfig() Parameters should be empty when nil passed, got %v", provider.Parameters)
					}
				} else {
					// Verify all parameters are copied correctly
					for key, expectedValue := range tt.parameters {
						if actualValue, exists := provider.Parameters[key]; !exists {
							t.Errorf("NewOpenAIProviderWithConfig() missing parameter %s", key)
						} else if actualValue != expectedValue {
							t.Errorf("NewOpenAIProviderWithConfig() parameter %s = %v, want %v", key, actualValue, expectedValue)
						}
					}
					// Verify no extra parameters were added
					if len(provider.Parameters) != len(tt.parameters) {
						t.Errorf("NewOpenAIProviderWithConfig() Parameters count = %d, want %d", len(provider.Parameters), len(tt.parameters))
					}
				}
			}

			// Verify client
			if provider.client == nil {
				t.Error("NewOpenAIProviderWithConfig() client is nil")
			}

			if provider.client.Timeout != 30*time.Second {
				t.Errorf("NewOpenAIProviderWithConfig() client timeout = %v, want %v", provider.client.Timeout, 30*time.Second)
			}
		})
	}
}

func TestOpenAIProvider_GenerateResponse_WithStoredConfig(t *testing.T) {
	// Test that stored configuration is used as defaults
	provider := NewOpenAIProviderWithConfig(
		"test-key",
		"https://api.openai.com/v1/chat/completions",
		"gpt-4-turbo",
		map[string]interface{}{
			"max_tokens":  1500,
			"temperature": 0.3,
			"top_p":       0.8,
		},
	)

	// Create a request without specifying model or parameters
	request := &types.ModelRequest{
		Model: "", // Empty model should use stored model
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Hello, GPT!",
			},
		},
		Parameters: nil, // No parameters should use stored parameters
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request uses stored configuration
		var reqBody OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify model name
		if reqBody.Model != "gpt-4-turbo" {
			t.Errorf("Request model = %s, want gpt-4-turbo", reqBody.Model)
		}

		// Verify parameters
		if reqBody.MaxTokens != 1500 {
			t.Errorf("Request max_tokens = %d, want 1500", reqBody.MaxTokens)
		}
		if reqBody.Temperature != 0.3 {
			t.Errorf("Request temperature = %f, want 0.3", reqBody.Temperature)
		}
		if reqBody.TopP != 0.8 {
			t.Errorf("Request top_p = %f, want 0.8", reqBody.TopP)
		}

		// Return success response
		response := OpenAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4-turbo",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Update provider to use test server
	provider.Endpoint = server.URL

	// Test the response
	ctx := context.Background()
	response, err := provider.GenerateResponse(ctx, request)

	if err != nil {
		t.Errorf("GenerateResponse() error = %v", err)
	}

	if response == nil {
		t.Error("GenerateResponse() returned nil response")
	} else if response.Content != "Hello! How can I help you today?" {
		t.Errorf("GenerateResponse() content = %s, want 'Hello! How can I help you today?'", response.Content)
	}
}

func TestOpenAIProvider_GenerateResponse_RequestOverridesStoredConfig(t *testing.T) {
	// Test that request parameters override stored configuration
	provider := NewOpenAIProviderWithConfig(
		"test-key",
		"https://api.openai.com/v1/chat/completions",
		"gpt-4", // Stored model
		map[string]interface{}{
			"max_tokens":  1000, // Stored parameters
			"temperature": 0.1,
			"top_p":       1.0,
		},
	)

	// Create a request that overrides stored configuration
	request := &types.ModelRequest{
		Model: "gpt-3.5-turbo", // Override stored model
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Hello, GPT!",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  500, // Override stored parameter
			"temperature": 0.5, // Override stored parameter
			"top_p":       0.9, // Override stored parameter
		},
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request uses request parameters (not stored ones)
		var reqBody OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify model name (should use request model, not stored)
		if reqBody.Model != "gpt-3.5-turbo" {
			t.Errorf("Request model = %s, want gpt-3.5-turbo", reqBody.Model)
		}

		// Verify parameters (should use request parameters, not stored)
		if reqBody.MaxTokens != 500 {
			t.Errorf("Request max_tokens = %d, want 500", reqBody.MaxTokens)
		}
		if reqBody.Temperature != 0.5 {
			t.Errorf("Request temperature = %f, want 0.5", reqBody.Temperature)
		}
		if reqBody.TopP != 0.9 {
			t.Errorf("Request top_p = %f, want 0.9", reqBody.TopP)
		}

		// Return success response
		response := OpenAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-3.5-turbo",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! I'm GPT-3.5-turbo.",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Update provider to use test server
	provider.Endpoint = server.URL

	// Test the response
	ctx := context.Background()
	response, err := provider.GenerateResponse(ctx, request)

	if err != nil {
		t.Errorf("GenerateResponse() error = %v", err)
	}

	if response == nil {
		t.Error("GenerateResponse() returned nil response")
	} else if response.Content != "Hello! I'm GPT-3.5-turbo." {
		t.Errorf("GenerateResponse() content = %s, want 'Hello! I'm GPT-3.5-turbo.'", response.Content)
	}
}

func TestOpenAIProvider_GetModelInfo_WithStoredModelName(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		wantName  string
	}{
		{
			name:      "custom model name",
			modelName: "gpt-4-turbo",
			wantName:  "gpt-4-turbo",
		},
		{
			name:      "default model name",
			modelName: "gpt-4",
			wantName:  "gpt-4",
		},
		{
			name:      "gpt-3.5 model name",
			modelName: "gpt-3.5-turbo",
			wantName:  "gpt-3.5-turbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProviderWithConfig(
				"test-key",
				"https://api.openai.com/v1/chat/completions",
				tt.modelName,
				nil,
			)

			modelInfo := provider.GetModelInfo()

			if modelInfo.Name != tt.wantName {
				t.Errorf("GetModelInfo() Name = %s, want %s", modelInfo.Name, tt.wantName)
			}

			if modelInfo.Version != tt.wantName {
				t.Errorf("GetModelInfo() Version = %s, want %s", modelInfo.Version, tt.wantName)
			}

			if modelInfo.Provider != "openai" {
				t.Errorf("GetModelInfo() Provider = %s, want openai", modelInfo.Provider)
			}
		})
	}
}

func TestOpenAIProvider_GenerateResponse(t *testing.T) {
	tests := []struct {
		name           string
		request        *types.ModelRequest
		mockResponse   interface{}
		mockStatusCode int
		wantErr        bool
		checkResponse  func(*testing.T, *types.RawResponse)
	}{
		{
			name: "successful response",
			request: &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello, GPT!",
					},
				},
				Parameters: map[string]interface{}{
					"max_tokens":  100,
					"temperature": 0.1,
				},
			},
			mockResponse: OpenAIResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gpt-4",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     10,
					CompletionTokens: 15,
					TotalTokens:      25,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *types.RawResponse) {
				if resp.Content != "Hello! How can I help you today?" {
					t.Errorf("expected content 'Hello! How can I help you today?', got '%s'", resp.Content)
				}
				if resp.ModelInfo["model"] != "gpt-4" {
					t.Errorf("expected model 'gpt-4', got '%v'", resp.ModelInfo["model"])
				}
				if resp.Metadata["provider"] != "openai" {
					t.Errorf("expected provider 'openai', got '%v'", resp.Metadata["provider"])
				}
				if resp.Metadata["api_version"] != "v1" {
					t.Errorf("expected api_version 'v1', got '%v'", resp.Metadata["api_version"])
				}
			},
		},
		{
			name: "API error response",
			request: &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse: OpenAIError{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
					Param   string `json:"param,omitempty"`
				}{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
					Code:    "invalid_api_key",
				},
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "empty API key",
			request: &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse:   nil,
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
		{
			name: "empty response content",
			request: &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse: OpenAIResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gpt-4",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     5,
					CompletionTokens: 0,
					TotalTokens:      5,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *types.RawResponse) {
				if resp.Content != "" {
					t.Errorf("expected empty content, got '%s'", resp.Content)
				}
			},
		},
		{
			name: "rate limit error",
			request: &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse: OpenAIError{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
					Param   string `json:"param,omitempty"`
				}{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
					Code:    "rate_limit_exceeded",
				},
			},
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("expected Authorization header 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
				}

				// Set response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Create provider
			apiKey := "test-key"
			if tt.name == "empty API key" {
				apiKey = ""
			}
			provider := NewOpenAIProvider(apiKey, server.URL)

			// Test GenerateResponse
			ctx := context.Background()
			resp, err := provider.GenerateResponse(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Error("expected response, got nil")
				return
			}

			// Run custom response checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestOpenAIProvider_GetModelInfo(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")
	info := provider.GetModelInfo()

	expected := types.ModelInfo{
		Name:            "gpt-4",
		Provider:        "openai",
		Version:         "gpt-4",
		Description:     "GPT-4 - Advanced language model for complex reasoning and analysis",
		ModelType:       "chat",
		ContextWindow:   8192,
		MaxOutputTokens: 4096,
	}

	if info.Name != expected.Name {
		t.Errorf("GetModelInfo() Name = %v, want %v", info.Name, expected.Name)
	}
	if info.Provider != expected.Provider {
		t.Errorf("GetModelInfo() Provider = %v, want %v", info.Provider, expected.Provider)
	}
	if info.Version != expected.Version {
		t.Errorf("GetModelInfo() Version = %v, want %v", info.Version, expected.Version)
	}
	if info.ModelType != expected.ModelType {
		t.Errorf("GetModelInfo() ModelType = %v, want %v", info.ModelType, expected.ModelType)
	}
	if info.ContextWindow != expected.ContextWindow {
		t.Errorf("GetModelInfo() ContextWindow = %v, want %v", info.ContextWindow, expected.ContextWindow)
	}
	if info.MaxOutputTokens != expected.MaxOutputTokens {
		t.Errorf("GetModelInfo() MaxOutputTokens = %v, want %v", info.MaxOutputTokens, expected.MaxOutputTokens)
	}
	if info.Description != expected.Description {
		t.Errorf("GetModelInfo() Description = %v, want %v", info.Description, expected.Description)
	}

	// Check pricing info
	if pricing, ok := info.PricingInfo["input_cost_per_1k_tokens"].(float64); !ok || pricing != 0.03 {
		t.Errorf("GetModelInfo() input_cost_per_1k_tokens = %v, want %v", pricing, 0.03)
	}
	if pricing, ok := info.PricingInfo["output_cost_per_1k_tokens"].(float64); !ok || pricing != 0.06 {
		t.Errorf("GetModelInfo() output_cost_per_1k_tokens = %v, want %v", pricing, 0.06)
	}
}

func TestOpenAIProvider_SupportsStreaming(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")
	if provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false for current implementation")
	}
}

func TestOpenAIProvider_ValidateConnection(t *testing.T) {
	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   interface{}
		wantErr        bool
	}{
		{
			name:           "successful connection",
			mockStatusCode: http.StatusOK,
			mockResponse: OpenAIResponse{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gpt-4",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "Hello",
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     1,
					CompletionTokens: 1,
					TotalTokens:      2,
				},
			},
			wantErr: false,
		},
		{
			name:           "connection failed",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse: OpenAIError{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
					Param   string `json:"param,omitempty"`
				}{
					Message: "Invalid API key",
					Type:    "authentication_error",
					Code:    "invalid_api_key",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			provider := NewOpenAIProvider("test-key", server.URL)
			err := provider.ValidateConnection()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOpenAIProvider_MessageConversion(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")

	request := &types.ModelRequest{
		Model: "gpt-4",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			map[string]interface{}{
				"role":    "user",
				"content": "What is 2+2?",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":        50,
			"temperature":       0.5,
			"top_p":             0.9,
			"frequency_penalty": 0.0,
			"presence_penalty":  0.0,
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to verify conversion
		var openaiReq OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		// Verify model
		if openaiReq.Model != "gpt-4" {
			t.Errorf("expected model 'gpt-4', got '%s'", openaiReq.Model)
		}

		// Verify messages
		if len(openaiReq.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(openaiReq.Messages))
		}

		if openaiReq.Messages[0].Role != "system" {
			t.Errorf("expected first message role 'system', got '%s'", openaiReq.Messages[0].Role)
		}

		if openaiReq.Messages[1].Role != "user" {
			t.Errorf("expected second message role 'user', got '%s'", openaiReq.Messages[1].Role)
		}

		// Verify parameters
		if openaiReq.MaxTokens != 50 {
			t.Errorf("expected max_tokens 50, got %d", openaiReq.MaxTokens)
		}

		if openaiReq.Temperature != 0.5 {
			t.Errorf("expected temperature 0.5, got %f", openaiReq.Temperature)
		}

		if openaiReq.TopP != 0.9 {
			t.Errorf("expected top_p 0.9, got %f", openaiReq.TopP)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenAIResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "2+2 equals 4.",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		})
	}))
	defer server.Close()

	// Update provider endpoint
	provider.Endpoint = server.URL

	// Test the conversion
	ctx := context.Background()
	_, err := provider.GenerateResponse(ctx, request)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIProvider_CalculateCost(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")

	tests := []struct {
		name             string
		promptTokens     int
		completionTokens int
		model            string
		expectedCost     float64
	}{
		{
			name:             "GPT-4 cost calculation",
			promptTokens:     1000,
			completionTokens: 500,
			model:            "gpt-4",
			expectedCost:     0.06, // (1000 * 0.03) + (500 * 0.06) = 30 + 30 = 60 cents
		},
		{
			name:             "GPT-4-turbo cost calculation",
			promptTokens:     1000,
			completionTokens: 500,
			model:            "gpt-4-turbo",
			expectedCost:     0.025, // (1000 * 0.01) + (500 * 0.03) = 10 + 15 = 25 cents
		},
		{
			name:             "GPT-3.5-turbo cost calculation",
			promptTokens:     1000,
			completionTokens: 500,
			model:            "gpt-3.5-turbo",
			expectedCost:     0.0025, // (1000 * 0.0015) + (500 * 0.002) = 1.5 + 1 = 2.5 cents
		},
		{
			name:             "Unknown model defaults to GPT-4 pricing",
			promptTokens:     1000,
			completionTokens: 500,
			model:            "unknown-model",
			expectedCost:     0.06,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := provider.calculateCost(tt.promptTokens, tt.completionTokens, tt.model)
			if cost != tt.expectedCost {
				t.Errorf("calculateCost() = %f, want %f", cost, tt.expectedCost)
			}
		})
	}
}

func TestOpenAIProvider_ErrorHandling(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")

	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   string
		expectedError  string
	}{
		{
			name:           "malformed JSON error response",
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   `{"invalid": json}`,
			expectedError:  "failed to parse error response",
		},
		{
			name:           "network error simulation",
			mockStatusCode: http.StatusOK,
			mockResponse:   "",
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server that returns malformed JSON
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			provider.Endpoint = server.URL

			request := &types.ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			}

			ctx := context.Background()
			_, err := provider.GenerateResponse(ctx, request)

			if tt.expectedError != "" {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			}
		})
	}
}
