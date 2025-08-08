package validator

import (
	"time"

	"genai-processing/internal/validator/rules"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// SafetyValidator implements the SafetyValidator interface for validating
// generated queries for safety and feasibility.
type SafetyValidator struct {
	config         *ValidationConfig
	rules          []interfaces.ValidationRule
	whitelist      *rules.WhitelistRule
	sanitization   *rules.SanitizationRule
	timeframe      *rules.TimeframeRule
	patterns       *rules.PatternsRule
	requiredFields *rules.RequiredFieldsRule
}

// NewSafetyValidator creates a new instance of SafetyValidator.
// This constructor initializes the validator with validation rules from configuration.
func NewSafetyValidator() *SafetyValidator {
	config, err := LoadDefaultValidationConfig()
	if err != nil {
		// Fallback to default configuration if config file cannot be loaded
		config = &ValidationConfig{}
		config.SafetyRules.AllowedLogSources = []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver"}
		config.SafetyRules.AllowedVerbs = []string{"get", "list", "create", "update", "patch", "delete", "watch"}
		config.SafetyRules.AllowedResources = []string{"pods", "services", "deployments", "configmaps", "secrets", "namespaces"}
		config.SafetyRules.ForbiddenPatterns = []string{"rm -rf", "delete --all", "system:admin", "cluster-admin"}

		// Add default sanitization config
		config.SafetyRules.Sanitization = map[string]interface{}{
			"max_pattern_length": 500,
			"max_query_length":   10000,
			"forbidden_chars":    []interface{}{"<", ">", "&", "\"", "'", "`", "|", ";", "$", "(", ")", "{", "}", "[", "]", "\\", "/", "!", "@", "#", "%", "^", "*", "+", "=", "~"},
		}

		// Add default timeframe config
		config.SafetyRules.TimeframeLimits = map[string]interface{}{
			"max_days_back":      90,
			"default_limit":      20,
			"max_limit":          1000,
			"min_limit":          1,
			"allowed_timeframes": []interface{}{"today", "yesterday", "1_hour_ago", "2_hours_ago", "3_hours_ago", "6_hours_ago", "12_hours_ago", "1_day_ago", "2_days_ago", "3_days_ago", "7_days_ago", "14_days_ago", "30_days_ago", "60_days_ago", "90_days_ago"},
		}

		// Add default required fields
		config.SafetyRules.RequiredFields = []string{"log_source"}
	}

	validator := &SafetyValidator{
		config: config,
	}

	// Initialize validation rules
	validator.initializeRules()

	return validator
}

// NewSafetyValidatorWithConfig creates a new instance of SafetyValidator with custom configuration.
func NewSafetyValidatorWithConfig(config *ValidationConfig) *SafetyValidator {
	validator := &SafetyValidator{
		config: config,
	}

	// Initialize validation rules
	validator.initializeRules()

	return validator
}

// ValidateQuery validates a structured query for safety and feasibility.
// This implementation applies comprehensive validation rules including:
// - Whitelist validation for log sources, verbs, and resources
// - Input sanitization to prevent injection attacks
// - Timeframe limits and constraints
// - Forbidden patterns and commands
// - Query limits and business rules
func (sv *SafetyValidator) ValidateQuery(query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	// Handle nil query
	if query == nil {
		return &interfaces.ValidationResult{
			IsValid:  false,
			RuleName: "null_query_validation",
			Severity: "critical",
			Message:  "Query cannot be nil",
			Details: map[string]interface{}{
				"rule_results":         make(map[string]*interfaces.ValidationResult),
				"total_rules_applied":  0,
				"validation_timestamp": time.Now().Format(time.RFC3339),
			},
			Recommendations: []string{"Provide a valid structured query"},
			Warnings:        []string{},
			Errors:          []string{"Query is nil"},
			Timestamp:       time.Now().Format(time.RFC3339),
			QuerySnapshot:   query,
		}, nil
	}

	// Initialize combined result
	combinedResult := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "comprehensive_safety_validation",
		Severity:        "info",
		Message:         "Query validation completed successfully",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Apply all validation rules
	ruleResults := make(map[string]*interfaces.ValidationResult)

	// Apply whitelist validation
	if sv.whitelist != nil && sv.whitelist.IsEnabled() {
		result := sv.whitelist.Validate(query)
		ruleResults["whitelist"] = result
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Apply sanitization validation
	if sv.sanitization != nil && sv.sanitization.IsEnabled() {
		result := sv.sanitization.Validate(query)
		ruleResults["sanitization"] = result
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Apply timeframe validation
	if sv.timeframe != nil && sv.timeframe.IsEnabled() {
		result := sv.timeframe.Validate(query)
		ruleResults["timeframe"] = result
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Apply patterns validation
	if sv.patterns != nil && sv.patterns.IsEnabled() {
		result := sv.patterns.Validate(query)
		ruleResults["patterns"] = result
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Apply required fields validation
	if sv.requiredFields != nil && sv.requiredFields.IsEnabled() {
		result := sv.requiredFields.Validate(query)
		ruleResults["required_fields"] = result
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Apply additional validation rules
	for _, rule := range sv.rules {
		if rule.IsEnabled() {
			result := rule.Validate(query)
			ruleResults[rule.GetRuleName()] = result
			if !result.IsValid {
				combinedResult.IsValid = false
				combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
			}
			combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
			combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
		}
	}

	// Update severity based on validation results
	if !combinedResult.IsValid {
		combinedResult.Severity = "critical"
		combinedResult.Message = "Query validation failed"
	} else if len(combinedResult.Warnings) > 0 {
		combinedResult.Severity = "warning"
		combinedResult.Message = "Query validation passed with warnings"
	}

	// Add rule results to details
	combinedResult.Details["rule_results"] = ruleResults
	combinedResult.Details["total_rules_applied"] = len(ruleResults)
	combinedResult.Details["validation_timestamp"] = combinedResult.Timestamp

	return combinedResult, nil
}

// GetApplicableRules returns all validation rules that are currently active.
func (sv *SafetyValidator) GetApplicableRules() []interfaces.ValidationRule {
	var activeRules []interfaces.ValidationRule

	// Add core rules
	if sv.whitelist != nil && sv.whitelist.IsEnabled() {
		activeRules = append(activeRules, sv.whitelist)
	}
	if sv.sanitization != nil && sv.sanitization.IsEnabled() {
		activeRules = append(activeRules, sv.sanitization)
	}
	if sv.timeframe != nil && sv.timeframe.IsEnabled() {
		activeRules = append(activeRules, sv.timeframe)
	}
	if sv.patterns != nil && sv.patterns.IsEnabled() {
		activeRules = append(activeRules, sv.patterns)
	}
	if sv.requiredFields != nil && sv.requiredFields.IsEnabled() {
		activeRules = append(activeRules, sv.requiredFields)
	}

	// Add additional rules
	for _, rule := range sv.rules {
		if rule.IsEnabled() {
			activeRules = append(activeRules, rule)
		}
	}

	return activeRules
}

// initializeRules initializes all validation rules from configuration
func (sv *SafetyValidator) initializeRules() {
	// Initialize whitelist rule
	if len(sv.config.SafetyRules.AllowedLogSources) > 0 ||
		len(sv.config.SafetyRules.AllowedVerbs) > 0 ||
		len(sv.config.SafetyRules.AllowedResources) > 0 {
		sv.whitelist = rules.NewWhitelistRule(
			sv.config.SafetyRules.AllowedLogSources,
			sv.config.SafetyRules.AllowedVerbs,
			sv.config.SafetyRules.AllowedResources,
		)
	}

	// Initialize sanitization rule
	if sv.config.SafetyRules.Sanitization != nil {
		sv.sanitization = rules.NewSanitizationRule(sv.config.SafetyRules.Sanitization)
	}

	// Initialize timeframe rule
	if sv.config.SafetyRules.TimeframeLimits != nil {
		sv.timeframe = rules.NewTimeframeRule(sv.config.SafetyRules.TimeframeLimits)
	}

	// Initialize patterns rule
	if len(sv.config.SafetyRules.ForbiddenPatterns) > 0 {
		sv.patterns = rules.NewPatternsRule(sv.config.SafetyRules.ForbiddenPatterns)
	}

	// Initialize required fields rule
	if len(sv.config.SafetyRules.RequiredFields) > 0 {
		sv.requiredFields = rules.NewRequiredFieldsRule(sv.config.SafetyRules.RequiredFields)
	}

	// Initialize additional rules based on configuration
	sv.initializeAdditionalRules()
}

// initializeAdditionalRules initializes additional validation rules
func (sv *SafetyValidator) initializeAdditionalRules() {
	// This method can be extended to add more validation rules
	// based on the configuration or specific requirements
}

// GetValidationStats returns statistics about validation operations
func (sv *SafetyValidator) GetValidationStats() map[string]interface{} {
	stats := make(map[string]interface{})

	activeRules := sv.GetApplicableRules()
	stats["total_active_rules"] = len(activeRules)
	stats["whitelist_enabled"] = sv.whitelist != nil && sv.whitelist.IsEnabled()
	stats["sanitization_enabled"] = sv.sanitization != nil && sv.sanitization.IsEnabled()
	stats["timeframe_enabled"] = sv.timeframe != nil && sv.timeframe.IsEnabled()
	stats["patterns_enabled"] = sv.patterns != nil && sv.patterns.IsEnabled()
	stats["required_fields_enabled"] = sv.requiredFields != nil && sv.requiredFields.IsEnabled()

	return stats
}
