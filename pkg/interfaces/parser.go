package interfaces

import (
	"genai-processing/pkg/types"
)

// Parser defines the interface for parsing responses from different LLM models.
// This interface handles the extraction and normalization of structured data
// from various model output formats (JSON, XML, markdown, etc.).
type Parser interface {
	// ParseResponse parses a raw response from an LLM provider into structured format.
	// This method handles model-specific output formatting quirks and extracts
	// the structured query data from the raw response content.
	//
	// Parameters:
	//   - raw: The raw response from the LLM provider
	//   - modelType: The type of model that generated the response
	//
	// Returns:
	//   - StructuredQuery: The parsed and structured query data
	//   - error: Any error that occurred during parsing
	ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error)

	// CanHandle determines whether this parser can handle responses from a specific model type.
	// This method allows the system to select the appropriate parser based on
	// the model that generated the response.
	//
	// Parameters:
	//   - modelType: The type of model to check compatibility with
	//
	// Returns:
	//   - bool: True if this parser can handle the specified model type
	CanHandle(modelType string) bool

	// GetConfidence returns the confidence score of the last parsing operation.
	// This method provides a measure of how reliable the parsed output is,
	// which can be used for quality assessment and fallback decisions.
	//
	// Returns:
	//   - float64: Confidence score between 0.0 and 1.0
	GetConfidence() float64
}

// ResponseExtractor defines the interface for model-specific response extraction.
// This interface handles the extraction of structured data from model-specific
// output formats (Claude's XML, OpenAI's JSON, etc.).
type ResponseExtractor interface {
	// ExtractContent extracts the main content from a raw model response.
	// This method handles model-specific output formatting and extracts
	// the relevant content for further processing.
	//
	// Parameters:
	//   - raw: The raw response from the LLM provider
	//
	// Returns:
	//   - string: The extracted content ready for parsing
	//   - error: Any error that occurred during extraction
	ExtractContent(raw *types.RawResponse) (string, error)

	// ExtractJSON extracts JSON data from the model response.
	// This method handles various JSON formats including markdown-wrapped JSON,
	// XML-wrapped JSON, and plain JSON responses.
	//
	// Parameters:
	//   - content: The extracted content from the model response
	//
	// Returns:
	//   - []byte: The extracted JSON data
	//   - error: Any error that occurred during JSON extraction
	ExtractJSON(content string) ([]byte, error)

	// ValidateExtraction validates that the extracted content is valid and complete.
	// This method performs basic validation to ensure the extraction was successful
	// and the content is suitable for further processing.
	//
	// Parameters:
	//   - content: The extracted content to validate
	//
	// Returns:
	//   - bool: True if the extraction is valid
	//   - error: Any validation errors encountered
	ValidateExtraction(content string) (bool, error)
}

// ResponseNormalizer defines the interface for standardizing parsed output.
// This interface handles the normalization of parsed responses to ensure
// consistent internal representation regardless of the source model.
type ResponseNormalizer interface {
	// NormalizeResponse normalizes a parsed response to the standard internal format.
	// This method ensures that responses from different models are converted
	// to a consistent internal representation for further processing.
	//
	// Parameters:
	//   - parsed: The parsed response to normalize
	//   - modelType: The type of model that generated the original response
	//
	// Returns:
	//   - StructuredQuery: The normalized structured query
	//   - error: Any error that occurred during normalization
	NormalizeResponse(parsed interface{}, modelType string) (*types.StructuredQuery, error)

	// ValidateSchema validates that a structured query conforms to the expected schema.
	// This method ensures that the normalized output meets the required
	// structure and field requirements for the audit query system.
	//
	// Parameters:
	//   - query: The structured query to validate
	//
	// Returns:
	//   - bool: True if the query conforms to the schema
	//   - error: Any validation errors encountered
	ValidateSchema(query *types.StructuredQuery) error

	// MapFields maps model-specific field names to standard field names.
	// This method handles the conversion of model-specific terminology
	// to the standardized field names used in the internal schema.
	//
	// Parameters:
	//   - modelFields: Map of model-specific field names to values
	//   - modelType: The type of model that generated the fields
	//
	// Returns:
	//   - map[string]interface{}: Map of standardized field names to values
	MapFields(modelFields map[string]interface{}, modelType string) map[string]interface{}
}
