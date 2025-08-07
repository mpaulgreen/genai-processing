package types

import (
	"encoding/json"
	"testing"
)

func TestProcessingRequest_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  ProcessingRequest
		expected string
		wantErr  bool
	}{
		{
			name: "valid request with all fields",
			request: ProcessingRequest{
				Query:     "Show me audit logs from last week",
				SessionID: "session-123",
				ModelType: "gpt-4",
			},
			expected: `{"query":"Show me audit logs from last week","session_id":"session-123","model_type":"gpt-4"}`,
			wantErr:  false,
		},
		{
			name: "valid request without model type",
			request: ProcessingRequest{
				Query:     "Find failed login attempts",
				SessionID: "session-456",
			},
			expected: `{"query":"Find failed login attempts","session_id":"session-456"}`,
			wantErr:  false,
		},
		{
			name: "request with empty strings",
			request: ProcessingRequest{
				Query:     "",
				SessionID: "",
				ModelType: "",
			},
			expected: `{"query":"","session_id":""}`,
			wantErr:  false,
		},
		{
			name: "request with special characters",
			request: ProcessingRequest{
				Query:     "Find logs with \"quotes\" and 'apostrophes'",
				SessionID: "session-789",
				ModelType: "claude-3",
			},
			expected: `{"query":"Find logs with \"quotes\" and 'apostrophes'","session_id":"session-789","model_type":"claude-3"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessingRequest.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("ProcessingRequest.MarshalJSON() = %v, want %v", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled ProcessingRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("ProcessingRequest.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify round-trip
			if unmarshaled.Query != tt.request.Query {
				t.Errorf("Query mismatch: got %v, want %v", unmarshaled.Query, tt.request.Query)
			}
			if unmarshaled.SessionID != tt.request.SessionID {
				t.Errorf("SessionID mismatch: got %v, want %v", unmarshaled.SessionID, tt.request.SessionID)
			}
			if unmarshaled.ModelType != tt.request.ModelType {
				t.Errorf("ModelType mismatch: got %v, want %v", unmarshaled.ModelType, tt.request.ModelType)
			}
		})
	}
}

func TestProcessingResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response ProcessingResponse
		expected string
		wantErr  bool
	}{
		{
			name: "valid response with all fields",
			response: ProcessingResponse{
				StructuredQuery: map[string]interface{}{
					"type": "audit_log",
					"filters": map[string]interface{}{
						"time_range": "last_week",
					},
				},
				Confidence: 0.95,
				ValidationInfo: map[string]interface{}{
					"warnings": []string{"time_range is approximate"},
				},
			},
			expected: `{"structured_query":{"filters":{"time_range":"last_week"},"type":"audit_log"},"confidence":0.95,"validation_info":{"warnings":["time_range is approximate"]}}`,
			wantErr:  false,
		},
		{
			name: "response with error",
			response: ProcessingResponse{
				StructuredQuery: nil,
				Confidence:      0.0,
				ValidationInfo:  nil,
				Error:           "Failed to parse query",
			},
			expected: `{"structured_query":null,"confidence":0,"validation_info":null,"error":"Failed to parse query"}`,
			wantErr:  false,
		},
		{
			name: "response with minimal fields",
			response: ProcessingResponse{
				StructuredQuery: "simple_string_query",
				Confidence:      0.5,
				ValidationInfo:  nil,
			},
			expected: `{"structured_query":"simple_string_query","confidence":0.5,"validation_info":null}`,
			wantErr:  false,
		},
		{
			name: "response with array structured query",
			response: ProcessingResponse{
				StructuredQuery: []interface{}{
					"filter1",
					"filter2",
					map[string]interface{}{"key": "value"},
				},
				Confidence:     0.8,
				ValidationInfo: map[string]interface{}{},
			},
			expected: `{"structured_query":["filter1","filter2",{"key":"value"}],"confidence":0.8,"validation_info":{}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessingResponse.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("ProcessingResponse.MarshalJSON() = %v, want %v", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled ProcessingResponse
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("ProcessingResponse.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify round-trip for basic fields
			if unmarshaled.Confidence != tt.response.Confidence {
				t.Errorf("Confidence mismatch: got %v, want %v", unmarshaled.Confidence, tt.response.Confidence)
			}
			if unmarshaled.Error != tt.response.Error {
				t.Errorf("Error mismatch: got %v, want %v", unmarshaled.Error, tt.response.Error)
			}
		})
	}
}

func TestInternalRequest_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  InternalRequest
		expected string
		wantErr  bool
	}{
		{
			name: "valid internal request with all fields",
			request: InternalRequest{
				RequestID: "req-123",
				ProcessingRequest: ProcessingRequest{
					Query:     "Show audit logs",
					SessionID: "session-123",
					ModelType: "gpt-4",
				},
				ProcessingOptions: map[string]interface{}{
					"timeout": 30,
					"retries": 3,
				},
			},
			expected: `{"request_id":"req-123","processing_request":{"query":"Show audit logs","session_id":"session-123","model_type":"gpt-4"},"processing_options":{"retries":3,"timeout":30}}`,
			wantErr:  false,
		},
		{
			name: "internal request without options",
			request: InternalRequest{
				RequestID: "req-456",
				ProcessingRequest: ProcessingRequest{
					Query:     "Find errors",
					SessionID: "session-456",
				},
			},
			expected: `{"request_id":"req-456","processing_request":{"query":"Find errors","session_id":"session-456"}}`,
			wantErr:  false,
		},
		{
			name: "internal request with empty options",
			request: InternalRequest{
				RequestID: "req-789",
				ProcessingRequest: ProcessingRequest{
					Query:     "Test query",
					SessionID: "session-789",
				},
				ProcessingOptions: map[string]interface{}{},
			},
			expected: `{"request_id":"req-789","processing_request":{"query":"Test query","session_id":"session-789"}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("InternalRequest.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("InternalRequest.MarshalJSON() = %v, want %v", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled InternalRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("InternalRequest.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify round-trip
			if unmarshaled.RequestID != tt.request.RequestID {
				t.Errorf("RequestID mismatch: got %v, want %v", unmarshaled.RequestID, tt.request.RequestID)
			}
			if unmarshaled.ProcessingRequest.Query != tt.request.ProcessingRequest.Query {
				t.Errorf("ProcessingRequest.Query mismatch: got %v, want %v", unmarshaled.ProcessingRequest.Query, tt.request.ProcessingRequest.Query)
			}
		})
	}
}

func TestModelRequest_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  ModelRequest
		expected string
		wantErr  bool
	}{
		{
			name: "valid model request with all fields",
			request: ModelRequest{
				Model: "gpt-4",
				Messages: []interface{}{
					map[string]interface{}{
						"role":    "system",
						"content": "You are a helpful assistant",
					},
					map[string]interface{}{
						"role":    "user",
						"content": "Parse this query",
					},
				},
				Parameters: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  1000,
				},
			},
			expected: `{"model":"gpt-4","messages":[{"content":"You are a helpful assistant","role":"system"},{"content":"Parse this query","role":"user"}],"parameters":{"max_tokens":1000,"temperature":0.7}}`,
			wantErr:  false,
		},
		{
			name: "model request without parameters",
			request: ModelRequest{
				Model: "claude-3",
				Messages: []interface{}{
					"Simple message",
					map[string]interface{}{
						"role":    "user",
						"content": "Process this",
					},
				},
			},
			expected: `{"model":"claude-3","messages":["Simple message",{"content":"Process this","role":"user"}]}`,
			wantErr:  false,
		},
		{
			name: "model request with empty messages",
			request: ModelRequest{
				Model:      "test-model",
				Messages:   []interface{}{},
				Parameters: map[string]interface{}{},
			},
			expected: `{"model":"test-model","messages":[]}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelRequest.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("ModelRequest.MarshalJSON() = %v, want %v", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled ModelRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("ModelRequest.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify round-trip
			if unmarshaled.Model != tt.request.Model {
				t.Errorf("Model mismatch: got %v, want %v", unmarshaled.Model, tt.request.Model)
			}
			if len(unmarshaled.Messages) != len(tt.request.Messages) {
				t.Errorf("Messages length mismatch: got %v, want %v", len(unmarshaled.Messages), len(tt.request.Messages))
			}
		})
	}
}

func TestRawResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response RawResponse
		expected string
		wantErr  bool
	}{
		{
			name: "valid raw response with all fields",
			response: RawResponse{
				Content: "{\"query\": \"audit logs\", \"filters\": {\"time\": \"last_week\"}}",
				ModelInfo: map[string]interface{}{
					"model":    "gpt-4",
					"version":  "4.0",
					"provider": "openai",
				},
				Metadata: map[string]interface{}{
					"tokens_used": 150,
					"latency_ms":  1200,
				},
			},
			expected: `{"content":"{\"query\": \"audit logs\", \"filters\": {\"time\": \"last_week\"}}","model_info":{"model":"gpt-4","provider":"openai","version":"4.0"},"metadata":{"latency_ms":1200,"tokens_used":150}}`,
			wantErr:  false,
		},
		{
			name: "raw response with error",
			response: RawResponse{
				Content: "",
				ModelInfo: map[string]interface{}{
					"model": "gpt-4",
				},
				Metadata: nil,
				Error:    "Model API timeout",
			},
			expected: `{"content":"","model_info":{"model":"gpt-4"},"error":"Model API timeout"}`,
			wantErr:  false,
		},
		{
			name: "raw response with minimal fields",
			response: RawResponse{
				Content:   "Simple response content",
				ModelInfo: nil,
				Metadata:  nil,
			},
			expected: `{"content":"Simple response content"}`,
			wantErr:  false,
		},
		{
			name: "raw response with empty maps",
			response: RawResponse{
				Content:   "Test content",
				ModelInfo: map[string]interface{}{},
				Metadata:  map[string]interface{}{},
			},
			expected: `{"content":"Test content"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawResponse.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("RawResponse.MarshalJSON() = %v, want %v", string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled RawResponse
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("RawResponse.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify round-trip
			if unmarshaled.Content != tt.response.Content {
				t.Errorf("Content mismatch: got %v, want %v", unmarshaled.Content, tt.response.Content)
			}
			if unmarshaled.Error != tt.response.Error {
				t.Errorf("Error mismatch: got %v, want %v", unmarshaled.Error, tt.response.Error)
			}
		})
	}
}

func TestStructFieldValidation(t *testing.T) {
	tests := []struct {
		name    string
		testFn  func() error
		wantErr bool
	}{
		{
			name: "ProcessingRequest with empty query",
			testFn: func() error {
				req := ProcessingRequest{
					Query:     "",
					SessionID: "session-123",
				}
				_, err := json.Marshal(req)
				return err
			},
			wantErr: false, // Empty query should be valid JSON
		},
		{
			name: "ProcessingResponse with invalid confidence range",
			testFn: func() error {
				resp := ProcessingResponse{
					StructuredQuery: "test",
					Confidence:      1.5, // Should be 0.0 to 1.0
				}
				_, err := json.Marshal(resp)
				return err
			},
			wantErr: false, // JSON marshaling should succeed even with invalid confidence
		},
		{
			name: "InternalRequest with empty request ID",
			testFn: func() error {
				req := InternalRequest{
					RequestID: "",
					ProcessingRequest: ProcessingRequest{
						Query:     "test",
						SessionID: "session-123",
					},
				}
				_, err := json.Marshal(req)
				return err
			},
			wantErr: false, // Empty request ID should be valid JSON
		},
		{
			name: "ModelRequest with empty model",
			testFn: func() error {
				req := ModelRequest{
					Model:    "",
					Messages: []interface{}{"test"},
				}
				_, err := json.Marshal(req)
				return err
			},
			wantErr: false, // Empty model should be valid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFn()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validation test failed: error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		testFn  func() error
		wantErr bool
	}{
		{
			name: "ProcessingRequest with very long query",
			testFn: func() error {
				longQuery := string(make([]byte, 10000)) // 10KB query
				req := ProcessingRequest{
					Query:     longQuery,
					SessionID: "session-123",
				}
				_, err := json.Marshal(req)
				return err
			},
			wantErr: false,
		},
		{
			name: "ProcessingResponse with nested structures",
			testFn: func() error {
				resp := ProcessingResponse{
					StructuredQuery: map[string]interface{}{
						"deep": map[string]interface{}{
							"nested": map[string]interface{}{
								"structure": []interface{}{
									map[string]interface{}{
										"key": "value",
									},
								},
							},
						},
					},
					Confidence: 0.8,
				}
				_, err := json.Marshal(resp)
				return err
			},
			wantErr: false,
		},
		{
			name: "ModelRequest with mixed message types",
			testFn: func() error {
				req := ModelRequest{
					Model: "gpt-4",
					Messages: []interface{}{
						"string message",
						123,
						map[string]interface{}{"key": "value"},
						[]interface{}{"array", "message"},
					},
				}
				_, err := json.Marshal(req)
				return err
			},
			wantErr: false,
		},
		{
			name: "RawResponse with special characters in content",
			testFn: func() error {
				resp := RawResponse{
					Content: "Content with \n newlines \t tabs \r carriage returns and \"quotes\"",
					ModelInfo: map[string]interface{}{
						"special": "value with \u0000 null bytes",
					},
				}
				_, err := json.Marshal(resp)
				return err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFn()
			if (err != nil) != tt.wantErr {
				t.Errorf("Edge case test failed: error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkProcessingRequest_Marshal(b *testing.B) {
	req := ProcessingRequest{
		Query:     "Show me audit logs from last week with high severity",
		SessionID: "session-benchmark-123",
		ModelType: "gpt-4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessingResponse_Marshal(b *testing.B) {
	resp := ProcessingResponse{
		StructuredQuery: map[string]interface{}{
			"type": "audit_log",
			"filters": map[string]interface{}{
				"time_range": "last_week",
				"severity":   "high",
			},
		},
		Confidence: 0.95,
		ValidationInfo: map[string]interface{}{
			"warnings": []string{"time_range is approximate"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}
