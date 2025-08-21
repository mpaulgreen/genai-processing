package rules

import (
	"strings"
	"testing"

	"genai-processing/pkg/types"
)

// TestNewComprehensiveInputValidationRule tests the constructor
func TestNewComprehensiveInputValidationRule(t *testing.T) {
	// Test with nil config (should use defaults)
	rule := NewComprehensiveInputValidationRule(nil)
	
	if rule == nil {
		t.Fatal("NewComprehensiveInputValidationRule returned nil")
	}
	
	if rule.GetRuleName() != "comprehensive_input_validation" {
		t.Errorf("Expected rule name 'comprehensive_input_validation', got '%s'", rule.GetRuleName())
	}
	
	if !rule.IsEnabled() {
		t.Error("Rule should be enabled by default")
	}
	
	if rule.GetSeverity() != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", rule.GetSeverity())
	}
}

// TestNewComprehensiveInputValidationRule_WithConfig tests constructor with custom config
func TestNewComprehensiveInputValidationRule_WithConfig(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source", "verb"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxQueryLength:   5000,
			MaxPatternLength: 200,
			ForbiddenChars:   []string{"<", ">"},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{"rm -rf", "system:admin"},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{"kube-apiserver"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit: 25,
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	if rule == nil {
		t.Fatal("NewComprehensiveInputValidationRule returned nil")
	}
	
	if rule.config != config {
		t.Error("Rule should use provided config")
	}
}

// TestComprehensiveInputValidationRule_ValidateNilQuery tests validation with nil query
func TestComprehensiveInputValidationRule_ValidateNilQuery(t *testing.T) {
	rule := NewComprehensiveInputValidationRule(nil)
	
	result := rule.Validate(nil)
	
	if result == nil {
		t.Fatal("Validate returned nil result")
	}
	
	if result.IsValid {
		t.Error("Expected validation to fail for nil query")
	}
	
	if result.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", result.Severity)
	}
	
	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for nil query")
	}
}

// TestComprehensiveInputValidationRule_ValidateRequiredFields tests required fields validation
func TestComprehensiveInputValidationRule_ValidateRequiredFields(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source", "verb"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxPatternLength: 500,
			ForbiddenChars:   []string{},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{"kube-apiserver"},
			AllowedVerbs:      []string{"get", "list"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit: 50,
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
		description string
	}{
		{
			name: "valid_query_with_required_fields",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
			},
			expectValid: true,
			description: "Query with all required fields should pass",
		},
		{
			name: "missing_log_source",
			query: &types.StructuredQuery{
				Verb: *types.NewStringOrArray("get"),
			},
			expectValid: false,
			description: "Missing log_source should fail validation",
		},
		{
			name: "missing_verb",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			expectValid: false,
			description: "Missing verb should fail validation",
		},
		{
			name: "empty_log_source",
			query: &types.StructuredQuery{
				LogSource: "",
				Verb:      *types.NewStringOrArray("get"),
			},
			expectValid: false,
			description: "Empty log_source should fail validation",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)
			
			if result == nil {
				t.Fatal("Validate returned nil result")
			}
			
			if result.IsValid != tt.expectValid {
				t.Errorf("%s: Expected IsValid to be %v, got %v. Errors: %v", 
					tt.description, tt.expectValid, result.IsValid, result.Errors)
			}
			
			if !tt.expectValid && len(result.Errors) == 0 {
				t.Errorf("%s: Expected validation errors for invalid query", tt.description)
			}
		})
	}
}

// TestComprehensiveInputValidationRule_ValidateCharacters tests character validation
func TestComprehensiveInputValidationRule_ValidateCharacters(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxPatternLength: 20,
			ForbiddenChars:   []string{"<", ">", "&"},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{"kube-apiserver"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit: 50,
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
		description string
	}{
		{
			name: "valid_characters",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "valid-user",
			},
			expectValid: true,
			description: "Query with valid characters should pass",
		},
		{
			name: "forbidden_chars_in_user_pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "user<script>",
			},
			expectValid: false,
			description: "Forbidden characters should fail validation",
		},
		{
			name: "pattern_too_long",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "this-is-a-very-long-pattern-that-exceeds-the-limit",
			},
			expectValid: false,
			description: "Pattern exceeding length limit should fail validation",
		},
		{
			name: "multiple_forbidden_chars",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         "user&admin",
				NamespacePattern:    "ns>admin",
			},
			expectValid: false,
			description: "Multiple forbidden characters should fail validation",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)
			
			if result == nil {
				t.Fatal("Validate returned nil result")
			}
			
			if result.IsValid != tt.expectValid {
				t.Errorf("%s: Expected IsValid to be %v, got %v. Errors: %v", 
					tt.description, tt.expectValid, result.IsValid, result.Errors)
			}
		})
	}
}

// TestComprehensiveInputValidationRule_ValidateSecurityPatterns tests security pattern validation
func TestComprehensiveInputValidationRule_ValidateSecurityPatterns(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxPatternLength: 500,
			ForbiddenChars:   []string{},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{"system:admin", "cluster-admin", "rm -rf"},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{"kube-apiserver"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit: 50,
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
		description string
	}{
		{
			name: "safe_query",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "regular-user",
			},
			expectValid: true,
			description: "Safe query should pass",
		},
		{
			name: "dangerous_system_admin_pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "system:admin",
			},
			expectValid: false,
			description: "Dangerous system:admin pattern should fail",
		},
		{
			name: "dangerous_cluster_admin_pattern",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				NamespacePattern:    "cluster-admin",
			},
			expectValid: false,
			description: "Dangerous cluster-admin pattern should fail",
		},
		{
			name: "case_insensitive_pattern_matching",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "SYSTEM:ADMIN",
			},
			expectValid: false,
			description: "Case insensitive pattern matching should catch uppercase",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)
			
			if result == nil {
				t.Fatal("Validate returned nil result")
			}
			
			if result.IsValid != tt.expectValid {
				t.Errorf("%s: Expected IsValid to be %v, got %v. Errors: %v", 
					tt.description, tt.expectValid, result.IsValid, result.Errors)
			}
		})
	}
}

// TestComprehensiveInputValidationRule_ValidateFieldValues tests field value validation
func TestComprehensiveInputValidationRule_ValidateFieldValues(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxPatternLength: 500,
			ForbiddenChars:   []string{},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources:     []string{"kube-apiserver", "oauth-server"},
			AllowedVerbs:          []string{"get", "list", "create"},
			AllowedResources:      []string{"pods", "services"},
			AllowedAuthDecisions:  []string{"allow", "forbid"},
			AllowedResponseStatus: []string{"200", "401", "403"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit: 50,
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
		description string
	}{
		{
			name: "valid_field_values",
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				Verb:           *types.NewStringOrArray("get"),
				Resource:       *types.NewStringOrArray("pods"),
				AuthDecision:   "allow",
				ResponseStatus: *types.NewStringOrArray("200"),
			},
			expectValid: true,
			description: "Valid field values should pass",
		},
		{
			name: "invalid_log_source",
			query: &types.StructuredQuery{
				LogSource: "invalid-source",
			},
			expectValid: false,
			description: "Invalid log source should fail",
		},
		{
			name: "invalid_verb",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("invalid-verb"),
			},
			expectValid: false,
			description: "Invalid verb should fail",
		},
		{
			name: "invalid_resource",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Resource:  *types.NewStringOrArray("invalid-resource"),
			},
			expectValid: false,
			description: "Invalid resource should fail",
		},
		{
			name: "invalid_auth_decision",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				AuthDecision: "invalid-decision",
			},
			expectValid: false,
			description: "Invalid auth decision should fail",
		},
		{
			name: "multiple_valid_values",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray([]string{"get", "list"}),
				Resource:  *types.NewStringOrArray([]string{"pods", "services"}),
			},
			expectValid: true,
			description: "Multiple valid values should pass",
		},
		{
			name: "mixed_valid_invalid_values",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray([]string{"get", "invalid-verb"}),
			},
			expectValid: false,
			description: "Mix of valid and invalid values should fail",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)
			
			if result == nil {
				t.Fatal("Validate returned nil result")
			}
			
			if result.IsValid != tt.expectValid {
				t.Errorf("%s: Expected IsValid to be %v, got %v. Errors: %v", 
					tt.description, tt.expectValid, result.IsValid, result.Errors)
			}
		})
	}
}

// TestComprehensiveInputValidationRule_ValidatePerformanceLimits tests performance limit validation
func TestComprehensiveInputValidationRule_ValidatePerformanceLimits(t *testing.T) {
	config := &types.InputValidationConfig{
		Enabled: true,
		RequiredFields: types.RequiredFieldsConfig{
			Mandatory: []string{"log_source"},
		},
		CharacterValidation: types.CharacterValidationConfig{
			MaxPatternLength: 500,
			ForbiddenChars:   []string{},
		},
		SecurityPatterns: types.SecurityPatternsConfig{
			ForbiddenPatterns: []string{},
		},
		FieldValues: types.FieldValuesConfig{
			AllowedLogSources: []string{"kube-apiserver"},
		},
		PerformanceLimits: types.PerformanceLimitsConfig{
			MaxResultLimit:    25,
			MaxArrayElements:  5,
			AllowedTimeframes: []string{"today", "yesterday", "1_hour_ago"},
		},
	}
	
	rule := NewComprehensiveInputValidationRule(config)
	
	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
		description string
	}{
		{
			name: "valid_limits",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				Limit:            20,
				Timeframe:        "today",
				ExcludeUsers:     []string{"user1", "user2"},
				ExcludeResources: []string{"resource1"},
			},
			expectValid: true,
			description: "Query within limits should pass",
		},
		{
			name: "exceeds_result_limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     100,
			},
			expectValid: false,
			description: "Exceeding result limit should fail",
		},
		{
			name: "invalid_timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "invalid-timeframe",
			},
			expectValid: false,
			description: "Invalid timeframe should fail",
		},
		{
			name: "exceeds_array_limit_exclude_users",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				ExcludeUsers: []string{"user1", "user2", "user3", "user4", "user5", "user6"},
			},
			expectValid: false,
			description: "Exceeding array limit for exclude_users should fail",
		},
		{
			name: "exceeds_array_limit_exclude_resources",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				ExcludeResources: []string{"res1", "res2", "res3", "res4", "res5", "res6"},
			},
			expectValid: false,
			description: "Exceeding array limit for exclude_resources should fail",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)
			
			if result == nil {
				t.Fatal("Validate returned nil result")
			}
			
			if result.IsValid != tt.expectValid {
				t.Errorf("%s: Expected IsValid to be %v, got %v. Errors: %v", 
					tt.description, tt.expectValid, result.IsValid, result.Errors)
			}
		})
	}
}

// TestComprehensiveInputValidationRule_ComprehensiveValidation tests all validation aspects together
func TestComprehensiveInputValidationRule_ComprehensiveValidation(t *testing.T) {
	rule := NewComprehensiveInputValidationRule(nil) // Use default config
	
	// Test a complex query that should pass all validations
	validQuery := &types.StructuredQuery{
		LogSource:   "kube-apiserver",
		Verb:        *types.NewStringOrArray("get"),
		Resource:    *types.NewStringOrArray("pods"),
		Limit:       20,
		Timeframe:   "today",
		UserPattern: "valid-user-pattern",
	}
	
	result := rule.Validate(validQuery)
	
	if result == nil {
		t.Fatal("Validate returned nil result")
	}
	
	if !result.IsValid {
		t.Errorf("Valid comprehensive query should pass validation. Errors: %v", result.Errors)
	}
	
	// Test query with multiple validation failures
	invalidQuery := &types.StructuredQuery{
		LogSource:   "invalid-source",           // Invalid field value
		UserPattern: "system:admin",             // Forbidden pattern
		Limit:       2000,                       // Exceeds limit
		Timeframe:   "invalid-timeframe",        // Invalid timeframe
	}
	// Missing required log_source (empty = invalid)
	invalidQuery.LogSource = ""
	
	result = rule.Validate(invalidQuery)
	
	if result == nil {
		t.Fatal("Validate returned nil result")
	}
	
	if result.IsValid {
		t.Error("Invalid comprehensive query should fail validation")
	}
	
	if len(result.Errors) == 0 {
		t.Error("Invalid query should have validation errors")
	}
	
	// Should have multiple errors for different validation failures
	if len(result.Errors) < 3 {
		t.Errorf("Expected multiple validation errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// TestComprehensiveInputValidationRule_GetRuleDescription tests the rule description
func TestComprehensiveInputValidationRule_GetRuleDescription(t *testing.T) {
	rule := NewComprehensiveInputValidationRule(nil)
	
	description := rule.GetRuleDescription()
	if description == "" {
		t.Error("Rule description should not be empty")
	}
	
	// Check that description mentions comprehensive validation
	if !strings.Contains(strings.ToLower(description), "comprehensive") || !strings.Contains(strings.ToLower(description), "input validation") {
		t.Errorf("Rule description should mention comprehensive input validation, got: %s", description)
	}
}

// TestComprehensiveInputValidationRule_ValidationDetails tests validation result details
func TestComprehensiveInputValidationRule_ValidationDetails(t *testing.T) {
	rule := NewComprehensiveInputValidationRule(nil)
	
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}
	
	result := rule.Validate(query)
	
	if result == nil {
		t.Fatal("Validate returned nil result")
	}
	
	// Check that validation details are included
	if result.Details == nil {
		t.Error("Validation result should include details")
	}
	
	if validationSections, ok := result.Details["validation_sections"]; !ok {
		t.Error("Validation details should include validation_sections")
	} else if validationSections == nil {
		t.Error("validation_sections should not be nil")
	}
	
	// Check timestamp format
	if result.Timestamp == "" {
		t.Error("Validation result should include timestamp")
	}
	
	// Check that query snapshot is preserved
	if result.QuerySnapshot != query {
		t.Error("Query snapshot should be preserved in validation result")
	}
}


// Benchmark tests
func BenchmarkComprehensiveInputValidationRule_Validate(b *testing.B) {
	rule := NewComprehensiveInputValidationRule(nil)
	
	query := &types.StructuredQuery{
		LogSource:   "kube-apiserver",
		Verb:        *types.NewStringOrArray("get"),
		Resource:    *types.NewStringOrArray("pods"),
		UserPattern: "test-user",
		Limit:       20,
		Timeframe:   "today",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := rule.Validate(query)
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}