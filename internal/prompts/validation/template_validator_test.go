package validation

import (
	"strings"
	"testing"

	promptErrors "genai-processing/internal/prompts/errors"
)

func TestNewTemplateValidator(t *testing.T) {
	validator := NewTemplateValidator()
	if validator == nil {
		t.Fatal("NewTemplateValidator returned nil")
	}
	
	if validator.placeholderRegex == nil {
		t.Error("Placeholder regex not initialized")
	}
	
	// Check default configuration
	if len(validator.placeholderConfig.Required) != 3 {
		t.Errorf("Expected 3 required placeholders, got %d", len(validator.placeholderConfig.Required))
	}
}

func TestValidateTemplate_EmptyTemplate(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []string{"", "   ", "\n\t  ", "   \n  "}
	
	for _, template := range tests {
		result := validator.ValidateTemplate(template)
		if !result.IsValid {
			t.Errorf("Empty template should be valid, got: %s", result.GetErrorSummary())
		}
		if len(result.Errors) > 0 {
			t.Errorf("Empty template should have no errors, got %d", len(result.Errors))
		}
	}
}

func TestValidateTemplate_ValidTemplates(t *testing.T) {
	validator := NewTemplateValidator()
	
	validTemplates := []string{
		"{system_prompt}\n{examples}\n{query}",
		"System: {system_prompt}\nExamples: {examples}\nQuery: {query}",
		"<instructions>{system_prompt}</instructions><examples>{examples}</examples><query>{query}</query>",
		"{system_prompt} {examples} {query} {timestamp}",
	}
	
	for _, template := range validTemplates {
		t.Run("valid_template", func(t *testing.T) {
			result := validator.ValidateTemplate(template)
			if !result.IsValid {
				t.Errorf("Valid template failed validation: %s\nTemplate: %s", result.GetErrorSummary(), template)
			}
		})
	}
}

func TestValidateTemplate_UnbalancedBraces(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []struct {
		template    string
		description string
	}{
		{"{system_prompt", "missing closing brace"},
		{"system_prompt}", "missing opening brace"},
		{"{system_prompt}{examples}}", "extra closing brace"},
		{"{{system_prompt}{examples}{query}", "extra opening brace"},
		{"{system_prompt}}{examples}{query}", "misplaced closing brace"},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := validator.ValidateTemplate(tt.template)
			if result.IsValid {
				t.Errorf("Template with %s should be invalid", tt.description)
			}
			
			hasUnbalancedError := false
			for _, err := range result.Errors {
				if err.Code == promptErrors.ErrorUnbalancedBraces {
					hasUnbalancedError = true
					break
				}
			}
			if !hasUnbalancedError {
				t.Errorf("Expected unbalanced braces error for %s", tt.description)
			}
		})
	}
}

func TestValidateTemplate_MissingRequiredPlaceholders(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []struct {
		template         string
		missingPlaceholder string
	}{
		{"{examples}{query}", "system_prompt"},
		{"{system_prompt}{query}", "examples"},
		{"{system_prompt}{examples}", "query"},
		{"{system_prompt}", "examples,query"},
		{"No placeholders at all", "system_prompt,examples,query"},
	}
	
	for _, tt := range tests {
		t.Run("missing_"+tt.missingPlaceholder, func(t *testing.T) {
			result := validator.ValidateTemplate(tt.template)
			if result.IsValid {
				t.Errorf("Template missing required placeholders should be invalid")
			}
			
			missingPlaceholders := strings.Split(tt.missingPlaceholder, ",")
			for _, missing := range missingPlaceholders {
				foundError := false
				for _, err := range result.Errors {
					if err.Code == promptErrors.ErrorMissingPlaceholder && strings.Contains(err.Message, missing) {
						foundError = true
						break
					}
				}
				if !foundError {
					t.Errorf("Expected missing placeholder error for '%s'", missing)
				}
			}
		})
	}
}

func TestValidateTemplate_InvalidPlaceholders(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []struct {
		template    string
		description string
		errorCode   promptErrors.TemplateErrorCode
	}{
		{"{}", "empty placeholder", promptErrors.ErrorInvalidPlaceholder},
		{"{system prompt}", "space in placeholder", promptErrors.ErrorInvalidPlaceholder},
		{"{123invalid}", "starts with number", promptErrors.ErrorInvalidPlaceholder},
		{"{invalid-name}", "contains hyphen", promptErrors.ErrorInvalidPlaceholder},
		{"{invalid.name}", "contains dot", promptErrors.ErrorInvalidPlaceholder},
		{"{system_prompt}{examples}{query}{unknown_placeholder}", "unknown placeholder", promptErrors.ErrorUnknownPlaceholder},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := validator.ValidateTemplate(tt.template)
			if result.IsValid {
				t.Errorf("Template with %s should be invalid", tt.description)
			}
			
			foundExpectedError := false
			for _, err := range result.Errors {
				if err.Code == tt.errorCode {
					foundExpectedError = true
					break
				}
			}
			if !foundExpectedError {
				t.Errorf("Expected error code %s for %s, got errors: %v", tt.errorCode, tt.description, result.Errors)
			}
		})
	}
}

func TestValidateTemplate_StructureWarnings(t *testing.T) {
	validator := NewTemplateValidator()
	
	// Test very long template warning
	longTemplate := "{system_prompt}{examples}{query}" + strings.Repeat("a", 10000)
	result := validator.ValidateTemplate(longTemplate)
	
	foundLengthWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "very long") {
			foundLengthWarning = true
			break
		}
	}
	if !foundLengthWarning {
		t.Error("Expected warning about very long template")
	}
	
	// Test double braces warning
	doubleBraceTemplate := "{system_prompt}{{examples}}{query}"
	result = validator.ValidateTemplate(doubleBraceTemplate)
	
	foundDoubleBraceWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "Double braces") {
			foundDoubleBraceWarning = true
			break
		}
	}
	if !foundDoubleBraceWarning {
		t.Error("Expected warning about double braces")
	}
}

func TestValidateTemplate_ComplexScenarios(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []struct {
		name     string
		template string
		expectValid bool
		expectedErrors []promptErrors.TemplateErrorCode
	}{
		{
			name:     "multiple_errors",
			template: "{system_prompt}{invalid placeholder}missing_query",
			expectValid: false,
			expectedErrors: []promptErrors.TemplateErrorCode{
				promptErrors.ErrorInvalidPlaceholder,
				promptErrors.ErrorMissingPlaceholder,
				promptErrors.ErrorMissingPlaceholder, // missing examples and query
			},
		},
		{
			name:     "literal_text_valid",
			template: "{system_prompt} literal text here {examples}{query}",
			expectValid: true,
			expectedErrors: []promptErrors.TemplateErrorCode{},
		},
		{
			name:     "nested_structure",
			template: "Instructions: {system_prompt}\n\nExamples:\n{examples}\n\nQuery: {query}\n\nJSON Response:",
			expectValid: true,
			expectedErrors: []promptErrors.TemplateErrorCode{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTemplate(tt.template)
			
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v. Errors: %s", tt.expectValid, result.IsValid, result.GetErrorSummary())
			}
			
			if len(tt.expectedErrors) != len(result.Errors) {
				t.Errorf("Expected %d errors, got %d", len(tt.expectedErrors), len(result.Errors))
			}
			
			for _, expectedCode := range tt.expectedErrors {
				found := false
				for _, err := range result.Errors {
					if err.Code == expectedCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s not found in result", expectedCode)
				}
			}
		})
	}
}

func TestQuickValidate(t *testing.T) {
	validator := NewTemplateValidator()
	
	tests := []struct {
		template    string
		expectError bool
		description string
	}{
		{"", false, "empty template"},
		{"{system_prompt}{examples}{query}", false, "valid template"},
		{"{system_prompt}{examples}", true, "missing required placeholder"},
		{"{system_prompt", true, "unbalanced braces"},
		{"system_prompt}", true, "unbalanced braces"},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := validator.QuickValidate(tt.template)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

func TestCustomPlaceholderConfig(t *testing.T) {
	customConfig := PlaceholderConfig{
		Required: []string{"custom_prompt", "custom_data"},
		Optional: []string{"custom_optional"},
		Pattern:  `\{([a-zA-Z_][a-zA-Z0-9_]*)\}`,
	}
	
	validator := NewTemplateValidatorWithConfig(customConfig)
	
	// Test valid template with custom placeholders
	validTemplate := "{custom_prompt}{custom_data}"
	result := validator.ValidateTemplate(validTemplate)
	if !result.IsValid {
		t.Errorf("Valid custom template failed validation: %s", result.GetErrorSummary())
	}
	
	// Test invalid template missing custom required placeholders
	invalidTemplate := "{custom_prompt}"
	result = validator.ValidateTemplate(invalidTemplate)
	if result.IsValid {
		t.Error("Template missing custom required placeholder should be invalid")
	}
	
	// Test unknown placeholder
	unknownTemplate := "{custom_prompt}{custom_data}{unknown}"
	result = validator.ValidateTemplate(unknownTemplate)
	if result.IsValid {
		t.Error("Template with unknown placeholder should be invalid")
	}
}

func TestGetSupportedPlaceholders(t *testing.T) {
	validator := NewTemplateValidator()
	placeholders := validator.GetSupportedPlaceholders()
	
	if len(placeholders) == 0 {
		t.Error("Should return supported placeholders")
	}
	
	// Check that required placeholders are included
	defaultConfig := DefaultPlaceholderConfig()
	for _, required := range defaultConfig.Required {
		found := false
		for _, supported := range placeholders {
			if supported == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required placeholder '%s' not found in supported placeholders", required)
		}
	}
}

func TestTemplateValidationResult_Methods(t *testing.T) {
	result := &promptErrors.TemplateValidationResult{IsValid: true}
	
	// Test AddError
	result.AddError(promptErrors.ErrorMissingPlaceholder, "test error", 10, "context")
	if result.IsValid {
		t.Error("AddError should set IsValid to false")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	
	// Test AddWarning
	result.AddWarning(promptErrors.ErrorInvalidStructure, "test warning", 20, "context")
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
	
	// Test AddSuggestion
	result.AddSuggestion("test suggestion")
	if len(result.Errors[0].Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(result.Errors[0].Suggestions))
	}
	
	// Test GetErrorSummary
	summary := result.GetErrorSummary()
	if !strings.Contains(summary, "test error") {
		t.Error("Error summary should contain error message")
	}
	if !strings.Contains(summary, "test warning") {
		t.Error("Error summary should contain warning message")
	}
}

// Benchmark tests
func BenchmarkValidateTemplate_Simple(b *testing.B) {
	validator := NewTemplateValidator()
	template := "{system_prompt}{examples}{query}"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateTemplate(template)
	}
}

func BenchmarkValidateTemplate_Complex(b *testing.B) {
	validator := NewTemplateValidator()
	template := `
Instructions: {system_prompt}

Examples:
{examples}

Query: {query}

Additional context: {optional}
Session: {session_id}
Model: {model_name}
`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateTemplate(template)
	}
}

func BenchmarkQuickValidate(b *testing.B) {
	validator := NewTemplateValidator()
	template := "{system_prompt}{examples}{query}"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.QuickValidate(template)
	}
}