package rules

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// MultiSourceRule implements validation for multi-source correlation configuration
type MultiSourceRule struct {
	config  map[string]interface{}
	enabled bool
}

// NewMultiSourceRule creates a new multi-source validation rule
func NewMultiSourceRule(config map[string]interface{}) *MultiSourceRule {
	return &MultiSourceRule{
		config:  config,
		enabled: true,
	}
}

// Validate applies multi-source correlation validation to the query
func (r *MultiSourceRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "multi_source_validation",
		Severity:        "info",
		Message:         "Multi-source validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Skip validation if no multi-source configuration
	if query.MultiSource == nil {
		return result
	}

	// Validate multi-source configuration
	r.validatePrimarySource(query.MultiSource, result)
	r.validateSecondarySources(query.MultiSource, result)
	r.validateSourceCompatibility(query.MultiSource, result)
	r.validateCorrelationWindow(query.MultiSource, result)
	r.validateCorrelationFields(query.MultiSource, result)
	r.validateJoinType(query.MultiSource, result)
	r.validateComplexityLimits(query.MultiSource, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Multi-source validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Review multi-source correlation configuration",
			"Ensure all log sources are valid and compatible",
			"Verify correlation fields are supported across all sources",
			"Check correlation window is within allowed limits",
			"Consider reducing query complexity for better performance")
	} else if len(result.Warnings) > 0 {
		result.Severity = "warning"
		result.Message = "Multi-source validation passed with warnings"
	}

	return result
}

// validatePrimarySource validates the primary log source
func (r *MultiSourceRule) validatePrimarySource(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	if config.PrimarySource == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Primary source is required for multi-source correlation")
		return
	}

	validSources := r.getValidLogSources()
	if !r.isValueInSlice(config.PrimarySource, validSources) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid primary source '%s'. Valid sources: %s",
				config.PrimarySource, strings.Join(validSources, ", ")))
	}
}

// validateSecondarySources validates secondary log sources
func (r *MultiSourceRule) validateSecondarySources(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	if len(config.SecondarySources) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "At least one secondary source is required for multi-source correlation")
		return
	}

	maxSources := r.getMaxSources()
	totalSources := 1 + len(config.SecondarySources) // Primary + secondary sources
	if totalSources > maxSources {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Too many sources for correlation. Maximum allowed: %d, got: %d",
				maxSources, totalSources))
	}

	validSources := r.getValidLogSources()
	seenSources := make(map[string]bool)
	seenSources[config.PrimarySource] = true

	for i, source := range config.SecondarySources {
		// Check if source is valid
		if !r.isValueInSlice(source, validSources) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid secondary source '%s' at index %d. Valid sources: %s",
					source, i, strings.Join(validSources, ", ")))
			continue
		}

		// Check for duplicates (including primary source)
		if seenSources[source] {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Duplicate source '%s' at index %d. Each source can only be used once", source, i))
			continue
		}
		seenSources[source] = true
	}
}

// validateSourceCompatibility validates compatibility between log sources
func (r *MultiSourceRule) validateSourceCompatibility(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	// Define incompatible source combinations
	incompatibleCombos := r.getIncompatibleSourceCombinations()
	
	allSources := append([]string{config.PrimarySource}, config.SecondarySources...)
	
	for _, combo := range incompatibleCombos {
		if r.containsAllSources(allSources, combo) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Source combination %v may have limited correlation capabilities",
					combo))
		}
	}

	// Check for performance warnings with specific combinations
	performanceWarningCombos := [][]string{
		{"node-auditd", "kube-apiserver", "oauth-server"}, // High volume combination
		{"kube-apiserver", "openshift-apiserver", "oauth-apiserver"}, // API heavy combination
	}

	for _, combo := range performanceWarningCombos {
		if r.containsAllSources(allSources, combo) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Source combination %v may impact query performance",
					combo))
		}
	}
}

// validateCorrelationWindow validates the correlation time window
func (r *MultiSourceRule) validateCorrelationWindow(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	if config.CorrelationWindow == "" {
		// Use default correlation window
		result.Warnings = append(result.Warnings, "No correlation window specified, using default")
		return
	}

	allowedWindows := r.getAllowedCorrelationWindows()
	if !r.isValueInSlice(config.CorrelationWindow, allowedWindows) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid correlation window '%s'. Allowed windows: %s",
				config.CorrelationWindow, strings.Join(allowedWindows, ", ")))
	}

	// Warn about performance implications for large windows
	largeWindows := []string{"12_hours", "24_hours"}
	if r.isValueInSlice(config.CorrelationWindow, largeWindows) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Large correlation window '%s' may impact query performance",
				config.CorrelationWindow))
	}
}

// validateCorrelationFields validates correlation fields
func (r *MultiSourceRule) validateCorrelationFields(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	if len(config.CorrelationFields) == 0 {
		result.Warnings = append(result.Warnings, "No correlation fields specified, using default correlation")
		return
	}

	maxFields := r.getMaxCorrelationFields()
	if len(config.CorrelationFields) > maxFields {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Too many correlation fields. Maximum allowed: %d, got: %d",
				maxFields, len(config.CorrelationFields)))
	}

	validFields := r.getAllowedCorrelationFields()
	seenFields := make(map[string]bool)

	for i, field := range config.CorrelationFields {
		// Check if field is valid
		if !r.isValueInSlice(field, validFields) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid correlation field '%s' at index %d. Valid fields: %s",
					field, i, strings.Join(validFields, ", ")))
			continue
		}

		// Check for duplicates
		if seenFields[field] {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Duplicate correlation field '%s' at index %d", field, i))
			continue
		}
		seenFields[field] = true
	}

	// Validate field compatibility with sources
	r.validateFieldSourceCompatibility(config, result)
}

// validateFieldSourceCompatibility checks if correlation fields are available in all sources
func (r *MultiSourceRule) validateFieldSourceCompatibility(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	allSources := append([]string{config.PrimarySource}, config.SecondarySources...)
	sourceFieldMap := r.getSourceFieldCompatibilityMap()

	for _, field := range config.CorrelationFields {
		incompatibleSources := []string{}
		
		for _, source := range allSources {
			if sourceFields, exists := sourceFieldMap[source]; exists {
				if !r.isValueInSlice(field, sourceFields) {
					incompatibleSources = append(incompatibleSources, source)
				}
			}
		}

		if len(incompatibleSources) > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Correlation field '%s' may not be available in sources: %s",
					field, strings.Join(incompatibleSources, ", ")))
		}
	}
}

// validateJoinType validates the join type for correlation
func (r *MultiSourceRule) validateJoinType(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	if config.JoinType == "" {
		// Default join type is acceptable
		return
	}

	allowedJoinTypes := []string{"inner", "left", "right", "full"}
	if !r.isValueInSlice(config.JoinType, allowedJoinTypes) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid join type '%s'. Allowed types: %s",
				config.JoinType, strings.Join(allowedJoinTypes, ", ")))
	}

	// Performance warnings for expensive joins
	expensiveJoins := []string{"full", "right"}
	if r.isValueInSlice(config.JoinType, expensiveJoins) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Join type '%s' may impact query performance", config.JoinType))
	}
}

// validateComplexityLimits validates overall correlation complexity
func (r *MultiSourceRule) validateComplexityLimits(config *types.MultiSourceConfig, result *interfaces.ValidationResult) {
	complexity := r.calculateCorrelationComplexity(config)
	maxComplexity := r.getMaxCorrelationComplexity()

	if complexity > maxComplexity {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Correlation complexity score %d exceeds maximum allowed %d",
				complexity, maxComplexity))
	} else if complexity > maxComplexity*3/4 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High correlation complexity score %d may impact performance", complexity))
	}

	// Add complexity details
	result.Details["correlation_complexity_score"] = complexity
	result.Details["max_complexity_allowed"] = maxComplexity
}

// calculateCorrelationComplexity calculates a complexity score for the correlation
func (r *MultiSourceRule) calculateCorrelationComplexity(config *types.MultiSourceConfig) int {
	complexity := 0

	// Base complexity for each source
	complexity += len(config.SecondarySources) * 10

	// Complexity for correlation fields
	complexity += len(config.CorrelationFields) * 5

	// Complexity based on correlation window
	windowComplexity := map[string]int{
		"1_minute":  1,
		"5_minutes": 2,
		"15_minutes": 3,
		"30_minutes": 4,
		"1_hour":    5,
		"2_hours":   7,
		"4_hours":   10,
		"6_hours":   12,
		"12_hours":  15,
		"24_hours":  20,
	}

	if windowComp, exists := windowComplexity[config.CorrelationWindow]; exists {
		complexity += windowComp
	} else {
		complexity += 5 // Default complexity
	}

	// Complexity based on join type
	joinComplexity := map[string]int{
		"inner": 1,
		"left":  2,
		"right": 3,
		"full":  5,
	}

	if joinComp, exists := joinComplexity[config.JoinType]; exists {
		complexity += joinComp
	} else {
		complexity += 2 // Default complexity
	}

	return complexity
}

// Helper methods for configuration retrieval
func (r *MultiSourceRule) getValidLogSources() []string {
	return []string{
		"kube-apiserver",
		"openshift-apiserver",
		"oauth-server",
		"oauth-apiserver",
		"node-auditd",
	}
}

func (r *MultiSourceRule) getMaxSources() int {
	if r.config != nil {
		if maxSources, ok := r.config["max_sources"].(int); ok {
			return maxSources
		}
	}
	return 5 // Default
}

func (r *MultiSourceRule) getAllowedCorrelationWindows() []string {
	if r.config != nil {
		if windows, ok := r.config["allowed_correlation_windows"].([]interface{}); ok {
			result := make([]string, len(windows))
			for i, w := range windows {
				if str, ok := w.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{
		"1_minute", "5_minutes", "10_minutes", "15_minutes", "30_minutes",
		"1_hour", "2_hours", "4_hours", "6_hours", "12_hours", "24_hours",
	}
}

func (r *MultiSourceRule) getMaxCorrelationFields() int {
	if r.config != nil {
		if maxFields, ok := r.config["max_correlation_fields"].(int); ok {
			return maxFields
		}
	}
	return 10 // Default
}

func (r *MultiSourceRule) getAllowedCorrelationFields() []string {
	if r.config != nil {
		if fields, ok := r.config["allowed_correlation_fields"].([]interface{}); ok {
			result := make([]string, len(fields))
			for i, f := range fields {
				if str, ok := f.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{
		"user", "source_ip", "user_agent", "session_id", "request_id",
		"timestamp", "namespace", "resource", "verb", "response_status",
	}
}

func (r *MultiSourceRule) getMaxCorrelationComplexity() int {
	if r.config != nil {
		if maxComplexity, ok := r.config["max_correlation_complexity"].(int); ok {
			return maxComplexity
		}
	}
	return 100 // Default
}

func (r *MultiSourceRule) getIncompatibleSourceCombinations() [][]string {
	return [][]string{
		// Add specific incompatible combinations as needed
		// For now, all sources are considered compatible but may have warnings
	}
}

func (r *MultiSourceRule) getSourceFieldCompatibilityMap() map[string][]string {
	return map[string][]string{
		"kube-apiserver": {
			"user", "source_ip", "user_agent", "timestamp", "namespace",
			"resource", "verb", "response_status", "request_id",
		},
		"openshift-apiserver": {
			"user", "source_ip", "user_agent", "timestamp", "namespace",
			"resource", "verb", "response_status", "request_id",
		},
		"oauth-server": {
			"user", "source_ip", "user_agent", "timestamp", "session_id",
			"response_status",
		},
		"oauth-apiserver": {
			"user", "source_ip", "user_agent", "timestamp", "namespace",
			"resource", "verb", "response_status", "session_id",
		},
		"node-auditd": {
			"user", "source_ip", "timestamp", // Limited fields for node audit
		},
	}
}

// Utility methods
func (r *MultiSourceRule) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func (r *MultiSourceRule) containsAllSources(sources []string, targetSources []string) bool {
	sourceMap := make(map[string]bool)
	for _, source := range sources {
		sourceMap[source] = true
	}

	for _, target := range targetSources {
		if !sourceMap[target] {
			return false
		}
	}
	return true
}

// Interface implementation methods
func (r *MultiSourceRule) GetRuleName() string {
	return "multi_source_validation"
}

func (r *MultiSourceRule) GetRuleDescription() string {
	return "Validates multi-source correlation configuration including source compatibility, correlation fields, and complexity limits"
}

func (r *MultiSourceRule) IsEnabled() bool {
	return r.enabled
}

func (r *MultiSourceRule) GetSeverity() string {
	return "critical"
}