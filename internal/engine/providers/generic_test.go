package providers

import (
	"context"
	"encoding/json"
	"genai-processing/pkg/types"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewGenericProvider(t *testing.T) {
	p := NewGenericProvider("key", "", "", nil, nil)
	if p.Endpoint == "" {
		t.Error("expected default endpoint")
	}
	if p.ModelName == "" {
		t.Error("expected default model name")
	}
	if p.Headers["Authorization"] == "" {
		t.Error("expected default Authorization header")
	}
}

func TestGenericProvider_GenerateResponse(t *testing.T) {
	// mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GenericChatResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{Role: "assistant", Content: `{"log_source":"kube-apiserver"}`},
					FinishReason: "stop",
				},
			},
			Model: "generic-model",
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{PromptTokens: 5, CompletionTokens: 7, TotalTokens: 12},
		})
	}))
	defer server.Close()

	p := NewGenericProvider("key", server.URL, "generic-model", nil, nil)

	req := &types.ModelRequest{
		Model:      "",
		Messages:   []interface{}{map[string]interface{}{"role": "user", "content": "Convert"}},
		Parameters: map[string]interface{}{"max_tokens": 5},
	}
	resp, err := p.GenerateResponse(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateResponse error: %v", err)
	}
	if resp == nil || resp.Content == "" {
		t.Fatalf("unexpected empty response")
	}
}

func TestGenericProvider_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(GenericAPIError{Error: struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code,omitempty"`
		}{Message: "Invalid API key", Type: "authentication_error"}})
	}))
	defer server.Close()

	p := NewGenericProvider("", server.URL, "model", nil, nil)
	_, err := p.GenerateResponse(context.Background(), &types.ModelRequest{Model: "model", Messages: []interface{}{map[string]interface{}{"role": "user", "content": "x"}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenericProvider_ValidateConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GenericChatResponse{Model: "generic-model"})
	}))
	defer server.Close()

	p := NewGenericProvider("key", server.URL, "generic-model", nil, nil)
	if err := p.ValidateConnection(); err != nil {
		t.Fatalf("validate error: %v", err)
	}
}
