package adapters

import (
	"genai-processing/pkg/types"
	"strings"
	"testing"
)

func TestNewGenericInputAdapter(t *testing.T) {
	a := NewGenericInputAdapter("test-key")
	if a.APIKey != "test-key" {
		t.Errorf("APIKey = %s", a.APIKey)
	}
	if a.ModelName == "" {
		t.Error("ModelName should have default")
	}
	if a.MaxTokens != 4000 {
		t.Errorf("MaxTokens = %d", a.MaxTokens)
	}
	if a.Temperature != 0.1 {
		t.Errorf("Temperature = %f", a.Temperature)
	}
}

func TestGenericInputAdapter_AdaptAndValidate(t *testing.T) {
	a := NewGenericInputAdapter("key")

	// Valid
	req := &types.InternalRequest{ProcessingRequest: types.ProcessingRequest{Query: "Show pod deletes"}}
	mr, err := a.AdaptRequest(req)
	if err != nil {
		t.Fatalf("adapt error: %v", err)
	}
	if mr.Model != a.ModelName {
		t.Errorf("model mismatch: %s", mr.Model)
	}
	if len(mr.Messages) == 0 {
		t.Error("expected messages")
	}
	if err := a.ValidateRequest(mr); err != nil {
		t.Errorf("validate error: %v", err)
	}

	// Nil request
	if _, err := a.AdaptRequest(nil); err == nil {
		t.Error("expected error for nil req")
	}
}

func TestGenericInputAdapter_FormatPrompt(t *testing.T) {
	a := NewGenericInputAdapter("k")
	// valid
	s, err := a.FormatPrompt("Find oauth errors", nil)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if !strings.Contains(s, "Convert this query to JSON:") {
		t.Error("missing instruction")
	}

	// with examples
	s, err = a.FormatPrompt("Find pods", []types.Example{{Input: "Who deleted pods?", Output: `{"log_source":"kube-apiserver"}`}})
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if !strings.Contains(s, "Examples:") {
		t.Error("missing examples")
	}

	// empty
	if _, err := a.FormatPrompt(" ", nil); err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestGenericInputAdapter_GetAPIParameters(t *testing.T) {
	a := NewGenericInputAdapter("token")
	p := a.GetAPIParameters()
	if p["api_key"] != "token" {
		t.Error("api_key mismatch")
	}
	if p["provider"] != "generic" {
		t.Error("provider mismatch")
	}
	headers, ok := p["headers"].(map[string]string)
	if !ok {
		t.Fatal("headers type mismatch")
	}
	if headers["Authorization"] == "" {
		t.Error("missing Authorization header")
	}
}

func TestGenericInputAdapter_ValidateRequest_Bounds(t *testing.T) {
	a := NewGenericInputAdapter("k")
	// invalid max tokens
	mr := &types.ModelRequest{Model: a.ModelName, Messages: []interface{}{map[string]interface{}{"role": "user", "content": "x"}}, Parameters: map[string]interface{}{"max_tokens": 50000}}
	if err := a.ValidateRequest(mr); err == nil {
		t.Error("expected error for max_tokens")
	}
	// invalid temp
	mr.Parameters = map[string]interface{}{"temperature": 9.9}
	if err := a.ValidateRequest(mr); err == nil {
		t.Error("expected error for temperature")
	}
}

// use strings.Contains in this file
