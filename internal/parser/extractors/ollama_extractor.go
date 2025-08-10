package extractors

import (
	"encoding/json"
	"fmt"
	"strings"

	"genai-processing/pkg/types"
)

// OllamaExtractor implements parsing for Ollama model responses
type OllamaExtractor struct {
	confidence float64
}

// NewOllamaExtractor creates a new OllamaExtractor
func NewOllamaExtractor() *OllamaExtractor {
	return &OllamaExtractor{
		confidence: 0.85, // High confidence for Ollama responses
	}
}

// CanHandle determines if this extractor can handle the given model type
func (o *OllamaExtractor) CanHandle(modelType string) bool {
	modelType = strings.ToLower(strings.TrimSpace(modelType))
	return strings.Contains(modelType, "ollama") ||
		strings.Contains(modelType, "llama") ||
		strings.Contains(modelType, "llama3") ||
		strings.Contains(modelType, "llama2") ||
		modelType == "local_llama"
}

// GetConfidence returns the confidence level for this extractor
func (o *OllamaExtractor) GetConfidence() float64 {
	return o.confidence
}

// ParseResponse extracts structured query from Ollama response
func (o *OllamaExtractor) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	if raw == nil || raw.Content == "" {
		return nil, fmt.Errorf("empty or nil response")
	}

	content := strings.TrimSpace(raw.Content)

	// Try to extract JSON from the response
	jsonContent, err := o.extractJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from Ollama response: %w", err)
	}

	// Parse the JSON into a structured query
	var query types.StructuredQuery
	if err := json.Unmarshal([]byte(jsonContent), &query); err != nil {
		return nil, fmt.Errorf("failed to parse JSON into structured query: %w", err)
	}

	// Validate the parsed query
	if err := o.validateQuery(&query); err != nil {
		return nil, fmt.Errorf("invalid structured query: %w", err)
	}

	return &query, nil
}

// extractJSON extracts JSON content from Ollama response
func (o *OllamaExtractor) extractJSON(content string) (string, error) {
	// Try to find JSON wrapped in markdown code blocks
	if strings.Contains(content, "```") {
		// Look for JSON in code blocks
		start := strings.Index(content, "```")
		if start != -1 {
			end := strings.Index(content[start+3:], "```")
			if end != -1 {
				codeBlock := strings.TrimSpace(content[start+3 : start+3+end])
				// Remove language identifier if present
				if strings.HasPrefix(codeBlock, "json") {
					codeBlock = strings.TrimSpace(codeBlock[4:])
				}
				// Validate that it's actually JSON
				var test interface{}
				if err := json.Unmarshal([]byte(codeBlock), &test); err == nil {
					return codeBlock, nil
				}
			}
		}
	}

	// Try to find JSON wrapped in XML tags
	if strings.Contains(content, "<json>") && strings.Contains(content, "</json>") {
		start := strings.Index(content, "<json>") + 6
		end := strings.Index(content, "</json>")
		if start != -1 && end != -1 && end > start {
			jsonContent := strings.TrimSpace(content[start:end])
			var test interface{}
			if err := json.Unmarshal([]byte(jsonContent), &test); err == nil {
				return jsonContent, nil
			}
		}
	}

	// Try to find JSON at the end of the response
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var test interface{}
			if err := json.Unmarshal([]byte(line), &test); err == nil {
				return line, nil
			}
		}
	}

	return "", fmt.Errorf("no valid JSON found in response")
}

// validateQuery validates the parsed structured query
func (o *OllamaExtractor) validateQuery(query *types.StructuredQuery) error {
	if query == nil {
		return fmt.Errorf("query is nil")
	}

	// Check required fields
	if query.LogSource == "" {
		return fmt.Errorf("log_source is required")
	}

	// Validate log source
	validLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver"}
	logSourceValid := false
	for _, valid := range validLogSources {
		if strings.EqualFold(query.LogSource, valid) {
			logSourceValid = true
			break
		}
	}
	if !logSourceValid {
		return fmt.Errorf("invalid log_source: %s", query.LogSource)
	}

	// Validate limit if present
	if query.Limit > 0 && query.Limit > 1000 {
		return fmt.Errorf("limit exceeds maximum allowed value of 1000")
	}

	// Validate timeframe if present
	if query.Timeframe != "" {
		// Basic timeframe validation - could be enhanced
		if !strings.Contains(strings.ToLower(query.Timeframe), "hour") &&
			!strings.Contains(strings.ToLower(query.Timeframe), "day") &&
			!strings.Contains(strings.ToLower(query.Timeframe), "week") &&
			!strings.Contains(strings.ToLower(query.Timeframe), "month") {
			return fmt.Errorf("invalid timeframe format: %s", query.Timeframe)
		}
	}

	return nil
}
