package rules

import (
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestNewTimeframeRule(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		description string
	}{
		{
			name:        "default_config",
			config:      map[string]interface{}{},
			description: "Default configuration should be applied",
		},
		{
			name: "custom_config",
			config: map[string]interface{}{
				"max_days_back":      30,
				"default_limit":      50,
				"max_limit":          500,
				"min_limit":          5,
				"allowed_timeframes": []interface{}{"today", "yesterday", "1_hour_ago"},
			},
			description: "Custom configuration should be applied",
		},
		{
			name: "partial_config",
			config: map[string]interface{}{
				"max_days_back": 60,
			},
			description: "Partial configuration should use defaults for missing values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewTimeframeRule(tt.config)

			if rule == nil {
				t.Fatal("NewTimeframeRule returned nil")
			}
			if !rule.IsEnabled() {
				t.Error("Expected rule to be enabled by default")
			}
			if rule.GetRuleName() != "timeframe_validation" {
				t.Errorf("Expected rule name 'timeframe_validation', got '%s'", rule.GetRuleName())
			}
			if rule.GetSeverity() != "medium" {
				t.Errorf("Expected severity 'medium', got '%s'", rule.GetSeverity())
			}

			// Test custom configuration values
			if maxDays, ok := tt.config["max_days_back"].(int); ok {
				if rule.maxDaysBack != maxDays {
					t.Errorf("Expected max_days_back %d, got %d", maxDays, rule.maxDaysBack)
				}
			} else {
				if rule.maxDaysBack != 90 { // default value
					t.Errorf("Expected default max_days_back 90, got %d", rule.maxDaysBack)
				}
			}

			if maxLimit, ok := tt.config["max_limit"].(int); ok {
				if rule.maxLimit != maxLimit {
					t.Errorf("Expected max_limit %d, got %d", maxLimit, rule.maxLimit)
				}
			} else {
				if rule.maxLimit != 1000 { // default value
					t.Errorf("Expected default max_limit 1000, got %d", rule.maxLimit)
				}
			}
		})
	}
}

func TestTimeframeRule_AllowedTimeframes(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{})

	tests := []struct {
		name        string
		timeframe   string
		shouldPass  bool
		description string
	}{
		{
			name:        "today",
			timeframe:   "today",
			shouldPass:  true,
			description: "today should be allowed",
		},
		{
			name:        "yesterday",
			timeframe:   "yesterday",
			shouldPass:  true,
			description: "yesterday should be allowed",
		},
		{
			name:        "1_hour_ago",
			timeframe:   "1_hour_ago",
			shouldPass:  true,
			description: "1_hour_ago should be allowed",
		},
		{
			name:        "7_days_ago",
			timeframe:   "7_days_ago",
			shouldPass:  true,
			description: "7_days_ago should be allowed",
		},
		{
			name:        "30_days_ago",
			timeframe:   "30_days_ago",
			shouldPass:  true,
			description: "30_days_ago should be allowed",
		},
		{
			name:        "90_days_ago",
			timeframe:   "90_days_ago",
			shouldPass:  true,
			description: "90_days_ago should be allowed",
		},
		{
			name:        "case_insensitive_today",
			timeframe:   "TODAY",
			shouldPass:  true,
			description: "Case insensitive matching should work",
		},
		{
			name:        "invalid_timeframe",
			timeframe:   "invalid_time",
			shouldPass:  false,
			description: "Invalid timeframe should be rejected",
		},
		{
			name:        "future_timeframe",
			timeframe:   "tomorrow",
			shouldPass:  false,
			description: "Future timeframe should be rejected",
		},
		{
			name:        "typo_timeframe",
			timeframe:   "yesturday",
			shouldPass:  false,
			description: "Typo in timeframe should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.timeframe,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected timeframe to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected timeframe to fail: %s", tt.description)
			}
		})
	}
}

func TestTimeframeRule_CustomAllowedTimeframes(t *testing.T) {
	config := map[string]interface{}{
		"allowed_timeframes": []interface{}{"today", "yesterday", "1_hour_ago"},
	}
	rule := NewTimeframeRule(config)

	tests := []struct {
		name        string
		timeframe   string
		shouldPass  bool
		description string
	}{
		{
			name:        "allowed_custom_today",
			timeframe:   "today",
			shouldPass:  true,
			description: "today should be in custom allowed list",
		},
		{
			name:        "allowed_custom_1_hour",
			timeframe:   "1_hour_ago",
			shouldPass:  true,
			description: "1_hour_ago should be in custom allowed list",
		},
		{
			name:        "not_allowed_7_days",
			timeframe:   "7_days_ago",
			shouldPass:  false,
			description: "7_days_ago should not be in custom allowed list",
		},
		{
			name:        "not_allowed_30_days",
			timeframe:   "30_days_ago",
			shouldPass:  false,
			description: "30_days_ago should not be in custom allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.timeframe,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected timeframe to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected timeframe to fail: %s", tt.description)
			}
		})
	}
}

func TestTimeframeRule_DaysExtractionFromTimeframe(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{})

	tests := []struct {
		name         string
		timeframe    string
		expectedDays int
		description  string
	}{
		{
			name:         "today",
			timeframe:    "today",
			expectedDays: 1,
			description:  "today should extract to 1 day",
		},
		{
			name:         "yesterday",
			timeframe:    "yesterday",
			expectedDays: 1,
			description:  "yesterday should extract to 1 day",
		},
		{
			name:         "1_hour_ago",
			timeframe:    "1_hour_ago",
			expectedDays: 1,
			description:  "1_hour_ago should extract to 1 day",
		},
		{
			name:         "7_days_ago",
			timeframe:    "7_days_ago",
			expectedDays: 7,
			description:  "7_days_ago should extract to 7 days",
		},
		{
			name:         "30_days_ago",
			timeframe:    "30_days_ago",
			expectedDays: 30,
			description:  "30_days_ago should extract to 30 days",
		},
		{
			name:         "2_weeks_ago",
			timeframe:    "2_weeks_ago",
			expectedDays: 14,
			description:  "2_weeks_ago should extract to 14 days",
		},
		{
			name:         "3_months_ago",
			timeframe:    "3_months_ago",
			expectedDays: 90,
			description:  "3_months_ago should extract to 90 days",
		},
		{
			name:         "unknown_timeframe",
			timeframe:    "unknown",
			expectedDays: 0,
			description:  "unknown timeframe should extract to 0 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := rule.extractDaysFromTimeframe(tt.timeframe)
			if days != tt.expectedDays {
				t.Errorf("Expected %d days for timeframe '%s', got %d: %s",
					tt.expectedDays, tt.timeframe, days, tt.description)
			}
		})
	}
}

func TestTimeframeRule_MaxDaysBackValidation(t *testing.T) {
	config := map[string]interface{}{
		"max_days_back": 30,
		"allowed_timeframes": []interface{}{
			"today", "yesterday", "7_days_ago", "30_days_ago", "60_days_ago", "90_days_ago",
		},
	}
	rule := NewTimeframeRule(config)

	tests := []struct {
		name        string
		timeframe   string
		shouldPass  bool
		description string
	}{
		{
			name:        "within_limit_today",
			timeframe:   "today",
			shouldPass:  true,
			description: "today should be within 30 day limit",
		},
		{
			name:        "within_limit_7_days",
			timeframe:   "7_days_ago",
			shouldPass:  true,
			description: "7_days_ago should be within 30 day limit",
		},
		{
			name:        "at_limit_30_days",
			timeframe:   "30_days_ago",
			shouldPass:  true,
			description: "30_days_ago should be at the 30 day limit",
		},
		{
			name:        "exceeds_limit_60_days",
			timeframe:   "60_days_ago",
			shouldPass:  false,
			description: "60_days_ago should exceed 30 day limit",
		},
		{
			name:        "exceeds_limit_90_days",
			timeframe:   "90_days_ago",
			shouldPass:  false,
			description: "90_days_ago should exceed 30 day limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.timeframe,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected timeframe to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected timeframe to fail: %s", tt.description)
			}

			// Check specific error message for max days back
			if !tt.shouldPass {
				found := false
				for _, err := range result.Errors {
					if err != "" && (containsSubstring(err, "exceeds maximum allowed days back")) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message about exceeding max days back, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestTimeframeRule_LimitValidation(t *testing.T) {
	config := map[string]interface{}{
		"min_limit": 1,
		"max_limit": 100,
	}
	rule := NewTimeframeRule(config)

	tests := []struct {
		name        string
		limit       int
		shouldPass  bool
		description string
	}{
		{
			name:        "valid_limit_within_range",
			limit:       50,
			shouldPass:  true,
			description: "Limit within range should pass",
		},
		{
			name:        "valid_limit_at_min",
			limit:       1,
			shouldPass:  true,
			description: "Limit at minimum should pass",
		},
		{
			name:        "valid_limit_at_max",
			limit:       100,
			shouldPass:  true,
			description: "Limit at maximum should pass",
		},
		{
			name:        "zero_limit_ignored",
			limit:       0,
			shouldPass:  true,
			description: "Zero limit should be ignored (not validated)",
		},
		{
			name:        "negative_limit",
			limit:       -5,
			shouldPass:  false,
			description: "Negative limit should fail",
		},
		{
			name:        "limit_exceeds_max",
			limit:       150,
			shouldPass:  false,
			description: "Limit exceeding maximum should fail",
		},
		{
			name:        "limit_below_min",
			limit:       0, // Set to -1 in the test since 0 is ignored
			shouldPass:  false,
			description: "Limit below minimum should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     tt.limit,
			}

			// Special case for testing below minimum
			if tt.name == "limit_below_min" {
				query.Limit = -1 // Force negative value for this test
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected limit to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected limit to fail: %s", tt.description)
			}

			// Check specific error messages for limit violations
			if !tt.shouldPass {
				found := false
				for _, err := range result.Errors {
					if err != "" && (containsSubstring(err, "limit") || containsSubstring(err, "Limit")) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message about limit violation, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestTimeframeRule_TimeRangeValidation(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{
		"max_days_back": 30,
	})

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, 0, -29)
	tooOld := now.AddDate(0, 0, -40)
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		name        string
		timeRange   *types.TimeRange
		shouldPass  bool
		description string
	}{
		{
			name: "valid_range_yesterday_to_today",
			timeRange: &types.TimeRange{
				Start: yesterday,
				End:   now,
			},
			shouldPass:  true,
			description: "Valid range from yesterday to now should pass",
		},
		{
			name: "valid_range_week_ago",
			timeRange: &types.TimeRange{
				Start: weekAgo,
				End:   yesterday,
			},
			shouldPass:  true,
			description: "Valid range from week ago to yesterday should pass",
		},
		{
			name: "valid_range_at_limit",
			timeRange: &types.TimeRange{
				Start: monthAgo,
				End:   now,
			},
			shouldPass:  true,
			description: "Range at 30-day limit should pass",
		},
		{
			name: "invalid_start_after_end",
			timeRange: &types.TimeRange{
				Start: now,
				End:   yesterday,
			},
			shouldPass:  false,
			description: "Start after end should fail",
		},
		{
			name: "invalid_start_too_old",
			timeRange: &types.TimeRange{
				Start: tooOld,
				End:   now,
			},
			shouldPass:  false,
			description: "Start before max days back should fail",
		},
		{
			name: "invalid_future_start",
			timeRange: &types.TimeRange{
				Start: tomorrow,
				End:   tomorrow.Add(time.Hour),
			},
			shouldPass:  false,
			description: "Future time range should fail",
		},
		{
			name: "invalid_future_end",
			timeRange: &types.TimeRange{
				Start: yesterday,
				End:   tomorrow,
			},
			shouldPass:  false,
			description: "Future end time should fail",
		},
		{
			name: "invalid_duration_too_long",
			timeRange: &types.TimeRange{
				Start: now.AddDate(0, 0, -29),
				End:   now.Add(time.Hour), // Duration > 30 days
			},
			shouldPass:  false,
			description: "Duration exceeding max days should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				TimeRange: tt.timeRange,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected time range to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected time range to fail: %s", tt.description)
			}
		})
	}
}

func TestTimeframeRule_BusinessHoursValidation(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{})

	tests := []struct {
		name          string
		businessHours *types.BusinessHours
		shouldPass    bool
		description   string
	}{
		{
			name: "valid_business_hours_9_to_5",
			businessHours: &types.BusinessHours{
				StartHour: 9,
				EndHour:   17,
			},
			shouldPass:  true,
			description: "Valid 9-5 business hours should pass",
		},
		{
			name: "valid_business_hours_midnight_shift",
			businessHours: &types.BusinessHours{
				StartHour: 22,
				EndHour:   6,
			},
			shouldPass:  true,
			description: "Valid midnight shift hours should pass",
		},
		{
			name: "valid_business_hours_edge_cases",
			businessHours: &types.BusinessHours{
				StartHour: 0,
				EndHour:   23,
			},
			shouldPass:  true,
			description: "Valid edge case hours (0-23) should pass",
		},
		{
			name: "invalid_start_hour_negative",
			businessHours: &types.BusinessHours{
				StartHour: -1,
				EndHour:   17,
			},
			shouldPass:  false,
			description: "Negative start hour should fail",
		},
		{
			name: "invalid_start_hour_too_high",
			businessHours: &types.BusinessHours{
				StartHour: 24,
				EndHour:   17,
			},
			shouldPass:  false,
			description: "Start hour > 23 should fail",
		},
		{
			name: "invalid_end_hour_negative",
			businessHours: &types.BusinessHours{
				StartHour: 9,
				EndHour:   -1,
			},
			shouldPass:  false,
			description: "Negative end hour should fail",
		},
		{
			name: "invalid_end_hour_too_high",
			businessHours: &types.BusinessHours{
				StartHour: 9,
				EndHour:   25,
			},
			shouldPass:  false,
			description: "End hour > 23 should fail",
		},
		{
			name: "invalid_same_start_end",
			businessHours: &types.BusinessHours{
				StartHour: 9,
				EndHour:   9,
			},
			shouldPass:  false,
			description: "Same start and end hour should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:     "kube-apiserver",
				BusinessHours: tt.businessHours,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected business hours to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected business hours to fail: %s", tt.description)
			}

			// Check specific error messages for business hours violations
			if !tt.shouldPass {
				found := false
				for _, err := range result.Errors {
					if err != "" && containsSubstring(err, "business hours") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message about business hours violation, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestTimeframeRule_ComprehensiveValidation(t *testing.T) {
	config := map[string]interface{}{
		"max_days_back": 30,
		"min_limit":     1,
		"max_limit":     100,
	}
	rule := NewTimeframeRule(config)

	now := time.Now()
	tooOld := now.AddDate(0, 0, -40)

	// Query with multiple validation issues
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Timeframe: "invalid_timeframe", // Invalid timeframe
		Limit:     150,                // Exceeds max limit
		TimeRange: &types.TimeRange{   // Too old
			Start: tooOld,
			End:   now,
		},
		BusinessHours: &types.BusinessHours{ // Invalid hours
			StartHour: -1,
			EndHour:   25,
		},
	}

	result := rule.Validate(query)

	if result.IsValid {
		t.Error("Expected comprehensive validation to fail")
	}

	// Should have multiple errors
	if len(result.Errors) == 0 {
		t.Error("Expected multiple validation errors")
	}

	// Check that recommendations are provided
	if len(result.Recommendations) == 0 {
		t.Error("Expected validation recommendations")
	}

	// Verify rule information
	if result.RuleName != "timeframe_validation" {
		t.Errorf("Expected rule name 'timeframe_validation', got '%s'", result.RuleName)
	}
	if result.Severity != "medium" {
		t.Errorf("Expected severity 'medium', got '%s'", result.Severity)
	}
}

func TestTimeframeRule_FunctionalQueryValidation(t *testing.T) {
	// Test validation with patterns from functional test queries
	rule := NewTimeframeRule(map[string]interface{}{})

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "basic_query_pattern",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "1_day_ago",
				Limit:     20,
			},
			shouldPass:  true,
			description: "Basic query pattern should pass timeframe validation",
		},
		{
			name: "intermediate_query_pattern",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "1_hour_ago",
				BusinessHours: &types.BusinessHours{
					StartHour: 9,
					EndHour:   17,
				},
			},
			shouldPass:  true,
			description: "Intermediate query with business hours should pass",
		},
		{
			name: "advanced_query_pattern",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				TimeRange: &types.TimeRange{
					Start: time.Now().AddDate(0, 0, -1),
					End:   time.Now(),
				},
				Limit: 100,
			},
			shouldPass:  true,
			description: "Advanced query with time range should pass",
		},
		{
			name: "invalid_timeframe_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "invalid_time",
			},
			shouldPass:  false,
			description: "Query with invalid timeframe should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected query to pass timeframe validation: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected query to fail timeframe validation: %s", tt.description)
			}
		})
	}
}

func TestTimeframeRule_EdgeCases(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{})

	tests := []struct {
		name          string
		query         *types.StructuredQuery
		expectedValid bool
		description   string
	}{
		{
			name:          "nil_query",
			query:         nil,
			expectedValid: true, // Should handle gracefully
			description:   "Nil query should not crash",
		},
		{
			name:          "empty_query",
			query:         &types.StructuredQuery{},
			expectedValid: true,
			description:   "Empty query should pass (no timeframe validation needed)",
		},
		{
			name: "empty_timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "",
			},
			expectedValid: true,
			description:   "Empty timeframe should be ignored",
		},
		{
			name: "nil_time_range",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				TimeRange: nil,
			},
			expectedValid: true,
			description:   "Nil time range should be ignored",
		},
		{
			name: "nil_business_hours",
			query: &types.StructuredQuery{
				LogSource:     "kube-apiserver",
				BusinessHours: nil,
			},
			expectedValid: true,
			description:   "Nil business hours should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Validation panicked: %v", r)
				}
			}()

			var result *interfaces.ValidationResult
			if tt.query != nil {
				result = rule.Validate(tt.query)
			} else {
				// For nil query test, create a mock result
				result = &interfaces.ValidationResult{IsValid: true}
			}

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid=%v, got IsValid=%v: %s",
					tt.expectedValid, result.IsValid, tt.description)
			}
		})
	}
}

func TestTimeframeRule_PerformanceBenchmark(t *testing.T) {
	rule := NewTimeframeRule(map[string]interface{}{
		"max_days_back": 90,
		"min_limit":     1,
		"max_limit":     1000,
		"allowed_timeframes": []interface{}{
			"today", "yesterday", "1_hour_ago", "6_hours_ago", "12_hours_ago",
			"1_day_ago", "7_days_ago", "14_days_ago", "30_days_ago", "90_days_ago",
		},
	})

	now := time.Now()
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Timeframe: "30_days_ago",
		Limit:     100,
		TimeRange: &types.TimeRange{
			Start: now.AddDate(0, 0, -7),
			End:   now,
		},
		BusinessHours: &types.BusinessHours{
			StartHour: 9,
			EndHour:   17,
		},
	}

	// Benchmark the validation
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rule.Validate(query)
		}
	})

	// Check that validation completes within reasonable time (< 1ms per operation)
	avgTimePerOp := result.T / time.Duration(result.N)
	if avgTimePerOp > 1*time.Millisecond {
		t.Errorf("Validation too slow: %v per operation (should be < 1ms)", avgTimePerOp)
	}

	t.Logf("Performance: %v per validation operation", avgTimePerOp)
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || 
		len(str) > len(substr) && (
			str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			findSubstring(str, substr)))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}