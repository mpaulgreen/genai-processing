package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"genai-processing/pkg/types"
)

// AppConfig represents the main application configuration
type AppConfig struct {
	Server  ServerConfig  `yaml:"server" validate:"required"`
	Models  ModelsConfig  `yaml:"models" validate:"required"`
	Prompts PromptsConfig `yaml:"prompts" validate:"required"`
}

// ServerConfig defines server-related configuration
type ServerConfig struct {
	Port            string        `yaml:"port" default:"8080"`
	Host            string        `yaml:"host" default:"0.0.0.0"`
	ReadTimeout     time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"10s"`
	MaxRequestSize  int64         `yaml:"max_request_size" default:"1048576"` // 1MB
}

// ModelsConfig defines model-related configuration
type ModelsConfig struct {
	DefaultProvider string                 `yaml:"default_provider" default:"claude"`
	Providers       map[string]ModelConfig `yaml:"providers" validate:"required"`
}

// ModelConfig defines configuration for a specific model provider
type ModelConfig struct {
	Provider        string            `yaml:"provider" validate:"required"`
	Endpoint        string            `yaml:"endpoint" validate:"required"`
	APIKey          string            `yaml:"api_key" validate:"required"`
	ModelName       string            `yaml:"model_name" validate:"required"`
	MaxTokens       int               `yaml:"max_tokens" default:"4000"`
	Temperature     float64           `yaml:"temperature" default:"0.1"`
	Timeout         time.Duration     `yaml:"timeout" default:"60s"`
	RetryAttempts   int               `yaml:"retry_attempts" default:"3"`
	RetryDelay      time.Duration     `yaml:"retry_delay" default:"1s"`
	Parameters      map[string]string `yaml:"parameters,omitempty"`
	InputAdapter    string            `yaml:"input_adapter" default:"generic"`
	OutputParser    string            `yaml:"output_parser" default:"generic"`
	PromptFormatter string            `yaml:"prompt_formatter" default:"generic"`
}

// PromptsConfig defines prompt-related configuration
type PromptsConfig struct {
	SystemPrompts map[string]string `yaml:"system_prompts" validate:"required"`
	Examples      []types.Example   `yaml:"examples" validate:"required"`
	Formats       PromptFormats     `yaml:"formats" validate:"required"`
	Validation    PromptValidation  `yaml:"validation" validate:"required"`
}

// PromptExample removed in favor of types.Example

// PromptFormats defines model-specific prompt formatting
type PromptFormats struct {
	Claude  PromptFormat `yaml:"claude" validate:"required"`
	OpenAI  PromptFormat `yaml:"openai" validate:"required"`
	Generic PromptFormat `yaml:"generic" validate:"required"`
}

// PromptFormat defines the structure for a specific model's prompt format
type PromptFormat struct {
	Template      string            `yaml:"template" validate:"required"`
	SystemMessage string            `yaml:"system_message,omitempty"`
	UserMessage   string            `yaml:"user_message,omitempty"`
	Parameters    map[string]string `yaml:"parameters,omitempty"`
}

// PromptValidation defines validation rules for prompts
type PromptValidation struct {
	MaxInputLength  int      `yaml:"max_input_length" default:"1000"`
	MaxOutputLength int      `yaml:"max_output_length" default:"2000"`
	RequiredFields  []string `yaml:"required_fields" default:"[\"log_source\"]"`
	ForbiddenWords  []string `yaml:"forbidden_words,omitempty"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Validate validates the AppConfig and returns a ValidationResult
func (c *AppConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate server configuration
	if serverResult := c.Server.Validate(); !serverResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, serverResult.Errors...)
	}

	// Validate models configuration
	if modelsResult := c.Models.Validate(); !modelsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, modelsResult.Errors...)
	}

	// Validate prompts configuration
	if promptsResult := c.Prompts.Validate(); !promptsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, promptsResult.Errors...)
	}

	return result
}

// Validate validates the ServerConfig
func (c *ServerConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate port
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid port number")
	}

	// Validate host
	if c.Host != "0.0.0.0" && c.Host != "localhost" && c.Host != "127.0.0.1" {
		if ip := net.ParseIP(c.Host); ip == nil {
			result.Valid = false
			result.Errors = append(result.Errors, "invalid host address")
		}
	}

	// Validate timeouts
	if c.ReadTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "read_timeout must be positive")
	}

	if c.WriteTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "write_timeout must be positive")
	}

	if c.IdleTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "idle_timeout must be positive")
	}

	if c.ShutdownTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "shutdown_timeout must be positive")
	}

	// Validate request size
	if c.MaxRequestSize <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_request_size must be positive")
	}

	return result
}

// Validate validates the ModelsConfig
func (c *ModelsConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate default provider exists
	if c.DefaultProvider == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "default_provider is required")
	} else if _, exists := c.Providers[c.DefaultProvider]; !exists {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("default_provider '%s' not found in providers", c.DefaultProvider))
	}

	// Validate providers
	if len(c.Providers) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one provider must be configured")
	}

	for name, provider := range c.Providers {
		if providerResult := provider.Validate(); !providerResult.Valid {
			result.Valid = false
			for _, err := range providerResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("provider '%s': %s", name, err))
			}
		}
	}

	return result
}

// Validate validates the ModelConfig
func (c *ModelConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate required fields
	if c.Provider == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "provider is required")
	}

	if c.Endpoint == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "endpoint is required")
	}

	// API key is only required for external providers, not for local models
	if c.Provider != "ollama" && c.Provider != "local" && c.Provider != "generic" {
		if c.APIKey == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "api_key is required")
		} else if !isValidAPIKey(c.APIKey) {
			result.Valid = false
			result.Errors = append(result.Errors, "api_key must be a valid key or environment variable placeholder")
		}
	}

	if c.ModelName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "model_name is required")
	}

	// Validate numeric fields
	if c.MaxTokens <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_tokens must be positive")
	}

	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		result.Valid = false
		result.Errors = append(result.Errors, "temperature must be between 0.0 and 2.0")
	}

	if c.Timeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "timeout must be positive")
	}

	if c.RetryAttempts < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "retry_attempts must be non-negative")
	}

	if c.RetryDelay < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "retry_delay must be non-negative")
	}

	// Validate endpoint format
	if !strings.HasPrefix(c.Endpoint, "http://") && !strings.HasPrefix(c.Endpoint, "https://") {
		result.Valid = false
		result.Errors = append(result.Errors, "endpoint must be a valid HTTP/HTTPS URL")
	}

	return result
}

// Validate validates the PromptsConfig
func (c *PromptsConfig) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate system prompts
	if len(c.SystemPrompts) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one system prompt must be configured")
	}

	// Check for required system prompts
	requiredPrompts := []string{"base", "claude_specific", "openai_specific"}
	for _, required := range requiredPrompts {
		if _, exists := c.SystemPrompts[required]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("system prompt '%s' is required", required))
		}
	}

	// Validate examples
	if len(c.Examples) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one example must be configured")
	}

	// Basic non-empty validation for examples
	for i, example := range c.Examples {
		if strings.TrimSpace(example.Input) == "" || strings.TrimSpace(example.Output) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("example %d: input and output cannot be empty", i+1))
		}
	}

	// Validate formats
	if formatsResult := c.Formats.Validate(); !formatsResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, formatsResult.Errors...)
	}

	// Validate validation rules
	if validationResult := c.Validation.Validate(); !validationResult.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, validationResult.Errors...)
	}

	return result
}

// PromptExample validation removed; using basic non-empty checks for types.Example in PromptsConfig.Validate

// Validate validates the PromptFormats
func (f *PromptFormats) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate Claude format
	if claudeResult := f.Claude.Validate(); !claudeResult.Valid {
		result.Valid = false
		for _, err := range claudeResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("claude format: %s", err))
		}
	}

	// Validate OpenAI format
	if openaiResult := f.OpenAI.Validate(); !openaiResult.Valid {
		result.Valid = false
		for _, err := range openaiResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("openai format: %s", err))
		}
	}

	// Validate Generic format
	if genericResult := f.Generic.Validate(); !genericResult.Valid {
		result.Valid = false
		for _, err := range genericResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("generic format: %s", err))
		}
	}

	return result
}

// Validate validates the PromptFormat
func (f *PromptFormat) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if strings.TrimSpace(f.Template) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template cannot be empty")
	}

	return result
}

// Validate validates the PromptValidation
func (v *PromptValidation) Validate() ValidationResult {
	result := ValidationResult{Valid: true}

	if v.MaxInputLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_input_length must be positive")
	}

	if v.MaxOutputLength <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "max_output_length must be positive")
	}

	if len(v.RequiredFields) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one required field must be specified")
	}

	return result
}

// isValidAPIKey checks if the API key is valid (either a real key or an environment variable placeholder)
func isValidAPIKey(apiKey string) bool {
	// Allow empty keys for local models
	if apiKey == "" {
		return true
	}

	// Check for environment variable placeholders with fallback values
	// Pattern: ${VAR_NAME:-default_value}
	if strings.HasPrefix(apiKey, "${") && strings.Contains(apiKey, ":-") && strings.HasSuffix(apiKey, "}") {
		return true
	}

	// Check for simple environment variable placeholders
	// Pattern: ${VAR_NAME}
	if strings.HasPrefix(apiKey, "${") && strings.HasSuffix(apiKey, "}") {
		return true
	}

	// Allow actual API keys (non-empty strings that don't look like placeholders)
	// Must not start with ${ and must not end with } unless it's a complete placeholder
	if len(strings.TrimSpace(apiKey)) > 0 && !strings.HasPrefix(apiKey, "${") && !strings.HasSuffix(apiKey, "}") {
		return true
	}

	return false
}

// GetDefaultConfig returns a default AppConfig with sensible defaults
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:            "8080",
			Host:            "0.0.0.0",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			MaxRequestSize:  1048576, // 1MB
		},
		Models: ModelsConfig{
			DefaultProvider: "claude",
			Providers: map[string]ModelConfig{
				"claude": {
					Provider:        "anthropic",
					Endpoint:        "https://api.anthropic.com/v1/messages",
					APIKey:          "${ANTHROPIC_API_KEY:-placeholder-key}",
					ModelName:       "claude-3-5-sonnet-20241022",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "claude_input_adapter",
					OutputParser:    "claude_extractor",
					PromptFormatter: "claude_formatter",
				},
				"openai": {
					Provider:        "openai",
					Endpoint:        "https://api.openai.com/v1/chat/completions",
					APIKey:          "${OPENAI_API_KEY:-placeholder-key}",
					ModelName:       "gpt-4",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "openai_input_adapter",
					OutputParser:    "openai_extractor",
					PromptFormatter: "openai_formatter",
				},
				"local_llama": {
					Provider:        "ollama",
					Endpoint:        "http://localhost:11434",
					APIKey:          "",
					ModelName:       "llama3.1:8b",
					MaxTokens:       4000,
					Temperature:     0.1,
					Timeout:         60 * time.Second,
					RetryAttempts:   3,
					RetryDelay:      1 * time.Second,
					InputAdapter:    "ollama_input_adapter",
					OutputParser:    "generic_extractor",
					PromptFormatter: "generic_formatter",
				},
			},
		},
		Prompts: PromptsConfig{
			SystemPrompts: map[string]string{
				"base": `You are an OpenShift audit query specialist. Convert natural language 
queries into structured JSON parameters for audit log analysis.

Always respond with valid JSON only. Do not include any markdown formatting,
explanations, or additional text outside the JSON structure.`,
				"claude_specific": `<instructions>
You are an OpenShift audit query specialist. Convert natural language queries 
into structured JSON parameters for audit log analysis.

Respond with a JSON object that matches the provided schema exactly.
</instructions>`,
				"openai_specific": `You are an OpenShift audit query specialist. Your task is to convert natural 
language queries into structured JSON parameters.

Respond with valid JSON only - no markdown formatting or explanations.`,
			},
			Examples: []types.Example{
				{
					Input: "Who deleted the customer CRD yesterday?",
					Output: `{
  "log_source": "kube-apiserver",
  "verb": "delete",
  "resource": "customresourcedefinitions",
  "resource_name_pattern": "customer",
  "timeframe": "yesterday",
  "exclude_users": ["system:"],
  "limit": 20
}`,
				},
				{
					Input: "Show me all failed authentication attempts in the last hour",
					Output: `{
  "log_source": "oauth-server",
  "timeframe": "1_hour_ago",
  "auth_decision": "error",
  "limit": 20
}`,
				},
			},
			Formats: PromptFormats{
				Claude: PromptFormat{
					Template: `<instructions>
{system_prompt}
</instructions>

<examples>
{examples}
</examples>

<query>
{query}
</query>

JSON Response:`,
				},
				OpenAI: PromptFormat{
					Template:      "System: {system_prompt}\n\nExamples:\n{examples}\n\nUser: Convert this query to JSON: {query}",
					SystemMessage: "{system_prompt}\n\nExamples:\n{examples}",
					UserMessage:   "Convert this query to JSON: {query}",
				},
				Generic: PromptFormat{
					Template: `{system_prompt}

Examples:
{examples}

Query: {query}

JSON Response:`,
				},
			},
			Validation: PromptValidation{
				MaxInputLength:  1000,
				MaxOutputLength: 2000,
				RequiredFields:  []string{"log_source"},
				ForbiddenWords:  []string{"rm -rf", "delete --all", "system:admin"},
			},
		},
	}
}
