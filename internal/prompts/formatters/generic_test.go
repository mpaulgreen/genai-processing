package formatters

import (
	"strings"
	"testing"

	"genai-processing/internal/prompts/validation"
	"genai-processing/pkg/types"
)

func TestGenericFormatter_FormatSystemPrompt(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		systemPrompt   string
		expected       string
		expectError    bool
	}{
		{
			name:         "basic system prompt",
			template:     "",
			systemPrompt: "You are an audit specialist",
			expected:     "You are an audit specialist",
			expectError:  false,
		},
		{
			name:         "empty system prompt",
			template:     "",
			systemPrompt: "",
			expected:     "",
			expectError:  false,
		},
		{
			name:         "multiline system prompt",
			template:     "",
			systemPrompt: "Line 1\nLine 2\nLine 3",
			expected:     "Line 1\nLine 2\nLine 3",
			expectError:  false,
		},
		{
			name:         "system prompt with unicode",
			template:     "",
			systemPrompt: "Unicode: 你好 🚀 ñáéíóú",
			expected:     "Unicode: 你好 🚀 ñáéíóú",
			expectError:  false,
		},
		{
			name:         "system prompt too long error",
			template:     "",
			systemPrompt: strings.Repeat("a", 50001),
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			result, err := formatter.FormatSystemPrompt(tt.systemPrompt)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenericFormatter_FormatExamples(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		examples    []types.Example
		expected    string
		expectError bool
	}{
		{
			name:     "empty examples",
			template: "",
			examples: []types.Example{},
			expected: "",
		},
		{
			name:     "single example",
			template: "",
			examples: []types.Example{
				{Input: "Test query", Output: `{"result": "test"}`},
			},
			expected: "Input: Test query\nOutput: {\"result\": \"test\"}\n",
		},
		{
			name:     "multiple examples",
			template: "",
			examples: []types.Example{
				{Input: "Query 1", Output: "Result 1"},
				{Input: "Query 2", Output: "Result 2"},
				{Input: "Query 3", Output: "Result 3"},
			},
			expected: "Input: Query 1\nOutput: Result 1\n\nInput: Query 2\nOutput: Result 2\n\nInput: Query 3\nOutput: Result 3\n",
		},
		{
			name:     "examples with complex JSON",
			template: "",
			examples: []types.Example{
				{
					Input:  "Who deleted CRDs?",
					Output: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions"}`,
				},
			},
			expected: "Input: Who deleted CRDs?\nOutput: {\"log_source\": \"kube-apiserver\", \"verb\": \"delete\", \"resource\": \"customresourcedefinitions\"}\n",
		},
		{
			name:     "examples with empty fields",
			template: "",
			examples: []types.Example{
				{Input: "", Output: ""},
				{Input: "Non-empty", Output: ""},
				{Input: "", Output: "Non-empty"},
			},
			expected: "Input: \nOutput: \n\nInput: Non-empty\nOutput: \n\nInput: \nOutput: Non-empty\n",
		},
		{
			name:        "too many examples error",
			template:    "",
			examples:    make([]types.Example, 101),
			expected:    "",
			expectError: true,
		},
		{
			name:     "example too long error",
			template: "",
			examples: []types.Example{
				{Input: strings.Repeat("a", 10001), Output: "test"},
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			result, err := formatter.FormatExamples(tt.examples)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestGenericFormatter_FormatComplete(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		systemPrompt string
		examples     []types.Example
		query        string
		expected     string
		expectError  bool
	}{
		{
			name:         "empty template fallback - all fields",
			template:     "",
			systemPrompt: "You are a specialist",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "Who deleted the CRD?",
			expected:     "You are a specialist\n\nExamples:\nInput: test\nOutput: result\n\n\nQuery: Who deleted the CRD?\n\nJSON Response:",
		},
		{
			name:         "empty template fallback - no examples",
			template:     "",
			systemPrompt: "System prompt",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "System prompt\n\nQuery: test query\n\nJSON Response:",
		},
		{
			name:         "empty template fallback - empty system prompt",
			template:     "",
			systemPrompt: "",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "test query",
			expected:     "Examples:\nInput: test\nOutput: result\n\n\nQuery: test query\n\nJSON Response:",
		},
		{
			name:         "whitespace template fallback",
			template:     "   \n\t  ",
			systemPrompt: "System prompt",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "System prompt\n\nQuery: test query\n\nJSON Response:",
		},
		{
			name:         "custom template with all placeholders",
			template:     "SYSTEM: {system_prompt}\nEXAMPLES: {examples}\nQUERY: {query}",
			systemPrompt: "Test system",
			examples:     []types.Example{{Input: "in", Output: "out"}},
			query:        "test query",
			expected:     "SYSTEM: Test system\nEXAMPLES: Input: in\nOutput: out\n\nQUERY: test query",
		},
		{
			name:         "template without placeholders - uses fallback",
			template:     "Static template with no substitutions",
			systemPrompt: "Test system",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "Test system\n\nQuery: test query\n\nJSON Response:",
		},
		{
			name:         "template with repeated placeholders - invalid template uses fallback",
			template:     "{system_prompt} - {query} - {system_prompt}",
			systemPrompt: "SYSTEM",
			examples:     []types.Example{},
			query:        "QUERY",
			expected:     "SYSTEM\n\nQuery: QUERY\n\nJSON Response:",
		},
		{
			name:         "valid template with all required placeholders",
			template:     "S:{system_prompt} E:{examples} Q:{query}",
			systemPrompt: "test",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "S:test E: Q:test query",
		},
		{
			name:         "whitespace system prompt handling",
			template:     "",
			systemPrompt: "   \n\t  ",
			examples:     []types.Example{},
			query:        "test",
			expected:     "Query: test\n\nJSON Response:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			result, err := formatter.FormatComplete(tt.systemPrompt, tt.examples, tt.query)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestGenericFormatter_DefaultStructure(t *testing.T) {
	formatter := NewGenericFormatter("")
	
	result, err := formatter.FormatComplete(
		"System prompt",
		[]types.Example{{Input: "test", Output: "result"}},
		"query",
	)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify generic structure elements
	if !strings.Contains(result, "System prompt") {
		t.Error("missing system prompt in output")
	}
	if !strings.Contains(result, "Examples:") {
		t.Error("missing Examples: label")
	}
	if !strings.Contains(result, "Query: query") {
		t.Error("missing Query: label with content")
	}
	if !strings.Contains(result, "JSON Response:") {
		t.Error("missing JSON Response: prompt")
	}
}

func TestGenericFormatter_TemplateSubstitution(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		systemPrompt string
		examples     []types.Example
		query        string
		contains     []string
		notContains  []string
	}{
		{
			name:         "all substitutions",
			template:     "SYSTEM:{system_prompt}|EXAMPLES:{examples}|QUERY:{query}",
			systemPrompt: "test_system",
			examples:     []types.Example{{Input: "test_in", Output: "test_out"}},
			query:        "test_query",
			contains:     []string{"SYSTEM:test_system", "EXAMPLES:Input: test_in", "QUERY:test_query"},
			notContains:  []string{"{system_prompt}", "{examples}", "{query}"},
		},
		{
			name:         "partial substitutions - invalid template uses fallback",
			template:     "Only system: {system_prompt}",
			systemPrompt: "my_system",
			examples:     []types.Example{{Input: "test_in", Output: "test_out"}},
			query:        "my_query",
			contains:     []string{"my_system", "my_query", "JSON Response:"},
			notContains:  []string{"Only system:", "{system_prompt}"},
		},
		{
			name:         "no substitutions - invalid template uses fallback",
			template:     "No placeholders here",
			systemPrompt: "system_value",
			examples:     []types.Example{{Input: "input_val", Output: "output_val"}},
			query:        "query_value",
			contains:     []string{"system_value", "query_value", "JSON Response:"},
			notContains:  []string{"No placeholders here"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			result, err := formatter.FormatComplete(tt.systemPrompt, tt.examples, tt.query)
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got: %q", expected, result)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("expected result to NOT contain %q, got: %q", notExpected, result)
				}
			}
		})
	}
}

func TestGenericFormatter_InterfaceCompliance(t *testing.T) {
	// Test that GenericFormatter implements the PromptFormatter interface
	var formatter interface{} = NewGenericFormatter("")
	if _, ok := formatter.(interface {
		FormatSystemPrompt(string) (string, error)
		FormatExamples([]types.Example) (string, error)
		FormatComplete(string, []types.Example, string) (string, error)
	}); !ok {
		t.Error("GenericFormatter does not implement PromptFormatter interface")
	}
}

func TestNewGenericFormatter(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"empty template", ""},
		{"simple template", "test template"},
		{"template with placeholders", "{system_prompt} {examples} {query}"},
		{"template with special chars", "!@#$%^&*()"},
		{"multiline template", "Line1\nLine2\nLine3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			if formatter == nil {
				t.Error("NewGenericFormatter returned nil")
			}
			if formatter.template != tt.template {
				t.Errorf("expected template %q, got %q", tt.template, formatter.template)
			}
		})
	}
}

// Edge case tests
func TestGenericFormatter_EdgeCases(t *testing.T) {
	formatter := NewGenericFormatter("")

	t.Run("large input handling", func(t *testing.T) {
		largePrompt := strings.Repeat("A", 10000)
		largeQuery := strings.Repeat("B", 5000)
		largeExamples := []types.Example{
			{Input: strings.Repeat("C", 2000), Output: strings.Repeat("D", 2000)},
		}

		result, err := formatter.FormatComplete(largePrompt, largeExamples, largeQuery)
		if err != nil {
			t.Errorf("unexpected error with large input: %v", err)
		}
		if len(result) == 0 {
			t.Error("empty result with large input")
		}
	})

	t.Run("special characters handling", func(t *testing.T) {
		specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		result, err := formatter.FormatComplete(specialChars, []types.Example{}, specialChars)
		if err != nil {
			t.Errorf("unexpected error with special chars: %v", err)
		}
		if !strings.Contains(result, specialChars) {
			t.Error("special characters not preserved in output")
		}
	})
}

// Benchmark tests
func BenchmarkGenericFormatter_FormatSystemPrompt(b *testing.B) {
	formatter := NewGenericFormatter("")
	systemPrompt := "You are an OpenShift audit specialist. Convert natural language queries into structured JSON parameters for audit log analysis."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatSystemPrompt(systemPrompt)
	}
}

func BenchmarkGenericFormatter_FormatExamples(b *testing.B) {
	formatter := NewGenericFormatter("")
	examples := []types.Example{
		{Input: "Who deleted the customer CRD yesterday?", Output: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions"}`},
		{Input: "Show me all failed authentication attempts", Output: `{"log_source": "oauth-server", "auth_decision": "error"}`},
		{Input: "List admin actions by john.doe", Output: `{"log_source": "kube-apiserver", "user": "john.doe"}`},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatExamples(examples)
	}
}

func BenchmarkGenericFormatter_FormatComplete(b *testing.B) {
	formatter := NewGenericFormatter("")
	systemPrompt := "You are an OpenShift audit specialist"
	examples := []types.Example{
		{Input: "test query", Output: `{"result": "test"}`},
		{Input: "another query", Output: `{"result": "another"}`},
	}
	query := "Who deleted the CRD?"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatComplete(systemPrompt, examples, query)
	}
}

func BenchmarkGenericFormatter_FormatCompleteWithTemplate(b *testing.B) {
	formatter := NewGenericFormatter("SYSTEM: {system_prompt}\n\nEXAMPLES:\n{examples}\n\nQUERY: {query}")
	systemPrompt := "You are an OpenShift audit specialist"
	examples := []types.Example{
		{Input: "test query", Output: `{"result": "test"}`},
	}
	query := "Who deleted the CRD?"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatComplete(systemPrompt, examples, query)
	}
}

// Enhanced error handling tests
func TestGenericFormatter_EnhancedErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		systemPrompt  string
		examples      []types.Example
		query         string
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty query error",
			template:      "",
			systemPrompt:  "test",
			examples:      []types.Example{},
			query:         "",
			expectError:   true,
			errorContains: "query cannot be empty",
		},
		{
			name:          "query too long error",
			template:      "",
			systemPrompt:  "test",
			examples:      []types.Example{},
			query:         strings.Repeat("a", 10001),
			expectError:   true,
			errorContains: "query too long",
		},
		{
			name:          "too many examples error",
			template:      "",
			systemPrompt:  "test",
			examples:      make([]types.Example, 101),
			query:         "test query",
			expectError:   true,
			errorContains: "too many examples",
		},
		{
			name:         "example too long error",
			template:     "",
			systemPrompt: "test",
			examples: []types.Example{
				{Input: strings.Repeat("a", 10001), Output: "test"},
			},
			query:         "test query",
			expectError:   true,
			errorContains: "example 1 too long",
		},
		{
			name:         "valid input success",
			template:     "",
			systemPrompt: "test",
			examples: []types.Example{
				{Input: "test input", Output: "test output"},
			},
			query:       "test query",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			result, err := formatter.FormatComplete(tt.systemPrompt, tt.examples, tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

func TestGenericFormatter_TemplateValidation(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		expectValid    bool
		expectFallback bool
	}{
		{
			name:           "empty template valid",
			template:       "",
			expectValid:    true,
			expectFallback: true,
		},
		{
			name:           "valid template",
			template:       "{system_prompt}{examples}{query}",
			expectValid:    true,
			expectFallback: false,
		},
		{
			name:           "invalid template missing required",
			template:       "{system_prompt}{examples}",
			expectValid:    false,
			expectFallback: true,
		},
		{
			name:           "invalid template unbalanced braces",
			template:       "{system_prompt}{examples}{query",
			expectValid:    false,
			expectFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatter(tt.template)
			
			if formatter.IsValid() != tt.expectValid {
				t.Errorf("Expected IsValid()=%v, got %v", tt.expectValid, formatter.IsValid())
			}
			
			if !tt.expectValid && formatter.GetLastError() == nil {
				t.Error("Expected error for invalid template")
			}
			
			// Test that invalid templates still work (fallback)
			result, err := formatter.FormatComplete("test", []types.Example{}, "test query")
			if err != nil {
				t.Errorf("Invalid template should fallback gracefully, got error: %v", err)
			}
			if len(result) == 0 {
				t.Error("Expected non-empty result even for invalid template")
			}
			
			// Check if fallback structure is used
			if tt.expectFallback {
				if !strings.Contains(result, "JSON Response:") {
					t.Error("Expected fallback Generic structure")
				}
			}
		})
	}
}

func TestGenericFormatter_WithCustomValidator(t *testing.T) {
	customConfig := validation.PlaceholderConfig{
		Required: []string{"custom_prompt"},
		Optional: []string{"custom_optional"},
		Pattern:  `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	}
	validator := validation.NewTemplateValidatorWithConfig(customConfig)
	
	tests := []struct {
		name        string
		template    string
		expectValid bool
	}{
		{
			name:        "valid custom template",
			template:    "{custom_prompt}",
			expectValid: true,
		},
		{
			name:        "invalid custom template",
			template:    "{system_prompt}",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewGenericFormatterWithValidator(tt.template, validator)
			
			if formatter.IsValid() != tt.expectValid {
				t.Errorf("Expected IsValid()=%v, got %v", tt.expectValid, formatter.IsValid())
			}
		})
	}
}

func TestGenericFormatter_FallbackHandling(t *testing.T) {
	// Test with invalid template that should fallback
	formatter := NewGenericFormatter("{invalid_template}")
	
	result, err := formatter.FormatComplete("test system", []types.Example{
		{Input: "test", Output: "result"},
	}, "test query")
	
	if err != nil {
		t.Errorf("Fallback should not error: %v", err)
	}
	
	// Check fallback structure
	expectedParts := []string{
		"test system",
		"Examples:",
		"Input: test",
		"Output: result",
		"Query:",
		"test query",
		"JSON Response:",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Fallback result missing expected part: %q\nResult: %s", part, result)
		}
	}
}

func TestGenericFormatter_PlaceholderExtension(t *testing.T) {
	template := "{system_prompt}{examples}{query}{timestamp}{session_id}{model_name}{provider}"
	formatter := NewGenericFormatter(template)
	
	result, err := formatter.FormatComplete("test", []types.Example{}, "test query")
	if err != nil {
		t.Errorf("Extended placeholders should work: %v", err)
	}
	
	// Should not contain the placeholder braces since they're replaced with empty strings
	extendedPlaceholders := []string{"{timestamp}", "{session_id}", "{model_name}", "{provider}"}
	for _, placeholder := range extendedPlaceholders {
		if strings.Contains(result, placeholder) {
			t.Errorf("Result should not contain unreplaced placeholder: %q", placeholder)
		}
	}
}

func TestGenericFormatter_EmptySystemPromptFallback(t *testing.T) {
	formatter := NewGenericFormatter("")
	
	result, err := formatter.FormatComplete("", []types.Example{
		{Input: "test", Output: "result"},
	}, "test query")
	
	if err != nil {
		t.Errorf("Empty system prompt should not error: %v", err)
	}
	
	// Should still include examples and query sections
	expectedParts := []string{
		"Examples:",
		"Input: test",
		"Output: result",
		"Query: test query",
		"JSON Response:",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nResult: %s", part, result)
		}
	}
	
	// Should not have empty system prompt section with extra newlines
	if strings.Contains(result, "\n\n\nExamples:") {
		t.Error("Extra newlines found - empty system prompt not handled correctly")
	}
}