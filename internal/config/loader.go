package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading from files and environment variables
type Loader struct {
	configDir string
}

// NewLoader creates a new configuration loader
func NewLoader(configDir string) *Loader {
	return &Loader{
		configDir: configDir,
	}
}

// LoadConfig loads the complete application configuration
func (l *Loader) LoadConfig() (*AppConfig, error) {
	// Start with default configuration
	config := GetDefaultConfig()

	// Load models configuration
	if err := l.loadModelsConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load models config: %w", err)
	}

	// Load prompts configuration
	if err := l.loadPromptsConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load prompts config: %w", err)
	}

	// Load rules configuration
	if err := l.loadRulesConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load rules config: %w", err)
	}

	// Load context configuration
	if err := l.loadContextConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load context config: %w", err)
	}

	// Apply environment variable overrides
	l.applyEnvironmentOverrides(config)

	// Validate the final configuration
	if result := config.Validate(); !result.Valid {
		return nil, fmt.Errorf("configuration validation failed: %v", result.Errors)
	}

	return config, nil
}

// loadModelsConfig loads models configuration from configs/models.yaml
func (l *Loader) loadModelsConfig(config *AppConfig) error {
	modelsPath := filepath.Join(l.configDir, "models.yaml")

	// Check if file exists
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		// File doesn't exist, use default configuration
		return nil
	}

	// Read and parse the file
	data, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("failed to read models config file: %w", err)
	}

	var modelsConfig ModelsConfig
	if err := yaml.Unmarshal(data, &modelsConfig); err != nil {
		return fmt.Errorf("failed to parse models config YAML: %w", err)
	}

	// Merge with existing configuration
	config.Models.DefaultProvider = modelsConfig.DefaultProvider
	if modelsConfig.Providers != nil {
		config.Models.Providers = modelsConfig.Providers
	}

	return nil
}

// loadPromptsConfig loads prompts configuration from configs/prompts.yaml
func (l *Loader) loadPromptsConfig(config *AppConfig) error {
	promptsPath := filepath.Join(l.configDir, "prompts.yaml")

	// Check if file exists
	if _, err := os.Stat(promptsPath); os.IsNotExist(err) {
		// File doesn't exist, use default configuration
		return nil
	}

	// Read and parse the file
	data, err := os.ReadFile(promptsPath)
	if err != nil {
		return fmt.Errorf("failed to read prompts config file: %w", err)
	}

	var promptsConfig PromptsConfig
	if err := yaml.Unmarshal(data, &promptsConfig); err != nil {
		return fmt.Errorf("failed to parse prompts config YAML: %w", err)
	}

	// Merge with existing configuration
	if promptsConfig.SystemPrompts != nil {
		config.Prompts.SystemPrompts = promptsConfig.SystemPrompts
	}
	if promptsConfig.Examples != nil {
		config.Prompts.Examples = promptsConfig.Examples
	}
	// Check if Formats has been set by checking if any field is non-empty
	if promptsConfig.Formats.Claude.Template != "" ||
		promptsConfig.Formats.OpenAI.Template != "" ||
		promptsConfig.Formats.Generic.Template != "" {
		config.Prompts.Formats = promptsConfig.Formats
	}
	// Check if Validation has been set by checking if any field is non-zero
	if promptsConfig.Validation.MaxInputLength > 0 ||
		promptsConfig.Validation.MaxOutputLength > 0 ||
		len(promptsConfig.Validation.RequiredFields) > 0 {
		config.Prompts.Validation = promptsConfig.Validation
	}

	return nil
}

// loadRulesConfig loads rules configuration from configs/rules.yaml
func (l *Loader) loadRulesConfig(config *AppConfig) error {
	rulesPath := filepath.Join(l.configDir, "rules.yaml")

	// Check if file exists
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		// File doesn't exist, use default configuration
		config.Rules = *GetDefaultRulesConfig()
		return nil
	}

	// Read and parse the file
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return fmt.Errorf("failed to read rules config file: %w", err)
	}

	var rulesConfig RulesConfig
	if err := yaml.Unmarshal(data, &rulesConfig); err != nil {
		return fmt.Errorf("failed to parse rules config YAML: %w", err)
	}

	// Use loaded configuration
	config.Rules = rulesConfig
	return nil
}

// loadContextConfig loads context configuration from configs/context.yaml
func (l *Loader) loadContextConfig(config *AppConfig) error {
	contextPath := filepath.Join(l.configDir, "context.yaml")

	// Check if file exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		// File doesn't exist, use default configuration
		config.Context = *GetDefaultContextConfig()
		return nil
	}

	// Read and parse the file
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read context config file: %w", err)
	}

	var contextConfig ContextConfig
	if err := yaml.Unmarshal(data, &contextConfig); err != nil {
		return fmt.Errorf("failed to parse context config YAML: %w", err)
	}

	// Use loaded configuration
	config.Context = contextConfig
	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to the configuration
func (l *Loader) applyEnvironmentOverrides(config *AppConfig) {
	// Server configuration overrides
	if port := os.Getenv("SERVER_PORT"); port != "" {
		config.Server.Port = port
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if readTimeout := os.Getenv("SERVER_READ_TIMEOUT"); readTimeout != "" {
		if duration, err := parseDuration(readTimeout); err == nil {
			config.Server.ReadTimeout = duration
		}
	}
	if writeTimeout := os.Getenv("SERVER_WRITE_TIMEOUT"); writeTimeout != "" {
		if duration, err := parseDuration(writeTimeout); err == nil {
			config.Server.WriteTimeout = duration
		}
	}
	if idleTimeout := os.Getenv("SERVER_IDLE_TIMEOUT"); idleTimeout != "" {
		if duration, err := parseDuration(idleTimeout); err == nil {
			config.Server.IdleTimeout = duration
		}
	}
	if shutdownTimeout := os.Getenv("SERVER_SHUTDOWN_TIMEOUT"); shutdownTimeout != "" {
		if duration, err := parseDuration(shutdownTimeout); err == nil {
			config.Server.ShutdownTimeout = duration
		}
	}
	if maxRequestSize := os.Getenv("SERVER_MAX_REQUEST_SIZE"); maxRequestSize != "" {
		if size, err := parseInt64(maxRequestSize); err == nil {
			config.Server.MaxRequestSize = size
		}
	}

	// Models configuration overrides
	if defaultProvider := os.Getenv("DEFAULT_PROVIDER"); defaultProvider != "" {
		config.Models.DefaultProvider = defaultProvider
	}

	// Apply provider-specific overrides
	for providerName, provider := range config.Models.Providers {
		prefix := strings.ToUpper(providerName) + "_"

		if apiKey := os.Getenv(prefix + "API_KEY"); apiKey != "" {
			provider.APIKey = apiKey
		}
		if endpoint := os.Getenv(prefix + "ENDPOINT"); endpoint != "" {
			provider.Endpoint = endpoint
		}
		if modelName := os.Getenv(prefix + "MODEL_NAME"); modelName != "" {
			provider.ModelName = modelName
		}
		if maxTokens := os.Getenv(prefix + "MAX_TOKENS"); maxTokens != "" {
			if tokens, err := parseInt(maxTokens); err == nil {
				provider.MaxTokens = tokens
			}
		}
		if temperature := os.Getenv(prefix + "TEMPERATURE"); temperature != "" {
			if temp, err := parseFloat64(temperature); err == nil {
				provider.Temperature = temp
			}
		}
		if timeout := os.Getenv(prefix + "TIMEOUT"); timeout != "" {
			if duration, err := parseDuration(timeout); err == nil {
				provider.Timeout = duration
			}
		}
		if retryAttempts := os.Getenv(prefix + "RETRY_ATTEMPTS"); retryAttempts != "" {
			if attempts, err := parseInt(retryAttempts); err == nil {
				provider.RetryAttempts = attempts
			}
		}
		if retryDelay := os.Getenv(prefix + "RETRY_DELAY"); retryDelay != "" {
			if delay, err := parseDuration(retryDelay); err == nil {
				provider.RetryDelay = delay
			}
		}
		if inputAdapter := os.Getenv(prefix + "INPUT_ADAPTER"); inputAdapter != "" {
			provider.InputAdapter = inputAdapter
		}
		if outputParser := os.Getenv(prefix + "OUTPUT_PARSER"); outputParser != "" {
			provider.OutputParser = outputParser
		}
		if promptFormatter := os.Getenv(prefix + "PROMPT_FORMATTER"); promptFormatter != "" {
			provider.PromptFormatter = promptFormatter
		}

		config.Models.Providers[providerName] = provider
	}

	// Context configuration overrides
	if cleanupIntervalStr := os.Getenv("CONTEXT_CLEANUP_INTERVAL"); cleanupIntervalStr != "" {
		if duration, err := time.ParseDuration(cleanupIntervalStr); err == nil {
			config.Context.CleanupInterval = duration
		}
	}
	if sessionTimeoutStr := os.Getenv("CONTEXT_SESSION_TIMEOUT"); sessionTimeoutStr != "" {
		if duration, err := time.ParseDuration(sessionTimeoutStr); err == nil {
			config.Context.SessionTimeout = duration
		}
	}
	if maxSessionsStr := os.Getenv("CONTEXT_MAX_SESSIONS"); maxSessionsStr != "" {
		if maxSessions, err := strconv.Atoi(maxSessionsStr); err == nil {
			config.Context.MaxSessions = maxSessions
		}
	}
	if maxMemoryMBStr := os.Getenv("CONTEXT_MAX_MEMORY_MB"); maxMemoryMBStr != "" {
		if maxMemoryMB, err := strconv.Atoi(maxMemoryMBStr); err == nil {
			config.Context.MaxMemoryMB = maxMemoryMB
		}
	}
	if enablePersistenceStr := os.Getenv("CONTEXT_ENABLE_PERSISTENCE"); enablePersistenceStr != "" {
		if enablePersistence, err := strconv.ParseBool(enablePersistenceStr); err == nil {
			config.Context.EnablePersistence = enablePersistence
		}
	}
	if persistencePath := os.Getenv("CONTEXT_PERSISTENCE_PATH"); persistencePath != "" {
		config.Context.PersistencePath = persistencePath
	}
	if persistenceFormat := os.Getenv("CONTEXT_PERSISTENCE_FORMAT"); persistenceFormat != "" {
		config.Context.PersistenceFormat = persistenceFormat
	}
	if persistenceIntervalStr := os.Getenv("CONTEXT_PERSISTENCE_INTERVAL"); persistenceIntervalStr != "" {
		if duration, err := time.ParseDuration(persistenceIntervalStr); err == nil {
			config.Context.PersistenceInterval = duration
		}
	}
	if enableAsyncPersistenceStr := os.Getenv("CONTEXT_ENABLE_ASYNC_PERSISTENCE"); enableAsyncPersistenceStr != "" {
		if enableAsyncPersistence, err := strconv.ParseBool(enableAsyncPersistenceStr); err == nil {
			config.Context.EnableAsyncPersistence = enableAsyncPersistence
		}
	}
}

// LoadConfigFromFile loads configuration from a specific file
func (l *Loader) LoadConfigFromFile(filePath string) (*AppConfig, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not exist: %s", filePath)
	}

	// Read and parse the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Apply environment variable overrides
	l.applyEnvironmentOverrides(&config)

	// Validate the configuration
	if result := config.Validate(); !result.Valid {
		return nil, fmt.Errorf("configuration validation failed: %v", result.Errors)
	}

	return &config, nil
}

// SaveConfig saves the configuration to a file
func (l *Loader) SaveConfig(config *AppConfig, filePath string) error {
	// Marshal the configuration to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveModelsConfig saves the models configuration to configs/models.yaml
func (l *Loader) SaveModelsConfig(config *AppConfig) error {
	modelsPath := filepath.Join(l.configDir, "models.yaml")

	// Create a models-only config for saving
	modelsConfig := ModelsConfig{
		DefaultProvider: config.Models.DefaultProvider,
		Providers:       config.Models.Providers,
	}

	// Marshal only the models config
	data, err := yaml.Marshal(modelsConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal models config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(modelsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(modelsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write models config file: %w", err)
	}

	return nil
}

// SavePromptsConfig saves the prompts configuration to configs/prompts.yaml
func (l *Loader) SavePromptsConfig(config *AppConfig) error {
	promptsPath := filepath.Join(l.configDir, "prompts.yaml")

	// Create a prompts-only config for saving
	promptsConfig := PromptsConfig{
		SystemPrompts: config.Prompts.SystemPrompts,
		Examples:      config.Prompts.Examples,
		Formats:       config.Prompts.Formats,
		Validation:    config.Prompts.Validation,
	}

	// Marshal only the prompts config
	data, err := yaml.Marshal(promptsConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal prompts config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(promptsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(promptsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write prompts config file: %w", err)
	}

	return nil
}

// SaveRulesConfig saves the rules configuration to configs/rules.yaml
func (l *Loader) SaveRulesConfig(config *AppConfig) error {
	rulesPath := filepath.Join(l.configDir, "rules.yaml")

	// Create a rules-only config for saving
	rulesConfig := RulesConfig{
		SafetyRules:    config.Rules.SafetyRules,
		Sanitization:   config.Rules.Sanitization,
		QueryLimits:    config.Rules.QueryLimits,
		BusinessHours:  config.Rules.BusinessHours,
		AnalysisLimits: config.Rules.AnalysisLimits,
		ResponseStatus: config.Rules.ResponseStatus,
		AuthDecisions:  config.Rules.AuthDecisions,
	}

	// Marshal only the rules config
	data, err := yaml.Marshal(rulesConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal rules config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(rulesPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(rulesPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write rules config file: %w", err)
	}

	return nil
}

// SaveContextConfig saves the context configuration to configs/context.yaml
func (l *Loader) SaveContextConfig(config *AppConfig) error {
	contextPath := filepath.Join(l.configDir, "context.yaml")

	// Create a context-only config for saving
	contextConfig := ContextConfig{
		CleanupInterval:        config.Context.CleanupInterval,
		SessionTimeout:         config.Context.SessionTimeout,
		MaxSessions:           config.Context.MaxSessions,
		MaxMemoryMB:           config.Context.MaxMemoryMB,
		EnablePersistence:     config.Context.EnablePersistence,
		PersistencePath:       config.Context.PersistencePath,
		PersistenceFormat:     config.Context.PersistenceFormat,
		PersistenceInterval:   config.Context.PersistenceInterval,
		EnableAsyncPersistence: config.Context.EnableAsyncPersistence,
	}

	// Marshal only the context config
	data, err := yaml.Marshal(contextConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal context config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(contextPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context config file: %w", err)
	}

	return nil
}

// ValidateConfigFile validates a configuration file without loading it
func (l *Loader) ValidateConfigFile(filePath string) (*ValidationResult, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("configuration file does not exist: %s", filePath)},
		}, nil
	}

	// Read and parse the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to read config file: %v", err)},
		}, nil
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to parse config YAML: %v", err)},
		}, nil
	}

	// Validate the configuration
	result := config.Validate()
	return &result, nil
}

// GetConfigFilePaths returns the paths of configuration files
func (l *Loader) GetConfigFilePaths() map[string]string {
	return map[string]string{
		"models":  filepath.Join(l.configDir, "models.yaml"),
		"prompts": filepath.Join(l.configDir, "prompts.yaml"),
		"rules":   filepath.Join(l.configDir, "rules.yaml"),
		"context": filepath.Join(l.configDir, "context.yaml"),
	}
}

// CheckConfigFiles checks which configuration files exist
func (l *Loader) CheckConfigFiles() map[string]bool {
	paths := l.GetConfigFilePaths()
	result := make(map[string]bool)

	for name, path := range paths {
		if _, err := os.Stat(path); err == nil {
			result[name] = true
		} else {
			result[name] = false
		}
	}

	return result
}

// Helper functions for parsing environment variables

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
