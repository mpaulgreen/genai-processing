package providers

import (
	"fmt"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ProviderFactory is responsible for creating and managing LLM providers
type ProviderFactory struct {
	configs map[string]*types.ProviderConfig
}

// NewProviderFactory creates a new ProviderFactory instance
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		configs: make(map[string]*types.ProviderConfig),
	}
}

// RegisterProvider registers a provider configuration for later use
func (f *ProviderFactory) RegisterProvider(providerType string, config *types.ProviderConfig) error {
	if providerType == "" {
		return fmt.Errorf("provider type cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("provider config cannot be nil")
	}
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for provider %s", providerType)
	}

	f.configs[providerType] = config
	return nil
}

// CreateProvider creates an LLM provider based on the specified model type
func (f *ProviderFactory) CreateProvider(modelType string) (interfaces.LLMProvider, error) {
	if modelType == "" {
		return nil, fmt.Errorf("model type cannot be empty")
	}

	// Get configuration for the provider type
	config, exists := f.configs[modelType]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", modelType)
	}

	// Create provider based on type with full configuration
	switch modelType {
	case "claude":
		return NewClaudeProvider(config.APIKey, config.Endpoint), nil
	case "openai":
		return NewOpenAIProviderWithConfig(config.APIKey, config.Endpoint, config.ModelName, config.Parameters), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", modelType)
	}
}

// GetSupportedProviders returns a list of all supported provider types
func (f *ProviderFactory) GetSupportedProviders() []string {
	supported := []string{"claude", "openai"}

	// Return intersection of supported and registered providers
	result := make([]string, 0)
	for _, provider := range supported {
		if _, exists := f.configs[provider]; exists {
			result = append(result, provider)
		}
	}

	return result
}

// GetProviderConfig returns the configuration for a specific provider type
func (f *ProviderFactory) GetProviderConfig(providerType string) (*types.ProviderConfig, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider type cannot be empty")
	}

	config, exists := f.configs[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type not found: %s", providerType)
	}

	return config, nil
}

// ValidateProvider validates that a provider can be created with the given configuration
func (f *ProviderFactory) ValidateProvider(providerType string) error {
	if providerType == "" {
		return fmt.Errorf("provider type cannot be empty")
	}

	// Check if provider type is supported
	supported := []string{"claude", "openai"}
	isSupported := false
	for _, supportedType := range supported {
		if providerType == supportedType {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Check if provider is registered
	if _, exists := f.configs[providerType]; !exists {
		return fmt.Errorf("provider type not registered: %s", providerType)
	}

	return nil
}

// CreateProviderWithConfig creates a provider with the given configuration
func (f *ProviderFactory) CreateProviderWithConfig(providerType string, config *types.ProviderConfig) (interfaces.LLMProvider, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider type cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("provider config cannot be nil")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for provider %s", providerType)
	}

	// Create provider based on type with full configuration
	switch providerType {
	case "claude":
		return NewClaudeProvider(config.APIKey, config.Endpoint), nil
	case "openai":
		return NewOpenAIProviderWithConfig(config.APIKey, config.Endpoint, config.ModelName, config.Parameters), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetDefaultConfig returns a default configuration for a provider type
func (f *ProviderFactory) GetDefaultConfig(providerType string) *types.ProviderConfig {
	switch providerType {
	case "claude":
		return &types.ProviderConfig{
			APIKey:    "", // Must be set by user
			Endpoint:  "https://api.anthropic.com/v1/messages",
			ModelName: "claude-3-5-sonnet-20241022",
			Parameters: map[string]interface{}{
				"max_tokens":  4000,
				"temperature": 0.1,
			},
		}
	case "openai":
		return &types.ProviderConfig{
			APIKey:    "", // Must be set by user
			Endpoint:  "https://api.openai.com/v1/chat/completions",
			ModelName: "gpt-4",
			Parameters: map[string]interface{}{
				"max_tokens":  4000,
				"temperature": 0.1,
				"top_p":       1.0,
			},
		}
	default:
		return nil
	}
}
