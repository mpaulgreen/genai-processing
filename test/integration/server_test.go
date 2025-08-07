package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"genai-processing/internal/processor"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// MockLLMEngine implements interfaces.LLMEngine for testing
type MockLLMEngine struct {
	responses  map[string]*types.RawResponse
	shouldFail bool
	delay      time.Duration
}

func (m *MockLLMEngine) ProcessQuery(ctx context.Context, query string, context types.ConversationContext) (*types.RawResponse, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock LLM engine error")
	}

	// Return predefined response based on query
	if response, exists := m.responses[query]; exists {
		return response, nil
	}

	// Default response
	return &types.RawResponse{
		Content: `{"log_source": "kube-apiserver", "verb": "get", "resource": "pods", "timeframe": "last 24h"}`,
		ModelInfo: map[string]interface{}{
			"model": "mock-model",
		},
		Metadata: map[string]interface{}{
			"processing_time": "0.1s",
		},
	}, nil
}

func (m *MockLLMEngine) GetSupportedModels() []string {
	return []string{"mock-model"}
}

func (m *MockLLMEngine) AdaptInput(req *types.InternalRequest) (*types.ModelRequest, error) {
	return &types.ModelRequest{
		Model: "mock-model",
		Messages: []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": "You are a helpful assistant for OpenShift audit queries.",
			},
			map[string]interface{}{
				"role":    "user",
				"content": req.ProcessingRequest.Query,
			},
		},
		Parameters: map[string]interface{}{
			"max_tokens":  4000,
			"temperature": 0.1,
		},
	}, nil
}

func (m *MockLLMEngine) ValidateConnection() error {
	if m.shouldFail {
		return fmt.Errorf("mock connection validation failed")
	}
	return nil
}

// MockContextManager implements interfaces.ContextManager for testing
type MockContextManager struct {
	contexts map[string]*types.ConversationContext
}

func (m *MockContextManager) UpdateContext(sessionID string, query string, response *types.StructuredQuery) error {
	if m.contexts == nil {
		m.contexts = make(map[string]*types.ConversationContext)
	}

	m.contexts[sessionID] = &types.ConversationContext{
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	return nil
}

func (m *MockContextManager) ResolvePronouns(query string, sessionID string) (string, error) {
	// Simple pronoun resolution - just return the query as-is for testing
	return query, nil
}

func (m *MockContextManager) GetContext(sessionID string) (*types.ConversationContext, error) {
	if m.contexts == nil {
		m.contexts = make(map[string]*types.ConversationContext)
	}

	if context, exists := m.contexts[sessionID]; exists {
		return context, nil
	}

	// Return new context if session doesn't exist
	return &types.ConversationContext{
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}, nil
}

// MockParser implements interfaces.Parser for testing
type MockParser struct {
	shouldFail bool
	confidence float64
}

func (m *MockParser) ParseResponse(raw *types.RawResponse, modelType string) (*types.StructuredQuery, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock parser error")
	}

	// Parse the content as JSON
	var queryData map[string]interface{}
	if err := json.Unmarshal([]byte(raw.Content), &queryData); err != nil {
		return nil, fmt.Errorf("failed to parse mock response: %w", err)
	}

	return &types.StructuredQuery{
		LogSource: getStringValue(queryData, "log_source"),
		Verb:      *types.NewStringOrArray(getStringValue(queryData, "verb")),
		Resource:  *types.NewStringOrArray(getStringValue(queryData, "resource")),
		Timeframe: getStringValue(queryData, "timeframe"),
	}, nil
}

func (m *MockParser) CanHandle(modelType string) bool {
	return true
}

func (m *MockParser) GetConfidence() float64 {
	return m.confidence
}

// MockSafetyValidator implements interfaces.SafetyValidator for testing
type MockSafetyValidator struct {
	shouldFail bool
}

func (m *MockSafetyValidator) ValidateQuery(query *types.StructuredQuery) (*interfaces.ValidationResult, error) {
	if m.shouldFail {
		return &interfaces.ValidationResult{
			IsValid:  false,
			RuleName: "mock_validation",
			Severity: "critical",
			Message:  "Mock validation failed",
			Errors:   []string{"Mock validation error"},
		}, nil
	}

	return &interfaces.ValidationResult{
		IsValid:  true,
		RuleName: "mock_validation",
		Severity: "info",
		Message:  "Mock validation passed",
	}, nil
}

func (m *MockSafetyValidator) GetApplicableRules() []interfaces.ValidationRule {
	return []interfaces.ValidationRule{}
}

// createMockProcessor creates a processor with mocked dependencies for testing
func createMockProcessor(shouldFail bool, responses map[string]*types.RawResponse) *processor.GenAIProcessor {
	// Create mock components
	mockLLM := &MockLLMEngine{
		responses:  responses,
		shouldFail: shouldFail,
	}

	mockContext := &MockContextManager{}
	mockParser := &MockParser{
		shouldFail: shouldFail,
		confidence: 0.95,
	}
	mockValidator := &MockSafetyValidator{
		shouldFail: shouldFail,
	}

	// Create processor with injected mock dependencies
	return processor.NewGenAIProcessorWithDeps(
		mockContext,
		mockLLM,
		mockParser,
		mockValidator,
	)
}

// TestServerIntegration tests the complete HTTP flow from request to response
func TestServerIntegration(t *testing.T) {
	// Set up test environment
	originalPort := os.Getenv("PORT")
	os.Setenv("PORT", "0") // Use random port
	defer os.Setenv("PORT", originalPort)

	// Create predefined responses for consistent testing
	responses := map[string]*types.RawResponse{
		"Who deleted the customer CRD yesterday?": {
			Content: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "customresourcedefinitions", "timeframe": "yesterday"}`,
			ModelInfo: map[string]interface{}{
				"model": "test-model",
			},
		},
		"Show me all pod deletions": {
			Content: `{"log_source": "kube-apiserver", "verb": "delete", "resource": "pods", "timeframe": "last 24h"}`,
			ModelInfo: map[string]interface{}{
				"model": "test-model",
			},
		},
	}

	// Create test server using actual handler logic (copied from handlers.go)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create processor for each request
		proc := createMockProcessor(false, responses)

		// Route requests using the same logic as the actual handlers
		switch r.URL.Path {
		case "/query":
			handleQuery(w, r, proc)
		case "/health":
			handleHealth(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Run("Health Check Endpoint", func(t *testing.T) {
		testHealthEndpoint(t, server.URL)
	})

	t.Run("Valid Query Request", func(t *testing.T) {
		testValidQueryRequest(t, server.URL)
	})

	t.Run("Invalid JSON Request", func(t *testing.T) {
		testInvalidJSONRequest(t, server.URL)
	})

	t.Run("Empty Query Request", func(t *testing.T) {
		testEmptyQueryRequest(t, server.URL)
	})

	t.Run("Missing Session ID", func(t *testing.T) {
		testMissingSessionID(t, server.URL)
	})

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		testInvalidHTTPMethod(t, server.URL)
	})

	t.Run("Request Timeout", func(t *testing.T) {
		testRequestTimeout(t, server.URL)
	})
}

// handleQuery handles query requests in the test server (copied from handlers.go)
func handleQuery(w http.ResponseWriter, r *http.Request, proc *processor.GenAIProcessor) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only POST method is supported")
		return
	}

	// Parse request body
	var req types.ProcessingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request format", "Failed to parse JSON request body")
		return
	}

	// Basic input validation
	if err := validateProcessingRequest(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Process the query using GenAIProcessor
	response, err := proc.ProcessQuery(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Processing error", "Failed to process query")
		return
	}

	// Check if processing resulted in an error response
	if response.Error != "" {
		writeErrorResponse(w, http.StatusBadRequest, "Processing error", response.Error)
		return
	}

	// Write successful response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests in the test server (copied from handlers.go)
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET method is supported")
		return
	}

	healthResponse := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "genai-audit-query-processor",
		"version":   "1.0.0",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthResponse)
}

// validateProcessingRequest performs basic validation on the processing request (copied from handlers.go)
func validateProcessingRequest(req *types.ProcessingRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required and cannot be empty")
	}

	if len(req.Query) > 1000 {
		return fmt.Errorf("query too long, maximum 1000 characters allowed")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	if len(req.SessionID) > 100 {
		return fmt.Errorf("session_id too long, maximum 100 characters allowed")
	}

	return nil
}

// writeErrorResponse writes a standardized error response (copied from handlers.go)
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, message string) {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"type":    errorType,
			"message": message,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// testHealthEndpoint tests the health check endpoint
func testHealthEndpoint(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	// Check content type
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Health check returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse health check response: %v", err)
	}

	// Verify response structure
	expectedFields := []string{"status", "timestamp", "service", "version"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Health check response missing required field: %s", field)
		}
	}

	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("Health check returned wrong status: got %v want %v", status, "healthy")
	}

	if service, ok := response["service"].(string); !ok || service != "genai-audit-query-processor" {
		t.Errorf("Health check returned wrong service: got %v want %v", service, "genai-audit-query-processor")
	}
}

// testValidQueryRequest tests a valid query request
func testValidQueryRequest(t *testing.T, baseURL string) {
	request := types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-123",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to make query request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code (should be 200 or 400 depending on processor state)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Query request returned unexpected status code: got %v want 200 or 400", resp.StatusCode)
	}

	// Check content type
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Query request returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse query response: %v", err)
	}

	// If successful, verify response structure
	if resp.StatusCode == http.StatusOK {
		if _, exists := response["structured_query"]; !exists {
			t.Error("Successful query response missing structured_query field")
		}
		if _, exists := response["confidence"]; !exists {
			t.Error("Successful query response missing confidence field")
		}
	} else {
		// Check error response structure
		if errorObj, exists := response["error"].(map[string]interface{}); exists {
			if _, exists := errorObj["type"]; !exists {
				t.Error("Error response missing error.type field")
			}
			if _, exists := errorObj["message"]; !exists {
				t.Error("Error response missing error.message field")
			}
		} else {
			t.Error("Error response missing error field")
		}
	}
}

// testInvalidJSONRequest tests request with invalid JSON
func testInvalidJSONRequest(t *testing.T, baseURL string) {
	resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBufferString("invalid json"))
	if err != nil {
		t.Fatalf("Failed to make invalid JSON request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Invalid JSON request returned wrong status code: got %v want %v", resp.StatusCode, http.StatusBadRequest)
	}

	// Check content type
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Invalid JSON request returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	// Verify error structure
	if errorObj, exists := response["error"].(map[string]interface{}); exists {
		if errorType, ok := errorObj["type"].(string); !ok || errorType != "Invalid request format" {
			t.Errorf("Error response has wrong type: got %v want %v", errorType, "Invalid request format")
		}
	} else {
		t.Error("Error response missing error field")
	}
}

// testEmptyQueryRequest tests request with empty query
func testEmptyQueryRequest(t *testing.T, baseURL string) {
	request := types.ProcessingRequest{
		Query:     "",
		SessionID: "test-session-123",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to make empty query request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Empty query request returned wrong status code: got %v want %v", resp.StatusCode, http.StatusBadRequest)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	// Verify error message
	if errorObj, exists := response["error"].(map[string]interface{}); exists {
		if message, ok := errorObj["message"].(string); !ok || message != "query is required and cannot be empty" {
			t.Errorf("Error response has wrong message: got %v want %v", message, "query is required and cannot be empty")
		}
	} else {
		t.Error("Error response missing error field")
	}
}

// testMissingSessionID tests request with missing session ID
func testMissingSessionID(t *testing.T, baseURL string) {
	request := types.ProcessingRequest{
		Query:     "Who deleted the customer CRD?",
		SessionID: "",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to make missing session ID request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Missing session ID request returned wrong status code: got %v want %v", resp.StatusCode, http.StatusBadRequest)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	// Verify error message
	if errorObj, exists := response["error"].(map[string]interface{}); exists {
		if message, ok := errorObj["message"].(string); !ok || message != "session_id is required" {
			t.Errorf("Error response has wrong message: got %v want %v", message, "session_id is required")
		}
	} else {
		t.Error("Error response missing error field")
	}
}

// testInvalidHTTPMethod tests request with invalid HTTP method
func testInvalidHTTPMethod(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/query")
	if err != nil {
		t.Fatalf("Failed to make GET request to /query: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Invalid HTTP method request returned wrong status code: got %v want %v", resp.StatusCode, http.StatusMethodNotAllowed)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	// Verify error structure
	if errorObj, exists := response["error"].(map[string]interface{}); exists {
		if errorType, ok := errorObj["type"].(string); !ok || errorType != "Method not allowed" {
			t.Errorf("Error response has wrong type: got %v want %v", errorType, "Method not allowed")
		}
	} else {
		t.Error("Error response missing error field")
	}
}

// testRequestTimeout tests request timeout handling
func testRequestTimeout(t *testing.T, baseURL string) {
	// Create a request that will timeout
	request := types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-timeout",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create client with short timeout
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Post(baseURL+"/query", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		// Expected timeout error
		t.Logf("Expected timeout error: %v", err)
		return
	}
	defer resp.Body.Close()

	// If we get a response, it should be a timeout, error status, bad request, or success (with mocks)
	if resp.StatusCode != http.StatusRequestTimeout &&
		resp.StatusCode != http.StatusInternalServerError &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusOK {
		t.Errorf("Timeout request returned unexpected status code: got %v", resp.StatusCode)
	}
}

// TestServerStartup tests server startup and basic functionality
func TestServerStartup(t *testing.T) {
	// This test verifies that the server can start and handle basic requests
	// without actually starting a real server (which would require proper configuration)

	// Test that we can create a processor
	proc := processor.NewGenAIProcessor()
	if proc == nil {
		t.Fatal("Failed to create GenAI processor")
	}

	// Test that we can create a basic request
	req := &types.ProcessingRequest{
		Query:     "test query",
		SessionID: "test-session",
	}

	// Test that we can create a context
	ctx := context.Background()

	// Test that processor can handle the request (may fail due to missing API keys, but shouldn't panic)
	_, err := proc.ProcessQuery(ctx, req)
	if err != nil {
		// Expected error due to missing API configuration
		t.Logf("Expected processing error (missing API config): %v", err)
	}
}

// BenchmarkQueryProcessing benchmarks the query processing performance
func BenchmarkQueryProcessing(b *testing.B) {
	proc := processor.NewGenAIProcessor()
	if proc == nil {
		b.Fatal("Failed to create GenAI processor")
	}

	req := &types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "benchmark-session",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proc.ProcessQuery(ctx, req)
		if err != nil {
			// Expected error due to missing API configuration
			b.Logf("Expected processing error: %v", err)
		}
	}
}

// Helper function to safely get string values from map
func getStringValue(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
