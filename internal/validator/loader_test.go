package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidationConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_config",
			configContent: `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver"
    - "kubelet"
  allowed_verbs:
    - "get"
    - "list"
  forbidden_patterns:
    - "DROP TABLE"
    - "rm -rf"
  required_fields:
    - "log_source"
    - "timeframe"
`,
			expectError: false,
		},
		{
			name: "empty_config",
			configContent: `
safety_rules:
  allowed_log_sources: []
  allowed_verbs: []
`,
			expectError: false,
		},
		{
			name: "invalid_yaml",
			configContent: `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver
  # Missing closing quote
`,
			expectError:   true,
			errorContains: "failed to parse config file",
		},
		{
			name: "minimal_valid_config",
			configContent: `
safety_rules: {}
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test_rules.yaml")
			
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Test loading the config
			config, err := LoadValidationConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("Expected non-nil config")
				return
			}

			// Verify config structure exists (field will be nil for empty config, which is expected)
			_ = config.SafetyRules.AllowedLogSources // Just verify field access works
		})
	}
}

func TestLoadValidationConfig_FileNotFound(t *testing.T) {
	_, err := LoadValidationConfig("nonexistent_file.yaml")
	
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	
	if !containsString(err.Error(), "failed to read config file") {
		t.Errorf("Expected error message about reading file, got: %s", err.Error())
	}
}

func TestLoadDefaultValidationConfig(t *testing.T) {
	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		os.Chdir(originalWd)
	}()

	// Test with valid config directory structure
	t.Run("with_valid_config", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		
		// Create configs directory and rules.yaml
		configDir := filepath.Join(tmpDir, "configs")
		err := os.Mkdir(configDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create configs directory: %v", err)
		}
		
		configContent := `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver"
    - "kubelet"
  allowed_verbs:
    - "get"
    - "list"
  forbidden_patterns:
    - "DROP TABLE"
  required_fields:
    - "log_source"
`
		configPath := filepath.Join(configDir, "rules.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create rules.yaml: %v", err)
		}

		config, err := LoadDefaultValidationConfig()
		if err != nil {
			t.Errorf("Unexpected error loading default config: %v", err)
		}
		
		if config == nil {
			t.Error("Expected non-nil config")
		}
		
		if len(config.SafetyRules.AllowedLogSources) != 2 {
			t.Errorf("Expected 2 allowed log sources, got %d", len(config.SafetyRules.AllowedLogSources))
		}
	})

	// Test with missing config file
	t.Run("with_missing_config", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		
		_, err := LoadDefaultValidationConfig()
		if err == nil {
			t.Error("Expected error when config file is missing")
		}
	})
}

func TestValidationConfig_Structure(t *testing.T) {
	configContent := `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver"
    - "kubelet"
  allowed_verbs:
    - "get"
    - "list"
    - "create"
  allowed_resources:
    - "pods"
    - "services"
  forbidden_patterns:
    - "DROP TABLE"
    - "rm -rf"
    - "curl"
  timeframe_limits:
    max_days: 30
    default_hours: 24
  sanitization:
    max_length: 1000
    forbidden_chars: ["<", ">", "&"]
  required_fields:
    - "log_source"
    - "timeframe"
  query_limits:
    max_results: 10000
    timeout_seconds: 300
  business_hours:
    start: "09:00"
    end: "17:00"
  analysis_limits:
    max_correlation_sources: 5
    max_pattern_complexity: 10
  response_status:
    allowed_codes: [200, 201, 400, 401, 403, 404, 500]
  auth_decisions:
    allowed_decisions: ["allow", "deny"]
  severity_levels:
    - "low"
    - "medium"
    - "high"
    - "critical"
  rule_categories:
    - "security"
    - "compliance"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_rules.yaml")
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadValidationConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test all structure fields are properly loaded
	if len(config.SafetyRules.AllowedLogSources) != 2 {
		t.Errorf("Expected 2 allowed log sources, got %d", len(config.SafetyRules.AllowedLogSources))
	}
	
	if len(config.SafetyRules.AllowedVerbs) != 3 {
		t.Errorf("Expected 3 allowed verbs, got %d", len(config.SafetyRules.AllowedVerbs))
	}
	
	if len(config.SafetyRules.AllowedResources) != 2 {
		t.Errorf("Expected 2 allowed resources, got %d", len(config.SafetyRules.AllowedResources))
	}
	
	if len(config.SafetyRules.ForbiddenPatterns) != 3 {
		t.Errorf("Expected 3 forbidden patterns, got %d", len(config.SafetyRules.ForbiddenPatterns))
	}
	
	if len(config.SafetyRules.RequiredFields) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(config.SafetyRules.RequiredFields))
	}
	
	if len(config.SafetyRules.SeverityLevels) != 4 {
		t.Errorf("Expected 4 severity levels, got %d", len(config.SafetyRules.SeverityLevels))
	}
	
	if len(config.SafetyRules.RuleCategories) != 2 {
		t.Errorf("Expected 2 rule categories, got %d", len(config.SafetyRules.RuleCategories))
	}

	// Test map fields are loaded
	if config.SafetyRules.TimeframeLimits == nil {
		t.Error("Expected TimeframeLimits to be loaded")
	}
	
	if config.SafetyRules.Sanitization == nil {
		t.Error("Expected Sanitization to be loaded")
	}
	
	if config.SafetyRules.QueryLimits == nil {
		t.Error("Expected QueryLimits to be loaded")
	}
}

func TestValidationConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty_rules.yaml")
	
	// Create empty file
	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty config file: %v", err)
	}

	config, err := LoadValidationConfig(configPath)
	if err != nil {
		t.Errorf("Unexpected error loading empty config: %v", err)
	}
	
	if config == nil {
		t.Error("Expected non-nil config even for empty file")
	}
}

// Benchmark configuration loading performance
func BenchmarkLoadValidationConfig(b *testing.B) {
	configContent := `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver"
    - "kubelet"
    - "kube-controller-manager"
    - "kube-scheduler"
    - "etcd"
  allowed_verbs:
    - "get"
    - "list"
    - "create"
    - "update"
    - "delete"
  forbidden_patterns:
    - "DROP TABLE"
    - "rm -rf"
    - "curl"
    - "wget"
    - "nc"
  required_fields:
    - "log_source"
    - "timeframe"
    - "verb"
`

	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "benchmark_rules.yaml")
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadValidationConfig(configPath)
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
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