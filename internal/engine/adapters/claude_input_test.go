package adapters

import (
	"encoding/json"
	"strings"
	"testing"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/types"
)

func TestNewClaudeInputAdapter(t *testing.T) {
	apiKey := "test-api-key"
	adapter := NewClaudeInputAdapter(apiKey)

	if adapter.APIKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, adapter.APIKey)
	}

	if adapter.ModelName != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model name claude-3-5-sonnet-20241022, got %s", adapter.ModelName)
	}

	if adapter.MaxTokens != 4000 {
		t.Errorf("Expected max tokens 4000, got %d", adapter.MaxTokens)
	}

	if adapter.Temperature != 0.1 {
		t.Errorf("Expected temperature 0.1, got %f", adapter.Temperature)
	}

	if adapter.SystemPrompt == "" {
		t.Error("Expected system prompt to be set")
	}
}

func TestClaudeInputAdapter_AdaptRequest(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	// Test with valid request
	req := &types.InternalRequest{
		RequestID: "test-request-id",
		ProcessingRequest: types.ProcessingRequest{
			Query:     "Who deleted the customer CRD yesterday?",
			SessionID: "test-session",
		},
	}

	modelRequest, err := adapter.AdaptRequest(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if modelRequest.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model claude-3-5-sonnet-20241022, got %s", modelRequest.Model)
	}

	if len(modelRequest.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(modelRequest.Messages))
	}

	// Test with nil request
	_, err = adapter.AdaptRequest(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}

	inputAdapterErr, ok := err.(*errors.InputAdapterError)
	if !ok {
		t.Error("Expected InputAdapterError type")
	}

	if inputAdapterErr.ModelType != "claude" {
		t.Errorf("Expected model type claude, got %s", inputAdapterErr.ModelType)
	}
}

func TestClaudeInputAdapter_FormatPrompt(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	// Test with valid prompt
	prompt := "Who deleted the customer CRD yesterday?"
	examples := []types.Example{}

	formatted, err := adapter.FormatPrompt(prompt, examples)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check for XML structure
	if !contains(formatted, "<instructions>") {
		t.Error("Expected <instructions> tag in formatted prompt")
	}

	if !contains(formatted, "<query>") {
		t.Error("Expected <query> tag in formatted prompt")
	}

	if !contains(formatted, prompt) {
		t.Error("Expected original prompt in formatted output")
	}

	// Test with examples
	examples = []types.Example{
		{
			Input:  "Who deleted the customer CRD?",
			Output: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions"}`,
		},
	}

	formatted, err = adapter.FormatPrompt(prompt, examples)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !contains(formatted, "<examples>") {
		t.Error("Expected <examples> tag when examples are provided")
	}

	// Test with empty prompt
	_, err = adapter.FormatPrompt("", examples)
	if err == nil {
		t.Error("Expected error for empty prompt")
	}

	inputAdapterErr, ok := err.(*errors.InputAdapterError)
	if !ok {
		t.Error("Expected InputAdapterError type")
	}

	if inputAdapterErr.Message != "prompt cannot be empty" {
		t.Errorf("Expected error message 'prompt cannot be empty', got %s", inputAdapterErr.Message)
	}

	// Test with whitespace-only prompt
	_, err = adapter.FormatPrompt("   ", examples)
	if err == nil {
		t.Error("Expected error for whitespace-only prompt")
	}
}

func TestClaudeInputAdapter_GetAPIParameters(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	params := adapter.GetAPIParameters()

	// Check required parameters
	if params["api_key"] != "test-api-key" {
		t.Errorf("Expected api_key test-api-key, got %v", params["api_key"])
	}

	if params["endpoint"] != "https://api.anthropic.com/v1/messages" {
		t.Errorf("Expected endpoint https://api.anthropic.com/v1/messages, got %v", params["endpoint"])
	}

	if params["method"] != "POST" {
		t.Errorf("Expected method POST, got %v", params["method"])
	}

	if params["provider"] != "anthropic" {
		t.Errorf("Expected provider anthropic, got %v", params["provider"])
	}

	// Check headers
	headers, ok := params["headers"].(map[string]string)
	if !ok {
		t.Fatal("Expected headers to be map[string]string")
	}

	if headers["x-api-key"] != "test-api-key" {
		t.Errorf("Expected x-api-key test-api-key, got %s", headers["x-api-key"])
	}

	if headers["anthropic-version"] != "2023-06-01" {
		t.Errorf("Expected anthropic-version 2023-06-01, got %s", headers["anthropic-version"])
	}

	// Check model parameters
	if params["model_name"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model_name claude-3-5-sonnet-20241022, got %v", params["model_name"])
	}

	if params["max_tokens"] != 4000 {
		t.Errorf("Expected max_tokens 4000, got %v", params["max_tokens"])
	}

	if params["temperature"] != 0.1 {
		t.Errorf("Expected temperature 0.1, got %v", params["temperature"])
	}
}

func TestClaudeInputAdapter_ValidateRequest(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	// Test with valid request
	validRequest := &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			ClaudeMessage{
				Role:    "user",
				Content: "test content",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}

	err := adapter.ValidateRequest(validRequest)
	if err != nil {
		t.Errorf("Expected no error for valid request, got %v", err)
	}

	// Test with nil request
	err = adapter.ValidateRequest(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}

	inputAdapterErr, ok := err.(*errors.InputAdapterError)
	if !ok {
		t.Error("Expected InputAdapterError type")
	}

	if inputAdapterErr.Message != "model request cannot be nil" {
		t.Errorf("Expected error message 'model request cannot be nil', got %s", inputAdapterErr.Message)
	}

	// Test with empty model name
	invalidRequest := &types.ModelRequest{
		Model: "",
		Messages: []interface{}{
			ClaudeMessage{
				Role:    "user",
				Content: "test content",
			},
		},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for empty model name")
	}

	// Test with no messages
	invalidRequest = &types.ModelRequest{
		Model:    "claude-3-5-sonnet-20241022",
		Messages: []interface{}{},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for empty messages")
	}

	// Test with invalid max_tokens
	invalidRequest = &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			ClaudeMessage{
				Role:    "user",
				Content: "test content",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens": 5000, // Exceeds limit
		},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for invalid max_tokens")
	}

	// Test with invalid temperature
	invalidRequest = &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			ClaudeMessage{
				Role:    "user",
				Content: "test content",
			},
		},
		Parameters: map[string]interface{}{
			"temperature": 1.5, // Exceeds limit
		},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for invalid temperature")
	}
}

func TestClaudeInputAdapter_SetModelName(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")
	newModelName := "claude-3-haiku-20240307"

	adapter.SetModelName(newModelName)

	if adapter.ModelName != newModelName {
		t.Errorf("Expected model name %s, got %s", newModelName, adapter.ModelName)
	}
}

func TestClaudeInputAdapter_SetMaxTokens(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	// Test valid max tokens
	err := adapter.SetMaxTokens(2000)
	if err != nil {
		t.Errorf("Expected no error for valid max tokens, got %v", err)
	}

	if adapter.MaxTokens != 2000 {
		t.Errorf("Expected max tokens 2000, got %d", adapter.MaxTokens)
	}

	// Test invalid max tokens (too high)
	err = adapter.SetMaxTokens(5000)
	if err == nil {
		t.Error("Expected error for max tokens exceeding limit")
	} else {
		inputAdapterErr, ok := err.(*errors.InputAdapterError)
		if !ok {
			t.Errorf("Expected InputAdapterError type, got %T", err)
		} else if inputAdapterErr.Message == "" {
			t.Error("Expected non-empty error message")
		}
	}

	// Test invalid max tokens (zero)
	err = adapter.SetMaxTokens(0)
	if err == nil {
		t.Error("Expected error for zero max tokens")
	}

	// Test invalid max tokens (negative)
	err = adapter.SetMaxTokens(-1)
	if err == nil {
		t.Error("Expected error for negative max tokens")
	}
}

func TestClaudeInputAdapter_SetTemperature(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")

	// Test valid temperature
	err := adapter.SetTemperature(0.5)
	if err != nil {
		t.Errorf("Expected no error for valid temperature, got %v", err)
	}

	if adapter.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", adapter.Temperature)
	}

	// Test invalid temperature (too high)
	err = adapter.SetTemperature(1.5)
	if err == nil {
		t.Error("Expected error for temperature exceeding limit")
	} else {
		inputAdapterErr, ok := err.(*errors.InputAdapterError)
		if !ok {
			t.Errorf("Expected InputAdapterError type, got %T", err)
		} else if inputAdapterErr.Message == "" {
			t.Error("Expected non-empty error message")
		}
	}

	// Test invalid temperature (negative)
	err = adapter.SetTemperature(-0.1)
	if err == nil {
		t.Error("Expected error for negative temperature")
	}
}

func TestClaudeInputAdapter_SetSystemPrompt(t *testing.T) {
	adapter := NewClaudeInputAdapter("test-api-key")
	newPrompt := "Custom system prompt for testing"

	adapter.SetSystemPrompt(newPrompt)

	if adapter.SystemPrompt != newPrompt {
		t.Errorf("Expected system prompt %s, got %s", newPrompt, adapter.SystemPrompt)
	}
}

func TestGetDefaultSystemPrompt(t *testing.T) {
	prompt := getDefaultSystemPrompt()

	if prompt == "" {
		t.Error("Expected non-empty system prompt")
	}

	// Check for key components in the prompt
	if !contains(prompt, "OpenShift audit query specialist") {
		t.Error("Expected 'OpenShift audit query specialist' in system prompt")
	}

	if !contains(prompt, "log_source") {
		t.Error("Expected 'log_source' field in system prompt")
	}

	if !contains(prompt, "kube-apiserver") {
		t.Error("Expected 'kube-apiserver' in system prompt")
	}

	if !contains(prompt, "oauth-server") {
		t.Error("Expected 'oauth-server' in system prompt")
	}

	if !contains(prompt, "customresourcedefinitions") {
		t.Error("Expected 'customresourcedefinitions' in system prompt")
	}
}

func TestClaudeMessage_JSON(t *testing.T) {
	message := ClaudeMessage{
		Role:    "user",
		Content: "test content",
	}

	// Test JSON marshaling
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Expected no error marshaling ClaudeMessage, got %v", err)
	}

	expected := `{"role":"user","content":"test content"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}

	// Test JSON unmarshaling
	var unmarshaled ClaudeMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Expected no error unmarshaling ClaudeMessage, got %v", err)
	}

	if unmarshaled.Role != message.Role {
		t.Errorf("Expected role %s, got %s", message.Role, unmarshaled.Role)
	}

	if unmarshaled.Content != message.Content {
		t.Errorf("Expected content %s, got %s", message.Content, unmarshaled.Content)
	}
}

func TestClaudeRequest_JSON(t *testing.T) {
	request := ClaudeRequest{
		Model:       "claude-3-5-sonnet-20241022",
		Messages:    []ClaudeMessage{{Role: "user", Content: "test"}},
		MaxTokens:   4000,
		Temperature: 0.1,
		System:      "test system prompt",
	}

	// Test JSON marshaling
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Expected no error marshaling ClaudeRequest, got %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !contains(jsonStr, "claude-3-5-sonnet-20241022") {
		t.Error("Expected model name in JSON")
	}

	if !contains(jsonStr, "4000") {
		t.Error("Expected max_tokens in JSON")
	}

	if !contains(jsonStr, "0.1") {
		t.Error("Expected temperature in JSON")
	}

	// Test JSON unmarshaling
	var unmarshaled ClaudeRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Expected no error unmarshaling ClaudeRequest, got %v", err)
	}

	if unmarshaled.Model != request.Model {
		t.Errorf("Expected model %s, got %s", request.Model, unmarshaled.Model)
	}

	if unmarshaled.MaxTokens != request.MaxTokens {
		t.Errorf("Expected max tokens %d, got %d", request.MaxTokens, unmarshaled.MaxTokens)
	}

	if unmarshaled.Temperature != request.Temperature {
		t.Errorf("Expected temperature %f, got %f", request.Temperature, unmarshaled.Temperature)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
