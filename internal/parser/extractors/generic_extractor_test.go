package extractors

import (
	"fmt"
	"genai-processing/pkg/types"
	"strings"
	"testing"
)

func TestNewGenericExtractor(t *testing.T) {
	extractor := NewGenericExtractor()
	if extractor == nil {
		t.Fatal("NewGenericExtractor returned nil")
	}
	if extractor.confidence != 0.0 {
		t.Errorf("Expected initial confidence to be 0.0, got %f", extractor.confidence)
	}
}

func TestGenericExtractor_CanHandle(t *testing.T) {
	extractor := NewGenericExtractor()

	tests := []struct {
		name      string
		modelType string
		expected  bool
	}{
		{"claude", "claude", true},
		{"openai", "openai", true},
		{"gpt-4", "gpt-4", true},
		{"llama", "llama", true},
		{"ollama", "ollama", true},
		{"unknown", "unknown", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.CanHandle(tt.modelType)
			if result != tt.expected {
				t.Errorf("CanHandle(%s) = %v, expected %v", tt.modelType, result, tt.expected)
			}
		})
	}
}

func TestGenericExtractor_GetConfidence(t *testing.T) {
	extractor := NewGenericExtractor()

	// Initial confidence should be 0.0
	if extractor.GetConfidence() != 0.0 {
		t.Errorf("Expected initial confidence to be 0.0, got %f", extractor.GetConfidence())
	}

	// Set confidence and check
	extractor.confidence = 0.75
	if extractor.GetConfidence() != 0.75 {
		t.Errorf("Expected confidence to be 0.75, got %f", extractor.GetConfidence())
	}
}

func TestGenericExtractor_ParseResponse_JSONDirect(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: `{"log_source":"kube-apiserver","verb":"get"}`}
	q, err := e.ParseResponse(raw, "any")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if q.LogSource == "" {
		t.Error("expected log_source")
	}
	if e.GetConfidence() <= 0 {
		t.Error("expected confidence > 0")
	}
}

func TestGenericExtractor_ParseResponse_MarkdownFence(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: "```json\n{\"log_source\":\"kube-apiserver\"}\n```"}
	q, err := e.ParseResponse(raw, "any")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if q.LogSource != "kube-apiserver" {
		t.Error("expected parsed log_source")
	}
}

func TestGenericExtractor_ParseResponse_MissingJSON(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: "no json here"}
	if _, err := e.ParseResponse(raw, "any"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenericExtractor_ParseResponse_ValidJSON(t *testing.T) {
	extractor := NewGenericExtractor()

	validJSON := `{
		"log_source": "oauth-server",
		"verb": "create",
		"resource": "tokens",
		"timeframe": "today",
		"limit": 50
	}`

	rawResponse := &types.RawResponse{
		Content: validJSON,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "oauth-server" {
		t.Errorf("Expected log_source to be 'oauth-server', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "create" {
		t.Errorf("Expected verb to be 'create', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "tokens" {
		t.Errorf("Expected resource to be 'tokens', got '%s'", query.Resource.GetString())
	}

	if query.Timeframe != "today" {
		t.Errorf("Expected timeframe to be 'today', got '%s'", query.Timeframe)
	}

	if query.Limit != 50 {
		t.Errorf("Expected limit to be 50, got %d", query.Limit)
	}

	// Check confidence for valid log_source
	confidence := extractor.GetConfidence()
	if confidence != 0.8 {
		t.Errorf("Expected confidence to be 0.8 for valid log_source, got %f", confidence)
	}
}

func TestGenericExtractor_ParseResponse_MarkdownWrapped(t *testing.T) {
	extractor := NewGenericExtractor()

	markdownContent := `Here is the JSON response:

` + "```" + `json
{
	"log_source": "openshift-apiserver",
	"timeframe": "yesterday",
	"auth_decision": "forbid",
	"limit": 100
}
` + "```" + `

This should extract the JSON properly.`

	rawResponse := &types.RawResponse{
		Content: markdownContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "openshift-apiserver" {
		t.Errorf("Expected log_source to be 'openshift-apiserver', got '%s'", query.LogSource)
	}

	if query.Timeframe != "yesterday" {
		t.Errorf("Expected timeframe to be 'yesterday', got '%s'", query.Timeframe)
	}

	if query.AuthDecision != "forbid" {
		t.Errorf("Expected auth_decision to be 'forbid', got '%s'", query.AuthDecision)
	}

	if query.Limit != 100 {
		t.Errorf("Expected limit to be 100, got %d", query.Limit)
	}
}

func TestGenericExtractor_ParseResponse_XMLWrapped(t *testing.T) {
	extractor := NewGenericExtractor()

	xmlContent := `Here is the response:

<json>
{
	"log_source": "kube-apiserver",
	"verb": "delete",
	"resource": "pods",
	"namespace": "default",
	"timeframe": "1_hour_ago",
	"limit": 25
}
</json>

This should work.`

	rawResponse := &types.RawResponse{
		Content: xmlContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source to be 'kube-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "delete" {
		t.Errorf("Expected verb to be 'delete', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "pods" {
		t.Errorf("Expected resource to be 'pods', got '%s'", query.Resource.GetString())
	}

	if query.Namespace.GetString() != "default" {
		t.Errorf("Expected namespace to be 'default', got '%s'", query.Namespace.GetString())
	}
}

func TestGenericExtractor_ParseResponse_CodeBlockWithoutLanguage(t *testing.T) {
	extractor := NewGenericExtractor()

	codeBlockContent := `Here is the response:

` + "```" + `
{
	"log_source": "oauth-server",
	"verb": "update",
	"resource": "users",
	"timeframe": "7_days_ago",
	"limit": 75
}
` + "```" + ``

	rawResponse := &types.RawResponse{
		Content: codeBlockContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "oauth-server" {
		t.Errorf("Expected log_source to be 'oauth-server', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "update" {
		t.Errorf("Expected verb to be 'update', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "users" {
		t.Errorf("Expected resource to be 'users', got '%s'", query.Resource.GetString())
	}
}

func TestGenericExtractor_ParseResponse_EmbeddedJSON(t *testing.T) {
	extractor := NewGenericExtractor()

	embeddedContent := `The query you requested would be:
	
	{"log_source": "kube-apiserver", "verb": "get", "resource": "secrets", "limit": 10}
	
	This should work for your audit query needs.`

	rawResponse := &types.RawResponse{
		Content: embeddedContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source to be 'kube-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "get" {
		t.Errorf("Expected verb to be 'get', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "secrets" {
		t.Errorf("Expected resource to be 'secrets', got '%s'", query.Resource.GetString())
	}

	if query.Limit != 10 {
		t.Errorf("Expected limit to be 10, got %d", query.Limit)
	}
}

func TestGenericExtractor_ParseResponse_DefaultLogSource(t *testing.T) {
	extractor := NewGenericExtractor()

	jsonWithoutLogSource := `{
		"verb": "list",
		"resource": "configmaps",
		"timeframe": "today",
		"limit": 30
	}`

	rawResponse := &types.RawResponse{
		Content: jsonWithoutLogSource,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	// Should get default log_source
	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected default log_source to be 'kube-apiserver', got '%s'", query.LogSource)
	}

	// Check confidence for default log_source
	confidence := extractor.GetConfidence()
	if confidence != 0.6 {
		t.Errorf("Expected confidence to be 0.6 for default log_source, got %f", confidence)
	}

	if query.Verb.GetString() != "list" {
		t.Errorf("Expected verb to be 'list', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "configmaps" {
		t.Errorf("Expected resource to be 'configmaps', got '%s'", query.Resource.GetString())
	}

	if query.Limit != 30 {
		t.Errorf("Expected limit to be 30, got %d", query.Limit)
	}
}

func TestGenericExtractor_ParseResponse_LimitNormalization(t *testing.T) {
	extractor := NewGenericExtractor()

	tests := []struct {
		name          string
		limit         int
		expectedLimit int
	}{
		{"negative limit", -5, 20},
		{"zero limit", 0, 0}, // Zero is allowed
		{"normal limit", 50, 50},
		{"high limit", 5000, 5000}, // Valid high limit
		{"excessive limit", 15000, 20}, // Over 10000 gets normalized
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonContent := fmt.Sprintf(`{
				"log_source": "kube-apiserver",
				"verb": "get",
				"limit": %d
			}`, tt.limit)

			rawResponse := &types.RawResponse{
				Content: jsonContent,
			}

			query, err := extractor.ParseResponse(rawResponse, "generic")
			if err != nil {
				t.Fatalf("ParseResponse failed: %v", err)
			}

			if query.Limit != tt.expectedLimit {
				t.Errorf("Expected limit to be %d, got %d", tt.expectedLimit, query.Limit)
			}
		})
	}
}

func TestGenericExtractor_ParseResponse_ErrorCases(t *testing.T) {
	extractor := NewGenericExtractor()

	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil response",
			content:     "",
			expectError: true,
			errorMsg:    "raw response is nil",
		},
		{
			name:        "empty content",
			content:     "   ",
			expectError: true,
			errorMsg:    "raw content is empty",
		},
		{
			name:        "no JSON content",
			content:     "This is just plain text with no JSON at all",
			expectError: true,
			errorMsg:    "no valid JSON content found in response",
		},
		{
			name:        "invalid JSON",
			content:     `{"log_source": "kube-apiserver", "invalid": json}`,
			expectError: true,
			errorMsg:    "failed to unmarshal JSON",
		},
		{
			name:        "malformed JSON braces",
			content:     `{"log_source": "kube-apiserver", "verb": "get"`,
			expectError: true,
			errorMsg:    "failed to unmarshal JSON",
		},
		{
			name:        "nested object with valid JSON",
			content:     `{"log_source": "oauth-server", "nested": {"key": "value"}, "verb": "create"}`,
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rawResponse *types.RawResponse
			if tt.content != "" {
				rawResponse = &types.RawResponse{Content: tt.content}
			}

			_, err := extractor.ParseResponse(rawResponse, "generic")
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenericExtractor_ParseResponse_ArrayFields(t *testing.T) {
	extractor := NewGenericExtractor()

	jsonWithArrays := `{
		"log_source": "kube-apiserver",
		"verb": ["create", "delete", "update"],
		"resource": ["pods", "services"],
		"namespace": ["default", "kube-system", "production"],
		"user": ["alice", "bob"],
		"exclude_users": ["system:", "kube-"],
		"timeframe": "today",
		"limit": 40
	}`

	rawResponse := &types.RawResponse{
		Content: jsonWithArrays,
	}

	query, err := extractor.ParseResponse(rawResponse, "generic")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	// Check array fields
	verbs := query.Verb.GetArray()
	if len(verbs) != 3 || verbs[0] != "create" || verbs[1] != "delete" || verbs[2] != "update" {
		t.Errorf("Expected verbs to be ['create', 'delete', 'update'], got %v", verbs)
	}

	resources := query.Resource.GetArray()
	if len(resources) != 2 || resources[0] != "pods" || resources[1] != "services" {
		t.Errorf("Expected resources to be ['pods', 'services'], got %v", resources)
	}

	namespaces := query.Namespace.GetArray()
	if len(namespaces) != 3 || namespaces[0] != "default" || namespaces[1] != "kube-system" || namespaces[2] != "production" {
		t.Errorf("Expected namespaces to be ['default', 'kube-system', 'production'], got %v", namespaces)
	}

	users := query.User.GetArray()
	if len(users) != 2 || users[0] != "alice" || users[1] != "bob" {
		t.Errorf("Expected users to be ['alice', 'bob'], got %v", users)
	}

	excludeUsers := query.ExcludeUsers
	if len(excludeUsers) != 2 || excludeUsers[0] != "system:" || excludeUsers[1] != "kube-" {
		t.Errorf("Expected exclude_users to be ['system:', 'kube-'], got %v", excludeUsers)
	}
}

func TestGenericExtractor_FindMatchingBrace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		startPos int
		expected int
	}{
		{
			name:     "simple braces",
			content:  `{"key": "value"}`,
			startPos: 0,
			expected: 15,
		},
		{
			name:     "nested braces",
			content:  `{"key": {"nested": "value"}}`,
			startPos: 0,
			expected: 27,
		},
		{
			name:     "no opening brace",
			content:  `"key": "value"`,
			startPos: 0,
			expected: -1,
		},
		{
			name:     "unmatched braces",
			content:  `{"key": "value"`,
			startPos: 0,
			expected: -1,
		},
		{
			name:     "start position not brace",
			content:  `{"key": "value"}`,
			startPos: 1,
			expected: -1,
		},
		{
			name:     "complex nested structure",
			content:  `{"a": {"b": {"c": "value"}}}`,
			startPos: 0,
			expected: 27,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findMatchingBrace(tt.content, tt.startPos)
			if result != tt.expected {
				t.Errorf("findMatchingBrace(%s, %d) = %d, expected %d", tt.content, tt.startPos, result, tt.expected)
			}
		})
	}
}

func TestGenericExtractor_ExtractJSON_Patterns(t *testing.T) {
	extractor := NewGenericExtractor()

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name:     "fenced code block with json",
			content:  "```json\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "fenced code block without language",
			content:  "```\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "xml-like json tags",
			content:  "<json>\n{\"test\": \"value\"}\n</json>",
			expected: `{"test": "value"}`,
		},
		{
			name:     "direct json object",
			content:  `{"test": "value"}`,
			expected: `{"test": "value"}`,
		},
		{
			name:     "direct json array",
			content:  `[{"test": "value"}]`,
			expected: `[{"test": "value"}]`,
		},
		{
			name:     "embedded json in text",
			content:  `Here is your JSON: {"test": "value"} and that's it.`,
			expected: `{"test": "value"}`,
		},
		{
			name:        "no json content",
			content:     "This is just plain text",
			expectError: true,
		},
		{
			name:     "json with whitespace",
			content:  "   {  \"test\"  :  \"value\"  }   ",
			expected: `{  "test"  :  "value"  }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.extractJSON(tt.content)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}
