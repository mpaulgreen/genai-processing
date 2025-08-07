package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"

	"github.com/google/uuid"
)

// LLMEngine implements the LLMEngine interface and coordinates adapter and provider
// to process natural language queries into structured responses.
type LLMEngine struct {
	// provider is the LLM provider that handles API communication
	provider interfaces.LLMProvider

	// adapter is the input adapter that formats requests for the specific model
	adapter interfaces.InputAdapter

	// mu protects concurrent access to the engine
	mu sync.RWMutex

	// logger for debugging and monitoring
	logger *log.Logger
}

// NewLLMEngine creates a new LLMEngine instance with the specified provider and adapter
func NewLLMEngine(provider interfaces.LLMProvider, adapter interfaces.InputAdapter) *LLMEngine {
	return &LLMEngine{
		provider: provider,
		adapter:  adapter,
		logger:   log.Default(),
	}
}

// ProcessQuery implements the LLMEngine interface.
// It coordinates the complete pipeline: adapt input → call provider → return raw response
func (e *LLMEngine) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.logger.Printf("Processing query: %s", query)

	// Step 1: Create internal request from query and context
	internalReq := &types.InternalRequest{
		RequestID: generateRequestID(),
		ProcessingRequest: types.ProcessingRequest{
			Query:     query,
			SessionID: context.SessionID,
		},
		ProcessingOptions: map[string]interface{}{
			"conversation_context": context,
		},
	}

	// Step 2: Adapt input using the adapter
	e.logger.Printf("Adapting input for model")
	modelReq, err := e.AdaptInput(internalReq)
	if err != nil {
		e.logger.Printf("Input adaptation failed: %v", err)
		return nil, fmt.Errorf("failed to adapt input: %w", err)
	}

	// Step 3: Call provider to generate response
	e.logger.Printf("Calling LLM provider")
	rawResponse, err := e.provider.GenerateResponse(ctx, modelReq)
	if err != nil {
		e.logger.Printf("Provider call failed: %v", err)
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	e.logger.Printf("Successfully processed query, response length: %d", len(rawResponse.Content))
	return rawResponse, nil
}

// GetSupportedModels implements the LLMEngine interface.
// It delegates to the provider to get supported models
func (e *LLMEngine) GetSupportedModels() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get model info from provider
	modelInfo := e.provider.GetModelInfo()

	// Return list with the current model
	models := []string{modelInfo.Name}

	// Add provider name as a supported model type
	models = append(models, modelInfo.Provider)

	return models
}

// AdaptInput implements the LLMEngine interface.
// It delegates to the adapter to format the request for the specific model
func (e *LLMEngine) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if req == nil {
		return nil, fmt.Errorf("internal request cannot be nil")
	}

	e.logger.Printf("Adapting input request ID: %s", req.RequestID)

	// Delegate to adapter for model-specific formatting
	modelReq, err := e.adapter.AdaptRequest(req)
	if err != nil {
		return nil, fmt.Errorf("adapter failed to adapt request: %w", err)
	}

	e.logger.Printf("Successfully adapted input for model: %s", modelReq.Model)
	return modelReq, nil
}

// SetLogger sets a custom logger for the engine
func (e *LLMEngine) SetLogger(logger *log.Logger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = logger
}

// GetProvider returns the current LLM provider
func (e *LLMEngine) GetProvider() interfaces.LLMProvider {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.provider
}

// GetAdapter returns the current input adapter
func (e *LLMEngine) GetAdapter() interfaces.InputAdapter {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.adapter
}

// UpdateProvider updates the LLM provider
func (e *LLMEngine) UpdateProvider(provider interfaces.LLMProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.provider = provider
	e.logger.Printf("Updated LLM provider to: %s", provider.GetModelInfo().Name)
}

// UpdateAdapter updates the input adapter
func (e *LLMEngine) UpdateAdapter(adapter interfaces.InputAdapter) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.adapter = adapter
	e.logger.Printf("Updated input adapter")
}

// ValidateConnection checks if the provider connection is working
func (e *LLMEngine) ValidateConnection() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.logger.Printf("Validating provider connection")

	// Check if provider supports connection validation
	if validator, ok := e.provider.(interface{ ValidateConnection() error }); ok {
		return validator.ValidateConnection()
	}

	// Fallback: try a simple test request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := &types.ModelRequest{
		Model: "test",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "test",
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens": 1,
		},
	}

	_, err := e.provider.GenerateResponse(ctx, testReq)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	e.logger.Printf("Provider connection validation successful")
	return nil
}

// generateRequestID generates a unique request ID using UUID
func generateRequestID() string {
	return fmt.Sprintf("req_%s", uuid.New().String())
}

// Ensure LLMEngine implements the LLMEngine interface
var _ interfaces.LLMEngine = (*LLMEngine)(nil)
