package rules

import (
	"fmt"
	"strings"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ComplianceRule implements validation for compliance framework requirements
type ComplianceRule struct {
	config  map[string]interface{}
	enabled bool
}

// NewComplianceRule creates a new compliance validation rule
func NewComplianceRule(config map[string]interface{}) *ComplianceRule {
	return &ComplianceRule{
		config:  config,
		enabled: true,
	}
}

// Validate applies compliance framework validation to the query
func (r *ComplianceRule) Validate(query *types.StructuredQuery) *interfaces.ValidationResult {
	result := &interfaces.ValidationResult{
		IsValid:         true,
		RuleName:        "compliance_validation",
		Severity:        "info",
		Message:         "Compliance validation passed",
		Details:         make(map[string]interface{}),
		Recommendations: []string{},
		Warnings:        []string{},
		Errors:          []string{},
		Timestamp:       time.Now().Format(time.RFC3339),
		QuerySnapshot:   query,
	}

	// Skip validation if no compliance framework configuration
	if query.ComplianceFramework == nil {
		return result
	}

	// Validate compliance framework configuration
	r.validateStandards(query.ComplianceFramework, result)
	r.validateControls(query.ComplianceFramework, result)
	r.validateRetentionRequirements(query, result)
	r.validateEvidenceRequirements(query.ComplianceFramework, result)
	r.validateAuditTrailRequirements(query, result)
	r.validateReportingRequirements(query.ComplianceFramework, result)
	r.validateStandardSpecificRequirements(query.ComplianceFramework, query, result)

	// Update message based on validation result
	if !result.IsValid {
		result.Message = "Compliance validation failed"
		result.Severity = "critical"
		result.Recommendations = append(result.Recommendations,
			"Review compliance framework configuration",
			"Ensure all required standards are supported",
			"Verify evidence fields meet compliance requirements",
			"Check retention periods comply with regulations",
			"Validate audit trail completeness")
	} else if len(result.Warnings) > 0 {
		result.Severity = "warning"
		result.Message = "Compliance validation passed with warnings"
	}

	return result
}

// validateStandards validates compliance standards
func (r *ComplianceRule) validateStandards(config *types.ComplianceFrameworkConfig, result *interfaces.ValidationResult) {
	if len(config.Standards) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "At least one compliance standard must be specified")
		return
	}

	allowedStandards := r.getAllowedStandards()
	maxStandards := r.getMaxStandards()

	if len(config.Standards) > maxStandards {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Too many compliance standards. Maximum allowed: %d, got: %d",
				maxStandards, len(config.Standards)))
	}

	seenStandards := make(map[string]bool)
	for i, standard := range config.Standards {
		// Check if standard is valid
		if !r.isValueInSlice(standard, allowedStandards) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid compliance standard '%s' at index %d. Allowed standards: %s",
					standard, i, strings.Join(allowedStandards, ", ")))
			continue
		}

		// Check for duplicates
		if seenStandards[standard] {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Duplicate compliance standard '%s' at index %d", standard, i))
			continue
		}
		seenStandards[standard] = true
	}
}

// validateControls validates compliance controls
func (r *ComplianceRule) validateControls(config *types.ComplianceFrameworkConfig, result *interfaces.ValidationResult) {
	if len(config.Controls) == 0 {
		result.Warnings = append(result.Warnings, "No compliance controls specified")
		return
	}

	allowedControls := r.getAllowedControls()
	maxControls := r.getMaxControls()

	if len(config.Controls) > maxControls {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Too many compliance controls. Maximum allowed: %d, got: %d",
				maxControls, len(config.Controls)))
	}

	// Validate each control
	seenControls := make(map[string]bool)
	for i, control := range config.Controls {
		// Check if control is valid
		if !r.isValueInSlice(control, allowedControls) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid compliance control '%s' at index %d. Allowed controls: %s",
					control, i, strings.Join(allowedControls, ", ")))
			continue
		}

		// Check for duplicates
		if seenControls[control] {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Duplicate compliance control '%s' at index %d", control, i))
			continue
		}
		seenControls[control] = true
	}

	// Validate control-standard compatibility
	r.validateControlStandardCompatibility(config, result)
}

// validateRetentionRequirements validates data retention requirements
func (r *ComplianceRule) validateRetentionRequirements(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	config := query.ComplianceFramework
	minRetentionDays := r.getMinRetentionDays()

	// Check if query timeframe meets retention requirements
	queryTimeframeDays := r.calculateQueryTimeframeDays(query)
	if queryTimeframeDays > minRetentionDays {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Query timeframe %d days exceeds minimum retention requirement %d days",
				queryTimeframeDays, minRetentionDays))
	}

	// Validate specific standard retention requirements
	for _, standard := range config.Standards {
		standardRetention := r.getStandardRetentionRequirement(standard)
		if queryTimeframeDays > standardRetention {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Query timeframe %d days may not meet %s retention requirement %d days",
					queryTimeframeDays, standard, standardRetention))
		}
	}
}

// validateEvidenceRequirements validates evidence collection requirements
func (r *ComplianceRule) validateEvidenceRequirements(config *types.ComplianceFrameworkConfig, result *interfaces.ValidationResult) {
	requiredEvidenceFields := r.getRequiredEvidenceFields()
	
	// Check if reporting includes evidence
	if config.Reporting != nil && !config.Reporting.IncludeEvidence {
		result.Warnings = append(result.Warnings,
			"Evidence collection is disabled but may be required for compliance")
	}

	// For queries that might need evidence, recommend enabling it
	if config.Reporting == nil {
		result.Warnings = append(result.Warnings,
			"No reporting configuration specified. Evidence collection recommended for compliance")
	}

	// Add evidence field requirements to details
	result.Details["required_evidence_fields"] = requiredEvidenceFields
}

// validateAuditTrailRequirements validates audit trail completeness
func (r *ComplianceRule) validateAuditTrailRequirements(query *types.StructuredQuery, result *interfaces.ValidationResult) {
	config := query.ComplianceFramework
	
	// Check for audit gap concerns
	maxAuditGapHours := r.getMaxAuditGapHours()
	if maxAuditGapHours > 0 {
		result.Details["max_audit_gap_hours"] = maxAuditGapHours
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Ensure audit trail has no gaps exceeding %d hours", maxAuditGapHours))
	}

	// Validate required audit fields are captured
	auditFields := r.getRequiredAuditFields()
	if len(auditFields) > 0 {
		result.Details["required_audit_fields"] = auditFields
	}

	// Check for comprehensive logging requirements
	for _, standard := range config.Standards {
		if r.requiresComprehensiveLogging(standard) {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Standard %s requires comprehensive audit logging", standard))
		}
	}
}

// validateReportingRequirements validates reporting configuration
func (r *ComplianceRule) validateReportingRequirements(config *types.ComplianceFrameworkConfig, result *interfaces.ValidationResult) {
	if config.Reporting == nil {
		result.Warnings = append(result.Warnings, "No reporting configuration specified")
		return
	}

	reporting := config.Reporting

	// Validate reporting format
	if reporting.Format != "" {
		allowedFormats := []string{"detailed", "summary", "executive", "technical"}
		if !r.isValueInSlice(reporting.Format, allowedFormats) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid reporting format '%s'. Allowed formats: %s",
					reporting.Format, strings.Join(allowedFormats, ", ")))
		}
	}

	// Validate evidence inclusion requirements
	if reporting.IncludeEvidence {
		result.Recommendations = append(result.Recommendations,
			"Evidence collection enabled - ensure adequate storage and retention")
	}

	// Standard-specific reporting requirements
	for _, standard := range config.Standards {
		r.validateStandardReportingRequirements(standard, reporting, result)
	}
}

// validateStandardSpecificRequirements validates requirements specific to each compliance standard
func (r *ComplianceRule) validateStandardSpecificRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	for _, standard := range config.Standards {
		switch standard {
		case "SOX":
			r.validateSOXRequirements(config, query, result)
		case "PCI-DSS":
			r.validatePCIDSSRequirements(config, query, result)
		case "GDPR":
			r.validateGDPRRequirements(config, query, result)
		case "HIPAA":
			r.validateHIPAARequirements(config, query, result)
		case "ISO27001":
			r.validateISO27001Requirements(config, query, result)
		case "NIST":
			r.validateNISTRequirements(config, query, result)
		case "CIS":
			r.validateCISRequirements(config, query, result)
		case "FedRAMP":
			r.validateFedRAMPRequirements(config, query, result)
		}
	}
}

// Standard-specific validation methods
func (r *ComplianceRule) validateSOXRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// SOX requires comprehensive financial system audit trails
	requiredControls := []string{"access_logging", "change_management", "audit_trail"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("SOX compliance typically requires '%s' control", control))
		}
	}

	// SOX requires 7+ years retention
	if r.calculateQueryTimeframeDays(query) > 2555 { // ~7 years
		result.Warnings = append(result.Warnings,
			"Query timeframe may exceed SOX 7-year retention requirement")
	}
}

func (r *ComplianceRule) validatePCIDSSRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// PCI-DSS requires specific access controls and monitoring
	requiredControls := []string{"access_logging", "authentication_monitoring", "data_protection"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("PCI-DSS compliance typically requires '%s' control", control))
		}
	}

	// PCI-DSS requires 1+ years retention
	if r.calculateQueryTimeframeDays(query) > 365 {
		result.Warnings = append(result.Warnings,
			"Query timeframe may exceed PCI-DSS 1-year retention requirement")
	}
}

func (r *ComplianceRule) validateGDPRRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// GDPR requires privacy and data protection controls
	requiredControls := []string{"data_protection", "access_logging", "audit_trail"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("GDPR compliance typically requires '%s' control", control))
		}
	}

	// GDPR has specific data retention limitations
	result.Recommendations = append(result.Recommendations,
		"GDPR requires data minimization - ensure query scope is necessary and proportionate")
}

func (r *ComplianceRule) validateHIPAARequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// HIPAA requires comprehensive audit controls for healthcare data
	requiredControls := []string{"access_logging", "audit_trail", "data_protection", "authentication_monitoring"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("HIPAA compliance typically requires '%s' control", control))
		}
	}

	// HIPAA requires 6+ years retention
	if r.calculateQueryTimeframeDays(query) > 2190 { // ~6 years
		result.Warnings = append(result.Warnings,
			"Query timeframe may exceed HIPAA 6-year retention requirement")
	}
}

func (r *ComplianceRule) validateISO27001Requirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// ISO 27001 requires comprehensive information security management
	requiredControls := []string{"access_logging", "incident_response", "vulnerability_management", "configuration_management"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("ISO 27001 compliance typically requires '%s' control", control))
		}
	}
}

func (r *ComplianceRule) validateNISTRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// NIST requires comprehensive cybersecurity framework controls
	requiredControls := []string{"access_logging", "incident_response", "vulnerability_management", "audit_trail"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("NIST compliance typically requires '%s' control", control))
		}
	}
}

func (r *ComplianceRule) validateCISRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// CIS requires specific security controls
	requiredControls := []string{"access_logging", "configuration_management", "vulnerability_management"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("CIS compliance typically requires '%s' control", control))
		}
	}
}

func (r *ComplianceRule) validateFedRAMPRequirements(config *types.ComplianceFrameworkConfig, query *types.StructuredQuery, result *interfaces.ValidationResult) {
	// FedRAMP requires comprehensive federal security controls
	requiredControls := []string{"access_logging", "audit_trail", "incident_response", "configuration_management", "vulnerability_management"}
	for _, control := range requiredControls {
		if !r.isValueInSlice(control, config.Controls) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("FedRAMP compliance typically requires '%s' control", control))
		}
	}

	// FedRAMP requires continuous monitoring
	result.Recommendations = append(result.Recommendations,
		"FedRAMP requires continuous monitoring - ensure audit queries support ongoing compliance")
}

// Helper validation methods
func (r *ComplianceRule) validateControlStandardCompatibility(config *types.ComplianceFrameworkConfig, result *interfaces.ValidationResult) {
	controlStandardMap := r.getControlStandardCompatibility()
	
	for _, control := range config.Controls {
		compatibleStandards, exists := controlStandardMap[control]
		if !exists {
			continue // Control not in compatibility map
		}

		// Check if any of the specified standards are compatible with this control
		isCompatible := false
		for _, standard := range config.Standards {
			if r.isValueInSlice(standard, compatibleStandards) {
				isCompatible = true
				break
			}
		}

		if !isCompatible {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Control '%s' may not be directly applicable to standards: %s",
					control, strings.Join(config.Standards, ", ")))
		}
	}
}

func (r *ComplianceRule) validateStandardReportingRequirements(standard string, reporting *types.ComplianceReportingConfig, result *interfaces.ValidationResult) {
	switch standard {
	case "SOX":
		if reporting.Format != "detailed" {
			result.Warnings = append(result.Warnings,
				"SOX compliance typically requires detailed reporting format")
		}
	case "PCI-DSS":
		if !reporting.IncludeEvidence {
			result.Warnings = append(result.Warnings,
				"PCI-DSS compliance typically requires evidence collection")
		}
	case "GDPR":
		result.Recommendations = append(result.Recommendations,
			"GDPR requires data subject rights - ensure reports support data subject requests")
	case "HIPAA":
		if !reporting.IncludeEvidence {
			result.Warnings = append(result.Warnings,
				"HIPAA compliance typically requires comprehensive evidence collection")
		}
	}
}

// Utility methods
func (r *ComplianceRule) calculateQueryTimeframeDays(query *types.StructuredQuery) int {
	// Simplified calculation - in practice would parse timeframe more thoroughly
	timeframeDays := map[string]int{
		"today":        1,
		"yesterday":    2,
		"7_days_ago":   7,
		"14_days_ago":  14,
		"30_days_ago":  30,
		"60_days_ago":  60,
		"90_days_ago":  90,
		"last_week":    7,
		"last_month":   30,
	}

	if days, exists := timeframeDays[query.Timeframe]; exists {
		return days
	}

	// Default to 30 days if timeframe not recognized
	return 30
}

// Configuration retrieval methods
func (r *ComplianceRule) getAllowedStandards() []string {
	if r.config != nil {
		if standards, ok := r.config["allowed_standards"].([]interface{}); ok {
			result := make([]string, len(standards))
			for i, s := range standards {
				if str, ok := s.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{"SOX", "PCI-DSS", "GDPR", "HIPAA", "ISO27001", "NIST", "CIS", "FedRAMP"}
}

func (r *ComplianceRule) getAllowedControls() []string {
	if r.config != nil {
		if controls, ok := r.config["allowed_controls"].([]interface{}); ok {
			result := make([]string, len(controls))
			for i, c := range controls {
				if str, ok := c.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{
		"access_logging", "data_protection", "authentication_monitoring",
		"privilege_management", "audit_trail", "change_management",
		"incident_response", "vulnerability_management", "configuration_management",
		"business_continuity",
	}
}

func (r *ComplianceRule) getMaxStandards() int {
	if r.config != nil {
		if maxStandards, ok := r.config["max_standards"].(int); ok {
			return maxStandards
		}
	}
	return 5 // Default
}

func (r *ComplianceRule) getMaxControls() int {
	if r.config != nil {
		if maxControls, ok := r.config["max_controls"].(int); ok {
			return maxControls
		}
	}
	return 10 // Default
}

func (r *ComplianceRule) getMinRetentionDays() int {
	if r.config != nil {
		if minDays, ok := r.config["min_retention_days"].(int); ok {
			return minDays
		}
	}
	return 365 // Default 1 year
}

func (r *ComplianceRule) getMaxAuditGapHours() int {
	if r.config != nil {
		if maxGap, ok := r.config["max_audit_gap_hours"].(int); ok {
			return maxGap
		}
	}
	return 24 // Default 24 hours
}

func (r *ComplianceRule) getRequiredEvidenceFields() []string {
	if r.config != nil {
		if fields, ok := r.config["required_evidence_fields"].([]interface{}); ok {
			result := make([]string, len(fields))
			for i, f := range fields {
				if str, ok := f.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}

	return []string{"timestamp", "user", "action", "resource", "outcome"}
}

func (r *ComplianceRule) getRequiredAuditFields() []string {
	return []string{"timestamp", "user", "source_ip", "action", "resource", "outcome", "request_id"}
}

func (r *ComplianceRule) getStandardRetentionRequirement(standard string) int {
	retentionMap := map[string]int{
		"SOX":     2555, // ~7 years
		"PCI-DSS": 365,  // 1 year
		"GDPR":    1095, // 3 years (varies by data type)
		"HIPAA":   2190, // 6 years
		"ISO27001": 365, // 1 year minimum
		"NIST":    1095, // 3 years
		"CIS":     365,  // 1 year
		"FedRAMP": 1095, // 3 years
	}

	if days, exists := retentionMap[standard]; exists {
		return days
	}
	return 365 // Default 1 year
}

func (r *ComplianceRule) requiresComprehensiveLogging(standard string) bool {
	comprehensiveStandards := []string{"SOX", "HIPAA", "FedRAMP", "ISO27001"}
	return r.isValueInSlice(standard, comprehensiveStandards)
}

func (r *ComplianceRule) getControlStandardCompatibility() map[string][]string {
	return map[string][]string{
		"access_logging":           {"SOX", "PCI-DSS", "GDPR", "HIPAA", "ISO27001", "NIST", "CIS", "FedRAMP"},
		"data_protection":          {"PCI-DSS", "GDPR", "HIPAA", "ISO27001", "NIST"},
		"authentication_monitoring": {"PCI-DSS", "HIPAA", "ISO27001", "NIST", "FedRAMP"},
		"privilege_management":     {"SOX", "PCI-DSS", "ISO27001", "NIST", "CIS", "FedRAMP"},
		"audit_trail":             {"SOX", "GDPR", "HIPAA", "ISO27001", "NIST", "FedRAMP"},
		"change_management":        {"SOX", "ISO27001", "NIST", "CIS", "FedRAMP"},
		"incident_response":        {"ISO27001", "NIST", "CIS", "FedRAMP"},
		"vulnerability_management": {"PCI-DSS", "ISO27001", "NIST", "CIS", "FedRAMP"},
		"configuration_management": {"ISO27001", "NIST", "CIS", "FedRAMP"},
		"business_continuity":      {"ISO27001", "NIST", "FedRAMP"},
	}
}

func (r *ComplianceRule) isValueInSlice(value string, slice []string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Interface implementation methods
func (r *ComplianceRule) GetRuleName() string {
	return "compliance_validation"
}

func (r *ComplianceRule) GetRuleDescription() string {
	return "Validates compliance framework requirements including standards, controls, retention, and evidence collection"
}

func (r *ComplianceRule) IsEnabled() bool {
	return r.enabled
}

func (r *ComplianceRule) GetSeverity() string {
	return "critical"
}