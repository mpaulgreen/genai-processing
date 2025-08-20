package validator

import (
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// TestSafetyValidator_InterfaceCompliance verifies that SafetyValidator
// properly implements the SafetyValidator interface.
func TestSafetyValidator_InterfaceCompliance(t *testing.T) {
	var _ interfaces.SafetyValidator = (*SafetyValidator)(nil)
}

// TestNewSafetyValidator tests the constructor function.
func TestNewSafetyValidator(t *testing.T) {
	validator := NewSafetyValidator()

	if validator == nil {
		t.Fatal("NewSafetyValidator returned nil")
	}
}

// TestSafetyValidator_ValidateQuery tests the ValidateQuery method
// with various input scenarios.
func TestSafetyValidator_ValidateQuery(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name: "valid query with all fields",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
				Limit:     20,
			},
			expectValid: true,
		},
		{
			name: "valid query with minimal fields",
			query: &types.StructuredQuery{
				LogSource: "oauth-server",
			},
			expectValid: true,
		},
		{
			name:        "nil query",
			query:       nil,
			expectValid: false, // Real implementation should reject nil queries
		},
		{
			name: "complex query with arrays",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				Verb:         *types.NewStringOrArray([]string{"get", "list"}),
				Resource:     *types.NewStringOrArray([]string{"pods", "services"}),
				Namespace:    *types.NewStringOrArray("default"),
				ExcludeUsers: []string{"system:", "kube-"},
				Limit:        100,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)

			// Check that no error is returned
			if err != nil {
				t.Errorf("ValidateQuery returned error: %v", err)
			}

			// Check that result is not nil
			if result == nil {
				t.Fatal("ValidateQuery returned nil result")
			}

			// Check that validation result matches expected
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}

			// Check that required fields are present
			if result.RuleName == "" {
				t.Error("ValidationResult.RuleName should not be empty")
			}

			if result.Severity == "" {
				t.Error("ValidationResult.Severity should not be empty")
			}

			if result.Message == "" {
				t.Error("ValidationResult.Message should not be empty")
			}

			if result.Timestamp == "" {
				t.Error("ValidationResult.Timestamp should not be empty")
			}

			// Check that query snapshot is preserved
			if result.QuerySnapshot != tt.query {
				t.Error("ValidationResult.QuerySnapshot should match input query")
			}

			// Check that validation result includes proper details
			if result.Details == nil {
				t.Error("ValidationResult.Details should not be nil")
			}

			// Check that validation result includes rule results
			if ruleResults, ok := result.Details["rule_results"]; !ok {
				t.Error("ValidationResult.Details should include rule_results")
			} else if ruleResults == nil {
				t.Error("rule_results should not be nil")
			}
		})
	}
}

// TestSafetyValidator_GetApplicableRules tests the GetApplicableRules method.
func TestSafetyValidator_GetApplicableRules(t *testing.T) {
	validator := NewSafetyValidator()

	rules := validator.GetApplicableRules()

	// Real implementation should return active rules
	if rules == nil {
		t.Fatal("GetApplicableRules returned nil")
	}

	if len(rules) == 0 {
		t.Error("Expected at least one active rule, got 0 rules")
	}

	// Check that rules have proper names and descriptions
	for _, rule := range rules {
		if rule.GetRuleName() == "" {
			t.Error("Rule should have a non-empty name")
		}
		if rule.GetRuleDescription() == "" {
			t.Error("Rule should have a non-empty description")
		}
		if rule.GetSeverity() == "" {
			t.Error("Rule should have a non-empty severity")
		}
	}
}

// TestSafetyValidator_ConsistentBehavior tests that the implementation
// behaves consistently with the same input.
func TestSafetyValidator_ConsistentBehavior(t *testing.T) {
	validator := NewSafetyValidator()

	// Test multiple calls to ensure consistent behavior
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
		Limit:     20,
	}

	result1, err1 := validator.ValidateQuery(query)
	result2, err2 := validator.ValidateQuery(query)

	// Both calls should succeed
	if err1 != nil || err2 != nil {
		t.Fatal("Both validation calls should succeed")
	}

	// Both results should be valid for safe queries
	if !result1.IsValid || !result2.IsValid {
		t.Error("Safe queries should return valid results")
	}

	// Both results should have the same rule name
	if result1.RuleName != result2.RuleName {
		t.Error("Implementation should return consistent rule names")
	}

	// Both results should have the same severity
	if result1.Severity != result2.Severity {
		t.Error("Implementation should return consistent severity")
	}

	// Both results should include proper details
	if result1.Details == nil || result2.Details == nil {
		t.Error("Results should include proper details")
	}
}

// TestSafetyValidator_ErrorHandling tests error handling scenarios.
func TestSafetyValidator_ErrorHandling(t *testing.T) {
	validator := NewSafetyValidator()

	// Test with various edge cases
	edgeCases := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name:        "nil query",
			query:       nil,
			expectValid: false,
		},
		{
			name:        "empty query",
			query:       &types.StructuredQuery{},
			expectValid: false, // Missing required log_source
		},
		{
			name: "invalid log source",
			query: &types.StructuredQuery{
				LogSource: "invalid-source",
			},
			expectValid: false,
		},
		{
			name: "invalid verb",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("invalid-verb"),
			},
			expectValid: false,
		},
		{
			name: "invalid resource",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Resource:  *types.NewStringOrArray("invalid-resource"),
			},
			expectValid: false,
		},
		{
			name: "exceeds limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     2000, // Exceeds max limit
			},
			expectValid: false,
		},
	}

	for _, tt := range edgeCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)

			// Implementation should never return errors
			if err != nil {
				t.Errorf("Implementation should not return errors, got: %v", err)
			}

			// Result should not be nil
			if result == nil {
				t.Fatal("Result should not be nil")
			}

			// Check validation result matches expectation
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}

			// Invalid queries should have errors
			if !tt.expectValid && len(result.Errors) == 0 {
				t.Error("Invalid queries should have validation errors")
			}
		})
	}
}

// TestSafetyValidator_WhitelistValidation tests whitelist validation specifically
func TestSafetyValidator_WhitelistValidation(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name: "valid log source",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			expectValid: true,
		},
		{
			name: "invalid log source",
			query: &types.StructuredQuery{
				LogSource: "invalid-source",
			},
			expectValid: false,
		},
		{
			name: "valid verb",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
			},
			expectValid: true,
		},
		{
			name: "invalid verb",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("invalid-verb"),
			},
			expectValid: false,
		},
		{
			name: "valid resource",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Resource:  *types.NewStringOrArray("pods"),
			},
			expectValid: true,
		},
		{
			name: "invalid resource",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Resource:  *types.NewStringOrArray("invalid-resource"),
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)
			if err != nil {
				t.Fatalf("ValidateQuery returned error: %v", err)
			}

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}
		})
	}
}

// TestSafetyValidator_TimeframeValidation tests timeframe validation specifically
func TestSafetyValidator_TimeframeValidation(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name: "valid timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "today",
			},
			expectValid: true,
		},
		{
			name: "invalid timeframe",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "invalid-timeframe",
			},
			expectValid: false,
		},
		{
			name: "valid limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     100,
			},
			expectValid: true,
		},
		{
			name: "exceeds max limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     2000,
			},
			expectValid: false,
		},
		{
			name: "below min limit",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Limit:     -1,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)
			if err != nil {
				t.Fatalf("ValidateQuery returned error: %v", err)
			}

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}
		})
	}
}

// TestSafetyValidator_SanitizationValidation tests sanitization validation specifically
func TestSafetyValidator_SanitizationValidation(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name: "valid pattern",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: "valid-pattern",
			},
			expectValid: true,
		},
		{
			name: "pattern with forbidden characters",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: "pattern<script>",
			},
			expectValid: false,
		},
		{
			name: "valid user pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "valid-user",
			},
			expectValid: true,
		},
		{
			name: "user pattern with forbidden characters",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "user;rm -rf",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)
			if err != nil {
				t.Fatalf("ValidateQuery returned error: %v", err)
			}

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}
		})
	}
}

// TestSafetyValidator_PatternsValidation tests forbidden patterns validation specifically
func TestSafetyValidator_PatternsValidation(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectValid bool
	}{
		{
			name: "safe query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			expectValid: true,
		},
		{
			name: "dangerous user pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "system:admin",
			},
			expectValid: false,
		},
		{
			name: "dangerous namespace pattern",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: "kube-system",
			},
			expectValid: false,
		},
		{
			name: "dangerous URI pattern",
			query: &types.StructuredQuery{
				LogSource:         "kube-apiserver",
				RequestURIPattern: "/api/v1/pods/.*/exec",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateQuery(tt.query)
			if err != nil {
				t.Fatalf("ValidateQuery returned error: %v", err)
			}

			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid to be %v, got %v", tt.expectValid, result.IsValid)
			}
		})
	}
}

// TestSafetyValidator_GetValidationStats tests the GetValidationStats method
func TestSafetyValidator_GetValidationStats(t *testing.T) {
	validator := NewSafetyValidator()

	stats := validator.GetValidationStats()

	// Check that stats contains expected fields
	if stats["total_active_rules"] == nil {
		t.Error("Stats should contain total_active_rules")
	}

	if stats["whitelist_enabled"] == nil {
		t.Error("Stats should contain whitelist_enabled")
	}

	if stats["sanitization_enabled"] == nil {
		t.Error("Stats should contain sanitization_enabled")
	}

	if stats["timeframe_enabled"] == nil {
		t.Error("Stats should contain timeframe_enabled")
	}

	if stats["patterns_enabled"] == nil {
		t.Error("Stats should contain patterns_enabled")
	}

	// Check that at least one rule is enabled
	totalRules, ok := stats["total_active_rules"].(int)
	if !ok {
		t.Error("total_active_rules should be an int")
	}
	if totalRules == 0 {
		t.Error("At least one validation rule should be active")
	}
}

// Benchmark validation performance
func BenchmarkSafetyValidator_ValidateQuery(b *testing.B) {
	validator := NewSafetyValidator()
	
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
		Timeframe: "today",
		UserPattern: "test-user",
		Namespace:   *types.NewStringOrArray("default"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := validator.ValidateQuery(query)
		if err != nil {
			b.Fatalf("Validation error: %v", err)
		}
		if result == nil {
			b.Fatal("Expected non-nil result")
		}
	}
}

func BenchmarkSafetyValidator_GetApplicableRules(b *testing.B) {
	validator := NewSafetyValidator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rules := validator.GetApplicableRules()
		if len(rules) == 0 {
			b.Error("Expected some applicable rules")
		}
	}
}

func TestSafetyValidator_ConcurrentValidation(t *testing.T) {
	validator := NewSafetyValidator()
	
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		UserPattern: "safe-pattern",
	}
	
	// Test concurrent validation
	numGoroutines := 10
	results := make(chan *interfaces.ValidationResult, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := validator.ValidateQuery(query)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	
	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result == nil {
				t.Error("Expected non-nil result from concurrent validation")
			}
		case err := <-errors:
			t.Errorf("Unexpected error from concurrent validation: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent validation results")
		}
	}
}

// TestSafetyValidator_InitializeRules tests the initializeRules method
func TestSafetyValidator_InitializeRules(t *testing.T) {
	config := &ValidationConfig{}
	
	// Test with minimal config
	validator := NewSafetyValidatorWithConfig(config)
	
	// Should handle empty config gracefully
	if validator.whitelist != nil {
		t.Error("Expected whitelist to be nil with empty config")
	}
	
	if validator.sanitization != nil {
		t.Error("Expected sanitization to be nil with empty config")
	}
	
	// Test with full config
	config.SafetyRules.AllowedLogSources = []string{"kube-apiserver"}
	config.SafetyRules.AllowedVerbs = []string{"get"}
	config.SafetyRules.AllowedResources = []string{"pods"}
	config.SafetyRules.ForbiddenPatterns = []string{"DROP TABLE"}
	config.SafetyRules.RequiredFields = []string{"log_source"}
	config.SafetyRules.Sanitization = map[string]interface{}{
		"max_pattern_length": 500,
	}
	config.SafetyRules.TimeframeLimits = map[string]interface{}{
		"max_days_back": 90,
	}
	
	validator = NewSafetyValidatorWithConfig(config)
	
	// Should initialize all rules
	if validator.whitelist == nil {
		t.Error("Expected whitelist to be initialized")
	}
	
	if validator.sanitization == nil {
		t.Error("Expected sanitization to be initialized")
	}
	
	if validator.timeframe == nil {
		t.Error("Expected timeframe to be initialized")
	}
	
	if validator.patterns == nil {
		t.Error("Expected patterns to be initialized")
	}
	
	if validator.requiredFields == nil {
		t.Error("Expected requiredFields to be initialized")
	}
}

// TestSafetyValidator_ValidationResult_Structure tests the validation result structure
func TestSafetyValidator_ValidationResult_Structure(t *testing.T) {
	validator := NewSafetyValidator()
	
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		UserPattern: "safe-pattern",
	}
	
	result, err := validator.ValidateQuery(query)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	// Check result structure
	if result.RuleName == "" {
		t.Error("Expected non-empty rule name")
	}
	
	if result.Severity == "" {
		t.Error("Expected non-empty severity")
	}
	
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
	
	if result.Timestamp == "" {
		t.Error("Expected non-empty timestamp")
	}
	
	if result.Details == nil {
		t.Error("Expected non-nil details")
	}
	
	if result.QuerySnapshot == nil {
		t.Error("Expected non-nil query snapshot")
	}
	
	// Check details structure
	if _, exists := result.Details["rule_results"]; !exists {
		t.Error("Expected rule_results in details")
	}
	
	if _, exists := result.Details["total_rules_applied"]; !exists {
		t.Error("Expected total_rules_applied in details")
	}
	
	if _, exists := result.Details["validation_timestamp"]; !exists {
		t.Error("Expected validation_timestamp in details")
	}
	
	// Validate timestamp format
	if _, err := time.Parse(time.RFC3339, result.Timestamp); err != nil {
		t.Errorf("Invalid timestamp format: %s", result.Timestamp)
	}
}

// TestSafetyValidator_CustomConfig tests validator with custom configuration
func TestSafetyValidator_CustomConfig(t *testing.T) {
	config := &ValidationConfig{}
	// Use a valid log source that's also in the schema validator's hardcoded list
	config.SafetyRules.AllowedLogSources = []string{"kube-apiserver"}
	config.SafetyRules.AllowedVerbs = []string{"get", "list"}
	config.SafetyRules.ForbiddenPatterns = []string{"DROP TABLE"}
	config.SafetyRules.RequiredFields = []string{"log_source"}
	
	// Apply defaults to the config like the loader does
	config.ApplyDefaults()
	
	validator := NewSafetyValidatorWithConfig(config)
	
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	
	if validator.config != config {
		t.Error("Expected custom config to be used")
	}
	
	// Test validation with custom config (using valid log source)
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
	}
	
	result, err := validator.ValidateQuery(query)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !result.IsValid {
		t.Errorf("Expected validation to pass with custom config. Errors: %v", result.Errors)
	}
	
	// Test invalid query with custom config
	invalidQuery := &types.StructuredQuery{
		LogSource: "invalid-source",
	}
	
	result, err = validator.ValidateQuery(invalidQuery)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result.IsValid {
		t.Error("Expected validation to fail for invalid log source")
	}
}
