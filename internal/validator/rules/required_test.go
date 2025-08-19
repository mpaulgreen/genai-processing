package rules

import (
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewRequiredFieldsRule(t *testing.T) {
	requiredFields := []string{"log_source", "verb", "resource"}
	rule := NewRequiredFieldsRule(requiredFields)

	if rule == nil {
		t.Fatal("NewRequiredFieldsRule returned nil")
	}
	if !rule.IsEnabled() {
		t.Error("Expected rule to be enabled by default")
	}
	if rule.GetRuleName() != "required_fields_validation" {
		t.Errorf("Expected rule name 'required_fields_validation', got '%s'", rule.GetRuleName())
	}
	if rule.GetSeverity() != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", rule.GetSeverity())
	}
}

func TestRequiredFieldsRule_LogSourceValidation(t *testing.T) {
	tests := []struct {
		name        string
		logSource   string
		shouldFail  bool
		description string
	}{
		{
			name:        "valid_kube_apiserver",
			logSource:   "kube-apiserver",
			shouldFail:  false,
			description: "kube-apiserver should be valid",
		},
		{
			name:        "valid_openshift_apiserver",
			logSource:   "openshift-apiserver",
			shouldFail:  false,
			description: "openshift-apiserver should be valid",
		},
		{
			name:        "valid_oauth_server",
			logSource:   "oauth-server",
			shouldFail:  false,
			description: "oauth-server should be valid",
		},
		{
			name:        "valid_oauth_apiserver",
			logSource:   "oauth-apiserver",
			shouldFail:  false,
			description: "oauth-apiserver should be valid",
		},
		{
			name:        "empty_log_source",
			logSource:   "",
			shouldFail:  true,
			description: "empty log_source should fail",
		},
		{
			name:        "whitespace_log_source",
			logSource:   "   ",
			shouldFail:  true,
			description: "whitespace-only log_source should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRequiredFieldsRule([]string{"log_source"})
			query := &types.StructuredQuery{
				LogSource: tt.logSource,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail for %s: %s", tt.name, tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass for %s: %s. Errors: %v", tt.name, tt.description, result.Errors)
			}
		})
	}
}

func TestRequiredFieldsRule_MultipleFields(t *testing.T) {
	tests := []struct {
		name           string
		query          *types.StructuredQuery
		requiredFields []string
		expectedErrors int
		description    string
	}{
		{
			name: "all_required_fields_present",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			requiredFields: []string{"log_source", "verb", "resource"},
			expectedErrors: 0,
			description:    "All required fields present should pass",
		},
		{
			name: "missing_log_source",
			query: &types.StructuredQuery{
				Verb:     *types.NewStringOrArray("get"),
				Resource: *types.NewStringOrArray("pods"),
			},
			requiredFields: []string{"log_source", "verb", "resource"},
			expectedErrors: 1,
			description:    "Missing log_source should fail",
		},
		{
			name: "missing_multiple_fields",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			requiredFields: []string{"log_source", "verb", "resource", "namespace"},
			expectedErrors: 3, // verb, resource, namespace missing
			description:    "Multiple missing fields should be detected",
		},
		{
			name: "empty_stringorarray_fields",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      types.StringOrArray{}, // Empty StringOrArray
				Resource:  *types.NewStringOrArray(""),  // StringOrArray with empty string
			},
			requiredFields: []string{"log_source", "verb", "resource"},
			expectedErrors: 2, // verb and resource are empty
			description:    "Empty StringOrArray fields should be detected as missing",
		},
		{
			name: "zero_limit_field",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     0, // Zero value
			},
			requiredFields: []string{"log_source", "limit"},
			expectedErrors: 1, // limit is zero
			description:    "Zero limit should be detected as missing",
		},
		{
			name: "valid_limit_field",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     20,
			},
			requiredFields: []string{"log_source", "limit"},
			expectedErrors: 0,
			description:    "Valid limit should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRequiredFieldsRule(tt.requiredFields)
			result := rule.Validate(tt.query)

			if tt.expectedErrors == 0 && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if tt.expectedErrors > 0 && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %s. Errors: %v", 
					tt.expectedErrors, len(result.Errors), tt.description, result.Errors)
			}

			// Verify error messages contain field names
			if tt.expectedErrors > 0 {
				for _, err := range result.Errors {
					if err == "" {
						t.Error("Error message should not be empty")
					}
				}
				if len(result.Recommendations) == 0 {
					t.Error("Failed validation should include recommendations")
				}
			}
		})
	}
}

func TestRequiredFieldsRule_AllFieldTypes(t *testing.T) {
	// Test all field types that are supported by isFieldPresent
	tests := []struct {
		fieldName   string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			fieldName: "log_source",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			shouldPass:  true,
			description: "log_source field validation",
		},
		{
			fieldName: "verb",
			query: &types.StructuredQuery{
				Verb: *types.NewStringOrArray("get"),
			},
			shouldPass:  true,
			description: "verb field validation",
		},
		{
			fieldName: "resource",
			query: &types.StructuredQuery{
				Resource: *types.NewStringOrArray("pods"),
			},
			shouldPass:  true,
			description: "resource field validation",
		},
		{
			fieldName: "namespace",
			query: &types.StructuredQuery{
				Namespace: *types.NewStringOrArray("default"),
			},
			shouldPass:  true,
			description: "namespace field validation",
		},
		{
			fieldName: "user",
			query: &types.StructuredQuery{
				User: *types.NewStringOrArray("test@company.com"),
			},
			shouldPass:  true,
			description: "user field validation",
		},
		{
			fieldName: "timeframe",
			query: &types.StructuredQuery{
				Timeframe: "24_hours_ago",
			},
			shouldPass:  true,
			description: "timeframe field validation",
		},
		{
			fieldName: "limit",
			query: &types.StructuredQuery{
				Limit: 20,
			},
			shouldPass:  true,
			description: "limit field validation",
		},
		{
			fieldName: "response_status",
			query: &types.StructuredQuery{
				ResponseStatus: *types.NewStringOrArray("200"),
			},
			shouldPass:  true,
			description: "response_status field validation",
		},
		{
			fieldName: "source_ip",
			query: &types.StructuredQuery{
				SourceIP: *types.NewStringOrArray("192.168.1.1"),
			},
			shouldPass:  true,
			description: "source_ip field validation",
		},
		{
			fieldName: "group_by",
			query: &types.StructuredQuery{
				GroupBy: *types.NewStringOrArray("user"),
			},
			shouldPass:  true,
			description: "group_by field validation",
		},
		{
			fieldName: "sort_by",
			query: &types.StructuredQuery{
				SortBy: "timestamp",
			},
			shouldPass:  true,
			description: "sort_by field validation",
		},
		{
			fieldName: "sort_order",
			query: &types.StructuredQuery{
				SortOrder: "desc",
			},
			shouldPass:  true,
			description: "sort_order field validation",
		},
		{
			fieldName: "subresource",
			query: &types.StructuredQuery{
				Subresource: "exec",
			},
			shouldPass:  true,
			description: "subresource field validation",
		},
		{
			fieldName: "auth_decision",
			query: &types.StructuredQuery{
				AuthDecision: "allow",
			},
			shouldPass:  true,
			description: "auth_decision field validation",
		},
		{
			fieldName: "resource_name_pattern",
			query: &types.StructuredQuery{
				ResourceNamePattern: "test-.*",
			},
			shouldPass:  true,
			description: "resource_name_pattern field validation",
		},
		{
			fieldName: "user_pattern",
			query: &types.StructuredQuery{
				UserPattern: ".*@company.com",
			},
			shouldPass:  true,
			description: "user_pattern field validation",
		},
		{
			fieldName: "namespace_pattern",
			query: &types.StructuredQuery{
				NamespacePattern: "test-.*",
			},
			shouldPass:  true,
			description: "namespace_pattern field validation",
		},
		{
			fieldName: "request_uri_pattern",
			query: &types.StructuredQuery{
				RequestURIPattern: "/api/v1/.*",
			},
			shouldPass:  true,
			description: "request_uri_pattern field validation",
		},
		{
			fieldName: "authorization_reason_pattern",
			query: &types.StructuredQuery{
				AuthorizationReasonPattern: "allowed",
			},
			shouldPass:  true,
			description: "authorization_reason_pattern field validation",
		},
		{
			fieldName: "response_message_pattern",
			query: &types.StructuredQuery{
				ResponseMessagePattern: "success",
			},
			shouldPass:  true,
			description: "response_message_pattern field validation",
		},
		{
			fieldName: "missing_annotation",
			query: &types.StructuredQuery{
				MissingAnnotation: "test.annotation",
			},
			shouldPass:  true,
			description: "missing_annotation field validation",
		},
		{
			fieldName: "request_object_filter",
			query: &types.StructuredQuery{
				RequestObjectFilter: "filter",
			},
			shouldPass:  true,
			description: "request_object_filter field validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			rule := NewRequiredFieldsRule([]string{tt.fieldName})
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected field '%s' to pass validation: %s. Errors: %v", 
					tt.fieldName, tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected field '%s' to fail validation: %s", tt.fieldName, tt.description)
			}
		})
	}
}

func TestRequiredFieldsRule_UnknownField(t *testing.T) {
	rule := NewRequiredFieldsRule([]string{"unknown_field"})
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	result := rule.Validate(query)

	// Unknown fields should be considered missing and fail validation
	if result.IsValid {
		t.Error("Expected validation to fail for unknown required field")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error for unknown field, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestRequiredFieldsRule_BasicQueryRequirements(t *testing.T) {
	// Test basic query requirements from functional tests
	// Based on basic_queries.md patterns
	basicRequiredFields := []string{"log_source"}
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "minimal_valid_basic_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("delete"),
				Resource:  *types.NewStringOrArray("pods"),
				Timeframe: "24_hours_ago",
			},
			shouldPass:  true,
			description: "Minimal basic query should pass with only log_source required",
		},
		{
			name: "query_with_exclude_users",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				Verb:         *types.NewStringOrArray("create"),
				Resource:     *types.NewStringOrArray("secrets"),
				ExcludeUsers: []string{"system:", "kube-"},
			},
			shouldPass:  true,
			description: "Query with exclude patterns should pass",
		},
		{
			name: "oauth_server_query",
			query: &types.StructuredQuery{
				LogSource:    "oauth-server",
				AuthDecision: "forbid",
				Timeframe:    "1_hour_ago",
			},
			shouldPass:  true,
			description: "OAuth server query should pass",
		},
		{
			name: "missing_log_source",
			query: &types.StructuredQuery{
				Verb:     *types.NewStringOrArray("get"),
				Resource: *types.NewStringOrArray("pods"),
			},
			shouldPass:  false,
			description: "Query without log_source should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRequiredFieldsRule(basicRequiredFields)
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected query to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected query to fail: %s", tt.description)
			}
		})
	}
}

func TestRequiredFieldsRule_IntermediateQueryRequirements(t *testing.T) {
	// Test intermediate query requirements
	// Based on intermediate_queries.md patterns which might require more fields
	intermediateRequiredFields := []string{"log_source", "timeframe"}
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "correlation_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "24_hours_ago",
				Verb:      *types.NewStringOrArray([]string{"create", "update", "patch"}),
				Resource:  *types.NewStringOrArray([]string{"roles", "rolebindings"}),
			},
			shouldPass:  true,
			description: "Correlation query with required fields should pass",
		},
		{
			name: "pattern_matching_query",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				Timeframe:    "1_hour_ago",
				UserPattern:  "^(?!system:).*",
				GroupBy:      *types.NewStringOrArray([]string{"user", "namespace"}),
			},
			shouldPass:  true,
			description: "Pattern matching query should pass",
		},
		{
			name: "missing_timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
			},
			shouldPass:  false,
			description: "Intermediate query missing timeframe should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewRequiredFieldsRule(intermediateRequiredFields)
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected query to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected query to fail: %s", tt.description)
			}
		})
	}
}

func TestRequiredFieldsRule_EdgeCases(t *testing.T) {
	rule := NewRequiredFieldsRule([]string{"log_source"})

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectedValid bool
		description string
	}{
		{
			name:          "nil_query",
			query:         nil,
			expectedValid: false,
			description:   "Nil query should fail gracefully",
		},
		{
			name:          "empty_query",
			query:         &types.StructuredQuery{},
			expectedValid: false,
			description:   "Empty query should fail when log_source is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Validation panicked: %v", r)
				}
			}()

			result := rule.Validate(tt.query)

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid=%v, got IsValid=%v: %s", 
					tt.expectedValid, result.IsValid, tt.description)
			}
		})
	}
}

func TestRequiredFieldsRule_PerformanceBenchmark(t *testing.T) {
	// Test performance with many required fields
	manyRequiredFields := []string{
		"log_source", "verb", "resource", "namespace", "user", "timeframe",
		"response_status", "source_ip", "group_by", "sort_by", "sort_order",
		"subresource", "auth_decision", "resource_name_pattern", "user_pattern",
		"namespace_pattern", "request_uri_pattern", "authorization_reason_pattern",
		"response_message_pattern", "missing_annotation", "request_object_filter",
	}
	rule := NewRequiredFieldsRule(manyRequiredFields)

	query := &types.StructuredQuery{
		LogSource:                   "kube-apiserver",
		Verb:                        *types.NewStringOrArray("get"),
		Resource:                    *types.NewStringOrArray("pods"),
		Namespace:                   *types.NewStringOrArray("default"),
		User:                        *types.NewStringOrArray("test@company.com"),
		Timeframe:                   "24_hours_ago",
		ResponseStatus:              *types.NewStringOrArray("200"),
		SourceIP:                    *types.NewStringOrArray("192.168.1.1"),
		GroupBy:                     *types.NewStringOrArray("user"),
		SortBy:                      "timestamp",
		SortOrder:                   "desc",
		Subresource:                 "exec",
		AuthDecision:                "allow",
		ResourceNamePattern:         "test-.*",
		UserPattern:                 ".*@company.com",
		NamespacePattern:            "test-.*",
		RequestURIPattern:           "/api/v1/.*",
		AuthorizationReasonPattern:  "allowed",
		ResponseMessagePattern:      "success",
		MissingAnnotation:           "test.annotation",
		RequestObjectFilter:         "filter",
	}

	// Benchmark the validation
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rule.Validate(query)
		}
	})

	// Check that validation completes within reasonable time (< 100μs per operation)
	avgTimePerOp := result.T / time.Duration(result.N)
	if avgTimePerOp > 100*time.Microsecond {
		t.Errorf("Validation too slow: %v per operation (should be < 100μs)", avgTimePerOp)
	}

	t.Logf("Performance: %v per validation operation", avgTimePerOp)
}