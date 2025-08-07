package context

import (
	"fmt"
	"testing"
	"time"

	"genai-processing/pkg/types"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager()

	if cm == nil {
		t.Fatal("NewContextManager() returned nil")
	}

	// Verify it implements the interface
	var _ interface{} = cm
}

func TestUpdateContext_NewSession(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-1"
	query := "Who deleted the customer CRD yesterday?"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
	}

	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Errorf("UpdateContext() failed: %v", err)
	}

	// Verify session was created
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Errorf("GetContext() failed: %v", err)
	}

	if context.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, context.SessionID)
	}

	if context.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if context.LastActivity.IsZero() {
		t.Error("Expected LastActivity to be set")
	}
}

func TestUpdateContext_ExistingSession(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-2"
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
	}

	// Create initial session
	err := cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("Initial UpdateContext() failed: %v", err)
	}

	// Get initial context
	initialContext, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	initialActivity := initialContext.LastActivity

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update existing session
	query2 := "When did he do it?"
	response2 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Timeframe: "yesterday",
	}

	err = cm.UpdateContext(sessionID, query2, response2)
	if err != nil {
		t.Errorf("Second UpdateContext() failed: %v", err)
	}

	// Verify session was updated
	updatedContext, err := cm.GetContext(sessionID)
	if err != nil {
		t.Errorf("GetContext() failed: %v", err)
	}

	if updatedContext.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, updatedContext.SessionID)
	}

	if !updatedContext.LastActivity.After(initialActivity) {
		t.Error("Expected LastActivity to be updated")
	}

	if !updatedContext.CreatedAt.Equal(initialContext.CreatedAt) {
		t.Error("Expected CreatedAt to remain unchanged")
	}
}

func TestResolvePronouns_StubImplementation(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-3"
	query := "When did he do it?"

	// Test that ResolvePronouns returns input unchanged (stub implementation)
	resolved, err := cm.ResolvePronouns(query, sessionID)
	if err != nil {
		t.Errorf("ResolvePronouns() failed: %v", err)
	}

	if resolved != query {
		t.Errorf("Expected resolved query to be unchanged, got %s", resolved)
	}
}

func TestGetContext_ExistingSession(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-4"
	query := "Show me all failed authentication attempts"
	response := &types.StructuredQuery{
		LogSource:    "oauth-server",
		AuthDecision: "error",
	}

	// Create session
	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Get context
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

func TestGetContext_NonExistentSession(t *testing.T) {
	cm := NewContextManager()
	sessionID := "non-existent-session"

	// Try to get non-existent session
	context, err := cm.GetContext(sessionID)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	if context != nil {
		t.Error("Expected nil context for non-existent session")
	}

	expectedError := "session not found: " + sessionID
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetSessionCount(t *testing.T) {
	cm := NewContextManager()
	contextManager := cm.(*ContextManager) // Cast to concrete type for helper methods

	// Initially should have 0 sessions
	count := contextManager.GetSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 sessions initially, got %d", count)
	}

	// Create a session
	sessionID := "test-session-5"
	query := "Who created pods in production?"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("create"),
		Resource:  *types.NewStringOrArray("pods"),
	}

	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Should have 1 session
	count = contextManager.GetSessionCount()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// Create another session
	sessionID2 := "test-session-6"
	err = cm.UpdateContext(sessionID2, query, response)
	if err != nil {
		t.Fatalf("Second UpdateContext() failed: %v", err)
	}

	// Should have 2 sessions
	count = contextManager.GetSessionCount()
	if count != 2 {
		t.Errorf("Expected 2 sessions, got %d", count)
	}
}

func TestClearAllSessions(t *testing.T) {
	cm := NewContextManager()
	contextManager := cm.(*ContextManager) // Cast to concrete type for helper methods

	// Create multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	query := "Test query"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	for _, sessionID := range sessions {
		err := cm.UpdateContext(sessionID, query, response)
		if err != nil {
			t.Fatalf("UpdateContext() failed for %s: %v", sessionID, err)
		}
	}

	// Verify sessions exist
	count := contextManager.GetSessionCount()
	if count != len(sessions) {
		t.Errorf("Expected %d sessions, got %d", len(sessions), count)
	}

	// Clear all sessions
	contextManager.ClearAllSessions()

	// Verify all sessions are gone
	count = contextManager.GetSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 sessions after clear, got %d", count)
	}

	// Verify individual sessions are gone
	for _, sessionID := range sessions {
		_, err := cm.GetContext(sessionID)
		if err == nil {
			t.Errorf("Expected error for cleared session %s", sessionID)
		}
	}
}

func TestContextManager_ThreadSafety(t *testing.T) {
	cm := NewContextManager()
	contextManager := cm.(*ContextManager) // Cast to concrete type for helper methods
	done := make(chan bool)
	numGoroutines := 10

	// Start multiple goroutines that update contexts concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			sessionID := fmt.Sprintf("concurrent-session-%d", id)
			query := "Concurrent query"
			response := &types.StructuredQuery{
				LogSource: "kube-apiserver",
			}

			err := cm.UpdateContext(sessionID, query, response)
			if err != nil {
				t.Errorf("UpdateContext() failed in goroutine %d: %v", id, err)
			}

			// Also test GetContext concurrently
			_, err = cm.GetContext(sessionID)
			if err != nil {
				t.Errorf("GetContext() failed in goroutine %d: %v", id, err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all sessions were created
	count := contextManager.GetSessionCount()
	if count != numGoroutines {
		t.Errorf("Expected %d sessions, got %d", numGoroutines, count)
	}
}
