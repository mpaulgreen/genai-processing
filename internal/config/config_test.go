package config

import (
	"genai-processing/pkg/types"
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
				APIKey:          "${CLAUDE_API_KEY:-test-key}",
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
					"base": "Base prompt",
					// Removed redundant system prompts - using base only
				},
				Examples: []types.Example{
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
				Examples:      []types.Example{},
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
				Examples:   []types.Example{},
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
			apiKey:  "${CLAUDE_API_KEY:-placeholder-key}",
			isValid: true,
		},
		{
			name:    "simple environment variable",
			apiKey:  "${CLAUDE_API_KEY}",
			isValid: true,
		},
		{
			name:    "actual API key",
			apiKey:  "sk-ant-api03-abc123",
			isValid: true,
		},
		{
			name:    "invalid placeholder format",
			apiKey:  "${CLAUDE_API_KEY",
			isValid: false,
		},
		{
			name:    "invalid placeholder format 2",
			apiKey:  "CLAUDE_API_KEY}",
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

func TestRulesConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    RulesConfig
		wantValid bool
	}{
		{
			name:      "valid config",
			config:    *GetDefaultRulesConfig(),
			wantValid: true,
		},
		{
			name: "missing allowed log sources",
			config: RulesConfig{
				SafetyRules: SafetyRules{
					AllowedLogSources: []string{},
					AllowedVerbs:      []string{"get", "list"},
					RequiredFields:    []string{"log_source"},
					TimeframeLimits: TimeframeLimits{
						MaxDaysBack:       90,
						DefaultLimit:      20,
						MaxLimit:          1000,
						MinLimit:          1,
						AllowedTimeframes: []string{"today"},
					},
				},
				Sanitization:   GetDefaultRulesConfig().Sanitization,
				QueryLimits:    GetDefaultRulesConfig().QueryLimits,
				BusinessHours:  GetDefaultRulesConfig().BusinessHours,
				AnalysisLimits: GetDefaultRulesConfig().AnalysisLimits,
				ResponseStatus: GetDefaultRulesConfig().ResponseStatus,
				AuthDecisions:  GetDefaultRulesConfig().AuthDecisions,
			},
			wantValid: false,
		},
		{
			name: "invalid timeframe limits",
			config: RulesConfig{
				SafetyRules: SafetyRules{
					AllowedLogSources: []string{"kube-apiserver"},
					AllowedVerbs:      []string{"get", "list"},
					RequiredFields:    []string{"log_source"},
					TimeframeLimits: TimeframeLimits{
						MaxDaysBack:       -1, // Invalid
						DefaultLimit:      20,
						MaxLimit:          1000,
						MinLimit:          1,
						AllowedTimeframes: []string{"today"},
					},
				},
				Sanitization:   GetDefaultRulesConfig().Sanitization,
				QueryLimits:    GetDefaultRulesConfig().QueryLimits,
				BusinessHours:  GetDefaultRulesConfig().BusinessHours,
				AnalysisLimits: GetDefaultRulesConfig().AnalysisLimits,
				ResponseStatus: GetDefaultRulesConfig().ResponseStatus,
				AuthDecisions:  GetDefaultRulesConfig().AuthDecisions,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("RulesConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestSafetyRules_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    SafetyRules
		wantValid bool
	}{
		{
			name: "valid config",
			config: SafetyRules{
				AllowedLogSources: []string{"kube-apiserver"},
				AllowedVerbs:      []string{"get", "list"},
				RequiredFields:    []string{"log_source"},
				TimeframeLimits: TimeframeLimits{
					MaxDaysBack:       90,
					DefaultLimit:      20,
					MaxLimit:          1000,
					MinLimit:          1,
					AllowedTimeframes: []string{"today"},
				},
			},
			wantValid: true,
		},
		{
			name: "empty allowed log sources",
			config: SafetyRules{
				AllowedLogSources: []string{},
				AllowedVerbs:      []string{"get", "list"},
				RequiredFields:    []string{"log_source"},
			},
			wantValid: false,
		},
		{
			name: "empty allowed verbs",
			config: SafetyRules{
				AllowedLogSources: []string{"kube-apiserver"},
				AllowedVerbs:      []string{},
				RequiredFields:    []string{"log_source"},
			},
			wantValid: false,
		},
		{
			name: "empty required fields",
			config: SafetyRules{
				AllowedLogSources: []string{"kube-apiserver"},
				AllowedVerbs:      []string{"get", "list"},
				RequiredFields:    []string{},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("SafetyRules.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestTimeframeLimits_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    TimeframeLimits
		wantValid bool
	}{
		{
			name: "valid config",
			config: TimeframeLimits{
				MaxDaysBack:       90,
				DefaultLimit:      20,
				MaxLimit:          1000,
				MinLimit:          1,
				AllowedTimeframes: []string{"today", "yesterday"},
			},
			wantValid: true,
		},
		{
			name: "negative max days back",
			config: TimeframeLimits{
				MaxDaysBack:       -1,
				DefaultLimit:      20,
				MaxLimit:          1000,
				MinLimit:          1,
				AllowedTimeframes: []string{"today"},
			},
			wantValid: false,
		},
		{
			name: "default limit greater than max limit",
			config: TimeframeLimits{
				MaxDaysBack:       90,
				DefaultLimit:      2000,
				MaxLimit:          1000,
				MinLimit:          1,
				AllowedTimeframes: []string{"today"},
			},
			wantValid: false,
		},
		{
			name: "min limit greater than default limit",
			config: TimeframeLimits{
				MaxDaysBack:       90,
				DefaultLimit:      20,
				MaxLimit:          1000,
				MinLimit:          50,
				AllowedTimeframes: []string{"today"},
			},
			wantValid: false,
		},
		{
			name: "empty allowed timeframes",
			config: TimeframeLimits{
				MaxDaysBack:       90,
				DefaultLimit:      20,
				MaxLimit:          1000,
				MinLimit:          1,
				AllowedTimeframes: []string{},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("TimeframeLimits.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestGetDefaultRulesConfig(t *testing.T) {
	config := GetDefaultRulesConfig()

	// Test that default rules config is valid
	result := config.Validate()
	if !result.Valid {
		t.Errorf("Default rules config should be valid, got errors: %v", result.Errors)
	}

	// Test that required fields are set
	if len(config.SafetyRules.AllowedLogSources) == 0 {
		t.Error("Default allowed log sources should be configured")
	}

	if len(config.SafetyRules.AllowedVerbs) == 0 {
		t.Error("Default allowed verbs should be configured")
	}

	if len(config.SafetyRules.RequiredFields) == 0 {
		t.Error("Default required fields should be configured")
	}

	if config.BusinessHours.DefaultTimezone == "" {
		t.Error("Default timezone should be set")
	}
}

func TestContextConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ContextConfig
		wantValid bool
	}{
		{
			name:      "valid config",
			config:    *GetDefaultContextConfig(),
			wantValid: true,
		},
		{
			name: "negative cleanup interval",
			config: ContextConfig{
				CleanupInterval:        -1 * time.Minute,
				SessionTimeout:         24 * time.Hour,
				MaxSessions:           10000,
				MaxMemoryMB:           100,
				EnablePersistence:     true,
				PersistencePath:       "./sessions",
				PersistenceFormat:     "json",
				PersistenceInterval:   30 * time.Second,
				EnableAsyncPersistence: true,
			},
			wantValid: false,
		},
		{
			name: "invalid persistence format",
			config: ContextConfig{
				CleanupInterval:        5 * time.Minute,
				SessionTimeout:         24 * time.Hour,
				MaxSessions:           10000,
				MaxMemoryMB:           100,
				EnablePersistence:     true,
				PersistencePath:       "./sessions",
				PersistenceFormat:     "invalid",
				PersistenceInterval:   30 * time.Second,
				EnableAsyncPersistence: true,
			},
			wantValid: false,
		},
		{
			name: "empty persistence path",
			config: ContextConfig{
				CleanupInterval:        5 * time.Minute,
				SessionTimeout:         24 * time.Hour,
				MaxSessions:           10000,
				MaxMemoryMB:           100,
				EnablePersistence:     true,
				PersistencePath:       "",
				PersistenceFormat:     "json",
				PersistenceInterval:   30 * time.Second,
				EnableAsyncPersistence: true,
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()
			if result.Valid != tt.wantValid {
				t.Errorf("ContextConfig.Validate() = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					t.Logf("Validation errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestGetDefaultContextConfig(t *testing.T) {
	config := GetDefaultContextConfig()

	// Test that default context config is valid
	result := config.Validate()
	if !result.Valid {
		t.Errorf("Default context config should be valid, got errors: %v", result.Errors)
	}

	// Test that required fields are set
	if config.CleanupInterval <= 0 {
		t.Error("Default cleanup interval should be positive")
	}

	if config.SessionTimeout <= 0 {
		t.Error("Default session timeout should be positive")
	}

	if config.MaxSessions <= 0 {
		t.Error("Default max sessions should be positive")
	}

	if config.PersistencePath == "" {
		t.Error("Default persistence path should be set")
	}

	if config.PersistenceFormat != "json" && config.PersistenceFormat != "gob" {
		t.Error("Default persistence format should be 'json' or 'gob'")
	}
}

func TestToContextManagerConfig(t *testing.T) {
	contextConfig := GetDefaultContextConfig()
	contextManagerConfig := contextConfig.ToContextManagerConfig()

	// Test that all fields are properly mapped
	if contextManagerConfig.CleanupInterval != contextConfig.CleanupInterval {
		t.Errorf("CleanupInterval mismatch: got %v, want %v", contextManagerConfig.CleanupInterval, contextConfig.CleanupInterval)
	}

	if contextManagerConfig.SessionTimeout != contextConfig.SessionTimeout {
		t.Errorf("SessionTimeout mismatch: got %v, want %v", contextManagerConfig.SessionTimeout, contextConfig.SessionTimeout)
	}

	if contextManagerConfig.MaxSessions != contextConfig.MaxSessions {
		t.Errorf("MaxSessions mismatch: got %v, want %v", contextManagerConfig.MaxSessions, contextConfig.MaxSessions)
	}

	if contextManagerConfig.MaxMemoryMB != contextConfig.MaxMemoryMB {
		t.Errorf("MaxMemoryMB mismatch: got %v, want %v", contextManagerConfig.MaxMemoryMB, contextConfig.MaxMemoryMB)
	}

	if contextManagerConfig.EnablePersistence != contextConfig.EnablePersistence {
		t.Errorf("EnablePersistence mismatch: got %v, want %v", contextManagerConfig.EnablePersistence, contextConfig.EnablePersistence)
	}

	if contextManagerConfig.PersistencePath != contextConfig.PersistencePath {
		t.Errorf("PersistencePath mismatch: got %v, want %v", contextManagerConfig.PersistencePath, contextConfig.PersistencePath)
	}

	if contextManagerConfig.PersistenceFormat != contextConfig.PersistenceFormat {
		t.Errorf("PersistenceFormat mismatch: got %v, want %v", contextManagerConfig.PersistenceFormat, contextConfig.PersistenceFormat)
	}

	if contextManagerConfig.PersistenceInterval != contextConfig.PersistenceInterval {
		t.Errorf("PersistenceInterval mismatch: got %v, want %v", contextManagerConfig.PersistenceInterval, contextConfig.PersistenceInterval)
	}

	if contextManagerConfig.EnableAsyncPersistence != contextConfig.EnableAsyncPersistence {
		t.Errorf("EnableAsyncPersistence mismatch: got %v, want %v", contextManagerConfig.EnableAsyncPersistence, contextConfig.EnableAsyncPersistence)
	}
}
