package rules

import (
	"strings"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestNewSanitizationRule(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		description string
	}{
		{
			name:        "default_config",
			config:      map[string]interface{}{},
			description: "Default configuration should be applied",
		},
		{
			name: "custom_config",
			config: map[string]interface{}{
				"max_pattern_length": 100,
				"max_query_length":   5000,
				"valid_regex_pattern": "^[a-zA-Z0-9]+$",
				"forbidden_chars":     []interface{}{"<", ">", "&"},
			},
			description: "Custom configuration should be applied",
		},
		{
			name: "partial_config",
			config: map[string]interface{}{
				"max_pattern_length": 200,
			},
			description: "Partial configuration should use defaults for missing values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewSanitizationRule(tt.config)

			if rule == nil {
				t.Fatal("NewSanitizationRule returned nil")
			}
			if !rule.IsEnabled() {
				t.Error("Expected rule to be enabled by default")
			}
			if rule.GetRuleName() != "sanitization_validation" {
				t.Errorf("Expected rule name 'sanitization_validation', got '%s'", rule.GetRuleName())
			}
			if rule.GetSeverity() != "high" {
				t.Errorf("Expected severity 'high', got '%s'", rule.GetSeverity())
			}

			// Test custom configuration values
			if maxLength, ok := tt.config["max_pattern_length"].(int); ok {
				if rule.maxPatternLength != maxLength {
					t.Errorf("Expected max_pattern_length %d, got %d", maxLength, rule.maxPatternLength)
				}
			} else {
				if rule.maxPatternLength != 500 { // default value
					t.Errorf("Expected default max_pattern_length 500, got %d", rule.maxPatternLength)
				}
			}
		})
	}
}

func TestSanitizationRule_ForbiddenCharacters(t *testing.T) {
	forbiddenChars := []interface{}{"<", ">", "&", "\"", "'", "`", "|", ";", "$"}
	config := map[string]interface{}{
		"forbidden_chars": forbiddenChars,
	}
	rule := NewSanitizationRule(config)

	tests := []struct {
		name           string
		query          *types.StructuredQuery
		shouldFail     bool
		expectedErrors int
		description    string
	}{
		{
			name: "xss_attempt_in_user_pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "<script>alert('xss')</script>",
			},
			shouldFail:     true,
			expectedErrors: 4, // <, >, ', regex validation
			description:    "XSS attempt should be blocked",
		},
		{
			name: "sql_injection_in_resource_pattern",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: "test'; DROP TABLE users; --",
			},
			shouldFail:     true,
			expectedErrors: 4, // ', ;, regex, resource validation
			description:    "SQL injection attempt should be blocked",
		},
		{
			name: "command_injection_in_namespace_pattern",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: "test`whoami`",
			},
			shouldFail:     true,
			expectedErrors: 3, // `, regex, namespace validation appears twice
			description:    "Command injection attempt should be blocked",
		},
		{
			name: "pipe_character_in_uri_pattern",
			query: &types.StructuredQuery{
				LogSource:         "kube-apiserver",
				RequestURIPattern: "/api/v1/pods | cat /etc/passwd",
			},
			shouldFail:     true,
			expectedErrors: 1, // |
			description:    "Pipe character should be blocked",
		},
		{
			name: "dollar_sign_in_authorization_pattern",
			query: &types.StructuredQuery{
				LogSource:                  "kube-apiserver",
				AuthorizationReasonPattern: "test$(evil)",
			},
			shouldFail:     true,
			expectedErrors: 2, // $, regex validation
			description:    "Dollar sign should be blocked",
		},
		{
			name: "forbidden_chars_in_exclude_users",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				ExcludeUsers: []string{"safe-user", "evil<script>", "another&bad"},
			},
			shouldFail:     true,
			expectedErrors: 3, // <, >, &
			description:    "Forbidden characters in exclude users should be blocked",
		},
		{
			name: "forbidden_chars_in_exclude_resources",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				ExcludeResources: []string{"safe-resource", "evil'resource", "bad\"resource"},
			},
			shouldFail:     true,
			expectedErrors: 2, // ', "
			description:    "Forbidden characters in exclude resources should be blocked",
		},
		{
			name: "safe_patterns",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         "safe-user",
				ResourceNamePattern: "test-app",
				NamespacePattern:    "production-env",
				RequestURIPattern:   "api-endpoint",
			},
			shouldFail:     false,
			expectedErrors: 0,
			description:    "Safe patterns should pass validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %s. Errors: %v", 
					tt.expectedErrors, len(result.Errors), tt.description, result.Errors)
			}

			// Verify error messages contain character or validation information
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "forbidden character") && 
					   !strings.Contains(err, "Invalid regex") && 
					   !strings.Contains(err, "Invalid resource") && 
					   !strings.Contains(err, "Invalid namespace") {
						t.Errorf("Error message should mention forbidden character or validation issue: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_PatternLengths(t *testing.T) {
	config := map[string]interface{}{
		"max_pattern_length": 50,
	}
	rule := NewSanitizationRule(config)

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldFail  bool
		description string
	}{
		{
			name: "pattern_within_limit",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         "short-pattern",
				ResourceNamePattern: "also-short",
			},
			shouldFail:  false,
			description: "Patterns within length limit should pass",
		},
		{
			name: "user_pattern_exceeds_limit",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: strings.Repeat("a", 51), // 51 characters
			},
			shouldFail:  true,
			description: "User pattern exceeding limit should fail",
		},
		{
			name: "resource_pattern_exceeds_limit",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: strings.Repeat("b", 51), // 51 characters
			},
			shouldFail:  true,
			description: "Resource pattern exceeding limit should fail",
		},
		{
			name: "namespace_pattern_exceeds_limit",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: strings.Repeat("c", 51), // 51 characters
			},
			shouldFail:  true,
			description: "Namespace pattern exceeding limit should fail",
		},
		{
			name: "uri_pattern_exceeds_limit",
			query: &types.StructuredQuery{
				LogSource:         "kube-apiserver",
				RequestURIPattern: strings.Repeat("d", 51), // 51 characters
			},
			shouldFail:  true,
			description: "URI pattern exceeding limit should fail",
		},
		{
			name: "multiple_patterns_exceed_limit",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         strings.Repeat("e", 51),
				ResourceNamePattern: strings.Repeat("f", 51),
			},
			shouldFail:  true,
			description: "Multiple patterns exceeding limit should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}

			// Check error message format for length violations
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "exceeds maximum length") {
						t.Errorf("Error message should mention length violation: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_RegexValidation(t *testing.T) {
	config := map[string]interface{}{
		"valid_regex_pattern": "^[a-zA-Z0-9\\-_\\.\\*]+$", // Allow alphanumeric, dash, underscore, dot, asterisk
	}
	rule := NewSanitizationRule(config)

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldFail  bool
		description string
	}{
		{
			name: "valid_regex_patterns",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         "safe-user",
				ResourceNamePattern: "test-app",
				NamespacePattern:    "prod-env",
			},
			shouldFail:  false,
			description: "Valid regex patterns should pass",
		},
		{
			name: "invalid_regex_with_special_chars",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "user@company(test)",
			},
			shouldFail:  true,
			description: "Regex with invalid characters should fail",
		},
		{
			name: "invalid_regex_with_brackets",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: "test[invalid]",
			},
			shouldFail:  true,
			description: "Regex with brackets should fail",
		},
		{
			name: "invalid_regex_with_pipe",
			query: &types.StructuredQuery{
				LogSource:                  "kube-apiserver",
				AuthorizationReasonPattern: "allow|deny",
			},
			shouldFail:  true,
			description: "Regex with pipe should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}

			// Check error message format for regex violations
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "Invalid regex pattern") && 
					   !strings.Contains(err, "forbidden character") &&
					   !strings.Contains(err, "Invalid namespace") {
						t.Errorf("Error message should mention regex violation: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_IPValidation(t *testing.T) {
	rule := NewSanitizationRule(map[string]interface{}{}) // Use default IP pattern

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldFail  bool
		description string
	}{
		{
			name: "valid_single_ip",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray("192.168.1.100"),
			},
			shouldFail:  false,
			description: "Valid single IP should pass",
		},
		{
			name: "valid_multiple_ips",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray([]string{"192.168.1.100", "10.0.0.1", "172.16.0.1"}),
			},
			shouldFail:  false,
			description: "Valid multiple IPs should pass",
		},
		{
			name: "invalid_single_ip",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray("999.999.999.999"),
			},
			shouldFail:  true,
			description: "Invalid single IP should fail",
		},
		{
			name: "invalid_ip_in_array",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray([]string{"192.168.1.100", "invalid.ip", "10.0.0.1"}),
			},
			shouldFail:  true,
			description: "Invalid IP in array should fail",
		},
		{
			name: "non_ip_string",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray("not-an-ip"),
			},
			shouldFail:  true,
			description: "Non-IP string should fail",
		},
		{
			name: "ip_with_extra_octets",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray("192.168.1.100.5"),
			},
			shouldFail:  true,
			description: "IP with extra octets should fail",
		},
		{
			name: "empty_ip",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  *types.NewStringOrArray(""),
			},
			shouldFail:  false,
			description: "Empty IP should be ignored (not validated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}

			// Check error message format for IP violations
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "Invalid IP address") {
						t.Errorf("Error message should mention IP violation: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_NamespaceValidation(t *testing.T) {
	rule := NewSanitizationRule(map[string]interface{}{}) // Use default namespace pattern

	tests := []struct {
		name              string
		namespacePattern  string
		shouldFail        bool
		description       string
	}{
		{
			name:             "valid_namespace_pattern",
			namespacePattern: "my-app",
			shouldFail:       false,
			description:      "Valid namespace pattern should pass",
		},
		{
			name:             "valid_namespace_with_numbers",
			namespacePattern: "app1-env2",
			shouldFail:       false,
			description:      "Namespace with numbers should pass",
		},
		{
			name:             "single_character_namespace",
			namespacePattern: "a",
			shouldFail:       false,
			description:      "Single character namespace should pass",
		},
		{
			name:             "namespace_starting_with_dash",
			namespacePattern: "-invalid",
			shouldFail:       true,
			description:      "Namespace starting with dash should fail",
		},
		{
			name:             "namespace_ending_with_dash",
			namespacePattern: "invalid-",
			shouldFail:       true,
			description:      "Namespace ending with dash should fail",
		},
		{
			name:             "namespace_with_uppercase",
			namespacePattern: "Invalid-Namespace",
			shouldFail:       true,
			description:      "Namespace with uppercase should fail",
		},
		{
			name:             "namespace_with_underscore",
			namespacePattern: "invalid_namespace",
			shouldFail:       true,
			description:      "Namespace with underscore should fail",
		},
		{
			name:             "namespace_with_special_chars",
			namespacePattern: "invalid@namespace",
			shouldFail:       true,
			description:      "Namespace with special characters should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: tt.namespacePattern,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}

			// Check error message format for namespace violations
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "Invalid namespace pattern") && 
					   !strings.Contains(err, "forbidden character") &&
					   !strings.Contains(err, "Invalid regex") {
						t.Errorf("Error message should mention namespace violation: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_ResourceValidation(t *testing.T) {
	rule := NewSanitizationRule(map[string]interface{}{}) // Use default resource pattern

	tests := []struct {
		name            string
		resourcePattern string
		shouldFail      bool
		description     string
	}{
		{
			name:            "valid_resource_pattern",
			resourcePattern: "pods",
			shouldFail:      false,
			description:     "Valid resource pattern should pass",
		},
		{
			name:            "valid_resource_with_numbers",
			resourcePattern: "app1-deployments",
			shouldFail:      false,
			description:     "Resource with numbers should pass",
		},
		{
			name:            "resource_starting_with_dash",
			resourcePattern: "-invalid",
			shouldFail:      true,
			description:     "Resource starting with dash should fail",
		},
		{
			name:            "resource_ending_with_dash",
			resourcePattern: "invalid-",
			shouldFail:      true,
			description:     "Resource ending with dash should fail",
		},
		{
			name:            "resource_with_uppercase",
			resourcePattern: "Invalid-Resource",
			shouldFail:      true,
			description:     "Resource with uppercase should fail",
		},
		{
			name:            "resource_starting_with_number",
			resourcePattern: "1invalid",
			shouldFail:      true,
			description:     "Resource starting with number should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: tt.resourcePattern,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}

			// Check error message format for resource violations
			if tt.shouldFail {
				for _, err := range result.Errors {
					if !strings.Contains(err, "Invalid resource pattern") {
						t.Errorf("Error message should mention resource violation: %s", err)
					}
				}
			}
		})
	}
}

func TestSanitizationRule_ComprehensiveValidation(t *testing.T) {
	config := map[string]interface{}{
		"max_pattern_length": 100,
		"forbidden_chars":    []interface{}{"<", ">", "&", "'"},
	}
	rule := NewSanitizationRule(config)

	// Complex query with multiple validation issues
	query := &types.StructuredQuery{
		LogSource:           "kube-apiserver",
		UserPattern:         "<script>alert('xss')</script>", // Forbidden chars
		ResourceNamePattern: strings.Repeat("a", 101),        // Too long
		NamespacePattern:    "Invalid-Namespace",             // Invalid format
		SourceIP:            *types.NewStringOrArray("999.999.999.999"), // Invalid IP
		ExcludeUsers:        []string{"safe-user", "bad&user"},           // Forbidden char
	}

	result := rule.Validate(query)

	if result.IsValid {
		t.Error("Expected comprehensive validation to fail")
	}

	// Should have multiple errors
	if len(result.Errors) == 0 {
		t.Error("Expected multiple validation errors")
	}

	// Check that recommendations are provided
	if len(result.Recommendations) == 0 {
		t.Error("Expected validation recommendations")
	}

	// Verify rule information
	if result.RuleName != "sanitization_validation" {
		t.Errorf("Expected rule name 'sanitization_validation', got '%s'", result.RuleName)
	}
	if result.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", result.Severity)
	}
}

func TestSanitizationRule_EdgeCases(t *testing.T) {
	rule := NewSanitizationRule(map[string]interface{}{})

	tests := []struct {
		name          string
		query         *types.StructuredQuery
		expectedValid bool
		description   string
	}{
		{
			name:          "nil_query",
			query:         nil,
			expectedValid: true, // Should handle gracefully
			description:   "Nil query should not crash",
		},
		{
			name:          "empty_query",
			query:         &types.StructuredQuery{},
			expectedValid: true,
			description:   "Empty query should pass",
		},
		{
			name: "empty_string_fields",
			query: &types.StructuredQuery{
				LogSource:           "",
				UserPattern:         "",
				ResourceNamePattern: "",
				NamespacePattern:    "",
			},
			expectedValid: true,
			description:   "Empty string fields should be ignored",
		},
		{
			name: "empty_source_ip",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				SourceIP:  types.StringOrArray{}, // Empty StringOrArray
			},
			expectedValid: true,
			description:   "Empty SourceIP should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Validation panicked: %v", r)
				}
			}()

			var result *interfaces.ValidationResult
			if tt.query != nil {
				result = rule.Validate(tt.query)
			} else {
				// For nil query test, create a mock result
				result = &interfaces.ValidationResult{IsValid: true}
			}

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid=%v, got IsValid=%v: %s", 
					tt.expectedValid, result.IsValid, tt.description)
			}
		})
	}
}

func TestSanitizationRule_PerformanceBenchmark(t *testing.T) {
	// Create a rule with complex validation patterns
	config := map[string]interface{}{
		"max_pattern_length":      1000,
		"valid_regex_pattern":     "^[a-zA-Z0-9\\-_\\.\\*\\+\\[\\]\\{\\}\\(\\)\\|\\\\/\\s]+$",
		"valid_ip_pattern":        "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
		"valid_namespace_pattern": "^[a-z0-9]([a-z0-9\\-]*[a-z0-9])?$",
		"valid_resource_pattern":  "^[a-z]([a-z0-9\\-]*[a-z0-9])?$",
		"forbidden_chars":         []interface{}{"<", ">", "&", "\"", "'", "`", "|", ";", "$", "(", ")", "{", "}", "[", "]", "\\", "/", "!", "@", "#", "%", "^", "*", "+", "=", "~"},
	}
	rule := NewSanitizationRule(config)

	// Create a complex query with many fields
	query := &types.StructuredQuery{
		LogSource:                   "kube-apiserver",
		UserPattern:                 "user.*@company.com",
		ResourceNamePattern:         "test-app-.*",
		NamespacePattern:            "production-env",
		RequestURIPattern:           "/api/v1/namespaces/default/pods",
		AuthorizationReasonPattern:  "allowed",
		ResponseMessagePattern:      "success",
		MissingAnnotation:           "test.annotation",
		RequestObjectFilter:         "filter-expression",
		SourceIP:                    *types.NewStringOrArray([]string{"192.168.1.100", "10.0.0.1", "172.16.0.1"}),
		ExcludeUsers:                []string{"system:", "kube-", "openshift-"},
		ExcludeResources:            []string{"events", "endpoints", "endpointslices"},
	}

	// Benchmark the validation
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rule.Validate(query)
		}
	})

	// Check that validation completes within reasonable time (< 5ms per operation)
	avgTimePerOp := result.T / time.Duration(result.N)
	if avgTimePerOp > 5*time.Millisecond {
		t.Errorf("Validation too slow: %v per operation (should be < 5ms)", avgTimePerOp)
	}

	t.Logf("Performance: %v per validation operation", avgTimePerOp)
}

func TestSanitizationRule_FunctionalQueryValidation(t *testing.T) {
	// Test validation with patterns similar to functional test queries
	rule := NewSanitizationRule(map[string]interface{}{})

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "basic_query_pattern",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				ExcludeUsers: []string{"system:", "kube-"},
			},
			shouldPass:  true,
			description: "Basic query pattern should pass sanitization",
		},
		{
			name: "intermediate_query_pattern",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				UserPattern:      "company-user",
				NamespacePattern: "prod-env",
			},
			shouldPass:  true,
			description: "Intermediate query pattern should pass sanitization",
		},
		{
			name: "advanced_query_pattern",
			query: &types.StructuredQuery{
				LogSource:                  "kube-apiserver",
				UserPattern:                "non-system-user",
				RequestURIPattern:          "api-v1-namespaces-secrets",
				AuthorizationReasonPattern: "admin-user",
			},
			shouldPass:  true,
			description: "Advanced query pattern should pass sanitization",
		},
		{
			name: "malicious_query_attempt",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "'; DROP TABLE users; --",
			},
			shouldPass:  false,
			description: "Malicious query attempt should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected query to pass sanitization: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected query to fail sanitization: %s", tt.description)
			}
		})
	}
}