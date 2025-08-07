package types

import "time"

// ModelInfo represents metadata about a language model.
// This struct contains information about the model's capabilities, version, and provider.
type ModelInfo struct {
	// Name is the unique identifier for the model
	Name string `json:"name"`

	// Provider is the company or organization that provides the model
	Provider string `json:"provider"`

	// Version is the specific version of the model
	Version string `json:"version"`

	// Description provides a human-readable description of the model
	Description string `json:"description,omitempty"`

	// ReleaseDate is when this model version was released
	ReleaseDate time.Time `json:"release_date,omitempty"`

	// ModelType indicates the type of model (e.g., "chat", "completion", "embedding")
	ModelType string `json:"model_type"`

	// ContextWindow is the maximum number of tokens the model can process in a single request
	ContextWindow int `json:"context_window"`

	// MaxOutputTokens is the maximum number of tokens the model can generate in a single response
	MaxOutputTokens int `json:"max_output_tokens"`

	// SupportedLanguages is a list of programming languages and natural languages the model supports
	SupportedLanguages []string `json:"supported_languages,omitempty"`

	// PricingInfo contains information about the model's pricing structure
	PricingInfo map[string]interface{} `json:"pricing_info,omitempty"`

	// PerformanceMetrics contains performance benchmarks and metrics for the model
	PerformanceMetrics map[string]interface{} `json:"performance_metrics,omitempty"`
}

// ModelConfig represents configuration settings for a language model.
// This struct contains the basic parameters needed to configure and use a model.
type ModelConfig struct {
	// ModelName is the unique identifier for the model
	ModelName string `json:"model_name"`

	// Provider is the company or organization that provides the model
	Provider string `json:"provider"`

	// MaxTokens is the maximum number of tokens to generate in a response
	MaxTokens int `json:"max_tokens" validate:"min=1"`

	// Temperature controls the randomness of the model's responses (0.0 to 2.0)
	Temperature float64 `json:"temperature" validate:"min=0,max=2"`

	// APIEndpoint is the base URL for the model's API
	APIEndpoint string `json:"api_endpoint"`
}

// TokenUsage represents token consumption statistics for a model request.
// This struct tracks both input and output token usage for cost and performance monitoring.
type TokenUsage struct {
	// PromptTokens is the number of tokens in the input prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the generated response
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used (prompt + completion)
	TotalTokens int `json:"total_tokens"`

	// EstimatedCost is the estimated cost of the request in the model's pricing currency
	EstimatedCost float64 `json:"estimated_cost,omitempty"`

	// Currency is the currency used for cost estimation
	Currency string `json:"currency,omitempty"`

	// ProcessingTime is the time taken to process the request
	ProcessingTime time.Duration `json:"processing_time,omitempty"`

	// TokensPerSecond is the rate of token processing
	TokensPerSecond float64 `json:"tokens_per_second,omitempty"`

	// ModelName is the name of the model that generated this token usage
	ModelName string `json:"model_name"`

	// RequestID is a unique identifier for the request that generated this usage
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is when this token usage was recorded
	Timestamp time.Time `json:"timestamp"`
}

// Example represents a few-shot example for prompt engineering.
// This struct contains input-output pairs used to guide model responses.
type Example struct {
	// Input is the example input query or prompt
	Input string `json:"input"`

	// Output is the expected output or response for the input
	Output string `json:"output"`

	// Description provides context about what this example demonstrates
	Description string `json:"description,omitempty"`

	// Category indicates the type or category of this example
	Category string `json:"category,omitempty"`

	// Tags contains keywords for categorizing and filtering examples
	Tags []string `json:"tags,omitempty"`

	// Confidence indicates the confidence level for this example's correctness
	Confidence float64 `json:"confidence,omitempty" validate:"omitempty,min=0,max=1"`

	// CreatedAt is when this example was created
	CreatedAt time.Time `json:"created_at,omitempty"`

	// UpdatedAt is when this example was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// IsActive indicates whether this example is currently active and should be used
	IsActive bool `json:"is_active"`

	// Priority determines the order in which examples are presented to the model
	Priority int `json:"priority,omitempty" validate:"omitempty,min=1"`

	// Metadata contains additional metadata about this example
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
