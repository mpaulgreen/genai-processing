package rules

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestMultiSourceRule_Validate(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		query          *types.StructuredQuery
		expectedValid  bool
		expectedErrors int
		expectedWarnings int
	}{
		{
			name:   "Valid multi-source configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server", "node-auditd"},
					CorrelationWindow: "30_minutes",
					CorrelationFields: []string{"user", "source_ip", "timestamp"},
					JoinType:          "inner",
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1, // node-auditd may have limited fields
		},
		{
			name:   "Missing primary source",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					SecondarySources: []string{"oauth-server"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name:   "Missing secondary sources",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource: "kube-apiserver",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name:   "Invalid primary source",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "invalid-source",
					SecondarySources: []string{"oauth-server"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name:   "Invalid secondary source",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"invalid-source"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name:   "Duplicate sources",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"kube-apiserver", "oauth-server"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name: "Too many sources",
			config: map[string]interface{}{
				"max_sources": 3,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server", "node-auditd", "openshift-apiserver"}, // 4 total > 3 max
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 3, // Performance warning + correlation window + correlation fields
		},
		{
			name:   "Invalid correlation window",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server"},
					CorrelationWindow: "invalid_window",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // No correlation fields warning
		},
		{
			name: "Too many correlation fields",
			config: map[string]interface{}{
				"max_correlation_fields": 2,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server"},
					CorrelationFields: []string{"user", "source_ip", "timestamp"}, // 3 > 2 max
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // No correlation window warning
		},
		{
			name:   "Invalid correlation field",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server"},
					CorrelationFields: []string{"invalid_field"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + field compatibility warning
		},
		{
			name:   "Duplicate correlation fields",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server"},
					CorrelationFields: []string{"user", "user"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 1, // No correlation window warning
		},
		{
			name:   "Invalid join type",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server"},
					JoinType:         "invalid_join",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 2, // No correlation window + No correlation fields
		},
		{
			name: "High complexity",
			config: map[string]interface{}{
				"max_correlation_complexity": 50,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server", "node-auditd", "openshift-apiserver"},
					CorrelationWindow: "24_hours",
					CorrelationFields: []string{"user", "source_ip", "timestamp", "namespace", "resource"},
					JoinType:          "full",
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 5, // Performance + large window + field compatibility + join type warnings
		},
		{
			name:   "No multi-source configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				// No MultiSource field
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name:   "Performance warning for large window",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"oauth-server"},
					CorrelationWindow: "24_hours",
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 2, // Large window + no correlation fields
		},
		{
			name:   "Performance warning for expensive join",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:    "kube-apiserver",
					SecondarySources: []string{"oauth-server"},
					JoinType:         "full",
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 3, // No correlation window + No correlation fields + Expensive join
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewMultiSourceRule(tt.config)
			result := rule.Validate(tt.query)

			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v, got %v", tt.expectedValid, result.IsValid)
			}

			if len(result.Errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(result.Errors), result.Errors)
			}

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("Expected %d warnings, got %d: %v", tt.expectedWarnings, len(result.Warnings), result.Warnings)
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

func TestMultiSourceRule_ValidatePrimarySource(t *testing.T) {
	tests := []struct {
		name          string
		primarySource string
		expectedValid bool
	}{
		{"Valid kube-apiserver", "kube-apiserver", true},
		{"Valid oauth-server", "oauth-server", true},
		{"Valid node-auditd", "node-auditd", true},
		{"Invalid source", "invalid-source", false},
		{"Empty source", "", false},
	}

	rule := NewMultiSourceRule(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Choose a different secondary source than the primary
			secondarySource := "oauth-server"
			if tt.primarySource == "oauth-server" {
				secondarySource = "node-auditd"
			}
			
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     tt.primarySource,
					SecondarySources:  []string{secondarySource},
					CorrelationWindow: "30_minutes",
					CorrelationFields: []string{"user", "source_ip"},
				},
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for primary source '%s', got %v", 
					tt.expectedValid, tt.primarySource, result.IsValid)
			}
		})
	}
}

func TestMultiSourceRule_ValidateCorrelationFields(t *testing.T) {
	tests := []struct {
		name              string
		correlationFields []string
		expectedValid     bool
		expectedWarnings  int
	}{
		{
			name:              "Valid fields",
			correlationFields: []string{"user", "source_ip", "timestamp"},
			expectedValid:     true,
			expectedWarnings:  0, // No warnings with proper setup
		},
		{
			name:              "No fields (warning)",
			correlationFields: []string{},
			expectedValid:     true,
			expectedWarnings:  1, // No correlation fields warning
		},
		{
			name:              "Invalid field",
			correlationFields: []string{"invalid_field"},
			expectedValid:     false,
			expectedWarnings:  1, // Field compatibility warning
		},
		{
			name:              "Mixed valid and invalid",
			correlationFields: []string{"user", "invalid_field"},
			expectedValid:     false,
			expectedWarnings:  1, // Field compatibility warning
		},
		{
			name:              "Field incompatible with node-auditd",
			correlationFields: []string{"namespace"},
			expectedValid:     true,
			expectedWarnings:  1, // Node-auditd compatibility warning
		},
	}

	rule := NewMultiSourceRule(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				MultiSource: &types.MultiSourceConfig{
					PrimarySource:     "kube-apiserver",
					SecondarySources:  []string{"node-auditd"},
					CorrelationWindow: "30_minutes",
					CorrelationFields: tt.correlationFields,
				},
			}

			result := rule.Validate(query)
			if result.IsValid != tt.expectedValid {
				t.Errorf("Expected IsValid = %v for fields %v, got %v", 
					tt.expectedValid, tt.correlationFields, result.IsValid)
			}

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("Expected %d warnings for fields %v, got %d: %v", 
					tt.expectedWarnings, tt.correlationFields, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestMultiSourceRule_CalculateCorrelationComplexity(t *testing.T) {
	rule := NewMultiSourceRule(nil)

	tests := []struct {
		name               string
		config             *types.MultiSourceConfig
		expectedComplexity int
	}{
		{
			name: "Simple correlation",
			config: &types.MultiSourceConfig{
				PrimarySource:     "kube-apiserver",
				SecondarySources:  []string{"oauth-server"},
				CorrelationWindow: "1_minute",
				CorrelationFields: []string{"user"},
				JoinType:          "inner",
			},
			expectedComplexity: 17, // 1*10 + 1*5 + 1 + 1
		},
		{
			name: "Complex correlation",
			config: &types.MultiSourceConfig{
				PrimarySource:     "kube-apiserver",
				SecondarySources:  []string{"oauth-server", "node-auditd"},
				CorrelationWindow: "24_hours",
				CorrelationFields: []string{"user", "source_ip", "timestamp"},
				JoinType:          "full",
			},
			expectedComplexity: 60, // 2*10 + 3*5 + 20 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := rule.calculateCorrelationComplexity(tt.config)
			if complexity != tt.expectedComplexity {
				t.Errorf("Expected complexity %d, got %d", tt.expectedComplexity, complexity)
			}
		})
	}
}

func TestMultiSourceRule_SourceFieldCompatibility(t *testing.T) {
	rule := NewMultiSourceRule(nil)
	fieldMap := rule.getSourceFieldCompatibilityMap()

	// Test that all expected sources have field mappings
	expectedSources := []string{
		"kube-apiserver", "openshift-apiserver", 
		"oauth-server", "oauth-apiserver", "node-auditd",
	}

	for _, source := range expectedSources {
		if fields, exists := fieldMap[source]; !exists {
			t.Errorf("Source '%s' missing from field compatibility map", source)
		} else if len(fields) == 0 {
			t.Errorf("Source '%s' has no compatible fields", source)
		}
	}

	// Test that node-auditd has limited fields (as expected)
	nodeFields := fieldMap["node-auditd"]
	if len(nodeFields) >= len(fieldMap["kube-apiserver"]) {
		t.Error("node-auditd should have fewer fields than kube-apiserver")
	}

	// Test that common fields exist across API sources
	commonFields := []string{"user", "source_ip", "timestamp"}
	apiSources := []string{"kube-apiserver", "openshift-apiserver", "oauth-apiserver"}

	for _, field := range commonFields {
		for _, source := range apiSources {
			sourceFields := fieldMap[source]
			if !rule.isValueInSlice(field, sourceFields) {
				t.Errorf("Common field '%s' missing from source '%s'", field, source)
			}
		}
	}
}

func TestMultiSourceRule_ConfigDefaults(t *testing.T) {
	rule := NewMultiSourceRule(nil)

	// Test default max sources
	maxSources := rule.getMaxSources()
	if maxSources <= 0 {
		t.Error("Max sources should be positive")
	}

	// Test default correlation windows
	windows := rule.getAllowedCorrelationWindows()
	if len(windows) == 0 {
		t.Error("Allowed correlation windows should not be empty")
	}

	// Test default correlation fields
	fields := rule.getAllowedCorrelationFields()
	if len(fields) == 0 {
		t.Error("Allowed correlation fields should not be empty")
	}

	// Test default max complexity
	maxComplexity := rule.getMaxCorrelationComplexity()
	if maxComplexity <= 0 {
		t.Error("Max correlation complexity should be positive")
	}
}

func TestMultiSourceRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"max_sources": 3,
		"allowed_correlation_windows": []interface{}{
			"1_minute", "5_minutes", "15_minutes",
		},
		"max_correlation_fields": 5,
		"allowed_correlation_fields": []interface{}{
			"user", "source_ip", "timestamp",
		},
		"max_correlation_complexity": 75,
	}

	rule := NewMultiSourceRule(customConfig)

	// Test custom max sources
	if rule.getMaxSources() != 3 {
		t.Errorf("Expected max sources 3, got %d", rule.getMaxSources())
	}

	// Test custom correlation windows
	windows := rule.getAllowedCorrelationWindows()
	if len(windows) != 3 {
		t.Errorf("Expected 3 correlation windows, got %d", len(windows))
	}

	// Test custom correlation fields
	fields := rule.getAllowedCorrelationFields()
	if len(fields) != 3 {
		t.Errorf("Expected 3 correlation fields, got %d", len(fields))
	}

	// Test custom max complexity
	if rule.getMaxCorrelationComplexity() != 75 {
		t.Errorf("Expected max complexity 75, got %d", rule.getMaxCorrelationComplexity())
	}
}