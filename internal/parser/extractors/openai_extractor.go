package extractors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// OpenAIExtractor implements the Parser interface for OpenAI model responses.
// It handles OpenAI-specific output formats including clean JSON responses
// and function call responses, providing confidence scoring for parsing quality assessment.
type OpenAIExtractor struct {
	confidence float64
}

// NewOpenAIExtractor creates a new instance of OpenAIExtractor.
func NewOpenAIExtractor() *OpenAIExtractor {
	return &OpenAIExtractor{
		confidence: 0.0,
	}
}

// ParseResponse parses a raw response from OpenAI into structured format.
// It handles clean JSON responses and function call responses from OpenAI models.
func (o *OpenAIExtractor) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw response is nil")
	}

	// Reset confidence for new parsing attempt
	o.confidence = 0.0

	// Extract content from raw response
	content, err := o.extractContent(raw.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Extract JSON from the content
	jsonData, err := o.extractJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	// Parse JSON into StructuredQuery
	var query types.StructuredQuery
	if err := json.Unmarshal(jsonData, &query); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate the parsed query
	if err := o.validateQuery(&query); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Calculate confidence based on parsing success and content quality
	o.confidence = o.calculateConfidence(content, jsonData, &query)

	return &query, nil
}

// CanHandle determines whether this parser can handle responses from OpenAI models.
func (o *OpenAIExtractor) CanHandle(modelType string) bool {
	// Handle various OpenAI model identifiers
	openaiPatterns := []string{
		"openai",
		"gpt",
		"gpt-3",
		"gpt-4",
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-3.5",
		"gpt-3.5-turbo",
	}

	modelTypeLower := strings.ToLower(modelType)
	for _, pattern := range openaiPatterns {
		if strings.Contains(modelTypeLower, pattern) {
			return true
		}
	}
	return false
}

// GetConfidence returns the confidence score of the last parsing operation.
func (o *OpenAIExtractor) GetConfidence() float64 {
	return o.confidence
}

// extractContent extracts the main content from OpenAI's response.
// It handles various output formats including clean JSON, function calls, and markdown blocks.
func (o *OpenAIExtractor) extractContent(rawContent string) (string, error) {
	if rawContent == "" {
		return "", fmt.Errorf("raw content is empty")
	}

	content := strings.TrimSpace(rawContent)

	// Handle function call responses (OpenAI function calling format)
	if o.isFunctionCallResponse(content) {
		return o.extractFunctionCallContent(content)
	}

	// Handle markdown-wrapped JSON (```json...```)
	markdownJSONPattern := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```$")
	if matches := markdownJSONPattern.FindStringSubmatch(content); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// Handle code block without language specification (```...```)
	codeBlockPattern := regexp.MustCompile("(?s)" + "```" + `\s*(.*?)\s*` + "```" + "$")
	if matches := codeBlockPattern.FindStringSubmatch(content); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// If no wrapping found, assume the content is already JSON
	// Check if it starts with { or [ to confirm it's JSON-like
	if strings.HasPrefix(strings.TrimSpace(content), "{") || strings.HasPrefix(strings.TrimSpace(content), "[") {
		return content, nil
	}

	// Try to find JSON content within the response
	jsonStart := strings.Index(content, "{")
	if jsonStart != -1 {
		jsonEnd := o.findMatchingBrace(content, jsonStart)
		if jsonEnd != -1 {
			return content[jsonStart : jsonEnd+1], nil
		}
	}

	return "", fmt.Errorf("no valid JSON content found in response")
}

// isFunctionCallResponse checks if the response is in OpenAI function calling format.
func (o *OpenAIExtractor) isFunctionCallResponse(content string) bool {
	// Look for function call indicators in the content
	functionCallPatterns := []string{
		`"function_call"`,
		`"tool_calls"`,
		`"name":`,
		`"arguments":`,
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range functionCallPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// extractFunctionCallContent extracts content from OpenAI function call responses.
func (o *OpenAIExtractor) extractFunctionCallContent(content string) (string, error) {
	// Try to parse as a function call response structure
	var functionCallResponse struct {
		FunctionCall *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function_call,omitempty"`
		ToolCalls []struct {
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
		Content string `json:"content,omitempty"`
	}

	if err := json.Unmarshal([]byte(content), &functionCallResponse); err != nil {
		return "", fmt.Errorf("failed to parse function call response: %w", err)
	}

	// Extract from function_call if present
	if functionCallResponse.FunctionCall != nil && functionCallResponse.FunctionCall.Arguments != "" {
		return functionCallResponse.FunctionCall.Arguments, nil
	}

	// Extract from tool_calls if present
	if len(functionCallResponse.ToolCalls) > 0 && functionCallResponse.ToolCalls[0].Function.Arguments != "" {
		return functionCallResponse.ToolCalls[0].Function.Arguments, nil
	}

	// Extract from content if present
	if functionCallResponse.Content != "" {
		return functionCallResponse.Content, nil
	}

	return "", fmt.Errorf("no valid content found in function call response")
}

// extractJSON extracts and validates JSON data from the content.
func (o *OpenAIExtractor) extractJSON(content string) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("content is empty")
	}

	// Clean up the content
	content = strings.TrimSpace(content)

	// Remove any trailing commas before closing braces/brackets
	content = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(content, "$1")

	// Validate that the content is valid JSON
	var test interface{}
	if err := json.Unmarshal([]byte(content), &test); err != nil {
		return nil, fmt.Errorf("invalid JSON content: %w", err)
	}

	return []byte(content), nil
}

// validateQuery validates the parsed query for required fields and constraints.
func (o *OpenAIExtractor) validateQuery(query *types.StructuredQuery) error {
	if query == nil {
		return fmt.Errorf("query is nil")
	}

	// Validate required fields
	if query.LogSource == "" {
		return fmt.Errorf("log_source is required")
	}

	// Validate log source values
	validLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver"}
	validSource := false
	for _, valid := range validLogSources {
		if query.LogSource == valid {
			validSource = true
			break
		}
	}
	if !validSource {
		return fmt.Errorf("invalid log_source: %s, must be one of %v", query.LogSource, validLogSources)
	}

	// Validate limit if specified
	if query.Limit > 0 && (query.Limit < 1 || query.Limit > 1000) {
		return fmt.Errorf("limit must be between 1 and 1000, got %d", query.Limit)
	}

	// Note: auth_decision validation now handled by FieldValuesRule in validator package
	// This allows configuration-driven validation instead of hardcoded values

	// Validate sort order if specified
	if query.SortOrder != "" {
		validOrders := []string{"asc", "desc"}
		validOrder := false
		for _, valid := range validOrders {
			if query.SortOrder == valid {
				validOrder = true
				break
			}
		}
		if !validOrder {
			return fmt.Errorf("invalid sort_order: %s, must be one of %v", query.SortOrder, validOrders)
		}
	}

	// Validate sort by if specified
	if query.SortBy != "" {
		validSortFields := []string{"timestamp", "user", "resource", "count"}
		validSortField := false
		for _, valid := range validSortFields {
			if query.SortBy == valid {
				validSortField = true
				break
			}
		}
		if !validSortField {
			return fmt.Errorf("invalid sort_by: %s, must be one of %v", query.SortBy, validSortFields)
		}
	}

	// Validate analysis type if specified
	if query.Analysis != nil {
		validAnalysisTypes := []string{"multi_namespace_access", "excessive_reads", "privilege_escalation", "anomaly_detection", "correlation"}
		validAnalysisType := false
		for _, valid := range validAnalysisTypes {
			if query.Analysis.Type == valid {
				validAnalysisType = true
				break
			}
		}
		if !validAnalysisType {
			return fmt.Errorf("invalid analysis.type: %s, must be one of %v", query.Analysis.Type, validAnalysisTypes)
		}
	}

	return nil
}

// calculateConfidence calculates a confidence score based on parsing success and content quality.
func (o *OpenAIExtractor) calculateConfidence(content string, jsonData []byte, query *types.StructuredQuery) float64 {
	confidence := 1.0

	// Reduce confidence if content was heavily processed
	if strings.Contains(content, "```") {
		confidence -= 0.1 // Markdown wrapping indicates potential formatting issues
	}

	// Reduce confidence if JSON required cleanup
	if len(jsonData) != len(content) {
		confidence -= 0.1 // JSON was modified during extraction
	}

	// Increase confidence for function call responses (OpenAI's preferred format)
	if o.isFunctionCallResponse(content) {
		confidence += 0.05 // Function calls are more reliable
	}

	// Reduce confidence for missing optional but commonly used fields
	if query.Timeframe == "" {
		confidence -= 0.05
	}
	if query.Limit == 0 {
		confidence -= 0.05
	}

	// Increase confidence for well-formed queries with multiple fields
	fieldCount := 0
	if !query.Verb.IsEmpty() {
		fieldCount++
	}
	if !query.Resource.IsEmpty() {
		fieldCount++
	}
	if !query.Namespace.IsEmpty() {
		fieldCount++
	}
	if !query.User.IsEmpty() {
		fieldCount++
	}
	if len(query.ExcludeUsers) > 0 {
		fieldCount++
	}

	if fieldCount >= 3 {
		confidence += 0.1 // Well-formed query with multiple relevant fields
	}

	// Increase confidence for complex queries with advanced features
	if query.TimeRange != nil {
		confidence += 0.05
	}
	if query.BusinessHours != nil {
		confidence += 0.05
	}
	if query.Analysis != nil {
		confidence += 0.05
	}

	// Ensure confidence stays within bounds
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// findMatchingBrace finds the matching closing brace for a given opening brace position.
func (o *OpenAIExtractor) findMatchingBrace(content string, startPos int) int {
	if startPos >= len(content) || content[startPos] != '{' {
		return -1
	}

	braceCount := 0
	for i := startPos; i < len(content); i++ {
		switch content[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				return i
			}
		}
	}

	return -1
}

// Ensure OpenAIExtractor implements the Parser interface
var _ interfaces.Parser = (*OpenAIExtractor)(nil)
