package extractors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ClaudeExtractor implements the Parser interface for Claude model responses.
// It handles Claude-specific output formats including markdown-wrapped JSON
// and provides confidence scoring for parsing quality assessment.
type ClaudeExtractor struct {
	confidence float64
}

// NewClaudeExtractor creates a new instance of ClaudeExtractor.
func NewClaudeExtractor() *ClaudeExtractor {
	return &ClaudeExtractor{
		confidence: 0.0,
	}
}

// ParseResponse parses a raw response from Claude into structured format.
// It handles markdown-wrapped JSON (```json...```) and extracts the structured query.
func (c *ClaudeExtractor) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw response is nil")
	}

	// Reset confidence for new parsing attempt
	c.confidence = 0.0

	// Extract content from raw response
	content, err := c.extractContent(raw.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Extract JSON from the content
	jsonData, err := c.extractJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	// Parse JSON into StructuredQuery
	var query types.StructuredQuery
	if err := json.Unmarshal(jsonData, &query); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate the parsed query
	if err := c.validateQuery(&query); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Calculate confidence based on parsing success and content quality
	c.confidence = c.calculateConfidence(content, jsonData, &query)

	return &query, nil
}

// CanHandle determines whether this parser can handle responses from Claude models.
func (c *ClaudeExtractor) CanHandle(modelType string) bool {
	// Handle various Claude model identifiers
	claudePatterns := []string{
		"claude",
		"anthropic",
		"claude-3",
		"claude-3-5-sonnet",
		"claude-3-opus",
		"claude-3-haiku",
	}

	modelTypeLower := strings.ToLower(modelType)
	for _, pattern := range claudePatterns {
		if strings.Contains(modelTypeLower, pattern) {
			return true
		}
	}
	return false
}

// GetConfidence returns the confidence score of the last parsing operation.
func (c *ClaudeExtractor) GetConfidence() float64 {
	return c.confidence
}

// extractContent extracts the main content from Claude's response.
// It handles various output formats including markdown blocks and plain text.
func (c *ClaudeExtractor) extractContent(rawContent string) (string, error) {
	if rawContent == "" {
		return "", fmt.Errorf("raw content is empty")
	}

	content := strings.TrimSpace(rawContent)

	// Handle markdown-wrapped JSON (```json...```)
	markdownJSONPattern := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```$")
	if matches := markdownJSONPattern.FindStringSubmatch(content); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// Handle XML-wrapped content (<json>...</json>)
	xmlJSONPattern := regexp.MustCompile(`(?s)<json>\s*(.*?)\s*</json>`)
	if matches := xmlJSONPattern.FindStringSubmatch(content); len(matches) > 1 {
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
		jsonEnd := c.findMatchingBrace(content, jsonStart)
		if jsonEnd != -1 {
			return content[jsonStart : jsonEnd+1], nil
		}
	}

	return "", fmt.Errorf("no valid JSON content found in response")
}

// extractJSON extracts and validates JSON data from the content.
func (c *ClaudeExtractor) extractJSON(content string) ([]byte, error) {
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
func (c *ClaudeExtractor) validateQuery(query *types.StructuredQuery) error {
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

	return nil
}

// calculateConfidence calculates a confidence score based on parsing success and content quality.
func (c *ClaudeExtractor) calculateConfidence(content string, jsonData []byte, query *types.StructuredQuery) float64 {
	confidence := 1.0

	// Reduce confidence if content was heavily processed
	if strings.Contains(content, "```") {
		confidence -= 0.1 // Markdown wrapping indicates potential formatting issues
	}

	// Reduce confidence if JSON required cleanup
	if len(jsonData) != len(content) {
		confidence -= 0.1 // JSON was modified during extraction
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
func (c *ClaudeExtractor) findMatchingBrace(content string, startPos int) int {
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

// Ensure ClaudeExtractor implements the Parser interface
var _ interfaces.Parser = (*ClaudeExtractor)(nil)
