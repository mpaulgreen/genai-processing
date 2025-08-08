package rules

import (
	"fmt"
	"regexp"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// PatternsRule implements validation for forbidden patterns and commands
type PatternsRule struct {
	forbiddenPatterns []string
	enabled           bool
}

// NewPatternsRule creates a new patterns validation rule
func NewPatternsRule(forbiddenPatterns []string) *PatternsRule {
	return &PatternsRule{
		forbiddenPatterns: forbiddenPatterns,
		enabled:           true,
	}
}

// Validate applies forbidden patterns validation to the query
func (p *PatternsRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "forbidden_patterns_validation",
		Severity:        "critical",
		Message:         "Forbidden patterns validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		QuerySnapshot:   query,
	}

	// Check all string fields for forbidden patterns
	p.checkStringFields(query, result)

	// Check array fields for forbidden patterns
	p.checkArrayFields(query, result)

	// Check patterns in specific fields
	p.checkSpecificPatterns(query, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Forbidden patterns validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Remove forbidden patterns from query parameters",
			"Avoid dangerous command patterns and system access",
			"Use safe, non-privileged patterns only",
			"Review query for potential security risks")
	}

	return result
}

// GetRuleName returns the rule name
func (p *PatternsRule) GetRuleName() string {
	return "forbidden_patterns_validation"
}

// GetRuleDescription returns the rule description
func (p *PatternsRule) GetRuleDescription() string {
	return "Validates that query does not contain forbidden patterns or dangerous commands"
}

// IsEnabled indicates if the rule is enabled
func (p *PatternsRule) IsEnabled() bool {
	return p.enabled
}

// GetSeverity returns the rule severity
func (p *PatternsRule) GetSeverity() string {
	return "critical"
}

// Helper methods
func (p *PatternsRule) checkStringFields(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	stringFields := map[string]string{
		"log_source":                   query.LogSource,
		"timeframe":                    query.Timeframe,
		"resource_name_pattern":        query.ResourceNamePattern,
		"user_pattern":                 query.UserPattern,
		"namespace_pattern":            query.NamespacePattern,
		"request_uri_pattern":          query.RequestURIPattern,
		"auth_decision":                query.AuthDecision,
		"sort_by":                      query.SortBy,
		"sort_order":                   query.SortOrder,
		"subresource":                  query.Subresource,
		"request_object_filter":        query.RequestObjectFilter,
		"authorization_reason_pattern": query.AuthorizationReasonPattern,
		"response_message_pattern":     query.ResponseMessagePattern,
		"missing_annotation":           query.MissingAnnotation,
	}

	for fieldName, fieldValue := range stringFields {
		if fieldValue != "" {
			for _, pattern := range p.forbiddenPatterns {
				if p.matchesPattern(fieldValue, pattern) {
					result.IsValid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Field '%s' contains forbidden pattern '%s': %s", fieldName, pattern, fieldValue))
				}
			}
		}
	}
}

func (p *PatternsRule) checkArrayFields(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Check StringOrArray fields
	arrayFields := map[string]*types.StringOrArray{
		"verb":            &query.Verb,
		"resource":        &query.Resource,
		"namespace":       &query.Namespace,
		"user":            &query.User,
		"response_status": &query.ResponseStatus,
		"source_ip":       &query.SourceIP,
		"group_by":        &query.GroupBy,
	}

	for fieldName, fieldValue := range arrayFields {
		if !fieldValue.IsEmpty() {
			if fieldValue.IsString() {
				for _, pattern := range p.forbiddenPatterns {
					if p.matchesPattern(fieldValue.GetString(), pattern) {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("Field '%s' contains forbidden pattern '%s': %s", fieldName, pattern, fieldValue.GetString()))
					}
				}
			} else if fieldValue.IsArray() {
				for _, item := range fieldValue.GetArray() {
					for _, pattern := range p.forbiddenPatterns {
						if p.matchesPattern(item, pattern) {
							result.IsValid = false
							result.Errors = append(result.Errors,
								fmt.Sprintf("Field '%s' contains forbidden pattern '%s': %s", fieldName, pattern, item))
						}
					}
				}
			}
		}
	}

	// Check string arrays
	if len(query.ExcludeUsers) > 0 {
		for _, user := range query.ExcludeUsers {
			for _, pattern := range p.forbiddenPatterns {
				if p.matchesPattern(user, pattern) {
					result.IsValid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Exclude user contains forbidden pattern '%s': %s", pattern, user))
				}
			}
		}
	}

	if len(query.ExcludeResources) > 0 {
		for _, resource := range query.ExcludeResources {
			for _, pattern := range p.forbiddenPatterns {
				if p.matchesPattern(resource, pattern) {
					result.IsValid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Exclude resource contains forbidden pattern '%s': %s", pattern, resource))
				}
			}
		}
	}
}

func (p *PatternsRule) checkSpecificPatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Check for dangerous URI patterns
	if query.RequestURIPattern != "" {
		dangerousURIPatterns := []string{
			"/api/v1/namespaces/.*/finalize",
			"/api/v1/namespaces/.*/status",
			"/api/v1/nodes/.*/proxy",
			"/api/v1/nodes/.*/status",
			"/api/v1/pods/.*/exec",
			"/api/v1/pods/.*/attach",
			"/api/v1/pods/.*/portforward",
			"/api/v1/pods/.*/log",
			"/api/v1/pods/.*/proxy",
			"/api/v1/services/.*/proxy",
		}

		for _, pattern := range dangerousURIPatterns {
			if p.matchesPattern(query.RequestURIPattern, pattern) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Request URI pattern contains dangerous pattern '%s': %s", pattern, query.RequestURIPattern))
			}
		}
	}

	// Check for dangerous namespace patterns
	if query.NamespacePattern != "" {
		dangerousNamespacePatterns := []string{
			"kube-system",
			"openshift-.*",
			"default",
			"kube-public",
			"kube-node-lease",
			"security",
			"prod.*",
			"production",
		}

		for _, pattern := range dangerousNamespacePatterns {
			if p.matchesPattern(query.NamespacePattern, pattern) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Namespace pattern contains dangerous pattern '%s': %s", pattern, query.NamespacePattern))
			}
		}
	}

	// Check for dangerous user patterns
	if query.UserPattern != "" {
		dangerousUserPatterns := []string{
			"system:admin",
			"system:masters",
			"cluster-admin",
			"admin",
		}

		for _, pattern := range dangerousUserPatterns {
			if p.matchesPattern(query.UserPattern, pattern) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("User pattern contains dangerous pattern '%s': %s", pattern, query.UserPattern))
			}
		}
	}

	// Check for dangerous resource patterns
	if query.ResourceNamePattern != "" {
		dangerousResourcePatterns := []string{
			"kube-system",
			"openshift-.*",
			"default",
			"kube-public",
			"kube-node-lease",
		}

		for _, pattern := range dangerousResourcePatterns {
			if p.matchesPattern(query.ResourceNamePattern, pattern) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Resource name pattern contains dangerous pattern '%s': %s", pattern, query.ResourceNamePattern))
			}
		}
	}
}

func (p *PatternsRule) matchesPattern(value, pattern string) bool {
	// Try exact match first
	if strings.EqualFold(value, pattern) {
		return true
	}

	// Try regex match if pattern contains regex characters
	if strings.Contains(pattern, ".*") || strings.Contains(pattern, "\\") {
		if matched, err := regexp.MatchString(pattern, value); err == nil && matched {
			return true
		}
	}

	// Try substring match (case-insensitive)
	if strings.Contains(strings.ToLower(value), strings.ToLower(pattern)) {
		return true
	}

	return false
}
