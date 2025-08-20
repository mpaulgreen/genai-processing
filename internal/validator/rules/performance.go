package rules

import (
	"fmt"
	"math"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// PerformanceRule implements validation for query performance and resource usage
type PerformanceRule struct {
	config  map[string]interface{}
	enabled bool
}

// NewPerformanceRule creates a new performance validation rule
func NewPerformanceRule(config map[string]interface{}) *PerformanceRule {
	return &PerformanceRule{
		config:  config,
		enabled: true,
	}
}

// Validate applies performance validation to the query
func (r *PerformanceRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "performance_validation",
		Severity:        "info",
		Message:         "Performance validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Calculate and validate query complexity
	complexityScore := r.calculateQueryComplexity(query)
	r.validateComplexityScore(complexityScore, result)

	// Validate resource usage estimates
	r.validateResourceUsage(query, complexityScore, result)

	// Validate execution time limits
	r.validateExecutionTimeLimits(query, complexityScore, result)

	// Validate result set size limits
	r.validateResultSetLimits(query, result)

	// Validate concurrent operation limits
	r.validateConcurrencyLimits(query, result)

	// Provide performance recommendations
	r.addPerformanceRecommendations(query, complexityScore, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Performance validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Reduce query complexity to improve performance",
			"Consider limiting result set size",
			"Optimize time range and filtering criteria",
			"Use more specific log sources and patterns")
	} else if len(result.Warnings) > 0 {
		result.Severity = "warning"
		result.Message = "Performance validation passed with warnings"
	}

	// Add complexity details to result
	result.Details["query_complexity_score"] = complexityScore
	result.Details["max_complexity_allowed"] = r.getMaxComplexityScore()
	result.Details["performance_tier"] = r.getPerformanceTier(complexityScore)

	return result
}

// calculateQueryComplexity calculates a complexity score for the query
func (r *PerformanceRule) calculateQueryComplexity(query *types.StructuredQuery) int {
	complexity := 0

	// Base complexity
	complexity += 10

	// Log source complexity
	complexity += r.calculateLogSourceComplexity(query.LogSource)

	// Field-based complexity
	complexity += r.calculateFieldComplexity(query)

	// Time range complexity
	complexity += r.calculateTimeRangeComplexity(query)

	// Pattern matching complexity
	complexity += r.calculatePatternComplexity(query)

	// Advanced analysis complexity
	if query.Analysis != nil {
		complexity += r.calculateAnalysisComplexity(query.Analysis)
	}

	// Multi-source complexity
	if query.MultiSource != nil {
		complexity += r.calculateMultiSourceComplexity(query.MultiSource)
	}

	// Behavioral analysis complexity
	if query.BehavioralAnalysis != nil {
		complexity += r.calculateBehavioralComplexity(query.BehavioralAnalysis)
	}

	// Compliance framework complexity
	if query.ComplianceFramework != nil {
		complexity += r.calculateComplianceComplexity(query.ComplianceFramework)
	}

	return complexity
}

// calculateLogSourceComplexity calculates complexity based on log source
func (r *PerformanceRule) calculateLogSourceComplexity(logSource string) int {
	// Different log sources have different volume and processing complexity
	sourceComplexity := map[string]int{
		"kube-apiserver":     15, // High volume, complex processing
		"openshift-apiserver": 12, // Moderate volume
		"oauth-server":        8,  // Lower volume
		"oauth-apiserver":     10, // Moderate volume
		"node-auditd":         20, // Very high volume, simple processing
	}

	if complexity, exists := sourceComplexity[logSource]; exists {
		return complexity
	}
	return 10 // Default complexity
}

// calculateFieldComplexity calculates complexity based on query fields
func (r *PerformanceRule) calculateFieldComplexity(query *types.StructuredQuery) int {
	complexity := 0

	// String/array field complexity
	fields := []*types.StringOrArray{
		&query.Verb, &query.Resource, &query.Namespace, &query.User,
		&query.ResponseStatus, &query.SourceIP, &query.GroupBy,
	}

	for _, field := range fields {
		if !field.IsEmpty() {
			if field.IsArray() {
				complexity += len(field.GetArray()) * 2
			} else {
				complexity += 3
			}
		}
	}

	// String arrays
	complexity += len(query.ExcludeUsers) * 2
	complexity += len(query.ExcludeResources) * 2

	// Pattern fields (more expensive)
	patterns := []string{
		query.UserPattern, query.NamespacePattern, query.ResourceNamePattern,
		query.RequestURIPattern, query.AuthorizationReasonPattern, query.ResponseMessagePattern,
	}

	for _, pattern := range patterns {
		if pattern != "" {
			complexity += 8 // Regex processing is expensive
		}
	}

	// Sorting and grouping
	if query.SortBy != "" {
		complexity += 5
	}

	return complexity
}

// calculateTimeRangeComplexity calculates complexity based on time range
func (r *PerformanceRule) calculateTimeRangeComplexity(query *types.StructuredQuery) int {
	timeframeComplexity := map[string]int{
		"today":         2,
		"yesterday":     4,
		"1_hour_ago":    1,
		"6_hours_ago":   3,
		"12_hours_ago":  5,
		"24_hours_ago":  8,
		"7_days_ago":    15,
		"14_days_ago":   25,
		"30_days_ago":   40,
		"60_days_ago":   60,
		"90_days_ago":   80,
		"last_week":     15,
		"last_month":    40,
	}

	if complexity, exists := timeframeComplexity[query.Timeframe]; exists {
		return complexity
	}

	// Custom time range complexity
	if query.TimeRange != nil {
		// Calculate based on duration (simplified)
		return 20 // Default for custom ranges
	}

	return 5 // Default complexity
}

// calculatePatternComplexity calculates complexity for pattern matching
func (r *PerformanceRule) calculatePatternComplexity(query *types.StructuredQuery) int {
	complexity := 0

	// Include changes adds significant complexity
	if query.IncludeChanges {
		complexity += 25
	}

	// Request object filtering
	if query.RequestObjectFilter != "" {
		complexity += 15
	}

	return complexity
}

// calculateAnalysisComplexity calculates complexity for advanced analysis
func (r *PerformanceRule) calculateAnalysisComplexity(analysis *types.AdvancedAnalysisConfig) int {
	complexity := 20 // Base analysis complexity

	// Analysis type complexity
	analysisTypeComplexity := map[string]int{
		"anomaly_detection":                                    30,
		"correlation":                                          25,
		"apt_reconnaissance_detection":                         40,
		"lateral_movement_detection":                           35,
		"behavioral_analysis":                                  45,
		"user_behavior_anomaly_detection":                     50,
		"cross_source_correlation":                             60,
		"timeline_reconstruction":                              40,
		"rbac_violation_privilege_escalation_analysis":        35,
		"oauth_token_manipulation_investigation":              30,
	}

	if typeComplexity, exists := analysisTypeComplexity[analysis.Type]; exists {
		complexity += typeComplexity
	} else {
		complexity += 20 // Default analysis complexity
	}

	// Statistical analysis adds complexity
	if analysis.StatisticalAnalysis != nil {
		complexity += 25
	}

	// Multi-stage correlation
	if analysis.MultiStageCorrelation {
		complexity += 20
	}

	// Grouping complexity
	if analysis.GroupBy != nil && !analysis.GroupBy.IsEmpty() {
		if analysis.GroupBy.IsArray() {
			complexity += len(analysis.GroupBy.GetArray()) * 5
		} else {
			complexity += 5
		}
	}

	return complexity
}

// calculateMultiSourceComplexity calculates complexity for multi-source correlation
func (r *PerformanceRule) calculateMultiSourceComplexity(multiSource *types.MultiSourceConfig) int {
	complexity := 30 // Base multi-source complexity

	// Complexity grows exponentially with number of sources
	sourceCount := 1 + len(multiSource.SecondarySources)
	complexity += int(math.Pow(float64(sourceCount), 1.5)) * 10

	// Correlation fields complexity
	complexity += len(multiSource.CorrelationFields) * 8

	// Correlation window complexity
	windowComplexity := map[string]int{
		"1_minute":  2,
		"5_minutes": 5,
		"15_minutes": 10,
		"30_minutes": 15,
		"1_hour":    20,
		"6_hours":   40,
		"24_hours":  80,
	}

	if winComplexity, exists := windowComplexity[multiSource.CorrelationWindow]; exists {
		complexity += winComplexity
	} else {
		complexity += 15 // Default window complexity
	}

	// Join type complexity
	joinComplexity := map[string]int{
		"inner": 5,
		"left":  10,
		"right": 15,
		"full":  25,
	}

	if joinComp, exists := joinComplexity[multiSource.JoinType]; exists {
		complexity += joinComp
	}

	return complexity
}

// calculateBehavioralComplexity calculates complexity for behavioral analysis
func (r *PerformanceRule) calculateBehavioralComplexity(behavioral *types.BehavioralAnalysisConfig) int {
	complexity := 20 // Base behavioral complexity

	if behavioral.UserProfiling {
		complexity += 30
	}

	if behavioral.BaselineComparison {
		complexity += 25
	}

	if behavioral.RiskScoring != nil && behavioral.RiskScoring.Enabled {
		complexity += 35
		complexity += len(behavioral.RiskScoring.RiskFactors) * 5

		if behavioral.RiskScoring.Algorithm == "ml_based" {
			complexity += 40
		}
	}

	if behavioral.AnomalyDetection != nil {
		complexity += 45
		if behavioral.AnomalyDetection.Algorithm == "isolation_forest" {
			complexity += 25
		}
	}

	return complexity
}

// calculateComplianceComplexity calculates complexity for compliance framework
func (r *PerformanceRule) calculateComplianceComplexity(compliance *types.ComplianceFrameworkConfig) int {
	complexity := 10 // Base compliance complexity

	// Complexity per standard
	complexity += len(compliance.Standards) * 8

	// Complexity per control
	complexity += len(compliance.Controls) * 5

	// Evidence collection adds complexity
	if compliance.Reporting != nil && compliance.Reporting.IncludeEvidence {
		complexity += 20
	}

	return complexity
}

// validateComplexityScore validates the overall complexity score
func (r *PerformanceRule) validateComplexityScore(complexity int, result *interfaces.ValidationResult) {
	maxComplexity := r.getMaxComplexityScore()
	warningThreshold := maxComplexity * 3 / 4

	if complexity > maxComplexity {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Query complexity score %d exceeds maximum allowed %d",
				complexity, maxComplexity))
	} else if complexity > warningThreshold {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High query complexity score %d may impact performance", complexity))
	}
}

// validateResourceUsage validates estimated resource usage
func (r *PerformanceRule) validateResourceUsage(query *types.StructuredQuery, complexity int, result *interfaces.ValidationResult) {
	// Estimate memory usage
	estimatedMemoryMB := r.estimateMemoryUsage(complexity, query)
	maxMemoryMB := r.getMaxMemoryUsageMB()

	if estimatedMemoryMB > maxMemoryMB {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Estimated memory usage %d MB exceeds limit %d MB",
				estimatedMemoryMB, maxMemoryMB))
	} else if estimatedMemoryMB > maxMemoryMB*3/4 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High estimated memory usage %d MB", estimatedMemoryMB))
	}

	// Estimate CPU usage
	estimatedCPUPercent := r.estimateCPUUsage(complexity, query)
	maxCPUPercent := r.getMaxCPUUsagePercent()

	if estimatedCPUPercent > maxCPUPercent {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Estimated CPU usage %d%% exceeds limit %d%%",
				estimatedCPUPercent, maxCPUPercent))
	} else if estimatedCPUPercent > maxCPUPercent*3/4 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High estimated CPU usage %d%%", estimatedCPUPercent))
	}

	result.Details["estimated_memory_mb"] = estimatedMemoryMB
	result.Details["estimated_cpu_percent"] = estimatedCPUPercent
}

// validateExecutionTimeLimits validates estimated execution time
func (r *PerformanceRule) validateExecutionTimeLimits(query *types.StructuredQuery, complexity int, result *interfaces.ValidationResult) {
	estimatedTimeSeconds := r.estimateExecutionTime(complexity, query)
	maxTimeSeconds := r.getMaxExecutionTimeSeconds()

	if estimatedTimeSeconds > maxTimeSeconds {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Estimated execution time %d seconds exceeds limit %d seconds",
				estimatedTimeSeconds, maxTimeSeconds))
	} else if estimatedTimeSeconds > maxTimeSeconds*3/4 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Long estimated execution time %d seconds", estimatedTimeSeconds))
	}

	result.Details["estimated_execution_seconds"] = estimatedTimeSeconds
}

// validateResultSetLimits validates result set size limits
func (r *PerformanceRule) validateResultSetLimits(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if query.Limit == 0 {
		query.Limit = 20 // Default limit
	}

	maxRawResults := r.getMaxRawResults()
	maxAggregatedResults := r.getMaxAggregatedResults()

	// Check if query uses aggregation
	usesAggregation := !query.GroupBy.IsEmpty() || query.Analysis != nil

	if usesAggregation {
		if query.Limit > maxAggregatedResults {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Aggregated result limit %d exceeds maximum %d",
					query.Limit, maxAggregatedResults))
		}
	} else {
		if query.Limit > maxRawResults {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Raw result limit %d exceeds maximum %d",
					query.Limit, maxRawResults))
		}
	}

	result.Details["uses_aggregation"] = usesAggregation
	result.Details["effective_limit"] = query.Limit
}

// validateConcurrencyLimits validates concurrent operation limits
func (r *PerformanceRule) validateConcurrencyLimits(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	concurrentSources := 1 // Primary source

	if query.MultiSource != nil {
		concurrentSources += len(query.MultiSource.SecondarySources)
	}

	maxConcurrentSources := r.getMaxConcurrentSources()
	if concurrentSources > maxConcurrentSources {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Concurrent sources %d exceeds limit %d",
				concurrentSources, maxConcurrentSources))
	}

	result.Details["concurrent_sources"] = concurrentSources
}

// addPerformanceRecommendations adds performance optimization recommendations
func (r *PerformanceRule) addPerformanceRecommendations(query *types.StructuredQuery, complexity int, result *interfaces.ValidationResult) {
	tier := r.getPerformanceTier(complexity)

	switch tier {
	case "high":
		result.Recommendations = append(result.Recommendations,
			"Consider breaking down complex queries into simpler parts",
			"Use more specific time ranges to reduce data volume",
			"Limit result set size for initial analysis",
			"Consider running during off-peak hours")
	case "medium":
		result.Recommendations = append(result.Recommendations,
			"Monitor query execution time",
			"Consider caching results for repeated queries")
	case "low":
		result.Recommendations = append(result.Recommendations,
			"Query should execute efficiently")
	}

	// Specific recommendations based on query characteristics
	if query.MultiSource != nil && len(query.MultiSource.SecondarySources) > 2 {
		result.Recommendations = append(result.Recommendations,
			"Consider reducing number of correlated sources for better performance")
	}

	if query.Analysis != nil && query.Analysis.StatisticalAnalysis != nil {
		result.Recommendations = append(result.Recommendations,
			"Statistical analysis may benefit from larger baseline periods")
	}

	if query.BehavioralAnalysis != nil && query.BehavioralAnalysis.UserProfiling && query.BehavioralAnalysis.AnomalyDetection != nil {
		result.Recommendations = append(result.Recommendations,
			"Combined behavioral analysis features may impact performance")
	}
}

// Resource estimation methods
func (r *PerformanceRule) estimateMemoryUsage(complexity int, query *types.StructuredQuery) int {
	baseMemory := 50 // Base memory in MB
	complexityMemory := complexity * 2

	// Additional memory for specific features
	if query.Analysis != nil {
		complexityMemory += 100
	}

	if query.MultiSource != nil {
		complexityMemory += len(query.MultiSource.SecondarySources) * 50
	}

	if query.BehavioralAnalysis != nil {
		complexityMemory += 150
	}

	return baseMemory + complexityMemory
}

func (r *PerformanceRule) estimateCPUUsage(complexity int, query *types.StructuredQuery) int {
	baseCPU := 10 // Base CPU percentage
	complexityCPU := complexity / 3

	// Additional CPU for specific features
	if query.Analysis != nil {
		complexityCPU += 25
	}

	if query.MultiSource != nil {
		complexityCPU += 20
	}

	total := baseCPU + complexityCPU
	if total > 100 {
		total = 100
	}

	return total
}

func (r *PerformanceRule) estimateExecutionTime(complexity int, query *types.StructuredQuery) int {
	baseTime := 5 // Base time in seconds
	complexityTime := complexity / 10

	// Additional time for specific features
	if query.Analysis != nil && query.Analysis.StatisticalAnalysis != nil {
		complexityTime += 60
	}

	if query.MultiSource != nil {
		complexityTime += len(query.MultiSource.SecondarySources) * 15
	}

	return baseTime + complexityTime
}

// Configuration retrieval methods
func (r *PerformanceRule) getMaxComplexityScore() int {
	if r.config != nil {
		if maxScore, ok := r.config["max_query_complexity_score"].(int); ok {
			return maxScore
		}
	}
	return 100 // Default
}

func (r *PerformanceRule) getMaxMemoryUsageMB() int {
	if r.config != nil {
		if maxMemory, ok := r.config["max_memory_usage_mb"].(int); ok {
			return maxMemory
		}
	}
	return 1024 // Default 1GB
}

func (r *PerformanceRule) getMaxCPUUsagePercent() int {
	if r.config != nil {
		if maxCPU, ok := r.config["max_cpu_usage_percent"].(int); ok {
			return maxCPU
		}
	}
	return 50 // Default 50%
}

func (r *PerformanceRule) getMaxExecutionTimeSeconds() int {
	if r.config != nil {
		if maxTime, ok := r.config["max_execution_time_seconds"].(int); ok {
			return maxTime
		}
	}
	return 300 // Default 5 minutes
}

func (r *PerformanceRule) getMaxRawResults() int {
	if r.config != nil {
		if maxResults, ok := r.config["max_raw_results"].(int); ok {
			return maxResults
		}
	}
	return 10000 // Default
}

func (r *PerformanceRule) getMaxAggregatedResults() int {
	if r.config != nil {
		if maxResults, ok := r.config["max_aggregated_results"].(int); ok {
			return maxResults
		}
	}
	return 1000 // Default
}

func (r *PerformanceRule) getMaxConcurrentSources() int {
	if r.config != nil {
		if maxSources, ok := r.config["max_concurrent_sources"].(int); ok {
			return maxSources
		}
	}
	return 5 // Default
}

func (r *PerformanceRule) getPerformanceTier(complexity int) string {
	maxComplexity := r.getMaxComplexityScore()

	if complexity > maxComplexity*2/3 {
		return "high"
	} else if complexity > maxComplexity/3 {
		return "medium"
	}
	return "low"
}

// Interface implementation methods
func (r *PerformanceRule) GetRuleName() string {
	return "performance_validation"
}

func (r *PerformanceRule) GetRuleDescription() string {
	return "Validates query performance, complexity limits, and resource usage to ensure efficient execution"
}

func (r *PerformanceRule) IsEnabled() bool {
	return r.enabled
}

func (r *PerformanceRule) GetSeverity() string {
	return "warning" // Performance issues are warnings, not critical failures
}