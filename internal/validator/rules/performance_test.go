package rules

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestPerformanceRule_Validate(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]interface{}
		query            *types.StructuredQuery
		expectedValid    bool
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name:   "Simple query with low complexity",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
				Timeframe: "1_hour_ago",
				Limit:     10,
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name:   "Complex query with analysis",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "30_days_ago",
				Limit:     500,
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "user_behavior_anomaly_detection",
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 2.5,
					},
				},
			},
			expectedValid:    false, // Now exceeds limits
			expectedErrors:   2, // Complexity + CPU usage errors
			expectedWarnings: 0, // No warnings, errors instead
		},
		{
			name: "Query exceeding complexity limit",
			config: map[string]interface{}{
				"max_query_complexity_score": 50,
			},
			query: &types.StructuredQuery{
				LogSource: "node-auditd", // High volume source
				Timeframe: "90_days_ago",  // Long timeframe
				Limit:     1000,
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "user_behavior_anomaly_detection",
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 2.5,
					},
				},
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "node-auditd",
					SecondarySources: []string{"kube-apiserver", "oauth-server"},
					CorrelationWindow: "24_hours",
				},
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled: true,
						Algorithm: "ml_based",
					},
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm: "isolation_forest",
					},
				},
			},
			expectedValid:  false,
			expectedErrors: 3, // Complexity + memory + CPU errors
		},
		{
			name: "Query with too high memory usage",
			config: map[string]interface{}{
				"max_memory_usage_mb": 100,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "behavioral_analysis",
				},
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server", "node-auditd"},
				},
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
				},
			},
			expectedValid:  false,
			expectedErrors: 3, // Complexity + memory + CPU errors
		},
		{
			name: "Query exceeding execution time limit",
			config: map[string]interface{}{
				"max_execution_time_seconds": 30,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "user_behavior_anomaly_detection",
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 2.5,
					},
				},
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server", "node-auditd"},
				},
			},
			expectedValid:  false,
			expectedErrors: 3, // Complexity + CPU + execution time errors
		},
		{
			name: "Query with too large result limit",
			config: map[string]interface{}{
				"max_raw_results": 100,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     500, // > 100 max
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "Query with too many concurrent sources",
			config: map[string]interface{}{
				"max_concurrent_sources": 2,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server", "node-auditd"}, // 3 total > 2 max
				},
			},
			expectedValid:  false,
			expectedErrors: 3, // Complexity + CPU + concurrency errors
		},
		{
			name:   "Query with high CPU usage warning",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "user_behavior_anomaly_detection",
				},
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server"},
				},
			},
			expectedValid:    false, // Now exceeds limits
			expectedErrors:   2, // Complexity + CPU usage errors
			expectedWarnings: 0, // No warnings, errors instead
		},
		{
			name:   "Aggregated query with grouping",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				GroupBy:   *types.NewStringOrArray([]string{"user", "namespace"}),
				Limit:     100,
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewPerformanceRule(tt.config)
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

			// Check that performance details are included
			if result.Details["query_complexity_score"] == nil {
				t.Error("Complexity score should be included in details")
			}

			if result.Details["performance_tier"] == nil {
				t.Error("Performance tier should be included in details")
			}
		})
	}
}

func TestPerformanceRule_CalculateQueryComplexity(t *testing.T) {
	rule := NewPerformanceRule(nil)

	tests := []struct {
		name               string
		query              *types.StructuredQuery
		expectedMinScore   int
		expectedMaxScore   int
	}{
		{
			name: "Simple query",
			query: &types.StructuredQuery{
				LogSource: "oauth-server",
				Verb:      *types.NewStringOrArray("get"),
				Timeframe: "1_hour_ago",
			},
			expectedMinScore: 10,
			expectedMaxScore: 50,
		},
		{
			name: "Complex query with all features",
			query: &types.StructuredQuery{
				LogSource: "node-auditd", // High complexity source
				Timeframe: "90_days_ago", // Long timeframe
				Verb:      *types.NewStringOrArray([]string{"create", "update", "delete"}),
				Resource:  *types.NewStringOrArray([]string{"pods", "secrets", "configmaps"}),
				UserPattern: "admin@.*",
				IncludeChanges: true,
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "user_behavior_anomaly_detection",
					StatisticalAnalysis: &types.StatisticalAnalysisConfig{
						PatternDeviationThreshold: 2.5,
					},
					MultiStageCorrelation: true,
				},
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "node-auditd",
					SecondarySources: []string{"kube-apiserver", "oauth-server"},
					CorrelationWindow: "24_hours",
					CorrelationFields: []string{"user", "source_ip", "timestamp"},
				},
				BehavioralAnalysis: &types.BehavioralAnalysisConfig{
					UserProfiling: true,
					BaselineComparison: true,
					RiskScoring: &types.RiskScoringConfig{
						Enabled: true,
						Algorithm: "ml_based",
						RiskFactors: []string{"privilege_level", "resource_sensitivity"},
					},
					AnomalyDetection: &types.AnomalyDetectionConfig{
						Algorithm: "isolation_forest",
					},
				},
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX", "PCI-DSS"},
					Controls:  []string{"access_logging", "audit_trail"},
				},
			},
			expectedMinScore: 200,
			expectedMaxScore: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := rule.calculateQueryComplexity(tt.query)

			if complexity < tt.expectedMinScore {
				t.Errorf("Expected complexity >= %d, got %d", tt.expectedMinScore, complexity)
			}

			if complexity > tt.expectedMaxScore {
				t.Errorf("Expected complexity <= %d, got %d", tt.expectedMaxScore, complexity)
			}
		})
	}
}

func TestPerformanceRule_CalculateLogSourceComplexity(t *testing.T) {
	rule := NewPerformanceRule(nil)

	tests := []struct {
		logSource        string
		expectedRange    [2]int // min, max
	}{
		{"kube-apiserver", [2]int{10, 20}},
		{"node-auditd", [2]int{15, 25}},
		{"oauth-server", [2]int{5, 15}},
		{"unknown-source", [2]int{8, 12}}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.logSource, func(t *testing.T) {
			complexity := rule.calculateLogSourceComplexity(tt.logSource)

			if complexity < tt.expectedRange[0] || complexity > tt.expectedRange[1] {
				t.Errorf("Expected complexity between %d and %d for source '%s', got %d",
					tt.expectedRange[0], tt.expectedRange[1], tt.logSource, complexity)
			}
		})
	}
}

func TestPerformanceRule_CalculateTimeRangeComplexity(t *testing.T) {
	rule := NewPerformanceRule(nil)

	tests := []struct {
		timeframe    string
		expectedMin  int
		expectedMax  int
	}{
		{"today", 1, 5},
		{"1_hour_ago", 1, 3},
		{"30_days_ago", 35, 45},
		{"90_days_ago", 75, 85},
		{"unknown_timeframe", 3, 7}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.timeframe, func(t *testing.T) {
			query := &types.StructuredQuery{
				Timeframe: tt.timeframe,
			}

			complexity := rule.calculateTimeRangeComplexity(query)

			if complexity < tt.expectedMin || complexity > tt.expectedMax {
				t.Errorf("Expected complexity between %d and %d for timeframe '%s', got %d",
					tt.expectedMin, tt.expectedMax, tt.timeframe, complexity)
			}
		})
	}
}

func TestPerformanceRule_EstimateResourceUsage(t *testing.T) {
	rule := NewPerformanceRule(nil)

	tests := []struct {
		name             string
		complexity       int
		query            *types.StructuredQuery
		expectedMemoryMin int
		expectedCPUMin   int
	}{
		{
			name:             "Low complexity query",
			complexity:       20,
			query:            &types.StructuredQuery{LogSource: "oauth-server"},
			expectedMemoryMin: 50,
			expectedCPUMin:   10,
		},
		{
			name:       "High complexity with analysis",
			complexity: 100,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Analysis: &types.AdvancedAnalysisConfig{
					Type: "anomaly_detection",
				},
			},
			expectedMemoryMin: 200,
			expectedCPUMin:   30,
		},
		{
			name:       "Multi-source query",
			complexity: 80,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					SecondarySources: []string{"oauth-server", "node-auditd"},
				},
			},
			expectedMemoryMin: 200,
			expectedCPUMin:   30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := rule.estimateMemoryUsage(tt.complexity, tt.query)
			cpu := rule.estimateCPUUsage(tt.complexity, tt.query)

			if memory < tt.expectedMemoryMin {
				t.Errorf("Expected memory >= %d MB, got %d MB", tt.expectedMemoryMin, memory)
			}

			if cpu < tt.expectedCPUMin {
				t.Errorf("Expected CPU >= %d%%, got %d%%", tt.expectedCPUMin, cpu)
			}

			// CPU should never exceed 100%
			if cpu > 100 {
				t.Errorf("CPU usage should not exceed 100%%, got %d%%", cpu)
			}
		})
	}
}

func TestPerformanceRule_GetPerformanceTier(t *testing.T) {
	rule := NewPerformanceRule(nil)
	maxComplexity := rule.getMaxComplexityScore()

	tests := []struct {
		complexity   int
		expectedTier string
	}{
		{maxComplexity / 6, "low"},     // < 1/3 of max
		{maxComplexity / 2, "medium"},  // 1/3 to 2/3 of max
		{maxComplexity * 3 / 4, "high"}, // > 2/3 of max
	}

	for _, tt := range tests {
		t.Run(tt.expectedTier, func(t *testing.T) {
			tier := rule.getPerformanceTier(tt.complexity)
			if tier != tt.expectedTier {
				t.Errorf("Expected tier '%s' for complexity %d, got '%s'",
					tt.expectedTier, tt.complexity, tier)
			}
		})
	}
}

func TestPerformanceRule_ConfigDefaults(t *testing.T) {
	rule := NewPerformanceRule(nil)

	// Test all default getters return positive values
	defaults := map[string]int{
		"max_complexity_score":    rule.getMaxComplexityScore(),
		"max_memory_usage_mb":     rule.getMaxMemoryUsageMB(),
		"max_cpu_usage_percent":   rule.getMaxCPUUsagePercent(),
		"max_execution_time_seconds": rule.getMaxExecutionTimeSeconds(),
		"max_raw_results":         rule.getMaxRawResults(),
		"max_aggregated_results":  rule.getMaxAggregatedResults(),
		"max_concurrent_sources":  rule.getMaxConcurrentSources(),
	}

	for name, value := range defaults {
		if value <= 0 {
			t.Errorf("Default %s should be positive, got %d", name, value)
		}
	}

	// Test some reasonable ranges
	if rule.getMaxCPUUsagePercent() > 100 {
		t.Error("Max CPU usage should not exceed 100%")
	}

	if rule.getMaxRawResults() < rule.getMaxAggregatedResults() {
		t.Error("Max raw results should be >= max aggregated results")
	}
}

func TestPerformanceRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"max_query_complexity_score": 75,
		"max_memory_usage_mb":        512,
		"max_cpu_usage_percent":      25,
		"max_execution_time_seconds": 120,
		"max_raw_results":            5000,
		"max_aggregated_results":     500,
		"max_concurrent_sources":     3,
	}

	rule := NewPerformanceRule(customConfig)

	// Test custom values are applied
	if rule.getMaxComplexityScore() != 75 {
		t.Errorf("Expected max complexity 75, got %d", rule.getMaxComplexityScore())
	}

	if rule.getMaxMemoryUsageMB() != 512 {
		t.Errorf("Expected max memory 512 MB, got %d", rule.getMaxMemoryUsageMB())
	}

	if rule.getMaxCPUUsagePercent() != 25 {
		t.Errorf("Expected max CPU 25%%, got %d", rule.getMaxCPUUsagePercent())
	}

	if rule.getMaxExecutionTimeSeconds() != 120 {
		t.Errorf("Expected max execution time 120s, got %d", rule.getMaxExecutionTimeSeconds())
	}

	if rule.getMaxRawResults() != 5000 {
		t.Errorf("Expected max raw results 5000, got %d", rule.getMaxRawResults())
	}

	if rule.getMaxAggregatedResults() != 500 {
		t.Errorf("Expected max aggregated results 500, got %d", rule.getMaxAggregatedResults())
	}

	if rule.getMaxConcurrentSources() != 3 {
		t.Errorf("Expected max concurrent sources 3, got %d", rule.getMaxConcurrentSources())
	}
}

func TestPerformanceRule_ValidationDetails(t *testing.T) {
	rule := NewPerformanceRule(nil)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Timeframe: "1_hour_ago",
		Limit:     50,
	}

	result := rule.Validate(query)

	// Check that all expected details are present
	expectedDetails := []string{
		"query_complexity_score",
		"max_complexity_allowed",
		"performance_tier",
		"estimated_memory_mb",
		"estimated_cpu_percent",
		"estimated_execution_seconds",
		"uses_aggregation",
		"effective_limit",
		"concurrent_sources",
	}

	for _, detail := range expectedDetails {
		if result.Details[detail] == nil {
			t.Errorf("Expected detail '%s' to be present in result", detail)
		}
	}

	// Verify some detail types
	if complexity, ok := result.Details["query_complexity_score"].(int); !ok || complexity <= 0 {
		t.Error("Complexity score should be a positive integer")
	}

	if tier, ok := result.Details["performance_tier"].(string); !ok || tier == "" {
		t.Error("Performance tier should be a non-empty string")
	}

	if usesAgg, ok := result.Details["uses_aggregation"].(bool); !ok {
		t.Error("Uses aggregation should be a boolean")
	} else if usesAgg && query.GroupBy.IsEmpty() && query.Analysis == nil {
		t.Error("Should not indicate aggregation when no grouping or analysis present")
	}
}