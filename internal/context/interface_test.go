package context

import (
	"testing"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// TestInterfaceCompliance verifies that ContextManager implements the required interface
func TestInterfaceCompliance(t *testing.T) {
	var _ interfaces.ContextManager = NewContextManager()
}

// TestInterfaceMethods verifies that all interface methods can be called
func TestInterfaceMethods(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-interface-session"
	query := "Test query"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	// Test UpdateContext
	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Errorf("UpdateContext() failed: %v", err)
	}

	// Test ResolvePronouns
	resolved, err := cm.ResolvePronouns(query, sessionID)
	if err != nil {
		t.Errorf("ResolvePronouns() failed: %v", err)
	}
	if resolved != query {
		t.Errorf("Expected resolved query to match input, got %s", resolved)
	}

	// Test GetContext
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Errorf("GetContext() failed: %v", err)
	}
	if context == nil {
		t.Fatal("GetContext() returned nil context")
	}
	if context.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, context.SessionID)
	}
}
