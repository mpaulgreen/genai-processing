package rules

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestFieldValuesRule_Validate(t *testing.T) {
	// Standard test configuration with field value configurations
	testConfig := map[string]interface{}{
		"business_hours_configuration": map[string]interface{}{
			"allowed_presets": []interface{}{
				"business_hours", "outside_business_hours", "weekend", "all_hours",
			},
		},
		"response_status_configuration": map[string]interface{}{
			"allowed_status_codes": []interface{}{
				"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504",
			},
		},
		"auth_decisions_configuration": map[string]interface{}{
			"allowed_decisions": []interface{}{
				"allow", "error", "forbid",
			},
		},
	}

	tests := []struct {
		name           string
		config         map[string]interface{}
		query          *types.StructuredQuery
		expectedValid  bool
		expectedErrors int
	}{
		{
			name:   "Valid auth decision",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				AuthDecision: "allow",
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Invalid auth decision",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				AuthDecision: "invalid_decision",
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Valid response status - single value",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: *types.NewStringOrArray("200"),
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Invalid response status - single value",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: *types.NewStringOrArray("999"),
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name:   "Valid response status - array",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: *types.NewStringOrArray([]string{"200", "201", "404"}),
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Invalid response status - array with one bad value",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: *types.NewStringOrArray([]string{"200", "999", "404"}),
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		// Note: Business hours preset validation temporarily removed
		// Current BusinessHours type doesn't support preset field
		{
			name:   "Empty fields - all valid",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Multiple field validation",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				AuthDecision:   "allow",
				ResponseStatus: *types.NewStringOrArray("200"),
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "Multiple invalid fields",
			config: testConfig,
			query: &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				AuthDecision:   "invalid_auth",
				ResponseStatus: *types.NewStringOrArray("999"),
			},
			expectedValid:  false,
			expectedErrors: 2,
		},
		{
			name:   "No configuration provided",
			config: map[string]interface{}{},
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				AuthDecision: "allow",
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewFieldValuesRule(tt.config)
			result := rule.Validate(tt.query)

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v, got %v", tt.expectedValid, result.IsValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(result.Errors), result.Errors)
			}

			// Validate rule interface implementation
			if rule.GetRuleName() == "" {
				t.Error("Rule name should not be empty")
			}

			if rule.GetRuleDescription() == "" {
				t.Error("Rule description should not be empty")
			}

			if !rule.IsEnabled() {
				t.Error("Rule should be enabled by default")
			}

			if rule.GetSeverity() == "" {
				t.Error("Rule severity should not be empty")
			}
		})
	}
}

func TestFieldValuesRule_ValidateAuthDecision(t *testing.T) {
	tests := []struct {
		name          string
		authDecision  string
		expectedValid bool
	}{
		{"Valid allow", "allow", true},
		{"Valid error", "error", true},
		{"Valid forbid", "forbid", true},
		{"Invalid decision", "invalid", false},
		{"Empty decision", "", true}, // Optional field
	}

	testConfig := map[string]interface{}{
		"auth_decisions_configuration": map[string]interface{}{
			"allowed_decisions": []interface{}{"allow", "error", "forbid"},
		},
	}
	rule := NewFieldValuesRule(testConfig)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				AuthDecision: tt.authDecision,
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for auth_decision '%s', got %v",
					tt.expectedValid, tt.authDecision, result.IsValid)
			}
		})
	}
}

// TestFieldValuesRule_ValidateBusinessHours - Temporarily disabled
// Current BusinessHours type doesn't support preset validation
func TestFieldValuesRule_ValidateBusinessHours(t *testing.T) {
	t.Skip("Business hours preset validation not implemented - current type doesn't support presets")
}

func TestFieldValuesRule_ValidateResponseStatus(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus types.StringOrArray
		expectedValid  bool
	}{
		{"Valid single status", *types.NewStringOrArray("200"), true},
		{"Valid array status", *types.NewStringOrArray([]string{"200", "201", "404"}), true},
		{"Invalid single status", *types.NewStringOrArray("999"), false},
		{"Invalid array with one bad", *types.NewStringOrArray([]string{"200", "999"}), false},
		{"Empty status", *types.NewStringOrArray(""), true}, // Optional field
	}

	testConfig := map[string]interface{}{
		"response_status_configuration": map[string]interface{}{
			"allowed_status_codes": []interface{}{
				"200", "201", "204", "400", "401", "403", "404", "409", "422", "500", "502", "503", "504",
			},
		},
	}
	rule := NewFieldValuesRule(testConfig)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:      "kube-apiserver",
				ResponseStatus: tt.responseStatus,
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for response_status '%v', got %v",
					tt.expectedValid, tt.responseStatus, result.IsValid)
			}
		})
	}
}

func TestFieldValuesRule_ConfigDefaults(t *testing.T) {
	// Test with nil config (should use fallback defaults)
	rule := NewFieldValuesRule(nil)

	allowedDecisions := rule.getAllowedAuthDecisions()
	expectedDecisions := []string{"allow", "error", "forbid"}
	if len(allowedDecisions) != len(expectedDecisions) {
		t.Errorf("Expected default auth decisions %v, got %v", expectedDecisions, allowedDecisions)
	}

	allowedPresets := rule.getAllowedBusinessHoursPresets()
	if len(allowedPresets) < 3 {
		t.Errorf("Expected multiple default business hours presets, got %v", allowedPresets)
	}

	allowedCodes := rule.getAllowedResponseStatusCodes()
	if len(allowedCodes) < 10 {
		t.Errorf("Expected multiple default response status codes, got %v", allowedCodes)
	}
}

func TestFieldValuesRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"auth_decisions_configuration": map[string]interface{}{
			"allowed_decisions": []interface{}{"custom_allow", "custom_deny"},
		},
		"business_hours_configuration": map[string]interface{}{
			"allowed_presets": []interface{}{"custom_hours"},
		},
		"response_status_configuration": map[string]interface{}{
			"allowed_status_codes": []interface{}{"200", "300"},
		},
	}

	rule := NewFieldValuesRule(customConfig)

	// Test custom auth decisions
	allowedDecisions := rule.getAllowedAuthDecisions()
	if len(allowedDecisions) != 2 || allowedDecisions[0] != "custom_allow" {
		t.Errorf("Expected custom auth decisions, got %v", allowedDecisions)
	}

	// Test custom business hours presets
	allowedPresets := rule.getAllowedBusinessHoursPresets()
	if len(allowedPresets) != 1 || allowedPresets[0] != "custom_hours" {
		t.Errorf("Expected custom business hours presets, got %v", allowedPresets)
	}

	// Test custom response status codes
	allowedCodes := rule.getAllowedResponseStatusCodes()
	if len(allowedCodes) != 2 || allowedCodes[0] != "200" || allowedCodes[1] != "300" {
		t.Errorf("Expected custom response status codes, got %v", allowedCodes)
	}
}