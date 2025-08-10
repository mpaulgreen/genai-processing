package extractors

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestNewOllamaExtractor(t *testing.T) {
	extractor := NewOllamaExtractor()

	if extractor == nil {
		t.Fatal("NewOllamaExtractor returned nil")
	}

	if extractor.GetConfidence() != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", extractor.GetConfidence())
	}
}

func TestOllamaExtractor_CanHandle(t *testing.T) {
	extractor := NewOllamaExtractor()

	tests := []struct {
		name      string
		modelType string
		expected  bool
	}{
		{"ollama", "ollama", true},
		{"llama", "llama", true},
		{"llama3", "llama3", true},
		{"llama2", "llama2", true},
		{"local_llama", "local_llama", true},
		{"llama3.1:8b", "llama3.1:8b", true},
		{"openai", "openai", false},
		{"claude", "claude", false},
		{"gpt-4", "gpt-4", false},
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

func TestOllamaExtractor_ParseResponse_ValidJSON(t *testing.T) {
	extractor := NewOllamaExtractor()

	response := &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 100}`,
	}

	query, err := extractor.ParseResponse(response, "llama3.1:8b")

	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source 'kube-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "get" {
		t.Errorf("Expected verb 'get', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "pods" {
		t.Errorf("Expected resource 'pods', got '%s'", query.Resource.GetString())
	}

	if query.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", query.Limit)
	}
}

func TestOllamaExtractor_ParseResponse_MarkdownWrapped(t *testing.T) {
	extractor := NewOllamaExtractor()

	response := &types.RawResponse{
		Content: "Here is the structured query:\n\n```json\n{\n  \"log_source\": \"openshift-apiserver\",\n  \"verb\": \"create\",\n  \"resource\": \"namespaces\",\n  \"timeframe\": \"last 24 hours\"\n}\n```",
	}

	query, err := extractor.ParseResponse(response, "llama3.1:8b")

	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query.LogSource != "openshift-apiserver" {
		t.Errorf("Expected log_source 'openshift-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "create" {
		t.Errorf("Expected verb 'create', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "namespaces" {
		t.Errorf("Expected resource 'namespaces', got '%s'", query.Resource.GetString())
	}

	if query.Timeframe != "last 24 hours" {
		t.Errorf("Expected timeframe 'last 24 hours', got '%s'", query.Timeframe)
	}
}

func TestOllamaExtractor_ParseResponse_XMLWrapped(t *testing.T) {
	extractor := NewOllamaExtractor()

	response := &types.RawResponse{
		Content: `Here is the query in JSON format:

<json>
{
  "log_source": "oauth-server",
  "verb": "list",
  "resource": "secrets",
  "limit": 50
}
</json>`,
	}

	query, err := extractor.ParseResponse(response, "llama3.1:8b")

	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query.LogSource != "oauth-server" {
		t.Errorf("Expected log_source 'oauth-server', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "list" {
		t.Errorf("Expected verb 'list', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "secrets" {
		t.Errorf("Expected resource 'secrets', got '%s'", query.Resource.GetString())
	}

	if query.Limit != 50 {
		t.Errorf("Expected limit 50, got %d", query.Limit)
	}
}

func TestOllamaExtractor_ParseResponse_ErrorCases(t *testing.T) {
	extractor := NewOllamaExtractor()

	tests := []struct {
		name    string
		content string
	}{
		{"nil response", ""},
		{"empty content", ""},
		{"no JSON content", "This is not JSON"},
		{"invalid JSON", "{invalid json}"},
		{"missing log_source", `{"verb": "get", "resource": "pods"}`},
		{"invalid log_source", `{"log_source": "invalid", "verb": "get"}`},
		{"exceeds limit", `{"log_source": "kube-apiserver", "limit": 2000}`},
		{"invalid timeframe", `{"log_source": "kube-apiserver", "timeframe": "invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &types.RawResponse{Content: tt.content}
			_, err := extractor.ParseResponse(response, "llama3.1:8b")

			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestOllamaExtractor_ParseResponse_ComplexQueries(t *testing.T) {
	extractor := NewOllamaExtractor()

	response := &types.RawResponse{
		Content: "```json\n{\n  \"log_source\": \"kube-apiserver\",\n  \"verb\": \"delete\",\n  \"resource\": \"customresourcedefinitions\",\n  \"user\": \"admin\",\n  \"namespace\": \"default\",\n  \"timeframe\": \"last 7 days\",\n  \"limit\": 500,\n  \"sort_by\": \"timestamp\",\n  \"sort_order\": \"desc\"\n}\n```",
	}

	query, err := extractor.ParseResponse(response, "llama3.1:8b")

	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if query.LogSource != "kube-apiserver" {
		t.Errorf("Expected log_source 'kube-apiserver', got '%s'", query.LogSource)
	}

	if query.Verb.GetString() != "delete" {
		t.Errorf("Expected verb 'delete', got '%s'", query.Verb.GetString())
	}

	if query.Resource.GetString() != "customresourcedefinitions" {
		t.Errorf("Expected resource 'customresourcedefinitions', got '%s'", query.Resource.GetString())
	}

	if query.User.GetString() != "admin" {
		t.Errorf("Expected user 'admin', got '%s'", query.User.GetString())
	}

	if query.Namespace.GetString() != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", query.Namespace.GetString())
	}

	if query.Timeframe != "last 7 days" {
		t.Errorf("Expected timeframe 'last 7 days', got '%s'", query.Timeframe)
	}

	if query.Limit != 500 {
		t.Errorf("Expected limit 500, got %d", query.Limit)
	}

	if query.SortBy != "timestamp" {
		t.Errorf("Expected sort_by 'timestamp', got '%s'", query.SortBy)
	}

	if query.SortOrder != "desc" {
		t.Errorf("Expected sort_order 'desc', got '%s'", query.SortOrder)
	}
}

func TestOllamaExtractor_ValidateQuery(t *testing.T) {
	extractor := NewOllamaExtractor()

	tests := []struct {
		name    string
		query   *types.StructuredQuery
		wantErr bool
	}{
		{
			name: "valid query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			wantErr: false,
		},
		{
			name:    "nil query",
			query:   nil,
			wantErr: true,
		},
		{
			name: "missing log_source",
			query: &types.StructuredQuery{
				Verb:     *types.NewStringOrArray("get"),
				Resource: *types.NewStringOrArray("pods"),
			},
			wantErr: true,
		},
		{
			name: "invalid log_source",
			query: &types.StructuredQuery{
				LogSource: "invalid",
				Verb:      *types.NewStringOrArray("get"),
			},
			wantErr: true,
		},
		{
			name: "exceeds limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     1500,
			},
			wantErr: true,
		},
		{
			name: "invalid timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extractor.validateQuery(tt.query)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
