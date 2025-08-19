package rules

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// RequiredFieldsRule implements validation for required fields
type RequiredFieldsRule struct {
	requiredFields []string
	enabled        bool
}

// NewRequiredFieldsRule creates a new required fields validation rule
func NewRequiredFieldsRule(requiredFields []string) *RequiredFieldsRule {
	return &RequiredFieldsRule{
		requiredFields: requiredFields,
		enabled:        true,
	}
}

// Validate applies required fields validation to the query
func (r *RequiredFieldsRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "required_fields_validation",
		Severity:        "critical",
		Message:         "Required fields validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		QuerySnapshot:   query,
	}

	// Handle nil query
	if query == nil {
		result.IsValid = false
		result.Message = "Required fields validation failed"
		result.Errors = append(result.Errors, "Query cannot be nil")
		return result
	}

	// Check each required field
	for _, field := range r.requiredFields {
		if !r.isFieldPresent(query, field) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Required field '%s' is missing or empty", field))
		}
	}

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Required fields validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Provide all required fields for the query",
			"Check the configuration for the list of required fields")
	}

	return result
}

// GetRuleName returns the rule name
func (r *RequiredFieldsRule) GetRuleName() string {
	return "required_fields_validation"
}

// GetRuleDescription returns the rule description
func (r *RequiredFieldsRule) GetRuleDescription() string {
	return "Validates that all required fields are present and non-empty"
}

// IsEnabled indicates if the rule is enabled
func (r *RequiredFieldsRule) IsEnabled() bool {
	return r.enabled
}

// GetSeverity returns the rule severity
func (r *RequiredFieldsRule) GetSeverity() string {
	return "critical"
}

// Helper methods
func (r *RequiredFieldsRule) isFieldPresent(query *types.StructuredQuery, field string) bool {
	switch field {
	case "log_source":
		return strings.TrimSpace(query.LogSource) != ""
	case "verb":
		return !query.Verb.IsEmpty()
	case "resource":
		return !query.Resource.IsEmpty()
	case "namespace":
		return !query.Namespace.IsEmpty()
	case "user":
		return !query.User.IsEmpty()
	case "timeframe":
		return query.Timeframe != ""
	case "limit":
		return query.Limit > 0
	case "response_status":
		return !query.ResponseStatus.IsEmpty()
	case "source_ip":
		return !query.SourceIP.IsEmpty()
	case "group_by":
		return !query.GroupBy.IsEmpty()
	case "sort_by":
		return query.SortBy != ""
	case "sort_order":
		return query.SortOrder != ""
	case "subresource":
		return query.Subresource != ""
	case "auth_decision":
		return query.AuthDecision != ""
	case "resource_name_pattern":
		return query.ResourceNamePattern != ""
	case "user_pattern":
		return query.UserPattern != ""
	case "namespace_pattern":
		return query.NamespacePattern != ""
	case "request_uri_pattern":
		return query.RequestURIPattern != ""
	case "authorization_reason_pattern":
		return query.AuthorizationReasonPattern != ""
	case "response_message_pattern":
		return query.ResponseMessagePattern != ""
	case "missing_annotation":
		return query.MissingAnnotation != ""
	case "request_object_filter":
		return query.RequestObjectFilter != ""
	default:
		return false
	}
}
