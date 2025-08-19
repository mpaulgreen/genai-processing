package normalizers

import (
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewJSONNormalizer(t *testing.T) {
	normalizer := NewJSONNormalizer()
	if normalizer == nil {
		t.Fatal("Expected non-nil JSON normalizer")
	}
}

func TestJSONNormalizer_Normalize_NilQuery(t *testing.T) {
	normalizer := NewJSONNormalizer()
	result, err := normalizer.Normalize(nil)
	
	if result != nil {
		t.Error("Expected nil result for nil query")
	}
	
	if err == nil {
		t.Error("Expected error for nil query")
	}
	
	if err.Error() != "normalize: query is nil" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestJSONNormalizer_LogSourceDefaults(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	tests := []struct {
		name         string
		inputSource  string
		expectedSource string
	}{
		{
			name:         "empty_log_source",
			inputSource:  "",
			expectedSource: "kube-apiserver",
		},
		{
			name:         "whitespace_only_log_source",
			inputSource:  "   ",
			expectedSource: "kube-apiserver",
		},
		{
			name:         "valid_log_source_unchanged",
			inputSource:  "oauth-server",
			expectedSource: "oauth-server",
		},
		{
			name:         "log_source_with_whitespace",
			inputSource:  "  openshift-apiserver  ",
			expectedSource: "  openshift-apiserver  ",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: tt.inputSource,
			}
			
			result, err := normalizer.Normalize(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.LogSource != tt.expectedSource {
				t.Errorf("Expected log_source '%s', got '%s'", tt.expectedSource, result.LogSource)
			}
		})
	}
}

func TestJSONNormalizer_LimitNormalization(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{
			name:          "zero_limit_gets_default",
			inputLimit:    0,
			expectedLimit: 20,
		},
		{
			name:          "negative_limit_gets_default",
			inputLimit:    -5,
			expectedLimit: 20,
		},
		{
			name:          "valid_limit_unchanged",
			inputLimit:    50,
			expectedLimit: 50,
		},
		{
			name:          "limit_at_max_unchanged",
			inputLimit:    1000,
			expectedLimit: 1000,
		},
		{
			name:          "limit_over_max_capped",
			inputLimit:    1500,
			expectedLimit: 1000,
		},
		{
			name:          "extremely_high_limit_capped",
			inputLimit:    999999,
			expectedLimit: 1000,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     tt.inputLimit,
			}
			
			result, err := normalizer.Normalize(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.Limit != tt.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tt.expectedLimit, result.Limit)
			}
		})
	}
}

func TestJSONNormalizer_StringOrArrayNormalization(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	tests := []struct {
		name     string
		input    types.StringOrArray
		expected types.StringOrArray
	}{
		{
			name:     "single_string_trimmed",
			input:    *types.NewStringOrArray("  get  "),
			expected: *types.NewStringOrArray("get"),
		},
		{
			name:     "array_with_whitespace_trimmed",
			input:    *types.NewStringOrArray([]string{"  create  ", "delete", "   "}),
			expected: *types.NewStringOrArray([]string{"create", "delete"}),
		},
		{
			name:     "array_with_empty_strings_filtered",
			input:    *types.NewStringOrArray([]string{"get", "", "list", "   "}),
			expected: *types.NewStringOrArray([]string{"get", "list"}),
		},
		{
			name:     "single_empty_string_trimmed",
			input:    *types.NewStringOrArray(""),
			expected: *types.NewStringOrArray(""),
		},
		{
			name:     "array_all_empty_becomes_empty",
			input:    *types.NewStringOrArray([]string{"", "   ", ""}),
			expected: *types.NewStringOrArray([]string{}),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      tt.input,
			}
			
			result, err := normalizer.Normalize(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// Compare results based on type
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

func TestJSONNormalizer_AllStringOrArrayFieldsNormalized(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	query := &types.StructuredQuery{
		LogSource:      "kube-apiserver",
		Verb:           *types.NewStringOrArray("  get  "),
		Resource:       *types.NewStringOrArray([]string{"  pods  ", "", "secrets"}),
		Namespace:      *types.NewStringOrArray("   default   "),
		User:           *types.NewStringOrArray([]string{"user1", "   ", "user2"}),
		ResponseStatus: *types.NewStringOrArray("  200  "),
		SourceIP:       *types.NewStringOrArray([]string{"192.168.1.1", "   10.0.0.1   "}),
		GroupBy:        *types.NewStringOrArray("  user  "),
	}
	
	result, err := normalizer.Normalize(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify all fields are normalized
	if result.Verb.GetString() != "get" {
		t.Errorf("Expected verb 'get', got '%s'", result.Verb.GetString())
	}
	
	resourceArr := result.Resource.GetArray()
	if len(resourceArr) != 2 || resourceArr[0] != "pods" || resourceArr[1] != "secrets" {
		t.Errorf("Expected resource ['pods', 'secrets'], got %v", resourceArr)
	}
	
	if result.Namespace.GetString() != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", result.Namespace.GetString())
	}
	
	userArr := result.User.GetArray()
	if len(userArr) != 2 || userArr[0] != "user1" || userArr[1] != "user2" {
		t.Errorf("Expected user ['user1', 'user2'], got %v", userArr)
	}
	
	if result.ResponseStatus.GetString() != "200" {
		t.Errorf("Expected response_status '200', got '%s'", result.ResponseStatus.GetString())
	}
	
	sourceArr := result.SourceIP.GetArray()
	if len(sourceArr) != 2 || sourceArr[0] != "192.168.1.1" || sourceArr[1] != "10.0.0.1" {
		t.Errorf("Expected source_ip ['192.168.1.1', '10.0.0.1'], got %v", sourceArr)
	}
	
	if result.GroupBy.GetString() != "user" {
		t.Errorf("Expected group_by 'user', got '%s'", result.GroupBy.GetString())
	}
}

func TestJSONNormalizer_TimeframeNormalization(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	tests := []struct {
		name              string
		inputTimeframe    string
		expectedTimeframe string
	}{
		{
			name:              "empty_timeframe_unchanged",
			inputTimeframe:    "",
			expectedTimeframe: "",
		},
		{
			name:              "recent_unchanged",
			inputTimeframe:    "recent",
			expectedTimeframe: "recent",
		},
		{
			name:              "default_unchanged",
			inputTimeframe:    "default",
			expectedTimeframe: "default",
		},
		{
			name:              "1h_normalized",
			inputTimeframe:    "1h",
			expectedTimeframe: "1_hour_ago",
		},
		{
			name:              "1_hour_normalized",
			inputTimeframe:    "1_hour",
			expectedTimeframe: "1_hour_ago",
		},
		{
			name:              "1-hour_normalized",
			inputTimeframe:    "1-hour",
			expectedTimeframe: "1_hour_ago",
		},
		{
			name:              "hour_normalized",
			inputTimeframe:    "hour",
			expectedTimeframe: "1_hour_ago",
		},
		{
			name:              "last_hour_normalized",
			inputTimeframe:    "last_hour",
			expectedTimeframe: "1_hour_ago",
		},
		{
			name:              "today_normalized",
			inputTimeframe:    "today",
			expectedTimeframe: "today",
		},
		{
			name:              "current_day_normalized",
			inputTimeframe:    "current_day",
			expectedTimeframe: "today",
		},
		{
			name:              "yesterday_normalized",
			inputTimeframe:    "yesterday",
			expectedTimeframe: "yesterday",
		},
		{
			name:              "prev_day_normalized",
			inputTimeframe:    "prev_day",
			expectedTimeframe: "yesterday",
		},
		{
			name:              "case_insensitive_today",
			inputTimeframe:    "TODAY",
			expectedTimeframe: "today",
		},
		{
			name:              "whitespace_trimmed",
			inputTimeframe:    "  yesterday  ",
			expectedTimeframe: "yesterday",
		},
		{
			name:              "unknown_timeframe_unchanged",
			inputTimeframe:    "7_days_ago",
			expectedTimeframe: "7_days_ago",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.inputTimeframe,
			}
			
			result, err := normalizer.Normalize(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.Timeframe != tt.expectedTimeframe {
				t.Errorf("Expected timeframe '%s', got '%s'", tt.expectedTimeframe, result.Timeframe)
			}
		})
	}
}

func TestJSONNormalizer_TimeRangeNormalization(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	
	tests := []struct {
		name            string
		inputTimeRange  *types.TimeRange
		expectSwapped   bool
		expectExtended  bool
		description     string
	}{
		{
			name:           "nil_time_range_unchanged",
			inputTimeRange: nil,
			description:    "Nil time range should remain nil",
		},
		{
			name: "valid_time_range_unchanged",
			inputTimeRange: &types.TimeRange{
				Start: oneHourAgo,
				End:   now,
			},
			description: "Valid time range should remain unchanged",
		},
		{
			name: "reversed_time_range_swapped",
			inputTimeRange: &types.TimeRange{
				Start: now,
				End:   oneHourAgo,
			},
			expectSwapped: true,
			description:   "Reversed time range should be swapped",
		},
		{
			name: "identical_times_extended",
			inputTimeRange: &types.TimeRange{
				Start: now,
				End:   now,
			},
			expectExtended: true,
			description:    "Identical start/end times should be extended by 1 hour",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				TimeRange: tt.inputTimeRange,
			}
			
			result, err := normalizer.Normalize(query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if tt.inputTimeRange == nil {
				if result.TimeRange != nil {
					t.Error("Expected nil time range to remain nil")
				}
				return
			}
			
			if result.TimeRange == nil {
				t.Fatal("Expected non-nil time range in result")
			}
			
			// Check if times were swapped
			if tt.expectSwapped {
				// When swapped, the earlier time becomes start, later time becomes end
				// Input was: Start=now(later), End=oneHourAgo(earlier)
				// After swap: Start=oneHourAgo(earlier), End=now(later)
				if !result.TimeRange.Start.Before(result.TimeRange.End) {
					t.Errorf("After swap, start should be before end: start=%v, end=%v", result.TimeRange.Start, result.TimeRange.End)
				}
				// Check that the times are actually from the input (just verify they're one hour apart)
				diff := result.TimeRange.End.Sub(result.TimeRange.Start)
				if diff < 59*time.Minute || diff > 61*time.Minute {
					t.Errorf("Expected times to be ~1 hour apart after swap, got %v", diff)
				}
			}
			
			// Check if identical times were extended
			if tt.expectExtended {
				expectedEnd := tt.inputTimeRange.Start.Add(time.Hour)
				if !result.TimeRange.Start.Equal(tt.inputTimeRange.Start) {
					t.Errorf("Expected start time to remain %v, got %v", tt.inputTimeRange.Start, result.TimeRange.Start)
				}
				if !result.TimeRange.End.Equal(expectedEnd) {
					t.Errorf("Expected end time to be extended to %v, got %v", expectedEnd, result.TimeRange.End)
				}
			}
			
			// Ensure end is always after start
			if result.TimeRange.End.Before(result.TimeRange.Start) {
				t.Error("End time should not be before start time after normalization")
			}
		})
	}
}

func TestJSONNormalizer_ComprehensiveNormalization(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	
	query := &types.StructuredQuery{
		LogSource:      "",  // Should get default
		Verb:           *types.NewStringOrArray([]string{"  GET  ", "", "  list  "}),
		Resource:       *types.NewStringOrArray("  pods  "),
		Namespace:      *types.NewStringOrArray([]string{"", "  default  ", "   "}),
		User:           *types.NewStringOrArray("  admin@company.com  "),
		Timeframe:      "  1h  ",  // Should be normalized
		Limit:          0,         // Should get default
		ResponseStatus: *types.NewStringOrArray("  200  "),
		SourceIP:       *types.NewStringOrArray([]string{"192.168.1.1", "   "}),
		GroupBy:        *types.NewStringOrArray("  user  "),
		TimeRange: &types.TimeRange{
			Start: now,        // Swapped order
			End:   oneHourAgo,
		},
	}
	
	result, err := normalizer.Normalize(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify all normalizations
	if result.LogSource != "kube-apiserver" {
		t.Errorf("Expected default log_source 'kube-apiserver', got '%s'", result.LogSource)
	}
	
	verbArr := result.Verb.GetArray()
	if len(verbArr) != 2 || verbArr[0] != "GET" || verbArr[1] != "list" {
		t.Errorf("Expected verb ['GET', 'list'], got %v", verbArr)
	}
	
	if result.Resource.GetString() != "pods" {
		t.Errorf("Expected resource 'pods', got '%s'", result.Resource.GetString())
	}
	
	nsArr := result.Namespace.GetArray()
	if len(nsArr) != 1 || nsArr[0] != "default" {
		t.Errorf("Expected namespace ['default'], got %v", nsArr)
	}
	
	if result.User.GetString() != "admin@company.com" {
		t.Errorf("Expected user 'admin@company.com', got '%s'", result.User.GetString())
	}
	
	if result.Timeframe != "1_hour_ago" {
		t.Errorf("Expected timeframe '1_hour_ago', got '%s'", result.Timeframe)
	}
	
	if result.Limit != 20 {
		t.Errorf("Expected limit 20, got %d", result.Limit)
	}
	
	if result.ResponseStatus.GetString() != "200" {
		t.Errorf("Expected response_status '200', got '%s'", result.ResponseStatus.GetString())
	}
	
	sourceArr := result.SourceIP.GetArray()
	if len(sourceArr) != 1 || sourceArr[0] != "192.168.1.1" {
		t.Errorf("Expected source_ip ['192.168.1.1'], got %v", sourceArr)
	}
	
	if result.GroupBy.GetString() != "user" {
		t.Errorf("Expected group_by 'user', got '%s'", result.GroupBy.GetString())
	}
	
	// Verify time range was swapped
	if !result.TimeRange.Start.Equal(oneHourAgo) {
		t.Errorf("Expected swapped start time %v, got %v", oneHourAgo, result.TimeRange.Start)
	}
	if !result.TimeRange.End.Equal(now) {
		t.Errorf("Expected swapped end time %v, got %v", now, result.TimeRange.End)
	}
}

func TestJSONNormalizer_PreservesOtherFields(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	query := &types.StructuredQuery{
		LogSource:                  "oauth-server",
		Verb:                       *types.NewStringOrArray("get"),
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
		IncludeChanges:             true,
	}
	
	result, err := normalizer.Normalize(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify all non-normalized fields are preserved
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
	if result.IncludeChanges != query.IncludeChanges {
		t.Errorf("Expected include_changes preserved")
	}
}

func TestJSONNormalizer_EdgeCases(t *testing.T) {
	normalizer := NewJSONNormalizer()
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		description string
	}{
		{
			name: "empty_query_with_defaults",
			query: &types.StructuredQuery{},
			description: "Empty query gets sensible defaults",
		},
		{
			name: "query_with_all_whitespace_fields",
			query: &types.StructuredQuery{
				LogSource: "   ",
				Verb:      *types.NewStringOrArray("   "),
				Resource:  *types.NewStringOrArray([]string{"   ", ""}),
				Timeframe: "   ",
			},
			description: "Query with whitespace-only fields",
		},
		{
			name: "query_with_extreme_limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     999999,
			},
			description: "Query with extremely high limit",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.Normalize(tt.query)
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
			
			// Basic sanity checks
			if result.LogSource == "" {
				t.Error("LogSource should never be empty after normalization")
			}
			if result.Limit <= 0 || result.Limit > 1000 {
				t.Errorf("Limit should be between 1-1000, got %d", result.Limit)
			}
		})
	}
}

// Benchmark performance of JSON normalization
func BenchmarkJSONNormalizer_Normalize(b *testing.B) {
	normalizer := NewJSONNormalizer()
	query := &types.StructuredQuery{
		LogSource:      "",
		Verb:           *types.NewStringOrArray([]string{"  get  ", "list", ""}),
		Resource:       *types.NewStringOrArray("  pods  "),
		Namespace:      *types.NewStringOrArray("  default  "),
		User:           *types.NewStringOrArray("admin@company.com"),
		Timeframe:      "1h",
		Limit:          0,
		ResponseStatus: *types.NewStringOrArray("200"),
		SourceIP:       *types.NewStringOrArray([]string{"192.168.1.1", "10.0.0.1"}),
		GroupBy:        *types.NewStringOrArray("user"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := normalizer.Normalize(query)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func TestJSONNormalizer_PerformanceTarget(t *testing.T) {
	normalizer := NewJSONNormalizer()
	query := &types.StructuredQuery{
		LogSource:      "",
		Verb:           *types.NewStringOrArray([]string{"  get  ", "list", ""}),
		Resource:       *types.NewStringOrArray("  pods  "),
		Namespace:      *types.NewStringOrArray("  default  "),
		User:           *types.NewStringOrArray("admin@company.com"),
		Timeframe:      "1h",
		Limit:          0,
		ResponseStatus: *types.NewStringOrArray("200"),
	}
	
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		_, err := normalizer.Normalize(query)
		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	// Target: < 1ms per operation (generous for normalization)
	target := time.Millisecond
	if avgDuration > target {
		t.Errorf("Performance target missed: average %v > target %v", avgDuration, target)
	}
	
	t.Logf("Performance: %v per normalization operation (%d iterations)", avgDuration, iterations)
}