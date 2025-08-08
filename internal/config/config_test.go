package config

import (
	"testing"
	"time"
)

func TestAppConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *AppConfig
		wantValid bool
	}{
		{
			name:      "valid config",
			config:    GetDefaultConfig(),
			wantValid: true,
		},
		{
			name: "invalid server config",
			config: &AppConfig{
				Server: ServerConfig{
					Port: "invalid",
				},
				Models:  GetDefaultConfig().Models,
				Prompts: GetDefaultConfig().Prompts,
			},
			wantValid: false,
		},
		{
			name: "invalid models config",
			config: &AppConfig{
				Server: GetDefaultConfig().Server,
				Models: ModelsConfig{
					DefaultProvider: "nonexistent",
					Providers:       map[string]ModelConfig{},
				},
				Prompts: GetDefaultConfig().Prompts,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("AppConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantValid bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Port:            "8080",
				Host:            "0.0.0.0",
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				IdleTimeout:     60 * time.Second,
				ShutdownTimeout: 10 * time.Second,
				MaxRequestSize:  1048576,
			},
			wantValid: true,
		},
		{
			name: "invalid port",
			config: ServerConfig{
				Port: "99999",
			},
			wantValid: false,
		},
		{
			name: "invalid host",
			config: ServerConfig{
				Port: "8080",
				Host: "invalid-host",
			},
			wantValid: false,
		},
		{
			name: "negative timeout",
			config: ServerConfig{
				Port:        "8080",
				Host:        "0.0.0.0",
				ReadTimeout: -1 * time.Second,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("ServerConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestModelConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ModelConfig
		wantValid bool
	}{
		{
			name: "valid config",
			config: ModelConfig{
				Provider:        "anthropic",
				Endpoint:        "https://api.anthropic.com/v1/messages",
				APIKey:          "${ANTHROPIC_API_KEY:-test-key}",
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
			wantValid: true,
		},
		{
			name: "missing provider",
			config: ModelConfig{
				Endpoint:  "https://api.anthropic.com/v1/messages",
				APIKey:    "test-key",
				ModelName: "claude-3-5-sonnet-20241022",
			},
			wantValid: false,
		},
		{
			name: "invalid endpoint",
			config: ModelConfig{
				Provider:  "anthropic",
				Endpoint:  "invalid-url",
				APIKey:    "test-key",
				ModelName: "claude-3-5-sonnet-20241022",
			},
			wantValid: false,
		},
		{
			name: "invalid temperature",
			config: ModelConfig{
				Provider:    "anthropic",
				Endpoint:    "https://api.anthropic.com/v1/messages",
				APIKey:      "test-key",
				ModelName:   "claude-3-5-sonnet-20241022",
				Temperature: 3.0,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("ModelConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestPromptsConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    PromptsConfig
		wantValid bool
	}{
		{
			name: "valid config",
			config: PromptsConfig{
				SystemPrompts: map[string]string{
					"base":            "Base prompt",
					"claude_specific": "Claude prompt",
					"openai_specific": "OpenAI prompt",
				},
				Examples: []PromptExample{
					{
						Input:  "test input",
						Output: "test output",
					},
				},
				Formats: PromptFormats{
					Claude:  PromptFormat{Template: "claude template"},
					OpenAI:  PromptFormat{Template: "openai template"},
					Generic: PromptFormat{Template: "generic template"},
				},
				Validation: PromptValidation{
					MaxInputLength:  1000,
					MaxOutputLength: 2000,
					RequiredFields:  []string{"log_source"},
				},
			},
			wantValid: true,
		},
		{
			name: "missing system prompts",
			config: PromptsConfig{
				SystemPrompts: map[string]string{},
				Examples:      []PromptExample{},
				Formats:       PromptFormats{},
				Validation:    PromptValidation{},
			},
			wantValid: false,
		},
		{
			name: "missing required system prompt",
			config: PromptsConfig{
				SystemPrompts: map[string]string{
					"base": "Base prompt",
				},
				Examples:   []PromptExample{},
				Formats:    PromptFormats{},
				Validation: PromptValidation{},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("PromptsConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// Test that default config is valid
	result := config.Validate()
	if !result.Valid {
		t.Errorf("Default config should be valid, got errors: %v", result.Errors)
	}

	// Test that required fields are set
	if config.Server.Port == "" {
		t.Error("Default server port should be set")
	}

	if config.Models.DefaultProvider == "" {
		t.Error("Default provider should be set")
	}

	if len(config.Models.Providers) == 0 {
		t.Error("Default providers should be configured")
	}

	if len(config.Prompts.SystemPrompts) == 0 {
		t.Error("Default system prompts should be configured")
	}
}

func TestIsValidAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		isValid bool
	}{
		{
			name:    "empty key",
			apiKey:  "",
			isValid: true,
		},
		{
			name:    "environment variable with fallback",
			apiKey:  "${ANTHROPIC_API_KEY:-placeholder-key}",
			isValid: true,
		},
		{
			name:    "simple environment variable",
			apiKey:  "${ANTHROPIC_API_KEY}",
			isValid: true,
		},
		{
			name:    "actual API key",
			apiKey:  "sk-ant-api03-abc123",
			isValid: true,
		},
		{
			name:    "invalid placeholder format",
			apiKey:  "${ANTHROPIC_API_KEY",
			isValid: false,
		},
		{
			name:    "invalid placeholder format 2",
			apiKey:  "ANTHROPIC_API_KEY}",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAPIKey(tt.apiKey)
			if result != tt.isValid {
				t.Errorf("isValidAPIKey(%q) = %v, want %v", tt.apiKey, result, tt.isValid)
			}
		})
	}
}
