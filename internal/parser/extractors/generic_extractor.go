package extractors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// GenericExtractor provides regex-based JSON extraction suitable as a fallback
type GenericExtractor struct {
	confidence float64
}

// NewGenericExtractor creates a new instance
func NewGenericExtractor() *GenericExtractor { return &GenericExtractor{confidence: 0.0} }

// ParseResponse extracts JSON using regex heuristics and validates basic fields
func (g *GenericExtractor) ParseResponse(raw *types.RawResponse, _ string) (*types.StructuredQuery, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw response is nil")
	}
	g.confidence = 0.0

	content := strings.TrimSpace(raw.Content)
	if content == "" {
		return nil, fmt.Errorf("raw content is empty")
	}

	jsonStr, err := g.extractJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	var query types.StructuredQuery
	if err := json.Unmarshal([]byte(jsonStr), &query); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if query.LogSource == "" {
		// Provide a minimal default to maximize fallback usefulness
		query.LogSource = "kube-apiserver"
		g.confidence = 0.6
	} else {
		g.confidence = 0.8
	}

	// Light sanity checks
	if query.Limit < 0 || query.Limit > 10000 {
		query.Limit = 20
	}

	return &query, nil
}

// CanHandle returns true for any model (generic)
func (g *GenericExtractor) CanHandle(_ string) bool { return true }

// GetConfidence returns last parse confidence
func (g *GenericExtractor) GetConfidence() float64 { return g.confidence }

// extractJSON attempts to pull a JSON object/array from arbitrary text
func (g *GenericExtractor) extractJSON(content string) (string, error) {
	// 1) fenced code block ```json ... ``` or ``` ... ```
	if m := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```$").FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1]), nil
	}
	if m := regexp.MustCompile("(?s)" + "```" + `\s*(.*?)\s*` + "```" + "$").FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1]), nil
	}

	// 2) XML-like <json>...</json>
	if m := regexp.MustCompile(`(?s)<json>\s*(.*?)\s*</json>`).FindStringSubmatch(content); len(m) > 1 {
		return strings.TrimSpace(m[1]), nil
	}

	// 3) Already JSON-like at start
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}

	// 4) Find first {...} balanced block
	start := strings.Index(content, "{")
	if start != -1 {
		end := findMatchingBrace(content, start)
		if end != -1 {
			fragment := content[start : end+1]
			return strings.TrimSpace(fragment), nil
		}
	}

	return "", fmt.Errorf("no valid JSON content found in response")
}

// findMatchingBrace looks for a matching closing brace in content
func findMatchingBrace(content string, start int) int {
	if start >= len(content) || content[start] != '{' {
		return -1
	}
	count := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			count++
		case '}':
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

// Ensure interface implementation
var _ interfaces.Parser = (*GenericExtractor)(nil)
