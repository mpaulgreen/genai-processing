package interfaces

import "genai-processing/pkg/types"

// ProviderFactory defines the interface for creating and managing LLM providers.
// This interface abstracts the factory pattern for creating different types of LLM providers.
type ProviderFactory interface {
	// RegisterProvider registers a provider configuration for later use.
	// This method allows dynamic registration of provider configurations.
	//
	// Parameters:
	//   - providerType: The type of provider (e.g., "claude", "openai")
	//   - config: The provider configuration
	//
	// Returns:
	//   - error: Any error that occurred during registration
	RegisterProvider(providerType string, config *types.ProviderConfig) error

	// CreateProvider creates an LLM provider based on the specified model type.
	// This method instantiates the appropriate provider using the registered configuration.
	//
	// Parameters:
	//   - modelType: The type of model/provider to create
	//
	// Returns:
	//   - LLMProvider: The created provider instance
	//   - error: Any error that occurred during creation
	CreateProvider(modelType string) (LLMProvider, error)

	// GetSupportedProviders returns a list of all supported provider types.
	// This method provides information about available providers for client applications.
	//
	// Returns:
	//   - []string: List of supported provider types
	GetSupportedProviders() []string

	// GetProviderConfig returns the configuration for a specific provider type.
	// This method retrieves the stored configuration for a provider.
	//
	// Parameters:
	//   - providerType: The type of provider
	//
	// Returns:
	//   - *ProviderConfig: The provider configuration
	//   - error: Any error that occurred during retrieval
	GetProviderConfig(providerType string) (*types.ProviderConfig, error)

	// ValidateProvider validates that a provider can be created with the given configuration.
	// This method performs validation without actually creating the provider.
	//
	// Parameters:
	//   - providerType: The type of provider to validate
	//
	// Returns:
	//   - error: Any error that occurred during validation
	ValidateProvider(providerType string) error

	// CreateProviderWithConfig creates a provider with a specific configuration.
	// This method allows creating providers with custom configurations.
	//
	// Parameters:
	//   - providerType: The type of provider to create
	//   - config: The custom configuration to use
	//
	// Returns:
	//   - LLMProvider: The created provider instance
	//   - error: Any error that occurred during creation
	CreateProviderWithConfig(providerType string, config *types.ProviderConfig) (LLMProvider, error)
}
