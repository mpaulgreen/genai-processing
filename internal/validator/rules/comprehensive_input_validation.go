package rules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ComprehensiveInputValidationRule consolidates all overlapping input validation concerns
// Replaces: SanitizationRule, PatternsRule, RequiredFieldsRule, and FieldValuesRule
type ComprehensiveInputValidationRule struct {
	config  *types.InputValidationConfig
	enabled bool
}


// NewComprehensiveInputValidationRule creates a new comprehensive input validation rule
func NewComprehensiveInputValidationRule(config *types.InputValidationConfig) *ComprehensiveInputValidationRule {
	if config == nil {
		config = getDefaultInputValidationConfig()
	}
	
	return &ComprehensiveInputValidationRule{
		config:  config,
		enabled: config.Enabled,
	}
}

// Validate applies comprehensive input validation in a single pass
func (r *ComprehensiveInputValidationRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "comprehensive_input_validation",
		Severity:        "info",
		Message:         "Input validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Handle nil query
	if query == nil {
		result.IsValid = false
		result.Severity = "critical"
		result.Message = "Input validation failed"
		result.Errors = append(result.Errors, "Query cannot be nil")
		return result
	}

	// Single-pass validation applying all validators
	r.validateRequiredFields(query, result)
	r.validateCharacters(query, result)
	r.validateSecurityPatterns(query, result)
	r.validateFieldValues(query, result)
	r.validatePerformanceLimits(query, result)

	// Set final result status
	if !result.IsValid {
		result.Severity = "critical"
		result.Message = "Input validation failed"
		result.Recommendations = append(result.Recommendations,
			"Fix all validation errors before proceeding",
			"Review query parameters for compliance with security policies")
	} else if len(result.Warnings) > 0 {
		result.Severity = "warning"
		result.Message = "Input validation passed with warnings"
	}

	// Add validation details
	result.Details["validation_sections"] = map[string]interface{}{
		"required_fields_checked":    len(r.config.RequiredFields.Mandatory),
		"character_validation_applied": true,
		"security_patterns_checked":  len(r.config.SecurityPatterns.ForbiddenPatterns),
		"field_values_validated":     true,
		"performance_limits_applied": true,
	}

	return result
}

// validateRequiredFields checks that mandatory fields are present
func (r *ComprehensiveInputValidationRule) validateRequiredFields(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	for _, field := range r.config.RequiredFields.Mandatory {
		if !r.isFieldPresent(query, field) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Required field '%s' is missing or empty", field))
		}
	}
}

// validateCharacters checks character encoding and format safety
func (r *ComprehensiveInputValidationRule) validateCharacters(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Check string fields for forbidden characters and length limits
	stringFields := r.getStringFields(query)
	
	for fieldName, fieldValue := range stringFields {
		if fieldValue == "" {
			continue
		}
		
		// Check length limits
		if len(fieldValue) > r.config.CharacterValidation.MaxPatternLength {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Field '%s' exceeds maximum length of %d characters", 
					fieldName, r.config.CharacterValidation.MaxPatternLength))
		}
		
		// Check for forbidden characters
		for _, forbiddenChar := range r.config.CharacterValidation.ForbiddenChars {
			if strings.Contains(fieldValue, forbiddenChar) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field '%s' contains forbidden character '%s'", 
						fieldName, forbiddenChar))
			}
		}
		
		// Validate regex patterns for specific fields
		if strings.HasSuffix(fieldName, "_pattern") && r.config.CharacterValidation.ValidRegexPattern != "" {
			if matched, _ := regexp.MatchString(r.config.CharacterValidation.ValidRegexPattern, fieldValue); !matched {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Field '%s' does not match recommended pattern format", fieldName))
			}
		}
		
		// Validate IP patterns
		if strings.Contains(fieldName, "ip") && r.config.CharacterValidation.ValidIPPattern != "" {
			if matched, _ := regexp.MatchString(r.config.CharacterValidation.ValidIPPattern, fieldValue); !matched {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Field '%s' does not appear to be a valid IP address", fieldName))
			}
		}
	}
}

// validateSecurityPatterns checks for dangerous security patterns
func (r *ComprehensiveInputValidationRule) validateSecurityPatterns(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Get all fields to check for patterns
	allFields := r.getAllFieldValues(query)
	
	for fieldName, fieldValue := range allFields {
		if fieldValue == "" {
			continue
		}
		
		// Check against forbidden patterns
		for _, pattern := range r.config.SecurityPatterns.ForbiddenPatterns {
			if strings.Contains(strings.ToLower(fieldValue), strings.ToLower(pattern)) {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field '%s' contains forbidden security pattern: %s", 
						fieldName, pattern))
			}
		}
	}
}

// validateFieldValues checks that field values are from allowed lists
func (r *ComprehensiveInputValidationRule) validateFieldValues(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Validate log source
	if query.LogSource != "" && !r.isInAllowedList(query.LogSource, r.config.FieldValues.AllowedLogSources) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Log source '%s' is not in allowed list", query.LogSource))
	}
	
	// Validate verbs
	r.validateStringOrArrayField("verb", query.Verb, r.config.FieldValues.AllowedVerbs, result)
	
	// Validate resources
	r.validateStringOrArrayField("resource", query.Resource, r.config.FieldValues.AllowedResources, result)
	
	// Validate auth decision
	if query.AuthDecision != "" && !r.isInAllowedList(query.AuthDecision, r.config.FieldValues.AllowedAuthDecisions) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Auth decision '%s' is not in allowed list", query.AuthDecision))
	}
	
	// Validate response status
	r.validateStringOrArrayField("response_status", query.ResponseStatus, r.config.FieldValues.AllowedResponseStatus, result)
}

// validatePerformanceLimits checks performance and resource limits
func (r *ComprehensiveInputValidationRule) validatePerformanceLimits(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// Validate result limit
	if query.Limit > r.config.PerformanceLimits.MaxResultLimit {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Result limit %d exceeds maximum allowed limit of %d", 
				query.Limit, r.config.PerformanceLimits.MaxResultLimit))
	}
	
	// Validate array sizes
	r.validateArraySize("exclude_users", len(query.ExcludeUsers), result)
	r.validateArraySize("exclude_resources", len(query.ExcludeResources), result)
	
	// Validate timeframe
	if query.Timeframe != "" && !r.isInAllowedList(query.Timeframe, r.config.PerformanceLimits.AllowedTimeframes) {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Timeframe '%s' is not in allowed list", query.Timeframe))
	}
}

// Helper methods

// isFieldPresent checks if a required field is present and non-empty
func (r *ComprehensiveInputValidationRule) isFieldPresent(query *types.StructuredQuery, field string) bool {
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

// getStringFields returns a map of string field names to their values
func (r *ComprehensiveInputValidationRule) getStringFields(query *types.StructuredQuery) map[string]string {
	return map[string]string{
		"log_source":                      query.LogSource,
		"timeframe":                       query.Timeframe,
		"subresource":                     query.Subresource,
		"auth_decision":                   query.AuthDecision,
		"resource_name_pattern":           query.ResourceNamePattern,
		"user_pattern":                    query.UserPattern,
		"namespace_pattern":               query.NamespacePattern,
		"request_uri_pattern":             query.RequestURIPattern,
		"authorization_reason_pattern":    query.AuthorizationReasonPattern,
		"response_message_pattern":        query.ResponseMessagePattern,
		"missing_annotation":              query.MissingAnnotation,
		"request_object_filter":           query.RequestObjectFilter,
		"sort_by":                         query.SortBy,
		"sort_order":                      query.SortOrder,
	}
}

// getAllFieldValues returns all field values for pattern checking
func (r *ComprehensiveInputValidationRule) getAllFieldValues(query *types.StructuredQuery) map[string]string {
	fields := r.getStringFields(query)
	
	// Add StringOrArray fields
	if !query.Verb.IsEmpty() {
		fields["verb"] = strings.Join(r.getStringOrArrayValues(query.Verb), " ")
	}
	if !query.Resource.IsEmpty() {
		fields["resource"] = strings.Join(r.getStringOrArrayValues(query.Resource), " ")
	}
	if !query.Namespace.IsEmpty() {
		fields["namespace"] = strings.Join(r.getStringOrArrayValues(query.Namespace), " ")
	}
	if !query.User.IsEmpty() {
		fields["user"] = strings.Join(r.getStringOrArrayValues(query.User), " ")
	}
	if !query.ResponseStatus.IsEmpty() {
		fields["response_status"] = strings.Join(r.getStringOrArrayValues(query.ResponseStatus), " ")
	}
	if !query.SourceIP.IsEmpty() {
		fields["source_ip"] = strings.Join(r.getStringOrArrayValues(query.SourceIP), " ")
	}
	if !query.GroupBy.IsEmpty() {
		fields["group_by"] = strings.Join(r.getStringOrArrayValues(query.GroupBy), " ")
	}
	
	// Add array fields
	if len(query.ExcludeUsers) > 0 {
		fields["exclude_users"] = strings.Join(query.ExcludeUsers, " ")
	}
	if len(query.ExcludeResources) > 0 {
		fields["exclude_resources"] = strings.Join(query.ExcludeResources, " ")
	}
	
	return fields
}

// validateStringOrArrayField validates StringOrArray fields against allowed lists
func (r *ComprehensiveInputValidationRule) validateStringOrArrayField(fieldName string, field types.StringOrArray, allowedValues []string, result *interfaces.ValidationResult) {
	if field.IsEmpty() {
		return
	}
	
	for _, value := range r.getStringOrArrayValues(field) {
		if !r.isInAllowedList(value, allowedValues) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Value '%s' in field '%s' is not in allowed list", value, fieldName))
		}
	}
}

// validateArraySize checks if array size exceeds limits
func (r *ComprehensiveInputValidationRule) validateArraySize(fieldName string, size int, result *interfaces.ValidationResult) {
	if size > r.config.PerformanceLimits.MaxArrayElements {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Array field '%s' has %d elements, exceeds maximum of %d", 
				fieldName, size, r.config.PerformanceLimits.MaxArrayElements))
	}
}

// isInAllowedList checks if a value is in the allowed list
func (r *ComprehensiveInputValidationRule) isInAllowedList(value string, allowedList []string) bool {
	for _, allowed := range allowedList {
		if value == allowed {
			return true
		}
	}
	return false
}

// getStringOrArrayValues extracts values from a StringOrArray field
func (r *ComprehensiveInputValidationRule) getStringOrArrayValues(field types.StringOrArray) []string {
	if field.IsString() {
		return []string{field.GetString()}
	} else if field.IsArray() {
		return field.GetArray()
	}
	return []string{}
}

// Interface implementation methods

// GetRuleName returns the rule name
func (r *ComprehensiveInputValidationRule) GetRuleName() string {
	return "comprehensive_input_validation"
}

// GetRuleDescription returns the rule description
func (r *ComprehensiveInputValidationRule) GetRuleDescription() string {
	return "Comprehensive input validation covering required fields, character safety, security patterns, field values, and performance limits"
}

// IsEnabled indicates if the rule is enabled
func (r *ComprehensiveInputValidationRule) IsEnabled() bool {
	return r.enabled
}

// GetSeverity returns the rule severity
func (r *ComprehensiveInputValidationRule) GetSeverity() string {
	return "critical"
}

// getDefaultInputValidationConfig provides default configuration when none is provided
func getDefaultInputValidationConfig() *types.InputValidationConfig {
	return &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory:   []string{"log_source"},
			Conditional: []string{},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxQueryLength:    10000,
			MaxPatternLength:  500,
			ForbiddenChars:    []string{"<", ">", "&", "\"", "'", "`", "|", ";", "$"},
			ValidRegexPattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+\\[\\]\\{\\}\\(\\)\\|\\\\/\\s]+$",
			ValidIPPattern:    "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{
				"system:admin",
				"system:masters",
				"cluster-admin",
				"delete --all",
				"delete --force",
				"privileged: true",
				"hostNetwork: true",
				"runAsUser: 0",
			},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{
				"kube-apiserver",
				"openshift-apiserver", 
				"oauth-server",
				"oauth-apiserver",
				"node-auditd",
			},
			AllowedVerbs: []string{
				"get", "list", "create", "update", "patch", "delete", "watch", "impersonate",
			},
			AllowedResources: []string{
				"pods", "services", "deployments", "configmaps", "secrets", "namespaces",
				"serviceaccounts", "roles", "rolebindings", "clusterroles", "clusterrolebindings",
				"customresourcedefinitions", "persistentvolumeclaims", "networkpolicies",
				"events", "nodes", "routes", "builds", "imagestreams", "projects",
				"users", "groups", "oauthclients", "securitycontextconstraints",
			},
			AllowedAuthDecisions: []string{"allow", "error", "forbid"},
			AllowedResponseStatus: []string{
				"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504",
			},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit:   50,
			MaxArrayElements: 15,
			MaxDaysBack:      90,
			AllowedTimeframes: []string{
				"today", "yesterday", "1_hour_ago", "6_hours_ago", "12_hours_ago",
				"1_day_ago", "3_days_ago", "7_days_ago", "30_days_ago", "90_days_ago",
			},
		},
	}
}