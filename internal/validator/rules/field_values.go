package rules

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// FieldValuesRule implements validation for specific field values using configuration
type FieldValuesRule struct {
	config  map[string]interface{}
	enabled bool
}

// NewFieldValuesRule creates a new field values validation rule
func NewFieldValuesRule(config map[string]interface{}) *FieldValuesRule {
	return &FieldValuesRule{
		config:  config,
		enabled: true,
	}
}

// Validate applies field values validation to the query
func (r *FieldValuesRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "field_values_validation",
		Severity:        "info",
		Message:         "Field values validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Validate auth decision
	r.validateAuthDecision(query, result)

	// Note: Business hours validation not implemented as current type doesn't support presets
	// This can be added when BusinessHours type is updated to support preset values

	// Validate response status codes
	r.validateResponseStatus(query, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Field values validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Use only allowed values for enum fields",
			"Check auth_decision values against allowed list",
			"Verify business hours presets are supported",
			"Ensure response status codes are valid")
	}

	return result
}

// validateAuthDecision validates auth decision values
func (r *FieldValuesRule) validateAuthDecision(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if query.AuthDecision == "" {
		return // Optional field
	}

	allowedDecisions := r.getAllowedAuthDecisions()
	if !r.isValueInSlice(query.AuthDecision, allowedDecisions) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid auth_decision '%s'. Allowed decisions: %s",
				query.AuthDecision, strings.Join(allowedDecisions, ", ")))
	}
}

// validateBusinessHours validates business hours configuration (placeholder for future preset support)
func (r *FieldValuesRule) validateBusinessHours(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Note: Current BusinessHours type doesn't support preset validation
	// This method is a placeholder for when preset support is added to the BusinessHours type
	return
}

// validateResponseStatus validates response status codes
func (r *FieldValuesRule) validateResponseStatus(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if query.ResponseStatus.IsEmpty() {
		return // Optional field
	}

	allowedStatusCodes := r.getAllowedResponseStatusCodes()
	
	// Validate single value
	if !query.ResponseStatus.IsArray() {
		statusCode := query.ResponseStatus.GetString()
		if !r.isValueInSlice(statusCode, allowedStatusCodes) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid response_status '%s'. Allowed status codes: %s",
					statusCode, strings.Join(allowedStatusCodes, ", ")))
		}
	} else {
		// Validate array values
		statusCodes := query.ResponseStatus.GetArray()
		for _, statusCode := range statusCodes {
			if !r.isValueInSlice(statusCode, allowedStatusCodes) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Invalid response_status '%s' in array. Allowed status codes: %s",
						statusCode, strings.Join(allowedStatusCodes, ", ")))
			}
		}
	}
}

// Helper methods for configuration retrieval
func (r *FieldValuesRule) getAllowedAuthDecisions() []string {
	if r.config == nil {
		return r.getDefaultAuthDecisions()
	}

	if authConfig, ok := r.config["auth_decisions_configuration"].(map[string]interface{}); ok {
		if allowedDecisions, ok := authConfig["allowed_decisions"].([]interface{}); ok {
			decisions := make([]string, len(allowedDecisions))
			for i, d := range allowedDecisions {
				if str, ok := d.(string); ok {
					decisions[i] = str
				}
			}
			return decisions
		}
	}

	return r.getDefaultAuthDecisions()
}

func (r *FieldValuesRule) getAllowedBusinessHoursPresets() []string {
	if r.config == nil {
		return r.getDefaultBusinessHoursPresets()
	}

	if bhConfig, ok := r.config["business_hours_configuration"].(map[string]interface{}); ok {
		if allowedPresets, ok := bhConfig["allowed_presets"].([]interface{}); ok {
			presets := make([]string, len(allowedPresets))
			for i, p := range allowedPresets {
				if str, ok := p.(string); ok {
					presets[i] = str
				}
			}
			return presets
		}
	}

	return r.getDefaultBusinessHoursPresets()
}

func (r *FieldValuesRule) getAllowedResponseStatusCodes() []string {
	if r.config == nil {
		return r.getDefaultResponseStatusCodes()
	}

	if rsConfig, ok := r.config["response_status_configuration"].(map[string]interface{}); ok {
		if allowedCodes, ok := rsConfig["allowed_status_codes"].([]interface{}); ok {
			codes := make([]string, len(allowedCodes))
			for i, c := range allowedCodes {
				if str, ok := c.(string); ok {
					codes[i] = str
				}
			}
			return codes
		}
	}

	return r.getDefaultResponseStatusCodes()
}

// Default values (fallback when no configuration is provided)
func (r *FieldValuesRule) getDefaultAuthDecisions() []string {
	// Return standard auth decisions as fallback
	return []string{"allow", "error", "forbid"}
}

func (r *FieldValuesRule) getDefaultBusinessHoursPresets() []string {
	// Return standard business hours presets as fallback
	return []string{"business_hours", "outside_business_hours", "weekend", "all_hours"}
}

func (r *FieldValuesRule) getDefaultResponseStatusCodes() []string {
	// Return standard HTTP status codes as fallback
	return []string{"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504"}
}

// Utility methods
func (r *FieldValuesRule) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Interface implementation methods
func (r *FieldValuesRule) GetRuleName() string {
	return "field_values_validation"
}

func (r *FieldValuesRule) GetRuleDescription() string {
	return "Validates specific field values against allowed lists from configuration"
}

func (r *FieldValuesRule) IsEnabled() bool {
	return r.enabled
}

func (r *FieldValuesRule) GetSeverity() string {
	return "critical"
}