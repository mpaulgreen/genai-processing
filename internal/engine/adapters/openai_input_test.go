package adapters

import (
	"encoding/json"
	"strings"
	"testing"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/types"
)

func TestNewOpenAIInputAdapter(t *testing.T) {
	apiKey := "test-api-key"
	adapter := NewOpenAIInputAdapter(apiKey)

	if adapter.APIKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, adapter.APIKey)
	}

	if adapter.ModelName != "gpt-4" {
		t.Errorf("Expected model name gpt-4, got %s", adapter.ModelName)
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

func TestOpenAIInputAdapter_AdaptRequest(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

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

	if modelRequest.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", modelRequest.Model)
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

	if inputAdapterErr.ModelType != "openai" {
		t.Errorf("Expected model type openai, got %s", inputAdapterErr.ModelType)
	}
}

func TestOpenAIInputAdapter_FormatPrompt(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

	// Test with valid prompt
	prompt := "Who deleted the customer CRD yesterday?"
	examples := []types.Example{}

	formatted, err := adapter.FormatPrompt(prompt, examples)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check for system/user message format
	if !contains(formatted, "Convert this query to JSON:") {
		t.Error("Expected 'Convert this query to JSON:' in formatted prompt")
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

	if !contains(formatted, "Examples:") {
		t.Error("Expected 'Examples:' when examples are provided")
	}

	if !contains(formatted, "Input: Who deleted the customer CRD?") {
		t.Error("Expected example input in formatted prompt")
	}

	if !contains(formatted, `Output: {"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions"}`) {
		t.Error("Expected example output in formatted prompt")
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

func TestOpenAIInputAdapter_GetAPIParameters(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

	params := adapter.GetAPIParameters()

	// Check required parameters
	if params["api_key"] != "test-api-key" {
		t.Errorf("Expected api_key test-api-key, got %v", params["api_key"])
	}

	if params["endpoint"] != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Expected endpoint https://api.openai.com/v1/chat/completions, got %v", params["endpoint"])
	}

	if params["method"] != "POST" {
		t.Errorf("Expected method POST, got %v", params["method"])
	}

	if params["provider"] != "openai" {
		t.Errorf("Expected provider openai, got %v", params["provider"])
	}

	// Check headers
	headers, ok := params["headers"].(map[string]string)
	if !ok {
		t.Fatal("Expected headers to be map[string]string")
	}

	if headers["Authorization"] != "Bearer test-api-key" {
		t.Errorf("Expected Authorization Bearer test-api-key, got %s", headers["Authorization"])
	}

	if headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", headers["Content-Type"])
	}

	// Check model parameters
	if params["model_name"] != "gpt-4" {
		t.Errorf("Expected model_name gpt-4, got %v", params["model_name"])
	}

	if params["max_tokens"] != 4000 {
		t.Errorf("Expected max_tokens 4000, got %v", params["max_tokens"])
	}

	if params["temperature"] != 0.1 {
		t.Errorf("Expected temperature 0.1, got %v", params["temperature"])
	}
}

func TestOpenAIInputAdapter_ValidateRequest(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

	// Test with valid request
	validRequest := &types.ModelRequest{
		Model: "gpt-4",
		Messages: []interface{}{
			OpenAIMessage{
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
			OpenAIMessage{
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
		Model:    "gpt-4",
		Messages: []interface{}{},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for empty messages")
	}

	// Test with invalid max_tokens
	invalidRequest = &types.ModelRequest{
		Model: "gpt-4",
		Messages: []interface{}{
			OpenAIMessage{
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
		Model: "gpt-4",
		Messages: []interface{}{
			OpenAIMessage{
				Role:    "user",
				Content: "test content",
			},
		},
		Parameters: map[string]interface{}{
			"temperature": 2.5, // Exceeds limit
		},
	}

	err = adapter.ValidateRequest(invalidRequest)
	if err == nil {
		t.Error("Expected error for invalid temperature")
	}
}

func TestOpenAIInputAdapter_SetModelName(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")
	newModelName := "gpt-4-turbo"

	adapter.SetModelName(newModelName)

	if adapter.ModelName != newModelName {
		t.Errorf("Expected model name %s, got %s", newModelName, adapter.ModelName)
	}
}

func TestOpenAIInputAdapter_SetMaxTokens(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

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

func TestOpenAIInputAdapter_SetTemperature(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")

	// Test valid temperature
	err := adapter.SetTemperature(0.5)
	if err != nil {
		t.Errorf("Expected no error for valid temperature, got %v", err)
	}

	if adapter.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", adapter.Temperature)
	}

	// Test invalid temperature (too high)
	err = adapter.SetTemperature(2.5)
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

func TestOpenAIInputAdapter_SetSystemPrompt(t *testing.T) {
	adapter := NewOpenAIInputAdapter("test-api-key")
	newPrompt := "Custom system prompt for testing"

	adapter.SetSystemPrompt(newPrompt)

	if adapter.SystemPrompt != newPrompt {
		t.Errorf("Expected system prompt %s, got %s", newPrompt, adapter.SystemPrompt)
	}
}

func TestGetOpenAIDefaultSystemPrompt(t *testing.T) {
	prompt := getOpenAIDefaultSystemPrompt()

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

func TestOpenAIMessage_JSON(t *testing.T) {
	message := OpenAIMessage{
		Role:    "user",
		Content: "test content",
	}

	// Test JSON marshaling
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Expected no error marshaling OpenAIMessage, got %v", err)
	}

	expected := `{"role":"user","content":"test content"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}

	// Test JSON unmarshaling
	var unmarshaled OpenAIMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Expected no error unmarshaling OpenAIMessage, got %v", err)
	}

	if unmarshaled.Role != message.Role {
		t.Errorf("Expected role %s, got %s", message.Role, unmarshaled.Role)
	}

	if unmarshaled.Content != message.Content {
		t.Errorf("Expected content %s, got %s", message.Content, unmarshaled.Content)
	}
}

func TestOpenAIRequest_JSON(t *testing.T) {
	request := OpenAIRequest{
		Model:       "gpt-4",
		Messages:    []OpenAIMessage{{Role: "user", Content: "test"}},
		MaxTokens:   4000,
		Temperature: 0.1,
	}

	// Test JSON marshaling
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Expected no error marshaling OpenAIRequest, got %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !contains(jsonStr, "gpt-4") {
		t.Error("Expected model name in JSON")
	}

	if !contains(jsonStr, "4000") {
		t.Error("Expected max_tokens in JSON")
	}

	if !contains(jsonStr, "0.1") {
		t.Error("Expected temperature in JSON")
	}

	// Test JSON unmarshaling
	var unmarshaled OpenAIRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Expected no error unmarshaling OpenAIRequest, got %v", err)
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

// Test comparing OpenAI and Claude adapters
func TestOpenAIvsClaudeAdapter(t *testing.T) {
	openAIAdapter := NewOpenAIInputAdapter("test-openai-key")
	claudeAdapter := NewClaudeInputAdapter("test-claude-key")

	// Test that both adapters have the same system prompt content
	openAIPrompt := openAIAdapter.SystemPrompt
	claudePrompt := claudeAdapter.SystemPrompt

	if openAIPrompt != claudePrompt {
		t.Error("Expected OpenAI and Claude adapters to have the same system prompt")
	}

	// Test that both adapters handle the same query correctly
	req := &types.InternalRequest{
		RequestID: "test-request-id",
		ProcessingRequest: types.ProcessingRequest{
			Query:     "Who deleted the customer CRD yesterday?",
			SessionID: "test-session",
		},
	}

	openAIRequest, err := openAIAdapter.AdaptRequest(req)
	if err != nil {
		t.Fatalf("OpenAI adapter failed: %v", err)
	}

	claudeRequest, err := claudeAdapter.AdaptRequest(req)
	if err != nil {
		t.Fatalf("Claude adapter failed: %v", err)
	}

	// Both should have the same model name pattern (though different actual names)
	if openAIRequest.Model == "" {
		t.Error("OpenAI request should have model name")
	}

	if claudeRequest.Model == "" {
		t.Error("Claude request should have model name")
	}

	// Both should have parameters
	if len(openAIRequest.Parameters) == 0 {
		t.Error("OpenAI request should have parameters")
	}

	if len(claudeRequest.Parameters) == 0 {
		t.Error("Claude request should have parameters")
	}

	// Check that OpenAI uses Bearer token auth while Claude uses x-api-key
	openAIHeaders := openAIRequest.Parameters["headers"].(map[string]string)
	claudeHeaders := claudeRequest.Parameters["headers"].(map[string]string)

	if !strings.HasPrefix(openAIHeaders["Authorization"], "Bearer ") {
		t.Error("OpenAI should use Bearer token authentication")
	}

	if claudeHeaders["x-api-key"] == "" {
		t.Error("Claude should use x-api-key authentication")
	}
}
