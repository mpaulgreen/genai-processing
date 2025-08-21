package rules

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// AdvancedAnalysisRule implements validation for advanced analysis configuration
type AdvancedAnalysisRule struct {
	config            map[string]interface{}
	timeWindowsConfig map[string]interface{}
	enabled           bool
}

// NewAdvancedAnalysisRule creates a new advanced analysis validation rule
func NewAdvancedAnalysisRule(config map[string]interface{}, timeWindowsConfig map[string]interface{}) *AdvancedAnalysisRule {
	return &AdvancedAnalysisRule{
		config:            config,
		timeWindowsConfig: timeWindowsConfig,
		enabled:           true,
	}
}

// Validate applies advanced analysis validation to the query
func (r *AdvancedAnalysisRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "advanced_analysis_validation",
		Severity:        "info",
		Message:         "Advanced analysis validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Skip validation if no analysis configuration
	if query.Analysis == nil {
		return result
	}

	// Validate analysis configuration
	r.validateAnalysisType(query.Analysis, result)
	r.validateKillChainPhase(query.Analysis, result)
	r.validateStatisticalAnalysis(query.Analysis, result)
	r.validateThresholds(query.Analysis, result)
	r.validateTimeWindows(query.Analysis, result)
	r.validateGroupingAndSorting(query.Analysis, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Advanced analysis validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Review advanced analysis configuration",
			"Ensure all required fields are present for the analysis type",
			"Verify statistical analysis parameters are within valid ranges",
			"Check kill chain phase requirements for APT analysis types")
	}

	return result
}

// validateAnalysisType validates the analysis type and its requirements
func (r *AdvancedAnalysisRule) validateAnalysisType(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	if analysis.Type == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Analysis type is required")
		return
	}

	// Get allowed analysis types from config
	allowedTypes := r.getAllowedAnalysisTypes()
	if !r.isValueInSlice(analysis.Type, allowedTypes) {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Invalid analysis type '%s'. Allowed types: %s", 
				analysis.Type, strings.Join(allowedTypes, ", ")))
		return
	}

	// Check APT analysis type requirements
	aptTypes := []string{
		"apt_reconnaissance_detection",
		"apt_lateral_movement_detection", 
		"apt_data_exfiltration_detection",
		"privilege_escalation_detection",
		"persistence_mechanism_detection",
		"defense_evasion_detection",
		"credential_harvesting_detection",
		"supply_chain_attack_detection",
		"living_off_the_land_detection",
		"c2_communication_detection",
	}

	if r.isValueInSlice(analysis.Type, aptTypes) {
		if analysis.KillChainPhase == "" {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Kill chain phase is required for APT analysis type '%s'", analysis.Type))
		}
	}

	// Check statistical analysis requirements
	statTypes := []string{
		"statistical_analysis",
		"anomaly_detection",
		"behavioral_analysis",
		"correlation_analysis",
		"temporal_pattern_analysis",
	}

	if r.isValueInSlice(analysis.Type, statTypes) {
		if analysis.StatisticalAnalysis == nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Statistical analysis parameters recommended for analysis type '%s'", analysis.Type))
		}
	}
}

// validateKillChainPhase validates kill chain phase configuration
func (r *AdvancedAnalysisRule) validateKillChainPhase(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	if analysis.KillChainPhase == "" {
		return // Optional field
	}

	allowedPhases := []string{
		"reconnaissance",
		"weaponization", 
		"delivery",
		"exploitation",
		"installation",
		"command_control",
		"actions_objectives",
		"initial_access",
		"execution",
		"persistence",
		"privilege_escalation",
		"defense_evasion",
		"credential_access",
		"discovery",
		"lateral_movement",
		"collection",
		"command_and_control",
		"exfiltration",
		"impact",
	}

	if !r.isValueInSlice(analysis.KillChainPhase, allowedPhases) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid kill chain phase '%s'. Allowed phases: %s",
				analysis.KillChainPhase, strings.Join(allowedPhases, ", ")))
	}
}

// validateStatisticalAnalysis validates statistical analysis parameters
func (r *AdvancedAnalysisRule) validateStatisticalAnalysis(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	if analysis.StatisticalAnalysis == nil {
		return // Optional field
	}

	stats := analysis.StatisticalAnalysis

	// Validate pattern deviation threshold (hardcoded data science constants)
	if stats.PatternDeviationThreshold != 0 {
		if stats.PatternDeviationThreshold < 0.1 || stats.PatternDeviationThreshold > 10.0 {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Pattern deviation threshold must be between 0.1 and 10.0, got %.2f",
					stats.PatternDeviationThreshold))
		}
	}

	// Validate confidence interval (hardcoded data science constants)
	if stats.ConfidenceInterval != 0 {
		if stats.ConfidenceInterval < 0.5 || stats.ConfidenceInterval > 0.99 {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Confidence interval must be between 0.5 and 0.99, got %.2f",
					stats.ConfidenceInterval))
		}
	}

	// Validate sample size minimum (hardcoded data science constants)
	if stats.SampleSizeMinimum != 0 {
		if stats.SampleSizeMinimum < 10 {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Sample size minimum must be at least 10, got %d",
					stats.SampleSizeMinimum))
		}
	}

	// Validate baseline window
	if stats.BaselineWindow != "" {
		allowedWindows := []string{
			"7_days", "14_days", "30_days", "60_days", "90_days",
		}
		if !r.isValueInSlice(stats.BaselineWindow, allowedWindows) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid baseline window '%s'. Allowed windows: %s",
					stats.BaselineWindow, strings.Join(allowedWindows, ", ")))
		}
	}
}

// validateThresholds validates threshold and numeric parameters
func (r *AdvancedAnalysisRule) validateThresholds(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	// Validate threshold
	if analysis.Threshold != 0 {
		maxThreshold := r.getMaxThresholdValue()
		minThreshold := r.getMinThresholdValue()
		
		if analysis.Threshold < minThreshold || analysis.Threshold > maxThreshold {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Threshold must be between %d and %d, got %d",
					minThreshold, maxThreshold, analysis.Threshold))
		}
	}
}

// validateTimeWindows validates time window configuration
func (r *AdvancedAnalysisRule) validateTimeWindows(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	if analysis.TimeWindow == "" {
		return // Optional field
	}

	allowedTimeWindows := r.getAllowedTimeWindows()
	if !r.isValueInSlice(analysis.TimeWindow, allowedTimeWindows) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid time window '%s'. Allowed windows: %s",
				analysis.TimeWindow, strings.Join(allowedTimeWindows, ", ")))
	}
}

// validateGroupingAndSorting validates grouping and sorting configuration
func (r *AdvancedAnalysisRule) validateGroupingAndSorting(analysis *types.AdvancedAnalysisConfig, result *interfaces.ValidationResult) {
	// Validate sort by field
	if analysis.SortBy != "" {
		allowedSortFields := r.getAllowedSortFields()
		if !r.isValueInSlice(analysis.SortBy, allowedSortFields) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid sort field '%s'. Allowed fields: %s",
					analysis.SortBy, strings.Join(allowedSortFields, ", ")))
		}
	}

	// Validate sort order
	if analysis.SortOrder != "" {
		allowedSortOrders := r.getAllowedSortOrders()
		if !r.isValueInSlice(analysis.SortOrder, allowedSortOrders) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid sort order '%s'. Allowed orders: %s",
					analysis.SortOrder, strings.Join(allowedSortOrders, ", ")))
		}
	}

	// Validate group by fields
	if analysis.GroupBy != nil && !analysis.GroupBy.IsEmpty() {
		maxGroupByFields := r.getMaxGroupByFields()
		
		if analysis.GroupBy.IsArray() {
			fields := analysis.GroupBy.GetArray()
			if len(fields) > maxGroupByFields {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Too many group by fields. Maximum allowed: %d, got: %d",
						maxGroupByFields, len(fields)))
			}
		}
	}
}

// Helper methods for configuration retrieval
func (r *AdvancedAnalysisRule) getAllowedAnalysisTypes() []string {
	if r.config == nil {
		return r.getDefaultAnalysisTypes()
	}

	if allowedTypes, ok := r.config["allowed_analysis_types"].([]interface{}); ok {
		types := make([]string, len(allowedTypes))
		for i, t := range allowedTypes {
			if str, ok := t.(string); ok {
				types[i] = str
			}
		}
		return types
	}

	return r.getDefaultAnalysisTypes()
}

func (r *AdvancedAnalysisRule) getDefaultAnalysisTypes() []string {
	// Return empty list to force dependency on configuration file
	// All analysis types are now defined in configs/rules.yaml as single source of truth
	return []string{}
}

func (r *AdvancedAnalysisRule) getMaxThresholdValue() int {
	if r.config != nil {
		if maxVal, ok := r.config["max_threshold_value"].(int); ok {
			return maxVal
		}
	}
	return 10 // Default fallback - should be configured in rules.yaml
}

func (r *AdvancedAnalysisRule) getMinThresholdValue() int {
	// Min threshold is hardcoded as a business rule - always 1
	return 1
}

func (r *AdvancedAnalysisRule) getMaxGroupByFields() int {
	if r.config != nil {
		if maxFields, ok := r.config["max_group_by_fields"].(int); ok {
			return maxFields
		}
	}
	return 5 // Default
}

func (r *AdvancedAnalysisRule) getAllowedTimeWindows() []string {
	// Use time_windows.allowed_time_windows as single source of truth
	if r.timeWindowsConfig != nil {
		if allowedWindows, ok := r.timeWindowsConfig["allowed_time_windows"].([]interface{}); ok {
			windows := make([]string, len(allowedWindows))
			for i, w := range allowedWindows {
				if str, ok := w.(string); ok {
					windows[i] = str
				}
			}
			return windows
		}
	}

	return r.getDefaultTimeWindows()
}

func (r *AdvancedAnalysisRule) getDefaultTimeWindows() []string {
	// Return empty list to force dependency on configuration file
	// All time windows are now defined in configs/rules.yaml as single source of truth
	return []string{}
}

// getAllowedSortFields returns allowed sort fields from configuration
func (r *AdvancedAnalysisRule) getAllowedSortFields() []string {
	if r.config == nil {
		return r.getDefaultSortFields()
	}
	if sortConfig, ok := r.config["sort_configuration"].(map[string]interface{}); ok {
		if allowedFields, ok := sortConfig["allowed_sort_fields"].([]interface{}); ok {
			fields := make([]string, len(allowedFields))
			for i, f := range allowedFields {
				if str, ok := f.(string); ok {
					fields[i] = str
				}
			}
			return fields
		}
	}
	return r.getDefaultSortFields()
}

// getAllowedSortOrders returns allowed sort orders from configuration
func (r *AdvancedAnalysisRule) getAllowedSortOrders() []string {
	if r.config == nil {
		return r.getDefaultSortOrders()
	}
	if sortConfig, ok := r.config["sort_configuration"].(map[string]interface{}); ok {
		if allowedOrders, ok := sortConfig["allowed_sort_orders"].([]interface{}); ok {
			orders := make([]string, len(allowedOrders))
			for i, o := range allowedOrders {
				if str, ok := o.(string); ok {
					orders[i] = str
				}
			}
			return orders
		}
	}
	return r.getDefaultSortOrders()
}

// getDefaultSortFields returns default sort fields when no configuration is available
func (r *AdvancedAnalysisRule) getDefaultSortFields() []string {
	// Return empty list to force dependency on configuration file
	// All sort fields are now defined in configs/rules.yaml as single source of truth
	return []string{}
}

// getDefaultSortOrders returns default sort orders when no configuration is available
func (r *AdvancedAnalysisRule) getDefaultSortOrders() []string {
	// Return empty list to force dependency on configuration file
	// All sort orders are now defined in configs/rules.yaml as single source of truth
	return []string{}
}

// Utility methods
func (r *AdvancedAnalysisRule) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Interface implementation methods
func (r *AdvancedAnalysisRule) GetRuleName() string {
	return "advanced_analysis_validation"
}

func (r *AdvancedAnalysisRule) GetRuleDescription() string {
	return "Validates advanced analysis configuration including APT detection, kill chain phases, and statistical analysis parameters"
}

func (r *AdvancedAnalysisRule) IsEnabled() bool {
	return r.enabled
}

func (r *AdvancedAnalysisRule) GetSeverity() string {
	return "critical"
}