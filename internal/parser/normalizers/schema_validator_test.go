package normalizers

import (
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewSchemaValidator(t *testing.T) {
	validator := NewSchemaValidator()
	if validator == nil {
		t.Fatal("Expected non-nil schema validator")
	}
}

func TestSchemaValidator_ValidateSchema_NilQuery(t *testing.T) {
	validator := NewSchemaValidator()
	err := validator.ValidateSchema(nil)
	
	if err == nil {
		t.Error("Expected error for nil query")
	}
	
	if err.Error() != "schema: query is nil" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestSchemaValidator_LogSourceValidation(t *testing.T) {
	validator := NewSchemaValidator()
	
	tests := []struct {
		name        string
		logSource   string
		expectError bool
		description string
	}{
		{
			name:        "valid_log_source",
			logSource:   "kube-apiserver",
			expectError: false,
			description: "Valid log source should pass",
		},
		{
			name:        "another_valid_log_source",
			logSource:   "oauth-server",
			expectError: false,
			description: "Another valid log source should pass",
		},
		{
			name:        "empty_log_source",
			logSource:   "",
			expectError: true,
			description: "Empty log source should fail",
		},
		{
			name:        "whitespace_only_log_source",
			logSource:   "   ",
			expectError: true,
			description: "Whitespace-only log source should fail",
		},
		{
			name:        "log_source_with_whitespace",
			logSource:   "  kube-apiserver  ",
			expectError: false,
			description: "Log source with whitespace should pass (gets trimmed)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: tt.logSource,
				Limit:     20, // Valid limit
			}
			
			err := validator.ValidateSchema(query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if tt.expectError && err != nil {
				if !contains(err.Error(), "log_source is required") {
					t.Errorf("Expected log_source error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestSchemaValidator_LimitValidation(t *testing.T) {
	validator := NewSchemaValidator()
	
	tests := []struct {
		name        string
		limit       int
		expectError bool
		description string
	}{
		{
			name:        "valid_limit_zero",
			limit:       0,
			expectError: false,
			description: "Limit of 0 should be valid",
		},
		{
			name:        "valid_limit_positive",
			limit:       20,
			expectError: false,
			description: "Positive limit should be valid",
		},
		{
			name:        "valid_limit_max",
			limit:       1000,
			expectError: false,
			description: "Maximum limit should be valid",
		},
		{
			name:        "invalid_limit_negative",
			limit:       -1,
			expectError: true,
			description: "Negative limit should be invalid",
		},
		{
			name:        "invalid_limit_too_high",
			limit:       1001,
			expectError: true,
			description: "Limit above 1000 should be invalid",
		},
		{
			name:        "invalid_limit_extremely_high",
			limit:       999999,
			expectError: true,
			description: "Extremely high limit should be invalid",
		},
		{
			name:        "invalid_limit_very_negative",
			limit:       -100,
			expectError: true,
			description: "Very negative limit should be invalid",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver", // Valid log source
				Limit:     tt.limit,
			}
			
			err := validator.ValidateSchema(query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if tt.expectError && err != nil {
				if !contains(err.Error(), "limit out of range") {
					t.Errorf("Expected limit error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestSchemaValidator_TimeframeValidation(t *testing.T) {
	validator := NewSchemaValidator()
	
	tests := []struct {
		name        string
		timeframe   string
		expectError bool
		description string
	}{
		{
			name:        "empty_timeframe",
			timeframe:   "",
			expectError: false,
			description: "Empty timeframe should be valid",
		},
		{
			name:        "today_timeframe",
			timeframe:   "today",
			expectError: false,
			description: "Today timeframe should be valid",
		},
		{
			name:        "yesterday_timeframe",
			timeframe:   "yesterday",
			expectError: false,
			description: "Yesterday timeframe should be valid",
		},
		{
			name:        "1_hour_ago_timeframe",
			timeframe:   "1_hour_ago",
			expectError: false,
			description: "1_hour_ago timeframe should be valid",
		},
		{
			name:        "case_insensitive_today",
			timeframe:   "TODAY",
			expectError: false,
			description: "Case insensitive TODAY should be valid",
		},
		{
			name:        "case_insensitive_yesterday",
			timeframe:   "YESTERDAY",
			expectError: false,
			description: "Case insensitive YESTERDAY should be valid",
		},
		{
			name:        "whitespace_around_today",
			timeframe:   "  today  ",
			expectError: false,
			description: "Timeframe with whitespace should be valid after trimming",
		},
		{
			name:        "unsupported_timeframe",
			timeframe:   "7_days_ago",
			expectError: true,
			description: "Unsupported timeframe should be invalid",
		},
		{
			name:        "invalid_timeframe",
			timeframe:   "invalid",
			expectError: true,
			description: "Invalid timeframe should be invalid",
		},
		{
			name:        "recent_timeframe",
			timeframe:   "recent",
			expectError: true,
			description: "Recent timeframe should be invalid (not in allowed list)",
		},
		{
			name:        "numeric_timeframe",
			timeframe:   "1h",
			expectError: true,
			description: "Numeric timeframe should be invalid (not normalized)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver", // Valid log source
				Limit:     20,               // Valid limit
				Timeframe: tt.timeframe,
			}
			
			err := validator.ValidateSchema(query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if tt.expectError && err != nil {
				if !contains(err.Error(), "unsupported timeframe") {
					t.Errorf("Expected timeframe error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestSchemaValidator_TimeRangeValidation(t *testing.T) {
	validator := NewSchemaValidator()
	
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	oneHourLater := now.Add(time.Hour)
	
	tests := []struct {
		name        string
		timeRange   *types.TimeRange
		expectError bool
		description string
	}{
		{
			name:        "nil_time_range",
			timeRange:   nil,
			expectError: false,
			description: "Nil time range should be valid",
		},
		{
			name: "valid_time_range",
			timeRange: &types.TimeRange{
				Start: oneHourAgo,
				End:   now,
			},
			expectError: false,
			description: "Valid time range (start before end) should be valid",
		},
		{
			name: "identical_times",
			timeRange: &types.TimeRange{
				Start: now,
				End:   now,
			},
			expectError: false,
			description: "Identical start and end times should be valid",
		},
		{
			name: "reversed_time_range",
			timeRange: &types.TimeRange{
				Start: now,
				End:   oneHourAgo,
			},
			expectError: true,
			description: "Reversed time range (end before start) should be invalid",
		},
		{
			name: "far_future_range",
			timeRange: &types.TimeRange{
				Start: now,
				End:   oneHourLater,
			},
			expectError: false,
			description: "Future time range should be valid",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver", // Valid log source
				Limit:     20,               // Valid limit
				Timeframe: "today",          // Valid timeframe
				TimeRange: tt.timeRange,
			}
			
			err := validator.ValidateSchema(query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if tt.expectError && err != nil {
				if !contains(err.Error(), "time_range.end before time_range.start") {
					t.Errorf("Expected time range error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestSchemaValidator_ComprehensiveValidation(t *testing.T) {
	validator := NewSchemaValidator()
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectError bool
		errorSubstr string
		description string
	}{
		{
			name: "completely_valid_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     50,
				Timeframe: "today",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			expectError: false,
			description: "Completely valid query should pass",
		},
		{
			name: "minimal_valid_query",
			query: &types.StructuredQuery{
				LogSource: "oauth-server",
				Limit:     0,
			},
			expectError: false,
			description: "Minimal valid query should pass",
		},
		{
			name: "query_with_invalid_log_source",
			query: &types.StructuredQuery{
				LogSource: "",
				Limit:     20,
				Timeframe: "today",
			},
			expectError: true,
			errorSubstr: "log_source is required",
			description: "Query with invalid log source should fail",
		},
		{
			name: "query_with_invalid_limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     -5,
				Timeframe: "today",
			},
			expectError: true,
			errorSubstr: "limit out of range",
			description: "Query with invalid limit should fail",
		},
		{
			name: "query_with_invalid_timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     20,
				Timeframe: "invalid_timeframe",
			},
			expectError: true,
			errorSubstr: "unsupported timeframe",
			description: "Query with invalid timeframe should fail",
		},
		{
			name: "query_with_invalid_time_range",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     20,
				Timeframe: "today",
				TimeRange: &types.TimeRange{
					Start: time.Now(),
					End:   time.Now().Add(-time.Hour),
				},
			},
			expectError: true,
			errorSubstr: "time_range.end before time_range.start",
			description: "Query with invalid time range should fail",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSchema(tt.query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if tt.expectError && err != nil && tt.errorSubstr != "" {
				if !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorSubstr, err.Error())
				}
			}
		})
	}
}

func TestSchemaValidator_FieldsNotValidated(t *testing.T) {
	validator := NewSchemaValidator()
	
	// Test that the validator doesn't validate fields that are not in its scope
	query := &types.StructuredQuery{
		LogSource:                  "kube-apiserver",
		Limit:                      20,
		Timeframe:                  "today",
		Verb:                       *types.NewStringOrArray("invalid_verb"),
		Resource:                   *types.NewStringOrArray("invalid_resource"),
		Namespace:                  *types.NewStringOrArray("invalid_namespace"),
		User:                       *types.NewStringOrArray("invalid_user"),
		ResponseStatus:             *types.NewStringOrArray("invalid_status"),
		ExcludeUsers:               []string{"invalid_pattern"},
		ResourceNamePattern:        "invalid_regex_[",
		UserPattern:                "invalid_regex_[",
		NamespacePattern:           "invalid_regex_[",
		RequestURIPattern:          "invalid_regex_[",
		AuthDecision:               "invalid_decision",
		SourceIP:                   *types.NewStringOrArray("invalid_ip"),
		GroupBy:                    *types.NewStringOrArray("invalid_group"),
		SortBy:                     "invalid_sort",
		SortOrder:                  "invalid_order",
		Subresource:                "invalid_subresource",
		IncludeChanges:             true,
		RequestObjectFilter:        "invalid_filter",
		ExcludeResources:           []string{"invalid_resource"},
		AuthorizationReasonPattern: "invalid_pattern",
		ResponseMessagePattern:     "invalid_pattern",
		MissingAnnotation:          "invalid_annotation",
	}
	
	err := validator.ValidateSchema(query)
	if err != nil {
		t.Errorf("Schema validator should not validate fields outside its scope, got error: %v", err)
	}
}

func TestSchemaValidator_EdgeCases(t *testing.T) {
	validator := NewSchemaValidator()
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectError bool
		description string
	}{
		{
			name: "query_with_zero_values",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     0,
				Timeframe: "",
			},
			expectError: false,
			description: "Query with zero values should be valid",
		},
		{
			name: "query_with_boundary_limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     1000,
				Timeframe: "",
			},
			expectError: false,
			description: "Query with boundary limit should be valid",
		},
		{
			name: "query_with_complex_fields",
			query: &types.StructuredQuery{
				LogSource: "openshift-apiserver",
				Limit:     500,
				Timeframe: "yesterday",
				TimeRange: &types.TimeRange{
					Start: time.Now().Add(-2 * time.Hour),
					End:   time.Now().Add(-time.Hour),
				},
			},
			expectError: false,
			description: "Query with complex valid fields should pass",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSchema(tt.query)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
		})
	}
}

func TestSchemaValidator_MultipleErrors(t *testing.T) {
	validator := NewSchemaValidator()
	
	// Test that validation stops at the first error (based on implementation)
	query := &types.StructuredQuery{
		LogSource: "",    // Invalid: empty log source
		Limit:     -1,    // Invalid: negative limit
		Timeframe: "bad", // Invalid: unsupported timeframe
	}
	
	err := validator.ValidateSchema(query)
	if err == nil {
		t.Error("Expected error for query with multiple validation issues")
	}
	
	// Should fail on the first error (log_source)
	if !contains(err.Error(), "log_source is required") {
		t.Errorf("Expected log_source error first, got: %s", err.Error())
	}
}

// Benchmark performance of schema validation
func BenchmarkSchemaValidator_ValidateSchema(b *testing.B) {
	validator := NewSchemaValidator()
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Limit:     50,
		Timeframe: "today",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
		Namespace: *types.NewStringOrArray("default"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateSchema(query)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func TestSchemaValidator_PerformanceTarget(t *testing.T) {
	validator := NewSchemaValidator()
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Limit:     50,
		Timeframe: "today",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
	}
	
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		err := validator.ValidateSchema(query)
		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	// Target: < 100µs per operation (very fast for validation)
	target := 100 * time.Microsecond
	if avgDuration > target {
		t.Errorf("Performance target missed: average %v > target %v", avgDuration, target)
	}
	
	t.Logf("Performance: %v per validation operation (%d iterations)", avgDuration, iterations)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && 
		(s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}