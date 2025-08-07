package validator

import (
	"fmt"
	"testing"

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
			expectValid: true, // Stub implementation always returns true
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

			// Check that stub implementation includes expected warnings
			if len(result.Warnings) == 0 {
				t.Error("Stub implementation should include warnings about being a stub")
			}

			// Check that stub implementation includes recommendations
			if len(result.Recommendations) == 0 {
				t.Error("Stub implementation should include TODO recommendations")
			}
		})
	}
}

// TestSafetyValidator_GetApplicableRules tests the GetApplicableRules method.
func TestSafetyValidator_GetApplicableRules(t *testing.T) {
	validator := NewSafetyValidator()

	rules := validator.GetApplicableRules()

	// Stub implementation should return empty slice
	if rules == nil {
		t.Fatal("GetApplicableRules returned nil")
	}

	if len(rules) != 0 {
		t.Errorf("Expected empty rules slice, got %d rules", len(rules))
	}
}

// TestSafetyValidator_StubBehavior tests that the stub implementation
// behaves as expected with consistent results.
func TestSafetyValidator_StubBehavior(t *testing.T) {
	validator := NewSafetyValidator()

	// Test multiple calls to ensure consistent stub behavior
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("pods"),
	}

	result1, err1 := validator.ValidateQuery(query)
	result2, err2 := validator.ValidateQuery(query)

	// Both calls should succeed
	if err1 != nil || err2 != nil {
		t.Fatal("Both validation calls should succeed")
	}

	// Both results should be valid (stub behavior)
	if !result1.IsValid || !result2.IsValid {
		t.Error("Stub implementation should always return valid results")
	}

	// Both results should have the same rule name
	if result1.RuleName != result2.RuleName {
		t.Error("Stub implementation should return consistent rule names")
	}

	// Both results should have the same severity
	if result1.Severity != result2.Severity {
		t.Error("Stub implementation should return consistent severity")
	}

	// Both results should include stub warnings
	if len(result1.Warnings) == 0 || len(result2.Warnings) == 0 {
		t.Error("Stub implementation should include warnings")
	}

	// Both results should include TODO recommendations
	if len(result1.Recommendations) == 0 || len(result2.Recommendations) == 0 {
		t.Error("Stub implementation should include recommendations")
	}
}

// TestSafetyValidator_ErrorHandling tests error handling scenarios.
func TestSafetyValidator_ErrorHandling(t *testing.T) {
	validator := NewSafetyValidator()

	// Test with various edge cases that should not cause errors
	edgeCases := []*types.StructuredQuery{
		nil,
		{},
		{
			LogSource: "",
			Verb:      *types.NewStringOrArray(""),
			Resource:  *types.NewStringOrArray([]string{}),
		},
		{
			LogSource: "invalid-source",
			Limit:     -1,
		},
	}

	for i, query := range edgeCases {
		t.Run(fmt.Sprintf("edge_case_%d", i), func(t *testing.T) {
			result, err := validator.ValidateQuery(query)

			// Stub implementation should never return errors
			if err != nil {
				t.Errorf("Stub implementation should not return errors, got: %v", err)
			}

			// Result should always be valid in stub implementation
			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if !result.IsValid {
				t.Error("Stub implementation should always return valid results")
			}
		})
	}
}
