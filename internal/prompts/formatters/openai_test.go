package formatters

import (
	"strings"
	"testing"

	"genai-processing/internal/prompts/validation"
	"genai-processing/pkg/types"
)

func TestOpenAIFormatter_FormatSystemPrompt(t *testing.T) {
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
			systemPrompt: "You are an OpenAI assistant",
			expected:     "You are an OpenAI assistant",
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
			name:         "complex system prompt",
			template:     "",
			systemPrompt: "You are an OpenShift audit specialist.\nProcess queries carefully.\nReturn only JSON.",
			expected:     "You are an OpenShift audit specialist.\nProcess queries carefully.\nReturn only JSON.",
			expectError:  false,
		},
		{
			name:         "system prompt with formatting",
			template:     "",
			systemPrompt: "Instructions:\n1. Parse query\n2. Generate JSON\n3. Validate output",
			expected:     "Instructions:\n1. Parse query\n2. Generate JSON\n3. Validate output",
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
			formatter := NewOpenAIFormatter(tt.template)
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

func TestOpenAIFormatter_FormatExamples(t *testing.T) {
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
				{Input: "Who accessed secrets?", Output: `{"log_source": "kube-apiserver", "resource": "secrets"}`},
			},
			expected: "Input: Who accessed secrets?\nOutput: {\"log_source\": \"kube-apiserver\", \"resource\": \"secrets\"}\n",
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
			name:     "examples with JSON formatting",
			template: "",
			examples: []types.Example{
				{
					Input:  "Show authentication failures",
					Output: `{"log_source": "oauth-server", "auth_decision": "error", "limit": 20}`,
				},
			},
			expected: "Input: Show authentication failures\nOutput: {\"log_source\": \"oauth-server\", \"auth_decision\": \"error\", \"limit\": 20}\n",
		},
		{
			name:     "examples with newlines in content",
			template: "",
			examples: []types.Example{
				{
					Input:  "Multi\nline\ninput",
					Output: "Multi\nline\noutput",
				},
			},
			expected: "Input: Multi\nline\ninput\nOutput: Multi\nline\noutput\n",
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
			formatter := NewOpenAIFormatter(tt.template)
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

func TestOpenAIFormatter_FormatComplete(t *testing.T) {
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
			name:         "empty template fallback - full content",
			template:     "",
			systemPrompt: "You are an assistant",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "Who deleted the pod?",
			expected:     "You are an assistant\n\nExamples:\nInput: test\nOutput: result\n\n\nConvert this query to JSON: Who deleted the pod?",
		},
		{
			name:         "empty template fallback - no examples",
			template:     "",
			systemPrompt: "System instruction",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "System instruction\n\nConvert this query to JSON: test query",
		},
		{
			name:         "whitespace template fallback",
			template:     "   \n\t  ",
			systemPrompt: "System prompt",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "System prompt\n\nConvert this query to JSON: test query",
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
			template:     "Fixed template without any substitutions",
			systemPrompt: "Test System",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "test query",
			expected:     "Test System\n\nExamples:\nInput: test\nOutput: result\n\n\nConvert this query to JSON: test query",
		},
		{
			name:         "template with selective placeholders - invalid template uses fallback",
			template:     "Instructions: {system_prompt}\nQuery to process: {query}",
			systemPrompt: "Process carefully",
			examples:     []types.Example{{Input: "test", Output: "result"}},
			query:        "user query",
			expected:     "Process carefully\n\nExamples:\nInput: test\nOutput: result\n\n\nConvert this query to JSON: user query",
		},
		{
			name:         "empty query handling",
			template:     "",
			systemPrompt: "",
			examples:     []types.Example{},
			query:        "test query",
			expected:     "\n\nConvert this query to JSON: test query",
		},
		{
			name:         "complex template with repeated placeholders - invalid template uses fallback",
			template:     "{system_prompt} | {query} | {system_prompt}",
			systemPrompt: "SYSTEM",
			examples:     []types.Example{},
			query:        "QUERY",
			expected:     "SYSTEM\n\nConvert this query to JSON: QUERY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewOpenAIFormatter(tt.template)
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

func TestOpenAIFormatter_DefaultStructure(t *testing.T) {
	formatter := NewOpenAIFormatter("")
	
	result, err := formatter.FormatComplete(
		"System prompt",
		[]types.Example{{Input: "test", Output: "result"}},
		"query",
	)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify OpenAI-specific structure elements
	if !strings.Contains(result, "System prompt") {
		t.Error("missing system prompt in output")
	}
	if !strings.Contains(result, "Examples:") {
		t.Error("missing Examples: section")
	}
	if !strings.Contains(result, "Convert this query to JSON:") {
		t.Error("missing OpenAI-specific conversion prompt")
	}
	if !strings.Contains(result, "query") {
		t.Error("missing query content")
	}
}

func TestOpenAIFormatter_TemplateSubstitution(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		inputs      map[string]string
		contains    []string
		notContains []string
	}{
		{
			name:     "all substitutions",
			template: "SYSTEM: {system_prompt}\nEXAMPLES: {examples}\nQUERY: {query}",
			inputs: map[string]string{
				"system_prompt": "test_system",
				"query":         "test_query",
			},
			contains:    []string{"SYSTEM: test_system", "QUERY: test_query"},
			notContains: []string{"{system_prompt}", "{query}"},
		},
		{
			name:     "no placeholders - invalid template uses fallback",
			template: "Static content only",
			inputs: map[string]string{
				"system_prompt": "system_prompt_value",
				"query":         "query_value",
			},
			contains:    []string{"system_prompt_value", "query_value", "Convert this query to JSON:"},
			notContains: []string{"Static content only"},
		},
		{
			name:     "partial substitution - invalid template uses fallback",
			template: "Only system: {system_prompt}",
			inputs: map[string]string{
				"system_prompt": "my_system",
				"query":         "my_query",
			},
			contains:    []string{"my_system", "my_query", "Convert this query to JSON:"},
			notContains: []string{"Only system:", "{system_prompt}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewOpenAIFormatter(tt.template)
			result, err := formatter.FormatComplete(
				tt.inputs["system_prompt"],
				[]types.Example{{Input: "test", Output: "result"}},
				tt.inputs["query"],
			)
			
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

func TestOpenAIFormatter_SpecialCases(t *testing.T) {
	formatter := NewOpenAIFormatter("")

	t.Run("empty system prompt handling", func(t *testing.T) {
		result, err := formatter.FormatComplete(
			"",
			[]types.Example{{Input: "test", Output: "result"}},
			"query",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should still include examples and query even without system prompt
		if !strings.Contains(result, "Examples:") {
			t.Error("missing Examples section with empty system prompt")
		}
		if !strings.Contains(result, "Convert this query to JSON:") {
			t.Error("missing conversion prompt with empty system prompt")
		}
	})

	t.Run("query with JSON content", func(t *testing.T) {
		query := `Convert this: {"test": "value"} to proper format`
		result, err := formatter.FormatComplete("system", []types.Example{}, query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, query) {
			t.Error("JSON content in query was not preserved")
		}
	})

	t.Run("multiline inputs", func(t *testing.T) {
		systemPrompt := "Line 1\nLine 2\nLine 3"
		query := "Multi\nline\nquery"
		result, err := formatter.FormatComplete(systemPrompt, []types.Example{}, query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, systemPrompt) || !strings.Contains(result, query) {
			t.Error("multiline content not properly preserved")
		}
	})
}

func TestOpenAIFormatter_InterfaceCompliance(t *testing.T) {
	// Test that OpenAIFormatter implements the PromptFormatter interface
	var formatter interface{} = NewOpenAIFormatter("")
	if _, ok := formatter.(interface {
		FormatSystemPrompt(string) (string, error)
		FormatExamples([]types.Example) (string, error)
		FormatComplete(string, []types.Example, string) (string, error)
	}); !ok {
		t.Error("OpenAIFormatter does not implement PromptFormatter interface")
	}
}

func TestNewOpenAIFormatter(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"empty template", ""},
		{"simple template", "basic template"},
		{"template with placeholders", "{system_prompt} - {examples} - {query}"},
		{"multiline template", "Line1\nLine2\nLine3"},
		{"template with special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewOpenAIFormatter(tt.template)
			if formatter == nil {
				t.Error("NewOpenAIFormatter returned nil")
			}
			if formatter.template != tt.template {
				t.Errorf("expected template %q, got %q", tt.template, formatter.template)
			}
		})
	}
}

// Edge case and stress tests
func TestOpenAIFormatter_EdgeCases(t *testing.T) {
	formatter := NewOpenAIFormatter("")

	t.Run("very large inputs", func(t *testing.T) {
		largeSystemPrompt := strings.Repeat("System instruction. ", 1000)
		largeQuery := strings.Repeat("Query content. ", 500)
		largeExamples := []types.Example{
			{Input: strings.Repeat("Input ", 200), Output: strings.Repeat("Output ", 200)},
		}

		result, err := formatter.FormatComplete(largeSystemPrompt, largeExamples, largeQuery)
		if err != nil {
			t.Errorf("unexpected error with large inputs: %v", err)
		}
		if len(result) == 0 {
			t.Error("empty result with large inputs")
		}
	})

	t.Run("unicode and special characters", func(t *testing.T) {
		unicodeContent := "Unicode: 你好 🚀 ñáéíóú âêîôû αβγδε ♠♥♦♣"
		specialChars := "Special: !@#$%^&*()_+-=[]{}|;':\",./<>?"

		result, err := formatter.FormatComplete(unicodeContent, []types.Example{}, specialChars)
		if err != nil {
			t.Errorf("unexpected error with unicode/special chars: %v", err)
		}
		if !strings.Contains(result, unicodeContent) || !strings.Contains(result, specialChars) {
			t.Error("unicode or special characters not preserved")
		}
	})

	t.Run("whitespace preservation", func(t *testing.T) {
		whitespaceContent := "   Leading spaces\n\tTabs\n  \n   Trailing   "
		result, err := formatter.FormatComplete(whitespaceContent, []types.Example{}, "query")
		if err != nil {
			t.Errorf("unexpected error with whitespace: %v", err)
		}
		if !strings.Contains(result, whitespaceContent) {
			t.Error("whitespace not preserved in system prompt")
		}
	})
}

// Benchmark tests
func BenchmarkOpenAIFormatter_FormatSystemPrompt(b *testing.B) {
	formatter := NewOpenAIFormatter("")
	systemPrompt := "You are an OpenAI-powered OpenShift audit specialist. Convert natural language queries into structured JSON parameters for audit log analysis. Always respond with valid JSON only."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatSystemPrompt(systemPrompt)
	}
}

func BenchmarkOpenAIFormatter_FormatExamples(b *testing.B) {
	formatter := NewOpenAIFormatter("")
	examples := []types.Example{
		{Input: "Who deleted the customer CRD yesterday?", Output: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions", "resource_name_pattern": "customer", "timeframe": "yesterday"}`},
		{Input: "Show me all failed authentication attempts in the last hour", Output: `{"log_source": "oauth-server", "timeframe": "1_hour_ago", "auth_decision": "error"}`},
		{Input: "List all admin actions by user john.doe this week", Output: `{"log_source": "kube-apiserver", "user": "john.doe", "timeframe": "7_days_ago"}`},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatExamples(examples)
	}
}

func BenchmarkOpenAIFormatter_FormatComplete(b *testing.B) {
	formatter := NewOpenAIFormatter("")
	systemPrompt := "You are an OpenShift audit specialist. Convert natural language queries into structured JSON parameters."
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

func BenchmarkOpenAIFormatter_FormatCompleteWithTemplate(b *testing.B) {
	formatter := NewOpenAIFormatter("SYSTEM: {system_prompt}\n\nEXAMPLES:\n{examples}\n\nQUERY: {query}")
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
func TestOpenAIFormatter_EnhancedErrorHandling(t *testing.T) {
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
			formatter := NewOpenAIFormatter(tt.template)
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

func TestOpenAIFormatter_TemplateValidation(t *testing.T) {
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
			formatter := NewOpenAIFormatter(tt.template)
			
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
				if !strings.Contains(result, "Convert this query to JSON:") {
					t.Error("Expected fallback OpenAI structure")
				}
			}
		})
	}
}

func TestOpenAIFormatter_WithCustomValidator(t *testing.T) {
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
			formatter := NewOpenAIFormatterWithValidator(tt.template, validator)
			
			if formatter.IsValid() != tt.expectValid {
				t.Errorf("Expected IsValid()=%v, got %v", tt.expectValid, formatter.IsValid())
			}
		})
	}
}

func TestOpenAIFormatter_FallbackHandling(t *testing.T) {
	// Test with invalid template that should fallback
	formatter := NewOpenAIFormatter("{invalid_template}")
	
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
		"Convert this query to JSON:",
		"test query",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Fallback result missing expected part: %q\nResult: %s", part, result)
		}
	}
}

func TestOpenAIFormatter_PlaceholderExtension(t *testing.T) {
	template := "{system_prompt}{examples}{query}{timestamp}{session_id}{model_name}{provider}"
	formatter := NewOpenAIFormatter(template)
	
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