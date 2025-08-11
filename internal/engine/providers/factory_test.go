package providers

import (
	"genai-processing/pkg/types"
	"testing"
)

func TestNewProviderFactory(t *testing.T) {
	factory := NewProviderFactory()

	if factory == nil {
		t.Fatal("NewProviderFactory() returned nil")
	}

	if factory.configs == nil {
		t.Error("NewProviderFactory() configs map is nil")
	}

	if len(factory.configs) != 0 {
		t.Errorf("NewProviderFactory() expected empty configs, got %d", len(factory.configs))
	}
}

func TestProviderFactory_RegisterProvider(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		name         string
		providerType string
		config       *types.ProviderConfig
		wantErr      bool
	}{
		{
			name:         "valid claude provider",
			providerType: "claude",
			config: &types.ProviderConfig{
				APIKey:   "test-key",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			wantErr: false,
		},
		{
			name:         "valid openai provider",
			providerType: "openai",
			config: &types.ProviderConfig{
				APIKey:    "test-key",
				Endpoint:  "https://api.openai.com/v1/chat/completions",
				ModelName: "gpt-4",
				Parameters: map[string]interface{}{
					"max_tokens":  4000,
					"temperature": 0.1,
				},
			},
			wantErr: false,
		},
		{
			name:         "empty provider type",
			providerType: "",
			config: &types.ProviderConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name:         "nil config",
			providerType: "claude",
			config:       nil,
			wantErr:      true,
		},
		{
			name:         "empty API key",
			providerType: "claude",
			config: &types.ProviderConfig{
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name:         "valid ollama provider",
			providerType: "ollama",
			config: &types.ProviderConfig{
				APIKey:    "", // No API key needed for local Ollama
				Endpoint:  "http://localhost:11434/api/generate",
				ModelName: "llama3.1:8b",
				Parameters: map[string]interface{}{
					"max_tokens":  4000,
					"temperature": 0.1,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.RegisterProvider(tt.providerType, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("RegisterProvider() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RegisterProvider() unexpected error: %v", err)
				}

				// Verify provider was registered
				if _, exists := factory.configs[tt.providerType]; !exists {
					t.Errorf("RegisterProvider() provider %s not found in configs", tt.providerType)
				}
			}
		})
	}
}

func TestProviderFactory_CreateProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Register a claude provider
	claudeConfig := &types.ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
	}

	// Register an openai provider
	openaiConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "gpt-4",
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}
	err = factory.RegisterProvider("openai", openaiConfig)
	if err != nil {
		t.Fatalf("Failed to register openai provider: %v", err)
	}

	// Register a generic provider
	genericConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "generic-model",
		Parameters: map[string]interface{}{
			"headers": map[string]string{"Authorization": "Bearer test-key"},
		},
	}
	err = factory.RegisterProvider("generic", genericConfig)
	if err != nil {
		t.Fatalf("Failed to register generic provider: %v", err)
	}

	// Register an ollama provider
	ollamaConfig := &types.ProviderConfig{
		APIKey:    "", // No API key needed for local Ollama
		Endpoint:  "http://localhost:11434/api/generate",
		ModelName: "llama3.1:8b",
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}
	err = factory.RegisterProvider("ollama", ollamaConfig)
	if err != nil {
		t.Fatalf("Failed to register ollama provider: %v", err)
	}

	tests := []struct {
		name      string
		modelType string
		wantErr   bool
	}{
		{
			name:      "supported claude provider",
			modelType: "claude",
			wantErr:   false,
		},
		{
			name:      "supported openai provider",
			modelType: "openai",
			wantErr:   false,
		},
		{
			name:      "supported generic provider",
			modelType: "generic",
			wantErr:   false,
		},
		{
			name:      "supported ollama provider",
			modelType: "ollama",
			wantErr:   false,
		},
		{
			name:      "unsupported provider type",
			modelType: "unsupported",
			wantErr:   true,
		},
		{
			name:      "empty model type",
			modelType: "",
			wantErr:   true,
		},
		{
			name:      "unregistered provider",
			modelType: "unsupported-provider",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.modelType)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateProvider() expected error, got nil")
				}
				if provider != nil {
					t.Error("CreateProvider() expected nil provider when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("CreateProvider() unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("CreateProvider() expected provider, got nil")
				}

				// Verify it's the correct type
				switch tt.modelType {
				case "claude":
					if _, ok := provider.(*ClaudeProvider); !ok {
						t.Error("CreateProvider() returned wrong provider type for claude")
					}
				case "openai":
					if _, ok := provider.(*OpenAIProvider); !ok {
						t.Error("CreateProvider() returned wrong provider type for openai")
					}
					// Verify OpenAI provider has correct configuration
					openaiProvider := provider.(*OpenAIProvider)
					if openaiProvider.ModelName != "gpt-4" {
						t.Errorf("OpenAI provider ModelName = %s, want gpt-4", openaiProvider.ModelName)
					}
					if openaiProvider.Parameters == nil {
						t.Error("OpenAI provider Parameters is nil")
					}
					if maxTokens, ok := openaiProvider.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
						t.Errorf("OpenAI provider max_tokens = %v, want 4000", maxTokens)
					}
				case "generic":
					if _, ok := provider.(*GenericProvider); !ok {
						t.Error("CreateProvider() returned wrong provider type for generic")
					}
					gp := provider.(*GenericProvider)
					if gp.ModelName != "generic-model" {
						t.Errorf("Generic provider ModelName = %s, want generic-model", gp.ModelName)
					}
				case "ollama":
					if _, ok := provider.(*OllamaProvider); !ok {
						t.Error("CreateProvider() returned wrong provider type for ollama")
					}
					op := provider.(*OllamaProvider)
					if op.ModelName != "llama3.1:8b" {
						t.Errorf("Ollama provider ModelName = %s, want llama3.1:8b", op.ModelName)
					}
					if op.Endpoint != "http://localhost:11434/api/generate" {
						t.Errorf("Ollama provider Endpoint = %s, want http://localhost:11434/api/generate", op.Endpoint)
					}
				}
			}
		})
	}
}

func TestProviderFactory_GetSupportedProviders(t *testing.T) {
	factory := NewProviderFactory()

	// Initially, no providers should be supported
	supported := factory.GetSupportedProviders()
	if len(supported) != 0 {
		t.Errorf("GetSupportedProviders() expected empty list, got %v", supported)
	}

	// Register a claude provider
	claudeConfig := &types.ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
	}

	// Now claude should be supported
	supported = factory.GetSupportedProviders()
	if len(supported) != 1 {
		t.Errorf("GetSupportedProviders() expected 1 provider, got %d", len(supported))
	}

	if supported[0] != "claude" {
		t.Errorf("GetSupportedProviders() expected 'claude', got '%s'", supported[0])
	}

	// Register an openai provider
	openaiConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "gpt-4",
	}
	err = factory.RegisterProvider("openai", openaiConfig)
	if err != nil {
		t.Fatalf("Failed to register openai provider: %v", err)
	}

	// Register an ollama provider
	ollamaConfig := &types.ProviderConfig{
		APIKey:    "",
		Endpoint:  "http://localhost:11434/api/generate",
		ModelName: "llama3.1:8b",
	}
	err = factory.RegisterProvider("ollama", ollamaConfig)
	if err != nil {
		t.Fatalf("Failed to register ollama provider: %v", err)
	}

	// Now three should be supported
	supported = factory.GetSupportedProviders()
	if len(supported) != 3 {
		t.Errorf("GetSupportedProviders() expected 3 providers, got %d", len(supported))
	}

	// Check that all providers are in the list
	hasClaude := false
	hasOpenAI := false
	hasOllama := false
	for _, provider := range supported {
		if provider == "claude" {
			hasClaude = true
		}
		if provider == "openai" {
			hasOpenAI = true
		}
		if provider == "ollama" {
			hasOllama = true
		}
	}

	if !hasClaude {
		t.Error("GetSupportedProviders() missing claude provider")
	}
	if !hasOpenAI {
		t.Error("GetSupportedProviders() missing openai provider")
	}
	if !hasOllama {
		t.Error("GetSupportedProviders() missing ollama provider")
	}
}

func TestProviderFactory_GetProviderConfig(t *testing.T) {
	factory := NewProviderFactory()

	// Register a claude provider
	claudeConfig := &types.ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
	}

	// Register an openai provider
	openaiConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "gpt-4",
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}
	err = factory.RegisterProvider("openai", openaiConfig)
	if err != nil {
		t.Fatalf("Failed to register openai provider: %v", err)
	}

	// Register a generic provider
	genericConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "generic-model",
	}
	err = factory.RegisterProvider("generic", genericConfig)
	if err != nil {
		t.Fatalf("Failed to register generic provider: %v", err)
	}

	tests := []struct {
		name         string
		providerType string
		wantErr      bool
	}{
		{
			name:         "existing claude provider",
			providerType: "claude",
			wantErr:      false,
		},
		{
			name:         "existing openai provider",
			providerType: "openai",
			wantErr:      false,
		},
		{
			name:         "existing generic provider",
			providerType: "generic",
			wantErr:      false,
		},
		{
			name:         "non-existent provider",
			providerType: "ollama",
			wantErr:      true,
		},
		{
			name:         "empty provider type",
			providerType: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := factory.GetProviderConfig(tt.providerType)

			if tt.wantErr {
				if err == nil {
					t.Error("GetProviderConfig() expected error, got nil")
				}
				if config != nil {
					t.Error("GetProviderConfig() expected nil config when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("GetProviderConfig() unexpected error: %v", err)
				}
				if config == nil {
					t.Error("GetProviderConfig() expected config, got nil")
				} else {
					if config.APIKey != "test-key" {
						t.Errorf("GetProviderConfig() expected API key 'test-key', got '%s'", config.APIKey)
					}

					// Verify OpenAI-specific configuration
					if tt.providerType == "openai" {
						if config.ModelName != "gpt-4" {
							t.Errorf("GetProviderConfig() expected model name 'gpt-4', got '%s'", config.ModelName)
						}
						if config.Parameters == nil {
							t.Error("GetProviderConfig() expected parameters for openai")
						} else {
							if maxTokens, ok := config.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
								t.Errorf("GetProviderConfig() expected max_tokens 4000, got %v", maxTokens)
							}
						}
					}
					if tt.providerType == "generic" {
						if config.ModelName != "generic-model" {
							t.Errorf("GetProviderConfig() expected model name 'generic-model', got '%s'", config.ModelName)
						}
					}
				}
			}
		})
	}
}

func TestProviderFactory_ValidateProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		register     bool
		wantErr      bool
	}{
		{
			name:         "valid registered claude provider",
			providerType: "claude",
			register:     true,
			wantErr:      false,
		},
		{
			name:         "valid registered openai provider",
			providerType: "openai",
			register:     true,
			wantErr:      false,
		},
		{
			name:         "valid registered generic provider",
			providerType: "generic",
			register:     true,
			wantErr:      false,
		},
		{
			name:         "valid registered ollama provider",
			providerType: "ollama",
			register:     true,
			wantErr:      false,
		},
		{
			name:         "valid unregistered provider",
			providerType: "claude",
			register:     false,
			wantErr:      true,
		},
		{
			name:         "unsupported provider type",
			providerType: "unsupported",
			register:     false,
			wantErr:      true,
		},
		{
			name:         "empty provider type",
			providerType: "",
			register:     false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new factory for each test case to avoid state sharing
			factory := NewProviderFactory()

			// Register provider if needed
			if tt.register {
				config := &types.ProviderConfig{
					APIKey:   "test-key",
					Endpoint: "https://api.anthropic.com/v1/messages",
				}
				if tt.providerType == "openai" {
					config.Endpoint = "https://api.openai.com/v1/chat/completions"
					config.ModelName = "gpt-4"
				}
				if tt.providerType == "generic" {
					config.Endpoint = "https://api.openai.com/v1/chat/completions"
					config.ModelName = "generic-model"
				}
				err := factory.RegisterProvider(tt.providerType, config)
				if err != nil {
					t.Fatalf("Failed to register provider: %v", err)
				}
			}

			err := factory.ValidateProvider(tt.providerType)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateProvider() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProvider() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProviderFactory_CreateProviderWithConfig(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		name         string
		providerType string
		config       *types.ProviderConfig
		wantErr      bool
	}{
		{
			name:         "valid claude provider",
			providerType: "claude",
			config: &types.ProviderConfig{
				APIKey:   "test-key",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			wantErr: false,
		},
		{
			name:         "valid openai provider with full config",
			providerType: "openai",
			config: &types.ProviderConfig{
				APIKey:    "test-key",
				Endpoint:  "https://api.openai.com/v1/chat/completions",
				ModelName: "gpt-4",
				Parameters: map[string]interface{}{
					"max_tokens":  4000,
					"temperature": 0.1,
				},
			},
			wantErr: false,
		},
		{
			name:         "valid generic provider",
			providerType: "generic",
			config: &types.ProviderConfig{
				APIKey:    "test-key",
				Endpoint:  "https://api.openai.com/v1/chat/completions",
				ModelName: "generic-model",
				Parameters: map[string]interface{}{
					"max_tokens":  4000,
					"temperature": 0.1,
					"headers":     map[string]string{"Authorization": "Bearer test-key"},
				},
			},
			wantErr: false,
		},
		{
			name:         "unsupported provider type",
			providerType: "unsupported",
			config: &types.ProviderConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name:         "nil config",
			providerType: "claude",
			config:       nil,
			wantErr:      true,
		},
		{
			name:         "empty API key",
			providerType: "claude",
			config: &types.ProviderConfig{
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name:         "empty provider type",
			providerType: "",
			config: &types.ProviderConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProviderWithConfig(tt.providerType, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateProviderWithConfig() expected error, got nil")
				}
				if provider != nil {
					t.Error("CreateProviderWithConfig() expected nil provider when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("CreateProviderWithConfig() unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("CreateProviderWithConfig() expected provider, got nil")
				}

				// Verify it's the correct type
				switch tt.providerType {
				case "claude":
					if _, ok := provider.(*ClaudeProvider); !ok {
						t.Error("CreateProviderWithConfig() returned wrong provider type for claude")
					}
				case "openai":
					if _, ok := provider.(*OpenAIProvider); !ok {
						t.Error("CreateProviderWithConfig() returned wrong provider type for openai")
					}
					// Verify OpenAI provider has correct configuration
					openaiProvider := provider.(*OpenAIProvider)
					if openaiProvider.ModelName != "gpt-4" {
						t.Errorf("OpenAI provider ModelName = %s, want gpt-4", openaiProvider.ModelName)
					}
					if openaiProvider.Parameters == nil {
						t.Error("OpenAI provider Parameters is nil")
					}
					if maxTokens, ok := openaiProvider.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
						t.Errorf("OpenAI provider max_tokens = %v, want 4000", maxTokens)
					}
					if temp, ok := openaiProvider.Parameters["temperature"].(float64); !ok || temp != 0.1 {
						t.Errorf("OpenAI provider temperature = %v, want 0.1", temp)
					}
				case "generic":
					if _, ok := provider.(*GenericProvider); !ok {
						t.Error("CreateProviderWithConfig() returned wrong provider type for generic")
					}
				}
			}
		})
	}
}

func TestProviderFactory_GetDefaultConfig(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		name         string
		providerType string
		expectNil    bool
	}{
		{
			name:         "claude provider",
			providerType: "claude",
			expectNil:    false,
		},
		{
			name:         "openai provider",
			providerType: "openai",
			expectNil:    false,
		},
		{
			name:         "generic provider",
			providerType: "generic",
			expectNil:    false,
		},
		{
			name:         "ollama provider",
			providerType: "ollama",
			expectNil:    false,
		},
		{
			name:         "unsupported provider",
			providerType: "unsupported",
			expectNil:    true,
		},
		{
			name:         "empty provider type",
			providerType: "",
			expectNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := factory.GetDefaultConfig(tt.providerType)

			if tt.expectNil {
				if config != nil {
					t.Error("GetDefaultConfig() expected nil config")
				}
			} else {
				if config == nil {
					t.Error("GetDefaultConfig() expected config, got nil")
				}

				// Verify claude config
				if tt.providerType == "claude" {
					if config.APIKey != "" {
						t.Error("GetDefaultConfig() expected empty API key for claude")
					}
					if config.Endpoint != "https://api.anthropic.com/v1/messages" {
						t.Errorf("GetDefaultConfig() expected endpoint 'https://api.anthropic.com/v1/messages', got '%s'", config.Endpoint)
					}
					if config.ModelName != "claude-3-5-sonnet-20241022" {
						t.Errorf("GetDefaultConfig() expected model name 'claude-3-5-sonnet-20241022', got '%s'", config.ModelName)
					}

					// Check parameters
					if config.Parameters == nil {
						t.Error("GetDefaultConfig() expected parameters map")
					} else {
						if maxTokens, ok := config.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
							t.Errorf("GetDefaultConfig() expected max_tokens 4000, got %v", maxTokens)
						}
						if temp, ok := config.Parameters["temperature"].(float64); !ok || temp != 0.1 {
							t.Errorf("GetDefaultConfig() expected temperature 0.1, got %v", temp)
						}
					}
				}

				// Verify openai config
				if tt.providerType == "openai" {
					if config.APIKey != "" {
						t.Error("GetDefaultConfig() expected empty API key for openai")
					}
					if config.Endpoint != "https://api.openai.com/v1/chat/completions" {
						t.Errorf("GetDefaultConfig() expected endpoint 'https://api.openai.com/v1/chat/completions', got '%s'", config.Endpoint)
					}
					if config.ModelName != "gpt-4" {
						t.Errorf("GetDefaultConfig() expected model name 'gpt-4', got '%s'", config.ModelName)
					}

					// Check parameters
					if config.Parameters == nil {
						t.Error("GetDefaultConfig() expected parameters map")
					} else {
						if maxTokens, ok := config.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
							t.Errorf("GetDefaultConfig() expected max_tokens 4000, got %v", maxTokens)
						}
						if temp, ok := config.Parameters["temperature"].(float64); !ok || temp != 0.1 {
							t.Errorf("GetDefaultConfig() expected temperature 0.1, got %v", temp)
						}

					}
				}
				// Verify generic config
				if tt.providerType == "generic" {
					if config.APIKey != "" {
						t.Error("GetDefaultConfig() expected empty API key for generic")
					}
					if config.Endpoint != "https://api.openai.com/v1/chat/completions" {
						t.Errorf("GetDefaultConfig() expected endpoint 'https://api.openai.com/v1/chat/completions', got '%s'", config.Endpoint)
					}
					if config.ModelName != "generic-model" {
						t.Errorf("GetDefaultConfig() expected model name 'generic-model', got '%s'", config.ModelName)
					}
					if config.Parameters == nil {
						t.Error("GetDefaultConfig() expected parameters map")
					} else {
						if maxTokens, ok := config.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
							t.Errorf("GetDefaultConfig() expected max_tokens 4000, got %v", maxTokens)
						}
						if temp, ok := config.Parameters["temperature"].(float64); !ok || temp != 0.1 {
							t.Errorf("GetDefaultConfig() expected temperature 0.1, got %v", temp)
						}
					}
				}
				// Verify ollama config
				if tt.providerType == "ollama" {
					if config.APIKey != "" {
						t.Error("GetDefaultConfig() expected empty API key for ollama")
					}
					if config.Endpoint != "http://localhost:11434/api/generate" {
						t.Errorf("GetDefaultConfig() expected endpoint 'http://localhost:11434/api/generate', got '%s'", config.Endpoint)
					}
					if config.ModelName != "llama3.1:8b" {
						t.Errorf("GetDefaultConfig() expected model name 'llama3.1:8b', got '%s'", config.ModelName)
					}
					if config.Parameters == nil {
						t.Error("GetDefaultConfig() expected parameters map")
					} else {
						if maxTokens, ok := config.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
							t.Errorf("GetDefaultConfig() expected max_tokens 4000, got %v", maxTokens)
						}
						if temp, ok := config.Parameters["temperature"].(float64); !ok || temp != 0.1 {
							t.Errorf("GetDefaultConfig() expected temperature 0.1, got %v", temp)
						}
					}
				}
			}
		})
	}
}

func TestProviderFactory_Integration(t *testing.T) {
	factory := NewProviderFactory()

	// Test complete workflow for Claude
	// 1. Get default config
	config := factory.GetDefaultConfig("claude")
	if config == nil {
		t.Fatal("GetDefaultConfig() returned nil for claude")
	}

	// 2. Set API key
	config.APIKey = "test-integration-key"

	// 3. Register provider
	err := factory.RegisterProvider("claude", config)
	if err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	// 4. Validate provider
	err = factory.ValidateProvider("claude")
	if err != nil {
		t.Fatalf("ValidateProvider() failed: %v", err)
	}

	// 5. Get supported providers
	supported := factory.GetSupportedProviders()
	if len(supported) != 1 || supported[0] != "claude" {
		t.Errorf("GetSupportedProviders() expected ['claude'], got %v", supported)
	}

	// 6. Create provider
	provider, err := factory.CreateProvider("claude")
	if err != nil {
		t.Fatalf("CreateProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("CreateProvider() returned nil provider")
	}

	// 7. Verify provider type
	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Error("CreateProvider() returned wrong provider type")
	}

	// 8. Test provider interface methods
	modelInfo := provider.GetModelInfo()
	if modelInfo.Name != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelInfo() expected name 'claude-3-5-sonnet-20241022', got '%s'", modelInfo.Name)
	}

	if provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false for current implementation")
	}
}

func TestProviderFactory_OpenAI_Integration(t *testing.T) {
	factory := NewProviderFactory()

	// Test complete workflow for OpenAI
	// 1. Get default config
	config := factory.GetDefaultConfig("openai")
	if config == nil {
		t.Fatal("GetDefaultConfig() returned nil for openai")
	}

	// 2. Set API key
	config.APIKey = "test-integration-key"

	// 3. Register provider
	err := factory.RegisterProvider("openai", config)
	if err != nil {
		t.Fatalf("RegisterProvider() failed: %v", err)
	}

	// 4. Validate provider
	err = factory.ValidateProvider("openai")
	if err != nil {
		t.Fatalf("ValidateProvider() failed: %v", err)
	}

	// 5. Get supported providers
	supported := factory.GetSupportedProviders()
	if len(supported) != 1 || supported[0] != "openai" {
		t.Errorf("GetSupportedProviders() expected ['openai'], got %v", supported)
	}

	// 6. Create provider
	provider, err := factory.CreateProvider("openai")
	if err != nil {
		t.Fatalf("CreateProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("CreateProvider() returned nil provider")
	}

	// 7. Verify provider type
	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Error("CreateProvider() returned wrong provider type")
	}

	// 8. Test provider interface methods
	modelInfo := provider.GetModelInfo()
	if modelInfo.Name != "gpt-4" {
		t.Errorf("GetModelInfo() expected name 'gpt-4', got '%s'", modelInfo.Name)
	}

	if provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false for current implementation")
	}

	// 9. Verify OpenAI provider has correct configuration
	openaiProvider := provider.(*OpenAIProvider)
	if openaiProvider.ModelName != "gpt-4" {
		t.Errorf("OpenAI provider ModelName = %s, want gpt-4", openaiProvider.ModelName)
	}
	if openaiProvider.Parameters == nil {
		t.Error("OpenAI provider Parameters is nil")
	}
	if maxTokens, ok := openaiProvider.Parameters["max_tokens"].(int); !ok || maxTokens != 4000 {
		t.Errorf("OpenAI provider max_tokens = %v, want 4000", maxTokens)
	}
	if temp, ok := openaiProvider.Parameters["temperature"].(float64); !ok || temp != 0.1 {
		t.Errorf("OpenAI provider temperature = %v, want 0.1", temp)
	}
}

func TestProviderFactory_BackwardCompatibility(t *testing.T) {
	factory := NewProviderFactory()

	// Test that Claude support remains intact
	claudeConfig := &types.ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
	}

	provider, err := factory.CreateProvider("claude")
	if err != nil {
		t.Fatalf("CreateProvider() failed for claude: %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Error("CreateProvider() returned wrong provider type for claude")
	}

	// Test that both providers can coexist
	openaiConfig := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		ModelName: "gpt-4",
	}
	err = factory.RegisterProvider("openai", openaiConfig)
	if err != nil {
		t.Fatalf("Failed to register openai provider: %v", err)
	}

	openaiProvider, err := factory.CreateProvider("openai")
	if err != nil {
		t.Fatalf("CreateProvider() failed for openai: %v", err)
	}

	if _, ok := openaiProvider.(*OpenAIProvider); !ok {
		t.Error("CreateProvider() returned wrong provider type for openai")
	}

	// Verify both are supported
	supported := factory.GetSupportedProviders()
	if len(supported) != 2 {
		t.Errorf("GetSupportedProviders() expected 2 providers, got %d", len(supported))
	}

	hasClaude := false
	hasOpenAI := false
	for _, provider := range supported {
		if provider == "claude" {
			hasClaude = true
		}
		if provider == "openai" {
			hasOpenAI = true
		}
	}

	if !hasClaude {
		t.Error("Backward compatibility: claude provider not supported")
	}
	if !hasOpenAI {
		t.Error("Backward compatibility: openai provider not supported")
	}
}
