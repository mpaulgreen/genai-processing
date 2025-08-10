package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"genai-processing/pkg/types"
)

func TestNewOllamaProvider(t *testing.T) {
	provider := NewOllamaProvider("", "http://localhost:11434/api/generate", "llama3.1:8b", nil, nil)

	if provider == nil {
		t.Fatal("NewOllamaProvider returned nil")
	}

	if provider.Endpoint != "http://localhost:11434/api/generate" {
		t.Errorf("Expected endpoint 'http://localhost:11434/api/generate', got '%s'", provider.Endpoint)
	}

	if provider.ModelName != "llama3.1:8b" {
		t.Errorf("Expected model name 'llama3.1:8b', got '%s'", provider.ModelName)
	}

	if provider.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", provider.Headers["Content-Type"])
	}
}

func TestNewOllamaProvider_Defaults(t *testing.T) {
	provider := NewOllamaProvider("", "", "", nil, nil)

	if provider.Endpoint != "http://localhost:11434/api/generate" {
		t.Errorf("Expected default endpoint 'http://localhost:11434/api/generate', got '%s'", provider.Endpoint)
	}

	if provider.ModelName != "llama3.1:8b" {
		t.Errorf("Expected default model name 'llama3.1:8b', got '%s'", provider.ModelName)
	}
}

func TestOllamaProvider_GenerateResponse_Success(t *testing.T) {
	// Create a test server that returns a valid Ollama response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse and verify request body
		var req types.OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", req.Model)
		}

		if req.Prompt == "" {
			t.Errorf("Expected non-empty prompt")
		}

		if req.Stream != false {
			t.Errorf("Expected stream false, got %v", req.Stream)
		}

		// Return a valid Ollama response
		response := types.OllamaResponse{
			Model:              "test-model",
			Response:           `{"log_source": "kube-apiserver", "resource": "pods"}`,
			Done:               true,
			TotalDuration:      1000,
			LoadDuration:       100,
			PromptEvalCount:    50,
			PromptEvalDuration: 500,
			EvalCount:          25,
			EvalDuration:       400,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	// Create a request with OllamaRequest in messages
	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "Convert this query to JSON: show me all pods",
				Stream: false,
				Options: map[string]interface{}{
					"temperature": 0.1,
					"num_predict": 100,
				},
				Format: "json",
			},
		},
		Parameters: map[string]interface{}{
			"temperature": 0.1,
			"max_tokens":  100,
		},
	}

	ctx := context.Background()
	resp, err := provider.GenerateResponse(ctx, req)

	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}

	if resp.Content != `{"log_source": "kube-apiserver", "resource": "pods"}` {
		t.Errorf("Expected response content, got '%s'", resp.Content)
	}

	if resp.ModelInfo["model"] != "test-model" {
		t.Errorf("Expected model info 'test-model', got '%v'", resp.ModelInfo["model"])
	}

	if resp.ModelInfo["done"] != true {
		t.Errorf("Expected done true, got %v", resp.ModelInfo["done"])
	}

	// Check token usage
	tokenUsage := resp.Metadata["token_usage"].(map[string]interface{})
	if tokenUsage["prompt_tokens"] != 50 {
		t.Errorf("Expected prompt_tokens 50, got %v", tokenUsage["prompt_tokens"])
	}

	if tokenUsage["completion_tokens"] != 25 {
		t.Errorf("Expected completion_tokens 25, got %v", tokenUsage["completion_tokens"])
	}

	if tokenUsage["total_tokens"] != 75 {
		t.Errorf("Expected total_tokens 75, got %v", tokenUsage["total_tokens"])
	}
}

func TestOllamaProvider_GenerateResponse_Fallback(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.OllamaResponse{
			Model:           "test-model",
			Response:        `{"log_source": "kube-apiserver", "resource": "pods"}`,
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       5,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	// Create a request with generic message format (fallback case)
	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Convert this query to JSON: show me all pods",
			},
		},
	}

	ctx := context.Background()
	resp, err := provider.GenerateResponse(ctx, req)

	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}

	if resp.Content != `{"log_source": "kube-apiserver", "resource": "pods"}` {
		t.Errorf("Expected response content, got '%s'", resp.Content)
	}
}

func TestOllamaProvider_GenerateResponse_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		errorResp := types.OllamaAPIError{
			Error: "Model not found",
		}
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateResponse(ctx, req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "ollama API error: Model not found" {
		t.Errorf("Expected error message 'ollama API error: Model not found', got '%s'", err.Error())
	}
}

func TestOllamaProvider_GenerateResponse_HTTPError(t *testing.T) {
	// Create a test server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateResponse(ctx, req)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "HTTP 500: Internal server error" {
		t.Errorf("Expected HTTP error, got '%s'", err.Error())
	}
}

func TestOllamaProvider_GetModelInfo(t *testing.T) {
	provider := NewOllamaProvider("", "", "llama3.1:8b", nil, nil)

	info := provider.GetModelInfo()

	if info.Name != "llama3.1:8b" {
		t.Errorf("Expected model name 'llama3.1:8b', got '%s'", info.Name)
	}

	if info.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", info.Provider)
	}

	if info.ModelType != "completion" {
		t.Errorf("Expected model type 'completion', got '%s'", info.ModelType)
	}

	if info.ContextWindow != 8192 {
		t.Errorf("Expected context window 8192, got %d", info.ContextWindow)
	}

	if info.MaxOutputTokens != 8192 {
		t.Errorf("Expected max output tokens 8192, got %d", info.MaxOutputTokens)
	}
}

func TestOllamaProvider_SupportsStreaming(t *testing.T) {
	provider := NewOllamaProvider("", "", "", nil, nil)

	if !provider.SupportsStreaming() {
		t.Error("Expected SupportsStreaming to return true")
	}
}

func TestOllamaProvider_ValidateConnection(t *testing.T) {
	// Create a test server that responds successfully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.OllamaResponse{
			Model:     "test-model",
			Response:  "Hello",
			Done:      true,
			EvalCount: 1,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	err := provider.ValidateConnection()
	if err != nil {
		t.Errorf("Expected successful connection validation, got error: %v", err)
	}
}

func TestOllamaProvider_ValidateConnection_Failure(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Model not found"))
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	err := provider.ValidateConnection()
	if err == nil {
		t.Fatal("Expected connection validation to fail")
	}

	if err.Error() != "ollama connection test failed with status: 404" {
		t.Errorf("Expected connection error, got '%s'", err.Error())
	}
}

func TestOllamaProvider_GenerateResponse_EmptyEndpoint(t *testing.T) {
	// Create provider with empty endpoint by setting it after construction
	provider := NewOllamaProvider("", "http://localhost:11434/api/generate", "test-model", nil, nil)
	provider.Endpoint = "" // Override to empty

	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateResponse(ctx, req)

	if err == nil {
		t.Fatal("Expected error for empty endpoint")
	}

	if err.Error() != "ollama provider endpoint is required" {
		t.Errorf("Expected endpoint error, got '%s'", err.Error())
	}
}

func TestOllamaProvider_GenerateResponse_ParameterOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.OllamaRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Check that parameters were overridden
		if temp, ok := req.Options["temperature"].(float64); !ok || temp != 0.5 {
			t.Errorf("Expected temperature 0.5, got %v (type: %T)", req.Options["temperature"], req.Options["temperature"])
		}

		if numPredict, ok := req.Options["num_predict"].(float64); !ok || numPredict != 200 {
			t.Errorf("Expected num_predict 200, got %v (type: %T)", req.Options["num_predict"], req.Options["num_predict"])
		}

		response := types.OllamaResponse{
			Model:           "test-model",
			Response:        "test response",
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       5,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider("", server.URL, "test-model", nil, nil)

	req := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
		Parameters: map[string]interface{}{
			"temperature": 0.5,
			"max_tokens":  200,
		},
	}

	ctx := context.Background()
	_, err := provider.GenerateResponse(ctx, req)

	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}
}
