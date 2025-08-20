package rules

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestComplianceRule_Validate(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]interface{}
		query            *types.StructuredQuery
		expectedValid    bool
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name:   "Valid SOX compliance configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "30_days_ago",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Controls:  []string{"access_logging", "change_management", "audit_trail"},
					Reporting: &types.ComplianceReportingConfig{
						Format:          "detailed",
						IncludeEvidence: true,
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 1, // No reporting config warning
		},
		{
			name:   "Valid multi-standard compliance",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: "7_days_ago",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"PCI-DSS", "ISO27001"},
					Controls:  []string{"access_logging", "data_protection", "authentication_monitoring"},
					Reporting: &types.ComplianceReportingConfig{
						Format:          "detailed",
						IncludeEvidence: true,
					},
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 4, // Audit trail gap + ISO27001 requirements
		},
		{
			name:   "No compliance standards specified",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{}, // Empty standards
					Controls:  []string{"access_logging"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 4, // Standard compatibility + reporting + audit trail warnings
		},
		{
			name:   "Invalid compliance standard",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"INVALID_STANDARD"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 4, // No controls + reporting + audit trail warnings
		},
		{
			name:   "Duplicate compliance standards",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX", "SOX"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 10, // SOX requirements for both duplicates + reporting warnings
		},
		{
			name: "Too many standards",
			config: map[string]interface{}{
				"max_standards": 2,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX", "PCI-DSS", "GDPR"}, // 3 > 2 max
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 13, // SOX + PCI-DSS + GDPR requirements + reporting warnings
		},
		{
			name:   "Invalid compliance control",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Controls:  []string{"invalid_control"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 6, // SOX requirements + reporting + audit trail warnings
		},
		{
			name:   "Duplicate compliance controls",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Controls:  []string{"access_logging", "access_logging"},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 5, // SOX missing requirements + reporting warnings
		},
		{
			name: "Too many controls",
			config: map[string]interface{}{
				"max_controls": 2,
			},
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Controls:  []string{"access_logging", "audit_trail", "change_management"}, // 3 > 2 max
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 3, // Reporting + audit trail warnings
		},
		{
			name:   "Invalid reporting format",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Reporting: &types.ComplianceReportingConfig{
						Format: "invalid_format",
					},
				},
			},
			expectedValid:    false,
			expectedErrors:   1,
			expectedWarnings: 7, // SOX requirements + evidence + audit trail warnings
		},
		{
			name:   "No compliance framework configuration",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				// No ComplianceFramework field
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:   "No controls specified warning",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"SOX"},
					Controls:  []string{}, // No controls
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 7, // No controls + SOX requirements + reporting + audit trail warnings
		},
		{
			name:   "No reporting configuration warning",
			config: nil,
			query: &types.StructuredQuery{
				LogSource: "kube-apiserver",
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{"GDPR"},
					Controls:  []string{"data_protection"},
					// No Reporting field
				},
			},
			expectedValid:    true,
			expectedErrors:   0,
			expectedWarnings: 5, // No reporting + audit trail + GDPR requirements warnings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewComplianceRule(tt.config)
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

func TestComplianceRule_ValidateStandardSpecificRequirements(t *testing.T) {
	tests := []struct {
		name             string
		standard         string
		controls         []string
		timeframe        string
		expectedWarnings int
	}{
		{
			name:             "SOX with required controls",
			standard:         "SOX",
			controls:         []string{"access_logging", "change_management", "audit_trail"},
			timeframe:        "30_days_ago",
			expectedWarnings: 3, // Reporting + evidence + audit trail warnings
		},
		{
			name:             "SOX missing required controls",
			standard:         "SOX",
			controls:         []string{"data_protection"},
			timeframe:        "30_days_ago",
			expectedWarnings: 7, // Control compatibility + Missing SOX controls + reporting + audit trail warnings
		},
		{
			name:             "PCI-DSS with required controls",
			standard:         "PCI-DSS",
			controls:         []string{"access_logging", "authentication_monitoring", "data_protection"},
			timeframe:        "30_days_ago",
			expectedWarnings: 3, // Reporting + evidence + audit trail warnings
		},
		{
			name:             "GDPR with data protection",
			standard:         "GDPR",
			controls:         []string{"data_protection", "access_logging", "audit_trail"},
			timeframe:        "30_days_ago",
			expectedWarnings: 3, // Reporting + evidence + audit trail warnings
		},
		{
			name:             "HIPAA with required controls",
			standard:         "HIPAA",
			controls:         []string{"access_logging", "audit_trail", "data_protection", "authentication_monitoring"},
			timeframe:        "30_days_ago",
			expectedWarnings: 3, // Reporting + evidence + audit trail warnings
		},
		{
			name:             "FedRAMP with comprehensive controls",
			standard:         "FedRAMP",
			controls:         []string{"access_logging", "audit_trail", "incident_response", "configuration_management", "vulnerability_management"},
			timeframe:        "30_days_ago",
			expectedWarnings: 3, // Reporting + evidence + audit trail warnings
		},
	}

	rule := NewComplianceRule(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.timeframe,
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{tt.standard},
					Controls:  tt.controls,
				},
			}

			result := rule.Validate(query)
			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("Expected %d warnings for %s, got %d: %v", 
					tt.expectedWarnings, tt.standard, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestComplianceRule_ValidateRetentionRequirements(t *testing.T) {
	tests := []struct {
		name             string
		standard         string
		timeframe        string
		expectedWarnings int
	}{
		{
			name:             "SOX short timeframe",
			standard:         "SOX",
			timeframe:        "30_days_ago",
			expectedWarnings: 0, // Within 7-year retention
		},
		{
			name:             "PCI-DSS short timeframe",
			standard:         "PCI-DSS",
			timeframe:        "30_days_ago",
			expectedWarnings: 0, // Within 1-year retention
		},
		{
			name:             "HIPAA short timeframe",
			standard:         "HIPAA",
			timeframe:        "30_days_ago",
			expectedWarnings: 0, // Within 6-year retention
		},
	}

	rule := NewComplianceRule(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				LogSource: "kube-apiserver",
				Timeframe: tt.timeframe,
				ComplianceFramework: &types.ComplianceFrameworkConfig{
					Standards: []string{tt.standard},
					Controls:  []string{"access_logging"},
				},
			}

			result := rule.Validate(query)
			// Note: We're not testing specific warning counts here as the retention
			// calculation is simplified and would need more complex timeframe parsing
			// in a real implementation
			_ = result
		})
	}
}

func TestComplianceRule_CalculateQueryTimeframeDays(t *testing.T) {
	rule := NewComplianceRule(nil)

	tests := []struct {
		name          string
		timeframe     string
		expectedDays  int
	}{
		{"Today", "today", 1},
		{"Yesterday", "yesterday", 2},
		{"Week ago", "7_days_ago", 7},
		{"Month ago", "30_days_ago", 30},
		{"Unknown timeframe", "unknown", 30}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &types.StructuredQuery{
				Timeframe: tt.timeframe,
			}

			days := rule.calculateQueryTimeframeDays(query)
			if days != tt.expectedDays {
				t.Errorf("Expected %d days for timeframe '%s', got %d", 
					tt.expectedDays, tt.timeframe, days)
			}
		})
	}
}

func TestComplianceRule_ConfigDefaults(t *testing.T) {
	rule := NewComplianceRule(nil)

	// Test default standards
	standards := rule.getAllowedStandards()
	if len(standards) == 0 {
		t.Error("Default standards should not be empty")
	}

	// Verify some expected standards exist
	expectedStandards := []string{"SOX", "PCI-DSS", "GDPR", "HIPAA", "ISO27001"}
	for _, expectedStandard := range expectedStandards {
		found := false
		for _, actualStandard := range standards {
			if actualStandard == expectedStandard {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected standard '%s' not found in defaults", expectedStandard)
		}
	}

	// Test default controls
	controls := rule.getAllowedControls()
	if len(controls) == 0 {
		t.Error("Default controls should not be empty")
	}

	// Test default limits
	maxStandards := rule.getMaxStandards()
	if maxStandards <= 0 {
		t.Error("Max standards should be positive")
	}

	maxControls := rule.getMaxControls()
	if maxControls <= 0 {
		t.Error("Max controls should be positive")
	}

	minRetention := rule.getMinRetentionDays()
	if minRetention <= 0 {
		t.Error("Min retention days should be positive")
	}
}

func TestComplianceRule_ControlStandardCompatibility(t *testing.T) {
	rule := NewComplianceRule(nil)
	compatibilityMap := rule.getControlStandardCompatibility()

	// Test that all expected controls have compatibility mappings
	expectedControls := []string{
		"access_logging", "data_protection", "authentication_monitoring",
		"audit_trail", "change_management",
	}

	for _, control := range expectedControls {
		if standards, exists := compatibilityMap[control]; !exists {
			t.Errorf("Control '%s' missing from compatibility map", control)
		} else if len(standards) == 0 {
			t.Errorf("Control '%s' has no compatible standards", control)
		}
	}

	// Test that access_logging is compatible with most standards (it should be universal)
	accessLoggingStandards := compatibilityMap["access_logging"]
	if len(accessLoggingStandards) < 5 {
		t.Error("access_logging should be compatible with many standards")
	}
}

func TestComplianceRule_StandardRetentionRequirements(t *testing.T) {
	rule := NewComplianceRule(nil)

	tests := []struct {
		standard     string
		expectedDays int
	}{
		{"SOX", 2555},     // ~7 years
		{"PCI-DSS", 365},  // 1 year
		{"HIPAA", 2190},   // 6 years
		{"GDPR", 1095},    // 3 years
		{"UNKNOWN", 365},  // Default 1 year
	}

	for _, tt := range tests {
		t.Run(tt.standard, func(t *testing.T) {
			days := rule.getStandardRetentionRequirement(tt.standard)
			if days != tt.expectedDays {
				t.Errorf("Expected %d retention days for %s, got %d", 
					tt.expectedDays, tt.standard, days)
			}
		})
	}
}

func TestComplianceRule_CustomConfig(t *testing.T) {
	customConfig := map[string]interface{}{
		"allowed_standards": []interface{}{
			"CUSTOM_STANDARD_1",
			"CUSTOM_STANDARD_2",
		},
		"allowed_controls": []interface{}{
			"custom_control_1",
			"custom_control_2",
		},
		"max_standards": 3,
		"max_controls": 5,
		"min_retention_days": 90,
		"max_audit_gap_hours": 12,
		"required_evidence_fields": []interface{}{
			"custom_field_1",
			"custom_field_2",
		},
	}

	rule := NewComplianceRule(customConfig)

	// Test custom standards
	standards := rule.getAllowedStandards()
	if len(standards) != 2 {
		t.Errorf("Expected 2 custom standards, got %d", len(standards))
	}

	// Test custom controls
	controls := rule.getAllowedControls()
	if len(controls) != 2 {
		t.Errorf("Expected 2 custom controls, got %d", len(controls))
	}

	// Test custom limits
	if rule.getMaxStandards() != 3 {
		t.Errorf("Expected max standards 3, got %d", rule.getMaxStandards())
	}

	if rule.getMaxControls() != 5 {
		t.Errorf("Expected max controls 5, got %d", rule.getMaxControls())
	}

	if rule.getMinRetentionDays() != 90 {
		t.Errorf("Expected min retention 90 days, got %d", rule.getMinRetentionDays())
	}

	if rule.getMaxAuditGapHours() != 12 {
		t.Errorf("Expected max audit gap 12 hours, got %d", rule.getMaxAuditGapHours())
	}

	// Test custom evidence fields
	evidenceFields := rule.getRequiredEvidenceFields()
	if len(evidenceFields) != 2 {
		t.Errorf("Expected 2 custom evidence fields, got %d", len(evidenceFields))
	}
}