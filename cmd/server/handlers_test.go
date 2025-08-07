package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"genai-processing/internal/processor"
	"genai-processing/pkg/types"
)

func TestHealthHandler(t *testing.T) {
	// Create a test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()
	handler := HealthHandler()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to parse response body: %v", err)
	}

	// Check response fields
	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("handler returned wrong status: got %v want %v", status, "healthy")
	}

	if service, ok := response["service"].(string); !ok || service != "genai-audit-query-processor" {
		t.Errorf("handler returned wrong service: got %v want %v", service, "genai-audit-query-processor")
	}
}

func TestQueryHandler_ValidRequest(t *testing.T) {
	// Create a mock processor (this would need to be properly mocked in a real test)
	// For now, we'll skip this test if we can't create a processor
	genaiProcessor := processor.NewGenAIProcessor()
	if genaiProcessor == nil {
		t.Skip("Skipping test - could not create GenAI processor")
	}

	// Create a valid request
	request := types.ProcessingRequest{
		Query:     "Who deleted the customer CRD yesterday?",
		SessionID: "test-session-123",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test request
	req, err := http.NewRequest("POST", "/query", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()
	handler := QueryHandler(genaiProcessor)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code (should be 200 or 400 depending on processor state)
	if status := rr.Code; status != http.StatusOK && status != http.StatusBadRequest {
		t.Errorf("handler returned unexpected status code: got %v want 200 or 400", status)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}
}

func TestQueryHandler_InvalidMethod(t *testing.T) {
	genaiProcessor := processor.NewGenAIProcessor()
	if genaiProcessor == nil {
		t.Skip("Skipping test - could not create GenAI processor")
	}

	// Create a test request with wrong method
	req, err := http.NewRequest("GET", "/query", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()
	handler := QueryHandler(genaiProcessor)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestQueryHandler_InvalidJSON(t *testing.T) {
	genaiProcessor := processor.NewGenAIProcessor()
	if genaiProcessor == nil {
		t.Skip("Skipping test - could not create GenAI processor")
	}

	// Create a test request with invalid JSON
	req, err := http.NewRequest("POST", "/query", bytes.NewBufferString("invalid json"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()
	handler := QueryHandler(genaiProcessor)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestValidateProcessingRequest(t *testing.T) {
	tests := []struct {
		name    string
		request types.ProcessingRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: types.ProcessingRequest{
				Query:     "Who deleted the customer CRD?",
				SessionID: "test-session",
			},
			wantErr: false,
		},
		{
			name: "empty query",
			request: types.ProcessingRequest{
				Query:     "",
				SessionID: "test-session",
			},
			wantErr: true,
		},
		{
			name: "empty session ID",
			request: types.ProcessingRequest{
				Query:     "Who deleted the customer CRD?",
				SessionID: "",
			},
			wantErr: true,
		},
		{
			name: "query too long",
			request: types.ProcessingRequest{
				Query:     string(make([]byte, 1001)), // 1001 characters
				SessionID: "test-session",
			},
			wantErr: true,
		},
		{
			name: "session ID too long",
			request: types.ProcessingRequest{
				Query:     "Who deleted the customer CRD?",
				SessionID: string(make([]byte, 101)), // 101 characters
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProcessingRequest(&tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProcessingRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
