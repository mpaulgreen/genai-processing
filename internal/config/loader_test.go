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
  claude_specific: "Custom Claude prompt"
  openai_specific: "Custom OpenAI prompt"

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

validation:
  max_input_length: 500
  max_output_length: 1000
  required_fields: ["log_source", "verb"]
  forbidden_words: ["bad_word"]
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

	if config.Prompts.Validation.MaxInputLength != 500 {
		t.Errorf("Expected max input length 500, got %d", config.Prompts.Validation.MaxInputLength)
	}

	if len(config.Prompts.Validation.RequiredFields) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(config.Prompts.Validation.RequiredFields))
	}
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
