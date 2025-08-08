package extractors

import (
	"genai-processing/pkg/types"
	"testing"
)

func TestGenericExtractor_ParseResponse_JSONDirect(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: `{"log_source":"kube-apiserver","verb":"get"}`}
	q, err := e.ParseResponse(raw, "any")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if q.LogSource == "" {
		t.Error("expected log_source")
	}
	if e.GetConfidence() <= 0 {
		t.Error("expected confidence > 0")
	}
}

func TestGenericExtractor_ParseResponse_MarkdownFence(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: "```json\n{\"log_source\":\"kube-apiserver\"}\n```"}
	q, err := e.ParseResponse(raw, "any")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if q.LogSource != "kube-apiserver" {
		t.Error("expected parsed log_source")
	}
}

func TestGenericExtractor_ParseResponse_MissingJSON(t *testing.T) {
	e := NewGenericExtractor()
	raw := &types.RawResponse{Content: "no json here"}
	if _, err := e.ParseResponse(raw, "any"); err == nil {
		t.Fatal("expected error")
	}
}
