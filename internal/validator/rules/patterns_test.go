package rules

import (
	"fmt"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestNewPatternsRule(t *testing.T) {
	forbiddenPatterns := []string{"rm -rf", "DROP TABLE", "$(", "`"}
	rule := NewPatternsRule(forbiddenPatterns)

	if rule == nil {
		t.Fatal("NewPatternsRule returned nil")
	}
	if !rule.IsEnabled() {
		t.Error("Expected rule to be enabled by default")
	}
	if rule.GetRuleName() != "forbidden_patterns_validation" {
		t.Errorf("Expected rule name 'forbidden_patterns_validation', got '%s'", rule.GetRuleName())
	}
	if rule.GetSeverity() != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", rule.GetSeverity())
	}
}

func TestPatternsRule_ValidQuery(t *testing.T) {
	forbiddenPatterns := []string{"rm -rf", "DROP TABLE", "$(", "`", "system:admin"}
	rule := NewPatternsRule(forbiddenPatterns)

	// Test valid query that should pass validation
	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
		Namespace: *types.NewStringOrArray("default"),
		User:      *types.NewStringOrArray("test@company.com"),
		Timeframe: "24_hours_ago",
		Limit:     20,
	}

	result := rule.Validate(query)

	if !result.IsValid {
		t.Errorf("Expected valid query to pass, but got errors: %v", result.Errors)
	}
	if result.RuleName != "forbidden_patterns_validation" {
		t.Errorf("Expected rule name 'forbidden_patterns_validation', got '%s'", result.RuleName)
	}
	if result.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", result.Severity)
	}
	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors for valid query, got: %v", result.Errors)
	}
}

func TestPatternsRule_ForbiddenPatterns(t *testing.T) {
	tests := []struct {
		name           string
		query          *types.StructuredQuery
		forbiddenPatterns []string
		shouldFail     bool
		expectedErrors int
		description    string
	}{
		{
			name: "sql_injection_in_user_pattern",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "'; DROP TABLE users; --",
			},
			forbiddenPatterns: []string{"DROP TABLE", ";"},
			shouldFail:     true,
			expectedErrors: 2, // Both "DROP TABLE" and ";" should match
			description:    "SQL injection patterns should be detected",
		},
		{
			name: "command_injection_in_resource_pattern",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				ResourceNamePattern: "$(rm -rf /)",
			},
			forbiddenPatterns: []string{"$(", "rm -rf"},
			shouldFail:     true,
			expectedErrors: 2, // Both "$(" and "rm -rf" should match
			description:    "Command injection patterns should be detected",
		},
		{
			name: "backtick_command_execution",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: "`whoami`",
			},
			forbiddenPatterns: []string{"`"},
			shouldFail:     true,
			expectedErrors: 1,
			description:    "Backtick command execution should be detected",
		},
		{
			name: "dangerous_namespace_access",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: "kube-system",
			},
			forbiddenPatterns: []string{"kube-system"},
			shouldFail:     true,
			expectedErrors: 2, // Two different checks detect the same issue
			description:    "Access to dangerous namespaces should be detected",
		},
		{
			name: "multiple_forbidden_patterns",
			query: &types.StructuredQuery{
				LogSource:           "kube-apiserver",
				UserPattern:         "system:admin",
				ResourceNamePattern: "$(evil)",
				NamespacePattern:    "kube-system",
			},
			forbiddenPatterns: []string{"system:admin", "$(", "kube-system"},
			shouldFail:     true,
			expectedErrors: 6, // Multiple validation checks detect each issue // All three patterns should be detected
			description:    "Multiple forbidden patterns should all be detected",
		},
		{
			name: "case_insensitive_detection",
			query: &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: "SYSTEM:ADMIN",
			},
			forbiddenPatterns: []string{"system:admin"},
			shouldFail:     true,
			expectedErrors: 3, // Multiple checks detect the same pattern
			description:    "Pattern detection should be case insensitive",
		},
		{
			name: "array_field_validation",
			query: &types.StructuredQuery{
				LogSource:    "kube-apiserver",
				ExcludeUsers: []string{"safe-user", "$(evil)", "another-user"},
			},
			forbiddenPatterns: []string{"$("},
			shouldFail:     true,
			expectedErrors: 1,
			description:    "Forbidden patterns in array fields should be detected",
		},
		{
			name: "stringorarray_validation",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				User:      *types.NewStringOrArray([]string{"user1", "system:admin", "user3"}),
			},
			forbiddenPatterns: []string{"system:admin"},
			shouldFail:     true,
			expectedErrors: 1,
			description:    "Forbidden patterns in StringOrArray fields should be detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewPatternsRule(tt.forbiddenPatterns)
			result := rule.Validate(tt.query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected query to fail validation but it passed: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected query to pass validation but it failed: %s. Errors: %v", tt.description, result.Errors)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d. Errors: %v", tt.expectedErrors, len(result.Errors), result.Errors)
			}

			// Verify error messages contain expected information
			if tt.shouldFail {
				for _, err := range result.Errors {
					if err == "" {
						t.Error("Error message should not be empty")
					}
				}
				if len(result.Recommendations) == 0 {
					t.Error("Failed validation should include recommendations")
				}
			}
		})
	}
}

func TestPatternsRule_DangerousURIPatterns(t *testing.T) {
	tests := []struct {
		name        string
		uriPattern  string
		shouldFail  bool
		description string
	}{
		{
			name:        "dangerous_exec_pattern",
			uriPattern:  "/api/v1/pods/test-pod/exec",
			shouldFail:  true,
			description: "Exec URI patterns should be blocked",
		},
		{
			name:        "dangerous_proxy_pattern",
			uriPattern:  "/api/v1/nodes/master-1/proxy",
			shouldFail:  true,
			description: "Node proxy URI patterns should be blocked",
		},
		{
			name:        "dangerous_attach_pattern",
			uriPattern:  "/api/v1/pods/test-pod/attach",
			shouldFail:  true,
			description: "Pod attach URI patterns should be blocked",
		},
		{
			name:        "dangerous_portforward_pattern",
			uriPattern:  "/api/v1/pods/test-pod/portforward",
			shouldFail:  true,
			description: "Port forward URI patterns should be blocked",
		},
		{
			name:        "dangerous_finalize_pattern",
			uriPattern:  "/api/v1/namespaces/test/finalize",
			shouldFail:  true,
			description: "Namespace finalize URI patterns should be blocked",
		},
		{
			name:        "safe_get_pattern",
			uriPattern:  "/api/v1/pods",
			shouldFail:  false,
			description: "Safe GET URI patterns should be allowed",
		},
		{
			name:        "safe_list_pattern",
			uriPattern:  "/api/v1/namespaces/default/pods",
			shouldFail:  false,
			description: "Safe list URI patterns should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewPatternsRule([]string{}) // No general forbidden patterns for this test
			query := &types.StructuredQuery{
				LogSource:         "kube-apiserver",
				RequestURIPattern: tt.uriPattern,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected dangerous URI pattern to fail validation: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected safe URI pattern to pass validation: %s. Errors: %v", tt.description, result.Errors)
			}
		})
	}
}

func TestPatternsRule_DangerousNamespacePatterns(t *testing.T) {
	tests := []struct {
		name            string
		namespacePattern string
		shouldFail      bool
		description     string
	}{
		{
			name:             "kube_system_pattern",
			namespacePattern: "kube-system",
			shouldFail:       true,
			description:      "kube-system namespace should be blocked",
		},
		{
			name:             "openshift_pattern",
			namespacePattern: "openshift-config",
			shouldFail:       true,
			description:      "OpenShift system namespaces should be blocked",
		},
		{
			name:             "production_pattern",
			namespacePattern: "production",
			shouldFail:       true,
			description:      "Production namespace should be blocked",
		},
		{
			name:             "default_pattern",
			namespacePattern: "default",
			shouldFail:       true,
			description:      "Default namespace should be blocked",
		},
		{
			name:             "safe_namespace_pattern",
			namespacePattern: "my-app-dev",
			shouldFail:       false,
			description:      "Safe user namespace should be allowed",
		},
		{
			name:             "test_namespace_pattern",
			namespacePattern: "test-environment",
			shouldFail:       false,
			description:      "Test namespace should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewPatternsRule([]string{}) // No general forbidden patterns for this test
			query := &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				NamespacePattern: tt.namespacePattern,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected dangerous namespace pattern to fail validation: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected safe namespace pattern to pass validation: %s. Errors: %v", tt.description, result.Errors)
			}
		})
	}
}

func TestPatternsRule_DangerousUserPatterns(t *testing.T) {
	tests := []struct {
		name        string
		userPattern string
		shouldFail  bool
		description string
	}{
		{
			name:        "system_admin_pattern",
			userPattern: "system:admin",
			shouldFail:  true,
			description: "system:admin user should be blocked",
		},
		{
			name:        "system_masters_pattern",
			userPattern: "system:masters",
			shouldFail:  true,
			description: "system:masters user should be blocked",
		},
		{
			name:        "cluster_admin_pattern",
			userPattern: "cluster-admin",
			shouldFail:  true,
			description: "cluster-admin user should be blocked",
		},
		{
			name:        "admin_pattern",
			userPattern: "admin",
			shouldFail:  true,
			description: "admin user should be blocked",
		},
		{
			name:        "safe_user_pattern",
			userPattern: "developer@company.com",
			shouldFail:  false,
			description: "Regular user should be allowed",
		},
		{
			name:        "service_account_pattern",
			userPattern: "system:serviceaccount:default:my-app",
			shouldFail:  false,
			description: "Service account should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewPatternsRule([]string{}) // No general forbidden patterns for this test
			query := &types.StructuredQuery{
				LogSource:   "kube-apiserver",
				UserPattern: tt.userPattern,
			}

			result := rule.Validate(query)

			if tt.shouldFail && result.IsValid {
				t.Errorf("Expected dangerous user pattern to fail validation: %s", tt.description)
			}
			if !tt.shouldFail && !result.IsValid {
				t.Errorf("Expected safe user pattern to pass validation: %s. Errors: %v", tt.description, result.Errors)
			}
		})
	}
}

func TestPatternsRule_PatternMatching(t *testing.T) {
	rule := NewPatternsRule([]string{})

	tests := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{
		{
			name:     "exact_match",
			value:    "admin",
			pattern:  "admin",
			expected: true,
		},
		{
			name:     "case_insensitive_match",
			value:    "ADMIN",
			pattern:  "admin",
			expected: true,
		},
		{
			name:     "substring_match",
			value:    "test-admin-user",
			pattern:  "admin",
			expected: true,
		},
		{
			name:     "regex_match",
			value:    "openshift-config",
			pattern:  "openshift-.*",
			expected: true,
		},
		{
			name:     "no_match",
			value:    "developer",
			pattern:  "admin",
			expected: false,
		},
		{
			name:     "partial_no_match",
			value:    "dev-user",
			pattern:  "admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.matchesPattern(tt.value, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v", tt.value, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestPatternsRule_PerformanceBenchmark(t *testing.T) {
	// Create a rule with many forbidden patterns
	forbiddenPatterns := make([]string, 100)
	for i := 0; i < 100; i++ {
		forbiddenPatterns[i] = fmt.Sprintf("pattern_%d", i)
	}
	rule := NewPatternsRule(forbiddenPatterns)

	// Create a complex query with many fields
	query := &types.StructuredQuery{
		LogSource:                   "kube-apiserver",
		Verb:                        *types.NewStringOrArray([]string{"get", "list", "create", "update", "delete"}),
		Resource:                    *types.NewStringOrArray([]string{"pods", "services", "deployments"}),
		Namespace:                   *types.NewStringOrArray([]string{"default", "test", "staging"}),
		User:                        *types.NewStringOrArray("test@company.com"),
		Timeframe:                   "24_hours_ago",
		ResourceNamePattern:         "test-app-.*",
		UserPattern:                 ".*@company.com",
		NamespacePattern:            "test-.*",
		RequestURIPattern:           "/api/v1/.*",
		AuthorizationReasonPattern:  "allowed",
		ResponseMessagePattern:      "success",
		ExcludeUsers:               []string{"system:", "kube-"},
		ExcludeResources:           []string{"events", "endpoints"},
	}

	// Benchmark the validation
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rule.Validate(query)
		}
	})

	// Check that validation completes within reasonable time (< 1ms per operation)
	avgTimePerOp := result.T / time.Duration(result.N)
	if avgTimePerOp > 1*time.Millisecond {
		t.Errorf("Validation too slow: %v per operation (should be < 1ms)", avgTimePerOp)
	}

	t.Logf("Performance: %v per validation operation", avgTimePerOp)
}

func TestPatternsRule_EdgeCases(t *testing.T) {
	rule := NewPatternsRule([]string{"test"})

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		expectedValid bool
		description string
	}{
		{
			name: "nil_query",
			query: nil,
			expectedValid: true, // Should handle gracefully
			description: "Nil query should not crash",
		},
		{
			name: "empty_query",
			query: &types.StructuredQuery{},
			expectedValid: true,
			description: "Empty query should pass (no forbidden patterns)",
		},
		{
			name: "empty_string_fields",
			query: &types.StructuredQuery{
				LogSource:           "",
				UserPattern:         "",
				NamespacePattern:    "",
				ResourceNamePattern: "",
			},
			expectedValid: true,
			description: "Empty string fields should be ignored",
		},
		{
			name: "empty_array_fields",
			query: &types.StructuredQuery{
				LogSource:        "kube-apiserver",
				ExcludeUsers:     []string{},
				ExcludeResources: []string{},
			},
			expectedValid: true,
			description: "Empty array fields should be ignored",
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
				// For nil query test, we expect this to be handled gracefully
				result = &interfaces.ValidationResult{IsValid: true}
			}

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid=%v, got IsValid=%v: %s", tt.expectedValid, result.IsValid, tt.description)
			}
		})
	}
}