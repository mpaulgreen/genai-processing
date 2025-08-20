package extractors

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestNewOpenAIExtractor(t *testing.T) {
	extractor := NewOpenAIExtractor()
	if extractor == nil {
		t.Fatal("NewOpenAIExtractor returned nil")
	}
	if extractor.confidence != 0.0 {
		t.Errorf("Expected initial confidence to be 0.0, got %f", extractor.confidence)
	}
}

func TestOpenAIExtractor_CanHandle(t *testing.T) {
	extractor := NewOpenAIExtractor()

	tests := []struct {
		name      string
		modelType string
		expected  bool
	}{
		{"openai", "openai", true},
		{"gpt", "gpt", true},
		{"gpt-3", "gpt-3", true},
		{"gpt-4", "gpt-4", true},
		{"gpt-4o", "gpt-4o", true},
		{"gpt-4-turbo", "gpt-4-turbo", true},
		{"gpt-3.5", "gpt-3.5", true},
		{"gpt-3.5-turbo", "gpt-3.5-turbo", true},
		{"OPENAI", "OPENAI", true},
		{"OpenAI", "OpenAI", true},
		{"GPT-4", "GPT-4", true},
		{"claude", "claude", false},
		{"llama", "llama", false},
		{"empty", "", false},
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

func TestOpenAIExtractor_GetConfidence(t *testing.T) {
	extractor := NewOpenAIExtractor()

	// Initial confidence should be 0.0
	if extractor.GetConfidence() != 0.0 {
		t.Errorf("Expected initial confidence to be 0.0, got %f", extractor.GetConfidence())
	}

	// Set confidence and check
	extractor.confidence = 0.85
	if extractor.GetConfidence() != 0.85 {
		t.Errorf("Expected confidence to be 0.85, got %f", extractor.GetConfidence())
	}
}

func TestOpenAIExtractor_ParseResponse_ValidJSON(t *testing.T) {
	extractor := NewOpenAIExtractor()

	validJSON := `{
		"log_source": "kube-apiserver",
		"verb": "delete",
		"resource": "customresourcedefinitions",
		"resource_name_pattern": "customer",
		"timeframe": "yesterday",
		"exclude_users": ["system:"],
		"limit": 20
	}`

	rawResponse := &types.RawResponse{
		Content: validJSON,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
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

	if query.Resource.GetString() != "customresourcedefinitions" {
		t.Errorf("Expected resource to be 'customresourcedefinitions', got '%s'", query.Resource.GetString())
	}

	if query.ResourceNamePattern != "customer" {
		t.Errorf("Expected resource_name_pattern to be 'customer', got '%s'", query.ResourceNamePattern)
	}

	if query.Timeframe != "yesterday" {
		t.Errorf("Expected timeframe to be 'yesterday', got '%s'", query.Timeframe)
	}

	if len(query.ExcludeUsers) != 1 || query.ExcludeUsers[0] != "system:" {
		t.Errorf("Expected exclude_users to be ['system:'], got %v", query.ExcludeUsers)
	}

	if query.Limit != 20 {
		t.Errorf("Expected limit to be 20, got %d", query.Limit)
	}

	// Check confidence
	confidence := extractor.GetConfidence()
	if confidence <= 0.0 || confidence > 1.0 {
		t.Errorf("Expected confidence to be between 0.0 and 1.0, got %f", confidence)
	}
}

func TestOpenAIExtractor_ParseResponse_FunctionCall(t *testing.T) {
	extractor := NewOpenAIExtractor()

	functionCallResponse := `{
		"function_call": {
			"name": "generate_audit_query",
			"arguments": "{\"log_source\": \"oauth-server\", \"timeframe\": \"1_hour_ago\", \"auth_decision\": \"error\", \"limit\": 20}"
		}
	}`

	rawResponse := &types.RawResponse{
		Content: functionCallResponse,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "oauth-server" {
		t.Errorf("Expected log_source to be 'oauth-server', got '%s'", query.LogSource)
	}

	if query.Timeframe != "1_hour_ago" {
		t.Errorf("Expected timeframe to be '1_hour_ago', got '%s'", query.Timeframe)
	}

	if query.AuthDecision != "error" {
		t.Errorf("Expected auth_decision to be 'error', got '%s'", query.AuthDecision)
	}

	if query.Limit != 20 {
		t.Errorf("Expected limit to be 20, got %d", query.Limit)
	}

	// Check confidence - function calls should have higher confidence
	confidence := extractor.GetConfidence()
	if confidence < 0.95 {
		t.Errorf("Expected high confidence for function call response, got %f", confidence)
	}
}

func TestOpenAIExtractor_ParseResponse_ToolCalls(t *testing.T) {
	extractor := NewOpenAIExtractor()

	toolCallResponse := `{
		"tool_calls": [
			{
				"function": {
					"name": "generate_audit_query",
					"arguments": "{\"log_source\": \"kube-apiserver\", \"verb\": \"create\", \"resource\": \"pods\", \"namespace\": \"production\", \"timeframe\": \"today\", \"exclude_users\": [\"system:\", \"kube-\"], \"limit\": 20}"
				}
			}
		]
	}`

	rawResponse := &types.RawResponse{
		Content: toolCallResponse,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source to be 'kube-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "create" {
		t.Errorf("Expected verb to be 'create', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "pods" {
		t.Errorf("Expected resource to be 'pods', got '%s'", query.Resource.GetString())
	}

	if query.Namespace.GetString() != "production" {
		t.Errorf("Expected namespace to be 'production', got '%s'", query.Namespace.GetString())
	}

	if len(query.ExcludeUsers) != 2 || query.ExcludeUsers[0] != "system:" || query.ExcludeUsers[1] != "kube-" {
		t.Errorf("Expected exclude_users to be ['system:', 'kube-'], got %v", query.ExcludeUsers)
	}
}

func TestOpenAIExtractor_ParseResponse_MarkdownWrapped(t *testing.T) {
	extractor := NewOpenAIExtractor()

	markdownContent := `Here is the JSON response:

` + "```" + `json
{
	"log_source": "openshift-apiserver",
	"verb": "update",
	"resource": "deployments",
	"timeframe": "7_days_ago",
	"limit": 50
}
` + "```" + `

This should extract the JSON properly.`

	rawResponse := &types.RawResponse{
		Content: markdownContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "openshift-apiserver" {
		t.Errorf("Expected log_source to be 'openshift-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "update" {
		t.Errorf("Expected verb to be 'update', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "deployments" {
		t.Errorf("Expected resource to be 'deployments', got '%s'", query.Resource.GetString())
	}

	if query.Timeframe != "7_days_ago" {
		t.Errorf("Expected timeframe to be '7_days_ago', got '%s'", query.Timeframe)
	}

	if query.Limit != 50 {
		t.Errorf("Expected limit to be 50, got %d", query.Limit)
	}
}

func TestOpenAIExtractor_ParseResponse_CodeBlockWithoutLanguage(t *testing.T) {
	extractor := NewOpenAIExtractor()

	codeBlockContent := `Here is the response:

` + "```" + `
{
	"log_source": "oauth-apiserver",
	"auth_decision": "allow",
	"request_uri_pattern": "console",
	"timeframe": "today",
	"limit": 20
}
` + "```" + ``

	rawResponse := &types.RawResponse{
		Content: codeBlockContent,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	if query.LogSource != "oauth-apiserver" {
		t.Errorf("Expected log_source to be 'oauth-apiserver', got '%s'", query.LogSource)
	}

	if query.AuthDecision != "allow" {
		t.Errorf("Expected auth_decision to be 'allow', got '%s'", query.AuthDecision)
	}

	if query.RequestURIPattern != "console" {
		t.Errorf("Expected request_uri_pattern to be 'console', got '%s'", query.RequestURIPattern)
	}
}

func TestOpenAIExtractor_ParseResponse_ArrayFields(t *testing.T) {
	extractor := NewOpenAIExtractor()

	jsonWithArrays := `{
		"log_source": "kube-apiserver",
		"verb": ["create", "delete", "update"],
		"resource": ["pods", "services", "configmaps"],
		"namespace": ["default", "kube-system"],
		"user": ["john.doe", "jane.smith"],
		"timeframe": "today",
		"limit": 20
	}`

	rawResponse := &types.RawResponse{
		Content: jsonWithArrays,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
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
	if len(resources) != 3 || resources[0] != "pods" || resources[1] != "services" || resources[2] != "configmaps" {
		t.Errorf("Expected resources to be ['pods', 'services', 'configmaps'], got %v", resources)
	}

	namespaces := query.Namespace.GetArray()
	if len(namespaces) != 2 || namespaces[0] != "default" || namespaces[1] != "kube-system" {
		t.Errorf("Expected namespaces to be ['default', 'kube-system'], got %v", namespaces)
	}

	users := query.User.GetArray()
	if len(users) != 2 || users[0] != "john.doe" || users[1] != "jane.smith" {
		t.Errorf("Expected users to be ['john.doe', 'jane.smith'], got %v", users)
	}
}

func TestOpenAIExtractor_ParseResponse_ComplexQueries(t *testing.T) {
	extractor := NewOpenAIExtractor()

	complexQuery := `{
		"log_source": "kube-apiserver",
		"verb": "delete",
		"resource": "customresourcedefinitions",
		"resource_name_pattern": "customer",
		"timeframe": "yesterday",
		"exclude_users": ["system:", "kube-"],
		"limit": 20,
		"sort_by": "timestamp",
		"sort_order": "desc",
		"auth_decision": "allow",
		"source_ip": ["192.168.1.1", "10.0.0.1"],
		"group_by": ["username", "resource"],
		"time_range": {
			"start": "2025-01-01T00:00:00Z",
			"end": "2025-01-02T00:00:00Z"
		},
		"business_hours": {
			"outside_only": true,
			"start_hour": 9,
			"end_hour": 17,
			"timezone": "UTC"
		},
		"analysis": {
			"type": "multi_namespace_access",
			"group_by": ["username"],
			"threshold": 5,
			"time_window": "1_hour",
			"sort_by": "count",
			"sort_order": "desc"
		}
	}`

	rawResponse := &types.RawResponse{
		Content: complexQuery,
	}

	query, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query == nil {
		t.Fatal("Expected non-nil query")
	}

	// Check basic fields
	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source to be 'kube-apiserver', got '%s'", query.LogSource)
	}

	// Check array fields
	sourceIPs := query.SourceIP.GetArray()
	if len(sourceIPs) != 2 || sourceIPs[0] != "192.168.1.1" || sourceIPs[1] != "10.0.0.1" {
		t.Errorf("Expected source_ip to be ['192.168.1.1', '10.0.0.1'], got %v", sourceIPs)
	}

	groupBy := query.GroupBy.GetArray()
	if len(groupBy) != 2 || groupBy[0] != "username" || groupBy[1] != "resource" {
		t.Errorf("Expected group_by to be ['username', 'resource'], got %v", groupBy)
	}

	// Check time range
	if query.TimeRange == nil {
		t.Fatal("Expected time_range to be set")
	}

	// Check business hours
	if query.BusinessHours == nil {
		t.Fatal("Expected business_hours to be set")
	}
	if !query.BusinessHours.OutsideOnly {
		t.Error("Expected business_hours.outside_only to be true")
	}
	if query.BusinessHours.StartHour != 9 {
		t.Errorf("Expected business_hours.start_hour to be 9, got %d", query.BusinessHours.StartHour)
	}
	if query.BusinessHours.EndHour != 17 {
		t.Errorf("Expected business_hours.end_hour to be 17, got %d", query.BusinessHours.EndHour)
	}

	// Check analysis
	if query.Analysis == nil {
		t.Fatal("Expected analysis to be set")
	}
	if query.Analysis.Type != "multi_namespace_access" {
		t.Errorf("Expected analysis.type to be 'multi_namespace_access', got '%s'", query.Analysis.Type)
	}
	if query.Analysis.Threshold != 5 {
		t.Errorf("Expected analysis.threshold to be 5, got %d", query.Analysis.Threshold)
	}
}

func TestOpenAIExtractor_ParseResponse_ErrorCases(t *testing.T) {
	extractor := NewOpenAIExtractor()

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
			errorMsg:    "no valid JSON content found in response",
		},
		{
			name:        "no JSON content",
			content:     "This is just plain text with no JSON",
			expectError: true,
			errorMsg:    "no valid JSON content found in response",
		},
		{
			name:        "invalid JSON",
			content:     `{"log_source": "kube-apiserver", "invalid": json}`,
			expectError: true,
			errorMsg:    "invalid JSON content",
		},
		{
			name:        "missing log_source",
			content:     `{"verb": "delete", "resource": "pods"}`,
			expectError: true,
			errorMsg:    "log_source is required",
		},
		{
			name:        "invalid log_source",
			content:     `{"log_source": "invalid-source", "verb": "delete"}`,
			expectError: true,
			errorMsg:    "invalid log_source",
		},
		{
			name:        "invalid limit",
			content:     `{"log_source": "kube-apiserver", "limit": 2000}`,
			expectError: true,
			errorMsg:    "limit must be between 1 and 1000",
		},
		{
			name:        "invalid auth_decision",
			content:     `{"log_source": "oauth-server", "auth_decision": "invalid"}`,
			expectError: false, // Auth decision validation moved to FieldValuesRule - extraction should succeed
			errorMsg:    "",
		},
		{
			name:        "invalid sort_order",
			content:     `{"log_source": "kube-apiserver", "sort_order": "invalid"}`,
			expectError: true,
			errorMsg:    "invalid sort_order",
		},
		{
			name:        "invalid sort_by",
			content:     `{"log_source": "kube-apiserver", "sort_by": "invalid"}`,
			expectError: true,
			errorMsg:    "invalid sort_by",
		},
		{
			name:        "invalid analysis type",
			content:     `{"log_source": "kube-apiserver", "analysis": {"type": "invalid"}}`,
			expectError: true,
			errorMsg:    "invalid analysis.type",
		},
		{
			name:        "invalid function call response",
			content:     `{"function_call": {"name": "test", "arguments": "invalid json"}}`,
			expectError: true,
			errorMsg:    "invalid JSON content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rawResponse *types.RawResponse
			if tt.content != "" {
				rawResponse = &types.RawResponse{Content: tt.content}
			}

			_, err := extractor.ParseResponse(rawResponse, "gpt-4")
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !contains(err.Error(), tt.errorMsg) {
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

func TestOpenAIExtractor_ConfidenceCalculation(t *testing.T) {
	extractor := NewOpenAIExtractor()

	// Test with well-formed query (should have high confidence)
	wellFormedQuery := `{
		"log_source": "kube-apiserver",
		"verb": "delete",
		"resource": "pods",
		"namespace": "production",
		"user": "john.doe",
		"exclude_users": ["system:"],
		"timeframe": "today",
		"limit": 20
	}`

	rawResponse := &types.RawResponse{Content: wellFormedQuery}
	_, err := extractor.ParseResponse(rawResponse, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	confidence := extractor.GetConfidence()
	if confidence < 0.9 {
		t.Errorf("Expected high confidence for well-formed query, got %f", confidence)
	}

	// Test with function call response (should have higher confidence)
	extractor2 := NewOpenAIExtractor()
	functionCallQuery := `{
		"function_call": {
			"name": "generate_audit_query",
			"arguments": "{\"log_source\": \"kube-apiserver\", \"verb\": \"delete\", \"resource\": \"pods\", \"namespace\": \"production\", \"user\": \"john.doe\", \"exclude_users\": [\"system:\"], \"timeframe\": \"today\", \"limit\": 20}"
		}
	}`

	rawResponse2 := &types.RawResponse{Content: functionCallQuery}
	_, err = extractor2.ParseResponse(rawResponse2, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	confidence2 := extractor2.GetConfidence()
	if confidence2 < 0.95 {
		t.Errorf("Expected very high confidence for function call response, got %f", confidence2)
	}

	// Test with minimal query (should have lower confidence)
	extractor3 := NewOpenAIExtractor()
	minimalQuery := `{
		"log_source": "kube-apiserver"
	}`

	rawResponse3 := &types.RawResponse{Content: minimalQuery}
	_, err = extractor3.ParseResponse(rawResponse3, "gpt-4")
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	confidence3 := extractor3.GetConfidence()
	if confidence3 > 0.9 {
		t.Errorf("Expected lower confidence for minimal query, got %f", confidence3)
	}
}

func TestOpenAIExtractor_FindMatchingBrace(t *testing.T) {
	extractor := NewOpenAIExtractor()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.findMatchingBrace(tt.content, tt.startPos)
			if result != tt.expected {
				t.Errorf("findMatchingBrace(%s, %d) = %d, expected %d", tt.content, tt.startPos, result, tt.expected)
			}
		})
	}
}

func TestOpenAIExtractor_IsFunctionCallResponse(t *testing.T) {
	extractor := NewOpenAIExtractor()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "function call response",
			content:  `{"function_call": {"name": "test", "arguments": "{}"}}`,
			expected: true,
		},
		{
			name:     "tool calls response",
			content:  `{"tool_calls": [{"function": {"name": "test", "arguments": "{}"}}]}`,
			expected: true,
		},
		{
			name:     "with name field",
			content:  `{"name": "test", "arguments": "{}"}`,
			expected: true,
		},
		{
			name:     "with arguments field",
			content:  `{"arguments": "{}"}`,
			expected: true,
		},
		{
			name:     "regular JSON",
			content:  `{"log_source": "kube-apiserver"}`,
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.isFunctionCallResponse(tt.content)
			if result != tt.expected {
				t.Errorf("isFunctionCallResponse(%s) = %v, expected %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestOpenAIExtractor_ExtractFunctionCallContent(t *testing.T) {
	extractor := NewOpenAIExtractor()

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name:     "function call with arguments",
			content:  `{"function_call": {"name": "test", "arguments": "{\"log_source\": \"kube-apiserver\"}"}}`,
			expected: `{"log_source": "kube-apiserver"}`,
		},
		{
			name:     "tool calls with arguments",
			content:  `{"tool_calls": [{"function": {"name": "test", "arguments": "{\"log_source\": \"oauth-server\"}"}}]}`,
			expected: `{"log_source": "oauth-server"}`,
		},
		{
			name:     "with content field",
			content:  `{"content": "{\"log_source\": \"openshift-apiserver\"}"}`,
			expected: `{"log_source": "openshift-apiserver"}`,
		},
		{
			name:        "no valid content",
			content:     `{"function_call": {"name": "test"}}`,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			content:     `{"function_call": {"name": "test", "arguments": "invalid"}}`,
			expected:    "invalid",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.extractFunctionCallContent(tt.content)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}
