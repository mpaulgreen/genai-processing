package providers

import (
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
		config       *ProviderConfig
		wantErr      bool
	}{
		{
			name:         "valid claude provider",
			providerType: "claude",
			config: &ProviderConfig{
				APIKey:   "test-key",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			wantErr: false,
		},
		{
			name:         "empty provider type",
			providerType: "",
			config: &ProviderConfig{
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
			config: &ProviderConfig{
				APIKey: "",
			},
			wantErr: true,
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
	claudeConfig := &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
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
			modelType: "openai",
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
				if _, ok := provider.(*ClaudeProvider); !ok {
					t.Error("CreateProvider() returned wrong provider type")
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
	claudeConfig := &ProviderConfig{
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
}

func TestProviderFactory_GetProviderConfig(t *testing.T) {
	factory := NewProviderFactory()

	// Register a claude provider
	claudeConfig := &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://api.anthropic.com/v1/messages",
	}
	err := factory.RegisterProvider("claude", claudeConfig)
	if err != nil {
		t.Fatalf("Failed to register claude provider: %v", err)
	}

	tests := []struct {
		name         string
		providerType string
		wantErr      bool
	}{
		{
			name:         "existing provider",
			providerType: "claude",
			wantErr:      false,
		},
		{
			name:         "non-existent provider",
			providerType: "openai",
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
				}

				if config.APIKey != "test-key" {
					t.Errorf("GetProviderConfig() expected API key 'test-key', got '%s'", config.APIKey)
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
			name:         "valid registered provider",
			providerType: "claude",
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
				config := &ProviderConfig{
					APIKey:   "test-key",
					Endpoint: "https://api.anthropic.com/v1/messages",
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
		config       *ProviderConfig
		wantErr      bool
	}{
		{
			name:         "valid claude provider",
			providerType: "claude",
			config: &ProviderConfig{
				APIKey:   "test-key",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			wantErr: false,
		},
		{
			name:         "unsupported provider type",
			providerType: "unsupported",
			config: &ProviderConfig{
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
			config: &ProviderConfig{
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name:         "empty provider type",
			providerType: "",
			config: &ProviderConfig{
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
				if _, ok := provider.(*ClaudeProvider); !ok {
					t.Error("CreateProviderWithConfig() returned wrong provider type")
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
			}
		})
	}
}

func TestProviderFactory_Integration(t *testing.T) {
	factory := NewProviderFactory()

	// Test complete workflow
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
