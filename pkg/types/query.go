package types

// ProcessingRequest represents the input request for natural language query processing.
// It contains the user's natural language query, session identifier for context management,
// and an optional model type specification for multi-model support.
type ProcessingRequest struct {
	// Query is the natural language query to be processed and converted to structured format
	Query string `json:"query"`

	// SessionID is a unique identifier for maintaining conversation context across multiple turns
	SessionID string `json:"session_id"`

	// ModelType is an optional field specifying which LLM model to use for processing.
	// If not specified, the system will use the default or best available model.
	ModelType string `json:"model_type,omitempty"`
}

// ProcessingResponse represents the output response from natural language query processing.
// It contains the structured query result, confidence score, validation information,
// and optional error details if processing failed.
type ProcessingResponse struct {
	// StructuredQuery contains the parsed and structured query in JSON format
	StructuredQuery interface{} `json:"structured_query"`

	// Confidence represents the confidence score (0.0 to 1.0) of the parsing accuracy
	Confidence float64 `json:"confidence"`

	// ValidationInfo contains information about validation results and any warnings
	ValidationInfo interface{} `json:"validation_info"`

	// Error contains error details if the processing failed
	Error string `json:"error,omitempty"`
}

// InternalRequest represents the internal processing request used within the system.
// This struct is used for internal communication between different processing components.
type InternalRequest struct {
	// RequestID is a unique identifier for tracking the request through the system
	RequestID string `json:"request_id"`

	// ProcessingRequest contains the original processing request
	ProcessingRequest ProcessingRequest `json:"processing_request"`

	// ProcessingOptions contains additional options for internal processing
	ProcessingOptions map[string]interface{} `json:"processing_options,omitempty"`
}

// ModelRequest represents the request structure for making API calls to language models.
// This struct is used when communicating with external LLM providers.
type ModelRequest struct {
	// Model specifies the model identifier to use for the request
	Model string `json:"model"`

	// Messages contains the conversation messages to send to the model
	Messages []interface{} `json:"messages"`

	// Parameters contains model-specific parameters (temperature, max_tokens, etc.)
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// RawResponse represents the raw response received from language model APIs.
// This struct captures the unprocessed response before any parsing or validation.
type RawResponse struct {
	// Content contains the raw response content from the model
	Content string `json:"content"`

	// ModelInfo contains information about the model that generated the response
	ModelInfo map[string]interface{} `json:"model_info,omitempty"`

	// Metadata contains additional metadata about the response
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Error contains error information if the model request failed
	Error string `json:"error,omitempty"`
}
