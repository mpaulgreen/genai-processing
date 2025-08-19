package errors

import (
	"fmt"
	"strings"
)

// TemplateErrorCode defines types of template validation errors
type TemplateErrorCode string

const (
	// Syntax errors
	ErrorMalformedTemplate TemplateErrorCode = "MALFORMED_TEMPLATE"
	ErrorUnbalancedBraces  TemplateErrorCode = "UNBALANCED_BRACES"
	ErrorInvalidSyntax     TemplateErrorCode = "INVALID_SYNTAX"
	
	// Placeholder errors
	ErrorMissingPlaceholder TemplateErrorCode = "MISSING_PLACEHOLDER"
	ErrorInvalidPlaceholder TemplateErrorCode = "INVALID_PLACEHOLDER"
	ErrorUnknownPlaceholder TemplateErrorCode = "UNKNOWN_PLACEHOLDER"
	
	// Structure errors
	ErrorEmptyTemplate     TemplateErrorCode = "EMPTY_TEMPLATE"
	ErrorInvalidStructure  TemplateErrorCode = "INVALID_STRUCTURE"
	ErrorCircularReference TemplateErrorCode = "CIRCULAR_REFERENCE"
)

// TemplateError represents a template validation error
type TemplateError struct {
	Code        TemplateErrorCode `json:"code"`
	Message     string            `json:"message"`
	Position    int               `json:"position,omitempty"`
	Context     string            `json:"context,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
}

func (e *TemplateError) Error() string {
	if e.Position > 0 {
		return fmt.Sprintf("%s at position %d: %s", e.Code, e.Position, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// TemplateValidationResult contains the result of template validation
type TemplateValidationResult struct {
	IsValid     bool             `json:"is_valid"`
	Errors      []*TemplateError `json:"errors,omitempty"`
	Warnings    []*TemplateError `json:"warnings,omitempty"`
	Placeholders []string        `json:"placeholders,omitempty"`
}

// AddError adds a validation error to the result
func (r *TemplateValidationResult) AddError(code TemplateErrorCode, message string, position int, context string) {
	r.IsValid = false
	r.Errors = append(r.Errors, &TemplateError{
		Code:     code,
		Message:  message,
		Position: position,
		Context:  context,
	})
}

// AddWarning adds a validation warning to the result
func (r *TemplateValidationResult) AddWarning(code TemplateErrorCode, message string, position int, context string) {
	r.Warnings = append(r.Warnings, &TemplateError{
		Code:     code,
		Message:  message,
		Position: position,
		Context:  context,
	})
}

// AddSuggestion adds a suggestion to the last error
func (r *TemplateValidationResult) AddSuggestion(suggestion string) {
	if len(r.Errors) > 0 {
		lastError := r.Errors[len(r.Errors)-1]
		lastError.Suggestions = append(lastError.Suggestions, suggestion)
	}
}

// GetErrorSummary returns a formatted summary of all errors
func (r *TemplateValidationResult) GetErrorSummary() string {
	if r.IsValid {
		return "Template is valid"
	}
	
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Template validation failed with %d error(s)", len(r.Errors)))
	
	for i, err := range r.Errors {
		summary.WriteString(fmt.Sprintf("\n%d. %s", i+1, err.Error()))
		if len(err.Suggestions) > 0 {
			summary.WriteString(fmt.Sprintf("\n   Suggestions: %s", strings.Join(err.Suggestions, ", ")))
		}
	}
	
	if len(r.Warnings) > 0 {
		summary.WriteString(fmt.Sprintf("\n\nWarnings (%d):", len(r.Warnings)))
		for i, warn := range r.Warnings {
			summary.WriteString(fmt.Sprintf("\n%d. %s", i+1, warn.Error()))
		}
	}
	
	return summary.String()
}

// Template error constructors for common cases
func NewMalformedTemplateError(message string, position int) *TemplateError {
	return &TemplateError{
		Code:     ErrorMalformedTemplate,
		Message:  message,
		Position: position,
	}
}

func NewMissingPlaceholderError(placeholder string) *TemplateError {
	return &TemplateError{
		Code:    ErrorMissingPlaceholder,
		Message: fmt.Sprintf("Required placeholder '%s' is missing", placeholder),
		Suggestions: []string{
			fmt.Sprintf("Add {%s} to your template", placeholder),
			"Check placeholder spelling and syntax",
		},
	}
}

func NewInvalidPlaceholderError(placeholder string, position int) *TemplateError {
	return &TemplateError{
		Code:     ErrorInvalidPlaceholder,
		Message:  fmt.Sprintf("Invalid placeholder '%s'", placeholder),
		Position: position,
		Suggestions: []string{
			"Use only alphanumeric characters and underscores",
			"Ensure placeholder is enclosed in curly braces",
		},
	}
}

func NewUnbalancedBracesError(position int, context string) *TemplateError {
	return &TemplateError{
		Code:     ErrorUnbalancedBraces,
		Message:  "Unbalanced curly braces detected",
		Position: position,
		Context:  context,
		Suggestions: []string{
			"Ensure all opening braces '{' have matching closing braces '}'",
			"Check for escaped braces if literal braces are needed",
		},
	}
}

func NewUnknownPlaceholderError(placeholder string, position int, validPlaceholders []string) *TemplateError {
	return &TemplateError{
		Code:     ErrorUnknownPlaceholder,
		Message:  fmt.Sprintf("Unknown placeholder '%s'", placeholder),
		Position: position,
		Suggestions: append([]string{
			fmt.Sprintf("Valid placeholders are: %s", strings.Join(validPlaceholders, ", ")),
		}, validPlaceholders...),
	}
}