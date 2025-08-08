package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ModelSelector handles the selection of LLM providers based on configuration,
// request parameters, and health status. It implements fallback logic and
// provider preference management.
type ModelSelector struct {
	// factory creates and manages LLM providers
	factory interfaces.ProviderFactory

	// providers holds configured providers with their health status
	providers map[string]*ProviderInfo

	// preferences defines the order of provider preference
	preferences []string

	// defaultProvider is the fallback provider when no specific model is requested
	defaultProvider string

	// healthCheckInterval is how often to check provider health
	healthCheckInterval time.Duration

	// mu protects concurrent access to the selector
	mu sync.RWMutex

	// logger for debugging and monitoring
	logger *log.Logger

	// healthChecker manages health check operations
	healthChecker *HealthChecker
}

// ProviderInfo holds information about a provider including its health status
type ProviderInfo struct {
	// Provider is the actual LLM provider instance
	Provider interfaces.LLMProvider

	// Config contains the provider configuration
	Config *types.ModelConfig

	// IsHealthy indicates whether the provider is currently healthy
	IsHealthy bool

	// LastHealthCheck is when the provider was last checked
	LastHealthCheck time.Time

	// HealthCheckCount tracks how many health checks have been performed
	HealthCheckCount int

	// LastError contains the last error encountered during health check
	LastError error

	// ResponseTime tracks the average response time for health checks
	ResponseTime time.Duration
}

// HealthChecker manages health check operations for providers
type HealthChecker struct {
	// interval is the time between health checks
	interval time.Duration

	// timeout is the timeout for individual health checks
	timeout time.Duration

	// stopChan signals the health checker to stop
	stopChan chan struct{}

	// wg waits for health checker goroutines to complete
	wg sync.WaitGroup
}

// SelectionRequest contains parameters for model selection
type SelectionRequest struct {
	// PreferredModel is the user's preferred model (optional)
	PreferredModel string

	// RequestType indicates the type of request for model selection
	RequestType string

	// Priority indicates the priority level (high, medium, low)
	Priority string

	// Context contains additional context for selection
	Context map[string]interface{}
}

// SelectionResult contains the result of model selection
type SelectionResult struct {
	// SelectedProvider is the selected LLM provider
	SelectedProvider interfaces.LLMProvider

	// ProviderName is the name of the selected provider
	ProviderName string

	// Reason explains why this provider was selected
	Reason string

	// Confidence indicates confidence in the selection
	Confidence float64

	// FallbackUsed indicates if a fallback provider was used
	FallbackUsed bool

	// HealthStatus contains health information about the selected provider
	HealthStatus *ProviderHealthStatus
}

// ProviderHealthStatus contains health information about a provider
type ProviderHealthStatus struct {
	IsHealthy    bool          `json:"is_healthy"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int           `json:"error_count"`
	SuccessRate  float64       `json:"success_rate"`
	LastError    string        `json:"last_error,omitempty"`
}

// NewModelSelector creates a new ModelSelector instance
func NewModelSelector(factory interfaces.ProviderFactory, config *SelectorConfig) *ModelSelector {
	if config == nil {
		config = &SelectorConfig{
			DefaultProvider:     "claude",
			HealthCheckInterval: 5 * time.Minute,
			HealthCheckTimeout:  10 * time.Second,
			Preferences:         []string{"claude", "openai"},
		}
	}

	selector := &ModelSelector{
		factory:             factory,
		providers:           make(map[string]*ProviderInfo),
		preferences:         config.Preferences,
		defaultProvider:     config.DefaultProvider,
		healthCheckInterval: config.HealthCheckInterval,
		logger:              log.Default(),
		healthChecker: &HealthChecker{
			interval: config.HealthCheckInterval,
			timeout:  config.HealthCheckTimeout,
			stopChan: make(chan struct{}),
		},
	}

	// Initialize providers
	selector.initializeProviders()

	// Start health checker
	selector.startHealthChecker()

	return selector
}

// SelectModel selects the appropriate LLM provider based on the request
func (s *ModelSelector) SelectModel(ctx context.Context, req *SelectionRequest) (*SelectionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Printf("Selecting model for request: %+v", req)

	// Step 1: Try to use preferred model if specified and healthy
	preferredModelSpecified := req.PreferredModel != ""
	if preferredModelSpecified {
		if provider, exists := s.providers[req.PreferredModel]; exists && provider.IsHealthy {
			s.logger.Printf("Using preferred model: %s", req.PreferredModel)
			return s.createSelectionResult(provider, req.PreferredModel, "preferred_model", 1.0, false), nil
		}
		s.logger.Printf("Preferred model %s not available or unhealthy, trying fallback", req.PreferredModel)
	}

	// Step 2: Try providers in preference order
	for _, providerName := range s.preferences {
		if provider, exists := s.providers[providerName]; exists && provider.IsHealthy {
			s.logger.Printf("Using preferred provider: %s", providerName)
			// Mark as fallback if a preferred model was specified but couldn't be used
			fallbackUsed := preferredModelSpecified
			return s.createSelectionResult(provider, providerName, "preference_order", 0.9, fallbackUsed), nil
		}
	}

	// Step 3: Try default provider as fallback
	if provider, exists := s.providers[s.defaultProvider]; exists && provider.IsHealthy {
		s.logger.Printf("Using default provider as fallback: %s", s.defaultProvider)
		return s.createSelectionResult(provider, s.defaultProvider, "default_fallback", 0.7, true), nil
	}

	// Step 4: Try any available healthy provider
	for providerName, provider := range s.providers {
		if provider.IsHealthy {
			s.logger.Printf("Using any available healthy provider: %s", providerName)
			return s.createSelectionResult(provider, providerName, "any_healthy", 0.5, true), nil
		}
	}

	// Step 5: All providers are unhealthy, return error
	s.logger.Printf("All providers are unhealthy")
	return nil, fmt.Errorf("no healthy providers available")
}

// GetProviderHealth returns health status for all providers
func (s *ModelSelector) GetProviderHealth() map[string]*ProviderHealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := make(map[string]*ProviderHealthStatus)
	for name, provider := range s.providers {
		health[name] = &ProviderHealthStatus{
			IsHealthy:    provider.IsHealthy,
			LastCheck:    provider.LastHealthCheck,
			ResponseTime: provider.ResponseTime,
			ErrorCount:   provider.HealthCheckCount,
			SuccessRate:  s.calculateSuccessRate(provider),
			LastError:    s.getLastError(provider),
		}
	}
	return health
}

// UpdatePreferences updates the provider preference order
func (s *ModelSelector) UpdatePreferences(preferences []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate that all preferences are valid providers
	for _, pref := range preferences {
		if _, exists := s.providers[pref]; !exists {
			return fmt.Errorf("invalid provider preference: %s", pref)
		}
	}

	s.preferences = preferences
	s.logger.Printf("Updated provider preferences: %v", preferences)
	return nil
}

// SetDefaultProvider sets the default provider
func (s *ModelSelector) SetDefaultProvider(providerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[providerName]; !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	s.defaultProvider = providerName
	s.logger.Printf("Set default provider to: %s", providerName)
	return nil
}

// ForceHealthCheck performs an immediate health check on all providers
func (s *ModelSelector) ForceHealthCheck(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Printf("Performing forced health check on all providers")

	for name, provider := range s.providers {
		if err := s.checkProviderHealth(ctx, name, provider); err != nil {
			s.logger.Printf("Health check failed for %s: %v", name, err)
		}
	}

	return nil
}

// Stop stops the health checker and cleans up resources
func (s *ModelSelector) Stop() {
	s.logger.Printf("Stopping model selector")
	close(s.healthChecker.stopChan)
	s.healthChecker.wg.Wait()
}

// initializeProviders initializes all configured providers
func (s *ModelSelector) initializeProviders() {
	supportedProviders := s.factory.GetSupportedProviders()

	for _, providerName := range supportedProviders {
		provider, err := s.factory.CreateProvider(providerName)
		if err != nil {
			s.logger.Printf("Failed to create provider %s: %v", providerName, err)
			continue
		}

		config, err := s.factory.GetProviderConfig(providerName)
		if err != nil {
			s.logger.Printf("Failed to get config for provider %s: %v", providerName, err)
			continue
		}

		s.providers[providerName] = &ProviderInfo{
			Provider:         provider,
			Config:           s.convertToModelConfig(config),
			IsHealthy:        false, // Will be set by health check
			LastHealthCheck:  time.Time{},
			HealthCheckCount: 0,
		}

		s.logger.Printf("Initialized provider: %s", providerName)
	}
}

// startHealthChecker starts the background health checker
func (s *ModelSelector) startHealthChecker() {
	s.healthChecker.wg.Add(1)
	go func() {
		defer s.healthChecker.wg.Done()
		ticker := time.NewTicker(s.healthCheckInterval)
		defer ticker.Stop()

		// Perform initial health check
		ctx, cancel := context.WithTimeout(context.Background(), s.healthChecker.timeout)
		s.performHealthCheck(ctx)
		cancel()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), s.healthChecker.timeout)
				s.performHealthCheck(ctx)
				cancel()
			case <-s.healthChecker.stopChan:
				return
			}
		}
	}()
}

// performHealthCheck performs health checks on all providers
func (s *ModelSelector) performHealthCheck(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Printf("Performing periodic health check")

	for name, provider := range s.providers {
		if err := s.checkProviderHealth(ctx, name, provider); err != nil {
			s.logger.Printf("Health check failed for %s: %v", name, err)
		}
	}
}

// checkProviderHealth checks the health of a specific provider
func (s *ModelSelector) checkProviderHealth(ctx context.Context, name string, provider *ProviderInfo) error {
	start := time.Now()

	err := provider.Provider.ValidateConnection()

	provider.HealthCheckCount++
	provider.LastHealthCheck = time.Now()
	provider.ResponseTime = time.Since(start)

	if err != nil {
		provider.IsHealthy = false
		provider.LastError = err
		return fmt.Errorf("health check failed for %s: %w", name, err)
	}

	provider.IsHealthy = true
	provider.LastError = nil
	return nil
}

// createSelectionResult creates a SelectionResult from provider information
func (s *ModelSelector) createSelectionResult(provider *ProviderInfo, name, reason string, confidence float64, fallbackUsed bool) *SelectionResult {
	return &SelectionResult{
		SelectedProvider: provider.Provider,
		ProviderName:     name,
		Reason:           reason,
		Confidence:       confidence,
		FallbackUsed:     fallbackUsed,
		HealthStatus: &ProviderHealthStatus{
			IsHealthy:    provider.IsHealthy,
			LastCheck:    provider.LastHealthCheck,
			ResponseTime: provider.ResponseTime,
			ErrorCount:   provider.HealthCheckCount,
			SuccessRate:  s.calculateSuccessRate(provider),
			LastError:    s.getLastError(provider),
		},
	}
}

// calculateSuccessRate calculates the success rate for a provider
func (s *ModelSelector) calculateSuccessRate(provider *ProviderInfo) float64 {
	if provider.HealthCheckCount == 0 {
		return 0.0
	}

	// Simple calculation - can be enhanced with more sophisticated metrics
	if provider.IsHealthy {
		return 1.0
	}
	return 0.0
}

// getLastError returns the last error as a string
func (s *ModelSelector) getLastError(provider *ProviderInfo) string {
	if provider.LastError != nil {
		return provider.LastError.Error()
	}
	return ""
}

// convertToModelConfig converts ProviderConfig to ModelConfig
func (s *ModelSelector) convertToModelConfig(config *types.ProviderConfig) *types.ModelConfig {
	// Extract MaxTokens from parameters if available
	maxTokens := 4000 // Default value
	if config.Parameters != nil {
		if mt, ok := config.Parameters["max_tokens"].(int); ok {
			maxTokens = mt
		}
	}

	// Extract Temperature from parameters if available
	temperature := 0.1 // Default value
	if config.Parameters != nil {
		if temp, ok := config.Parameters["temperature"].(float64); ok {
			temperature = temp
		}
	}

	return &types.ModelConfig{
		ModelName:   config.ModelName,
		Provider:    "unknown", // ProviderConfig doesn't have Provider field
		MaxTokens:   maxTokens,
		Temperature: temperature,
		APIEndpoint: config.Endpoint,
	}
}

// SelectorConfig holds configuration for the ModelSelector
type SelectorConfig struct {
	// DefaultProvider is the default provider to use
	DefaultProvider string

	// Preferences is the ordered list of provider preferences
	Preferences []string

	// HealthCheckInterval is how often to check provider health
	HealthCheckInterval time.Duration

	// HealthCheckTimeout is the timeout for individual health checks
	HealthCheckTimeout time.Duration
}
