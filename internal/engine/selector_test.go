package engine

import (
	"context"
	"testing"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// SelectorMockLLMProvider implements the LLMProvider interface for testing
type SelectorMockLLMProvider struct {
	name          string
	isHealthy     bool
	shouldFail    bool
	responseTime  time.Duration
	validateError error
}

func (m *SelectorMockLLMProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	if m.shouldFail {
		return nil, errors.NewProcessingError("provider_error", "mock provider failed", "mock_provider", false)
	}

	time.Sleep(m.responseTime) // Simulate response time

	return &types.RawResponse{
		Content: "mock response",
		ModelInfo: map[string]interface{}{
			"model": m.name,
		},
		Metadata: map[string]interface{}{
			"provider": m.name,
		},
	}, nil
}

func (m *SelectorMockLLMProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:     m.name,
		Provider: "mock",
		Version:  "1.0",
	}
}

func (m *SelectorMockLLMProvider) SupportsStreaming() bool {
	return false
}

func (m *SelectorMockLLMProvider) ValidateConnection() error {
	if m.validateError != nil {
		return m.validateError
	}
	if !m.isHealthy {
		return errors.NewProcessingError("health_check_failed", "mock provider is unhealthy", "mock_provider", true)
	}
	return nil
}

// MockProviderFactory implements the ProviderFactory interface for testing
type MockProviderFactory struct {
	providers map[string]*SelectorMockLLMProvider
	configs   map[string]*types.ProviderConfig
}

func NewMockProviderFactory() *MockProviderFactory {
	return &MockProviderFactory{
		providers: make(map[string]*SelectorMockLLMProvider),
		configs:   make(map[string]*types.ProviderConfig),
	}
}

func (m *MockProviderFactory) RegisterProvider(providerType string, config *types.ProviderConfig) error {
	m.configs[providerType] = config
	return nil
}

func (m *MockProviderFactory) CreateProvider(modelType string) (interfaces.LLMProvider, error) {
	provider, exists := m.providers[modelType]
	if !exists {
		return nil, errors.NewProcessingError("provider_not_found", "provider not found: "+modelType, "mock_factory", false)
	}
	return provider, nil
}

func (m *MockProviderFactory) GetSupportedProviders() []string {
	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

func (m *MockProviderFactory) GetProviderConfig(providerType string) (*types.ProviderConfig, error) {
	config, exists := m.configs[providerType]
	if !exists {
		return nil, errors.NewProcessingError("config_not_found", "config not found: "+providerType, "mock_factory", false)
	}
	return config, nil
}

func (m *MockProviderFactory) ValidateProvider(providerType string) error {
	if _, exists := m.providers[providerType]; !exists {
		return errors.NewProcessingError("provider_not_found", "provider not found: "+providerType, "mock_factory", false)
	}
	return nil
}

func (m *MockProviderFactory) CreateProviderWithConfig(providerType string, config *types.ProviderConfig) (interfaces.LLMProvider, error) {
	return m.CreateProvider(providerType)
}

// Helper function to create a mock provider
func (m *MockProviderFactory) AddMockProvider(name string, isHealthy bool, responseTime time.Duration) {
	m.providers[name] = &SelectorMockLLMProvider{
		name:         name,
		isHealthy:    isHealthy,
		responseTime: responseTime,
	}

	// Add default config
	m.configs[name] = &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://test.endpoint.com",
		ModelName: name + "-model",
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}
}

func TestNewModelSelector(t *testing.T) {
	factory := NewMockProviderFactory()

	// Test with nil config (should use defaults)
	selector := NewModelSelector(factory, nil)
	if selector == nil {
		t.Fatal("NewModelSelector() returned nil")
	}

	if selector.defaultProvider != "claude" {
		t.Errorf("Expected default provider 'claude', got '%s'", selector.defaultProvider)
	}

	if len(selector.preferences) != 2 {
		t.Errorf("Expected 2 preferences, got %d", len(selector.preferences))
	}

	// Test with custom config
	config := &SelectorConfig{
		DefaultProvider:     "openai",
		Preferences:         []string{"openai", "claude"},
		HealthCheckInterval: 1 * time.Minute,
		HealthCheckTimeout:  5 * time.Second,
	}

	selector = NewModelSelector(factory, config)
	if selector.defaultProvider != "openai" {
		t.Errorf("Expected default provider 'openai', got '%s'", selector.defaultProvider)
	}

	if len(selector.preferences) != 2 {
		t.Errorf("Expected 2 preferences, got %d", len(selector.preferences))
	}
}

func TestModelSelector_SelectModel_PreferredModel(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	req := &SelectionRequest{
		PreferredModel: "claude",
	}

	result, err := selector.SelectModel(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}

	if result.ProviderName != "claude" {
		t.Errorf("Expected provider 'claude', got '%s'", result.ProviderName)
	}

	if result.Reason != "preferred_model" {
		t.Errorf("Expected reason 'preferred_model', got '%s'", result.Reason)
	}

	if result.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", result.Confidence)
	}

	if result.FallbackUsed {
		t.Error("Expected fallback not to be used")
	}
}

func TestModelSelector_SelectModel_PreferredModelUnhealthy(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", false, 100*time.Millisecond) // Unhealthy
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	req := &SelectionRequest{
		PreferredModel: "claude",
	}

	result, err := selector.SelectModel(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}

	if result.ProviderName != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", result.ProviderName)
	}

	if result.Reason != "preference_order" {
		t.Errorf("Expected reason 'preference_order', got '%s'", result.Reason)
	}

	// When preferred model is unhealthy, it should use fallback
	if !result.FallbackUsed {
		t.Error("Expected fallback to be used when preferred model is unhealthy")
	}
}

func TestModelSelector_SelectModel_PreferenceOrder(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	req := &SelectionRequest{
		PreferredModel: "", // No preferred model
	}

	result, err := selector.SelectModel(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}

	// Should select first in preference order (claude)
	if result.ProviderName != "claude" {
		t.Errorf("Expected provider 'claude', got '%s'", result.ProviderName)
	}

	if result.Reason != "preference_order" {
		t.Errorf("Expected reason 'preference_order', got '%s'", result.Reason)
	}
}

func TestModelSelector_SelectModel_DefaultFallback(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", false, 100*time.Millisecond) // Unhealthy
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	req := &SelectionRequest{
		PreferredModel: "",
	}

	result, err := selector.SelectModel(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}

	if result.ProviderName != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", result.ProviderName)
	}

	if result.Reason != "preference_order" {
		t.Errorf("Expected reason 'preference_order', got '%s'", result.Reason)
	}
}

func TestModelSelector_SelectModel_AllUnhealthy(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", false, 100*time.Millisecond) // Unhealthy
	factory.AddMockProvider("openai", false, 50*time.Millisecond)  // Unhealthy

	selector := NewModelSelector(factory, nil)

	req := &SelectionRequest{
		PreferredModel: "",
	}

	_, err := selector.SelectModel(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error when all providers are unhealthy")
	}

	expectedError := "no healthy providers available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestModelSelector_GetProviderHealth(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", false, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Force health check
	err := selector.ForceHealthCheck(context.Background())
	if err != nil {
		t.Fatalf("ForceHealthCheck failed: %v", err)
	}

	health := selector.GetProviderHealth()

	if len(health) != 2 {
		t.Errorf("Expected 2 health entries, got %d", len(health))
	}

	// Check Claude health
	if claudeHealth, exists := health["claude"]; exists {
		if !claudeHealth.IsHealthy {
			t.Error("Expected Claude to be healthy")
		}
		if claudeHealth.SuccessRate != 1.0 {
			t.Errorf("Expected Claude success rate 1.0, got %f", claudeHealth.SuccessRate)
		}
	} else {
		t.Error("Claude health not found")
	}

	// Check OpenAI health
	if openaiHealth, exists := health["openai"]; exists {
		if openaiHealth.IsHealthy {
			t.Error("Expected OpenAI to be unhealthy")
		}
		if openaiHealth.SuccessRate != 0.0 {
			t.Errorf("Expected OpenAI success rate 0.0, got %f", openaiHealth.SuccessRate)
		}
	} else {
		t.Error("OpenAI health not found")
	}
}

func TestModelSelector_UpdatePreferences(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Update preferences
	newPreferences := []string{"openai", "claude"}
	err := selector.UpdatePreferences(newPreferences)
	if err != nil {
		t.Fatalf("UpdatePreferences failed: %v", err)
	}

	if len(selector.preferences) != 2 {
		t.Errorf("Expected 2 preferences, got %d", len(selector.preferences))
	}

	if selector.preferences[0] != "openai" {
		t.Errorf("Expected first preference 'openai', got '%s'", selector.preferences[0])
	}

	if selector.preferences[1] != "claude" {
		t.Errorf("Expected second preference 'claude', got '%s'", selector.preferences[1])
	}
}

func TestModelSelector_UpdatePreferences_InvalidProvider(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Try to update with invalid provider
	newPreferences := []string{"invalid_provider"}
	err := selector.UpdatePreferences(newPreferences)
	if err == nil {
		t.Fatal("Expected error when updating with invalid provider")
	}

	expectedError := "invalid provider preference: invalid_provider"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestModelSelector_SetDefaultProvider(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Set default provider
	err := selector.SetDefaultProvider("openai")
	if err != nil {
		t.Fatalf("SetDefaultProvider failed: %v", err)
	}

	if selector.defaultProvider != "openai" {
		t.Errorf("Expected default provider 'openai', got '%s'", selector.defaultProvider)
	}
}

func TestModelSelector_SetDefaultProvider_InvalidProvider(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Try to set invalid default provider
	err := selector.SetDefaultProvider("invalid_provider")
	if err == nil {
		t.Fatal("Expected error when setting invalid default provider")
	}

	expectedError := "provider not found: invalid_provider"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestModelSelector_Stop(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Stop the selector
	selector.Stop()

	// Verify that the health checker has stopped
	// This is a basic test - in a real scenario, you might want to add
	// more sophisticated verification that the goroutine has actually stopped
}

func TestModelSelector_ConvertToModelConfig(t *testing.T) {
	factory := NewMockProviderFactory()
	selector := NewModelSelector(factory, nil)

	config := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://test.endpoint.com",
		ModelName: "test-model",
		Parameters: map[string]interface{}{
			"max_tokens":  8000,
			"temperature": 0.5,
		},
	}

	modelConfig := selector.convertToModelConfig(config)

	if modelConfig.ModelName != "test-model" {
		t.Errorf("Expected model name 'test-model', got '%s'", modelConfig.ModelName)
	}

	if modelConfig.MaxTokens != 8000 {
		t.Errorf("Expected max tokens 8000, got %d", modelConfig.MaxTokens)
	}

	if modelConfig.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", modelConfig.Temperature)
	}

	if modelConfig.APIEndpoint != "https://test.endpoint.com" {
		t.Errorf("Expected endpoint 'https://test.endpoint.com', got '%s'", modelConfig.APIEndpoint)
	}
}

func TestModelSelector_ConvertToModelConfig_DefaultValues(t *testing.T) {
	factory := NewMockProviderFactory()
	selector := NewModelSelector(factory, nil)

	config := &types.ProviderConfig{
		APIKey:    "test-key",
		Endpoint:  "https://test.endpoint.com",
		ModelName: "test-model",
		// No parameters - should use defaults
	}

	modelConfig := selector.convertToModelConfig(config)

	if modelConfig.MaxTokens != 4000 {
		t.Errorf("Expected default max tokens 4000, got %d", modelConfig.MaxTokens)
	}

	if modelConfig.Temperature != 0.1 {
		t.Errorf("Expected default temperature 0.1, got %f", modelConfig.Temperature)
	}
}

func TestModelSelector_Integration(t *testing.T) {
	factory := NewMockProviderFactory()
	factory.AddMockProvider("claude", true, 100*time.Millisecond)
	factory.AddMockProvider("openai", true, 50*time.Millisecond)

	selector := NewModelSelector(factory, nil)

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	// Test multiple selection scenarios
	testCases := []struct {
		name           string
		preferredModel string
		expectedReason string
		expectFallback bool
	}{
		{
			name:           "Preferred model available",
			preferredModel: "claude",
			expectedReason: "preferred_model",
			expectFallback: false,
		},
		{
			name:           "No preferred model",
			preferredModel: "",
			expectedReason: "preference_order",
			expectFallback: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &SelectionRequest{
				PreferredModel: tc.preferredModel,
			}

			result, err := selector.SelectModel(context.Background(), req)
			if err != nil {
				t.Fatalf("SelectModel failed: %v", err)
			}

			if result.Reason != tc.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tc.expectedReason, result.Reason)
			}

			if result.FallbackUsed != tc.expectFallback {
				t.Errorf("Expected fallback %t, got %t", tc.expectFallback, result.FallbackUsed)
			}

			if result.SelectedProvider == nil {
				t.Error("Expected selected provider to not be nil")
			}

			if result.HealthStatus == nil {
				t.Error("Expected health status to not be nil")
			}
		})
	}
}
