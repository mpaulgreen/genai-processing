package processor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"genai-processing/internal/config"
	"genai-processing/internal/parser/recovery"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// Mock implementations for testing

// mockContextManager implements interfaces.ContextManager for testing
type mockContextManager struct {
	sessions map[string]*types.ConversationContext
	pronouns map[string]string
	errors   map[string]error
}

func newMockContextManager() *mockContextManager {
	return &mockContextManager{
		sessions: make(map[string]*types.ConversationContext),
		pronouns: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockContextManager) UpdateContext(sessionID string, query string, response *types.StructuredQuery) error {
	if m.sessions[sessionID] == nil {
		m.sessions[sessionID] = &types.ConversationContext{
			SessionID:    sessionID,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
	}
	return nil
}

func (m *mockContextManager) UpdateContextWithUser(sessionID string, userID string, query string, response *types.StructuredQuery) error {
	if m.sessions[sessionID] == nil {
		m.sessions[sessionID] = &types.ConversationContext{
			SessionID:    sessionID,
			UserID:       userID,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
	} else {
		m.sessions[sessionID].UserID = userID
		m.sessions[sessionID].LastActivity = time.Now()
	}
	return nil
}

func (m *mockContextManager) ResolvePronouns(query string, sessionID string) (string, error) {
	if err, exists := m.errors[query]; exists {
		return query, err
	}
	if resolved, exists := m.pronouns[query]; exists {
		return resolved, nil
	}
	return query, nil
}

func (m *mockContextManager) GetContext(sessionID string) (*types.ConversationContext, error) {
	if context, exists := m.sessions[sessionID]; exists {
		return context, nil
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

// mockLLMEngine implements interfaces.LLMEngine for testing
type mockLLMEngine struct {
	responses map[string]*types.RawResponse
	errors    map[string]error
}

func newMockLLMEngine() *mockLLMEngine {
	return &mockLLMEngine{
		responses: make(map[string]*types.RawResponse),
		errors:    make(map[string]error),
	}
}

func (m *mockLLMEngine) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	if err, exists := m.errors[query]; exists {
		return nil, err
	}
	if response, exists := m.responses[query]; exists {
		return response, nil
	}
	// Default mock response
	return &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 20}`,
		ModelInfo: map[string]interface{}{
			"model": "claude-3-5-sonnet-20241022",
		},
	}, nil
}

func (m *mockLLMEngine) GetSupportedModels() []string {
	return []string{"claude-3-5-sonnet-20241022"}
}

func (m *mockLLMEngine) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	return &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": req.ProcessingRequest.Query,
			},
		},
	}, nil
}

func (m *mockLLMEngine) ValidateConnection() error {
	// Mock implementation - always succeeds
	return nil
}

// mockParser implements interfaces.Parser for testing
func newMockRetryParser() *recovery.RetryParser {
	// Create a real RetryParser with mock configuration
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          3,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.95,
		EnableReprompting:   false, // Disable for testing
	}

	// Create RetryParser with nil dependencies for testing
	retryParser := recovery.NewRetryParser(retryConfig, nil, nil)

	// Register a mock parser that always succeeds
	mockParser := &mockParser{
		queries:    make(map[string]*types.StructuredQuery),
		errors:     make(map[string]error),
		confidence: 0.95,
	}

	retryParser.RegisterParser(recovery.StrategySpecific, mockParser)
	retryParser.RegisterParser(recovery.StrategyGeneric, mockParser)

	return retryParser
}

// mockParser implements interfaces.Parser for testing
type mockParser struct {
	queries    map[string]*types.StructuredQuery
	errors     map[string]error
	confidence float64
}

func (m *mockParser) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	if err, exists := m.errors[raw.Content]; exists {
		return nil, err
	}
	if query, exists := m.queries[raw.Content]; exists {
		return query, nil
	}
	// Default mock query
	return &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("pods"),
		Limit:     20,
	}, nil
}

func (m *mockParser) CanHandle(modelType string) bool {
	return true
}

func (m *mockParser) GetConfidence() float64 {
	return m.confidence
}

// mockSafetyValidator implements interfaces.SafetyValidator for testing
type mockSafetyValidator struct {
	results map[string]*interfaces.ValidationResult
	errors  map[string]error
}

func newMockSafetyValidator() *mockSafetyValidator {
	return &mockSafetyValidator{
		results: make(map[string]*interfaces.ValidationResult),
		errors:  make(map[string]error),
	}
}

func (m *mockSafetyValidator) ValidateQuery(query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	queryKey := query.LogSource + "_" + query.Verb.GetString()
	if err, exists := m.errors[queryKey]; exists {
		return nil, err
	}
	if result, exists := m.results[queryKey]; exists {
		return result, nil
	}
	// Default mock validation result
	return &interfaces.ValidationResult{
		IsValid:   true,
		RuleName:  "mock_validator",
		Severity:  "info",
		Message:   "Query validation completed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

func (m *mockSafetyValidator) GetApplicableRules() []interfaces.ValidationRule {
	return []interfaces.ValidationRule{}
}

// spyProvider implements interfaces.LLMProvider and records whether GenerateResponse was called
type spyProvider struct {
	called bool
}

var _ interfaces.LLMProvider = (*spyProvider)(nil)

func (s *spyProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	s.called = true
	return &types.RawResponse{Content: `{"ok": true}`}, nil
}

func (s *spyProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{Name: "claude-3-5-sonnet-20241022", Provider: "anthropic"}
}

func (s *spyProvider) SupportsStreaming() bool { return false }

func (s *spyProvider) ValidateConnection() error { return nil }

// engineWithProvider implements interfaces.LLMEngine and exposes GetProvider for direct provider access
type engineWithProvider struct {
	provider    interfaces.LLMProvider
	adaptCalled bool
}

var _ interfaces.LLMEngine = (*engineWithProvider)(nil)

func (e *engineWithProvider) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	// Fallback path; should not be used when provider path is taken
	return &types.RawResponse{Content: `fallback`}, nil
}

func (e *engineWithProvider) GetSupportedModels() []string {
	return []string{"claude-3-5-sonnet-20241022"}
}

func (e *engineWithProvider) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	e.adaptCalled = true
	return &types.ModelRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []interface{}{
			map[string]interface{}{"role": "user", "content": req.ProcessingRequest.Query},
		},
	}, nil
}

func (e *engineWithProvider) ValidateConnection() error { return nil }

// Not part of the interface; used by the processor via type assertion
func (e *engineWithProvider) GetProvider() interfaces.LLMProvider { return e.provider }

// Test functions

func TestNewGenAIProcessor(t *testing.T) {
	processor := NewGenAIProcessor()

	if processor == nil {
		t.Fatal("NewGenAIProcessor returned nil")
	}

	if processor.contextManager == nil {
		t.Error("contextManager should not be nil")
	}

	if processor.llmEngine == nil {
		t.Error("llmEngine should not be nil")
	}

	if processor.RetryParser == nil {
		t.Error("RetryParser should not be nil")
	}

	if processor.safetyValidator == nil {
		t.Error("safetyValidator should not be nil")
	}

	if processor.defaultModel == "" {
		t.Error("defaultModel should not be empty")
	}

	if processor.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestProcessQuery_Success(t *testing.T) {
	// Create processor with mocked dependencies
	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       newMockLLMEngine(),
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-1",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.StructuredQuery == nil {
		t.Error("StructuredQuery should not be nil")
	}

	if response.Confidence == 0.0 {
		t.Error("Confidence should be greater than 0")
	}

	if response.ValidationInfo == nil {
		t.Error("ValidationInfo should not be nil")
	}

	if response.Error != "" {
		t.Errorf("Error should be empty, got: %s", response.Error)
	}
}

func TestProcessQuery_ContextResolutionFailure(t *testing.T) {
	mockContext := newMockContextManager()
	mockContext.errors = map[string]error{
		"Who deleted the customer CRD yesterday?": fmt.Errorf("context resolution failed"),
	}

	processor := &GenAIProcessor{
		contextManager:  mockContext,
		llmEngine:       newMockLLMEngine(),
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-2",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery should not return error for context resolution failure: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Error == "" {
		t.Error("Response should contain error information")
	}

	if !strings.Contains(response.Error, "context_resolution_failed") {
		t.Errorf("Error should contain 'context_resolution_failed', got: %s", response.Error)
	}
}

func TestProcessQuery_LLMProcessingFailure(t *testing.T) {
	mockLLM := newMockLLMEngine()
	mockLLM.errors = map[string]error{
		"Who deleted the customer CRD yesterday?": fmt.Errorf("LLM API error"),
	}

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       mockLLM,
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-3",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery should not return error for LLM processing failure: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Error == "" {
		t.Error("Response should contain error information")
	}

	if !strings.Contains(response.Error, "llm_processing_failed") {
		t.Errorf("Error should contain 'llm_processing_failed', got: %s", response.Error)
	}
}

func TestProcessQuery_ParsingFailure(t *testing.T) {
	// Create a RetryParser with a mock parser that always fails
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          1,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.95,
		EnableReprompting:   false,
	}

	failingParser := &mockParser{errors: make(map[string]error)}
	failingParser.errors = map[string]error{
		`{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 20}`: fmt.Errorf("parsing failed"),
	}

	retryParser := recovery.NewRetryParser(retryConfig, nil, nil)
	retryParser.RegisterParser(recovery.StrategySpecific, failingParser)
	retryParser.RegisterParser(recovery.StrategyGeneric, failingParser)

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       newMockLLMEngine(),
		RetryParser:     retryParser,
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-4",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery should not return error for parsing failure: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Error == "" {
		t.Error("Response should contain error information")
	}

	if !strings.Contains(response.Error, "parsing_failed") {
		t.Errorf("Error should contain 'parsing_failed', got: %s", response.Error)
	}
}

func TestProcessQuery_ValidationFailure(t *testing.T) {
	mockValidator := newMockSafetyValidator()
	mockValidator.errors = map[string]error{
		"kube-apiserver_get": fmt.Errorf("validation failed"),
	}

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       newMockLLMEngine(),
		RetryParser:     newMockRetryParser(),
		safetyValidator: mockValidator,
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-5",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery should not return error for validation failure: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Error == "" {
		t.Error("Response should contain error information")
	}

	if !strings.Contains(response.Error, "validation_failed") {
		t.Errorf("Error should contain 'validation_failed', got: %s", response.Error)
	}
}

func TestProcessQuery_WithPronounResolution(t *testing.T) {
	mockContext := newMockContextManager()
	mockContext.pronouns = map[string]string{
		"Who deleted it?": "Who deleted the customer CRD?",
	}

	processor := &GenAIProcessor{
		contextManager:  mockContext,
		llmEngine:       newMockLLMEngine(),
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted it?",
		SessionID: "test-session-6",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.StructuredQuery == nil {
		t.Error("StructuredQuery should not be nil")
	}

	if response.Error != "" {
		t.Errorf("Error should be empty, got: %s", response.Error)
	}
}

func TestProcessQuery_WithCustomStructuredQuery(t *testing.T) {
	// Create a RetryParser with a mock parser that returns custom queries
	retryConfig := &recovery.RetryConfig{
		MaxRetries:          1,
		RetryDelay:          time.Millisecond * 10,
		ConfidenceThreshold: 0.95,
		EnableReprompting:   false,
	}

	customParser := &mockParser{queries: make(map[string]*types.StructuredQuery)}
	customQuery := &types.StructuredQuery{
		LogSource: "oauth-server",
		Verb:      *types.NewStringOrArray("get"),
		Resource:  *types.NewStringOrArray("users"),
		Timeframe: "1_hour_ago",
		Limit:     50,
	}
	customParser.queries = map[string]*types.StructuredQuery{
		`{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 20}`: customQuery,
	}

	retryParser := recovery.NewRetryParser(retryConfig, nil, nil)
	retryParser.RegisterParser(recovery.StrategySpecific, customParser)
	retryParser.RegisterParser(recovery.StrategyGeneric, customParser)

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       newMockLLMEngine(),
		RetryParser:     retryParser,
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Show me all failed authentication attempts in the last hour",
		SessionID: "test-session-7",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.StructuredQuery == nil {
		t.Error("StructuredQuery should not be nil")
	}

	// Verify the custom query was returned
	query, ok := response.StructuredQuery.(*types.StructuredQuery)
	if !ok {
		t.Fatal("StructuredQuery should be of type *types.StructuredQuery")
	}

	if query.LogSource != "oauth-server" {
		t.Errorf("Expected LogSource 'oauth-server', got: %s", query.LogSource)
	}

	if query.Timeframe != "1_hour_ago" {
		t.Errorf("Expected Timeframe '1_hour_ago', got: %s", query.Timeframe)
	}

	if query.Limit != 50 {
		t.Errorf("Expected Limit 50, got: %d", query.Limit)
	}
}

func TestProcessQuery_WithCustomValidationResult(t *testing.T) {
	mockValidator := newMockSafetyValidator()
	customResult := &interfaces.ValidationResult{
		IsValid:   false,
		RuleName:  "custom_rule",
		Severity:  "critical",
		Message:   "Query contains forbidden patterns",
		Errors:    []string{"Forbidden resource access"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	mockValidator.results = map[string]*interfaces.ValidationResult{
		"kube-apiserver_get": customResult,
	}

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       newMockLLMEngine(),
		RetryParser:     newMockRetryParser(),
		safetyValidator: mockValidator,
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-8",
	}

	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, req)

	if err != nil {
		t.Fatalf("ProcessQuery failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.ValidationInfo == nil {
		t.Error("ValidationInfo should not be nil")
	}

	// Verify the custom validation result was returned
	validation, ok := response.ValidationInfo.(*interfaces.ValidationResult)
	if !ok {
		t.Fatal("ValidationInfo should be of type *interfaces.ValidationResult")
	}

	if validation.IsValid {
		t.Error("Expected validation to be invalid")
	}

	if validation.RuleName != "custom_rule" {
		t.Errorf("Expected RuleName 'custom_rule', got: %s", validation.RuleName)
	}

	if validation.Severity != "critical" {
		t.Errorf("Expected Severity 'critical', got: %s", validation.Severity)
	}

	if len(validation.Errors) == 0 {
		t.Error("Expected validation errors")
	}
}

// Ensure the processor uses AdaptInput and the direct provider path when the engine exposes GetProvider()
func TestProcessQuery_UsesAdapterAndProviderDirectPath(t *testing.T) {
	prov := &spyProvider{}
	eng := &engineWithProvider{provider: prov}

	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       eng,
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{Query: "test adapter path", SessionID: "sess-123"}
	ctx := context.Background()
	resp, err := processor.ProcessQuery(ctx, req)
	if err != nil {
		t.Fatalf("ProcessQuery returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Response should not be nil")
	}
	if !eng.adaptCalled {
		t.Error("Expected AdaptInput to be called")
	}
	if !prov.called {
		t.Error("Expected provider.GenerateResponse to be called directly via GetProvider path")
	}
}

func TestResolveContext(t *testing.T) {
	mockContext := newMockContextManager()
	mockContext.pronouns = map[string]string{
		"Who deleted it?":     "Who deleted the customer CRD?",
		"Show me his actions": "Show me john.doe's actions",
	}

	processor := &GenAIProcessor{
		contextManager: mockContext,
		logger:         log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	tests := []struct {
		name      string
		query     string
		sessionID string
		expected  string
	}{
		{
			name:      "No pronouns to resolve",
			query:     "Who deleted the customer CRD?",
			sessionID: "test-session",
			expected:  "Who deleted the customer CRD?",
		},
		{
			name:      "With pronoun resolution",
			query:     "Who deleted it?",
			sessionID: "test-session",
			expected:  "Who deleted the customer CRD?",
		},
		{
			name:      "Another pronoun resolution",
			query:     "Show me his actions",
			sessionID: "test-session",
			expected:  "Show me john.doe's actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.resolveContext(tt.query, tt.sessionID)
			if err != nil {
				t.Fatalf("resolveContext failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCreateErrorResponse(t *testing.T) {
	processor := &GenAIProcessor{
		logger: log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	err := fmt.Errorf("test error")
	response := processor.createErrorResponse("test_error_type", err)

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.StructuredQuery != nil {
		t.Error("StructuredQuery should be nil for error response")
	}

	if response.Confidence != 0.0 {
		t.Error("Confidence should be 0.0 for error response")
	}

	if response.ValidationInfo != nil {
		t.Error("ValidationInfo should be nil for error response")
	}

	if response.Error == "" {
		t.Error("Error should not be empty")
	}

	if !strings.Contains(response.Error, "test_error_type") {
		t.Errorf("Error should contain 'test_error_type', got: %s", response.Error)
	}

	if !strings.Contains(response.Error, "test error") {
		t.Errorf("Error should contain 'test error', got: %s", response.Error)
	}
}

func TestSimpleLLMEngine(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	engine := createLLMEngine(mockProvider)

	if engine == nil {
		t.Fatal("createLLMEngine returned nil")
	}

	models := engine.GetSupportedModels()
	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	if models[0] != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected 'claude-3-5-sonnet-20241022', got '%s'", models[0])
	}
}

func TestNewGenAIProcessorFromConfig_ValidatesAndBuilds(t *testing.T) {
	app := config.GetDefaultConfig()
	// Ensure API keys present so providers pass registration
	c := app.Models.Providers["claude"]
	c.APIKey = "test-key"
	app.Models.Providers["claude"] = c

	p, err := NewGenAIProcessorFromConfig(app)
	if err != nil {
		t.Fatalf("NewGenAIProcessorFromConfig returned error: %v", err)
	}
	if p == nil {
		t.Fatal("processor is nil")
	}
	if p.defaultModel == "" {
		t.Error("defaultModel should be set from config")
	}
}

func TestProcessQuery_TimeoutAndRetry(t *testing.T) {
	// Build a processor with small timeout and one retry using mock engineWithProvider path
	prov := &flakyProvider{fails: intPtr(1)} // first attempt fails, second succeeds
	eng := &engineWithProvider{provider: prov}
	processor := &GenAIProcessor{
		contextManager:  newMockContextManager(),
		llmEngine:       eng,
		RetryParser:     newMockRetryParser(),
		safetyValidator: newMockSafetyValidator(),
		defaultModel:    "claude-3-5-sonnet-20241022",
		providerTimeout: 50 * time.Millisecond,
		retryAttempts:   1,
		retryDelay:      10 * time.Millisecond,
		logger:          log.New(log.Writer(), "[TestProcessor] ", log.LstdFlags),
	}

	req := &types.ProcessingRequest{Query: "retry please", SessionID: "sess-retry"}
	ctx := context.Background()
	resp, err := processor.ProcessQuery(ctx, req)
	if err != nil {
		t.Fatalf("ProcessQuery returned error: %v", err)
	}
	if resp == nil || resp.Error != "" {
		t.Fatalf("expected success after retry, got resp=%v err=%v", resp, err)
	}
}

// flakyProvider fails a fixed number of initial attempts, then succeeds
type flakyProvider struct{ fails *int }

func intPtr(i int) *int { return &i }

func (f *flakyProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	if f.fails != nil && *f.fails > 0 {
		*f.fails = *f.fails - 1
		return nil, fmt.Errorf("temporary error: timeout")
	}
	return &types.RawResponse{Content: `{"log_source":"kube-apiserver","limit":20}`}, nil
}

func (f *flakyProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{Name: "claude-3-5-sonnet-20241022", Provider: "anthropic"}
}
func (f *flakyProvider) SupportsStreaming() bool   { return false }
func (f *flakyProvider) ValidateConnection() error { return nil }

// mockLLMProvider implements interfaces.LLMProvider for testing
type mockLLMProvider struct{}

func (m *mockLLMProvider) GenerateResponse(ctx context.Context, request *types.ModelRequest) (*types.RawResponse, error) {
	return &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "limit": 20}`,
		ModelInfo: map[string]interface{}{
			"model": "claude-3-5-sonnet-20241022",
		},
	}, nil
}

func (m *mockLLMProvider) GetModelInfo() types.ModelInfo {
	return types.ModelInfo{
		Name:     "claude-3-5-sonnet-20241022",
		Provider: "anthropic",
	}
}

func (m *mockLLMProvider) SupportsStreaming() bool {
	return false
}

func (m *mockLLMProvider) ValidateConnection() error {
	return nil
}

// Helper function to add missing imports
func init() {
	// This function ensures all required imports are available
	_ = fmt.Sprintf
	_ = strings.Contains
}
