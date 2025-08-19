package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/test/config")
	if loader.configDir != "/test/config" {
		t.Errorf("Expected configDir to be '/test/config', got '%s'", loader.configDir)
	}
}

func TestLoadConfig_DefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Load configuration (should use defaults since no files exist)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading default config, got: %v", err)
	}

	// Verify default values
	if config.Server.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", config.Server.Port)
	}
	if config.Models.DefaultProvider != "claude" {
		t.Errorf("Expected default provider 'claude', got %s", config.Models.DefaultProvider)
	}
	if len(config.Models.Providers) == 0 {
		t.Error("Expected default providers to be configured")
	}
}

func TestLoadConfig_WithModelsFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a models.yaml file
	modelsYAML := `
default_provider: "openai"
providers:
  openai:
    provider: "openai"
    endpoint: "https://api.openai.com/v1/chat/completions"
    api_key: "test-key"
    model_name: "gpt-4"
    max_tokens: 2000
    temperature: 0.2
    timeout: "30s"
    retry_attempts: 2
    retry_delay: "500ms"
    input_adapter: "openai_input_adapter"
    output_parser: "openai_extractor"
    prompt_formatter: "openai_formatter"
`

	modelsPath := filepath.Join(tempDir, "models.yaml")
	if err := os.WriteFile(modelsPath, []byte(modelsYAML), 0644); err != nil {
		t.Fatalf("Failed to write test models file: %v", err)
	}

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with models file, got: %v", err)
	}

	// Verify loaded values
	if config.Models.DefaultProvider != "openai" {
		t.Errorf("Expected default provider 'openai', got %s", config.Models.DefaultProvider)
	}

	openaiProvider, exists := config.Models.Providers["openai"]
	if !exists {
		t.Fatal("Expected openai provider to exist")
	}

	if openaiProvider.ModelName != "gpt-4" {
		t.Errorf("Expected model name 'gpt-4', got %s", openaiProvider.ModelName)
	}
	if openaiProvider.MaxTokens != 2000 {
		t.Errorf("Expected max tokens 2000, got %d", openaiProvider.MaxTokens)
	}
	if openaiProvider.Temperature != 0.2 {
		t.Errorf("Expected temperature 0.2, got %f", openaiProvider.Temperature)
	}
	if openaiProvider.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", openaiProvider.Timeout)
	}
}

func TestLoadConfig_WithPromptsFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a prompts.yaml file
	promptsYAML := `
system_prompts:
  base: "Custom base prompt"
  # Removed redundant system prompts - using base only

examples:
  - input: "Test query"
    output: '{"test": "response"}'

formats:
  claude:
    template: "Custom Claude template"
  openai:
    template: "Custom OpenAI template"
  generic:
    template: "Custom generic template"
`

	promptsPath := filepath.Join(tempDir, "prompts.yaml")
	if err := os.WriteFile(promptsPath, []byte(promptsYAML), 0644); err != nil {
		t.Fatalf("Failed to write test prompts file: %v", err)
	}

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with prompts file, got: %v", err)
	}

	// Verify loaded values
	if config.Prompts.SystemPrompts["base"] != "Custom base prompt" {
		t.Errorf("Expected custom base prompt, got %s", config.Prompts.SystemPrompts["base"])
	}

	if len(config.Prompts.Examples) != 1 {
		t.Errorf("Expected 1 example, got %d", len(config.Prompts.Examples))
	}

	if config.Prompts.Examples[0].Input != "Test query" {
		t.Errorf("Expected example input 'Test query', got %s", config.Prompts.Examples[0].Input)
	}

	if config.Prompts.Formats.Claude.Template != "Custom Claude template" {
		t.Errorf("Expected custom Claude template, got %s", config.Prompts.Formats.Claude.Template)
	}

	// Note: Validation configuration has been moved to rules.yaml
}

func TestLoadConfig_EnvironmentOverrides(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Set environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("DEFAULT_PROVIDER", "openai")
	os.Setenv("CLAUDE_API_KEY", "env-claude-key")
	os.Setenv("OPENAI_MODEL_NAME", "gpt-3.5-turbo")
	os.Setenv("OPENAI_MAX_TOKENS", "3000")
	os.Setenv("OPENAI_TEMPERATURE", "0.3")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("DEFAULT_PROVIDER")
		os.Unsetenv("CLAUDE_API_KEY")
		os.Unsetenv("OPENAI_MODEL_NAME")
		os.Unsetenv("OPENAI_MAX_TOKENS")
		os.Unsetenv("OPENAI_TEMPERATURE")
	}()

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with environment overrides, got: %v", err)
	}

	// Verify environment variable overrides
	if config.Server.Port != "9090" {
		t.Errorf("Expected port 9090 from environment, got %s", config.Server.Port)
	}
	if config.Server.Host != "localhost" {
		t.Errorf("Expected host localhost from environment, got %s", config.Server.Host)
	}
	if config.Models.DefaultProvider != "openai" {
		t.Errorf("Expected default provider 'openai' from environment, got %s", config.Models.DefaultProvider)
	}

	claudeProvider := config.Models.Providers["claude"]
	if claudeProvider.APIKey != "env-claude-key" {
		t.Errorf("Expected Claude API key from environment, got %s", claudeProvider.APIKey)
	}

	openaiProvider := config.Models.Providers["openai"]
	if openaiProvider.ModelName != "gpt-3.5-turbo" {
		t.Errorf("Expected OpenAI model name from environment, got %s", openaiProvider.ModelName)
	}
	if openaiProvider.MaxTokens != 3000 {
		t.Errorf("Expected OpenAI max tokens from environment, got %d", openaiProvider.MaxTokens)
	}
	if openaiProvider.Temperature != 0.3 {
		t.Errorf("Expected OpenAI temperature from environment, got %f", openaiProvider.Temperature)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Ensure deterministic environment (unset server-related overrides that could affect this test)
	envKeys := []string{
		"SERVER_PORT",
		"SERVER_HOST",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
		"SERVER_SHUTDOWN_TIMEOUT",
		"SERVER_MAX_REQUEST_SIZE",
	}
	original := make(map[string]*string)
	for _, k := range envKeys {
		if v, ok := os.LookupEnv(k); ok {
			vv := v
			original[k] = &vv
			os.Unsetenv(k)
		} else {
			original[k] = nil
		}
	}
	defer func() {
		for k, v := range original {
			if v == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *v)
			}
		}
	}()

	// Create a complete config file
	configYAML := `
server:
  port: "9090"
  host: "localhost"
  read_timeout: "45s"
  write_timeout: "45s"
  idle_timeout: "90s"
  shutdown_timeout: "15s"
  max_request_size: 2097152

models:
  default_provider: "openai"
  providers:
    openai:
      provider: "openai"
      endpoint: "https://api.openai.com/v1/chat/completions"
      api_key: "test-key"
      model_name: "gpt-4"
      max_tokens: 2000
      temperature: 0.2
      timeout: "30s"
      retry_attempts: 2
      retry_delay: "500ms"

prompts:
  system_prompts:
    base: "Test base prompt"
    claude_specific: "Test Claude prompt"
    openai_specific: "Test OpenAI prompt"
  examples:
    - input: "Test query"
      output: '{"test": "response"}'
  formats:
    claude:
      template: "Test Claude template"
    openai:
      template: "Test OpenAI template"
    generic:
      template: "Test generic template"
  validation:
    max_input_length: 500
    max_output_length: 1000
    required_fields: ["log_source"]

rules:
  safety_rules:
    allowed_log_sources:
      - "kube-apiserver"
      - "openshift-apiserver"
    allowed_verbs:
      - "get"
      - "list"
      - "create"
    allowed_resources:
      - "pods"
      - "services"
    forbidden_patterns:
      - "rm -rf"
    timeframe_limits:
      max_days_back: 90
      default_limit: 20
      max_limit: 1000
      min_limit: 1
      allowed_timeframes:
        - "today"
        - "yesterday"
    required_fields:
      - "log_source"
  sanitization:
    max_query_length: 10000
    max_pattern_length: 500
    max_user_pattern_length: 200
    max_namespace_pattern_length: 200
    max_resource_pattern_length: 200
    valid_regex_pattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+]+$"
    valid_ip_pattern: "^[0-9\\.]+$"
    valid_namespace_pattern: "^[a-z0-9\\-]+$"
    valid_resource_pattern: "^[a-z0-9\\-]+$"
    forbidden_chars:
      - "<"
      - ">"
  query_limits:
    max_exclude_users: 50
    max_exclude_resources: 50
    max_group_by_fields: 5
    max_sort_fields: 3
    max_verb_array_size: 10
    max_resource_array_size: 20
    max_namespace_array_size: 50
    max_user_array_size: 100
    max_response_status_array_size: 10
    max_source_ip_array_size: 20
  business_hours:
    default_start_hour: 9
    default_end_hour: 17
    default_timezone: "UTC"
    max_hour_value: 23
    min_hour_value: 0
  analysis_limits:
    max_threshold_value: 10000
    min_threshold_value: 1
    allowed_analysis_types:
      - "anomaly_detection"
    allowed_time_windows:
      - "short"
    allowed_sort_fields:
      - "timestamp"
    allowed_sort_orders:
      - "asc"
  response_status:
    allowed_status_codes:
      - "200"
      - "400"
      - "401"
    min_status_code: 100
    max_status_code: 599
  auth_decisions:
    allowed_decisions:
      - "allow"
      - "forbid"

context:
  cleanup_interval: "10m"
  session_timeout: "12h"
  max_sessions: 5000
  max_memory_mb: 50
  enable_persistence: true
  persistence_path: "./test_sessions"
  persistence_format: "json"
  persistence_interval: "60s"
  enable_async_persistence: false
`

	configPath := filepath.Join(tempDir, "test_config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load configuration from file
	config, err := loader.LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading config from file, got: %v", err)
	}

	// Verify loaded values
	if config.Server.Port != "9090" {
		t.Errorf("Expected port 9090, got %s", config.Server.Port)
	}
	if config.Server.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", config.Server.Host)
	}
	if config.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Expected read timeout 45s, got %v", config.Server.ReadTimeout)
	}
	if config.Server.MaxRequestSize != 2097152 {
		t.Errorf("Expected max request size 2097152, got %d", config.Server.MaxRequestSize)
	}

	if config.Models.DefaultProvider != "openai" {
		t.Errorf("Expected default provider 'openai', got %s", config.Models.DefaultProvider)
	}

	openaiProvider := config.Models.Providers["openai"]
	if openaiProvider.ModelName != "gpt-4" {
		t.Errorf("Expected model name 'gpt-4', got %s", openaiProvider.ModelName)
	}
}

func TestLoadConfigFromFile_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	configPath := filepath.Join(tempDir, "nonexistent.yaml")
	_, err := loader.LoadConfigFromFile(configPath)
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
	if !strings.Contains(err.Error(), "configuration file does not exist") {
		t.Errorf("Expected error about file not existing, got: %v", err)
	}
}

func TestLoadConfigFromFile_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a file with invalid YAML
	invalidYAML := `
server:
  port: "9090"
  host: "localhost"
models:
  default_provider: "openai"
  providers:
    openai:
      provider: "openai"
      endpoint: "https://api.openai.com/v1/chat/completions"
      api_key: "test-key"
      model_name: "gpt-4"
      max_tokens: 2000
      temperature: 0.2
      timeout: "30s"
      retry_attempts: 2
      retry_delay: "500ms"
prompts:
  system_prompts:
    base: "Test base prompt"
    claude_specific: "Test Claude prompt"
    openai_specific: "Test OpenAI prompt"
  examples:
    - input: "Test query"
      output: '{"test": "response"}'
  formats:
    claude:
      template: "Test Claude template"
    openai:
      template: "Test OpenAI template"
    generic:
      template: "Test generic template"
  validation:
    max_input_length: 500
    max_output_length: 1000
    required_fields: ["log_source"]
invalid_yaml: [unclosed_bracket
`

	configPath := filepath.Join(tempDir, "invalid_config.yaml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err := loader.LoadConfigFromFile(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse config YAML") {
		t.Errorf("Expected error about YAML parsing, got: %v", err)
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a test configuration
	config := GetDefaultConfig()
	config.Server.Port = "9090"
	config.Server.Host = "localhost"

	// Save configuration
	configPath := filepath.Join(tempDir, "saved_config.yaml")
	err := loader.SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}

	// Load the saved configuration and verify
	loadedConfig, err := loader.LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading saved config, got: %v", err)
	}

	if loadedConfig.Server.Port != "9090" {
		t.Errorf("Expected saved port 9090, got %s", loadedConfig.Server.Port)
	}
	if loadedConfig.Server.Host != "localhost" {
		t.Errorf("Expected saved host localhost, got %s", loadedConfig.Server.Host)
	}
}

func TestValidateConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Test with valid config file
	validYAML := `
server:
  port: "8080"
  host: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"
  shutdown_timeout: "10s"
  max_request_size: 1048576

models:
  default_provider: "claude"
  providers:
    claude:
      provider: "anthropic"
      endpoint: "https://api.anthropic.com/v1/messages"
      api_key: "test-key"
      model_name: "claude-3-5-sonnet-20241022"
      max_tokens: 4000
      temperature: 0.1
      timeout: "60s"
      retry_attempts: 3
      retry_delay: "1s"

prompts:
  system_prompts:
    base: "Test base prompt"
    claude_specific: "Test Claude prompt"
    openai_specific: "Test OpenAI prompt"
  examples:
    - input: "Test query"
      output: '{"test": "response"}'
  formats:
    claude:
      template: "Test Claude template"
    openai:
      template: "Test OpenAI template"
    generic:
      template: "Test generic template"
  validation:
    max_input_length: 1000
    max_output_length: 2000
    required_fields: ["log_source"]

rules:
  safety_rules:
    allowed_log_sources:
      - "kube-apiserver"
    allowed_verbs:
      - "get"
      - "list"
    allowed_resources:
      - "pods"
    forbidden_patterns:
      - "rm -rf"
    timeframe_limits:
      max_days_back: 90
      default_limit: 20
      max_limit: 1000
      min_limit: 1
      allowed_timeframes:
        - "today"
    required_fields:
      - "log_source"
  sanitization:
    max_query_length: 10000
    max_pattern_length: 500
    max_user_pattern_length: 200
    max_namespace_pattern_length: 200
    max_resource_pattern_length: 200
    valid_regex_pattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+]+$"
    valid_ip_pattern: "^[0-9\\.]+$"
    valid_namespace_pattern: "^[a-z0-9\\-]+$"
    valid_resource_pattern: "^[a-z0-9\\-]+$"
    forbidden_chars:
      - "<"
  query_limits:
    max_exclude_users: 50
    max_exclude_resources: 50
    max_group_by_fields: 5
    max_sort_fields: 3
    max_verb_array_size: 10
    max_resource_array_size: 20
    max_namespace_array_size: 50
    max_user_array_size: 100
    max_response_status_array_size: 10
    max_source_ip_array_size: 20
  business_hours:
    default_start_hour: 9
    default_end_hour: 17
    default_timezone: "UTC"
    max_hour_value: 23
    min_hour_value: 0
  analysis_limits:
    max_threshold_value: 10000
    min_threshold_value: 1
    allowed_analysis_types:
      - "anomaly_detection"
    allowed_time_windows:
      - "short"
    allowed_sort_fields:
      - "timestamp"
    allowed_sort_orders:
      - "asc"
  response_status:
    allowed_status_codes:
      - "200"
      - "400"
    min_status_code: 100
    max_status_code: 599
  auth_decisions:
    allowed_decisions:
      - "allow"

context:
  cleanup_interval: "5m"
  session_timeout: "24h"
  max_sessions: 10000
  max_memory_mb: 100
  enable_persistence: true
  persistence_path: "./sessions"
  persistence_format: "json"
  persistence_interval: "30s"
  enable_async_persistence: true
`

	configPath := filepath.Join(tempDir, "valid_config.yaml")
	if err := os.WriteFile(configPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write valid config file: %v", err)
	}

	result, err := loader.ValidateConfigFile(configPath)
	if err != nil {
		t.Fatalf("Expected no error validating config file, got: %v", err)
	}
	if !result.Valid {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}

	// Test with non-existent file
	nonexistentPath := filepath.Join(tempDir, "nonexistent.yaml")
	result, err = loader.ValidateConfigFile(nonexistentPath)
	if err != nil {
		t.Fatalf("Expected no error validating non-existent file, got: %v", err)
	}
	if result.Valid {
		t.Error("Expected invalid result for non-existent file")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error message for non-existent file")
	}
}

func TestGetConfigFilePaths(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	paths := loader.GetConfigFilePaths()
	expectedPaths := map[string]string{
		"models":  filepath.Join(tempDir, "models.yaml"),
		"prompts": filepath.Join(tempDir, "prompts.yaml"),
	}

	for name, expectedPath := range expectedPaths {
		if actualPath, exists := paths[name]; !exists {
			t.Errorf("Expected path for '%s' to exist", name)
		} else if actualPath != expectedPath {
			t.Errorf("Expected path for '%s' to be '%s', got '%s'", name, expectedPath, actualPath)
		}
	}
}

func TestCheckConfigFiles(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Initially, no files should exist
	exists := loader.CheckConfigFiles()
	if exists["models"] {
		t.Error("Expected models file to not exist initially")
	}
	if exists["prompts"] {
		t.Error("Expected prompts file to not exist initially")
	}

	// Create models file
	modelsPath := filepath.Join(tempDir, "models.yaml")
	if err := os.WriteFile(modelsPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test models file: %v", err)
	}

	// Check again
	exists = loader.CheckConfigFiles()
	if !exists["models"] {
		t.Error("Expected models file to exist after creation")
	}
	if exists["prompts"] {
		t.Error("Expected prompts file to still not exist")
	}
}

func TestSaveModelsConfig(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	config := GetDefaultConfig()
	config.Models.DefaultProvider = "openai"

	err := loader.SaveModelsConfig(config)
	if err != nil {
		t.Fatalf("Expected no error saving models config, got: %v", err)
	}

	// Verify file was created
	modelsPath := filepath.Join(tempDir, "models.yaml")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		t.Error("Expected models config file to be created")
	}

	// Verify content by loading it back
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading saved models config, got: %v", err)
	}

	if loadedConfig.Models.DefaultProvider != "openai" {
		t.Errorf("Expected saved default provider 'openai', got %s", loadedConfig.Models.DefaultProvider)
	}
}

func TestSavePromptsConfig(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	config := GetDefaultConfig()
	config.Prompts.SystemPrompts["base"] = "Custom base prompt"

	err := loader.SavePromptsConfig(config)
	if err != nil {
		t.Fatalf("Expected no error saving prompts config, got: %v", err)
	}

	// Verify file was created
	promptsPath := filepath.Join(tempDir, "prompts.yaml")
	if _, err := os.Stat(promptsPath); os.IsNotExist(err) {
		t.Error("Expected prompts config file to be created")
	}

	// Verify content by loading it back
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading saved prompts config, got: %v", err)
	}

	if loadedConfig.Prompts.SystemPrompts["base"] != "Custom base prompt" {
		t.Errorf("Expected saved custom base prompt, got %s", loadedConfig.Prompts.SystemPrompts["base"])
	}
}

func TestLoadConfig_WithRulesFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a rules.yaml file
	rulesYAML := `
safety_rules:
  allowed_log_sources:
    - "kube-apiserver"
    - "openshift-apiserver"
  allowed_verbs:
    - "get"
    - "list"
    - "create"
  allowed_resources:
    - "pods"
    - "services"
  forbidden_patterns:
    - "rm -rf"
    - "delete --all"
  timeframe_limits:
    max_days_back: 30
    default_limit: 10
    max_limit: 100
    min_limit: 1
    allowed_timeframes:
      - "today"
      - "yesterday"
  required_fields:
    - "log_source"

sanitization:
  max_query_length: 5000
  max_pattern_length: 250
  max_user_pattern_length: 100
  max_namespace_pattern_length: 100
  max_resource_pattern_length: 100
  valid_regex_pattern: "^[a-zA-Z0-9\\-_\\*\\.\\?\\+]+$"
  valid_ip_pattern: "^[0-9\\.]+$"
  valid_namespace_pattern: "^[a-z0-9\\-]+$"
  valid_resource_pattern: "^[a-z0-9\\-]+$"
  forbidden_chars:
    - "<"
    - ">"

query_limits:
  max_exclude_users: 25
  max_exclude_resources: 25
  max_group_by_fields: 3
  max_sort_fields: 2
  max_verb_array_size: 5
  max_resource_array_size: 10
  max_namespace_array_size: 25
  max_user_array_size: 50
  max_response_status_array_size: 5
  max_source_ip_array_size: 10

business_hours:
  default_start_hour: 8
  default_end_hour: 18
  default_timezone: "EST"
  max_hour_value: 23
  min_hour_value: 0

analysis_limits:
  max_threshold_value: 5000
  min_threshold_value: 5
  allowed_analysis_types:
    - "anomaly_detection"
    - "correlation"
  allowed_time_windows:
    - "short"
    - "medium"
  allowed_sort_fields:
    - "timestamp"
    - "user"
  allowed_sort_orders:
    - "asc"
    - "desc"

response_status:
  allowed_status_codes:
    - "200"
    - "400"
    - "401"
    - "403"
    - "404"
  min_status_code: 100
  max_status_code: 599

auth_decisions:
  allowed_decisions:
    - "allow"
    - "forbid"
`

	rulesPath := filepath.Join(tempDir, "rules.yaml")
	if err := os.WriteFile(rulesPath, []byte(rulesYAML), 0644); err != nil {
		t.Fatalf("Failed to write test rules file: %v", err)
	}

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with rules file, got: %v", err)
	}

	// Verify loaded values
	if len(config.Rules.SafetyRules.AllowedLogSources) != 2 {
		t.Errorf("Expected 2 allowed log sources, got %d", len(config.Rules.SafetyRules.AllowedLogSources))
	}

	if config.Rules.SafetyRules.TimeframeLimits.MaxDaysBack != 30 {
		t.Errorf("Expected max days back 30, got %d", config.Rules.SafetyRules.TimeframeLimits.MaxDaysBack)
	}

	if config.Rules.Sanitization.MaxQueryLength != 5000 {
		t.Errorf("Expected max query length 5000, got %d", config.Rules.Sanitization.MaxQueryLength)
	}

	if config.Rules.BusinessHours.DefaultStartHour != 8 {
		t.Errorf("Expected default start hour 8, got %d", config.Rules.BusinessHours.DefaultStartHour)
	}

	if config.Rules.BusinessHours.DefaultTimezone != "EST" {
		t.Errorf("Expected timezone EST, got %s", config.Rules.BusinessHours.DefaultTimezone)
	}
}

func TestLoadConfig_NoRulesFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Load configuration (should use defaults since no rules file exists)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config without rules file, got: %v", err)
	}

	// Verify default values are used
	defaultRules := GetDefaultRulesConfig()
	if len(config.Rules.SafetyRules.AllowedLogSources) != len(defaultRules.SafetyRules.AllowedLogSources) {
		t.Errorf("Expected default allowed log sources count %d, got %d", 
			len(defaultRules.SafetyRules.AllowedLogSources), 
			len(config.Rules.SafetyRules.AllowedLogSources))
	}

	if config.Rules.SafetyRules.TimeframeLimits.MaxDaysBack != defaultRules.SafetyRules.TimeframeLimits.MaxDaysBack {
		t.Errorf("Expected default max days back %d, got %d", 
			defaultRules.SafetyRules.TimeframeLimits.MaxDaysBack, 
			config.Rules.SafetyRules.TimeframeLimits.MaxDaysBack)
	}
}

func TestSaveRulesConfig(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	config := GetDefaultConfig()
	config.Rules.SafetyRules.TimeframeLimits.MaxDaysBack = 60
	config.Rules.BusinessHours.DefaultTimezone = "PST"

	err := loader.SaveRulesConfig(config)
	if err != nil {
		t.Fatalf("Expected no error saving rules config, got: %v", err)
	}

	// Verify file was created
	rulesPath := filepath.Join(tempDir, "rules.yaml")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Error("Expected rules config file to be created")
	}

	// Verify content by loading it back
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading saved rules config, got: %v", err)
	}

	if loadedConfig.Rules.SafetyRules.TimeframeLimits.MaxDaysBack != 60 {
		t.Errorf("Expected saved max days back 60, got %d", loadedConfig.Rules.SafetyRules.TimeframeLimits.MaxDaysBack)
	}

	if loadedConfig.Rules.BusinessHours.DefaultTimezone != "PST" {
		t.Errorf("Expected saved timezone PST, got %s", loadedConfig.Rules.BusinessHours.DefaultTimezone)
	}
}

func TestGetConfigFilePaths_IncludesRules(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	paths := loader.GetConfigFilePaths()
	
	expectedPaths := map[string]string{
		"models":  filepath.Join(tempDir, "models.yaml"),
		"prompts": filepath.Join(tempDir, "prompts.yaml"),
		"rules":   filepath.Join(tempDir, "rules.yaml"),
	}

	for name, expectedPath := range expectedPaths {
		if actualPath, exists := paths[name]; !exists {
			t.Errorf("Expected path for '%s' to exist", name)
		} else if actualPath != expectedPath {
			t.Errorf("Expected path for '%s' to be '%s', got '%s'", name, expectedPath, actualPath)
		}
	}
}

func TestCheckConfigFiles_IncludesRules(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Initially, no files should exist
	exists := loader.CheckConfigFiles()
	if exists["rules"] {
		t.Error("Expected rules file to not exist initially")
	}

	// Create rules file
	rulesPath := filepath.Join(tempDir, "rules.yaml")
	if err := os.WriteFile(rulesPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test rules file: %v", err)
	}

	// Check again
	exists = loader.CheckConfigFiles()
	if !exists["rules"] {
		t.Error("Expected rules file to exist after creation")
	}
}

func TestLoadConfig_WithContextFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Create a context.yaml file
	contextYAML := `
cleanup_interval: "10m"
session_timeout: "12h"
max_sessions: 5000
max_memory_mb: 50
enable_persistence: false
persistence_path: "./custom_sessions"
persistence_format: "gob"
persistence_interval: "60s"
enable_async_persistence: false
`

	contextPath := filepath.Join(tempDir, "context.yaml")
	if err := os.WriteFile(contextPath, []byte(contextYAML), 0644); err != nil {
		t.Fatalf("Failed to write test context file: %v", err)
	}

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with context file, got: %v", err)
	}

	// Verify loaded values
	if config.Context.CleanupInterval != 10*time.Minute {
		t.Errorf("Expected cleanup interval 10m, got %v", config.Context.CleanupInterval)
	}

	if config.Context.SessionTimeout != 12*time.Hour {
		t.Errorf("Expected session timeout 12h, got %v", config.Context.SessionTimeout)
	}

	if config.Context.MaxSessions != 5000 {
		t.Errorf("Expected max sessions 5000, got %d", config.Context.MaxSessions)
	}

	if config.Context.MaxMemoryMB != 50 {
		t.Errorf("Expected max memory 50MB, got %d", config.Context.MaxMemoryMB)
	}

	if config.Context.EnablePersistence != false {
		t.Errorf("Expected enable persistence false, got %v", config.Context.EnablePersistence)
	}

	if config.Context.PersistencePath != "./custom_sessions" {
		t.Errorf("Expected persistence path './custom_sessions', got %s", config.Context.PersistencePath)
	}

	if config.Context.PersistenceFormat != "gob" {
		t.Errorf("Expected persistence format 'gob', got %s", config.Context.PersistenceFormat)
	}

	if config.Context.PersistenceInterval != 60*time.Second {
		t.Errorf("Expected persistence interval 60s, got %v", config.Context.PersistenceInterval)
	}

	if config.Context.EnableAsyncPersistence != false {
		t.Errorf("Expected enable async persistence false, got %v", config.Context.EnableAsyncPersistence)
	}
}

func TestLoadConfig_NoContextFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Load configuration (should use defaults since no context file exists)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config without context file, got: %v", err)
	}

	// Verify default values are used
	defaultContext := GetDefaultContextConfig()
	if config.Context.CleanupInterval != defaultContext.CleanupInterval {
		t.Errorf("Expected default cleanup interval %v, got %v", 
			defaultContext.CleanupInterval, config.Context.CleanupInterval)
	}

	if config.Context.SessionTimeout != defaultContext.SessionTimeout {
		t.Errorf("Expected default session timeout %v, got %v", 
			defaultContext.SessionTimeout, config.Context.SessionTimeout)
	}

	if config.Context.MaxSessions != defaultContext.MaxSessions {
		t.Errorf("Expected default max sessions %d, got %d", 
			defaultContext.MaxSessions, config.Context.MaxSessions)
	}

	if config.Context.PersistenceFormat != defaultContext.PersistenceFormat {
		t.Errorf("Expected default persistence format %s, got %s", 
			defaultContext.PersistenceFormat, config.Context.PersistenceFormat)
	}
}

func TestSaveContextConfig(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	config := GetDefaultConfig()
	config.Context.CleanupInterval = 15 * time.Minute
	config.Context.MaxSessions = 8000
	config.Context.PersistenceFormat = "gob"

	err := loader.SaveContextConfig(config)
	if err != nil {
		t.Fatalf("Expected no error saving context config, got: %v", err)
	}

	// Verify file was created
	contextPath := filepath.Join(tempDir, "context.yaml")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Error("Expected context config file to be created")
	}

	// Verify content by loading it back
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading saved context config, got: %v", err)
	}

	if loadedConfig.Context.CleanupInterval != 15*time.Minute {
		t.Errorf("Expected saved cleanup interval 15m, got %v", loadedConfig.Context.CleanupInterval)
	}

	if loadedConfig.Context.MaxSessions != 8000 {
		t.Errorf("Expected saved max sessions 8000, got %d", loadedConfig.Context.MaxSessions)
	}

	if loadedConfig.Context.PersistenceFormat != "gob" {
		t.Errorf("Expected saved persistence format 'gob', got %s", loadedConfig.Context.PersistenceFormat)
	}
}

func TestLoadConfig_ContextEnvironmentOverrides(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Set environment variables
	os.Setenv("CONTEXT_CLEANUP_INTERVAL", "20m")
	os.Setenv("CONTEXT_SESSION_TIMEOUT", "48h")
	os.Setenv("CONTEXT_MAX_SESSIONS", "15000")
	os.Setenv("CONTEXT_MAX_MEMORY_MB", "200")
	os.Setenv("CONTEXT_ENABLE_PERSISTENCE", "false")
	os.Setenv("CONTEXT_PERSISTENCE_PATH", "/tmp/custom_sessions")
	os.Setenv("CONTEXT_PERSISTENCE_FORMAT", "gob")
	os.Setenv("CONTEXT_PERSISTENCE_INTERVAL", "120s")
	os.Setenv("CONTEXT_ENABLE_ASYNC_PERSISTENCE", "false")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("CONTEXT_CLEANUP_INTERVAL")
		os.Unsetenv("CONTEXT_SESSION_TIMEOUT")
		os.Unsetenv("CONTEXT_MAX_SESSIONS")
		os.Unsetenv("CONTEXT_MAX_MEMORY_MB")
		os.Unsetenv("CONTEXT_ENABLE_PERSISTENCE")
		os.Unsetenv("CONTEXT_PERSISTENCE_PATH")
		os.Unsetenv("CONTEXT_PERSISTENCE_FORMAT")
		os.Unsetenv("CONTEXT_PERSISTENCE_INTERVAL")
		os.Unsetenv("CONTEXT_ENABLE_ASYNC_PERSISTENCE")
	}()

	// Load configuration
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading config with context environment overrides, got: %v", err)
	}

	// Verify environment variable overrides
	if config.Context.CleanupInterval != 20*time.Minute {
		t.Errorf("Expected cleanup interval 20m from environment, got %v", config.Context.CleanupInterval)
	}
	if config.Context.SessionTimeout != 48*time.Hour {
		t.Errorf("Expected session timeout 48h from environment, got %v", config.Context.SessionTimeout)
	}
	if config.Context.MaxSessions != 15000 {
		t.Errorf("Expected max sessions 15000 from environment, got %d", config.Context.MaxSessions)
	}
	if config.Context.MaxMemoryMB != 200 {
		t.Errorf("Expected max memory 200MB from environment, got %d", config.Context.MaxMemoryMB)
	}
	if config.Context.EnablePersistence != false {
		t.Errorf("Expected enable persistence false from environment, got %v", config.Context.EnablePersistence)
	}
	if config.Context.PersistencePath != "/tmp/custom_sessions" {
		t.Errorf("Expected persistence path from environment, got %s", config.Context.PersistencePath)
	}
	if config.Context.PersistenceFormat != "gob" {
		t.Errorf("Expected persistence format 'gob' from environment, got %s", config.Context.PersistenceFormat)
	}
	if config.Context.PersistenceInterval != 120*time.Second {
		t.Errorf("Expected persistence interval 120s from environment, got %v", config.Context.PersistenceInterval)
	}
	if config.Context.EnableAsyncPersistence != false {
		t.Errorf("Expected enable async persistence false from environment, got %v", config.Context.EnableAsyncPersistence)
	}
}

func TestGetConfigFilePaths_IncludesContext(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	paths := loader.GetConfigFilePaths()
	
	expectedPaths := map[string]string{
		"models":  filepath.Join(tempDir, "models.yaml"),
		"prompts": filepath.Join(tempDir, "prompts.yaml"),
		"rules":   filepath.Join(tempDir, "rules.yaml"),
		"context": filepath.Join(tempDir, "context.yaml"),
	}

	for name, expectedPath := range expectedPaths {
		if actualPath, exists := paths[name]; !exists {
			t.Errorf("Expected path for '%s' to exist", name)
		} else if actualPath != expectedPath {
			t.Errorf("Expected path for '%s' to be '%s', got '%s'", name, expectedPath, actualPath)
		}
	}
}

func TestCheckConfigFiles_IncludesContext(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	// Initially, no files should exist
	exists := loader.CheckConfigFiles()
	if exists["context"] {
		t.Error("Expected context file to not exist initially")
	}

	// Create context file
	contextPath := filepath.Join(tempDir, "context.yaml")
	if err := os.WriteFile(contextPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test context file: %v", err)
	}

	// Check again
	exists = loader.CheckConfigFiles()
	if !exists["context"] {
		t.Error("Expected context file to exist after creation")
	}
}
