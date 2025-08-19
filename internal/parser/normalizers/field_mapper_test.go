package normalizers

import (
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewFieldMapper(t *testing.T) {
	mapper := NewFieldMapper()
	if mapper == nil {
		t.Fatal("Expected non-nil field mapper")
	}
}

func TestFieldMapper_MapFields_NilQuery(t *testing.T) {
	mapper := NewFieldMapper()
	result, err := mapper.MapFields(nil)
	
	if result != nil {
		t.Error("Expected nil result for nil query")
	}
	
	if err == nil {
		t.Error("Expected error for nil query")
	}
	
	if err.Error() != "field mapper: query is nil" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestFieldMapper_LogSourceAliases(t *testing.T) {
	mapper := NewFieldMapper()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "oauth_api_server_underscore",
			input:    "oauth_api_server",
			expected: "oauth-server",
		},
		{
			name:     "oauth_apiserver_no_separator",
			input:    "oauth-apiserver",
			expected: "oauth-server",
		},
		{
			name:     "oauthserver_compact",
			input:    "oauthserver",
			expected: "oauth-server",
		},
		{
			name:     "openshiftapiserver_compact",
			input:    "openshiftapiserver",
			expected: "openshift-apiserver",
		},
		{
			name:     "openshift_api_server_underscore",
			input:    "openshift_api_server",
			expected: "openshift-apiserver",
		},
		{
			name:     "kube_api_server_underscore",
			input:    "kube_api_server",
			expected: "kube-apiserver",
		},
		{
			name:     "kubeapiserver_compact",
			input:    "kubeapiserver",
			expected: "kube-apiserver",
		},
		{
			name:     "case_insensitive_oauth",
			input:    "OAUTH-APISERVER",
			expected: "oauth-server",
		},
		{
			name:     "case_insensitive_openshift",
			input:    "OpenShift_API_Server",
			expected: "openshift-apiserver",
		},
		{
			name:     "whitespace_trimming",
			input:    "  oauth_api_server  ",
			expected: "oauth-server",
		},
		{
			name:     "already_canonical",
			input:    "kube-apiserver",
			expected: "kube-apiserver",
		},
		{
			name:     "unknown_log_source",
			input:    "unknown-server",
			expected: "unknown-server",
		},
		{
			name:     "empty_log_source",
			input:    "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: tt.input,
			}
			
			result, err := mapper.MapFields(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.LogSource != tt.expected {
				t.Errorf("Expected log_source '%s', got '%s'", tt.expected, result.LogSource)
			}
		})
	}
}

func TestFieldMapper_VerbMapping(t *testing.T) {
	mapper := NewFieldMapper()
	
	tests := []struct {
		name     string
		input    types.StringOrArray
		expected types.StringOrArray
	}{
		{
			name:     "single_verb_create",
			input:    *types.NewStringOrArray("create"),
			expected: *types.NewStringOrArray("create"),
		},
		{
			name:     "single_verb_post_to_create",
			input:    *types.NewStringOrArray("post"),
			expected: *types.NewStringOrArray("create"),
		},
		{
			name:     "single_verb_read_to_get",
			input:    *types.NewStringOrArray("read"),
			expected: *types.NewStringOrArray("get"),
		},
		{
			name:     "single_verb_case_insensitive",
			input:    *types.NewStringOrArray("POST"),
			expected: *types.NewStringOrArray("create"),
		},
		{
			name:     "single_verb_whitespace",
			input:    *types.NewStringOrArray("  patch  "),
			expected: *types.NewStringOrArray("patch"),
		},
		{
			name:     "multiple_verbs_mixed",
			input:    *types.NewStringOrArray([]string{"post", "get", "READ", "  delete  "}),
			expected: *types.NewStringOrArray([]string{"create", "get", "get", "delete"}),
		},
		{
			name:     "multiple_verbs_with_empty",
			input:    *types.NewStringOrArray([]string{"post", "", "get", "   "}),
			expected: *types.NewStringOrArray([]string{"create", "get"}),
		},
		{
			name:     "unknown_verb",
			input:    *types.NewStringOrArray("unknown"),
			expected: *types.NewStringOrArray("unknown"),
		},
		{
			name:     "empty_array",
			input:    *types.NewStringOrArray([]string{}),
			expected: *types.NewStringOrArray([]string{}),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      tt.input,
			}
			
			result, err := mapper.MapFields(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Compare string representations for easier debugging
			if tt.expected.IsString() && result.Verb.IsString() {
				if result.Verb.GetString() != tt.expected.GetString() {
					t.Errorf("Expected verb '%s', got '%s'", tt.expected.GetString(), result.Verb.GetString())
				}
			} else if tt.expected.IsArray() && result.Verb.IsArray() {
				expectedArr := tt.expected.GetArray()
				resultArr := result.Verb.GetArray()
				
				if len(expectedArr) != len(resultArr) {
					t.Errorf("Expected %d verbs, got %d", len(expectedArr), len(resultArr))
					return
				}
				
				for i, expected := range expectedArr {
					if resultArr[i] != expected {
						t.Errorf("Expected verb[%d] '%s', got '%s'", i, expected, resultArr[i])
					}
				}
			} else {
				t.Errorf("Type mismatch: expected and result have different types")
			}
		})
	}
}

func TestFieldMapper_ResponseStatusMapping(t *testing.T) {
	mapper := NewFieldMapper()
	
	tests := []struct {
		name     string
		input    types.StringOrArray
		expected types.StringOrArray
	}{
		{
			name:     "single_ok_to_200",
			input:    *types.NewStringOrArray("ok"),
			expected: *types.NewStringOrArray("200"),
		},
		{
			name:     "single_OK_case_insensitive",
			input:    *types.NewStringOrArray("OK"),
			expected: *types.NewStringOrArray("200"),
		},
		{
			name:     "single_200_unchanged",
			input:    *types.NewStringOrArray("200"),
			expected: *types.NewStringOrArray("200"),
		},
		{
			name:     "single_404_unchanged",
			input:    *types.NewStringOrArray("404"),
			expected: *types.NewStringOrArray("404"),
		},
		{
			name:     "multiple_status_mixed",
			input:    *types.NewStringOrArray([]string{"ok", "404", "500"}),
			expected: *types.NewStringOrArray([]string{"200", "404", "500"}),
		},
		{
			name:     "whitespace_trimming",
			input:    *types.NewStringOrArray("  ok  "),
			expected: *types.NewStringOrArray("200"),
		},
		{
			name:     "unknown_status",
			input:    *types.NewStringOrArray("unknown"),
			expected: *types.NewStringOrArray("unknown"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: tt.input,
			}
			
			result, err := mapper.MapFields(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Compare results
			if tt.expected.IsString() && result.ResponseStatus.IsString() {
				if result.ResponseStatus.GetString() != tt.expected.GetString() {
					t.Errorf("Expected response_status '%s', got '%s'", tt.expected.GetString(), result.ResponseStatus.GetString())
				}
			} else if tt.expected.IsArray() && result.ResponseStatus.IsArray() {
				expectedArr := tt.expected.GetArray()
				resultArr := result.ResponseStatus.GetArray()
				
				if len(expectedArr) != len(resultArr) {
					t.Errorf("Expected %d statuses, got %d", len(expectedArr), len(resultArr))
					return
				}
				
				for i, expected := range expectedArr {
					if resultArr[i] != expected {
						t.Errorf("Expected status[%d] '%s', got '%s'", i, expected, resultArr[i])
					}
				}
			}
		})
	}
}

func TestFieldMapper_ComprehensiveMapping(t *testing.T) {
	mapper := NewFieldMapper()
	
	query := &types.StructuredQuery{
		LogSource:      "OAUTH_API_SERVER",
		Verb:           *types.NewStringOrArray([]string{"POST", "read", "  patch  "}),
		ResponseStatus: *types.NewStringOrArray([]string{"ok", "  404  ", "500"}),
		Resource:       *types.NewStringOrArray("pods"),
		Namespace:      *types.NewStringOrArray("default"),
		User:           *types.NewStringOrArray("admin"),
		Timeframe:      "today",
		Limit:          10,
	}
	
	result, err := mapper.MapFields(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify log source mapping
	if result.LogSource != "oauth-server" {
		t.Errorf("Expected log_source 'oauth-server', got '%s'", result.LogSource)
	}
	
	// Verify verb mapping
	expectedVerbs := []string{"create", "get", "patch"}
	resultVerbs := result.Verb.GetArray()
	if len(resultVerbs) != len(expectedVerbs) {
		t.Errorf("Expected %d verbs, got %d", len(expectedVerbs), len(resultVerbs))
	}
	for i, expected := range expectedVerbs {
		if resultVerbs[i] != expected {
			t.Errorf("Expected verb[%d] '%s', got '%s'", i, expected, resultVerbs[i])
		}
	}
	
	// Verify response status mapping
	expectedStatuses := []string{"200", "404", "500"}
	resultStatuses := result.ResponseStatus.GetArray()
	if len(resultStatuses) != len(expectedStatuses) {
		t.Errorf("Expected %d statuses, got %d", len(expectedStatuses), len(resultStatuses))
	}
	for i, expected := range expectedStatuses {
		if resultStatuses[i] != expected {
			t.Errorf("Expected status[%d] '%s', got '%s'", i, expected, resultStatuses[i])
		}
	}
	
	// Verify other fields unchanged
	if result.Resource.GetString() != "pods" {
		t.Errorf("Expected resource unchanged, got '%s'", result.Resource.GetString())
	}
	if result.Namespace.GetString() != "default" {
		t.Errorf("Expected namespace unchanged, got '%s'", result.Namespace.GetString())
	}
	if result.User.GetString() != "admin" {
		t.Errorf("Expected user unchanged, got '%s'", result.User.GetString())
	}
	if result.Timeframe != "today" {
		t.Errorf("Expected timeframe unchanged, got '%s'", result.Timeframe)
	}
	if result.Limit != 10 {
		t.Errorf("Expected limit unchanged, got %d", result.Limit)
	}
}

func TestFieldMapper_EdgeCases(t *testing.T) {
	mapper := NewFieldMapper()
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		description string
	}{
		{
			name: "empty_query",
			query: &types.StructuredQuery{},
			description: "Empty query with defaults",
		},
		{
			name: "only_log_source",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			description: "Query with only log source",
		},
		{
			name: "empty_string_arrays",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray(""),
				Resource:  *types.NewStringOrArray([]string{}),
			},
			description: "Query with empty string arrays",
		},
		{
			name: "whitespace_only_fields",
			query: &types.StructuredQuery{
				LogSource: "   ",
				Verb:      *types.NewStringOrArray("   "),
			},
			description: "Query with whitespace-only fields",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapper.MapFields(tt.query)
			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.description, err)
			}
			
			if result == nil {
				t.Fatalf("Expected non-nil result for %s", tt.description)
			}
			
			// Verify result is a proper copy, not the same instance
			if result == tt.query {
				t.Error("Expected result to be a copy, not the same instance")
			}
		})
	}
}

func TestFieldMapper_PreservesOtherFields(t *testing.T) {
	mapper := NewFieldMapper()
	
	now := time.Now()
	timeRange := &types.TimeRange{
		Start: now.Add(-time.Hour),
		End:   now,
	}
	
	query := &types.StructuredQuery{
		LogSource:                  "kube-apiserver",
		Verb:                       *types.NewStringOrArray("get"),
		Resource:                   *types.NewStringOrArray("pods"),
		Namespace:                  *types.NewStringOrArray("default"),
		User:                       *types.NewStringOrArray("admin"),
		Timeframe:                  "today",
		Limit:                      20,
		ResponseStatus:             *types.NewStringOrArray("200"),
		SourceIP:                   *types.NewStringOrArray("192.168.1.1"),
		GroupBy:                    *types.NewStringOrArray("user"),
		SortBy:                     "timestamp",
		SortOrder:                  "desc",
		Subresource:                "status",
		AuthDecision:               "allow",
		ResourceNamePattern:        "test-.*",
		UserPattern:                "admin.*",
		NamespacePattern:           "prod-.*",
		RequestURIPattern:          "/api/.*",
		AuthorizationReasonPattern: "RBAC.*",
		ResponseMessagePattern:     "success.*",
		MissingAnnotation:          "deprecated",
		RequestObjectFilter:        "spec.replicas",
		ExcludeUsers:               []string{"system:admin"},
		ExcludeResources:           []string{"events"},
		TimeRange:                  timeRange,
		BusinessHours:              &types.BusinessHours{StartHour: 9, EndHour: 17},
		IncludeChanges:             true,
	}
	
	result, err := mapper.MapFields(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify all non-mapped fields are preserved
	if result.Limit != query.Limit {
		t.Errorf("Expected limit preserved")
	}
	if result.SortBy != query.SortBy {
		t.Errorf("Expected sort_by preserved")
	}
	if result.SortOrder != query.SortOrder {
		t.Errorf("Expected sort_order preserved")
	}
	if result.Subresource != query.Subresource {
		t.Errorf("Expected subresource preserved")
	}
	if result.AuthDecision != query.AuthDecision {
		t.Errorf("Expected auth_decision preserved")
	}
	if result.ResourceNamePattern != query.ResourceNamePattern {
		t.Errorf("Expected resource_name_pattern preserved")
	}
	if result.UserPattern != query.UserPattern {
		t.Errorf("Expected user_pattern preserved")
	}
	if result.NamespacePattern != query.NamespacePattern {
		t.Errorf("Expected namespace_pattern preserved")
	}
	if result.RequestURIPattern != query.RequestURIPattern {
		t.Errorf("Expected request_uri_pattern preserved")
	}
	if result.AuthorizationReasonPattern != query.AuthorizationReasonPattern {
		t.Errorf("Expected authorization_reason_pattern preserved")
	}
	if result.ResponseMessagePattern != query.ResponseMessagePattern {
		t.Errorf("Expected response_message_pattern preserved")
	}
	if result.MissingAnnotation != query.MissingAnnotation {
		t.Errorf("Expected missing_annotation preserved")
	}
	if result.RequestObjectFilter != query.RequestObjectFilter {
		t.Errorf("Expected request_object_filter preserved")
	}
	if len(result.ExcludeUsers) != len(query.ExcludeUsers) {
		t.Errorf("Expected exclude_users preserved")
	}
	if len(result.ExcludeResources) != len(query.ExcludeResources) {
		t.Errorf("Expected exclude_resources preserved")
	}
	if result.TimeRange != query.TimeRange {
		t.Errorf("Expected time_range preserved")
	}
	if result.BusinessHours != query.BusinessHours {
		t.Errorf("Expected business_hours preserved")
	}
	if result.IncludeChanges != query.IncludeChanges {
		t.Errorf("Expected include_changes preserved")
	}
}

// Benchmark performance of field mapping
func BenchmarkFieldMapper_MapFields(b *testing.B) {
	mapper := NewFieldMapper()
	query := &types.StructuredQuery{
		LogSource:      "oauth_api_server",
		Verb:           *types.NewStringOrArray([]string{"post", "read", "patch"}),
		ResponseStatus: *types.NewStringOrArray([]string{"ok", "404", "500"}),
		Resource:       *types.NewStringOrArray("pods"),
		Namespace:      *types.NewStringOrArray("default"),
		User:           *types.NewStringOrArray("admin"),
		Timeframe:      "today",
		Limit:          20,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mapper.MapFields(query)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func TestFieldMapper_PerformanceTarget(t *testing.T) {
	mapper := NewFieldMapper()
	query := &types.StructuredQuery{
		LogSource:      "oauth_api_server",
		Verb:           *types.NewStringOrArray([]string{"post", "read", "patch"}),
		ResponseStatus: *types.NewStringOrArray([]string{"ok", "404", "500"}),
	}
	
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		_, err := mapper.MapFields(query)
		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	// Target: < 1ms per operation (very generous for field mapping)
	target := time.Millisecond
	if avgDuration > target {
		t.Errorf("Performance target missed: average %v > target %v", avgDuration, target)
	}
	
	t.Logf("Performance: %v per mapping operation (%d iterations)", avgDuration, iterations)
}