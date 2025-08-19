package recovery

import (
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewFallbackHandler(t *testing.T) {
	handler := NewFallbackHandler()
	if handler == nil {
		t.Fatal("Expected non-nil fallback handler")
	}
}

func TestFallbackHandler_CreateMinimalQuery_NilRawResponse(t *testing.T) {
	handler := NewFallbackHandler()
	
	result, err := handler.CreateMinimalQuery(nil, "any_model", "show me all pods")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	// Verify defaults
	if result.LogSource != "kube-apiserver" {
		t.Errorf("Expected default log_source 'kube-apiserver', got '%s'", result.LogSource)
	}
	
	if result.Limit != 20 {
		t.Errorf("Expected default limit 20, got %d", result.Limit)
	}
}

func TestFallbackHandler_LogSourceHeuristics(t *testing.T) {
	handler := NewFallbackHandler()
	
	tests := []struct {
		name               string
		rawContent         string
		originalQuery      string
		expectedLogSource  string
		description        string
	}{
		{
			name:               "default_kube_apiserver",
			rawContent:         "some generic content",
			originalQuery:      "show me pods",
			expectedLogSource:  "kube-apiserver",
			description:        "Default should be kube-apiserver",
		},
		{
			name:               "oauth_in_raw_content",
			rawContent:         "failed oauth authentication",
			originalQuery:      "show me failures",
			expectedLogSource:  "oauth-server",
			description:        "OAuth in raw content should trigger oauth-server",
		},
		{
			name:               "oauth_in_original_query",
			rawContent:         "",
			originalQuery:      "show me oauth login failures",
			expectedLogSource:  "oauth-server",
			description:        "OAuth in original query should trigger oauth-server",
		},
		{
			name:               "openshift_in_raw_content",
			rawContent:         "openshift resource creation",
			originalQuery:      "show me errors",
			expectedLogSource:  "openshift-apiserver",
			description:        "OpenShift in raw content should trigger openshift-apiserver",
		},
		{
			name:               "openshift_in_original_query",
			rawContent:         "",
			originalQuery:      "show me openshift specific resources",
			expectedLogSource:  "openshift-apiserver",
			description:        "OpenShift in original query should trigger openshift-apiserver",
		},
		{
			name:               "oauth_case_insensitive",
			rawContent:         "OAUTH failure occurred",
			originalQuery:      "",
			expectedLogSource:  "oauth-server",
			description:        "OAuth should be case insensitive",
		},
		{
			name:               "openshift_case_insensitive",
			rawContent:         "",
			originalQuery:      "Show me OPENSHIFT logs",
			expectedLogSource:  "openshift-apiserver",
			description:        "OpenShift should be case insensitive",
		},
		{
			name:               "oauth_partial_word",
			rawContent:         "The OAuth2 server returned an error",
			originalQuery:      "",
			expectedLogSource:  "oauth-server",
			description:        "OAuth as part of larger word should match",
		},
		{
			name:               "openshift_partial_word",
			rawContent:         "",
			originalQuery:      "Check OpenShift-specific resources",
			expectedLogSource:  "openshift-apiserver",
			description:        "OpenShift as part of larger phrase should match",
		},
		{
			name:               "both_oauth_and_openshift_oauth_wins",
			rawContent:         "OAuth error in OpenShift",
			originalQuery:      "",
			expectedLogSource:  "oauth-server",
			description:        "When both present, oauth should win (first in switch)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &types.RawResponse{Content: tt.rawContent}
			
			result, err := handler.CreateMinimalQuery(raw, "any_model", tt.originalQuery)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.LogSource != tt.expectedLogSource {
				t.Errorf("Expected log_source '%s', got '%s' for %s", 
					tt.expectedLogSource, result.LogSource, tt.description)
			}
		})
	}
}

func TestFallbackHandler_TimeframeHeuristics(t *testing.T) {
	handler := NewFallbackHandler()
	
	tests := []struct {
		name              string
		rawContent        string
		originalQuery     string
		expectedTimeframe string
		description       string
	}{
		{
			name:              "no_timeframe_hints",
			rawContent:        "some generic content",
			originalQuery:     "show me pods",
			expectedTimeframe: "",
			description:       "No timeframe keywords should result in empty timeframe",
		},
		{
			name:              "today_in_raw_content",
			rawContent:        "events that happened today",
			originalQuery:     "show me failures",
			expectedTimeframe: "today",
			description:       "Today in raw content should set timeframe",
		},
		{
			name:              "today_in_original_query",
			rawContent:        "",
			originalQuery:     "show me today's authentication failures",
			expectedTimeframe: "today",
			description:       "Today in original query should set timeframe",
		},
		{
			name:              "yesterday_in_raw_content",
			rawContent:        "yesterday's error logs",
			originalQuery:     "",
			expectedTimeframe: "yesterday",
			description:       "Yesterday in raw content should set timeframe",
		},
		{
			name:              "yesterday_in_original_query",
			rawContent:        "",
			originalQuery:     "what happened yesterday?",
			expectedTimeframe: "yesterday",
			description:       "Yesterday in original query should set timeframe",
		},
		{
			name:              "hour_in_raw_content",
			rawContent:        "in the last hour",
			originalQuery:     "",
			expectedTimeframe: "1_hour_ago",
			description:       "Hour in raw content should set timeframe",
		},
		{
			name:              "hour_in_original_query",
			rawContent:        "",
			originalQuery:     "show me failures from the past hour",
			expectedTimeframe: "1_hour_ago",
			description:       "Hour in original query should set timeframe",
		},
		{
			name:              "today_case_insensitive",
			rawContent:        "TODAY's events",
			originalQuery:     "",
			expectedTimeframe: "today",
			description:       "Today should be case insensitive",
		},
		{
			name:              "yesterday_case_insensitive",
			rawContent:        "",
			originalQuery:     "What happened YESTERDAY?",
			expectedTimeframe: "yesterday",
			description:       "Yesterday should be case insensitive",
		},
		{
			name:              "hour_case_insensitive",
			rawContent:        "",
			originalQuery:     "In the last HOUR",
			expectedTimeframe: "1_hour_ago",
			description:       "Hour should be case insensitive",
		},
		{
			name:              "hour_partial_word",
			rawContent:        "",
			originalQuery:     "show me hourly statistics",
			expectedTimeframe: "1_hour_ago",
			description:       "Hour as part of larger word should match",
		},
		{
			name:              "multiple_timeframes_today_wins",
			rawContent:        "today and yesterday",
			originalQuery:     "",
			expectedTimeframe: "today",
			description:       "When multiple timeframes present, today should win (first in if-else)",
		},
		{
			name:              "yesterday_and_hour_yesterday_wins",
			rawContent:        "yesterday hour",
			originalQuery:     "",
			expectedTimeframe: "yesterday",
			description:       "When yesterday and hour present, yesterday should win",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &types.RawResponse{Content: tt.rawContent}
			
			result, err := handler.CreateMinimalQuery(raw, "any_model", tt.originalQuery)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.Timeframe != tt.expectedTimeframe {
				t.Errorf("Expected timeframe '%s', got '%s' for %s", 
					tt.expectedTimeframe, result.Timeframe, tt.description)
			}
		})
	}
}

func TestFallbackHandler_ComprehensiveMapping(t *testing.T) {
	handler := NewFallbackHandler()
	
	tests := []struct {
		name               string
		rawContent         string
		originalQuery      string
		expectedLogSource  string
		expectedTimeframe  string
		description        string
	}{
		{
			name:               "oauth_with_today",
			rawContent:         "OAuth login failed today",
			originalQuery:      "",
			expectedLogSource:  "oauth-server",
			expectedTimeframe:  "today",
			description:        "Should detect both OAuth and today",
		},
		{
			name:               "openshift_with_yesterday",
			rawContent:         "",
			originalQuery:      "Show OpenShift resources from yesterday",
			expectedLogSource:  "openshift-apiserver",
			expectedTimeframe:  "yesterday",
			description:        "Should detect both OpenShift and yesterday",
		},
		{
			name:               "kube_with_hour",
			rawContent:         "Kubernetes pod failures in the last hour",
			originalQuery:      "",
			expectedLogSource:  "kube-apiserver",
			expectedTimeframe:  "1_hour_ago",
			description:        "Should default to kube-apiserver and detect hour",
		},
		{
			name:               "complex_mixed_content",
			rawContent:         "OAuth authentication failed",
			originalQuery:      "show me today's OpenShift errors",
			expectedLogSource:  "oauth-server",
			expectedTimeframe:  "today",
			description:        "Should handle complex mixed content from both sources",
		},
		{
			name:               "no_hints_defaults",
			rawContent:         "",
			originalQuery:      "show me logs",
			expectedLogSource:  "kube-apiserver",
			expectedTimeframe:  "",
			description:        "Should use defaults when no hints present",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &types.RawResponse{Content: tt.rawContent}
			
			result, err := handler.CreateMinimalQuery(raw, "any_model", tt.originalQuery)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.LogSource != tt.expectedLogSource {
				t.Errorf("Expected log_source '%s', got '%s' for %s", 
					tt.expectedLogSource, result.LogSource, tt.description)
			}
			
			if result.Timeframe != tt.expectedTimeframe {
				t.Errorf("Expected timeframe '%s', got '%s' for %s", 
					tt.expectedTimeframe, result.Timeframe, tt.description)
			}
		})
	}
}

func TestFallbackHandler_PreservesOtherDefaults(t *testing.T) {
	handler := NewFallbackHandler()
	
	raw := &types.RawResponse{Content: "OAuth errors today"}
	result, err := handler.CreateMinimalQuery(raw, "any_model", "show me failures")
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify that only LogSource, Limit, and Timeframe are set
	// All other fields should be zero values
	if result.Verb.GetString() != "" && len(result.Verb.GetArray()) != 0 {
		t.Errorf("Expected empty Verb, got '%v'", result.Verb)
	}
	
	if result.Resource.GetString() != "" && len(result.Resource.GetArray()) != 0 {
		t.Errorf("Expected empty Resource, got '%v'", result.Resource)
	}
	
	if result.Namespace.GetString() != "" && len(result.Namespace.GetArray()) != 0 {
		t.Errorf("Expected empty Namespace, got '%v'", result.Namespace)
	}
	
	if result.User.GetString() != "" && len(result.User.GetArray()) != 0 {
		t.Errorf("Expected empty User, got '%v'", result.User)
	}
	
	if result.ResponseStatus.GetString() != "" && len(result.ResponseStatus.GetArray()) != 0 {
		t.Errorf("Expected empty ResponseStatus, got '%v'", result.ResponseStatus)
	}
	
	if len(result.ExcludeUsers) != 0 {
		t.Errorf("Expected empty ExcludeUsers, got %v", result.ExcludeUsers)
	}
	
	if result.ResourceNamePattern != "" {
		t.Errorf("Expected empty ResourceNamePattern, got '%s'", result.ResourceNamePattern)
	}
	
	if result.UserPattern != "" {
		t.Errorf("Expected empty UserPattern, got '%s'", result.UserPattern)
	}
	
	if result.NamespacePattern != "" {
		t.Errorf("Expected empty NamespacePattern, got '%s'", result.NamespacePattern)
	}
	
	if result.RequestURIPattern != "" {
		t.Errorf("Expected empty RequestURIPattern, got '%s'", result.RequestURIPattern)
	}
	
	if result.AuthDecision != "" {
		t.Errorf("Expected empty AuthDecision, got '%s'", result.AuthDecision)
	}
	
	if result.SourceIP.GetString() != "" && len(result.SourceIP.GetArray()) != 0 {
		t.Errorf("Expected empty SourceIP, got '%v'", result.SourceIP)
	}
	
	if result.GroupBy.GetString() != "" && len(result.GroupBy.GetArray()) != 0 {
		t.Errorf("Expected empty GroupBy, got '%v'", result.GroupBy)
	}
	
	if result.SortBy != "" {
		t.Errorf("Expected empty SortBy, got '%s'", result.SortBy)
	}
	
	if result.SortOrder != "" {
		t.Errorf("Expected empty SortOrder, got '%s'", result.SortOrder)
	}
	
	if result.Subresource != "" {
		t.Errorf("Expected empty Subresource, got '%s'", result.Subresource)
	}
	
	if result.IncludeChanges != false {
		t.Errorf("Expected IncludeChanges false, got %v", result.IncludeChanges)
	}
	
	if result.TimeRange != nil {
		t.Errorf("Expected nil TimeRange, got %v", result.TimeRange)
	}
	
	if result.BusinessHours != nil {
		t.Errorf("Expected nil BusinessHours, got %v", result.BusinessHours)
	}
	
	if result.Analysis != nil {
		t.Errorf("Expected nil Analysis, got %v", result.Analysis)
	}
}

func TestFallbackHandler_EdgeCases(t *testing.T) {
	handler := NewFallbackHandler()
	
	tests := []struct {
		name          string
		rawResponse   *types.RawResponse
		modelType     string
		originalQuery string
		description   string
	}{
		{
			name:          "nil_raw_response",
			rawResponse:   nil,
			modelType:     "gpt-4",
			originalQuery: "show me logs",
			description:   "Nil raw response should work",
		},
		{
			name:          "empty_raw_content",
			rawResponse:   &types.RawResponse{Content: ""},
			modelType:     "claude",
			originalQuery: "oauth failures today",
			description:   "Empty raw content should work",
		},
		{
			name:          "empty_original_query",
			rawResponse:   &types.RawResponse{Content: "OAuth errors today"},
			modelType:     "llama",
			originalQuery: "",
			description:   "Empty original query should work",
		},
		{
			name:          "both_empty",
			rawResponse:   &types.RawResponse{Content: ""},
			modelType:     "mistral",
			originalQuery: "",
			description:   "Both empty should work with defaults",
		},
		{
			name:          "whitespace_only_content",
			rawResponse:   &types.RawResponse{Content: "   "},
			modelType:     "gpt-3.5",
			originalQuery: "   ",
			description:   "Whitespace-only content should work",
		},
		{
			name:          "very_long_content",
			rawResponse:   &types.RawResponse{Content: "This is a very long piece of content that contains oauth authentication failures and mentions today's events in OpenShift clusters with lots of additional text to test handling of large inputs."},
			modelType:     "claude-2",
			originalQuery: "show me authentication logs from yesterday",
			description:   "Very long content should be handled correctly",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.CreateMinimalQuery(tt.rawResponse, tt.modelType, tt.originalQuery)
			
			if err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
			
			if result == nil {
				t.Errorf("Expected non-nil result for %s", tt.description)
			}
			
			// Basic sanity checks
			if result != nil {
				if result.LogSource == "" {
					t.Errorf("LogSource should never be empty for %s", tt.description)
				}
				
				if result.Limit <= 0 {
					t.Errorf("Limit should be positive for %s, got %d", tt.description, result.Limit)
				}
			}
		})
	}
}

func TestFallbackHandler_ModelTypeIgnored(t *testing.T) {
	handler := NewFallbackHandler()
	
	// Test that different model types don't affect the output
	modelTypes := []string{"gpt-4", "claude-3", "llama-2", "mistral", "", "unknown-model"}
	
	rawResponse := &types.RawResponse{Content: "OAuth failure today"}
	originalQuery := "show me errors"
	
	var firstResult *types.StructuredQuery
	
	for i, modelType := range modelTypes {
		result, err := handler.CreateMinimalQuery(rawResponse, modelType, originalQuery)
		if err != nil {
			t.Fatalf("Unexpected error for model type '%s': %v", modelType, err)
		}
		
		if i == 0 {
			firstResult = result
		} else {
			// All results should be identical regardless of model type
			if result.LogSource != firstResult.LogSource {
				t.Errorf("LogSource differs for model '%s': expected '%s', got '%s'", 
					modelType, firstResult.LogSource, result.LogSource)
			}
			if result.Limit != firstResult.Limit {
				t.Errorf("Limit differs for model '%s': expected %d, got %d", 
					modelType, firstResult.Limit, result.Limit)
			}
			if result.Timeframe != firstResult.Timeframe {
				t.Errorf("Timeframe differs for model '%s': expected '%s', got '%s'", 
					modelType, firstResult.Timeframe, result.Timeframe)
			}
		}
	}
}

func TestFallbackHandler_SpecialCharacters(t *testing.T) {
	handler := NewFallbackHandler()
	
	tests := []struct {
		name          string
		content       string
		query         string
		expectedLog   string
		expectedTime  string
	}{
		{
			name:         "content_with_special_chars",
			content:      "OAuth-2.0 failure! Today's @logs #show $errors",
			query:        "",
			expectedLog:  "oauth-server",
			expectedTime: "today",
		},
		{
			name:         "query_with_special_chars",
			content:      "",
			query:        "Show me OpenShift (yesterday) logs: [critical]",
			expectedLog:  "openshift-apiserver",
			expectedTime: "yesterday",
		},
		{
			name:         "unicode_content",
			content:      "OAuth🔑 authentication failed today🗓️",
			query:        "",
			expectedLog:  "oauth-server",
			expectedTime: "today",
		},
		{
			name:         "mixed_case_keywords",
			content:      "oAuTh errors from OpenShift",
			query:        "show me logs from YeStErDaY",
			expectedLog:  "oauth-server",
			expectedTime: "yesterday",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &types.RawResponse{Content: tt.content}
			result, err := handler.CreateMinimalQuery(raw, "any", tt.query)
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result.LogSource != tt.expectedLog {
				t.Errorf("Expected log_source '%s', got '%s'", tt.expectedLog, result.LogSource)
			}
			
			if result.Timeframe != tt.expectedTime {
				t.Errorf("Expected timeframe '%s', got '%s'", tt.expectedTime, result.Timeframe)
			}
		})
	}
}

// Benchmark performance of fallback handler
func BenchmarkFallbackHandler_CreateMinimalQuery(b *testing.B) {
	handler := NewFallbackHandler()
	raw := &types.RawResponse{
		Content: "OAuth authentication failed today in OpenShift cluster",
	}
	originalQuery := "show me authentication failures from yesterday"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.CreateMinimalQuery(raw, "gpt-4", originalQuery)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func TestFallbackHandler_PerformanceTarget(t *testing.T) {
	handler := NewFallbackHandler()
	raw := &types.RawResponse{
		Content: "OAuth authentication failed today in OpenShift cluster",
	}
	originalQuery := "show me authentication failures from yesterday"
	
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		_, err := handler.CreateMinimalQuery(raw, "gpt-4", originalQuery)
		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	// Target: < 100µs per operation (very fast for text processing)
	target := 100 * time.Microsecond
	if avgDuration > target {
		t.Errorf("Performance target missed: average %v > target %v", avgDuration, target)
	}
	
	t.Logf("Performance: %v per fallback operation (%d iterations)", avgDuration, iterations)
}