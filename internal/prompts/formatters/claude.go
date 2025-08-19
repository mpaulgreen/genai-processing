package formatters

import (
	"fmt"
	"strings"

	"genai-processing/internal/prompts/validation"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// Ensure ClaudeFormatter implements interfaces.PromptFormatter
var _ interfaces.PromptFormatter = (*ClaudeFormatter)(nil)

type ClaudeFormatter struct {
	template  string
	validator *validation.TemplateValidator
	isValid   bool
	lastError error
}

// NewClaudeFormatter creates a new Claude formatter with template validation
func NewClaudeFormatter(template string) *ClaudeFormatter {
	formatter := &ClaudeFormatter{
		template:  template,
		validator: validation.NewTemplateValidator(),
		isValid:   true,
	}
	
	// Validate template if not empty (empty templates use fallback)
	if strings.TrimSpace(template) != "" {
		if err := formatter.validator.QuickValidate(template); err != nil {
			formatter.isValid = false
			formatter.lastError = err
		}
	}
	
	return formatter
}

// NewClaudeFormatterWithValidator creates a formatter with a custom validator
func NewClaudeFormatterWithValidator(template string, validator *validation.TemplateValidator) *ClaudeFormatter {
	formatter := &ClaudeFormatter{
		template:  template,
		validator: validator,
		isValid:   true,
	}
	
	if strings.TrimSpace(template) != "" {
		if err := validator.QuickValidate(template); err != nil {
			formatter.isValid = false
			formatter.lastError = err
		}
	}
	
	return formatter
}

// IsValid returns whether the formatter's template is valid
func (f *ClaudeFormatter) IsValid() bool {
	return f.isValid
}

// GetLastError returns the last validation error
func (f *ClaudeFormatter) GetLastError() error {
	return f.lastError
}

func (f *ClaudeFormatter) FormatSystemPrompt(systemPrompt string) (string, error) {
	// Basic input validation
	if len(systemPrompt) > 50000 { // Reasonable limit
		return "", fmt.Errorf("system prompt too long: %d characters (max 50000)", len(systemPrompt))
	}
	
	// Identity for Claude; template applies in FormatComplete
	return systemPrompt, nil
}

func (f *ClaudeFormatter) FormatExamples(examples []types.Example) (string, error) {
	if len(examples) == 0 {
		return "", nil
	}
	
	// Validate examples
	if len(examples) > 100 { // Reasonable limit
		return "", fmt.Errorf("too many examples: %d (max 100)", len(examples))
	}
	
	var b strings.Builder
	for i, ex := range examples {
		// Validate individual example
		if len(ex.Input) > 10000 || len(ex.Output) > 10000 {
			return "", fmt.Errorf("example %d too long (input: %d chars, output: %d chars, max 10000 each)", 
				i+1, len(ex.Input), len(ex.Output))
		}
		
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("Input: %s\n", ex.Input))
		b.WriteString(fmt.Sprintf("Output: %s\n", ex.Output))
	}
	return b.String(), nil
}

func (f *ClaudeFormatter) FormatComplete(systemPrompt string, examples []types.Example, query string) (string, error) {
	// Check if template was invalid during construction
	if !f.isValid && strings.TrimSpace(f.template) != "" {
		// Fallback to default formatting with warning
		return f.formatWithFallback(systemPrompt, examples, query)
	}
	
	// Validate inputs
	if len(query) == 0 {
		return "", fmt.Errorf("query cannot be empty")
	}
	if len(query) > 10000 {
		return "", fmt.Errorf("query too long: %d characters (max 10000)", len(query))
	}
	
	// If no template, use fallback XML-style layout
	if strings.TrimSpace(f.template) == "" {
		return f.formatWithFallback(systemPrompt, examples, query)
	}

	// Format examples with error handling
	exRendered, err := f.FormatExamples(examples)
	if err != nil {
		return "", fmt.Errorf("failed to format examples: %w", err)
	}
	
	// Apply template with replacements
	out := f.template
	out = strings.ReplaceAll(out, "{system_prompt}", systemPrompt)
	out = strings.ReplaceAll(out, "{examples}", exRendered)
	out = strings.ReplaceAll(out, "{query}", query)
	
	// Optional: Add additional placeholder replacements for extensibility
	out = strings.ReplaceAll(out, "{timestamp}", "")
	out = strings.ReplaceAll(out, "{session_id}", "")
	out = strings.ReplaceAll(out, "{model_name}", "")
	out = strings.ReplaceAll(out, "{provider}", "")
	
	return out, nil
}

// formatWithFallback provides the default Claude XML-style formatting
func (f *ClaudeFormatter) formatWithFallback(systemPrompt string, examples []types.Example, query string) (string, error) {
	var b strings.Builder
	
	// Build XML structure
	b.WriteString("<instructions>\n")
	b.WriteString(systemPrompt)
	b.WriteString("\n</instructions>\n\n")
	
	// Add examples if present
	if len(examples) > 0 {
		exStr, err := f.FormatExamples(examples)
		if err != nil {
			return "", fmt.Errorf("failed to format examples in fallback: %w", err)
		}
		if exStr != "" {
			b.WriteString("<examples>\n")
			b.WriteString(exStr)
			b.WriteString("</examples>\n\n")
		}
	}
	
	b.WriteString("<query>\n")
	b.WriteString(query)
	b.WriteString("\n</query>\n\nJSON Response:")
	
	return b.String(), nil
}
