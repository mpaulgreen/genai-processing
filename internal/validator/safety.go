package validator

import (
	"fmt"
	"time"

	"genai-processing/internal/parser/normalizers"
	"genai-processing/internal/validator/rules"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// SafetyValidator implements the SafetyValidator interface for validating
// generated queries for safety and feasibility with enhanced capabilities.
type SafetyValidator struct {
	config              *ValidationConfig
	schemaValidator     interfaces.SchemaValidator
	ruleEngine          *RuleEngine
	// Legacy rules for backward compatibility
	rules               []interfaces.ValidationRule
	sanitization        *rules.SanitizationRule
	patterns            *rules.PatternsRule
	requiredFields      *rules.RequiredFieldsRule
}

// NewSafetyValidator creates a new instance of SafetyValidator.
// This constructor initializes the validator with validation rules from configuration
// and integrates with the enhanced schema validator and rule engine.
func NewSafetyValidator() *SafetyValidator {
	config, err := LoadDefaultValidationConfig()
	if err != nil {
		// Fallback to default configuration if config file cannot be loaded
		config = &ValidationConfig{}
		config.SafetyRules.AllowedLogSources = []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"}
		config.SafetyRules.AllowedVerbs = []string{"get", "list", "create", "update", "patch", "delete", "watch"}
		config.SafetyRules.AllowedResources = []string{"pods", "services", "deployments", "configmaps", "secrets", "namespaces"}
		config.SafetyRules.ForbiddenPatterns = []string{"rm -rf", "delete --all", "system:admin", "cluster-admin"}

		// Add default sanitization config
		config.SafetyRules.Sanitization = map[string]interface{}{
			"max_pattern_length": 500,
			"max_query_length":   10000,
			"forbidden_chars":    []interface{}{"<", ">", "&", "\"", "'", "`", "|", ";", "$", "(", ")", "{", "}", "[", "]", "\\", "/", "!", "@", "#", "%", "^", "*", "+", "=", "~"},
		}


		// Add default required fields
		config.SafetyRules.RequiredFields = []string{"log_source"}
		
		// Apply defaults for rule engine
		config.ApplyDefaults()
	}

	validator := &SafetyValidator{
		config:          config,
		schemaValidator: normalizers.NewSchemaValidator(),
		ruleEngine:      NewRuleEngine(config),
	}

	// Initialize legacy validation rules for backward compatibility
	validator.initializeLegacyRules()

	return validator
}

// NewSafetyValidatorWithConfig creates a new instance of SafetyValidator with custom configuration.
func NewSafetyValidatorWithConfig(config *ValidationConfig) *SafetyValidator {
	validator := &SafetyValidator{
		config:          config,
		schemaValidator: normalizers.NewSchemaValidator(),
		ruleEngine:      NewRuleEngine(config),
	}

	// Initialize legacy validation rules for backward compatibility
	validator.initializeLegacyRules()

	return validator
}

// ValidateQuery validates a structured query for safety and feasibility.
// This enhanced implementation integrates schema validation with comprehensive safety rules:
// Phase 1: Schema validation (types, constraints, dependencies)
// Phase 2: Basic safety rules (whitelist, sanitization, patterns)
// Phase 3: Advanced safety rules (analysis, multi-source, behavioral)
// Phase 4: Performance and compliance validation
func (sv *SafetyValidator) ValidateQuery(query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	// Handle nil query
	if query == nil {
		return sv.createErrorResult("null_query_validation", "Query cannot be nil", []string{"Query is nil"}, nil), nil
	}

	// Initialize combined result
	combinedResult := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "enhanced_comprehensive_validation",
		Severity:        "info",
		Message:         "Query validation completed successfully",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Phase 1: Schema Validation (highest priority)
	schemaErr := sv.schemaValidator.ValidateSchema(query)
	if schemaErr != nil {
		return sv.convertSchemaErrorToValidationResult(schemaErr, query), nil
	}

	// Phase 2: Basic Safety Rules
	basicRuleResults := sv.applyBasicRules(query)
	
	// Check if any basic rules failed (fail fast for critical errors)
	for ruleName, result := range basicRuleResults {
		if !result.IsValid && result.Severity == "critical" {
			combinedResult.IsValid = false
			combinedResult.Severity = "critical"
			combinedResult.Message = fmt.Sprintf("Critical validation failure in %s", ruleName)
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
			combinedResult.Details["failed_rule"] = ruleName
			combinedResult.Details["rule_results"] = basicRuleResults
			return combinedResult, nil
		}
	}

	// Phase 3: Advanced Safety Rules via RuleEngine (if basic rules passed)
	advancedRuleResults := make(map[string]*interfaces.ValidationResult)
	if sv.ruleEngine != nil {
		ruleEngineResult, err := sv.ruleEngine.EvaluateRules(query)
		if err != nil {
			combinedResult.IsValid = false
			combinedResult.Severity = "critical"
			combinedResult.Message = fmt.Sprintf("Rule engine evaluation failed: %v", err)
			combinedResult.Errors = append(combinedResult.Errors, err.Error())
			return combinedResult, nil
		}

		// Extract individual rule results from the rule engine result
		if ruleResults, ok := ruleEngineResult.Details["rule_results"].(map[string]*interfaces.ValidationResult); ok {
			advancedRuleResults = ruleResults
		}
	}

	// Phase 4: Aggregate all results
	sv.aggregateResults(combinedResult, basicRuleResults, advancedRuleResults)

	return combinedResult, nil
}

// GetApplicableRules returns all validation rules that are currently active.
func (sv *SafetyValidator) GetApplicableRules() []interfaces.ValidationRule {
	var activeRules []interfaces.ValidationRule

	// Add core rules
	if sv.sanitization != nil && sv.sanitization.IsEnabled() {
		activeRules = append(activeRules, sv.sanitization)
	}
	if sv.patterns != nil && sv.patterns.IsEnabled() {
		activeRules = append(activeRules, sv.patterns)
	}
	if sv.requiredFields != nil && sv.requiredFields.IsEnabled() {
		activeRules = append(activeRules, sv.requiredFields)
	}

	// Add additional legacy rules
	for _, rule := range sv.rules {
		if rule.IsEnabled() {
			activeRules = append(activeRules, rule)
		}
	}

	// Add rules from the rule engine
	if sv.ruleEngine != nil {
		for _, rule := range sv.ruleEngine.rules {
			if rule.IsEnabled() {
				activeRules = append(activeRules, rule)
			}
		}
	}

	return activeRules
}

// initializeLegacyRules initializes legacy validation rules from configuration
func (sv *SafetyValidator) initializeLegacyRules() {
	// Initialize sanitization rule with enhanced configuration
	// The sanitization rule now includes timeframe validation and essential limits
	if sv.config.SafetyRules.Sanitization != nil {
		sv.sanitization = rules.NewSanitizationRule(sv.config.SafetyRules.Sanitization)
	} else {
		// Create with default enhanced sanitization config
		defaultConfig := map[string]interface{}{
			"max_pattern_length": 500,
			"max_query_length":   10000,
			"forbidden_chars":    []interface{}{"<", ">", "&", "\"", "'", "`", "|", ";", "$"},
			"max_result_limit":   50,
			"max_array_elements": 15,
			"max_days_back":      90,
			"allowed_timeframes": []interface{}{"today", "yesterday", "1_hour_ago", "6_hours_ago", "12_hours_ago", "1_day_ago", "3_days_ago", "7_days_ago", "30_days_ago", "90_days_ago"},
		}
		sv.sanitization = rules.NewSanitizationRule(defaultConfig)
	}

	// Initialize patterns rule
	if len(sv.config.SafetyRules.ForbiddenPatterns) > 0 {
		sv.patterns = rules.NewPatternsRule(sv.config.SafetyRules.ForbiddenPatterns)
	}

	// Initialize required fields rule
	if len(sv.config.SafetyRules.RequiredFields) > 0 {
		sv.requiredFields = rules.NewRequiredFieldsRule(sv.config.SafetyRules.RequiredFields)
	}

	// Initialize additional legacy rules based on configuration
	sv.initializeAdditionalLegacyRules()
}

// initializeAdditionalLegacyRules initializes additional legacy validation rules
func (sv *SafetyValidator) initializeAdditionalLegacyRules() {
	// This method can be extended to add more legacy validation rules
	// based on the configuration or specific requirements
}

// GetValidationStats returns statistics about validation operations
func (sv *SafetyValidator) GetValidationStats() map[string]interface{} {
	stats := make(map[string]interface{})

	activeRules := sv.GetApplicableRules()
	stats["total_active_rules"] = len(activeRules)
	stats["schema_validator_enabled"] = sv.schemaValidator != nil
	stats["sanitization_enabled"] = sv.sanitization != nil && sv.sanitization.IsEnabled()
	stats["patterns_enabled"] = sv.patterns != nil && sv.patterns.IsEnabled()
	stats["required_fields_enabled"] = sv.requiredFields != nil && sv.requiredFields.IsEnabled()
	stats["rule_engine_enabled"] = sv.ruleEngine != nil

	// Add rule engine statistics if available
	if sv.ruleEngine != nil {
		engineStats := sv.ruleEngine.GetEngineStats()
		for key, value := range engineStats {
			stats["engine_"+key] = value
		}
	}

	return stats
}

// createErrorResult creates a standardized error result
func (sv *SafetyValidator) createErrorResult(ruleName, message string, errors []string, query *types.StructuredQuery) *interfaces.ValidationResult {
	return &interfaces.ValidationResult{
		IsValid:         false,
		RuleName:        ruleName,
		Severity:        "critical",
		Message:         message,
		Details:         map[string]interface{}{
			"validation_timestamp": time.Now().Format(time.RFC3339),
			"rule_results":         map[string]*interfaces.ValidationResult{}, // Empty rule results for error cases
		},
		Recommendations: []string{"Review and fix the validation errors"},
		Warnings:        []string{},
		Errors:          errors,
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}
}

// convertSchemaErrorToValidationResult converts schema validation errors to ValidationResult
func (sv *SafetyValidator) convertSchemaErrorToValidationResult(err error, query *types.StructuredQuery) *interfaces.ValidationResult {
	return &interfaces.ValidationResult{
		IsValid:         false,
		RuleName:        "schema_validation",
		Severity:        "critical",
		Message:         "Schema validation failed",
		Details:         map[string]interface{}{
			"schema_error":         err.Error(),
			"validation_timestamp": time.Now().Format(time.RFC3339),
			"rule_results":         map[string]*interfaces.ValidationResult{}, // Empty rule results for schema errors
		},
		Recommendations: []string{"Fix schema validation errors", "Ensure all required fields are present and valid"},
		Warnings:        []string{},
		Errors:          []string{err.Error()},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}
}

// applyBasicRules applies basic safety validation rules
func (sv *SafetyValidator) applyBasicRules(query *types.StructuredQuery) map[string]*interfaces.ValidationResult {
	ruleResults := make(map[string]*interfaces.ValidationResult)

	// Apply required fields validation first
	if sv.requiredFields != nil && sv.requiredFields.IsEnabled() {
		ruleResults["required_fields"] = sv.requiredFields.Validate(query)
	}

	// Apply enhanced sanitization validation (includes timeframe and essential limits)
	if sv.sanitization != nil && sv.sanitization.IsEnabled() {
		ruleResults["sanitization"] = sv.sanitization.Validate(query)
	}

	// Apply patterns validation
	if sv.patterns != nil && sv.patterns.IsEnabled() {
		ruleResults["patterns"] = sv.patterns.Validate(query)
	}

	// Apply additional basic rules
	for _, rule := range sv.rules {
		if rule.IsEnabled() {
			ruleResults[rule.GetRuleName()] = rule.Validate(query)
		}
	}

	return ruleResults
}


// aggregateResults combines results from all validation phases
func (sv *SafetyValidator) aggregateResults(combinedResult *interfaces.ValidationResult, basicResults, advancedResults map[string]*interfaces.ValidationResult) {
	allResults := make(map[string]*interfaces.ValidationResult)
	
	// Merge basic and advanced results
	for k, v := range basicResults {
		allResults[k] = v
	}
	for k, v := range advancedResults {
		allResults[k] = v
	}

	// Aggregate errors, warnings, and recommendations
	for _, result := range allResults {
		if !result.IsValid {
			combinedResult.IsValid = false
			combinedResult.Errors = append(combinedResult.Errors, result.Errors...)
		}
		combinedResult.Warnings = append(combinedResult.Warnings, result.Warnings...)
		combinedResult.Recommendations = append(combinedResult.Recommendations, result.Recommendations...)
	}

	// Update severity and message
	if !combinedResult.IsValid {
		combinedResult.Severity = "critical"
		combinedResult.Message = "Query validation failed"
	} else if len(combinedResult.Warnings) > 0 {
		combinedResult.Severity = "warning"
		combinedResult.Message = "Query validation passed with warnings"
	}

	// Add detailed results
	combinedResult.Details["rule_results"] = allResults
	combinedResult.Details["total_rules_applied"] = len(allResults)
	combinedResult.Details["basic_rules_count"] = len(basicResults)
	combinedResult.Details["advanced_rules_count"] = len(advancedResults)
	combinedResult.Details["validation_timestamp"] = combinedResult.Timestamp
}
