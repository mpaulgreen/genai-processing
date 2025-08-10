package adapters

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestNewOllamaInputAdapter(t *testing.T) {
	adapter := NewOllamaInputAdapter("test-key")

	if adapter == nil {
		t.Fatal("NewOllamaInputAdapter returned nil")
	}

	if adapter.ModelName != "llama3.1:8b" {
		t.Errorf("Expected default model name 'llama3.1:8b', got '%s'", adapter.ModelName)
	}

	if adapter.MaxTokens != 4000 {
		t.Errorf("Expected default max tokens 4000, got %d", adapter.MaxTokens)
	}

	if adapter.Temperature != 0.1 {
		t.Errorf("Expected default temperature 0.1, got %f", adapter.Temperature)
	}
}

func TestOllamaInputAdapter_AdaptRequest(t *testing.T) {
	adapter := NewOllamaInputAdapter("test-key")
	adapter.SetModelName("test-model")
	adapter.SetSystemPrompt("You are a helpful assistant.")

	req := &types.InternalRequest{
		RequestID: "test-request",
		ProcessingRequest: types.ProcessingRequest{
			Query: "Convert this to JSON: show me all pods",
		},
	}

	modelReq, err := adapter.AdaptRequest(req)
	if err != nil {
		t.Fatalf("AdaptRequest failed: %v", err)
	}

	if modelReq.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", modelReq.Model)
	}

	if len(modelReq.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(modelReq.Messages))
	}

	// Check that the message contains an OllamaRequest
	ollamaReq, ok := modelReq.Messages[0].(types.OllamaRequest)
	if !ok {
		t.Fatal("Expected message to be OllamaRequest type")
	}

	if ollamaReq.Model != "test-model" {
		t.Errorf("Expected OllamaRequest model 'test-model', got '%s'", ollamaReq.Model)
	}

	if ollamaReq.Stream != false {
		t.Errorf("Expected Stream to be false, got %v", ollamaReq.Stream)
	}

	if ollamaReq.Format != "json" {
		t.Errorf("Expected Format to be 'json', got '%s'", ollamaReq.Format)
	}

	// Check options
	if ollamaReq.Options == nil {
		t.Fatal("Expected Options to be set")
	}

	if temp, ok := ollamaReq.Options["temperature"].(float64); !ok || temp != 0.1 {
		t.Errorf("Expected temperature 0.1 in options, got %v", temp)
	}

	if numPredict, ok := ollamaReq.Options["num_predict"].(int); !ok || numPredict != 4000 {
		t.Errorf("Expected num_predict 4000 in options, got %v", numPredict)
	}
}

func TestOllamaInputAdapter_FormatPrompt(t *testing.T) {
	adapter := NewOllamaInputAdapter("test-key")
	adapter.SetSystemPrompt("You are a helpful assistant.")

	examples := []types.Example{
		{
			Input:  "show pods",
			Output: `{"log_source": "kube-apiserver", "resource": "pods"}`,
		},
	}

	prompt, err := adapter.FormatPrompt("show me all pods", examples)
	if err != nil {
		t.Fatalf("FormatPrompt failed: %v", err)
	}

	expectedContains := []string{
		"You are a helpful assistant.",
		"Examples:",
		"Input: show pods",
		"Output: {\"log_source\": \"kube-apiserver\", \"resource\": \"pods\"}",
		"Convert this query to JSON: show me all pods",
		"Please respond with valid JSON only.",
	}

	for _, expected := range expectedContains {
		if !containsSubstring(prompt, expected) {
			t.Errorf("Formatted prompt should contain '%s', but got: %s", expected, prompt)
		}
	}
}

func TestOllamaInputAdapter_GetAPIParameters(t *testing.T) {
	adapter := NewOllamaInputAdapter("test-key")
	adapter.SetModelName("test-model")
	adapter.SetSystemPrompt("You are a helpful assistant.")

	params := adapter.GetAPIParameters()

	if params["provider"] != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%v'", params["provider"])
	}

	if params["model_name"] != "test-model" {
		t.Errorf("Expected model_name 'test-model', got '%v'", params["model_name"])
	}

	if params["format"] != "json" {
		t.Errorf("Expected format 'json', got '%v'", params["format"])
	}

	if params["system"] != "You are a helpful assistant." {
		t.Errorf("Expected system prompt, got '%v'", params["system"])
	}
}

func TestOllamaInputAdapter_ValidateRequest(t *testing.T) {
	adapter := NewOllamaInputAdapter("test-key")

	// Valid request
	validReq := &types.ModelRequest{
		Model: "test-model",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  1000,
			"temperature": 0.5,
		},
	}

	err := adapter.ValidateRequest(validReq)
	if err != nil {
		t.Errorf("Expected valid request to pass validation, got error: %v", err)
	}

	// Invalid request - no model
	invalidReq := &types.ModelRequest{
		Model: "",
		Messages: []interface{}{
			types.OllamaRequest{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
	}

	err = adapter.ValidateRequest(invalidReq)
	if err == nil {
		t.Error("Expected invalid request (no model) to fail validation")
	}

	// Invalid request - no messages
	invalidReq2 := &types.ModelRequest{
		Model:    "test-model",
		Messages: []interface{}{},
	}

	err = adapter.ValidateRequest(invalidReq2)
	if err == nil {
		t.Error("Expected invalid request (no messages) to fail validation")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
