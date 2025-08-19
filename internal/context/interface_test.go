package context

import (
	"testing"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// TestInterfaceCompliance verifies that ContextManager implements the required interface
func TestInterfaceCompliance(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm := NewContextManagerWithConfig(config)
	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	var _ interfaces.ContextManager = cm
}

// TestInterfaceMethods verifies that all interface methods can be called
func TestInterfaceMethods(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm := NewContextManagerWithConfig(config)
	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

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

	// Test UpdateContextWithUser (enhanced interface method)
	userID := "test.user"
	err = cm.UpdateContextWithUser(sessionID, userID, "Another query", response)
	if err != nil {
		t.Errorf("UpdateContextWithUser() failed: %v", err)
	}

	// Verify user was set
	context, err = cm.GetContext(sessionID)
	if err != nil {
		t.Errorf("GetContext() failed after UpdateContextWithUser: %v", err)
	}
	if context.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, context.UserID)
	}
}
