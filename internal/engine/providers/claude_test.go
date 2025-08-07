package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewClaudeProvider(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "valid provider with custom endpoint",
			apiKey:   "test-key",
			endpoint: "https://custom.anthropic.com/v1/messages",
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
			provider := NewClaudeProvider(tt.apiKey, tt.endpoint)

			if provider.APIKey != tt.apiKey {
				t.Errorf("NewClaudeProvider() APIKey = %v, want %v", provider.APIKey, tt.apiKey)
			}

			if tt.endpoint == "" {
				if provider.Endpoint != "https://api.anthropic.com/v1/messages" {
					t.Errorf("NewClaudeProvider() Endpoint = %v, want %v", provider.Endpoint, "https://api.anthropic.com/v1/messages")
				}
			} else {
				if provider.Endpoint != tt.endpoint {
					t.Errorf("NewClaudeProvider() Endpoint = %v, want %v", provider.Endpoint, tt.endpoint)
				}
			}

			if provider.client == nil {
				t.Error("NewClaudeProvider() client is nil")
			}

			if provider.client.Timeout != 30*time.Second {
				t.Errorf("NewClaudeProvider() client timeout = %v, want %v", provider.client.Timeout, 30*time.Second)
			}
		})
	}
}

func TestClaudeProvider_GenerateResponse(t *testing.T) {
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
				Model: "claude-3-5-sonnet-20241022",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello, Claude!",
					},
				},
				Parameters: map[string]interface{}{
					"max_tokens":  100,
					"temperature": 0.1,
				},
			},
			mockResponse: ClaudeResponse{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{
						Type: "text",
						Text: "Hello! How can I help you today?",
					},
				},
				Model:      "claude-3-5-sonnet-20241022",
				StopReason: "end_turn",
				Usage: struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				}{
					InputTokens:  10,
					OutputTokens: 15,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *types.RawResponse) {
				if resp.Content != "Hello! How can I help you today?" {
					t.Errorf("expected content 'Hello! How can I help you today?', got '%s'", resp.Content)
				}
				if resp.ModelInfo["model"] != "claude-3-5-sonnet-20241022" {
					t.Errorf("expected model 'claude-3-5-sonnet-20241022', got '%v'", resp.ModelInfo["model"])
				}
				if resp.Metadata["provider"] != "anthropic" {
					t.Errorf("expected provider 'anthropic', got '%v'", resp.Metadata["provider"])
				}
			},
		},
		{
			name: "API error response",
			request: &types.ModelRequest{
				Model: "claude-3-5-sonnet-20241022",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse: ClaudeError{
				Type:    "invalid_request_error",
				Message: "Invalid API key",
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "empty API key",
			request: &types.ModelRequest{
				Model: "claude-3-5-sonnet-20241022",
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
				Model: "claude-3-5-sonnet-20241022",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello",
					},
				},
			},
			mockResponse: ClaudeResponse{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{},
				Model:      "claude-3-5-sonnet-20241022",
				StopReason: "end_turn",
				Usage: struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				}{
					InputTokens:  5,
					OutputTokens: 0,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
				}
				if r.Header.Get("x-api-key") != "test-key" {
					t.Errorf("expected x-api-key header 'test-key', got '%s'", r.Header.Get("x-api-key"))
				}
				if r.Header.Get("anthropic-version") != "2023-06-01" {
					t.Errorf("expected anthropic-version header '2023-06-01', got '%s'", r.Header.Get("anthropic-version"))
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
			provider := NewClaudeProvider(apiKey, server.URL)

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

func TestClaudeProvider_GetModelInfo(t *testing.T) {
	provider := NewClaudeProvider("test-key", "")
	info := provider.GetModelInfo()

	expected := types.ModelInfo{
		Name:            "claude-3-5-sonnet-20241022",
		Provider:        "anthropic",
		Version:         "20241022",
		Description:     "Claude 3.5 Sonnet - Advanced language model for complex reasoning and analysis",
		ModelType:       "chat",
		ContextWindow:   200000,
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
}

func TestClaudeProvider_SupportsStreaming(t *testing.T) {
	provider := NewClaudeProvider("test-key", "")
	if provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false for current implementation")
	}
}

func TestClaudeProvider_ValidateConnection(t *testing.T) {
	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   interface{}
		wantErr        bool
	}{
		{
			name:           "successful connection",
			mockStatusCode: http.StatusOK,
			mockResponse: ClaudeResponse{
				ID:   "msg_test",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{
						Type: "text",
						Text: "Hello",
					},
				},
				Model:      "claude-3-5-sonnet-20241022",
				StopReason: "end_turn",
				Usage: struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				}{
					InputTokens:  1,
					OutputTokens: 1,
				},
			},
			wantErr: false,
		},
		{
			name:           "connection failed",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse: ClaudeError{
				Type:    "authentication_error",
				Message: "Invalid API key",
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

			provider := NewClaudeProvider("test-key", server.URL)
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

func TestClaudeProvider_MessageConversion(t *testing.T) {
	provider := NewClaudeProvider("test-key", "")

	request := &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
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
			"max_tokens":  50,
			"temperature": 0.5,
			"system":      "You are a math tutor.",
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to verify conversion
		var claudeReq ClaudeRequest
		if err := json.NewDecoder(r.Body).Decode(&claudeReq); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		// Verify model
		if claudeReq.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("expected model 'claude-3-5-sonnet-20241022', got '%s'", claudeReq.Model)
		}

		// Verify messages
		if len(claudeReq.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(claudeReq.Messages))
		}

		if claudeReq.Messages[0].Role != "system" {
			t.Errorf("expected first message role 'system', got '%s'", claudeReq.Messages[0].Role)
		}

		if claudeReq.Messages[1].Role != "user" {
			t.Errorf("expected second message role 'user', got '%s'", claudeReq.Messages[1].Role)
		}

		// Verify parameters
		if claudeReq.MaxTokens != 50 {
			t.Errorf("expected max_tokens 50, got %d", claudeReq.MaxTokens)
		}

		if claudeReq.Temperature != 0.5 {
			t.Errorf("expected temperature 0.5, got %f", claudeReq.Temperature)
		}

		if claudeReq.System != "You are a math tutor." {
			t.Errorf("expected system 'You are a math tutor.', got '%s'", claudeReq.System)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ClaudeResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: "2+2 equals 4.",
				},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 5,
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
