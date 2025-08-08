package rules

import (
	"fmt"
	"regexp"
	"strings"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// SanitizationRule implements input sanitization to prevent injection attacks
type SanitizationRule struct {
	forbiddenChars     []string
	maxPatternLength   int
	maxQueryLength     int
	validRegexPattern  string
	validIPPattern     string
	validNamespacePattern string
	validResourcePattern  string
	enabled            bool
}

// NewSanitizationRule creates a new sanitization validation rule
func NewSanitizationRule(config map[string]interface{}) *SanitizationRule {
	rule := &SanitizationRule{
		enabled: true,
	}

	// Extract configuration values
	if maxLength, ok := config["max_pattern_length"].(int); ok {
		rule.maxPatternLength = maxLength
	} else {
		rule.maxPatternLength = 500
	}

	if maxQueryLength, ok := config["max_query_length"].(int); ok {
		rule.maxQueryLength = maxQueryLength
	} else {
		rule.maxQueryLength = 10000
	}

	if validRegex, ok := config["valid_regex_pattern"].(string); ok {
		rule.validRegexPattern = validRegex
	} else {
		rule.validRegexPattern = "^[a-zA-Z0-9\\-_\\*\\.\\?\\+\\[\\]\\{\\}\\(\\)\\|\\\\/\\s]+$"
	}

	if validIP, ok := config["valid_ip_pattern"].(string); ok {
		rule.validIPPattern = validIP
	} else {
		rule.validIPPattern = "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
	}

	if validNamespace, ok := config["valid_namespace_pattern"].(string); ok {
		rule.validNamespacePattern = validNamespace
	} else {
		rule.validNamespacePattern = "^[a-z0-9]([a-z0-9\\-]*[a-z0-9])?$"
	}

	if validResource, ok := config["valid_resource_pattern"].(string); ok {
		rule.validResourcePattern = validResource
	} else {
		rule.validResourcePattern = "^[a-z]([a-z0-9\\-]*[a-z0-9])?$"
	}

	if forbiddenChars, ok := config["forbidden_chars"].([]interface{}); ok {
		for _, char := range forbiddenChars {
			if str, ok := char.(string); ok {
				rule.forbiddenChars = append(rule.forbiddenChars, str)
			}
		}
	} else {
		// Default forbidden characters
		rule.forbiddenChars = []string{"<", ">", "&", "\"", "'", "`", "|", ";", "$", "(", ")", "{", "}", "[", "]", "\\", "/", "!", "@", "#", "%", "^", "*", "+", "=", "~"}
	}

	return rule
}

// Validate applies sanitization validation to the query
func (s *SanitizationRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:        true,
		RuleName:       "sanitization_validation",
		Severity:       "high",
		Message:        "Input sanitization validation passed",
		Details:        make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:       []string{},
		Errors:         []string{},
		QuerySnapshot:  query,
	}

	// Check for forbidden characters in patterns
	s.checkForbiddenChars(query, result)

	// Check pattern lengths
	s.checkPatternLengths(query, result)

	// Validate regex patterns
	s.validateRegexPatterns(query, result)

	// Validate IP patterns
	s.validateIPPatterns(query, result)

	// Validate namespace patterns
	s.validateNamespacePatterns(query, result)

	// Validate resource patterns
	s.validateResourcePatterns(query, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Input sanitization validation failed"
		result.Severity = "high"
		result.Recommendations = append(result.Recommendations,
			"Remove forbidden characters from patterns",
			"Use only alphanumeric characters, hyphens, and underscores",
			"Keep patterns within length limits",
			"Use valid regex patterns only")
	}

	return result
}

// GetRuleName returns the rule name
func (s *SanitizationRule) GetRuleName() string {
	return "sanitization_validation"
}

// GetRuleDescription returns the rule description
func (s *SanitizationRule) GetRuleDescription() string {
	return "Validates input sanitization to prevent injection attacks"
}

// IsEnabled indicates if the rule is enabled
func (s *SanitizationRule) IsEnabled() bool {
	return s.enabled
}

// GetSeverity returns the rule severity
func (s *SanitizationRule) GetSeverity() string {
	return "high"
}

// Helper methods
func (s *SanitizationRule) checkForbiddenChars(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	patterns := []string{
		query.ResourceNamePattern,
		query.UserPattern,
		query.NamespacePattern,
		query.RequestURIPattern,
		query.AuthorizationReasonPattern,
		query.ResponseMessagePattern,
		query.MissingAnnotation,
		query.RequestObjectFilter,
	}

	for _, pattern := range patterns {
		if pattern != "" {
			for _, forbidden := range s.forbiddenChars {
				if strings.Contains(pattern, forbidden) {
					result.IsValid = false
					result.Errors = append(result.Errors, 
						fmt.Sprintf("Pattern contains forbidden character '%s': %s", forbidden, pattern))
				}
			}
		}
	}

	// Check exclude arrays
	if len(query.ExcludeUsers) > 0 {
		for _, user := range query.ExcludeUsers {
			for _, forbidden := range s.forbiddenChars {
				if strings.Contains(user, forbidden) {
					result.IsValid = false
					result.Errors = append(result.Errors, 
						fmt.Sprintf("Exclude user contains forbidden character '%s': %s", forbidden, user))
				}
			}
		}
	}

	if len(query.ExcludeResources) > 0 {
		for _, resource := range query.ExcludeResources {
			for _, forbidden := range s.forbiddenChars {
				if strings.Contains(resource, forbidden) {
					result.IsValid = false
					result.Errors = append(result.Errors, 
						fmt.Sprintf("Exclude resource contains forbidden character '%s': %s", forbidden, resource))
				}
			}
		}
	}
}

func (s *SanitizationRule) checkPatternLengths(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	patterns := map[string]string{
		"resource_name_pattern": query.ResourceNamePattern,
		"user_pattern":          query.UserPattern,
		"namespace_pattern":     query.NamespacePattern,
		"request_uri_pattern":   query.RequestURIPattern,
	}

	for name, pattern := range patterns {
		if pattern != "" && len(pattern) > s.maxPatternLength {
			result.IsValid = false
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Pattern '%s' exceeds maximum length of %d characters", name, s.maxPatternLength))
		}
	}
}

func (s *SanitizationRule) validateRegexPatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	patterns := []string{
		query.ResourceNamePattern,
		query.UserPattern,
		query.NamespacePattern,
		query.RequestURIPattern,
		query.AuthorizationReasonPattern,
		query.ResponseMessagePattern,
	}

	for _, pattern := range patterns {
		if pattern != "" {
			if !s.isValidRegex(pattern) {
				result.IsValid = false
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Invalid regex pattern: %s", pattern))
			}
		}
	}
}

func (s *SanitizationRule) validateIPPatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if !query.SourceIP.IsEmpty() {
		if query.SourceIP.IsString() {
			if !s.isValidIP(query.SourceIP.GetString()) {
				result.IsValid = false
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Invalid IP address: %s", query.SourceIP.GetString()))
			}
		} else if query.SourceIP.IsArray() {
			for _, ip := range query.SourceIP.GetArray() {
				if !s.isValidIP(ip) {
					result.IsValid = false
					result.Errors = append(result.Errors, 
						fmt.Sprintf("Invalid IP address: %s", ip))
				}
			}
		}
	}
}

func (s *SanitizationRule) validateNamespacePatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if query.NamespacePattern != "" {
		if !s.isValidNamespacePattern(query.NamespacePattern) {
			result.IsValid = false
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Invalid namespace pattern: %s", query.NamespacePattern))
		}
	}
}

func (s *SanitizationRule) validateResourcePatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	if query.ResourceNamePattern != "" {
		if !s.isValidResourcePattern(query.ResourceNamePattern) {
			result.IsValid = false
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Invalid resource pattern: %s", query.ResourceNamePattern))
		}
	}
}

func (s *SanitizationRule) isValidRegex(pattern string) bool {
	if s.validRegexPattern == "" {
		return true // Skip validation if no pattern defined
	}
	matched, _ := regexp.MatchString(s.validRegexPattern, pattern)
	return matched
}

func (s *SanitizationRule) isValidIP(ip string) bool {
	if s.validIPPattern == "" {
		return true // Skip validation if no pattern defined
	}
	matched, _ := regexp.MatchString(s.validIPPattern, ip)
	return matched
}

func (s *SanitizationRule) isValidNamespacePattern(pattern string) bool {
	if s.validNamespacePattern == "" {
		return true // Skip validation if no pattern defined
	}
	matched, _ := regexp.MatchString(s.validNamespacePattern, pattern)
	return matched
}

func (s *SanitizationRule) isValidResourcePattern(pattern string) bool {
	if s.validResourcePattern == "" {
		return true // Skip validation if no pattern defined
	}
	matched, _ := regexp.MatchString(s.validResourcePattern, pattern)
	return matched
}
