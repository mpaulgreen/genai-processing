package errors

import (
	"encoding/json"
	"testing"
	"time"
)

// Test that all error types implement the error interface correctly
func TestErrorInterface(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name: "ProcessingError",
			error: &ProcessingError{
				Type:        "test",
				Message:     "test message",
				Component:   "test_component",
				Recoverable: true,
			},
			expected: "[test_component] test: test message",
		},
		{
			name: "InputAdapterError",
			error: &InputAdapterError{
				ProcessingError: ProcessingError{
					Type:        "input_adapter",
					Message:     "adapter failed",
					Component:   "adapter",
					Recoverable: false,
				},
				ModelType:   "gpt-4",
				AdapterType: "openai",
			},
			expected: "[adapter] Input adapter error for gpt-4 (openai): adapter failed",
		},
		{
			name: "ParsingError",
			error: &ParsingError{
				ProcessingError: ProcessingError{
					Type:        "parsing",
					Message:     "parse failed",
					Component:   "parser",
					Recoverable: false,
				},
				ParserType: "json",
				Confidence: 0.85,
			},
			expected: "[parser] Parsing error with json (confidence: 0.85): parse failed",
		},
		{
			name: "ValidationError with field",
			error: &ValidationError{
				ProcessingError: ProcessingError{
					Type:        "validation",
					Message:     "validation failed",
					Component:   "validator",
					Recoverable: true,
				},
				RuleName:  "required_field",
				FieldName: "query",
			},
			expected: "[validator] Validation error for field 'query' (rule: required_field): validation failed",
		},
		{
			name: "ValidationError without field",
			error: &ValidationError{
				ProcessingError: ProcessingError{
					Type:        "validation",
					Message:     "validation failed",
					Component:   "validator",
					Recoverable: true,
				},
				RuleName: "required_field",
			},
			expected: "[validator] Validation error (rule: required_field): validation failed",
		},
		{
			name: "ProviderError with status code",
			error: &ProviderError{
				ProcessingError: ProcessingError{
					Type:        "provider",
					Message:     "API failed",
					Component:   "provider",
					Recoverable: true,
				},
				ProviderName: "openai",
				StatusCode:   429,
			},
			expected: "[provider] Provider error from openai (HTTP 429): API failed",
		},
		{
			name: "ProviderError without status code",
			error: &ProviderError{
				ProcessingError: ProcessingError{
					Type:        "provider",
					Message:     "API failed",
					Component:   "provider",
					Recoverable: true,
				},
				ProviderName: "openai",
			},
			expected: "[provider] Provider error from openai: API failed",
		},
		{
			name: "ContextError",
			error: &ContextError{
				ProcessingError: ProcessingError{
					Type:        "context",
					Message:     "session failed",
					Component:   "context",
					Recoverable: true,
				},
				SessionID:   "session123",
				ContextType: "session",
				Operation:   "create",
			},
			expected: "[context] Context error in session operation (create): session failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.error.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Test factory functions
func TestFactoryFunctions(t *testing.T) {
	t.Run("NewProcessingError", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		if err.Type != "test" {
			t.Errorf("Expected Type 'test', got %s", err.Type)
		}
		if err.Message != "test message" {
			t.Errorf("Expected Message 'test message', got %s", err.Message)
		}
		if err.Component != "test_component" {
			t.Errorf("Expected Component 'test_component', got %s", err.Component)
		}
		if !err.Recoverable {
			t.Error("Expected Recoverable to be true")
		}
		if err.Details == nil {
			t.Error("Expected Details to be initialized")
		}
		if err.Suggestions == nil {
			t.Error("Expected Suggestions to be initialized")
		}
		if err.Timestamp.IsZero() {
			t.Error("Expected Timestamp to be set")
		}
	})

	t.Run("NewInputAdapterError", func(t *testing.T) {
		err := NewInputAdapterError("adapter failed", "adapter", "gpt-4", "openai", false)

		if err.Type != "input_adapter" {
			t.Errorf("Expected Type 'input_adapter', got %s", err.Type)
		}
		if err.ModelType != "gpt-4" {
			t.Errorf("Expected ModelType 'gpt-4', got %s", err.ModelType)
		}
		if err.AdapterType != "openai" {
			t.Errorf("Expected AdapterType 'openai', got %s", err.AdapterType)
		}
		if err.Recoverable {
			t.Error("Expected Recoverable to be false")
		}
	})

	t.Run("NewParsingError", func(t *testing.T) {
		err := NewParsingError("parse failed", "parser", "json", 0.85, "raw content")

		if err.Type != "parsing" {
			t.Errorf("Expected Type 'parsing', got %s", err.Type)
		}
		if err.ParserType != "json" {
			t.Errorf("Expected ParserType 'json', got %s", err.ParserType)
		}
		if err.Confidence != 0.85 {
			t.Errorf("Expected Confidence 0.85, got %f", err.Confidence)
		}
		if err.RawContent != "raw content" {
			t.Errorf("Expected RawContent 'raw content', got %s", err.RawContent)
		}
		if err.Recoverable {
			t.Error("Expected Recoverable to be false")
		}
	})

	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("validation failed", "validator", "required_field", "query", "string", "null")

		if err.Type != "validation" {
			t.Errorf("Expected Type 'validation', got %s", err.Type)
		}
		if err.RuleName != "required_field" {
			t.Errorf("Expected RuleName 'required_field', got %s", err.RuleName)
		}
		if err.FieldName != "query" {
			t.Errorf("Expected FieldName 'query', got %s", err.FieldName)
		}
		if err.ExpectedValue != "string" {
			t.Errorf("Expected ExpectedValue 'string', got %s", err.ExpectedValue)
		}
		if err.ActualValue != "null" {
			t.Errorf("Expected ActualValue 'null', got %s", err.ActualValue)
		}
		if !err.Recoverable {
			t.Error("Expected Recoverable to be true")
		}
	})

	t.Run("NewProviderError", func(t *testing.T) {
		err := NewProviderError("API failed", "provider", "openai", 429, "/v1/chat/completions", true)

		if err.Type != "provider" {
			t.Errorf("Expected Type 'provider', got %s", err.Type)
		}
		if err.ProviderName != "openai" {
			t.Errorf("Expected ProviderName 'openai', got %s", err.ProviderName)
		}
		if err.StatusCode != 429 {
			t.Errorf("Expected StatusCode 429, got %d", err.StatusCode)
		}
		if err.APIEndpoint != "/v1/chat/completions" {
			t.Errorf("Expected APIEndpoint '/v1/chat/completions', got %s", err.APIEndpoint)
		}
		if !err.Retryable {
			t.Error("Expected Retryable to be true")
		}
		if !err.Recoverable {
			t.Error("Expected Recoverable to be true")
		}
	})

	t.Run("NewContextError", func(t *testing.T) {
		err := NewContextError("session failed", "context", "session123", "session", "create")

		if err.Type != "context" {
			t.Errorf("Expected Type 'context', got %s", err.Type)
		}
		if err.SessionID != "session123" {
			t.Errorf("Expected SessionID 'session123', got %s", err.SessionID)
		}
		if err.ContextType != "session" {
			t.Errorf("Expected ContextType 'session', got %s", err.ContextType)
		}
		if err.Operation != "create" {
			t.Errorf("Expected Operation 'create', got %s", err.Operation)
		}
		if !err.Recoverable {
			t.Error("Expected Recoverable to be true")
		}
	})
}

// Test enhancement methods
func TestEnhancementMethods(t *testing.T) {
	t.Run("WithDetails", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		// Test adding details
		enhanced := err.WithDetails("key1", "value1").WithDetails("key2", 123)

		if len(enhanced.Details) != 2 {
			t.Errorf("Expected 2 details, got %d", len(enhanced.Details))
		}
		if enhanced.Details["key1"] != "value1" {
			t.Errorf("Expected key1='value1', got %v", enhanced.Details["key1"])
		}
		if enhanced.Details["key2"] != 123 {
			t.Errorf("Expected key2=123, got %v", enhanced.Details["key2"])
		}

		// Test that original error is modified (same pointer)
		if err != enhanced {
			t.Error("Expected same pointer returned")
		}
	})

	t.Run("WithDetails nil map", func(t *testing.T) {
		err := &ProcessingError{
			Type:        "test",
			Message:     "test message",
			Component:   "test_component",
			Recoverable: true,
			Details:     nil, // Explicitly nil
		}

		enhanced := err.WithDetails("key", "value")

		if enhanced.Details == nil {
			t.Error("Expected Details to be initialized")
		}
		if enhanced.Details["key"] != "value" {
			t.Errorf("Expected key='value', got %v", enhanced.Details["key"])
		}
	})

	t.Run("WithSuggestion", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		// Test adding suggestions
		enhanced := err.WithSuggestion("suggestion1").WithSuggestion("suggestion2")

		if len(enhanced.Suggestions) != 2 {
			t.Errorf("Expected 2 suggestions, got %d", len(enhanced.Suggestions))
		}
		if enhanced.Suggestions[0] != "suggestion1" {
			t.Errorf("Expected suggestion1, got %s", enhanced.Suggestions[0])
		}
		if enhanced.Suggestions[1] != "suggestion2" {
			t.Errorf("Expected suggestion2, got %s", enhanced.Suggestions[1])
		}

		// Test that original error is modified (same pointer)
		if err != enhanced {
			t.Error("Expected same pointer returned")
		}
	})

	t.Run("WithSuggestions", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		// Test adding multiple suggestions
		enhanced := err.WithSuggestions("suggestion1", "suggestion2", "suggestion3")

		if len(enhanced.Suggestions) != 3 {
			t.Errorf("Expected 3 suggestions, got %d", len(enhanced.Suggestions))
		}
		expected := []string{"suggestion1", "suggestion2", "suggestion3"}
		for i, suggestion := range expected {
			if enhanced.Suggestions[i] != suggestion {
				t.Errorf("Expected %s, got %s", suggestion, enhanced.Suggestions[i])
			}
		}

		// Test that original error is modified (same pointer)
		if err != enhanced {
			t.Error("Expected same pointer returned")
		}
	})

	t.Run("WithSuggestions empty", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		enhanced := err.WithSuggestions()

		if len(enhanced.Suggestions) != 0 {
			t.Errorf("Expected 0 suggestions, got %d", len(enhanced.Suggestions))
		}
	})
}

// Test helper methods
func TestHelperMethods(t *testing.T) {
	t.Run("IsRecoverable", func(t *testing.T) {
		recoverableErr := NewProcessingError("test", "test message", "test_component", true)
		nonRecoverableErr := NewProcessingError("test", "test message", "test_component", false)

		if !recoverableErr.IsRecoverable() {
			t.Error("Expected IsRecoverable() to return true")
		}
		if nonRecoverableErr.IsRecoverable() {
			t.Error("Expected IsRecoverable() to return false")
		}
	})

	t.Run("GetErrorType", func(t *testing.T) {
		err := NewProcessingError("test_type", "test message", "test_component", true)

		if err.GetErrorType() != "test_type" {
			t.Errorf("Expected GetErrorType() to return 'test_type', got %s", err.GetErrorType())
		}
	})

	t.Run("GetComponent", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "test_component", true)

		if err.GetComponent() != "test_component" {
			t.Errorf("Expected GetComponent() to return 'test_component', got %s", err.GetComponent())
		}
	})

	t.Run("GetTimestamp", func(t *testing.T) {
		before := time.Now()
		err := NewProcessingError("test", "test message", "test_component", true)
		after := time.Now()

		timestamp := err.GetTimestamp()
		if timestamp.Before(before) || timestamp.After(after) {
			t.Errorf("Expected timestamp to be between %v and %v, got %v", before, after, timestamp)
		}
	})
}

// Test JSON serialization
func TestJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		error    interface{}
		expected map[string]interface{}
	}{
		{
			name: "ProcessingError",
			error: &ProcessingError{
				Type:        "test",
				Message:     "test message",
				Component:   "test_component",
				Recoverable: true,
				Details:     map[string]interface{}{"key": "value"},
				Suggestions: []string{"suggestion1", "suggestion2"},
				Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: map[string]interface{}{
				"type":        "test",
				"message":     "test message",
				"component":   "test_component",
				"recoverable": true,
				"details":     map[string]interface{}{"key": "value"},
				"suggestions": []interface{}{"suggestion1", "suggestion2"},
				"timestamp":   "2023-01-01T12:00:00Z",
			},
		},
		{
			name: "InputAdapterError",
			error: &InputAdapterError{
				ProcessingError: ProcessingError{
					Type:        "input_adapter",
					Message:     "adapter failed",
					Component:   "adapter",
					Recoverable: false,
					Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Details:     make(map[string]interface{}),
					Suggestions: make([]string, 0),
				},
				ModelType:   "gpt-4",
				AdapterType: "openai",
			},
			expected: map[string]interface{}{
				"type":         "input_adapter",
				"message":      "adapter failed",
				"component":    "adapter",
				"recoverable":  false,
				"details":      map[string]interface{}{},
				"suggestions":  []interface{}{},
				"timestamp":    "2023-01-01T12:00:00Z",
				"model_type":   "gpt-4",
				"adapter_type": "openai",
			},
		},
		{
			name: "ParsingError",
			error: &ParsingError{
				ProcessingError: ProcessingError{
					Type:        "parsing",
					Message:     "parse failed",
					Component:   "parser",
					Recoverable: false,
					Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Details:     make(map[string]interface{}),
					Suggestions: make([]string, 0),
				},
				RawContent: "raw content",
				ParserType: "json",
				Confidence: 0.85,
			},
			expected: map[string]interface{}{
				"type":        "parsing",
				"message":     "parse failed",
				"component":   "parser",
				"recoverable": false,
				"details":     map[string]interface{}{},
				"suggestions": []interface{}{},
				"timestamp":   "2023-01-01T12:00:00Z",
				"raw_content": "raw content",
				"parser_type": "json",
				"confidence":  0.85,
			},
		},
		{
			name: "ValidationError",
			error: &ValidationError{
				ProcessingError: ProcessingError{
					Type:        "validation",
					Message:     "validation failed",
					Component:   "validator",
					Recoverable: true,
					Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Details:     make(map[string]interface{}),
					Suggestions: make([]string, 0),
				},
				RuleName:      "required_field",
				FieldName:     "query",
				ExpectedValue: "string",
				ActualValue:   "null",
			},
			expected: map[string]interface{}{
				"type":           "validation",
				"message":        "validation failed",
				"component":      "validator",
				"recoverable":    true,
				"details":        map[string]interface{}{},
				"suggestions":    []interface{}{},
				"timestamp":      "2023-01-01T12:00:00Z",
				"rule_name":      "required_field",
				"field_name":     "query",
				"expected_value": "string",
				"actual_value":   "null",
			},
		},
		{
			name: "ProviderError",
			error: &ProviderError{
				ProcessingError: ProcessingError{
					Type:        "provider",
					Message:     "API failed",
					Component:   "provider",
					Recoverable: true,
					Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Details:     make(map[string]interface{}),
					Suggestions: make([]string, 0),
				},
				ProviderName: "openai",
				StatusCode:   429,
				APIEndpoint:  "/v1/chat/completions",
				Retryable:    true,
			},
			expected: map[string]interface{}{
				"type":          "provider",
				"message":       "API failed",
				"component":     "provider",
				"recoverable":   true,
				"details":       map[string]interface{}{},
				"suggestions":   []interface{}{},
				"timestamp":     "2023-01-01T12:00:00Z",
				"provider_name": "openai",
				"status_code":   429.0,
				"api_endpoint":  "/v1/chat/completions",
				"retryable":     true,
			},
		},
		{
			name: "ContextError",
			error: &ContextError{
				ProcessingError: ProcessingError{
					Type:        "context",
					Message:     "session failed",
					Component:   "context",
					Recoverable: true,
					Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Details:     make(map[string]interface{}),
					Suggestions: make([]string, 0),
				},
				SessionID:   "session123",
				ContextType: "session",
				Operation:   "create",
			},
			expected: map[string]interface{}{
				"type":         "context",
				"message":      "session failed",
				"component":    "context",
				"recoverable":  true,
				"details":      map[string]interface{}{},
				"suggestions":  []interface{}{},
				"timestamp":    "2023-01-01T12:00:00Z",
				"session_id":   "session123",
				"context_type": "session",
				"operation":    "create",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.error)
			if err != nil {
				t.Fatalf("Failed to marshal error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// Check that all expected fields are present
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					// For empty maps and slices, they might be omitted in JSON
					if expectedMap, ok := expectedValue.(map[string]interface{}); ok && len(expectedMap) == 0 {
						// Empty map is acceptable to be omitted
						continue
					} else if expectedSlice, ok := expectedValue.([]interface{}); ok && len(expectedSlice) == 0 {
						// Empty slice is acceptable to be omitted
						continue
					} else {
						t.Errorf("Missing field: %s", key)
					}
				} else {
					// Handle map comparison specially
					if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
						if actualMap, ok := actualValue.(map[string]interface{}); ok {
							for mapKey, mapExpectedValue := range expectedMap {
								if mapActualValue, exists := actualMap[mapKey]; !exists {
									t.Errorf("Missing map field %s in %s", mapKey, key)
								} else if mapActualValue != mapExpectedValue {
									t.Errorf("Map field %s.%s: expected %v, got %v", key, mapKey, mapExpectedValue, mapActualValue)
								}
							}
						} else {
							t.Errorf("Field %s: expected map, got %T", key, actualValue)
						}
					} else if expectedSlice, ok := expectedValue.([]interface{}); ok {
						if actualSlice, ok := actualValue.([]interface{}); ok {
							if len(actualSlice) != len(expectedSlice) {
								t.Errorf("Field %s: expected %d items, got %d", key, len(expectedSlice), len(actualSlice))
							} else {
								for i, expectedItem := range expectedSlice {
									if actualItem := actualSlice[i]; actualItem != expectedItem {
										t.Errorf("Field %s[%d]: expected %v, got %v", key, i, expectedItem, actualItem)
									}
								}
							}
						} else {
							t.Errorf("Field %s: expected slice, got %T", key, actualValue)
						}
					} else if actualValue != expectedValue {
						t.Errorf("Field %s: expected %v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// Test error constants
func TestErrorConstants(t *testing.T) {
	t.Run("ErrorType constants", func(t *testing.T) {
		expectedTypes := map[string]string{
			"ErrorTypeInputAdapter": "input_adapter",
			"ErrorTypeParsing":      "parsing",
			"ErrorTypeValidation":   "validation",
			"ErrorTypeProvider":     "provider",
			"ErrorTypeContext":      "context",
			"ErrorTypeSystem":       "system",
		}

		for name, expected := range expectedTypes {
			switch name {
			case "ErrorTypeInputAdapter":
				if ErrorTypeInputAdapter != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeInputAdapter)
				}
			case "ErrorTypeParsing":
				if ErrorTypeParsing != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeParsing)
				}
			case "ErrorTypeValidation":
				if ErrorTypeValidation != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeValidation)
				}
			case "ErrorTypeProvider":
				if ErrorTypeProvider != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeProvider)
				}
			case "ErrorTypeContext":
				if ErrorTypeContext != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeContext)
				}
			case "ErrorTypeSystem":
				if ErrorTypeSystem != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ErrorTypeSystem)
				}
			}
		}
	})

	t.Run("Component constants", func(t *testing.T) {
		expectedComponents := map[string]string{
			"ComponentLLMEngine":    "llm_engine",
			"ComponentInputAdapter": "input_adapter",
			"ComponentParser":       "parser",
			"ComponentValidator":    "validator",
			"ComponentContext":      "context",
			"ComponentProvider":     "provider",
			"ComponentProcessor":    "processor",
		}

		for name, expected := range expectedComponents {
			switch name {
			case "ComponentLLMEngine":
				if ComponentLLMEngine != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentLLMEngine)
				}
			case "ComponentInputAdapter":
				if ComponentInputAdapter != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentInputAdapter)
				}
			case "ComponentParser":
				if ComponentParser != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentParser)
				}
			case "ComponentValidator":
				if ComponentValidator != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentValidator)
				}
			case "ComponentContext":
				if ComponentContext != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentContext)
				}
			case "ComponentProvider":
				if ComponentProvider != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentProvider)
				}
			case "ComponentProcessor":
				if ComponentProcessor != expected {
					t.Errorf("Expected %s to be %s, got %s", name, expected, ComponentProcessor)
				}
			}
		}
	})
}

// Test suggestion arrays
func TestSuggestionArrays(t *testing.T) {
	t.Run("SuggestionsForInputAdapter", func(t *testing.T) {
		expected := []string{
			"Check if the model type is supported",
			"Verify the adapter configuration",
			"Try using a different input adapter",
			"Review the request format requirements",
		}

		if len(SuggestionsForInputAdapter) != len(expected) {
			t.Errorf("Expected %d suggestions, got %d", len(expected), len(SuggestionsForInputAdapter))
		}

		for i, suggestion := range expected {
			if SuggestionsForInputAdapter[i] != suggestion {
				t.Errorf("Expected suggestion %d to be %s, got %s", i, suggestion, SuggestionsForInputAdapter[i])
			}
		}
	})

	t.Run("SuggestionsForParsing", func(t *testing.T) {
		expected := []string{
			"Check if the model response format is correct",
			"Try using a different parser",
			"Verify the response contains valid JSON",
			"Review the model's output format",
		}

		if len(SuggestionsForParsing) != len(expected) {
			t.Errorf("Expected %d suggestions, got %d", len(expected), len(SuggestionsForParsing))
		}

		for i, suggestion := range expected {
			if SuggestionsForParsing[i] != suggestion {
				t.Errorf("Expected suggestion %d to be %s, got %s", i, suggestion, SuggestionsForParsing[i])
			}
		}
	})

	t.Run("SuggestionsForValidation", func(t *testing.T) {
		expected := []string{
			"Review the validation rules",
			"Check the input parameters",
			"Verify the query structure",
			"Ensure all required fields are present",
		}

		if len(SuggestionsForValidation) != len(expected) {
			t.Errorf("Expected %d suggestions, got %d", len(expected), len(SuggestionsForValidation))
		}

		for i, suggestion := range expected {
			if SuggestionsForValidation[i] != suggestion {
				t.Errorf("Expected suggestion %d to be %s, got %s", i, suggestion, SuggestionsForValidation[i])
			}
		}
	})

	t.Run("SuggestionsForProvider", func(t *testing.T) {
		expected := []string{
			"Check the API endpoint configuration",
			"Verify authentication credentials",
			"Check network connectivity",
			"Review rate limiting settings",
		}

		if len(SuggestionsForProvider) != len(expected) {
			t.Errorf("Expected %d suggestions, got %d", len(expected), len(SuggestionsForProvider))
		}

		for i, suggestion := range expected {
			if SuggestionsForProvider[i] != suggestion {
				t.Errorf("Expected suggestion %d to be %s, got %s", i, suggestion, SuggestionsForProvider[i])
			}
		}
	})

	t.Run("SuggestionsForContext", func(t *testing.T) {
		expected := []string{
			"Check session configuration",
			"Verify context storage",
			"Review session lifecycle",
			"Check for session conflicts",
		}

		if len(SuggestionsForContext) != len(expected) {
			t.Errorf("Expected %d suggestions, got %d", len(expected), len(SuggestionsForContext))
		}

		for i, suggestion := range expected {
			if SuggestionsForContext[i] != suggestion {
				t.Errorf("Expected suggestion %d to be %s, got %s", i, suggestion, SuggestionsForContext[i])
			}
		}
	})
}

// Test error scenarios with table-driven tests
func TestErrorScenarios(t *testing.T) {
	tests := []struct {
		name                string
		setupError          func() error
		expectedType        string
		expectedRecoverable bool
		expectedMessage     string
	}{
		{
			name: "Input adapter error scenario",
			setupError: func() error {
				return NewInputAdapterError("Model not supported", ComponentInputAdapter, "unsupported-model", "custom", false)
			},
			expectedType:        "input_adapter",
			expectedRecoverable: false,
			expectedMessage:     "Model not supported",
		},
		{
			name: "Parsing error scenario",
			setupError: func() error {
				return NewParsingError("Invalid JSON format", ComponentParser, "json", 0.0, "invalid json")
			},
			expectedType:        "parsing",
			expectedRecoverable: false,
			expectedMessage:     "Invalid JSON format",
		},
		{
			name: "Validation error scenario",
			setupError: func() error {
				return NewValidationError("Required field missing", ComponentValidator, "required_field", "query", "non-empty string", "")
			},
			expectedType:        "validation",
			expectedRecoverable: true,
			expectedMessage:     "Required field missing",
		},
		{
			name: "Provider error scenario",
			setupError: func() error {
				return NewProviderError("Rate limit exceeded", ComponentProvider, "openai", 429, "/v1/chat/completions", true)
			},
			expectedType:        "provider",
			expectedRecoverable: true,
			expectedMessage:     "Rate limit exceeded",
		},
		{
			name: "Context error scenario",
			setupError: func() error {
				return NewContextError("Session expired", ComponentContext, "session123", "session", "validate")
			},
			expectedType:        "context",
			expectedRecoverable: true,
			expectedMessage:     "Session expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupError()

			// Test error interface
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}

			// Test type casting and specific fields
			switch e := err.(type) {
			case *InputAdapterError:
				if e.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, e.Type)
				}
				if e.Recoverable != tt.expectedRecoverable {
					t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, e.Recoverable)
				}
				if e.Message != tt.expectedMessage {
					t.Errorf("Expected message %s, got %s", tt.expectedMessage, e.Message)
				}
			case *ParsingError:
				if e.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, e.Type)
				}
				if e.Recoverable != tt.expectedRecoverable {
					t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, e.Recoverable)
				}
				if e.Message != tt.expectedMessage {
					t.Errorf("Expected message %s, got %s", tt.expectedMessage, e.Message)
				}
			case *ValidationError:
				if e.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, e.Type)
				}
				if e.Recoverable != tt.expectedRecoverable {
					t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, e.Recoverable)
				}
				if e.Message != tt.expectedMessage {
					t.Errorf("Expected message %s, got %s", tt.expectedMessage, e.Message)
				}
			case *ProviderError:
				if e.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, e.Type)
				}
				if e.Recoverable != tt.expectedRecoverable {
					t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, e.Recoverable)
				}
				if e.Message != tt.expectedMessage {
					t.Errorf("Expected message %s, got %s", tt.expectedMessage, e.Message)
				}
			case *ContextError:
				if e.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, e.Type)
				}
				if e.Recoverable != tt.expectedRecoverable {
					t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, e.Recoverable)
				}
				if e.Message != tt.expectedMessage {
					t.Errorf("Expected message %s, got %s", tt.expectedMessage, e.Message)
				}
			default:
				t.Errorf("Unexpected error type: %T", err)
			}
		})
	}
}

// Test error chaining and enhancement
func TestErrorChaining(t *testing.T) {
	t.Run("Complex error with all enhancements", func(t *testing.T) {
		err := NewProcessingError("test", "base error", "test_component", true)

		// Chain multiple enhancements
		enhanced := err.
			WithDetails("user_id", "12345").
			WithDetails("request_id", "req-67890").
			WithSuggestion("Try again later").
			WithSuggestions("Check your input", "Verify configuration").
			WithDetails("timestamp", time.Now().Unix())

		// Verify all enhancements are applied
		if len(enhanced.Details) != 3 {
			t.Errorf("Expected 3 details, got %d", len(enhanced.Details))
		}
		if len(enhanced.Suggestions) != 3 {
			t.Errorf("Expected 3 suggestions, got %d", len(enhanced.Suggestions))
		}
		if enhanced.Details["user_id"] != "12345" {
			t.Errorf("Expected user_id='12345', got %v", enhanced.Details["user_id"])
		}
		if enhanced.Details["request_id"] != "req-67890" {
			t.Errorf("Expected request_id='req-67890', got %v", enhanced.Details["request_id"])
		}
		if enhanced.Suggestions[0] != "Try again later" {
			t.Errorf("Expected first suggestion 'Try again later', got %s", enhanced.Suggestions[0])
		}
	})
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("Empty message", func(t *testing.T) {
		err := NewProcessingError("test", "", "test_component", true)
		expected := "[test_component] test: "
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("Empty component", func(t *testing.T) {
		err := NewProcessingError("test", "test message", "", true)
		expected := "[] test: test message"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("Empty type", func(t *testing.T) {
		err := NewProcessingError("", "test message", "test_component", true)
		expected := "[test_component] : test message"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("Nil details map", func(t *testing.T) {
		err := &ProcessingError{
			Type:        "test",
			Message:     "test message",
			Component:   "test_component",
			Recoverable: true,
			Details:     nil,
		}

		enhanced := err.WithDetails("key", "value")
		if enhanced.Details == nil {
			t.Error("Expected Details to be initialized after WithDetails")
		}
	})

	t.Run("Nil suggestions slice", func(t *testing.T) {
		err := &ProcessingError{
			Type:        "test",
			Message:     "test message",
			Component:   "test_component",
			Recoverable: true,
			Suggestions: nil,
		}

		enhanced := err.WithSuggestion("suggestion")
		if enhanced.Suggestions == nil {
			t.Error("Expected Suggestions to be initialized after WithSuggestion")
		}
	})
}
