package rules

import (
	"fmt"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// WhitelistRule implements validation for allowed log sources, verbs, and resources
type WhitelistRule struct {
	allowedLogSources []string
	allowedVerbs      []string
	allowedResources  []string
	enabled           bool
}

// NewWhitelistRule creates a new whitelist validation rule
func NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources []string) *WhitelistRule {
	return &WhitelistRule{
		allowedLogSources: allowedLogSources,
		allowedVerbs:      allowedVerbs,
		allowedResources:  allowedResources,
		enabled:           true,
	}
}

// Validate applies whitelist validation to the query
func (w *WhitelistRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "whitelist_validation",
		Severity:        "critical",
		Message:         "Whitelist validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		QuerySnapshot:   query,
	}

	// Validate log source
	if query.LogSource != "" {
		if !w.isAllowedLogSource(query.LogSource) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Log source '%s' is not in allowed whitelist", query.LogSource))
		}
	}

	// Validate verbs
	if !query.Verb.IsEmpty() {
		if query.Verb.IsString() {
			if !w.isAllowedVerb(query.Verb.GetString()) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Verb '%s' is not in allowed whitelist", query.Verb.GetString()))
			}
		} else if query.Verb.IsArray() {
			for _, verb := range query.Verb.GetArray() {
				if !w.isAllowedVerb(verb) {
					result.IsValid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Verb '%s' is not in allowed whitelist", verb))
				}
			}
		}
	}

	// Validate resources
	if !query.Resource.IsEmpty() {
		if query.Resource.IsString() {
			if !w.isAllowedResource(query.Resource.GetString()) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Resource '%s' is not in allowed whitelist", query.Resource.GetString()))
			}
		} else if query.Resource.IsArray() {
			for _, resource := range query.Resource.GetArray() {
				if !w.isAllowedResource(resource) {
					result.IsValid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Resource '%s' is not in allowed whitelist", resource))
				}
			}
		}
	}

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Whitelist validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Use only allowed log sources, verbs, and resources from the whitelist",
			"Check the configuration for the complete list of allowed values")
	}

	return result
}

// GetRuleName returns the rule name
func (w *WhitelistRule) GetRuleName() string {
	return "whitelist_validation"
}

// GetRuleDescription returns the rule description
func (w *WhitelistRule) GetRuleDescription() string {
	return "Validates that log sources, verbs, and resources are in the allowed whitelist"
}

// IsEnabled indicates if the rule is enabled
func (w *WhitelistRule) IsEnabled() bool {
	return w.enabled
}

// GetSeverity returns the rule severity
func (w *WhitelistRule) GetSeverity() string {
	return "critical"
}

// Helper methods
func (w *WhitelistRule) isAllowedLogSource(source string) bool {
	for _, allowed := range w.allowedLogSources {
		if strings.EqualFold(source, allowed) {
			return true
		}
	}
	return false
}

func (w *WhitelistRule) isAllowedVerb(verb string) bool {
	for _, allowed := range w.allowedVerbs {
		if strings.EqualFold(verb, allowed) {
			return true
		}
	}
	return false
}

func (w *WhitelistRule) isAllowedResource(resource string) bool {
	for _, allowed := range w.allowedResources {
		if strings.EqualFold(resource, allowed) {
			return true
		}
	}
	return false
}
