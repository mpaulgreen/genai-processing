package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// MockLLMProvider implements the LLMProvider interface for testing
type MockLLMProvider struct {
	shouldFail    bool
	responseDelay time.Duration
	response      *types.RawResponse
	callCount     int
	lastRequest   *types.ModelRequest
	mu            sync.Mutex // Protect fields from concurrent access
}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		response: &types.RawResponse{
			Content: `{"log_source": "kube-apiserver", "verb": "get", "limit": 20}`,
			ModelInfo: map[string]interface{}{
				"model": "test-model",
			},
		},
	}
}

func (m *MockLLMProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.callCount++
	m.lastRequest = request

	if m.shouldFail {
		return nil, errors.New("mock provider error")
	}

	if m.responseDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.responseDelay):
		}
	}

	return m.response, nil
}

func (m *MockLLMProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:     "test-model",
		Provider: "test-provider",
		Version:  "1.0",
	}
}

func (m *MockLLMProvider) SupportsStreaming() bool {
	return false
}

func (m *MockLLMProvider) ValidateConnection() error {
	if m.shouldFail {
		return errors.New("mock connection error")
	}
	return nil
}

// GetCallCount returns the number of times GenerateResponse was called (thread-safe)
func (m *MockLLMProvider) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// MockInputAdapter implements the InputAdapter interface for testing
type MockInputAdapter struct {
	shouldFail     bool
	adaptedRequest *types.ModelRequest
	callCount      int
	lastRequest    *types.InternalRequest
	mu             sync.Mutex // Protect fields from concurrent access
}

func NewMockInputAdapter() *MockInputAdapter {
	return &MockInputAdapter{
		adaptedRequest: &types.ModelRequest{
			Model: "test-model",
			Messages: []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "test query",
				},
			},
		},
	}
}

func (m *MockInputAdapter) AdaptRequest(req *types.InternalRequest) (*types.ModelRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.callCount++
	m.lastRequest = req

	if m.shouldFail {
		return nil, errors.New("mock adapter error")
	}

	return m.adaptedRequest, nil
}

func (m *MockInputAdapter) FormatPrompt(prompt string, examples []types.Example) (string, error) {
	if m.shouldFail {
		return "", errors.New("mock format error")
	}
	return "formatted prompt", nil
}

func (m *MockInputAdapter) GetAPIParameters() map[string]interface{} {
	return map[string]interface{}{
		"test_param": "test_value",
	}
}

// GetCallCount returns the number of times AdaptRequest was called (thread-safe)
func (m *MockInputAdapter) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestNewLLMEngine(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()

	engine := NewLLMEngine(provider, adapter)

	if engine == nil {
		t.Fatal("NewLLMEngine returned nil")
	}

	if engine.provider != provider {
		t.Error("Provider not set correctly")
	}

	if engine.adapter != adapter {
		t.Error("Adapter not set correctly")
	}
}

func TestLLMEngine_ProcessQuery_Success(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	ctx := context.Background()
	query := "Who deleted the customer CRD yesterday?"
	context := types.ConversationContext{
		SessionID: "test-session",
		UserID:    "test-user",
	}

	response, err := engine.ProcessQuery(ctx, query, context)

	if err != nil {
		t.Fatalf("ProcessQuery failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.Content == "" {
		t.Error("Response content is empty")
	}

	// Verify adapter was called
	if adapter.callCount != 1 {
		t.Errorf("Expected adapter to be called once, got %d", adapter.callCount)
	}

	// Verify provider was called
	if provider.callCount != 1 {
		t.Errorf("Expected provider to be called once, got %d", provider.callCount)
	}

	// Verify the request was passed correctly
	if adapter.lastRequest == nil {
		t.Error("Adapter lastRequest is nil")
	} else if adapter.lastRequest.ProcessingRequest.Query != query {
		t.Errorf("Expected query '%s', got '%s'", query, adapter.lastRequest.ProcessingRequest.Query)
	}

	// Verify request ID format (should be UUID-based)
	if adapter.lastRequest.RequestID == "" {
		t.Error("Request ID should not be empty")
	} else if len(adapter.lastRequest.RequestID) < 40 { // UUID + "req_" prefix
		t.Errorf("Request ID should be UUID-based, got: %s", adapter.lastRequest.RequestID)
	}
}

func TestLLMEngine_ProcessQuery_AdapterFailure(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	adapter.shouldFail = true
	engine := NewLLMEngine(provider, adapter)

	ctx := context.Background()
	query := "test query"
	context := types.ConversationContext{
		SessionID: "test-session",
	}

	_, err := engine.ProcessQuery(ctx, query, context)

	if err == nil {
		t.Fatal("Expected error when adapter fails")
	}

	if err.Error() != "failed to adapt input: adapter failed to adapt request: mock adapter error" {
		t.Errorf("Expected adapter error, got: %v", err)
	}

	// Verify provider was not called
	if provider.callCount != 0 {
		t.Errorf("Expected provider not to be called, got %d calls", provider.callCount)
	}
}

func TestLLMEngine_ProcessQuery_ProviderFailure(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.shouldFail = true
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	ctx := context.Background()
	query := "test query"
	context := types.ConversationContext{
		SessionID: "test-session",
	}

	_, err := engine.ProcessQuery(ctx, query, context)

	if err == nil {
		t.Fatal("Expected error when provider fails")
	}

	if err.Error() != "failed to generate response: mock provider error" {
		t.Errorf("Expected provider error, got: %v", err)
	}

	// Verify adapter was called
	if adapter.callCount != 1 {
		t.Errorf("Expected adapter to be called once, got %d", adapter.callCount)
	}
}

func TestLLMEngine_ProcessQuery_ContextCancellation(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.responseDelay = 100 * time.Millisecond
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	query := "test query"
	convContext := types.ConversationContext{
		SessionID: "test-session",
	}

	_, err := engine.ProcessQuery(ctx, query, convContext)

	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context deadline exceeded or canceled, got: %v", err)
	}
}

func TestLLMEngine_GetSupportedModels(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	models := engine.GetSupportedModels()

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	expectedModels := map[string]bool{
		"test-model":    true,
		"test-provider": true,
	}

	for _, model := range models {
		if !expectedModels[model] {
			t.Errorf("Unexpected model: %s", model)
		}
	}
}

func TestLLMEngine_AdaptInput_Success(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	req := &types.InternalRequest{
		RequestID: "test-request",
		ProcessingRequest: types.ProcessingRequest{
			Query:     "test query",
			SessionID: "test-session",
		},
	}

	modelReq, err := engine.AdaptInput(req)

	if err != nil {
		t.Fatalf("AdaptInput failed: %v", err)
	}

	if modelReq == nil {
		t.Fatal("ModelRequest is nil")
	}

	if modelReq.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", modelReq.Model)
	}

	// Verify adapter was called
	if adapter.callCount != 1 {
		t.Errorf("Expected adapter to be called once, got %d", adapter.callCount)
	}
}

func TestLLMEngine_AdaptInput_NilRequest(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	_, err := engine.AdaptInput(nil)

	if err == nil {
		t.Fatal("Expected error for nil request")
	}

	if err.Error() != "internal request cannot be nil" {
		t.Errorf("Expected nil request error, got: %v", err)
	}
}

func TestLLMEngine_AdaptInput_AdapterFailure(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	adapter.shouldFail = true
	engine := NewLLMEngine(provider, adapter)

	req := &types.InternalRequest{
		RequestID: "test-request",
		ProcessingRequest: types.ProcessingRequest{
			Query:     "test query",
			SessionID: "test-session",
		},
	}

	_, err := engine.AdaptInput(req)

	if err == nil {
		t.Fatal("Expected error when adapter fails")
	}

	if err.Error() != "adapter failed to adapt request: mock adapter error" {
		t.Errorf("Expected adapter error, got: %v", err)
	}
}

func TestLLMEngine_ValidateConnection_Success(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	err := engine.ValidateConnection()

	if err != nil {
		t.Fatalf("ValidateConnection failed: %v", err)
	}
}

func TestLLMEngine_ValidateConnection_Failure(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.shouldFail = true
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	err := engine.ValidateConnection()

	if err == nil {
		t.Fatal("Expected error when connection validation fails")
	}

	if err.Error() != "mock connection error" {
		t.Errorf("Expected connection error, got: %v", err)
	}
}

func TestLLMEngine_UpdateProvider(t *testing.T) {
	provider1 := NewMockLLMProvider()
	provider2 := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider1, adapter)

	// Verify initial provider
	if engine.GetProvider() != provider1 {
		t.Error("Initial provider not set correctly")
	}

	// Update provider
	engine.UpdateProvider(provider2)

	// Verify provider was updated
	if engine.GetProvider() != provider2 {
		t.Error("Provider not updated correctly")
	}
}

func TestLLMEngine_UpdateAdapter(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter1 := NewMockInputAdapter()
	adapter2 := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter1)

	// Verify initial adapter
	if engine.GetAdapter() != adapter1 {
		t.Error("Initial adapter not set correctly")
	}

	// Update adapter
	engine.UpdateAdapter(adapter2)

	// Verify adapter was updated
	if engine.GetAdapter() != adapter2 {
		t.Error("Adapter not updated correctly")
	}
}

func TestLLMEngine_ConcurrentAccess(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	// Test concurrent access to ProcessQuery
	ctx := context.Background()
	query := "test query"
	context := types.ConversationContext{
		SessionID: "test-session",
	}

	// Run multiple goroutines concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := engine.ProcessQuery(ctx, query, context)
			if err != nil {
				t.Errorf("ProcessQuery failed in goroutine: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total calls
	expectedCalls := numGoroutines
	if adapter.GetCallCount() != expectedCalls {
		t.Errorf("Expected %d adapter calls, got %d", expectedCalls, adapter.GetCallCount())
	}

	if provider.GetCallCount() != expectedCalls {
		t.Errorf("Expected %d provider calls, got %d", expectedCalls, provider.GetCallCount())
	}
}

func TestLLMEngine_InterfaceCompliance(t *testing.T) {
	provider := NewMockLLMProvider()
	adapter := NewMockInputAdapter()
	engine := NewLLMEngine(provider, adapter)

	// This test ensures the engine implements the LLMEngine interface
	var _ interfaces.LLMEngine = engine

	// Test that ValidateConnection is callable (interface compliance)
	err := engine.ValidateConnection()
	if err != nil {
		t.Errorf("ValidateConnection should not fail for mock provider: %v", err)
	}
}
