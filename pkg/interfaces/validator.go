package interfaces

import (
	"genai-processing/pkg/types"
)

// SafetyValidator defines the interface for validating generated queries for safety and feasibility.
// This interface implements the whitelist approach for allowed commands and patterns,
// ensuring compliance with OpenShift security policies.
type SafetyValidator interface {
	// ValidateQuery validates a structured query for safety and feasibility.
	// This method applies all applicable validation rules to ensure the query
	// is safe to execute and complies with security policies.
	//
	// Parameters:
	//   - query: The structured query to validate
	//
	// Returns:
	//   - ValidationResult: The validation outcome with details and recommendations
	//   - error: Any error that occurred during validation
	ValidateQuery(query *types.StructuredQuery) (*ValidationResult, error)

	// GetApplicableRules returns all validation rules that are currently active.
	// This method provides visibility into the validation rules being applied,
	// which is useful for debugging, monitoring, and rule management.
	//
	// Returns:
	//   - []ValidationRule: List of all active validation rules
	GetApplicableRules() []ValidationRule
}

// ValidationRule defines the interface for individual validation rules.
// This interface allows for the implementation of specific validation logic
// for different aspects of query safety and feasibility.
type ValidationRule interface {
	// Validate applies the validation rule to a structured query.
	// This method implements the specific validation logic for this rule
	// and returns the validation outcome.
	//
	// Parameters:
	//   - query: The structured query to validate
	//
	// Returns:
	//   - ValidationResult: The validation outcome for this specific rule
	Validate(query *types.StructuredQuery) *ValidationResult

	// GetRuleName returns the unique identifier for this validation rule.
	// This method provides a human-readable name for the rule, which is
	// useful for logging, debugging, and rule management.
	//
	// Returns:
	//   - string: The unique name/identifier for this rule
	GetRuleName() string

	// GetRuleDescription returns a description of what this rule validates.
	// This method provides a clear explanation of the rule's purpose and
	// what aspects of the query it validates.
	//
	// Returns:
	//   - string: Description of the rule's validation purpose
	GetRuleDescription() string

	// IsEnabled indicates whether this validation rule is currently active.
	// This method allows for dynamic enabling/disabling of validation rules
	// without removing them from the validation pipeline.
	//
	// Returns:
	//   - bool: True if the rule is currently enabled
	IsEnabled() bool

	// GetSeverity returns the severity level of this validation rule.
	// This method indicates how critical the rule is for query safety,
	// which affects how validation failures are handled.
	//
	// Returns:
	//   - string: Severity level (e.g., "critical", "warning", "info")
	GetSeverity() string
}

// ValidationResult represents the outcome of a validation operation.
// This struct provides detailed information about validation results,
// including success status, warnings, errors, and recommendations.
type ValidationResult struct {
	// IsValid indicates whether the validation was successful.
	// This field provides a quick boolean check for validation success.
	IsValid bool `json:"is_valid"`

	// RuleName is the name of the validation rule that produced this result.
	// This field identifies which specific rule generated this validation outcome.
	RuleName string `json:"rule_name"`

	// Severity indicates the severity level of the validation result.
	// This field helps determine how to handle the validation outcome.
	Severity string `json:"severity"`

	// Message provides a human-readable description of the validation result.
	// This field explains what was validated and the outcome in plain language.
	Message string `json:"message"`

	// Details contains additional information about the validation result.
	// This field provides context-specific details that may be useful for
	// understanding or addressing the validation outcome.
	Details map[string]interface{} `json:"details,omitempty"`

	// Recommendations provides suggestions for addressing validation issues.
	// This field offers actionable advice for resolving validation problems
	// or improving query safety.
	Recommendations []string `json:"recommendations,omitempty"`

	// Warnings contains non-critical issues that don't prevent validation success.
	// This field lists potential concerns that should be reviewed but don't
	// constitute validation failures.
	Warnings []string `json:"warnings,omitempty"`

	// Errors contains critical issues that prevent validation success.
	// This field lists problems that must be resolved before the query
	// can be considered safe for execution.
	Errors []string `json:"errors,omitempty"`

	// Timestamp indicates when the validation was performed.
	// This field provides audit trail information for validation operations.
	Timestamp string `json:"timestamp"`

	// QuerySnapshot contains a snapshot of the query that was validated.
	// This field preserves the state of the query at validation time for
	// debugging and audit purposes.
	QuerySnapshot *types.StructuredQuery `json:"query_snapshot,omitempty"`
}
