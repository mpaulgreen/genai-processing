package formatters

import (
	"strings"
	"testing"

	"genai-processing/internal/prompts/validation"
	"genai-processing/pkg/types"
)

func TestClaudeFormatter_FormatSystemPrompt(t *testing.T) {
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
			systemPrompt: "You are an OpenShift audit specialist",
			expected:     "You are an OpenShift audit specialist",
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
			name:         "system prompt with newlines",
			template:     "",
			systemPrompt: "Line 1\nLine 2\nLine 3",
			expected:     "Line 1\nLine 2\nLine 3",
			expectError:  false,
		},
		{
			name:         "system prompt with special characters",
			template:     "",
			systemPrompt: "Special chars: @#$%^&*()_+{}|:<>?",
			expected:     "Special chars: @#$%^&*()_+{}|:<>?",
			expectError:  false,
		},
		{
			name:         "system prompt too long",
			template:     "",
			systemPrompt: strings.Repeat("a", 50001),
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewClaudeFormatter(tt.template)
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

func TestClaudeFormatter_FormatExamples(t *testing.T) {
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
				{Input: "Who deleted the CRD?", Output: `{"verb": "delete"}`},
			},
			expected: "Input: Who deleted the CRD?\nOutput: {\"verb\": \"delete\"}\n",
		},
		{
			name:     "multiple examples",
			template: "",
			examples: []types.Example{
				{Input: "Query 1", Output: "Result 1"},
				{Input: "Query 2", Output: "Result 2"},
			},
			expected: "Input: Query 1\nOutput: Result 1\n\nInput: Query 2\nOutput: Result 2\n",
		},
		{
			name:     "examples with newlines",
			template: "",
			examples: []types.Example{
				{Input: "Multi\nLine\nInput", Output: "Multi\nLine\nOutput"},
			},
			expected: "Input: Multi\nLine\nInput\nOutput: Multi\nLine\nOutput\n",
		},
		{
			name:     "examples with empty strings",
			template: "",
			examples: []types.Example{
				{Input: "", Output: ""},
				{Input: "test", Output: ""},
			},
			expected: "Input: \nOutput: \n\nInput: test\nOutput: \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewClaudeFormatter(tt.template)
			result, err := formatter.FormatExamples(tt.examples)

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

func TestClaudeFormatter_FormatComplete(t *testing.T) {
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
			name:         "empty template fallback",
			template:     "",
			systemPrompt: "You are a specialist",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "Who deleted the CRD?",
			expected:     "<instructions>\nYou are a specialist\n</instructions>\n\n<examples>\nInput: test\nOutput: result\n</examples>\n\n<query>\nWho deleted the CRD?\n</query>\n\nJSON Response:",
		},
		{
			name:         "whitespace template fallback",
			template:     "   \n\t  ",
			systemPrompt: "System prompt",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "<instructions>\nSystem prompt\n</instructions>\n\n<query>\ntest query\n</query>\n\nJSON Response:",
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
			name:         "template missing placeholders",
			template:     "Static template without placeholders",
			systemPrompt: "Test system",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "<instructions>\nTest system\n</instructions>\n\n<query>\ntest query\n</query>\n\nJSON Response:",
		},
		{
			name:         "template with partial placeholders",
			template:     "System: {system_prompt}\nQuery: {query}",
			systemPrompt: "Test system",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "test query",
			expected:     "<instructions>\nTest system\n</instructions>\n\n<examples>\nInput: test\nOutput: result\n</examples>\n\n<query>\ntest query\n</query>\n\nJSON Response:",
		},
		{
			name:         "empty system prompt but valid query",
			template:     "",
			systemPrompt: "",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "<instructions>\n\n</instructions>\n\n<query>\ntest query\n</query>\n\nJSON Response:",
		},
		{
			name:         "complex template with multiple replacements",
			template:     "{system_prompt} | {examples} | {query} | {system_prompt}",
			systemPrompt: "SYSTEM",
			examples:     []types.Example{{Input: "Q", Output: "A"}},
			query:        "QUERY",
			expected:     "SYSTEM | Input: Q\nOutput: A\n | QUERY | SYSTEM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewClaudeFormatter(tt.template)
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

func TestClaudeFormatter_XMLStructure(t *testing.T) {
	formatter := NewClaudeFormatter("")
	
	result, err := formatter.FormatComplete(
		"System prompt",
		[]types.Example{{Input: "test", Output: "result"}},
		"query",
	)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify XML-like structure
	if !strings.Contains(result, "<instructions>") {
		t.Error("missing <instructions> tag")
	}
	if !strings.Contains(result, "</instructions>") {
		t.Error("missing </instructions> tag")
	}
	if !strings.Contains(result, "<examples>") {
		t.Error("missing <examples> tag")
	}
	if !strings.Contains(result, "</examples>") {
		t.Error("missing </examples> tag")
	}
	if !strings.Contains(result, "<query>") {
		t.Error("missing <query> tag")
	}
	if !strings.Contains(result, "</query>") {
		t.Error("missing </query> tag")
	}
	if !strings.Contains(result, "JSON Response:") {
		t.Error("missing JSON Response prompt")
	}
}

func TestClaudeFormatter_InterfaceCompliance(t *testing.T) {
	// Test that ClaudeFormatter implements the PromptFormatter interface
	var formatter interface{} = NewClaudeFormatter("")
	if _, ok := formatter.(interface {
		FormatSystemPrompt(string) (string, error)
		FormatExamples([]types.Example) (string, error)
		FormatComplete(string, []types.Example, string) (string, error)
	}); !ok {
		t.Error("ClaudeFormatter does not implement PromptFormatter interface")
	}
}

func TestNewClaudeFormatter(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"empty template", ""},
		{"simple template", "test template"},
		{"complex template", "{system_prompt}\n{examples}\n{query}"},
		{"template with special chars", "@#$%^&*()_+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewClaudeFormatter(tt.template)
			if formatter == nil {
				t.Error("NewClaudeFormatter returned nil")
			}
			if formatter.template != tt.template {
				t.Errorf("expected template %q, got %q", tt.template, formatter.template)
			}
		})
	}
}

// Benchmark tests
func BenchmarkClaudeFormatter_FormatSystemPrompt(b *testing.B) {
	formatter := NewClaudeFormatter("")
	systemPrompt := "You are an OpenShift audit specialist. Convert natural language queries into structured JSON parameters."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatSystemPrompt(systemPrompt)
	}
}

func BenchmarkClaudeFormatter_FormatExamples(b *testing.B) {
	formatter := NewClaudeFormatter("")
	examples := []types.Example{
		{Input: "Who deleted the customer CRD yesterday?", Output: `{"log_source": "kube-apiserver", "verb": "delete"}`},
		{Input: "Show me all failed authentication attempts", Output: `{"log_source": "oauth-server", "auth_decision": "error"}`},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatExamples(examples)
	}
}

func BenchmarkClaudeFormatter_FormatComplete(b *testing.B) {
	formatter := NewClaudeFormatter("")
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
func TestClaudeFormatter_EnhancedErrorHandling(t *testing.T) {
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
			formatter := NewClaudeFormatter(tt.template)
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

func TestClaudeFormatter_TemplateValidation(t *testing.T) {
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
			formatter := NewClaudeFormatter(tt.template)
			
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
				if !strings.Contains(result, "<instructions>") {
					t.Error("Expected fallback XML structure")
				}
			}
		})
	}
}

func TestClaudeFormatter_WithCustomValidator(t *testing.T) {
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
			formatter := NewClaudeFormatterWithValidator(tt.template, validator)
			
			if formatter.IsValid() != tt.expectValid {
				t.Errorf("Expected IsValid()=%v, got %v", tt.expectValid, formatter.IsValid())
			}
		})
	}
}

func TestClaudeFormatter_FallbackHandling(t *testing.T) {
	// Test with invalid template that should fallback
	formatter := NewClaudeFormatter("{invalid_template}")
	
	result, err := formatter.FormatComplete("test system", []types.Example{
		{Input: "test", Output: "result"},
	}, "test query")
	
	if err != nil {
		t.Errorf("Fallback should not error: %v", err)
	}
	
	// Check fallback structure
	expectedParts := []string{
		"<instructions>",
		"test system",
		"</instructions>",
		"<examples>",
		"Input: test",
		"Output: result",
		"</examples>",
		"<query>",
		"test query",
		"</query>",
		"JSON Response:",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Fallback result missing expected part: %q\nResult: %s", part, result)
		}
	}
}

func TestClaudeFormatter_PlaceholderExtension(t *testing.T) {
	template := "{system_prompt}{examples}{query}{timestamp}{session_id}{model_name}{provider}"
	formatter := NewClaudeFormatter(template)
	
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