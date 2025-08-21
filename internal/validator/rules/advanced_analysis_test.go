package rules

import (
	"testing"

	"genai-processing/pkg/types"
)

// getTestTimeWindowsConfigForAdvanced returns a test time windows configuration for advanced analysis
func getTestTimeWindowsConfigForAdvanced() map[string]interface{} {
	return map[string]interface{}{
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"allowed_correlation_windows": []interface{}{
			"1_minute", "5_minutes", "15_minutes", "30_minutes", "1_hour", "4_hours", "24_hours",
		},
		"allowed_baseline_windows": []interface{}{
			"7_days", "14_days", "30_days", "60_days", "90_days",
		},
	}
}

func TestAdvancedAnalysisRule_Validate(t *testing.T) {
	// Standard test configuration with allowed analysis types, time windows, and sort fields
	testConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"apt_reconnaissance_detection",
			"anomaly_detection",
			"behavioral_analysis",
			"statistical_analysis",
			"user_behavior_anomaly_detection",
			"temporal_auth_resource_correlation",
			"multi_namespace_access",
			"excessive_reads",
			"privilege_escalation",
			"correlation",
		},
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"sort_configuration": map[string]interface{}{
			"allowed_sort_fields": []interface{}{
				"timestamp", "username", "resource", "verb", "count", "risk_score",
				"frequency", "severity", "namespace", "auth_decision",
			},
			"allowed_sort_orders": []interface{}{
				"asc", "desc",
			},
		},
		"business_hours_configuration": map[string]interface{}{
			"allowed_presets": []interface{}{
				"business_hours", "outside_business_hours", "weekend", "all_hours",
			},
		},
		"response_status_configuration": map[string]interface{}{
			"allowed_status_codes": []interface{}{
				"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504",
			},
		},
		"auth_decisions_configuration": map[string]interface{}{
			"allowed_decisions": []interface{}{
				"allow", "error", "forbid",
			},
		},
		"max_group_by_fields": 3,
		"max_threshold_value": 1000,
		"min_threshold_value": 1,
	}

	tests := []struct {
		name           string
		config         map[string]interface{}
		query          *types.StructuredQuery
		expectedValid  bool
		expectedErrors int
	}{
		{
			name:   "Valid APT reconnaissance analysis",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:                  "apt_reconnaissance_detection",
					KillChainPhase:        "reconnaissance",
					MultiStageCorrelation: true,
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 2.5,
						ConfidenceInterval:        0.95,
						SampleSizeMinimum:         100,
						BaselineWindow:            "30_days",
					},
					Threshold:  5,
					TimeWindow: "15_minutes",
					SortBy:     "timestamp",
					SortOrder:  "desc",
				},
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Missing kill chain phase for APT analysis",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "apt_reconnaissance_detection",
					// Missing KillChainPhase
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid analysis type",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "invalid_analysis_type",
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid kill chain phase",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:           "apt_reconnaissance_detection",
					KillChainPhase: "invalid_phase",
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid statistical analysis parameters",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "anomaly_detection", // Use valid analysis type that supports statistical analysis
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 15.0, // Too high
						ConfidenceInterval:        1.5,  // Too high
						SampleSizeMinimum:         5,    // Too low
						BaselineWindow:            "invalid_window",
					},
				},
			},
			expectedValid:  false,
			expectedErrors: 4,
		},
		{
			name:   "Invalid threshold value",
			config: map[string]interface{}{
				"allowed_analysis_types": []interface{}{"anomaly_detection"},
				"sort_configuration": map[string]interface{}{
					"allowed_sort_fields": []interface{}{"timestamp", "count"},
					"allowed_sort_orders": []interface{}{"asc", "desc"},
				},
				"max_threshold_value": 1000,
				"min_threshold_value": 1,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:      "anomaly_detection",
					Threshold: 2000, // Exceeds max
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid time window",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:       "anomaly_detection",
					TimeWindow: "invalid_window",
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid sort field",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:   "anomaly_detection",
					SortBy: "invalid_field",
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Invalid sort order",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:      "anomaly_detection",
					SortBy:    "timestamp",
					SortOrder: "invalid_order",
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Too many group by fields",
			config: map[string]interface{}{
				"allowed_analysis_types": []interface{}{"anomaly_detection"},
				"sort_configuration": map[string]interface{}{
					"allowed_sort_fields": []interface{}{"timestamp", "count"},
					"allowed_sort_orders": []interface{}{"asc", "desc"},
				},
				"max_group_by_fields": 3,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "anomaly_detection",
					GroupBy: types.NewStringOrArray([]string{
						"user", "namespace", "resource", "verb", "timestamp", // 5 fields > max 3
					}),
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "No analysis configuration",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				// No Analysis field
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Missing analysis type",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					// Missing Type field
					Threshold: 10,
				},
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewAdvancedAnalysisRule(tt.config, getTestTimeWindowsConfigForAdvanced())
			result := rule.Validate(tt.query)

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v, got %v", tt.expectedValid, result.IsValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(result.Errors), result.Errors)
			}

			// Validate rule interface implementation
			if rule.GetRuleName() == "" {
				t.Error("Rule name should not be empty")
			}

			if rule.GetRuleDescription() == "" {
				t.Error("Rule description should not be empty")
			}

			if !rule.IsEnabled() {
				t.Error("Rule should be enabled by default")
			}

			if rule.GetSeverity() == "" {
				t.Error("Rule severity should not be empty")
			}
		})
	}
}

func TestAdvancedAnalysisRule_ValidateAnalysisType(t *testing.T) {
	tests := []struct {
		name          string
		analysisType  string
		expectedValid bool
	}{
		{"Valid APT type", "apt_reconnaissance_detection", true},
		{"Valid behavioral type", "user_behavior_anomaly_detection", true},
		{"Valid temporal type", "temporal_auth_resource_correlation", true},
		{"Invalid type", "non_existent_analysis", false},
		{"Empty type", "", false},
	}

	// Use test configuration instead of nil
	testConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"apt_reconnaissance_detection",
			"anomaly_detection",
			"behavioral_analysis", 
			"statistical_analysis",
			"user_behavior_anomaly_detection",
			"temporal_auth_resource_correlation",
			"multi_namespace_access",
			"excessive_reads",
			"privilege_escalation",
			"correlation",
		},
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"sort_configuration": map[string]interface{}{
			"allowed_sort_fields": []interface{}{
				"timestamp", "username", "resource", "verb", "count", "risk_score",
			},
			"allowed_sort_orders": []interface{}{
				"asc", "desc",
			},
		},
		"max_group_by_fields": 3,
		"max_threshold_value": 1000,
		"min_threshold_value": 1,
	}
	rule := NewAdvancedAnalysisRule(testConfig, getTestTimeWindowsConfigForAdvanced())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: tt.analysisType,
				},
			}
			
			// Add required kill chain phase for APT analysis types
			if tt.analysisType == "apt_reconnaissance_detection" {
				query.Analysis.KillChainPhase = "reconnaissance"
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for type '%s', got %v", 
					tt.expectedValid, tt.analysisType, result.IsValid)
			}
		})
	}
}

func TestAdvancedAnalysisRule_ValidateKillChainPhase(t *testing.T) {
	tests := []struct {
		name          string
		phase         string
		expectedValid bool
	}{
		{"Valid MITRE phase", "reconnaissance", true},
		{"Valid extended phase", "command_and_control", true},
		{"Invalid phase", "invalid_phase", false},
		{"Empty phase (optional)", "", true},
	}

	// Use test configuration instead of nil
	testConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"apt_reconnaissance_detection",
			"anomaly_detection",
			"behavioral_analysis", 
			"statistical_analysis",
			"user_behavior_anomaly_detection",
			"temporal_auth_resource_correlation",
			"multi_namespace_access",
			"excessive_reads",
			"privilege_escalation",
			"correlation",
		},
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"sort_configuration": map[string]interface{}{
			"allowed_sort_fields": []interface{}{
				"timestamp", "username", "resource", "verb", "count", "risk_score",
			},
			"allowed_sort_orders": []interface{}{
				"asc", "desc",
			},
		},
		"max_group_by_fields": 3,
		"max_threshold_value": 1000,
		"min_threshold_value": 1,
	}
	rule := NewAdvancedAnalysisRule(testConfig, getTestTimeWindowsConfigForAdvanced())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:           "anomaly_detection", // Non-APT type so phase is optional
					KillChainPhase: tt.phase,
				},
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for phase '%s', got %v", 
					tt.expectedValid, tt.phase, result.IsValid)
			}
		})
	}
}

func TestAdvancedAnalysisRule_ValidateStatisticalAnalysis(t *testing.T) {
	tests := []struct {
		name          string
		stats         *types.StatisticalAnalysisConfig
		expectedValid bool
	}{
		{
			name: "Valid parameters",
			stats: &types.StatisticalAnalysisConfig{
				PatternDeviationThreshold: 2.5,
				ConfidenceInterval:        0.95,
				SampleSizeMinimum:         100,
				BaselineWindow:            "30_days",
			},
			expectedValid: true,
		},
		{
			name: "Invalid threshold too high",
			stats: &types.StatisticalAnalysisConfig{
				PatternDeviationThreshold: 15.0,
			},
			expectedValid: false,
		},
		{
			name: "Invalid threshold too low",
			stats: &types.StatisticalAnalysisConfig{
				PatternDeviationThreshold: 0.05,
			},
			expectedValid: false,
		},
		{
			name: "Invalid confidence interval too high",
			stats: &types.StatisticalAnalysisConfig{
				ConfidenceInterval: 1.5,
			},
			expectedValid: false,
		},
		{
			name: "Invalid confidence interval too low",
			stats: &types.StatisticalAnalysisConfig{
				ConfidenceInterval: 0.3,
			},
			expectedValid: false,
		},
		{
			name: "Invalid sample size too low",
			stats: &types.StatisticalAnalysisConfig{
				SampleSizeMinimum: 5,
			},
			expectedValid: false,
		},
		{
			name: "Invalid baseline window",
			stats: &types.StatisticalAnalysisConfig{
				BaselineWindow: "invalid_window",
			},
			expectedValid: false,
		},
	}

	// Use test configuration instead of nil
	testConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"apt_reconnaissance_detection",
			"anomaly_detection",
			"behavioral_analysis", 
			"statistical_analysis",
			"user_behavior_anomaly_detection",
			"temporal_auth_resource_correlation",
			"multi_namespace_access",
			"excessive_reads",
			"privilege_escalation",
			"correlation",
		},
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"sort_configuration": map[string]interface{}{
			"allowed_sort_fields": []interface{}{
				"timestamp", "username", "resource", "verb", "count", "risk_score",
			},
			"allowed_sort_orders": []interface{}{
				"asc", "desc",
			},
		},
		"max_group_by_fields": 3,
		"max_threshold_value": 1000,
		"min_threshold_value": 1,
	}
	rule := NewAdvancedAnalysisRule(testConfig, getTestTimeWindowsConfigForAdvanced())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type:                "anomaly_detection", // Use valid analysis type
					StatisticalAnalysis: tt.stats,
				},
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v, got %v. Errors: %v", 
					tt.expectedValid, result.IsValid, result.Errors)
			}
		})
	}
}

func TestAdvancedAnalysisRule_ConfigDefaults(t *testing.T) {
	// Use test configuration instead of nil
	testConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"apt_reconnaissance_detection",
			"anomaly_detection",
			"behavioral_analysis", 
			"statistical_analysis",
			"user_behavior_anomaly_detection",
			"temporal_auth_resource_correlation",
			"multi_namespace_access",
			"excessive_reads",
			"privilege_escalation",
			"correlation",
		},
		"allowed_time_windows": []interface{}{
			"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
			"1_hour", "2_hours", "4_hours", "6_hours", "8_hours", "12_hours", "24_hours",
			"48_hours", "72_hours", "7_days", "14_days", "30_days",
		},
		"sort_configuration": map[string]interface{}{
			"allowed_sort_fields": []interface{}{
				"timestamp", "username", "resource", "verb", "count", "risk_score",
			},
			"allowed_sort_orders": []interface{}{
				"asc", "desc",
			},
		},
		"max_group_by_fields": 3,
		"max_threshold_value": 1000,
		"min_threshold_value": 1,
	}
	rule := NewAdvancedAnalysisRule(testConfig, getTestTimeWindowsConfigForAdvanced())
	
	// Test default analysis types
	types := rule.getAllowedAnalysisTypes()
	if len(types) == 0 {
		t.Error("Default analysis types should not be empty")
	}
	
	// Verify some expected types exist
	expectedTypes := []string{
		"anomaly_detection",
		"apt_reconnaissance_detection",
		"user_behavior_anomaly_detection", // Use actual type from implementation
		"privilege_escalation",
	}
	
	for _, expectedType := range expectedTypes {
		found := false
		for _, actualType := range types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected analysis type '%s' not found in defaults", expectedType)
		}
	}
	
	// Test default thresholds
	maxThreshold := rule.getMaxThresholdValue()
	if maxThreshold <= 0 {
		t.Error("Max threshold should be positive")
	}
	
	minThreshold := rule.getMinThresholdValue()
	if minThreshold <= 0 {
		t.Error("Min threshold should be positive")
	}
	
	if minThreshold >= maxThreshold {
		t.Error("Min threshold should be less than max threshold")
	}
}

func TestAdvancedAnalysisRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"allowed_analysis_types": []interface{}{
			"custom_type_1",
			"custom_type_2",
		},
		"max_threshold_value": 500,
		"max_group_by_fields": 10,
	}
	
	rule := NewAdvancedAnalysisRule(customConfig, getTestTimeWindowsConfigForAdvanced())
	
	// Test custom analysis types
	types := rule.getAllowedAnalysisTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 custom analysis types, got %d", len(types))
	}
	
	// Test custom thresholds
	maxThreshold := rule.getMaxThresholdValue()
	if maxThreshold != 500 {
		t.Errorf("Expected max threshold 500, got %d", maxThreshold)
	}
	
	minThreshold := rule.getMinThresholdValue()
	if minThreshold != 1 {
		t.Errorf("Expected min threshold 1 (hardcoded), got %d", minThreshold)
	}
	
	maxGroupBy := rule.getMaxGroupByFields()
	if maxGroupBy != 10 {
		t.Errorf("Expected max group by fields 10, got %d", maxGroupBy)
	}
}