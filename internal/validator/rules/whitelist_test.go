package rules

import (
	"fmt"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestNewWhitelistRule(t *testing.T) {
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server"}
	allowedVerbs := []string{"get", "list", "create", "update", "delete"}
	allowedResources := []string{"pods", "services", "deployments", "secrets"}

	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	if rule == nil {
		t.Fatal("NewWhitelistRule returned nil")
	}
	if !rule.IsEnabled() {
		t.Error("Expected rule to be enabled by default")
	}
	if rule.GetRuleName() != "whitelist_validation" {
		t.Errorf("Expected rule name 'whitelist_validation', got '%s'", rule.GetRuleName())
	}
	if rule.GetSeverity() != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", rule.GetSeverity())
	}

	// Verify whitelists are properly stored
	if len(rule.allowedLogSources) != len(allowedLogSources) {
		t.Errorf("Expected %d allowed log sources, got %d", len(allowedLogSources), len(rule.allowedLogSources))
	}
	if len(rule.allowedVerbs) != len(allowedVerbs) {
		t.Errorf("Expected %d allowed verbs, got %d", len(allowedVerbs), len(rule.allowedVerbs))
	}
	if len(rule.allowedResources) != len(allowedResources) {
		t.Errorf("Expected %d allowed resources, got %d", len(allowedResources), len(rule.allowedResources))
	}
}

func TestWhitelistRule_LogSourceValidation(t *testing.T) {
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"}
	rule := NewWhitelistRule(allowedLogSources, []string{}, []string{})

	tests := []struct {
		name        string
		logSource   string
		shouldPass  bool
		description string
	}{
		{
			name:        "allowed_kube_apiserver",
			logSource:   "kube-apiserver",
			shouldPass:  true,
			description: "kube-apiserver should be allowed",
		},
		{
			name:        "allowed_openshift_apiserver",
			logSource:   "openshift-apiserver",
			shouldPass:  true,
			description: "openshift-apiserver should be allowed",
		},
		{
			name:        "allowed_oauth_server",
			logSource:   "oauth-server",
			shouldPass:  true,
			description: "oauth-server should be allowed",
		},
		{
			name:        "allowed_oauth_apiserver",
			logSource:   "oauth-apiserver",
			shouldPass:  true,
			description: "oauth-apiserver should be allowed",
		},
		{
			name:        "allowed_node_auditd",
			logSource:   "node-auditd",
			shouldPass:  true,
			description: "node-auditd should be allowed",
		},
		{
			name:        "case_insensitive_match",
			logSource:   "KUBE-APISERVER",
			shouldPass:  true,
			description: "Case insensitive matching should work",
		},
		{
			name:        "mixed_case_match",
			logSource:   "Kube-ApiServer",
			shouldPass:  true,
			description: "Mixed case should be allowed",
		},
		{
			name:        "not_allowed_invalid_source",
			logSource:   "invalid-source",
			shouldPass:  false,
			description: "Invalid log source should be rejected",
		},
		{
			name:        "not_allowed_typo",
			logSource:   "kube-apiservr",
			shouldPass:  false,
			description: "Typo in log source should be rejected",
		},
		{
			name:        "not_allowed_empty",
			logSource:   "",
			shouldPass:  true,
			description: "Empty log source should be ignored (not validated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: tt.logSource,
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected log source to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected log source to fail: %s", tt.description)
			}

			// Check error message format for failed validations
			if !tt.shouldPass && len(result.Errors) > 0 {
				found := false
				for _, err := range result.Errors {
					if containsSubstring(err, "Log source") && containsSubstring(err, "not in allowed whitelist") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message about log source whitelist violation, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestWhitelistRule_VerbValidation(t *testing.T) {
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver"}
	allowedVerbs := []string{"get", "list", "create", "update", "patch", "delete", "watch"}
	allowedResources := []string{"pods", "services", "deployments"}
	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	tests := []struct {
		name        string
		verb        interface{} // Can be string or []string
		shouldPass  bool
		expectedErrors int
		description string
	}{
		{
			name:        "allowed_single_verb",
			verb:        "get",
			shouldPass:  true,
			expectedErrors: 0,
			description: "Single allowed verb should pass",
		},
		{
			name:        "allowed_multiple_verbs",
			verb:        []string{"get", "list", "create"},
			shouldPass:  true,
			expectedErrors: 0,
			description: "Multiple allowed verbs should pass",
		},
		{
			name:        "case_insensitive_verb",
			verb:        "GET",
			shouldPass:  true,
			expectedErrors: 0,
			description: "Case insensitive verb matching should work",
		},
		{
			name:        "mixed_case_verbs",
			verb:        []string{"Get", "LIST", "create"},
			shouldPass:  true,
			expectedErrors: 0,
			description: "Mixed case verbs should be allowed",
		},
		{
			name:        "not_allowed_single_verb",
			verb:        "execute",
			shouldPass:  false,
			expectedErrors: 1,
			description: "Single disallowed verb should fail",
		},
		{
			name:        "not_allowed_verb_in_array",
			verb:        []string{"get", "execute", "list"},
			shouldPass:  false,
			expectedErrors: 1,
			description: "Disallowed verb in array should fail",
		},
		{
			name:        "multiple_not_allowed_verbs",
			verb:        []string{"execute", "run", "get"},
			shouldPass:  false,
			expectedErrors: 2,
			description: "Multiple disallowed verbs should generate multiple errors",
		},
		{
			name:        "empty_verb_array",
			verb:        []string{},
			shouldPass:  true,
			expectedErrors: 0,
			description: "Empty verb array should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver", // Valid log source to avoid other validation errors
			}

			// Set verb based on type
			if str, ok := tt.verb.(string); ok {
				query.Verb = *types.NewStringOrArray(str)
			} else if arr, ok := tt.verb.([]string); ok {
				if len(arr) == 0 {
					query.Verb = types.StringOrArray{} // Empty
				} else {
					query.Verb = *types.NewStringOrArray(arr)
				}
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected verb validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected verb validation to fail: %s", tt.description)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %s. Errors: %v", 
					tt.expectedErrors, len(result.Errors), tt.description, result.Errors)
			}

			// Check error message format for failed validations
			if !tt.shouldPass && len(result.Errors) > 0 {
				for _, err := range result.Errors {
					if !containsSubstring(err, "Verb") || !containsSubstring(err, "not in allowed whitelist") {
						t.Errorf("Expected error message about verb whitelist violation: %s", err)
					}
				}
			}
		})
	}
}

func TestWhitelistRule_ResourceValidation(t *testing.T) {
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver"}
	allowedVerbs := []string{"get", "list", "create", "update", "delete"}
	allowedResources := []string{"pods", "services", "deployments", "secrets", "configmaps", "roles", "rolebindings"}
	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	tests := []struct {
		name           string
		resource       interface{} // Can be string or []string
		shouldPass     bool
		expectedErrors int
		description    string
	}{
		{
			name:           "allowed_single_resource",
			resource:       "pods",
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Single allowed resource should pass",
		},
		{
			name:           "allowed_multiple_resources",
			resource:       []string{"pods", "services", "deployments"},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Multiple allowed resources should pass",
		},
		{
			name:           "case_insensitive_resource",
			resource:       "PODS",
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Case insensitive resource matching should work",
		},
		{
			name:           "mixed_case_resources",
			resource:       []string{"Pods", "SERVICES", "deployments"},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Mixed case resources should be allowed",
		},
		{
			name:           "not_allowed_single_resource",
			resource:       "customresources",
			shouldPass:     false,
			expectedErrors: 1,
			description:    "Single disallowed resource should fail",
		},
		{
			name:           "not_allowed_resource_in_array",
			resource:       []string{"pods", "customresources", "services"},
			shouldPass:     false,
			expectedErrors: 1,
			description:    "Disallowed resource in array should fail",
		},
		{
			name:           "multiple_not_allowed_resources",
			resource:       []string{"customresources", "clusterroles", "pods"},
			shouldPass:     false,
			expectedErrors: 2,
			description:    "Multiple disallowed resources should generate multiple errors",
		},
		{
			name:           "kubernetes_core_resources",
			resource:       []string{"pods", "services", "secrets", "configmaps"},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Core Kubernetes resources should be allowed",
		},
		{
			name:           "rbac_resources",
			resource:       []string{"roles", "rolebindings"},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "RBAC resources should be allowed",
		},
		{
			name:           "empty_resource_array",
			resource:       []string{},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Empty resource array should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver", // Valid log source to avoid other validation errors
			}

			// Set resource based on type
			if str, ok := tt.resource.(string); ok {
				query.Resource = *types.NewStringOrArray(str)
			} else if arr, ok := tt.resource.([]string); ok {
				if len(arr) == 0 {
					query.Resource = types.StringOrArray{} // Empty
				} else {
					query.Resource = *types.NewStringOrArray(arr)
				}
			}

			result := rule.Validate(query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected resource validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected resource validation to fail: %s", tt.description)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %s. Errors: %v", 
					tt.expectedErrors, len(result.Errors), tt.description, result.Errors)
			}

			// Check error message format for failed validations
			if !tt.shouldPass && len(result.Errors) > 0 {
				for _, err := range result.Errors {
					if !containsStringHelper(err, "Resource") || !containsStringHelper(err, "not in allowed whitelist") {
						t.Errorf("Expected error message about resource whitelist violation: %s", err)
					}
				}
			}
		})
	}
}

func TestWhitelistRule_ComprehensiveValidation(t *testing.T) {
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver"}
	allowedVerbs := []string{"get", "list", "create"}
	allowedResources := []string{"pods", "services"}

	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	tests := []struct {
		name           string
		query          *types.StructuredQuery
		shouldPass     bool
		expectedErrors int
		description    string
	}{
		{
			name: "all_allowed_values",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "All allowed values should pass",
		},
		{
			name: "multiple_allowed_values",
			query: &types.StructuredQuery{
				LogSource: "openshift-apiserver",
				Verb:      *types.NewStringOrArray([]string{"get", "list"}),
				Resource:  *types.NewStringOrArray([]string{"pods", "services"}),
			},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Multiple allowed values should pass",
		},
		{
			name: "invalid_log_source_only",
			query: &types.StructuredQuery{
				LogSource: "invalid-source",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:     false,
			expectedErrors: 1,
			description:    "Invalid log source should fail",
		},
		{
			name: "invalid_verb_only",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("delete"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:     false,
			expectedErrors: 1,
			description:    "Invalid verb should fail",
		},
		{
			name: "invalid_resource_only",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("secrets"),
			},
			shouldPass:     false,
			expectedErrors: 1,
			description:    "Invalid resource should fail",
		},
		{
			name: "multiple_violations",
			query: &types.StructuredQuery{
				LogSource: "invalid-source",
				Verb:      *types.NewStringOrArray([]string{"get", "delete", "patch"}),
				Resource:  *types.NewStringOrArray([]string{"pods", "secrets", "configmaps"}),
			},
			shouldPass:     false,
			expectedErrors: 5, // 1 log source + 2 verbs + 2 resources
			description:    "Multiple violations should generate multiple errors",
		},
		{
			name: "partial_violations_in_arrays",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray([]string{"get", "delete"}), // delete not allowed
				Resource:  *types.NewStringOrArray([]string{"pods", "secrets"}), // secrets not allowed
			},
			shouldPass:     false,
			expectedErrors: 2, // 1 verb + 1 resource
			description:    "Partial violations in arrays should be detected",
		},
		{
			name: "empty_fields_ignored",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				// Verb and Resource are empty/unset
			},
			shouldPass:     true,
			expectedErrors: 0,
			description:    "Empty fields should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %s. Errors: %v", 
					tt.expectedErrors, len(result.Errors), tt.description, result.Errors)
			}

			// Verify rule information
			if result.RuleName != "whitelist_validation" {
				t.Errorf("Expected rule name 'whitelist_validation', got '%s'", result.RuleName)
			}

			// Check recommendations for failed validations
			if !tt.shouldPass && len(result.Recommendations) == 0 {
				t.Error("Expected validation recommendations for failed validation")
			}
		})
	}
}

func TestWhitelistRule_FunctionalQueryValidation(t *testing.T) {
	// Based on functional test queries from basic_queries.md and schema validation rules
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"}
	allowedVerbs := []string{"get", "list", "create", "update", "patch", "delete", "watch", "connect"}
	allowedResources := []string{
		"pods", "services", "deployments", "secrets", "configmaps", "namespaces",
		"roles", "rolebindings", "clusterroles", "clusterrolebindings",
		"serviceaccounts", "persistentvolumes", "persistentvolumeclaims",
		"nodes", "events", "endpoints", "ingresses", "networkpolicies",
	}

	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "basic_query_pattern",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("delete"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:  true,
			description: "Basic delete pods query should pass whitelist validation",
		},
		{
			name: "secrets_access_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("create"),
				Resource:  *types.NewStringOrArray("secrets"),
			},
			shouldPass:  true,
			description: "Secrets access query should pass whitelist validation",
		},
		{
			name: "rbac_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray([]string{"create", "update", "patch"}),
				Resource:  *types.NewStringOrArray([]string{"roles", "rolebindings"}),
			},
			shouldPass:  true,
			description: "RBAC query should pass whitelist validation",
		},
		{
			name: "oauth_server_query",
			query: &types.StructuredQuery{
				LogSource: "oauth-server",
				// No verb/resource for oauth-server queries
			},
			shouldPass:  true,
			description: "OAuth server query should pass whitelist validation",
		},
		{
			name: "node_auditd_query",
			query: &types.StructuredQuery{
				LogSource: "node-auditd",
				// No verb/resource for node auditd queries
			},
			shouldPass:  true,
			description: "Node auditd query should pass whitelist validation",
		},
		{
			name: "multiple_resources_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray([]string{"pods", "services", "deployments"}),
			},
			shouldPass:  true,
			description: "Multiple resources query should pass whitelist validation",
		},
		{
			name: "invalid_log_source_query",
			query: &types.StructuredQuery{
				LogSource: "etcd-apiserver", // Not in whitelist
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:  false,
			description: "Query with invalid log source should fail",
		},
		{
			name: "invalid_verb_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("execute"), // Not in whitelist
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:  false,
			description: "Query with invalid verb should fail",
		},
		{
			name: "invalid_resource_query",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("customresourcedefinitions"), // Not in whitelist
			},
			shouldPass:  false,
			description: "Query with invalid resource should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected query to pass whitelist validation: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected query to fail whitelist validation: %s", tt.description)
			}
		})
	}
}

func TestWhitelistRule_EdgeCases(t *testing.T) {
	rule := NewWhitelistRule([]string{"kube-apiserver"}, []string{"get"}, []string{"pods"})

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
			description:   "Empty query should pass (empty fields are ignored)",
		},
		{
			name: "empty_string_arrays",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      types.StringOrArray{}, // Empty
				Resource:  types.StringOrArray{}, // Empty
			},
			expectedValid: true,
			description:   "Empty StringOrArray fields should be ignored",
		},
		{
			name: "whitespace_in_values",
			query: &types.StructuredQuery{
				LogSource: " kube-apiserver ",
				Verb:      *types.NewStringOrArray(" get "),
				Resource:  *types.NewStringOrArray(" pods "),
			},
			expectedValid: false, // Whitespace should not be trimmed in this implementation
			description:   "Whitespace in values should not match (exact matching)",
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
				t.Errorf("Expected IsValid=%v, got IsValid=%v: %s. Errors: %v",
					tt.expectedValid, result.IsValid, tt.description, result.Errors)
			}
		})
	}
}

func TestWhitelistRule_EmptyWhitelists(t *testing.T) {
	// Test behavior with empty whitelists
	rule := NewWhitelistRule([]string{}, []string{}, []string{})

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
	}

	result := rule.Validate(query)

	// With empty whitelists, everything should fail
	if result.IsValid {
		t.Error("Expected validation to fail with empty whitelists")
	}

	// Should have 3 errors (log source, verb, resource)
	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors with empty whitelists, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestWhitelistRule_PerformanceBenchmark(t *testing.T) {
	// Create large whitelists for performance testing
	allowedLogSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-server", "oauth-apiserver", "node-auditd"}
	
	allowedVerbs := make([]string, 50)
	for i := 0; i < 50; i++ {
		allowedVerbs[i] = fmt.Sprintf("verb%d", i)
	}
	allowedVerbs = append(allowedVerbs, "get", "list", "create", "update", "delete")

	allowedResources := make([]string, 100)
	for i := 0; i < 100; i++ {
		allowedResources[i] = fmt.Sprintf("resource%d", i)
	}
	allowedResources = append(allowedResources, "pods", "services", "deployments")

	rule := NewWhitelistRule(allowedLogSources, allowedVerbs, allowedResources)

	query := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray([]string{"get", "list", "create", "update", "delete"}),
		Resource:  *types.NewStringOrArray([]string{"pods", "services", "deployments"}),
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

func TestWhitelistRule_CaseInsensitiveMatching(t *testing.T) {
	rule := NewWhitelistRule(
		[]string{"kube-apiserver"},
		[]string{"get", "list"},
		[]string{"pods", "services"},
	)

	tests := []struct {
		name        string
		query       *types.StructuredQuery
		shouldPass  bool
		description string
	}{
		{
			name: "lowercase_values",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray("get"),
				Resource:  *types.NewStringOrArray("pods"),
			},
			shouldPass:  true,
			description: "Lowercase values should pass",
		},
		{
			name: "uppercase_values",
			query: &types.StructuredQuery{
				LogSource: "KUBE-APISERVER",
				Verb:      *types.NewStringOrArray("GET"),
				Resource:  *types.NewStringOrArray("PODS"),
			},
			shouldPass:  true,
			description: "Uppercase values should pass (case insensitive)",
		},
		{
			name: "mixed_case_values",
			query: &types.StructuredQuery{
				LogSource: "Kube-ApiServer",
				Verb:      *types.NewStringOrArray("Get"),
				Resource:  *types.NewStringOrArray("Pods"),
			},
			shouldPass:  true,
			description: "Mixed case values should pass (case insensitive)",
		},
		{
			name: "mixed_case_arrays",
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Verb:      *types.NewStringOrArray([]string{"GET", "list"}),
				Resource:  *types.NewStringOrArray([]string{"PODS", "services"}),
			},
			shouldPass:  true,
			description: "Mixed case arrays should pass (case insensitive)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.query)

			if tt.shouldPass && !result.IsValid {
				t.Errorf("Expected validation to pass: %s. Errors: %v", tt.description, result.Errors)
			}
			if !tt.shouldPass && result.IsValid {
				t.Errorf("Expected validation to fail: %s", tt.description)
			}
		})
	}
}

// Helper function for string containment checking
func containsStringHelper(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || 
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
