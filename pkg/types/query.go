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

// ProcessingResponse represents the output response from the natural language processing pipeline.
// It contains the structured query, confidence score, validation information, and optional error details.
type ProcessingResponse struct {
	// StructuredQuery contains the parsed and structured JSON representation of the natural language query
	StructuredQuery *StructuredQuery `json:"structured_query"`

	// Confidence is a float64 value between 0.0 and 1.0 indicating the confidence level
	// of the parsing and processing results
	Confidence float64 `json:"confidence"`

	// ValidationInfo contains the results of safety validation and rule checking
	ValidationInfo *ValidationInfo `json:"validation_info"`

	// Error contains error details if processing failed, nil if successful
	Error *ProcessingError `json:"error,omitempty"`
}

// InternalRequest represents the internal request structure used for processing within the system.
// This is used for internal communication between components and contains enriched context.
type InternalRequest struct {
	// Query is the natural language query to be processed
	Query string `json:"query"`

	// Context contains conversation context and session information
	Context *ConversationContext `json:"context"`

	// Examples contains few-shot examples for the specific query pattern
	Examples []Example `json:"examples"`

	// SystemPrompt contains the system prompt template for the model
	SystemPrompt string `json:"system_prompt"`

	// ModelConfig contains model-specific configuration and parameters
	ModelConfig *ModelConfig `json:"model_config"`
}

// ModelRequest represents the request structure sent to LLM model APIs.
// This is a model-agnostic structure that gets adapted for specific model requirements.
type ModelRequest struct {
	// Payload contains the actual request payload in the format expected by the specific model API
	Payload interface{} `json:"payload"`

	// Headers contains HTTP headers required for the API request (e.g., authentication)
	Headers map[string]string `json:"headers"`

	// Endpoint is the API endpoint URL for the model service
	Endpoint string `json:"endpoint"`

	// Method is the HTTP method to use for the request (typically POST)
	Method string `json:"method"`

	// Parameters contains additional model-specific parameters (temperature, max_tokens, etc.)
	Parameters map[string]interface{} `json:"parameters"`
}

// RawResponse represents the raw response received from LLM model APIs.
// This contains the unprocessed response before parsing and normalization.
type RawResponse struct {
	// Content is the raw text content returned by the model
	Content string `json:"content"`

	// ModelType identifies which model generated this response
	ModelType string `json:"model_type"`

	// Metadata contains additional response metadata (model version, timestamps, etc.)
	Metadata map[string]interface{} `json:"metadata"`

	// TokenUsage contains information about token consumption if available
	TokenUsage *TokenUsage `json:"token_usage,omitempty"`
}

// StructuredQuery represents the structured JSON output following the OpenShift audit query schema.
// This is the normalized output that can be used for actual audit log queries.
type StructuredQuery struct {
	// LogSource specifies which log source to query (kube-apiserver, oauth-server, etc.)
	LogSource string `json:"log_source"`

	// Verb specifies the action verb to filter by (get, create, delete, etc.)
	Verb StringOrArray `json:"verb,omitempty"`

	// Resource specifies the resource type to filter by (pods, namespaces, etc.)
	Resource StringOrArray `json:"resource,omitempty"`

	// Namespace specifies the namespace to filter by
	Namespace StringOrArray `json:"namespace,omitempty"`

	// User specifies the user to filter by
	User StringOrArray `json:"user,omitempty"`

	// Timeframe specifies the time range for the query (today, yesterday, 1_hour_ago, etc.)
	Timeframe string `json:"timeframe,omitempty"`

	// Limit specifies the maximum number of results to return
	Limit int `json:"limit,omitempty"`

	// ResponseStatus specifies the HTTP response status to filter by
	ResponseStatus StringOrArray `json:"response_status,omitempty"`

	// ExcludeUsers specifies users to exclude from the results
	ExcludeUsers []string `json:"exclude_users,omitempty"`

	// ResourceNamePattern specifies a pattern to match resource names
	ResourceNamePattern string `json:"resource_name_pattern,omitempty"`

	// UserPattern specifies a pattern to match user names
	UserPattern string `json:"user_pattern,omitempty"`

	// NamespacePattern specifies a pattern to match namespace names
	NamespacePattern string `json:"namespace_pattern,omitempty"`

	// RequestURIPattern specifies a pattern to match request URIs
	RequestURIPattern string `json:"request_uri_pattern,omitempty"`

	// AuthDecision specifies the authentication decision to filter by
	AuthDecision string `json:"auth_decision,omitempty"`

	// SourceIP specifies the source IP address to filter by
	SourceIP StringOrArray `json:"source_ip,omitempty"`

	// GroupBy specifies fields to group results by
	GroupBy StringOrArray `json:"group_by,omitempty"`

	// SortBy specifies the field to sort results by
	SortBy string `json:"sort_by,omitempty"`

	// SortOrder specifies the sort order (asc, desc)
	SortOrder string `json:"sort_order,omitempty"`

	// TimeRange specifies a custom time range for the query
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// BusinessHours specifies business hours for time-based filtering
	BusinessHours *BusinessHours `json:"business_hours,omitempty"`

	// Analysis specifies analysis configuration for the query
	Analysis *AnalysisConfig `json:"analysis,omitempty"`

	// Subresource specifies the Kubernetes subresource to filter by
	Subresource string `json:"subresource,omitempty"`

	// IncludeChanges specifies whether to include before/after comparisons
	IncludeChanges bool `json:"include_changes,omitempty"`

	// RequestObjectFilter specifies filter based on request object content
	RequestObjectFilter string `json:"request_object_filter,omitempty"`

	// ExcludeResources specifies resource patterns to exclude
	ExcludeResources []string `json:"exclude_resources,omitempty"`

	// AuthorizationReasonPattern specifies pattern for authorization reason matching
	AuthorizationReasonPattern string `json:"authorization_reason_pattern,omitempty"`

	// ResponseMessagePattern specifies pattern for response message matching
	ResponseMessagePattern string `json:"response_message_pattern,omitempty"`

	// MissingAnnotation specifies annotation that should be missing
	MissingAnnotation string `json:"missing_annotation,omitempty"`
}

// StringOrArray represents a field that can be either a single string or an array of strings.
// This allows for flexible query specifications where a field can match one or multiple values.
type StringOrArray struct {
	// Value contains the string value if this represents a single string
	Value string `json:"value,omitempty"`

	// Values contains the array of strings if this represents multiple values
	Values []string `json:"values,omitempty"`

	// IsArray indicates whether this represents an array (true) or single value (false)
	IsArray bool `json:"is_array"`
}

// TimeRange represents a custom time range for query filtering.
type TimeRange struct {
	// Start specifies the start of the time range (ISO 8601 format)
	Start string `json:"start"`

	// End specifies the end of the time range (ISO 8601 format)
	End string `json:"end"`
}

// BusinessHours represents business hours configuration for time-based filtering.
type BusinessHours struct {
	// OutsideOnly specifies whether to filter for outside business hours only
	OutsideOnly bool `json:"outside_only,omitempty"`

	// StartHour specifies the start hour (0-23)
	StartHour int `json:"start_hour"`

	// EndHour specifies the end hour (0-23)
	EndHour int `json:"end_hour"`

	// DaysOfWeek specifies which days of the week are considered business days
	DaysOfWeek []string `json:"days_of_week,omitempty"`

	// Timezone specifies the timezone (default: UTC)
	Timezone string `json:"timezone,omitempty"`
}

// AnalysisConfig represents configuration for query analysis features.
type AnalysisConfig struct {
	// Type specifies the type of analysis to perform
	Type string `json:"type"`

	// GroupBy specifies fields to group results by for analysis
	GroupBy StringOrArray `json:"group_by,omitempty"`

	// Threshold specifies the threshold for analysis triggers
	Threshold int `json:"threshold,omitempty"`

	// TimeWindow specifies the time window for analysis
	TimeWindow string `json:"time_window,omitempty"`

	// AnomalyDetection enables anomaly detection in the results
	AnomalyDetection bool `json:"anomaly_detection,omitempty"`

	// TrendAnalysis enables trend analysis in the results
	TrendAnalysis bool `json:"trend_analysis,omitempty"`

	// RiskScoring enables risk scoring for the results
	RiskScoring bool `json:"risk_scoring,omitempty"`
}

// ConversationContext represents the conversation context for multi-turn interactions.
type ConversationContext struct {
	// SessionID is the unique session identifier
	SessionID string `json:"session_id"`

	// History contains the conversation history
	History []ConversationTurn `json:"history"`

	// ResolvedReferences contains resolved pronouns and references
	ResolvedReferences map[string]string `json:"resolved_references"`

	// LastQueryTime is the timestamp of the last query
	LastQueryTime string `json:"last_query_time"`
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	// Query is the user's query
	Query string `json:"query"`

	// Response is the system's response
	Response *StructuredQuery `json:"response"`

	// Timestamp is when this turn occurred
	Timestamp string `json:"timestamp"`
}

// Example represents a few-shot example for model training.
type Example struct {
	// Input is the natural language input
	Input string `json:"input"`

	// Output is the expected structured output
	Output *StructuredQuery `json:"output"`
}

// ModelConfig represents configuration for a specific model.
type ModelConfig struct {
	// ModelName is the name of the model
	ModelName string `json:"model_name"`

	// Provider is the provider of the model (anthropic, openai, ollama, etc.)
	Provider string `json:"provider"`

	// MaxTokens is the maximum number of tokens for the response
	MaxTokens int `json:"max_tokens"`

	// Temperature controls the randomness of the response
	Temperature float64 `json:"temperature"`

	// APIEndpoint is the API endpoint for the model
	APIEndpoint string `json:"api_endpoint"`
}

// ValidationInfo represents the results of safety validation.
type ValidationInfo struct {
	// IsValid indicates whether the query passed all validation rules
	IsValid bool `json:"is_valid"`

	// Violations contains any validation rule violations
	Violations []ValidationViolation `json:"violations"`

	// Warnings contains any validation warnings
	Warnings []string `json:"warnings"`

	// AppliedRules lists the validation rules that were applied
	AppliedRules []string `json:"applied_rules"`
}

// ValidationViolation represents a specific validation rule violation.
type ValidationViolation struct {
	// Rule is the name of the violated rule
	Rule string `json:"rule"`

	// Message describes the violation
	Message string `json:"message"`

	// Severity indicates the severity level of the violation
	Severity string `json:"severity"`

	// Field indicates which field violated the rule
	Field string `json:"field,omitempty"`
}

// ProcessingError represents an error that occurred during processing.
type ProcessingError struct {
	// Type is the type of error that occurred
	Type string `json:"type"`

	// Message is the error message
	Message string `json:"message"`

	// Details contains additional error details
	Details map[string]interface{} `json:"details,omitempty"`

	// Component indicates which component generated the error
	Component string `json:"component"`

	// Recoverable indicates whether the error is recoverable
	Recoverable bool `json:"recoverable"`

	// Suggestions contains suggestions for resolving the error
	Suggestions []string `json:"suggestions,omitempty"`

	// Timestamp is when the error occurred
	Timestamp string `json:"timestamp"`
}

// TokenUsage represents token consumption information from model responses.
type TokenUsage struct {
	// InputTokens is the number of input tokens used
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the number of output tokens generated
	OutputTokens int `json:"output_tokens"`

	// TotalTokens is the total number of tokens used
	TotalTokens int `json:"total_tokens"`
}
