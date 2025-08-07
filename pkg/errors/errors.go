package errors

import (
	"fmt"
	"time"
)

// ProcessingError represents the base error structure for all processing errors.
// It provides comprehensive error information including type, message, details,
// component identification, recoverability, suggestions, and timestamp.
type ProcessingError struct {
	// Type identifies the category of error (e.g., "validation", "parsing", "adapter")
	Type string `json:"type"`

	// Message provides a human-readable error description
	Message string `json:"message"`

	// Details contains additional error context as key-value pairs
	Details map[string]interface{} `json:"details,omitempty"`

	// Component identifies which component generated the error
	Component string `json:"component"`

	// Recoverable indicates whether the error can be recovered from
	Recoverable bool `json:"recoverable"`

	// Suggestions provides actionable recommendations for resolving the error
	Suggestions []string `json:"suggestions,omitempty"`

	// Timestamp records when the error occurred
	Timestamp time.Time `json:"timestamp"`
}

// Error implements the error interface
func (e *ProcessingError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Component, e.Type, e.Message)
}

// InputAdapterError represents errors that occur during input adaptation
// for model-specific request formatting.
type InputAdapterError struct {
	ProcessingError
	// ModelType identifies the model type that was being adapted for
	ModelType string `json:"model_type"`
	// AdapterType identifies the specific adapter that failed
	AdapterType string `json:"adapter_type"`
}

// Error implements the error interface
func (e *InputAdapterError) Error() string {
	return fmt.Sprintf("[%s] Input adapter error for %s (%s): %s",
		e.Component, e.ModelType, e.AdapterType, e.Message)
}

// ParsingError represents errors that occur during response parsing
// and extraction from LLM model outputs.
type ParsingError struct {
	ProcessingError
	// RawContent contains the raw response content that failed to parse
	RawContent string `json:"raw_content,omitempty"`
	// ParserType identifies the specific parser that failed
	ParserType string `json:"parser_type"`
	// Confidence indicates the parsing confidence level (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// Error implements the error interface
func (e *ParsingError) Error() string {
	return fmt.Sprintf("[%s] Parsing error with %s (confidence: %.2f): %s",
		e.Component, e.ParserType, e.Confidence, e.Message)
}

// ValidationError represents errors that occur during safety validation
// and rule checking of generated queries.
type ValidationError struct {
	ProcessingError
	// RuleName identifies the specific validation rule that failed
	RuleName string `json:"rule_name"`
	// FieldName identifies the field that failed validation
	FieldName string `json:"field_name,omitempty"`
	// ExpectedValue describes the expected value or pattern
	ExpectedValue string `json:"expected_value,omitempty"`
	// ActualValue describes the actual value that caused the failure
	ActualValue string `json:"actual_value,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.FieldName != "" {
		return fmt.Sprintf("[%s] Validation error for field '%s' (rule: %s): %s",
			e.Component, e.FieldName, e.RuleName, e.Message)
	}
	return fmt.Sprintf("[%s] Validation error (rule: %s): %s",
		e.Component, e.RuleName, e.Message)
}

// ProviderError represents errors that occur during LLM provider interactions
// such as API failures, authentication issues, or network problems.
type ProviderError struct {
	ProcessingError
	// ProviderName identifies the LLM provider that failed
	ProviderName string `json:"provider_name"`
	// StatusCode contains the HTTP status code if applicable
	StatusCode int `json:"status_code,omitempty"`
	// APIEndpoint identifies the API endpoint that was called
	APIEndpoint string `json:"api_endpoint,omitempty"`
	// Retryable indicates whether the error is retryable
	Retryable bool `json:"retryable"`
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("[%s] Provider error from %s (HTTP %d): %s",
			e.Component, e.ProviderName, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("[%s] Provider error from %s: %s",
		e.Component, e.ProviderName, e.Message)
}

// ContextError represents errors that occur during context management
// such as session handling, pronoun resolution, or history tracking.
type ContextError struct {
	ProcessingError
	// SessionID identifies the session that encountered the error
	SessionID string `json:"session_id,omitempty"`
	// ContextType identifies the type of context operation that failed
	ContextType string `json:"context_type"`
	// Operation identifies the specific operation that failed
	Operation string `json:"operation"`
}

// Error implements the error interface
func (e *ContextError) Error() string {
	return fmt.Sprintf("[%s] Context error in %s operation (%s): %s",
		e.Component, e.ContextType, e.Operation, e.Message)
}

// Error creation helper functions

// NewProcessingError creates a new ProcessingError with the given parameters
func NewProcessingError(errorType, message, component string, recoverable bool) *ProcessingError {
	return &ProcessingError{
		Type:        errorType,
		Message:     message,
		Component:   component,
		Recoverable: recoverable,
		Timestamp:   time.Now(),
		Details:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
	}
}

// NewInputAdapterError creates a new InputAdapterError
func NewInputAdapterError(message, component, modelType, adapterType string, recoverable bool) *InputAdapterError {
	return &InputAdapterError{
		ProcessingError: *NewProcessingError("input_adapter", message, component, recoverable),
		ModelType:       modelType,
		AdapterType:     adapterType,
	}
}

// NewParsingError creates a new ParsingError
func NewParsingError(message, component, parserType string, confidence float64, rawContent string) *ParsingError {
	return &ParsingError{
		ProcessingError: *NewProcessingError("parsing", message, component, false),
		ParserType:      parserType,
		Confidence:      confidence,
		RawContent:      rawContent,
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(message, component, ruleName, fieldName string, expectedValue, actualValue string) *ValidationError {
	return &ValidationError{
		ProcessingError: *NewProcessingError("validation", message, component, true),
		RuleName:        ruleName,
		FieldName:       fieldName,
		ExpectedValue:   expectedValue,
		ActualValue:     actualValue,
	}
}

// NewProviderError creates a new ProviderError
func NewProviderError(message, component, providerName string, statusCode int, apiEndpoint string, retryable bool) *ProviderError {
	return &ProviderError{
		ProcessingError: *NewProcessingError("provider", message, component, retryable),
		ProviderName:    providerName,
		StatusCode:      statusCode,
		APIEndpoint:     apiEndpoint,
		Retryable:       retryable,
	}
}

// NewContextError creates a new ContextError
func NewContextError(message, component, sessionID, contextType, operation string) *ContextError {
	return &ContextError{
		ProcessingError: *NewProcessingError("context", message, component, true),
		SessionID:       sessionID,
		ContextType:     contextType,
		Operation:       operation,
	}
}

// Error enhancement helper functions

// WithDetails adds key-value details to an error
func (e *ProcessingError) WithDetails(key string, value interface{}) *ProcessingError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithSuggestion adds a suggestion to an error
func (e *ProcessingError) WithSuggestion(suggestion string) *ProcessingError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple suggestions to an error
func (e *ProcessingError) WithSuggestions(suggestions ...string) *ProcessingError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// IsRecoverable checks if an error is recoverable
func (e *ProcessingError) IsRecoverable() bool {
	return e.Recoverable
}

// GetErrorType returns the error type
func (e *ProcessingError) GetErrorType() string {
	return e.Type
}

// GetComponent returns the component that generated the error
func (e *ProcessingError) GetComponent() string {
	return e.Component
}

// GetTimestamp returns when the error occurred
func (e *ProcessingError) GetTimestamp() time.Time {
	return e.Timestamp
}

// Common error types for consistency
const (
	ErrorTypeInputAdapter = "input_adapter"
	ErrorTypeParsing      = "parsing"
	ErrorTypeValidation   = "validation"
	ErrorTypeProvider     = "provider"
	ErrorTypeContext      = "context"
	ErrorTypeSystem       = "system"
)

// Common components for consistency
const (
	ComponentLLMEngine    = "llm_engine"
	ComponentInputAdapter = "input_adapter"
	ComponentParser       = "parser"
	ComponentValidator    = "validator"
	ComponentContext      = "context"
	ComponentProvider     = "provider"
	ComponentProcessor    = "processor"
)

// Common suggestions for different error types
var (
	SuggestionsForInputAdapter = []string{
		"Check if the model type is supported",
		"Verify the adapter configuration",
		"Try using a different input adapter",
		"Review the request format requirements",
	}

	SuggestionsForParsing = []string{
		"Check if the model response format is correct",
		"Try using a different parser",
		"Verify the response contains valid JSON",
		"Review the model's output format",
	}

	SuggestionsForValidation = []string{
		"Review the validation rules",
		"Check the input parameters",
		"Verify the query structure",
		"Ensure all required fields are present",
	}

	SuggestionsForProvider = []string{
		"Check the API endpoint configuration",
		"Verify authentication credentials",
		"Check network connectivity",
		"Review rate limiting settings",
	}

	SuggestionsForContext = []string{
		"Check session configuration",
		"Verify context storage",
		"Review session lifecycle",
		"Check for session conflicts",
	}
)
