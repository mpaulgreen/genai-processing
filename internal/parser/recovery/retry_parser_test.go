package recovery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"genai-processing/pkg/errors"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// MockParser implements the Parser interface for testing
type MockParser struct {
	canHandle    bool
	shouldFail   bool
	confidence   float64
	modelType    string
	parseResults []*types.StructuredQuery
	parseErrors  []error
	callCount    int
}

func NewMockParser(canHandle bool, shouldFail bool, confidence float64) *MockParser {
	return &MockParser{
		canHandle:  canHandle,
		shouldFail: shouldFail,
		confidence: confidence,
		callCount:  0,
	}
}

func (m *MockParser) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	m.callCount++
	m.modelType = modelType

	if m.shouldFail {
		if m.callCount <= len(m.parseErrors) {
			return nil, m.parseErrors[m.callCount-1]
		}
		return nil, errors.NewParsingError("mock parsing failed", errors.ComponentParser, "mock_parser", 0.0, raw.Content)
	}

	if m.callCount <= len(m.parseResults) {
		return m.parseResults[m.callCount-1], nil
	}

	// Default successful result
	return &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Limit:     20,
	}, nil
}

func (m *MockParser) CanHandle(modelType string) bool {
	return m.canHandle
}

func (m *MockParser) GetConfidence() float64 {
	return m.confidence
}

// MockLLMEngine implements the LLMEngine interface for testing
type MockLLMEngine struct {
	shouldFail bool
	responses  []*types.RawResponse
	callCount  int
}

func NewMockLLMEngine(shouldFail bool) *MockLLMEngine {
	return &MockLLMEngine{
		shouldFail: shouldFail,
		callCount:  0,
	}
}

func (m *MockLLMEngine) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	m.callCount++

	if m.shouldFail {
		return nil, errors.NewProcessingError("mock llm failed", "mock_llm", errors.ComponentLLMEngine, true)
	}

	if m.callCount <= len(m.responses) {
		return m.responses[m.callCount-1], nil
	}

	// Default successful response
	return &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "limit": 20}`,
	}, nil
}

func (m *MockLLMEngine) GetSupportedModels() []string {
	return []string{"claude", "openai"}
}

func (m *MockLLMEngine) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	return &types.ModelRequest{}, nil
}

func (m *MockLLMEngine) ValidateConnection() error {
	return nil
}

// MockContextManager implements the ContextManager interface for testing
type MockContextManager struct {
	shouldFail bool
	context    *types.ConversationContext
}

func NewMockContextManager(shouldFail bool) *MockContextManager {
	return &MockContextManager{
		shouldFail: shouldFail,
		context:    types.NewConversationContext("test-session", "test-user"),
	}
}

func (m *MockContextManager) GetContext(sessionID string) (*types.ConversationContext, error) {
	if m.shouldFail {
		return nil, errors.NewContextError("mock context failed", "mock_context", sessionID, "get", "context")
	}
	return m.context, nil
}

func (m *MockContextManager) UpdateContext(sessionID string, query string, response *types.StructuredQuery) error {
	return nil
}

func (m *MockContextManager) ResolvePronouns(query string, sessionID string) (string, error) {
	return query, nil
}

// Test helper functions
func createTestRawResponse(content string) *types.RawResponse {
	return &types.RawResponse{
		Content: content,
	}
}

func createTestStructuredQuery() *types.StructuredQuery {
	return &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Limit:     20,
	}
}

// Test cases
func TestNewRetryParser(t *testing.T) {
	tests := []struct {
		name        string
		config      *RetryConfig
		llmEngine   interfaces.LLMEngine
		contextMgr  interfaces.ContextManager
		expectError bool
	}{
		{
			name:        "valid configuration",
			config:      &RetryConfig{MaxRetries: 3, ConfidenceThreshold: 0.8},
			llmEngine:   NewMockLLMEngine(false),
			contextMgr:  NewMockContextManager(false),
			expectError: false,
		},
		{
			name:        "nil configuration uses defaults",
			config:      nil,
			llmEngine:   NewMockLLMEngine(false),
			contextMgr:  NewMockContextManager(false),
			expectError: false,
		},
		{
			name:        "nil dependencies allowed",
			config:      &RetryConfig{MaxRetries: 3},
			llmEngine:   nil,
			contextMgr:  nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryParser := NewRetryParser(tt.config, tt.llmEngine, tt.contextMgr)

			if retryParser == nil {
				t.Fatal("NewRetryParser returned nil")
			}

			if tt.config == nil {
				// Check default values
				config := retryParser.GetConfiguration()
				if config.MaxRetries != 3 {
					t.Errorf("expected default MaxRetries 3, got %d", config.MaxRetries)
				}
				if config.ConfidenceThreshold != 0.7 {
					t.Errorf("expected default ConfidenceThreshold 0.7, got %f", config.ConfidenceThreshold)
				}
			}
		})
	}
}

func TestRetryParser_RegisterParser(t *testing.T) {
	retryParser := NewRetryParser(nil, nil, nil)

	specificParser := NewMockParser(true, false, 0.9)
	genericParser := NewMockParser(true, false, 0.7)
	errorParser := NewMockParser(true, false, 0.5)

	// Register parsers
	retryParser.RegisterParser(StrategySpecific, specificParser)
	retryParser.RegisterParser(StrategyGeneric, genericParser)
	retryParser.RegisterParser(StrategyError, errorParser)

	// Check statistics
	stats := retryParser.GetRetryStatistics()
	strategies := stats["registered_strategies"].([]string)

	expectedStrategies := []string{"specific", "generic", "error"}
	for _, expected := range expectedStrategies {
		found := false
		for _, actual := range strategies {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected strategy %s not found in registered strategies", expected)
		}
	}
}

func TestRetryParser_ParseWithRetry_Success(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          2,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   false,
	}

	retryParser := NewRetryParser(config, nil, nil)

	// Register a successful parser
	successParser := NewMockParser(true, false, 0.9)
	successParser.parseResults = []*types.StructuredQuery{createTestStructuredQuery()}
	retryParser.RegisterParser(StrategySpecific, successParser)

	raw := createTestRawResponse(`{"log_source": "kube-apiserver", "limit": 20}`)

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, raw, "claude", "test query", "test-session")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.LogSource != "kube-apiserver" {
		t.Errorf("expected LogSource kube-apiserver, got %s", result.LogSource)
	}

	// Should succeed on first attempt
	if successParser.callCount != 1 {
		t.Errorf("expected 1 parser call, got %d", successParser.callCount)
	}
}

func TestRetryParser_ParseWithRetry_FailureRecovery(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          2,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   false,
	}

	retryParser := NewRetryParser(config, nil, nil)

	// Register parsers with different behaviors
	failingParser := NewMockParser(true, true, 0.0)
	successParser := NewMockParser(true, false, 0.9)
	successParser.parseResults = []*types.StructuredQuery{createTestStructuredQuery()}

	retryParser.RegisterParser(StrategySpecific, failingParser)
	retryParser.RegisterParser(StrategyGeneric, successParser)

	raw := createTestRawResponse(`{"log_source": "kube-apiserver", "limit": 20}`)

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, raw, "claude", "test query", "test-session")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Should fail on specific strategy, succeed on generic
	if failingParser.callCount < 1 {
		t.Errorf("expected at least 1 call to failing parser, got %d", failingParser.callCount)
	}
	if successParser.callCount != 1 {
		t.Errorf("expected 1 call to success parser, got %d", successParser.callCount)
	}
}

func TestRetryParser_ParseWithRetry_Reprompting(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          1,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   true,
		RepromptTemplate:    "Please provide JSON for: %s",
	}

	// Mock LLM engine that succeeds on re-prompt
	llmEngine := NewMockLLMEngine(false)
	llmEngine.responses = []*types.RawResponse{
		{Content: `{"log_source": "oauth-server", "limit": 10}`},
	}

	contextMgr := NewMockContextManager(false)
	retryParser := NewRetryParser(config, llmEngine, contextMgr)

	// Register parser with low confidence
	lowConfidenceParser := NewMockParser(true, false, 0.6)
	highConfidenceParser := NewMockParser(true, false, 0.9)
	highConfidenceParser.parseResults = []*types.StructuredQuery{createTestStructuredQuery()}

	retryParser.RegisterParser(StrategySpecific, lowConfidenceParser)
	retryParser.RegisterParser(StrategyGeneric, highConfidenceParser)

	raw := createTestRawResponse(`{"log_source": "kube-apiserver", "limit": 20}`)

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, raw, "claude", "test query", "test-session")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Should attempt re-prompting due to low confidence
	if llmEngine.callCount < 1 {
		t.Errorf("expected at least 1 LLM call for re-prompting, got %d", llmEngine.callCount)
	}
}

func TestRetryParser_ParseWithRetry_AllStrategiesFail(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          1,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   false,
	}

	retryParser := NewRetryParser(config, nil, nil)

	// Register all failing parsers
	failingParser1 := NewMockParser(true, true, 0.0)
	failingParser2 := NewMockParser(true, true, 0.0)
	failingParser3 := NewMockParser(true, true, 0.0)

	retryParser.RegisterParser(StrategySpecific, failingParser1)
	retryParser.RegisterParser(StrategyGeneric, failingParser2)
	retryParser.RegisterParser(StrategyError, failingParser3)

	raw := createTestRawResponse(`invalid content`)

	ctx := context.Background()
	result, err := retryParser.ParseWithRetry(ctx, raw, "claude", "test query", "test-session")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != nil {
		t.Fatal("expected nil result, got result")
	}

	// Should have tried all strategies
	if failingParser1.callCount < 1 {
		t.Errorf("expected at least 1 call to specific parser, got %d", failingParser1.callCount)
	}
	if failingParser2.callCount < 1 {
		t.Errorf("expected at least 1 call to generic parser, got %d", failingParser2.callCount)
	}
	if failingParser3.callCount < 1 {
		t.Errorf("expected at least 1 call to error parser, got %d", failingParser3.callCount)
	}

	// Check error type
	if parsingErr, ok := err.(*errors.ParsingError); !ok {
		t.Errorf("expected ParsingError, got %T", err)
	} else {
		if parsingErr.GetErrorType() != "parsing" {
			t.Errorf("expected error type 'parsing', got %s", parsingErr.GetErrorType())
		}
	}
}

func TestRetryParser_ParseWithRetry_ContextCancellation(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          5,
		RetryDelay:          time.Second * 1,
		ConfidenceThreshold: 0.8,
		EnableReprompting:   false,
	}

	retryParser := NewRetryParser(config, nil, nil)

	// Register a slow parser
	slowParser := NewMockParser(true, true, 0.0)
	retryParser.RegisterParser(StrategySpecific, slowParser)

	raw := createTestRawResponse(`test content`)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	result, err := retryParser.ParseWithRetry(ctx, raw, "claude", "test query", "test-session")

	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}

	if result != nil {
		t.Fatal("expected nil result, got result")
	}

	// Should be context cancellation error
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("expected context error, got %v", err)
	}
}

func TestRetryParser_IsRecoverableError(t *testing.T) {
	retryParser := NewRetryParser(nil, nil, nil)

	tests := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedResult: false,
		},
		{
			name:           "parsing error with low confidence",
			err:            errors.NewParsingError("test", errors.ComponentParser, "test", 0.5, "test content"),
			expectedResult: true,
		},
		{
			name:           "parsing error with high confidence",
			err:            errors.NewParsingError("test", errors.ComponentParser, "test", 0.9, "test content"),
			expectedResult: false,
		},
		{
			name:           "processing error recoverable",
			err:            errors.NewProcessingError("test", "test", errors.ComponentParser, true),
			expectedResult: true,
		},
		{
			name:           "processing error not recoverable",
			err:            errors.NewProcessingError("test", "test", errors.ComponentParser, false),
			expectedResult: false,
		},
		{
			name:           "json error pattern",
			err:            fmt.Errorf("invalid json format"),
			expectedResult: true,
		},
		{
			name:           "parsing error pattern",
			err:            fmt.Errorf("parsing failed"),
			expectedResult: true,
		},
		{
			name:           "timeout error pattern",
			err:            fmt.Errorf("request timeout"),
			expectedResult: true,
		},
		{
			name:           "generic error",
			err:            fmt.Errorf("some other error"),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryParser.IsRecoverableError(tt.err)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestRetryParser_CreateFallbackQuery(t *testing.T) {
	retryParser := NewRetryParser(nil, nil, nil)

	tests := []struct {
		name           string
		content        string
		expectedSource string
		expectedTime   string
	}{
		{
			name:           "oauth content",
			content:        "oauth server logs",
			expectedSource: "oauth-server",
			expectedTime:   "",
		},
		{
			name:           "openshift content",
			content:        "openshift api server",
			expectedSource: "openshift-apiserver",
			expectedTime:   "",
		},
		{
			name:           "today content",
			content:        "logs from today",
			expectedSource: "kube-apiserver",
			expectedTime:   "today",
		},
		{
			name:           "yesterday content",
			content:        "yesterday's logs",
			expectedSource: "kube-apiserver",
			expectedTime:   "yesterday",
		},
		{
			name:           "hour content",
			content:        "last hour",
			expectedSource: "kube-apiserver",
			expectedTime:   "1_hour_ago",
		},
		{
			name:           "empty content",
			content:        "",
			expectedSource: "kube-apiserver",
			expectedTime:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := createTestRawResponse(tt.content)
			result := retryParser.CreateFallbackQuery(raw, "test-model")

			if result.LogSource != tt.expectedSource {
				t.Errorf("expected LogSource %s, got %s", tt.expectedSource, result.LogSource)
			}

			if result.Timeframe != tt.expectedTime {
				t.Errorf("expected Timeframe %s, got %s", tt.expectedTime, result.Timeframe)
			}

			if result.Limit != 20 {
				t.Errorf("expected Limit 20, got %d", result.Limit)
			}
		})
	}
}

func TestRetryParser_ConfigurationValidation(t *testing.T) {
	retryParser := NewRetryParser(nil, nil, nil)

	tests := []struct {
		name        string
		config      *RetryConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: &RetryConfig{
				MaxRetries:          3,
				RetryDelay:          time.Second,
				ConfidenceThreshold: 0.8,
				EnableReprompting:   true,
				RepromptTemplate:    "test template",
			},
			expectError: false,
		},
		{
			name: "negative max retries",
			config: &RetryConfig{
				MaxRetries: -1,
			},
			expectError: true,
		},
		{
			name: "negative retry delay",
			config: &RetryConfig{
				RetryDelay: -time.Second,
			},
			expectError: true,
		},
		{
			name: "confidence threshold too low",
			config: &RetryConfig{
				ConfidenceThreshold: -0.1,
			},
			expectError: true,
		},
		{
			name: "confidence threshold too high",
			config: &RetryConfig{
				ConfidenceThreshold: 1.1,
			},
			expectError: true,
		},
		{
			name: "reprompting enabled without template",
			config: &RetryConfig{
				EnableReprompting: true,
				RepromptTemplate:  "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := retryParser.SetConfiguration(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestRetryParser_GetRetryStatistics(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:          5,
		RetryDelay:          time.Second * 2,
		ConfidenceThreshold: 0.75,
		EnableReprompting:   true,
		RepromptTemplate:    "test template",
	}

	retryParser := NewRetryParser(config, nil, nil)

	// Register some parsers
	retryParser.RegisterParser(StrategySpecific, NewMockParser(true, false, 0.9))
	retryParser.RegisterParser(StrategyGeneric, NewMockParser(true, false, 0.7))

	stats := retryParser.GetRetryStatistics()

	// Check all expected fields
	if stats["max_retries"] != 5 {
		t.Errorf("expected max_retries 5, got %v", stats["max_retries"])
	}
	if stats["retry_delay"] != "2s" {
		t.Errorf("expected retry_delay 2s, got %v", stats["retry_delay"])
	}
	if stats["confidence_threshold"] != 0.75 {
		t.Errorf("expected confidence_threshold 0.75, got %v", stats["confidence_threshold"])
	}
	if stats["enable_reprompting"] != true {
		t.Errorf("expected enable_reprompting true, got %v", stats["enable_reprompting"])
	}

	strategies := stats["registered_strategies"].([]string)
	if len(strategies) != 2 {
		t.Errorf("expected 2 registered strategies, got %d", len(strategies))
	}
}

func TestRetryParser_EdgeCases(t *testing.T) {
	t.Run("nil raw response", func(t *testing.T) {
		retryParser := NewRetryParser(nil, nil, nil)
		ctx := context.Background()

		result, err := retryParser.ParseWithRetry(ctx, nil, "test-model", "test query", "test-session")

		if err == nil {
			t.Fatal("expected error for nil raw response, got nil")
		}
		if result != nil {
			t.Fatal("expected nil result for nil raw response, got result")
		}
	})

	t.Run("no registered parsers", func(t *testing.T) {
		retryParser := NewRetryParser(nil, nil, nil)
		ctx := context.Background()
		raw := createTestRawResponse("test content")

		result, err := retryParser.ParseWithRetry(ctx, raw, "test-model", "test query", "test-session")

		if err == nil {
			t.Fatal("expected error for no registered parsers, got nil")
		}
		if result != nil {
			t.Fatal("expected nil result for no registered parsers, got result")
		}
	})

	t.Run("parser cannot handle model type", func(t *testing.T) {
		retryParser := NewRetryParser(nil, nil, nil)
		parser := NewMockParser(false, false, 0.9) // cannot handle any model
		retryParser.RegisterParser(StrategySpecific, parser)

		ctx := context.Background()
		raw := createTestRawResponse("test content")

		result, err := retryParser.ParseWithRetry(ctx, raw, "test-model", "test query", "test-session")

		if err == nil {
			t.Fatal("expected error for incompatible parser, got nil")
		}
		if result != nil {
			t.Fatal("expected nil result for incompatible parser, got result")
		}
	})
}
