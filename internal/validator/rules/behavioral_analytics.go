package rules

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// BehavioralAnalyticsRule implements validation for behavioral analytics configuration
type BehavioralAnalyticsRule struct {
	config  map[string]interface{}
	enabled bool
}

// NewBehavioralAnalyticsRule creates a new behavioral analytics validation rule
func NewBehavioralAnalyticsRule(config map[string]interface{}) *BehavioralAnalyticsRule {
	return &BehavioralAnalyticsRule{
		config:  config,
		enabled: true,
	}
}

// Validate applies behavioral analytics validation to the query
func (r *BehavioralAnalyticsRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "behavioral_analytics_validation",
		Severity:        "info",
		Message:         "Behavioral analytics validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Skip validation if no behavioral analysis configuration
	if query.BehavioralAnalysis == nil {
		return result
	}

	// Validate behavioral analysis configuration
	r.validateUserProfiling(query.BehavioralAnalysis, result)
	r.validateBaselineConfiguration(query.BehavioralAnalysis, result)
	r.validateRiskScoring(query.BehavioralAnalysis, result)
	r.validateAnomalyDetection(query.BehavioralAnalysis, result)
	r.validateDependencies(query.BehavioralAnalysis, result)
	r.validatePerformanceImpact(query.BehavioralAnalysis, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Behavioral analytics validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Review behavioral analytics configuration",
			"Ensure user profiling is enabled when using risk scoring",
			"Verify baseline window is specified for anomaly detection",
			"Check risk scoring parameters are within valid ranges",
			"Validate anomaly detection algorithm parameters")
	} else if len(result.Warnings) > 0 {
		result.Severity = "warning"
		result.Message = "Behavioral analytics validation passed with warnings"
	}

	return result
}

// validateUserProfiling validates user profiling configuration
func (r *BehavioralAnalyticsRule) validateUserProfiling(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	// User profiling is optional but recommended for behavioral analysis
	if !config.UserProfiling {
		result.Warnings = append(result.Warnings, 
			"User profiling is disabled. Consider enabling for better behavioral insights")
	}

	// If user profiling is enabled, validate baseline requirements
	if config.UserProfiling {
		if config.BaselineWindow == "" {
			result.Warnings = append(result.Warnings,
				"Baseline window not specified. Default baseline period will be used")
		}

		if config.LearningPeriod == "" {
			result.Warnings = append(result.Warnings,
				"Learning period not specified. Default learning period will be used")
		}
	}
}

// validateBaselineConfiguration validates baseline and learning period configuration
func (r *BehavioralAnalyticsRule) validateBaselineConfiguration(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	// Validate baseline window
	if config.BaselineWindow != "" {
		allowedWindows := r.getAllowedBaselineWindows()
		if !r.isValueInSlice(config.BaselineWindow, allowedWindows) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid baseline window '%s'. Allowed windows: %s",
					config.BaselineWindow, strings.Join(allowedWindows, ", ")))
		}

		// Validate baseline window against configuration limits
		if err := r.validateBaselineWindowLimits(config.BaselineWindow, result); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	// Validate learning period
	if config.LearningPeriod != "" {
		allowedPeriods := r.getAllowedLearningPeriods()
		if !r.isValueInSlice(config.LearningPeriod, allowedPeriods) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid learning period '%s'. Allowed periods: %s",
					config.LearningPeriod, strings.Join(allowedPeriods, ", ")))
		}
	}
}

// validateRiskScoring validates risk scoring configuration
func (r *BehavioralAnalyticsRule) validateRiskScoring(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	if config.RiskScoring == nil {
		return // Optional field
	}

	riskConfig := config.RiskScoring

	// Validate risk scoring algorithm
	if riskConfig.Algorithm != "" {
		allowedAlgorithms := []string{"weighted_sum", "composite", "ml_based"}
		if !r.isValueInSlice(riskConfig.Algorithm, allowedAlgorithms) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid risk scoring algorithm '%s'. Allowed algorithms: %s",
					riskConfig.Algorithm, strings.Join(allowedAlgorithms, ", ")))
		}
	}

	// Validate risk factors
	if len(riskConfig.RiskFactors) > 0 {
		allowedFactors := r.getAllowedRiskFactors()
		maxFactors := r.getMaxRiskFactors()

		if len(riskConfig.RiskFactors) > maxFactors {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Too many risk factors. Maximum allowed: %d, got: %d",
					maxFactors, len(riskConfig.RiskFactors)))
		}

		// Validate each risk factor
		for i, factor := range riskConfig.RiskFactors {
			if !r.isValueInSlice(factor, allowedFactors) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Invalid risk factor '%s' at index %d. Allowed factors: %s",
						factor, i, strings.Join(allowedFactors, ", ")))
			}
		}
	}

	// Validate weighting scheme
	if riskConfig.WeightingScheme != nil {
		if err := r.validateWeightingScheme(riskConfig.WeightingScheme, result); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}
}

// validateAnomalyDetection validates anomaly detection configuration
func (r *BehavioralAnalyticsRule) validateAnomalyDetection(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	if config.AnomalyDetection == nil {
		return // Optional field
	}

	anomalyConfig := config.AnomalyDetection

	// Validate anomaly detection algorithm
	if anomalyConfig.Algorithm != "" {
		allowedAlgorithms := []string{"isolation_forest", "z_score", "statistical", "threshold_based"}
		if !r.isValueInSlice(anomalyConfig.Algorithm, allowedAlgorithms) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid anomaly detection algorithm '%s'. Allowed algorithms: %s",
					anomalyConfig.Algorithm, strings.Join(allowedAlgorithms, ", ")))
		}
	}

	// Validate contamination parameter
	if anomalyConfig.Contamination < 0.0 || anomalyConfig.Contamination > 1.0 {
		if anomalyConfig.Contamination != 0.0 { // Allow 0.0 as unset value
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Contamination must be between 0.0 and 1.0, got %.3f",
					anomalyConfig.Contamination))
		}
	}

	// Validate sensitivity parameter
	if anomalyConfig.Sensitivity < 0.0 || anomalyConfig.Sensitivity > 1.0 {
		if anomalyConfig.Sensitivity != 0.0 { // Allow 0.0 as unset value
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Sensitivity must be between 0.0 and 1.0, got %.3f",
					anomalyConfig.Sensitivity))
		}
	}

	// Validate threshold parameter
	thresholdLimits := r.getAnomalyThresholdLimits()
	if anomalyConfig.Threshold != 0.0 { // Only validate if set
		if anomalyConfig.Threshold < thresholdLimits["min"].(float64) || 
		   anomalyConfig.Threshold > thresholdLimits["max"].(float64) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Anomaly threshold must be between %.1f and %.1f, got %.3f",
					thresholdLimits["min"].(float64), thresholdLimits["max"].(float64), anomalyConfig.Threshold))
		}
	}

	// Algorithm-specific validations
	r.validateAlgorithmSpecificParameters(anomalyConfig, result)
}

// validateDependencies validates dependencies between behavioral analysis components
func (r *BehavioralAnalyticsRule) validateDependencies(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	// Risk scoring requires user profiling
	if config.RiskScoring != nil && config.RiskScoring.Enabled && !config.UserProfiling {
		result.IsValid = false
		result.Errors = append(result.Errors,
			"Risk scoring requires user profiling to be enabled")
	}

	// Anomaly detection requires baseline comparison or user profiling
	if config.AnomalyDetection != nil && !config.BaselineComparison && !config.UserProfiling {
		result.Warnings = append(result.Warnings,
			"Anomaly detection works best with baseline comparison or user profiling enabled")
	}

	// Baseline comparison requires baseline window
	if config.BaselineComparison && config.BaselineWindow == "" {
		result.IsValid = false
		result.Errors = append(result.Errors,
			"Baseline comparison requires baseline_window to be specified")
	}

	// Check for logical configuration conflicts
	if config.RiskScoring != nil && config.AnomalyDetection != nil {
		// Both risk scoring and anomaly detection enabled - validate compatibility
		if config.RiskScoring.Algorithm == "ml_based" && config.AnomalyDetection.Algorithm == "isolation_forest" {
			result.Warnings = append(result.Warnings,
				"ML-based risk scoring with isolation forest may be computationally intensive")
		}
	}
}

// validatePerformanceImpact validates performance implications of behavioral analysis
func (r *BehavioralAnalyticsRule) validatePerformanceImpact(config *types.BehavioralAnalysisConfig, result *interfaces.ValidationResult) {
	performanceScore := r.calculatePerformanceImpact(config)
	maxPerformanceScore := r.getMaxPerformanceScore()

	if performanceScore > maxPerformanceScore {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High performance impact score %d. Consider simplifying behavioral analysis configuration",
				performanceScore))
	}

	// Add performance impact details
	result.Details["behavioral_performance_score"] = performanceScore
	result.Details["max_performance_score"] = maxPerformanceScore

	// Specific performance warnings
	if config.UserProfiling && config.BaselineComparison && config.RiskScoring != nil && config.AnomalyDetection != nil {
		result.Warnings = append(result.Warnings,
			"All behavioral analysis features enabled may impact query performance")
	}
}

// Helper validation methods
func (r *BehavioralAnalyticsRule) validateBaselineWindowLimits(window string, result *interfaces.ValidationResult) error {
	limits := r.getBaselineWindowLimits()
	
	windowDays := map[string]int{
		"7_days":  7,
		"14_days": 14,
		"30_days": 30,
		"60_days": 60,
		"90_days": 90,
	}

	if days, exists := windowDays[window]; exists {
		if days < limits["min_baseline_days"].(int) {
			return fmt.Errorf("baseline window too short. Minimum: %d days", limits["min_baseline_days"].(int))
		}
		if days > limits["max_baseline_days"].(int) {
			return fmt.Errorf("baseline window too long. Maximum: %d days", limits["max_baseline_days"].(int))
		}
	}

	return nil
}

func (r *BehavioralAnalyticsRule) validateWeightingScheme(scheme map[string]float64, result *interfaces.ValidationResult) error {
	if len(scheme) == 0 {
		return fmt.Errorf("weighting scheme cannot be empty")
	}

	totalWeight := 0.0
	for factor, weight := range scheme {
		if weight < 0.0 || weight > 1.0 {
			return fmt.Errorf("weight for factor '%s' must be between 0.0 and 1.0, got %.3f", factor, weight)
		}
		totalWeight += weight
	}

	// Allow some tolerance for floating point arithmetic
	if totalWeight < 0.95 || totalWeight > 1.05 {
		return fmt.Errorf("total weight must sum to approximately 1.0, got %.3f", totalWeight)
	}

	return nil
}

func (r *BehavioralAnalyticsRule) validateAlgorithmSpecificParameters(config *types.AnomalyDetectionConfig, result *interfaces.ValidationResult) {
	switch config.Algorithm {
	case "isolation_forest":
		// Isolation forest typically works well with contamination between 0.01 and 0.3
		if config.Contamination > 0.3 {
			result.Warnings = append(result.Warnings,
				"High contamination value for isolation forest may reduce detection accuracy")
		}
	case "z_score":
		// Z-score typically uses threshold values between 2.0 and 4.0
		if config.Threshold != 0.0 && (config.Threshold < 2.0 || config.Threshold > 4.0) {
			result.Warnings = append(result.Warnings,
				"Z-score threshold typically works best between 2.0 and 4.0")
		}
	case "statistical":
		// Statistical methods may require specific sensitivity settings
		if config.Sensitivity == 0.0 {
			result.Warnings = append(result.Warnings,
				"Statistical anomaly detection typically requires sensitivity to be specified")
		}
	}
}

func (r *BehavioralAnalyticsRule) calculatePerformanceImpact(config *types.BehavioralAnalysisConfig) int {
	score := 0

	// Base score for behavioral analysis
	score += 10

	if config.UserProfiling {
		score += 15
	}

	if config.BaselineComparison {
		score += 10
	}

	if config.RiskScoring != nil && config.RiskScoring.Enabled {
		score += 20
		if config.RiskScoring.Algorithm == "ml_based" {
			score += 15
		}
		score += len(config.RiskScoring.RiskFactors) * 2
	}

	if config.AnomalyDetection != nil {
		score += 25
		if config.AnomalyDetection.Algorithm == "isolation_forest" {
			score += 10
		} else if config.AnomalyDetection.Algorithm == "ml_based" {
			score += 20
		}
	}

	return score
}

// Configuration retrieval methods
func (r *BehavioralAnalyticsRule) getAllowedBaselineWindows() []string {
	return []string{"7_days", "14_days", "30_days", "60_days", "90_days"}
}

func (r *BehavioralAnalyticsRule) getAllowedLearningPeriods() []string {
	return []string{"1_day", "3_days", "7_days", "14_days", "30_days"}
}

func (r *BehavioralAnalyticsRule) getAllowedRiskFactors() []string {
	if r.config != nil {
		if factors, ok := r.config["allowed_risk_factors"].([]interface{}); ok {
			result := make([]string, len(factors))
			for i, f := range factors {
				if str, ok := f.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{
		"privilege_level", "resource_sensitivity", "timing_anomaly", "access_pattern", 
		"frequency_deviation", "location_anomaly", "user_agent_change", "authentication_method",
		"session_duration", "data_volume", "network_pattern", "command_pattern",
	}
}

func (r *BehavioralAnalyticsRule) getMaxRiskFactors() int {
	if r.config != nil {
		if maxFactors, ok := r.config["max_risk_factors"].(int); ok {
			return maxFactors
		}
	}
	return 10 // Default
}

func (r *BehavioralAnalyticsRule) getBaselineWindowLimits() map[string]interface{} {
	if r.config != nil {
		if limits, ok := r.config["baseline_window_limits"].(map[string]interface{}); ok {
			return limits
		}
	}

	return map[string]interface{}{
		"min_baseline_days": 7,
		"max_baseline_days": 90,
	}
}

func (r *BehavioralAnalyticsRule) getAnomalyThresholdLimits() map[string]interface{} {
	if r.config != nil {
		if limits, ok := r.config["anomaly_threshold_limits"].(map[string]interface{}); ok {
			return limits
		}
	}

	return map[string]interface{}{
		"min": 0.1,
		"max": 10.0,
	}
}

func (r *BehavioralAnalyticsRule) getMaxPerformanceScore() int {
	if r.config != nil {
		if maxScore, ok := r.config["max_performance_score"].(int); ok {
			return maxScore
		}
	}
	return 100 // Default
}

// Utility methods
func (r *BehavioralAnalyticsRule) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Interface implementation methods
func (r *BehavioralAnalyticsRule) GetRuleName() string {
	return "behavioral_analytics_validation"
}

func (r *BehavioralAnalyticsRule) GetRuleDescription() string {
	return "Validates behavioral analytics configuration including user profiling, risk scoring, and anomaly detection parameters"
}

func (r *BehavioralAnalyticsRule) IsEnabled() bool {
	return r.enabled
}

func (r *BehavioralAnalyticsRule) GetSeverity() string {
	return "critical"
}