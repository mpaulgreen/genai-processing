package rules

import (
	"testing"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestBehavioralAnalyticsRule_Validate(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]interface{}
		query            *types.StructuredQuery
		expectedValid    bool
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name:   "Valid behavioral analytics configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling:      true,
					BaselineComparison: true,
					BaselineWindow:     "30_days",
					LearningPeriod:     "7_days",
					RiskScoring: &types.RiskScoringConfig{
						Enabled:   true,
						Algorithm: "weighted_sum",
						RiskFactors: []string{"privilege_level", "resource_sensitivity", "timing_anomaly"},
						WeightingScheme: map[string]float64{
							"privilege_level":     0.4,
							"resource_sensitivity": 0.3,
							"timing_anomaly":      0.3,
						},
					},
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm:     "isolation_forest",
						Contamination: 0.1,
						Sensitivity:   0.8,
						Threshold:     2.5,
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1, // Performance warning for all features enabled
		},
		{
			name:   "Risk scoring without user profiling",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: false,
					RiskScoring: &types.RiskScoringConfig{
						Enabled: true,
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // Warning about disabled user profiling
		},
		{
			name:   "Baseline comparison without baseline window",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					BaselineComparison: true,
					// Missing BaselineWindow
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // Warning about disabled user profiling
		},
		{
			name:   "Invalid baseline window",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					BaselineWindow: "invalid_window",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // Warning about disabled user profiling
		},
		{
			name:   "Invalid learning period",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					LearningPeriod: "invalid_period",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // Warning about disabled user profiling
		},
		{
			name:   "Invalid risk scoring algorithm",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled:   true,
						Algorithm: "invalid_algorithm",
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // Baseline window + Learning period warnings
		},
		{
			name: "Too many risk factors",
			config: map[string]interface{}{
				"max_risk_factors": 2,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled:     true,
						RiskFactors: []string{"privilege_level", "resource_sensitivity", "timing_anomaly"}, // 3 > 2 max
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // Baseline window + Learning period warnings
		},
		{
			name:   "Invalid risk factor",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled:     true,
						RiskFactors: []string{"invalid_factor"},
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // Baseline window + Learning period warnings
		},
		{
			name:   "Invalid weighting scheme - weights don't sum to 1",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled: true,
						WeightingScheme: map[string]float64{
							"privilege_level": 0.3,
							"timing_anomaly":  0.4, // Total: 0.7, not 1.0
						},
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // Baseline window + Learning period warnings
		},
		{
			name:   "Invalid weighting scheme - negative weight",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled: true,
						WeightingScheme: map[string]float64{
							"privilege_level": -0.1, // Negative weight
							"timing_anomaly":  1.1,
						},
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // Baseline window + Learning period warnings
		},
		{
			name:   "Invalid anomaly detection algorithm",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm: "invalid_algorithm",
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // User profiling disabled + Anomaly detection without baseline
		},
		{
			name:   "Invalid contamination value",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm:     "isolation_forest",
						Contamination: 1.5, // > 1.0
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 3, // User profiling disabled + High contamination + Anomaly detection without baseline
		},
		{
			name:   "Invalid sensitivity value",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm:   "isolation_forest",
						Sensitivity: -0.1, // < 0.0
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // User profiling disabled + Anomaly detection without baseline
		},
		{
			name:   "Invalid threshold value",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm: "z_score",
						Threshold: 15.0, // > 10.0 default max
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 3, // User profiling disabled + Z-score threshold warning + Anomaly detection without baseline
		},
		{
			name:   "No behavioral analysis configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				// No BehavioralAnalysis field
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Disabled user profiling warning",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: false,
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name:   "Anomaly detection without baseline or profiling warning",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling:      false,
					BaselineComparison: false,
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm: "isolation_forest",
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 2, // Disabled profiling + anomaly without baseline
		},
		{
			name:   "High contamination warning for isolation forest",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm:     "isolation_forest",
						Contamination: 0.4, // > 0.3 threshold for warning
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 3, // User profiling disabled + High contamination + Anomaly detection without baseline
		},
		{
			name: "Baseline window too short",
			config: map[string]interface{}{
				"baseline_window_limits": map[string]interface{}{
					"min_baseline_days": 14,
					"max_baseline_days": 90,
				},
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					BaselineWindow: "7_days", // < 14 day minimum
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // User profiling disabled warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewBehavioralAnalyticsRule(tt.config)
			result := rule.Validate(tt.query)

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v, got %v", tt.expectedValid, result.IsValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(result.Errors), result.Errors)
			}

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("Expected %d warnings, got %d: %v", tt.expectedWarnings, len(result.Warnings), result.Warnings)
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

func TestBehavioralAnalyticsRule_ValidateWeightingScheme(t *testing.T) {
	tests := []struct {
		name          string
		scheme        map[string]float64
		expectedValid bool
	}{
		{
			name: "Valid weights summing to 1.0",
			scheme: map[string]float64{
				"privilege_level":     0.4,
				"resource_sensitivity": 0.3,
				"timing_anomaly":      0.3,
			},
			expectedValid: true,
		},
		{
			name: "Valid weights with tolerance",
			scheme: map[string]float64{
				"privilege_level": 0.33,
				"timing_anomaly":  0.34,
				"access_pattern":  0.33,
			},
			expectedValid: true,
		},
		{
			name: "Weights don't sum to 1.0",
			scheme: map[string]float64{
				"privilege_level": 0.3,
				"timing_anomaly":  0.4,
			},
			expectedValid: false,
		},
		{
			name: "Negative weight",
			scheme: map[string]float64{
				"privilege_level": -0.1,
				"timing_anomaly":  1.1,
			},
			expectedValid: false,
		},
		{
			name: "Weight greater than 1.0",
			scheme: map[string]float64{
				"privilege_level": 1.5,
			},
			expectedValid: false,
		},
		{
			name:          "Empty scheme",
			scheme:        map[string]float64{},
			expectedValid: false,
		},
	}

	rule := NewBehavioralAnalyticsRule(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &interfaces.ValidationResult{
				IsValid: true,
				Errors:  []string{},
			}

			err := rule.validateWeightingScheme(tt.scheme, result)
			isValid := (err == nil)

			if isValid != tt.expectedValid {
				t.Errorf("Expected valid = %v for scheme %v, got %v (error: %v)", 
					tt.expectedValid, tt.scheme, isValid, err)
			}
		})
	}
}

func TestBehavioralAnalyticsRule_CalculatePerformanceImpact(t *testing.T) {
	rule := NewBehavioralAnalyticsRule(nil)

	tests := []struct {
		name           string
		config         *types.BehavioralAnalysisConfig
		expectedMinScore int
	}{
		{
			name: "Minimal configuration",
			config: &types.BehavioralAnalysisConfig{
				UserProfiling: false,
			},
			expectedMinScore: 10, // Base score only
		},
		{
			name: "User profiling only",
			config: &types.BehavioralAnalysisConfig{
				UserProfiling: true,
			},
			expectedMinScore: 25, // Base + user profiling
		},
		{
			name: "Full configuration",
			config: &types.BehavioralAnalysisConfig{
				UserProfiling:      true,
				BaselineComparison: true,
				RiskScoring: &types.RiskScoringConfig{
					Enabled:   true,
					Algorithm: "ml_based",
					RiskFactors: []string{"factor1", "factor2", "factor3"},
				},
				AnomalyDetection: &types.AnomalyDetectionConfig{
					Algorithm: "isolation_forest",
				},
			},
			expectedMinScore: 100, // High complexity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := rule.calculatePerformanceImpact(tt.config)
			if score < tt.expectedMinScore {
				t.Errorf("Expected performance score >= %d, got %d", tt.expectedMinScore, score)
			}
		})
	}
}

func TestBehavioralAnalyticsRule_ConfigDefaults(t *testing.T) {
	rule := NewBehavioralAnalyticsRule(nil)

	// Test default risk factors
	factors := rule.getAllowedRiskFactors()
	if len(factors) == 0 {
		t.Error("Default risk factors should not be empty")
	}

	// Verify some expected factors exist
	expectedFactors := []string{
		"privilege_level",
		"resource_sensitivity", 
		"timing_anomaly",
		"access_pattern",
	}

	for _, expectedFactor := range expectedFactors {
		found := false
		for _, actualFactor := range factors {
			if actualFactor == expectedFactor {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected risk factor '%s' not found in defaults", expectedFactor)
		}
	}

	// Test default baseline windows
	windows := rule.getAllowedBaselineWindows()
	if len(windows) == 0 {
		t.Error("Default baseline windows should not be empty")
	}

	// Test default learning periods
	periods := rule.getAllowedLearningPeriods()
	if len(periods) == 0 {
		t.Error("Default learning periods should not be empty")
	}

	// Test default limits
	maxFactors := rule.getMaxRiskFactors()
	if maxFactors <= 0 {
		t.Error("Max risk factors should be positive")
	}

	maxScore := rule.getMaxPerformanceScore()
	if maxScore <= 0 {
		t.Error("Max performance score should be positive")
	}
}

func TestBehavioralAnalyticsRule_AlgorithmSpecificValidation(t *testing.T) {
	rule := NewBehavioralAnalyticsRule(nil)

	tests := []struct {
		name             string
		algorithm        string
		contamination    float64
		threshold        float64
		sensitivity      float64
		expectedWarnings int
	}{
		{
			name:             "Isolation forest with high contamination",
			algorithm:        "isolation_forest",
			contamination:    0.4,
			expectedWarnings: 3, // User profiling disabled + High contamination + Anomaly without baseline
		},
		{
			name:             "Z-score with low threshold",
			algorithm:        "z_score",
			threshold:        1.5,
			expectedWarnings: 3, // User profiling disabled + Z-score threshold + Anomaly without baseline
		},
		{
			name:             "Z-score with high threshold",
			algorithm:        "z_score",
			threshold:        5.0,
			expectedWarnings: 3, // User profiling disabled + Z-score threshold + Anomaly without baseline
		},
		{
			name:             "Statistical without sensitivity",
			algorithm:        "statistical",
			sensitivity:      0.0,
			expectedWarnings: 3, // User profiling disabled + Statistical sensitivity + Anomaly without baseline
		},
		{
			name:             "Isolation forest with good parameters",
			algorithm:        "isolation_forest",
			contamination:    0.1,
			expectedWarnings: 2, // User profiling disabled + Anomaly without baseline
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm:     tt.algorithm,
						Contamination: tt.contamination,
						Threshold:     tt.threshold,
						Sensitivity:   tt.sensitivity,
					},
				},
			}

			result := rule.Validate(query)
			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("Expected %d warnings for %s, got %d: %v", 
					tt.expectedWarnings, tt.algorithm, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestBehavioralAnalyticsRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"allowed_risk_factors": []interface{}{
			"custom_factor_1",
			"custom_factor_2",
		},
		"max_risk_factors": 5,
		"baseline_window_limits": map[string]interface{}{
			"min_baseline_days": 14,
			"max_baseline_days": 60,
		},
		"anomaly_threshold_limits": map[string]interface{}{
			"min": 0.5,
			"max": 5.0,
		},
		"max_performance_score": 150,
	}

	rule := NewBehavioralAnalyticsRule(customConfig)

	// Test custom risk factors
	factors := rule.getAllowedRiskFactors()
	if len(factors) != 2 {
		t.Errorf("Expected 2 custom risk factors, got %d", len(factors))
	}

	// Test custom max risk factors
	maxFactors := rule.getMaxRiskFactors()
	if maxFactors != 5 {
		t.Errorf("Expected max risk factors 5, got %d", maxFactors)
	}

	// Test custom baseline limits
	limits := rule.getBaselineWindowLimits()
	if limits["min_baseline_days"].(int) != 14 {
		t.Errorf("Expected min baseline days 14, got %v", limits["min_baseline_days"])
	}

	// Test custom anomaly threshold limits
	thresholdLimits := rule.getAnomalyThresholdLimits()
	if thresholdLimits["min"].(float64) != 0.5 {
		t.Errorf("Expected min threshold 0.5, got %v", thresholdLimits["min"])
	}

	// Test custom max performance score
	maxScore := rule.getMaxPerformanceScore()
	if maxScore != 150 {
		t.Errorf("Expected max performance score 150, got %d", maxScore)
	}
}