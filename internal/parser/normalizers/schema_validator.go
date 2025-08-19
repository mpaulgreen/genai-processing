package normalizers

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// =============================================================================
// VALIDATION ERROR TYPES
// =============================================================================

// ValidationError represents a structured validation error with detailed information
type ValidationError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Field       string                 `json:"field"`
	Expected    string                 `json:"expected,omitempty"`
	Actual      string                 `json:"actual,omitempty"`
	Suggestion  string                 `json:"suggestion,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Severity    string                 `json:"severity"`
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (field: %s)", ve.Code, ve.Message, ve.Field)
}

// QueryComplexity represents the complexity analysis of a query
type QueryComplexity struct {
	Score                 int                    `json:"score"`
	Level                 string                 `json:"level"`  // Low, Medium, High
	Components            map[string]int         `json:"components"`
	PerformanceWarnings   []string               `json:"performance_warnings,omitempty"`
	ResourceUsage         map[string]interface{} `json:"resource_usage"`
}

// =============================================================================
// SCHEMA VALIDATOR MAIN STRUCT
// =============================================================================

// SchemaValidator implements interfaces.SchemaValidator with enhanced capabilities
type SchemaValidator struct {
	// Valid log sources for validation
	validLogSources []string
	// Valid HTTP verbs
	validVerbs []string
	// Valid timeframes
	validTimeframes []string
	// Valid analysis types
	validAnalysisTypes []string
	// Valid auth decisions
	validAuthDecisions []string
	// Performance thresholds
	complexityThresholds map[string]int
}

func NewSchemaValidator() interfaces.SchemaValidator {
	return &SchemaValidator{
		validLogSources: []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"},
		validVerbs:      []string{"get", "list", "create", "update", "patch", "delete", "watch", "connect", "proxy", "redirect", "bind"},
		validTimeframes: []string{"today", "yesterday", "1_hour_ago", "6_hours_ago", "12_hours_ago", "24_hours_ago", "7_days_ago", "30_days_ago", "last_week", "last_month"},
		validAnalysisTypes: []string{"anomaly_detection", "behavioral_analysis", "correlation_analysis", "statistical_analysis", "threat_hunting", "apt_reconnaissance_detection", "apt_lateral_movement_detection", "apt_data_exfiltration_detection", "privilege_escalation_detection", "rapid_operations_detection", "user_behavior_profiling", "resource_access_pattern_analysis", "authentication_anomaly_detection", "network_pattern_analysis", "temporal_pattern_analysis"},
		validAuthDecisions: []string{"allow", "error", "forbid"},
		complexityThresholds: map[string]int{
			"low":    20,
			"medium": 50,
		},
	}
}

// =============================================================================
// MAIN VALIDATION ENTRY POINT
// =============================================================================

// ValidateSchema enforces comprehensive schema constraints on StructuredQuery
func (v *SchemaValidator) ValidateSchema(q *types.StructuredQuery) error {
	if q == nil {
		return &ValidationError{
			Code:     "FIELD_REQUIRED",
			Message:  "query cannot be nil",
			Field:    "query",
			Severity: "ERROR",
		}
	}

	// Phase 1: Required field validation
	if err := v.validateRequiredFields(q); err != nil {
		return err
	}

	// Phase 2: Basic field validation
	if err := v.validateBasicFields(q); err != nil {
		return err
	}

	// Phase 3: Advanced field validation
	if err := v.validateAdvancedFields(q); err != nil {
		return err
	}

	// Phase 4: Complex object validation
	if err := v.validateComplexObjects(q); err != nil {
		return err
	}

	// Phase 5: Cross-field validation
	if err := v.validateCrossFieldDependencies(q); err != nil {
		return err
	}

	// Phase 6: Performance and complexity validation
	if err := v.validatePerformanceImpact(q); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PHASE 1: REQUIRED FIELD VALIDATION
// =============================================================================

// validateRequiredFields validates all required fields
func (v *SchemaValidator) validateRequiredFields(q *types.StructuredQuery) error {
	// log_source is required
	if strings.TrimSpace(q.LogSource) == "" {
		return &ValidationError{
			Code:       "FIELD_REQUIRED",
			Message:    "log_source is required for all queries",
			Field:      "log_source",
			Suggestion: "Add log_source field with value: kube-apiserver, openshift-apiserver, oauth-server, oauth-apiserver, or node-auditd",
			Severity:   "ERROR",
		}
	}

	// Validate log_source is in allowed values
	if !v.isValidLogSource(q.LogSource) {
		return &ValidationError{
			Code:       "FIELD_ENUM",
			Message:    "invalid log source",
			Field:      "log_source",
			Expected:   strings.Join(v.validLogSources, ", "),
			Actual:     q.LogSource,
			Suggestion: "Use one of the valid log sources: " + strings.Join(v.validLogSources, ", "),
			Severity:   "ERROR",
		}
	}

	return nil
}

// =============================================================================
// PHASE 2: BASIC FIELD VALIDATION
// =============================================================================

// validateBasicFields validates basic filtering fields
func (v *SchemaValidator) validateBasicFields(q *types.StructuredQuery) error {
	// Validate limit
	if q.Limit < 0 || q.Limit > 1000 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "limit value out of allowed range",
			Field:      "limit",
			Expected:   "1-1000",
			Actual:     strconv.Itoa(q.Limit),
			Suggestion: "Set limit between 1 and 1000",
			Severity:   "ERROR",
		}
	}

	// Validate verb
	if err := v.validateStringOrArray(q.Verb, "verb", v.validVerbs, 10); err != nil {
		return err
	}

	// Validate namespace format
	if err := v.validateNamespaces(q.Namespace); err != nil {
		return err
	}

	// Validate user format
	if err := v.validateUsers(q.User); err != nil {
		return err
	}

	// Validate timeframe
	if q.Timeframe != "" && !v.isValidTimeframe(q.Timeframe) {
		return &ValidationError{
			Code:       "FIELD_ENUM",
			Message:    "invalid timeframe value",
			Field:      "timeframe",
			Expected:   strings.Join(v.validTimeframes, ", "),
			Actual:     q.Timeframe,
			Suggestion: "Use one of the valid timeframes: " + strings.Join(v.validTimeframes, ", "),
			Severity:   "ERROR",
		}
	}

	// Validate source_ip
	if err := v.validateSourceIPs(q.SourceIP); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PHASE 3: ADVANCED FIELD VALIDATION
// =============================================================================

// validateAdvancedFields validates advanced filtering fields
func (v *SchemaValidator) validateAdvancedFields(q *types.StructuredQuery) error {
	// Validate regex patterns
	if err := v.validateRegexPattern(q.UserPattern, "user_pattern"); err != nil {
		return err
	}
	if err := v.validateRegexPattern(q.NamespacePattern, "namespace_pattern"); err != nil {
		return err
	}
	if err := v.validateRegexPattern(q.ResourceNamePattern, "resource_name_pattern"); err != nil {
		return err
	}
	if err := v.validateRegexPattern(q.RequestURIPattern, "request_uri_pattern"); err != nil {
		return err
	}

	// Validate response_status
	if err := v.validateResponseStatus(q.ResponseStatus); err != nil {
		return err
	}

	// Validate auth_decision
	if q.AuthDecision != "" && !v.isValidAuthDecision(q.AuthDecision) {
		return &ValidationError{
			Code:       "FIELD_ENUM",
			Message:    "invalid auth_decision value",
			Field:      "auth_decision",
			Expected:   strings.Join(v.validAuthDecisions, ", "),
			Actual:     q.AuthDecision,
			Suggestion: "Use one of the valid auth decisions: " + strings.Join(v.validAuthDecisions, ", "),
			Severity:   "ERROR",
		}
	}

	// Validate exclude_users array
	if len(q.ExcludeUsers) > 50 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "too many exclude_users patterns",
			Field:      "exclude_users",
			Expected:   "maximum 50 elements",
			Actual:     strconv.Itoa(len(q.ExcludeUsers)),
			Suggestion: "Reduce the number of exclude patterns to 50 or fewer",
			Severity:   "ERROR",
		}
	}

	// Check for empty strings in exclude_users
	for i, user := range q.ExcludeUsers {
		if strings.TrimSpace(user) == "" {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "empty string not allowed in exclude_users",
				Field:      fmt.Sprintf("exclude_users[%d]", i),
				Suggestion: "Remove empty strings from exclude_users array",
				Severity:   "ERROR",
			}
		}
	}

	// Validate time_range
	if err := v.validateTimeRange(q.TimeRange); err != nil {
		return err
	}

	// Validate business_hours
	if err := v.validateBusinessHours(q.BusinessHours); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PHASE 4: COMPLEX OBJECT VALIDATION
// =============================================================================

// validateComplexObjects validates complex nested objects
func (v *SchemaValidator) validateComplexObjects(q *types.StructuredQuery) error {
	// Validate multi_source configuration
	if err := v.ValidateMultiSource(q.MultiSource); err != nil {
		return err
	}

	// Validate advanced analysis configuration
	if err := v.ValidateAdvancedAnalysis(q.Analysis); err != nil {
		return err
	}

	// Validate behavioral analysis configuration
	if err := v.ValidateBehavioralAnalysis(q.BehavioralAnalysis); err != nil {
		return err
	}

	// Validate threat intelligence configuration
	if err := v.ValidateThreatIntelligence(q.ThreatIntelligence); err != nil {
		return err
	}

	// Validate machine learning configuration
	if err := v.ValidateMachineLearning(q.MachineLearning); err != nil {
		return err
	}

	// Validate detection criteria configuration
	if err := v.ValidateDetectionCriteria(q.DetectionCriteria); err != nil {
		return err
	}

	// Validate security context configuration
	if err := v.ValidateSecurityContext(q.SecurityContext); err != nil {
		return err
	}

	// Validate compliance framework configuration
	if err := v.ValidateComplianceFramework(q.ComplianceFramework); err != nil {
		return err
	}

	// Validate temporal analysis configuration
	if err := v.ValidateTemporalAnalysis(q.TemporalAnalysis); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PHASE 5: CROSS-FIELD VALIDATION
// =============================================================================

// validateCrossFieldDependencies validates field relationships and dependencies
func (v *SchemaValidator) validateCrossFieldDependencies(q *types.StructuredQuery) error {
	// Mutual exclusion: timeframe and time_range
	if q.Timeframe != "" && q.TimeRange != nil {
		return &ValidationError{
			Code:       "FIELD_CONFLICT",
			Message:    "timeframe and time_range are mutually exclusive",
			Field:      "timeframe,time_range",
			Suggestion: "Use either timeframe or time_range, not both",
			Severity:   "ERROR",
		}
	}

	// Log source compatibility validation
	if err := v.validateLogSourceCompatibility(q); err != nil {
		return err
	}

	// Analysis field dependencies
	if err := v.validateAnalysisDependencies(q); err != nil {
		return err
	}

	// Behavioral analysis dependencies
	if err := v.validateBehavioralAnalysisDependencies(q); err != nil {
		return err
	}

	// Machine learning dependencies
	if err := v.validateMachineLearningDependencies(q); err != nil {
		return err
	}

	// Threat intelligence dependencies
	if err := v.validateThreatIntelligenceDependencies(q); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PHASE 6: PERFORMANCE VALIDATION
// =============================================================================

// validatePerformanceImpact validates query complexity and performance implications
func (v *SchemaValidator) validatePerformanceImpact(q *types.StructuredQuery) error {
	complexity := v.calculateQueryComplexity(q)

	// Generate performance warnings for high complexity queries
	if complexity.Score > v.complexityThresholds["medium"] {
		// This is a warning, not an error, so we don't return an error
		// Instead, we could log or store warnings for later retrieval
	}

	// Check for extremely high limits
	if q.Limit > 500 {
		return &ValidationError{
			Code:       "PERFORMANCE_WARNING",
			Message:    "large limit may impact performance",
			Field:      "limit",
			Actual:     strconv.Itoa(q.Limit),
			Suggestion: "Consider reducing limit to 500 or less for better performance",
			Severity:   "WARNING",
		}
	}

	return nil
}

// =============================================================================
// FIELD-SPECIFIC VALIDATION HELPERS
// =============================================================================

// validateNamespaces validates namespace format according to DNS label rules
func (v *SchemaValidator) validateNamespaces(field types.StringOrArray) error {
	if field.GetValue() == nil {
		return nil // Optional field
	}

	// DNS label regex pattern
	dnsLabelPattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	regex, err := regexp.Compile(dnsLabelPattern)
	if err != nil {
		return &ValidationError{
			Code:     "INTERNAL_ERROR",
			Message:  "failed to compile namespace validation regex",
			Field:    "namespace",
			Severity: "ERROR",
		}
	}

	validateNamespace := func(namespace string, fieldPath string) error {
		if len(namespace) == 0 {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "namespace cannot be empty",
				Field:      fieldPath,
				Suggestion: "Provide a valid namespace name",
				Severity:   "ERROR",
			}
		}

		if len(namespace) > 63 {
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    "namespace name too long",
				Field:      fieldPath,
				Expected:   "1-63 characters",
				Actual:     strconv.Itoa(len(namespace)),
				Suggestion: "Reduce namespace name to 63 characters or fewer",
				Severity:   "ERROR",
			}
		}

		if !regex.MatchString(namespace) {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "invalid namespace format",
				Field:      fieldPath,
				Expected:   "DNS label format (lowercase letters, numbers, hyphens)",
				Actual:     namespace,
				Suggestion: "Use lowercase letters, numbers, and hyphens only",
				Severity:   "ERROR",
			}
		}

		return nil
	}

	if field.IsString() {
		return validateNamespace(field.GetString(), "namespace")
	} else if field.IsArray() {
		for i, namespace := range field.GetArray() {
			if err := validateNamespace(namespace, fmt.Sprintf("namespace[%d]", i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateUsers validates user format (email or system user)
func (v *SchemaValidator) validateUsers(field types.StringOrArray) error {
	if field.GetValue() == nil {
		return nil // Optional field
	}

	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	emailRegex, err := regexp.Compile(emailPattern)
	if err != nil {
		return &ValidationError{
			Code:     "INTERNAL_ERROR",
			Message:  "failed to compile email validation regex",
			Field:    "user",
			Severity: "ERROR",
		}
	}

	validateUser := func(user string, fieldPath string) error {
		if len(user) == 0 {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "user cannot be empty",
				Field:      fieldPath,
				Suggestion: "Provide a valid user identifier",
				Severity:   "ERROR",
			}
		}

		if len(user) > 256 {
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    "user name too long",
				Field:      fieldPath,
				Expected:   "1-256 characters",
				Actual:     strconv.Itoa(len(user)),
				Suggestion: "Reduce user name to 256 characters or fewer",
				Severity:   "ERROR",
			}
		}

		// If it contains @, validate as email
		if strings.Contains(user, "@") && !emailRegex.MatchString(user) {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "invalid email format",
				Field:      fieldPath,
				Expected:   "valid email address",
				Actual:     user,
				Suggestion: "Use a valid email format (user@domain.com)",
				Severity:   "ERROR",
			}
		}

		return nil
	}

	if field.IsString() {
		return validateUser(field.GetString(), "user")
	} else if field.IsArray() {
		for i, user := range field.GetArray() {
			if err := validateUser(user, fmt.Sprintf("user[%d]", i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSourceIPs validates IP addresses and CIDR notation
func (v *SchemaValidator) validateSourceIPs(field types.StringOrArray) error {
	if field.GetValue() == nil {
		return nil // Optional field
	}

	validateIP := func(ip string, fieldPath string) error {
		if ip == "" {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "IP address cannot be empty",
				Field:      fieldPath,
				Suggestion: "Provide a valid IP address or CIDR notation",
				Severity:   "ERROR",
			}
		}

		// Check if it's CIDR notation
		if strings.Contains(ip, "/") {
			_, _, err := net.ParseCIDR(ip)
			if err != nil {
				return &ValidationError{
					Code:       "FIELD_FORMAT",
					Message:    "invalid CIDR notation",
					Field:      fieldPath,
					Expected:   "valid CIDR format (e.g., 192.168.1.0/24)",
					Actual:     ip,
					Suggestion: "Use valid CIDR notation with correct subnet mask",
					Severity:   "ERROR",
				}
			}
		} else {
			// Check if it's a valid IP address
			if net.ParseIP(ip) == nil {
				return &ValidationError{
					Code:       "FIELD_FORMAT",
					Message:    "invalid IP address",
					Field:      fieldPath,
					Expected:   "valid IPv4 or IPv6 address",
					Actual:     ip,
					Suggestion: "Use a valid IP address format",
					Severity:   "ERROR",
				}
			}
		}

		return nil
	}

	if field.IsString() {
		return validateIP(field.GetString(), "source_ip")
	} else if field.IsArray() {
		for i, ip := range field.GetArray() {
			if err := validateIP(ip, fmt.Sprintf("source_ip[%d]", i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateRegexPattern validates regular expression syntax and safety
func (v *SchemaValidator) validateRegexPattern(pattern, fieldName string) error {
	if pattern == "" {
		return nil // Optional field
	}

	// Test regex compilation
	_, err := regexp.Compile(pattern)
	if err != nil {
		return &ValidationError{
			Code:       "FIELD_FORMAT",
			Message:    "invalid regular expression syntax",
			Field:      fieldName,
			Expected:   "valid regex pattern",
			Actual:     pattern,
			Suggestion: "Fix regex syntax errors",
			Severity:   "ERROR",
			Details:    map[string]interface{}{"regex_error": err.Error()},
		}
	}

	// Check for catastrophic backtracking patterns
	dangerousPatterns := []string{
		`(.+)+`,        // Nested quantifiers
		`(.*)∗`,        // Nested quantifiers
		`(.+)∗`,        // Nested quantifiers
		`(a|a)∗`,       // Alternation with overlap
		`(a∗)∗`,        // Nested star quantifiers
		`(a+)+`,        // Nested plus quantifiers
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(pattern, dangerous) {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "regex pattern may cause catastrophic backtracking",
				Field:      fieldName,
				Actual:     pattern,
				Suggestion: "Simplify regex pattern to avoid performance issues",
				Severity:   "ERROR",
			}
		}
	}

	// Calculate basic complexity score
	complexity := len(pattern) + strings.Count(pattern, "(") + strings.Count(pattern, "[") + strings.Count(pattern, "∗") + strings.Count(pattern, "+")
	if complexity > 100 {
		return &ValidationError{
			Code:       "PERFORMANCE_WARNING",
			Message:    "regex pattern is very complex",
			Field:      fieldName,
			Actual:     pattern,
			Suggestion: "Consider simplifying the regex pattern for better performance",
			Severity:   "WARNING",
		}
	}

	return nil
}

// validateResponseStatus validates HTTP status codes and ranges
func (v *SchemaValidator) validateResponseStatus(field types.StringOrArray) error {
	if field.GetValue() == nil {
		return nil // Optional field
	}

	validateStatus := func(status string, fieldPath string) error {
		if status == "" {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "response status cannot be empty",
				Field:      fieldPath,
				Suggestion: "Provide a valid HTTP status code",
				Severity:   "ERROR",
			}
		}

		// Handle range syntax (>=400, <500, etc.)
		if strings.HasPrefix(status, ">=") || strings.HasPrefix(status, "<=") || 
		   strings.HasPrefix(status, ">") || strings.HasPrefix(status, "<") {
			
			valueStr := ""
			
			if strings.HasPrefix(status, ">=") {
				valueStr = status[2:]
			} else if strings.HasPrefix(status, "<=") {
				valueStr = status[2:]
			} else if strings.HasPrefix(status, ">") {
				valueStr = status[1:]
			} else if strings.HasPrefix(status, "<") {
				valueStr = status[1:]
			}

			code, err := strconv.Atoi(valueStr)
			if err != nil {
				return &ValidationError{
					Code:       "FIELD_FORMAT",
					Message:    "invalid status code in range expression",
					Field:      fieldPath,
					Expected:   "valid integer status code",
					Actual:     status,
					Suggestion: "Use valid HTTP status code (100-599)",
					Severity:   "ERROR",
				}
			}

			if code < 100 || code > 599 {
				return &ValidationError{
					Code:       "FIELD_RANGE",
					Message:    "HTTP status code out of range",
					Field:      fieldPath,
					Expected:   "100-599",
					Actual:     strconv.Itoa(code),
					Suggestion: "Use valid HTTP status code range",
					Severity:   "ERROR",
				}
			}

			return nil
		}

		// Handle exact status code
		code, err := strconv.Atoi(status)
		if err != nil {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "invalid HTTP status code format",
				Field:      fieldPath,
				Expected:   "integer status code or range (>=400)",
				Actual:     status,
				Suggestion: "Use a valid HTTP status code or range syntax",
				Severity:   "ERROR",
			}
		}

		if code < 100 || code > 599 {
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    "HTTP status code out of range",
				Field:      fieldPath,
				Expected:   "100-599",
				Actual:     strconv.Itoa(code),
				Suggestion: "Use valid HTTP status code",
				Severity:   "ERROR",
			}
		}

		return nil
	}

	if field.IsString() {
		return validateStatus(field.GetString(), "response_status")
	} else if field.IsArray() {
		for i, status := range field.GetArray() {
			if err := validateStatus(status, fmt.Sprintf("response_status[%d]", i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateTimeRange validates time range objects
func (v *SchemaValidator) validateTimeRange(timeRange *types.TimeRange) error {
	if timeRange == nil {
		return nil // Optional field
	}

	// Validate that both start and end are provided
	if timeRange.Start.IsZero() {
		return &ValidationError{
			Code:       "FIELD_REQUIRED",
			Message:    "time_range.start is required",
			Field:      "time_range.start",
			Suggestion: "Provide a valid ISO 8601 timestamp",
			Severity:   "ERROR",
		}
	}

	if timeRange.End.IsZero() {
		return &ValidationError{
			Code:       "FIELD_REQUIRED",
			Message:    "time_range.end is required",
			Field:      "time_range.end",
			Suggestion: "Provide a valid ISO 8601 timestamp",
			Severity:   "ERROR",
		}
	}

	// Validate logical order
	if timeRange.End.Before(timeRange.Start) {
		return &ValidationError{
			Code:       "FIELD_CONFLICT",
			Message:    "time_range.end cannot be before time_range.start",
			Field:      "time_range",
			Expected:   "end >= start",
			Actual:     fmt.Sprintf("start: %s, end: %s", timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339)),
			Suggestion: "Ensure end time is after start time",
			Severity:   "ERROR",
		}
	}

	// Validate duration (maximum 90 days)
	duration := timeRange.End.Sub(timeRange.Start)
	maxDuration := 90 * 24 * time.Hour
	if duration > maxDuration {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "time range duration exceeds maximum allowed",
			Field:      "time_range",
			Expected:   "maximum 90 days",
			Actual:     fmt.Sprintf("%.1f days", duration.Hours()/24),
			Suggestion: "Reduce time range to 90 days or less",
			Severity:   "ERROR",
		}
	}

	return nil
}

// validateBusinessHours validates business hours configuration
func (v *SchemaValidator) validateBusinessHours(businessHours *types.BusinessHours) error {
	if businessHours == nil {
		return nil // Optional field
	}

	// Validate hour ranges
	if businessHours.StartHour < 0 || businessHours.StartHour > 23 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "start_hour out of range",
			Field:      "business_hours.start_hour",
			Expected:   "0-23",
			Actual:     strconv.Itoa(businessHours.StartHour),
			Suggestion: "Set start_hour between 0 and 23",
			Severity:   "ERROR",
		}
	}

	if businessHours.EndHour < 0 || businessHours.EndHour > 23 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "end_hour out of range",
			Field:      "business_hours.end_hour",
			Expected:   "0-23",
			Actual:     strconv.Itoa(businessHours.EndHour),
			Suggestion: "Set end_hour between 0 and 23",
			Severity:   "ERROR",
		}
	}

	// Validate timezone
	if businessHours.Timezone != "" {
		_, err := time.LoadLocation(businessHours.Timezone)
		if err != nil {
			return &ValidationError{
				Code:       "FIELD_FORMAT",
				Message:    "invalid timezone identifier",
				Field:      "business_hours.timezone",
				Expected:   "valid timezone (e.g., UTC, EST, America/New_York)",
				Actual:     businessHours.Timezone,
				Suggestion: "Use a valid IANA timezone identifier",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// validateStringOrArray validates StringOrArray fields
func (v *SchemaValidator) validateStringOrArray(field types.StringOrArray, fieldName string, validValues []string, maxElements int) error {
	if field.GetValue() == nil {
		return nil // Optional field
	}

	if field.IsString() {
		value := field.GetString()
		if !v.isValueInSlice(value, validValues) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    fmt.Sprintf("invalid %s value", fieldName),
				Field:      fieldName,
				Expected:   strings.Join(validValues, ", "),
				Actual:     value,
				Suggestion: fmt.Sprintf("Use one of the valid %s values: %s", fieldName, strings.Join(validValues, ", ")),
				Severity:   "ERROR",
			}
		}
	} else if field.IsArray() {
		values := field.GetArray()
		if len(values) > maxElements {
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    fmt.Sprintf("too many %s values", fieldName),
				Field:      fieldName,
				Expected:   fmt.Sprintf("maximum %d elements", maxElements),
				Actual:     strconv.Itoa(len(values)),
				Suggestion: fmt.Sprintf("Reduce the number of %s values to %d or fewer", fieldName, maxElements),
				Severity:   "ERROR",
			}
		}

		// Check for duplicates and invalid values
		seen := make(map[string]bool)
		for i, value := range values {
			if seen[value] {
				return &ValidationError{
					Code:       "FIELD_FORMAT",
					Message:    fmt.Sprintf("duplicate %s value", fieldName),
					Field:      fmt.Sprintf("%s[%d]", fieldName, i),
					Actual:     value,
					Suggestion: fmt.Sprintf("Remove duplicate %s values", fieldName),
					Severity:   "ERROR",
				}
			}
			seen[value] = true

			if !v.isValueInSlice(value, validValues) {
				return &ValidationError{
					Code:       "FIELD_ENUM",
					Message:    fmt.Sprintf("invalid %s value", fieldName),
					Field:      fmt.Sprintf("%s[%d]", fieldName, i),
					Expected:   strings.Join(validValues, ", "),
					Actual:     value,
					Suggestion: fmt.Sprintf("Use one of the valid %s values: %s", fieldName, strings.Join(validValues, ", ")),
					Severity:   "ERROR",
				}
			}
		}
	}

	return nil
}

// =============================================================================
// COMPLEX OBJECT VALIDATION METHODS
// =============================================================================

// ValidateMultiSource validates multi-source correlation configuration
func (v *SchemaValidator) ValidateMultiSource(config *types.MultiSourceConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate primary source
	if !v.isValidLogSource(config.PrimarySource) {
		return &ValidationError{
			Code:       "FIELD_ENUM",
			Message:    "invalid primary_source",
			Field:      "multi_source.primary_source",
			Expected:   strings.Join(v.validLogSources, ", "),
			Actual:     config.PrimarySource,
			Suggestion: "Use one of the valid log sources",
			Severity:   "ERROR",
		}
	}

	// Validate secondary sources
	if len(config.SecondarySources) == 0 {
		return &ValidationError{
			Code:       "FIELD_REQUIRED",
			Message:    "secondary_sources cannot be empty",
			Field:      "multi_source.secondary_sources",
			Suggestion: "Provide at least one secondary source for correlation",
			Severity:   "ERROR",
		}
	}

	seen := make(map[string]bool)
	seen[config.PrimarySource] = true

	for i, source := range config.SecondarySources {
		if !v.isValidLogSource(source) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid secondary source",
				Field:      fmt.Sprintf("multi_source.secondary_sources[%d]", i),
				Expected:   strings.Join(v.validLogSources, ", "),
				Actual:     source,
				Suggestion: "Use one of the valid log sources",
				Severity:   "ERROR",
			}
		}

		if seen[source] {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "primary source cannot be in secondary sources",
				Field:      fmt.Sprintf("multi_source.secondary_sources[%d]", i),
				Actual:     source,
				Suggestion: "Remove primary source from secondary sources list",
				Severity:   "ERROR",
			}
		}
		seen[source] = true
	}

	// Validate correlation window format
	if config.CorrelationWindow != "" {
		validWindows := []string{"1_minute", "5_minutes", "15_minutes", "30_minutes", "1_hour", "2_hours", "6_hours", "12_hours", "24_hours"}
		if !v.isValueInSlice(config.CorrelationWindow, validWindows) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid correlation_window format",
				Field:      "multi_source.correlation_window",
				Expected:   strings.Join(validWindows, ", "),
				Actual:     config.CorrelationWindow,
				Suggestion: "Use one of the valid time window formats",
				Severity:   "ERROR",
			}
		}
	}

	// Validate correlation fields
	validCorrelationFields := []string{"user", "source_ip", "user_agent", "timestamp", "namespace", "verb", "resource"}
	for i, field := range config.CorrelationFields {
		if !v.isValueInSlice(field, validCorrelationFields) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid correlation field",
				Field:      fmt.Sprintf("multi_source.correlation_fields[%d]", i),
				Expected:   strings.Join(validCorrelationFields, ", "),
				Actual:     field,
				Suggestion: "Use one of the valid correlation fields",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// ValidateAdvancedAnalysis validates advanced analysis configuration
func (v *SchemaValidator) ValidateAdvancedAnalysis(config *types.AdvancedAnalysisConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate analysis type (required)
	if config.Type == "" {
		return &ValidationError{
			Code:       "FIELD_REQUIRED",
			Message:    "analysis type is required",
			Field:      "analysis.type",
			Suggestion: "Specify one of the valid analysis types",
			Severity:   "ERROR",
		}
	}

	if !v.isValueInSlice(config.Type, v.validAnalysisTypes) {
		return &ValidationError{
			Code:       "FIELD_ENUM",
			Message:    "invalid analysis type",
			Field:      "analysis.type",
			Expected:   strings.Join(v.validAnalysisTypes, ", "),
			Actual:     config.Type,
			Suggestion: "Use one of the valid analysis types",
			Severity:   "ERROR",
		}
	}

	// Validate kill chain phase for APT analysis types
	aptTypes := []string{"apt_reconnaissance_detection", "apt_lateral_movement_detection", "apt_data_exfiltration_detection"}
	if v.isValueInSlice(config.Type, aptTypes) && config.KillChainPhase == "" {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "kill_chain_phase is required for APT analysis types",
			Field:      "analysis.kill_chain_phase",
			Suggestion: "Specify a kill chain phase: reconnaissance, weaponization, delivery, exploitation, installation, command_control, actions_objectives",
			Severity:   "ERROR",
		}
	}

	// Validate kill chain phase values
	if config.KillChainPhase != "" {
		validPhases := []string{"reconnaissance", "weaponization", "delivery", "exploitation", "installation", "command_control", "actions_objectives"}
		if !v.isValueInSlice(config.KillChainPhase, validPhases) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid kill chain phase",
				Field:      "analysis.kill_chain_phase",
				Expected:   strings.Join(validPhases, ", "),
				Actual:     config.KillChainPhase,
				Suggestion: "Use one of the valid kill chain phases",
				Severity:   "ERROR",
			}
		}
	}

	// Validate statistical analysis parameters
	if config.StatisticalAnalysis != nil {
		if err := v.validateStatisticalAnalysis(config.StatisticalAnalysis); err != nil {
			return err
		}
	}

	return nil
}

// ValidateBehavioralAnalysis validates behavioral analysis configuration
func (v *SchemaValidator) ValidateBehavioralAnalysis(config *types.BehavioralAnalysisConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate baseline window format
	if config.BaselineWindow != "" {
		validWindows := []string{"7_days", "14_days", "30_days", "60_days", "90_days"}
		if !v.isValueInSlice(config.BaselineWindow, validWindows) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid baseline window",
				Field:      "behavioral_analysis.baseline_window",
				Expected:   strings.Join(validWindows, ", "),
				Actual:     config.BaselineWindow,
				Suggestion: "Use one of the valid baseline windows",
				Severity:   "ERROR",
			}
		}
	}

	// Validate risk scoring dependency
	if config.RiskScoring != nil && !config.UserProfiling {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "risk_scoring requires user_profiling to be enabled",
			Field:      "behavioral_analysis.risk_scoring",
			Suggestion: "Enable user_profiling when using risk_scoring",
			Severity:   "ERROR",
		}
	}

	return nil
}

// ValidateDetectionCriteria validates detection criteria configuration
func (v *SchemaValidator) ValidateDetectionCriteria(config *types.DetectionCriteriaConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate rapid operations detection
	if config.RapidOperations != nil {
		if config.RapidOperations.Threshold <= 0 {
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    "rapid operations threshold must be positive",
				Field:      "detection_criteria.rapid_operations.threshold",
				Expected:   "positive integer",
				Actual:     strconv.Itoa(config.RapidOperations.Threshold),
				Suggestion: "Set threshold to a positive integer",
				Severity:   "ERROR",
			}
		}

		if config.RapidOperations.TimeWindow != "" {
			validWindows := []string{"30_seconds", "1_minute", "5_minutes", "15_minutes", "30_minutes", "1_hour"}
			if !v.isValueInSlice(config.RapidOperations.TimeWindow, validWindows) {
				return &ValidationError{
					Code:       "FIELD_ENUM",
					Message:    "invalid time window for rapid operations",
					Field:      "detection_criteria.rapid_operations.time_window",
					Expected:   strings.Join(validWindows, ", "),
					Actual:     config.RapidOperations.TimeWindow,
					Suggestion: "Use one of the valid time windows",
					Severity:   "ERROR",
				}
			}
		}
	}

	return nil
}

// ValidateComplianceFramework validates compliance framework configuration
func (v *SchemaValidator) ValidateComplianceFramework(config *types.ComplianceFrameworkConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate compliance standards
	validStandards := []string{"SOX", "PCI-DSS", "GDPR", "HIPAA", "ISO27001", "NIST", "FedRAMP"}
	for i, standard := range config.Standards {
		if !v.isValueInSlice(standard, validStandards) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid compliance standard",
				Field:      fmt.Sprintf("compliance_framework.standards[%d]", i),
				Expected:   strings.Join(validStandards, ", "),
				Actual:     standard,
				Suggestion: "Use one of the valid compliance standards",
				Severity:   "ERROR",
			}
		}
	}

	// Validate controls mapping
	validControls := []string{"access_logging", "data_protection", "audit_trail", "user_authentication", "authorization", "data_encryption", "incident_response"}
	for i, control := range config.Controls {
		if !v.isValueInSlice(control, validControls) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid compliance control",
				Field:      fmt.Sprintf("compliance_framework.controls[%d]", i),
				Expected:   strings.Join(validControls, ", "),
				Actual:     control,
				Suggestion: "Use one of the valid compliance controls",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// ValidateTemporalAnalysis validates temporal analysis configuration
func (v *SchemaValidator) ValidateTemporalAnalysis(config *types.TemporalAnalysisConfig) error {
	if config == nil {
		return nil // Optional field
	}

	// Validate pattern type
	if config.PatternType != "" {
		validTypes := []string{"periodic", "irregular", "trending", "cyclical", "seasonal"}
		if !v.isValueInSlice(config.PatternType, validTypes) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid pattern type",
				Field:      "temporal_analysis.pattern_type",
				Expected:   strings.Join(validTypes, ", "),
				Actual:     config.PatternType,
				Suggestion: "Use one of the valid pattern types",
				Severity:   "ERROR",
			}
		}
	}

	// Validate anomaly threshold
	if config.AnomalyThreshold < 0.1 || config.AnomalyThreshold > 10.0 {
		if config.AnomalyThreshold != 0.0 { // Allow 0.0 as unset value
			return &ValidationError{
				Code:       "FIELD_RANGE",
				Message:    "anomaly threshold out of range",
				Field:      "temporal_analysis.anomaly_threshold",
				Expected:   "0.1-10.0",
				Actual:     fmt.Sprintf("%.2f", config.AnomalyThreshold),
				Suggestion: "Set anomaly threshold between 0.1 and 10.0",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// Placeholder validation methods for types not yet fully defined
func (v *SchemaValidator) ValidateThreatIntelligence(config *types.ThreatIntelligenceConfig) error {
	// TODO: Implement when ThreatIntelligenceConfig is fully defined
	return nil
}

func (v *SchemaValidator) ValidateMachineLearning(config *types.MachineLearningConfig) error {
	// TODO: Implement when MachineLearningConfig is fully defined
	return nil
}

func (v *SchemaValidator) ValidateSecurityContext(config *types.SecurityContextConfig) error {
	if config == nil {
		return nil
	}

	// Validate pod security standards
	if config.PodSecurityStandards != "" {
		validStandards := []string{"privileged", "baseline", "restricted"}
		if !v.isValueInSlice(config.PodSecurityStandards, validStandards) {
			return &ValidationError{
				Code:       "FIELD_ENUM",
				Message:    "invalid pod security standard",
				Field:      "security_context.pod_security_standards",
				Expected:   strings.Join(validStandards, ", "),
				Actual:     config.PodSecurityStandards,
				Suggestion: "Use one of the valid pod security standards",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// validateStatisticalAnalysis validates statistical analysis parameters
func (v *SchemaValidator) validateStatisticalAnalysis(config *types.StatisticalAnalysisConfig) error {
	// Validate pattern deviation threshold
	if config.PatternDeviationThreshold < 0.1 || config.PatternDeviationThreshold > 10.0 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "pattern deviation threshold out of range",
			Field:      "analysis.statistical_analysis.pattern_deviation_threshold",
			Expected:   "0.1-10.0",
			Actual:     fmt.Sprintf("%.2f", config.PatternDeviationThreshold),
			Suggestion: "Set pattern deviation threshold between 0.1 and 10.0",
			Severity:   "ERROR",
		}
	}

	// Validate confidence interval
	if config.ConfidenceInterval < 0.5 || config.ConfidenceInterval > 0.99 {
		return &ValidationError{
			Code:       "FIELD_RANGE",
			Message:    "confidence interval out of range",
			Field:      "analysis.statistical_analysis.confidence_interval",
			Expected:   "0.5-0.99",
			Actual:     fmt.Sprintf("%.2f", config.ConfidenceInterval),
			Suggestion: "Set confidence interval between 0.5 and 0.99",
			Severity:   "ERROR",
		}
	}

	return nil
}

// =============================================================================
// CROSS-FIELD DEPENDENCY VALIDATION
// =============================================================================

// validateLogSourceCompatibility validates field compatibility with log sources
func (v *SchemaValidator) validateLogSourceCompatibility(q *types.StructuredQuery) error {
	logSource := q.LogSource

	// Compatibility matrix validation
	
	// node-auditd incompatibilities
	if logSource == "node-auditd" {
		if q.Verb.GetValue() != nil {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "verb field not applicable to node-auditd log source",
				Field:      "verb",
				Suggestion: "Remove verb field when using node-auditd log source",
				Severity:   "ERROR",
			}
		}

		if q.Resource.GetValue() != nil {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "resource field not applicable to node-auditd log source",
				Field:      "resource",
				Suggestion: "Remove resource field when using node-auditd log source",
				Severity:   "ERROR",
			}
		}

		if q.AuthDecision != "" {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "auth_decision field not applicable to node-auditd log source",
				Field:      "auth_decision",
				Suggestion: "Remove auth_decision field when using node-auditd log source",
				Severity:   "ERROR",
			}
		}
	}

	// oauth-server and oauth-apiserver incompatibilities
	if logSource == "oauth-server" || logSource == "oauth-apiserver" {
		if q.Resource.GetValue() != nil && logSource == "oauth-server" {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "resource field not applicable to oauth-server log source",
				Field:      "resource",
				Suggestion: "Remove resource field when using oauth-server log source",
				Severity:   "ERROR",
			}
		}
	}

	// kube-apiserver and openshift-apiserver incompatibilities  
	if logSource == "kube-apiserver" || logSource == "openshift-apiserver" {
		if q.AuthDecision != "" {
			return &ValidationError{
				Code:       "FIELD_CONFLICT",
				Message:    "auth_decision field not applicable to " + logSource + " log source",
				Field:      "auth_decision",
				Suggestion: "Remove auth_decision field when using " + logSource + " log source",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// validateAnalysisDependencies validates analysis field dependencies
func (v *SchemaValidator) validateAnalysisDependencies(q *types.StructuredQuery) error {
	if q.Analysis == nil {
		return nil
	}

	// APT analysis requires kill_chain_phase
	aptTypes := []string{"apt_reconnaissance_detection", "apt_lateral_movement_detection", "apt_data_exfiltration_detection"}
	if v.isValueInSlice(q.Analysis.Type, aptTypes) && q.Analysis.KillChainPhase == "" {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "kill_chain_phase is required for APT analysis types",
			Field:      "analysis.kill_chain_phase",
			Suggestion: "Specify a kill chain phase when using APT analysis types",
			Severity:   "ERROR",
		}
	}

	// Statistical analysis dependencies
	if q.Analysis.StatisticalAnalysis != nil {
		statTypes := []string{"statistical_analysis", "anomaly_detection", "behavioral_analysis"}
		if !v.isValueInSlice(q.Analysis.Type, statTypes) {
			return &ValidationError{
				Code:       "FIELD_DEPENDENCY",
				Message:    "statistical_analysis requires compatible analysis type",
				Field:      "analysis.statistical_analysis",
				Expected:   "analysis.type must be one of: " + fmt.Sprintf("%v", statTypes),
				Actual:     q.Analysis.Type,
				Suggestion: "Use a statistical analysis type or remove statistical_analysis config",
				Severity:   "ERROR",
			}
		}
	}

	return nil
}

// validateBehavioralAnalysisDependencies validates behavioral analysis dependencies
func (v *SchemaValidator) validateBehavioralAnalysisDependencies(q *types.StructuredQuery) error {
	if q.BehavioralAnalysis == nil {
		return nil
	}

	// Risk scoring requires user profiling
	if q.BehavioralAnalysis.RiskScoring != nil && !q.BehavioralAnalysis.UserProfiling {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "risk_scoring requires user_profiling to be enabled",
			Field:      "behavioral_analysis.risk_scoring",
			Suggestion: "Enable user_profiling when using risk_scoring",
			Severity:   "ERROR",
		}
	}

	// Anomaly detection requires baseline
	if q.BehavioralAnalysis.AnomalyDetection != nil && q.BehavioralAnalysis.BaselineWindow == "" {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "anomaly_detection requires baseline_window to be specified",
			Field:      "behavioral_analysis.baseline_window",
			Suggestion: "Specify a baseline window when using anomaly detection",
			Severity:   "ERROR",
		}
	}

	return nil
}

// validateMachineLearningDependencies validates machine learning dependencies
func (v *SchemaValidator) validateMachineLearningDependencies(q *types.StructuredQuery) error {
	if q.MachineLearning == nil {
		return nil
	}

	// Feature engineering requires model type
	if q.MachineLearning.FeatureEngineering != nil && q.MachineLearning.ModelType == "" {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "feature_engineering requires model_type to be specified",
			Field:      "machine_learning.model_type",
			Suggestion: "Specify a model type when using feature engineering",
			Severity:   "ERROR",
		}
	}

	return nil
}

// validateThreatIntelligenceDependencies validates threat intelligence dependencies
func (v *SchemaValidator) validateThreatIntelligenceDependencies(q *types.StructuredQuery) error {
	if q.ThreatIntelligence == nil {
		return nil
	}

	// IOC correlation requires feed sources
	if q.ThreatIntelligence.IOCCorrelation && len(q.ThreatIntelligence.FeedSources) == 0 {
		return &ValidationError{
			Code:       "FIELD_DEPENDENCY",
			Message:    "ioc_correlation requires feed_sources to be specified",
			Field:      "threat_intelligence.feed_sources",
			Suggestion: "Specify threat intelligence feed sources when using IOC correlation",
			Severity:   "ERROR",
		}
	}

	return nil
}

// =============================================================================
// QUERY COMPLEXITY CALCULATION
// =============================================================================

// calculateQueryComplexity calculates the complexity score of a query
func (v *SchemaValidator) calculateQueryComplexity(q *types.StructuredQuery) *QueryComplexity {
	complexity := &QueryComplexity{
		Score:         0,
		Components:    make(map[string]int),
		ResourceUsage: make(map[string]interface{}),
	}

	// Basic fields: 1 point each
	if q.Verb.GetValue() != nil {
		complexity.Score += 1
		complexity.Components["verb"] = 1
	}
	if q.Resource.GetValue() != nil {
		complexity.Score += 1
		complexity.Components["resource"] = 1
	}
	if q.Namespace.GetValue() != nil {
		complexity.Score += 1
		complexity.Components["namespace"] = 1
	}
	if q.User.GetValue() != nil {
		complexity.Score += 1
		complexity.Components["user"] = 1
	}
	if q.Timeframe != "" {
		complexity.Score += 1
		complexity.Components["timeframe"] = 1
	}
	if q.TimeRange != nil {
		complexity.Score += 2
		complexity.Components["time_range"] = 2
	}

	// Pattern matching: 3 points each
	if q.UserPattern != "" {
		complexity.Score += 3
		complexity.Components["user_pattern"] = 3
	}
	if q.NamespacePattern != "" {
		complexity.Score += 3
		complexity.Components["namespace_pattern"] = 3
	}
	if q.ResourceNamePattern != "" {
		complexity.Score += 3
		complexity.Components["resource_name_pattern"] = 3
	}
	if q.RequestURIPattern != "" {
		complexity.Score += 3
		complexity.Components["request_uri_pattern"] = 3
	}

	// Multi-source correlation: 5 points
	if q.MultiSource != nil {
		complexity.Score += 5
		complexity.Components["multi_source"] = 5
		// Additional points for each secondary source
		complexity.Score += len(q.MultiSource.SecondarySources)
		complexity.Components["secondary_sources"] = len(q.MultiSource.SecondarySources)
	}

	// Advanced analysis: 10 points
	if q.Analysis != nil {
		complexity.Score += 10
		complexity.Components["analysis"] = 10
		
		// Statistical analysis adds complexity
		if q.Analysis.StatisticalAnalysis != nil {
			complexity.Score += 5
			complexity.Components["statistical_analysis"] = 5
		}
	}

	// Behavioral analysis: 8 points
	if q.BehavioralAnalysis != nil {
		complexity.Score += 8
		complexity.Components["behavioral_analysis"] = 8
		
		if q.BehavioralAnalysis.RiskScoring != nil {
			complexity.Score += 3
			complexity.Components["risk_scoring"] = 3
		}
	}

	// Machine learning: 15 points
	if q.MachineLearning != nil {
		complexity.Score += 15
		complexity.Components["machine_learning"] = 15
	}

	// Threat intelligence: 12 points
	if q.ThreatIntelligence != nil {
		complexity.Score += 12
		complexity.Components["threat_intelligence"] = 12
	}

	// Detection criteria: 6 points
	if q.DetectionCriteria != nil {
		complexity.Score += 6
		complexity.Components["detection_criteria"] = 6
	}

	// Security context: 4 points
	if q.SecurityContext != nil {
		complexity.Score += 4
		complexity.Components["security_context"] = 4
	}

	// Compliance framework: 7 points
	if q.ComplianceFramework != nil {
		complexity.Score += 7
		complexity.Components["compliance_framework"] = 7
	}

	// Temporal analysis: 9 points
	if q.TemporalAnalysis != nil {
		complexity.Score += 9
		complexity.Components["temporal_analysis"] = 9
	}

	// High limit adds complexity
	if q.Limit > 100 {
		complexity.Score += 2
		complexity.Components["high_limit"] = 2
	}

	// Determine complexity level
	if complexity.Score < v.complexityThresholds["low"] {
		complexity.Level = "Low"
	} else if complexity.Score < v.complexityThresholds["medium"] {
		complexity.Level = "Medium"
	} else {
		complexity.Level = "High"
	}

	// Generate performance warnings
	if complexity.Level == "High" {
		complexity.PerformanceWarnings = append(complexity.PerformanceWarnings,
			"High complexity query may experience slow execution")
	}

	if q.Limit > 500 {
		complexity.PerformanceWarnings = append(complexity.PerformanceWarnings,
			"Large result limit may impact performance")
	}

	if q.MultiSource != nil && len(q.MultiSource.SecondarySources) > 2 {
		complexity.PerformanceWarnings = append(complexity.PerformanceWarnings,
			"Multi-source correlation across many sources may be slow")
	}

	// Estimate resource usage
	complexity.ResourceUsage["estimated_memory_mb"] = v.estimateMemoryUsage(q)
	complexity.ResourceUsage["estimated_cpu_cores"] = v.estimateCPUUsage(q)
	complexity.ResourceUsage["estimated_network_mb"] = v.estimateNetworkUsage(q)

	return complexity
}

// estimateMemoryUsage estimates memory usage based on query complexity
func (v *SchemaValidator) estimateMemoryUsage(q *types.StructuredQuery) int {
	baseMemory := 10 // Base 10 MB

	// Add memory for result set
	baseMemory += q.Limit / 10 // 1 MB per 10 results

	// Add memory for complex analysis
	if q.Analysis != nil {
		baseMemory += 50
	}
	if q.MachineLearning != nil {
		baseMemory += 200
	}
	if q.BehavioralAnalysis != nil {
		baseMemory += 75
	}
	if q.MultiSource != nil {
		baseMemory += len(q.MultiSource.SecondarySources) * 25
	}

	return baseMemory
}

// estimateCPUUsage estimates CPU usage based on query complexity
func (v *SchemaValidator) estimateCPUUsage(q *types.StructuredQuery) float64 {
	baseCPU := 0.1 // Base 0.1 cores

	// Add CPU for pattern matching
	if q.UserPattern != "" || q.NamespacePattern != "" || q.ResourceNamePattern != "" || q.RequestURIPattern != "" {
		baseCPU += 0.2
	}

	// Add CPU for complex analysis
	if q.Analysis != nil {
		baseCPU += 0.5
		if q.Analysis.StatisticalAnalysis != nil {
			baseCPU += 0.3
		}
	}
	if q.MachineLearning != nil {
		baseCPU += 2.0
	}
	if q.BehavioralAnalysis != nil {
		baseCPU += 0.8
	}
	if q.MultiSource != nil {
		baseCPU += float64(len(q.MultiSource.SecondarySources)) * 0.3
	}

	return baseCPU
}

// estimateNetworkUsage estimates network usage based on query complexity
func (v *SchemaValidator) estimateNetworkUsage(q *types.StructuredQuery) int {
	baseNetwork := 1 // Base 1 MB

	// Add network for multi-source correlation
	if q.MultiSource != nil {
		baseNetwork += len(q.MultiSource.SecondarySources) * 5
	}

	// Add network for large result sets
	baseNetwork += q.Limit / 100 // 1 MB per 100 results

	// Add network for threat intelligence feeds
	if q.ThreatIntelligence != nil && q.ThreatIntelligence.IOCCorrelation {
		baseNetwork += 10
	}

	return baseNetwork
}

// GetQueryComplexity returns the complexity analysis for a query (public method for external use)
func (v *SchemaValidator) GetQueryComplexity(q *types.StructuredQuery) *QueryComplexity {
	return v.calculateQueryComplexity(q)
}

// =============================================================================
// HELPER UTILITY METHODS
// =============================================================================

// isValidLogSource checks if the log source is valid
func (v *SchemaValidator) isValidLogSource(source string) bool {
	for _, valid := range v.validLogSources {
		if valid == source {
			return true
		}
	}
	return false
}

// isValidTimeframe checks if the timeframe is valid
func (v *SchemaValidator) isValidTimeframe(timeframe string) bool {
	for _, valid := range v.validTimeframes {
		if valid == timeframe {
			return true
		}
	}
	return false
}

// isValidAuthDecision checks if the auth decision is valid
func (v *SchemaValidator) isValidAuthDecision(decision string) bool {
	for _, valid := range v.validAuthDecisions {
		if valid == decision {
			return true
		}
	}
	return false
}

// isValueInSlice checks if a value exists in a slice
func (v *SchemaValidator) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}