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

	// Verify conversation history was added
	if len(context.ConversationHistory) != 1 {
		t.Errorf("Expected 1 conversation entry, got %d", len(context.ConversationHistory))
	}

	entry := context.ConversationHistory[0]
	if entry.Query != query {
		t.Errorf("Expected query %s, got %s", query, entry.Query)
	}

	if entry.Response != response {
		t.Errorf("Expected response %v, got %v", response, entry.Response)
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

	// Verify conversation history was updated
	if len(updatedContext.ConversationHistory) != 2 {
		t.Errorf("Expected 2 conversation entries, got %d", len(updatedContext.ConversationHistory))
	}
}

func TestResolvePronouns_UserPronouns(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-pronouns"

	// First query to establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
		Timeframe: "yesterday",
	}

	err := cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Test pronoun resolution
	testCases := []struct {
		input    string
		expected string
	}{
		{"When did he do it?", "When did john.doe do customresourcedefinitions?"},
		{"What did she delete?", "What did john.doe delete?"},
		{"Show me actions by that user", "Show me actions by john.doe"},
		{"What did the user do?", "What did john.doe do?"},
		{"Show me what this user did", "Show me what john.doe did"},
	}

	for _, tc := range testCases {
		resolved, err := cm.ResolvePronouns(tc.input, sessionID)
		if err != nil {
			t.Errorf("ResolvePronouns() failed for '%s': %v", tc.input, err)
			continue
		}

		if resolved != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, resolved)
		}
	}
}

func TestResolvePronouns_ResourceReferences(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-resources"

	// First query to establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource:           "kube-apiserver",
		Verb:                *types.NewStringOrArray("delete"),
		Resource:            *types.NewStringOrArray("customresourcedefinitions"),
		ResourceNamePattern: "customer",
		User:                *types.NewStringOrArray("john.doe"),
	}

	err := cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Test resource reference resolution
	testCases := []struct {
		input    string
		expected string
	}{
		{"When was it deleted?", "When was customresourcedefinitions deleted?"},
		{"Show me that resource", "Show me customresourcedefinitions"},
		{"What happened to the resource?", "What happened to customresourcedefinitions?"},
		{"Show me that CRD", "Show me customer"},
		{"What happened to the CRD?", "What happened to customer?"},
	}

	for _, tc := range testCases {
		resolved, err := cm.ResolvePronouns(tc.input, sessionID)
		if err != nil {
			t.Errorf("ResolvePronouns() failed for '%s': %v", tc.input, err)
			continue
		}

		if resolved != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, resolved)
		}
	}
}

func TestResolvePronouns_TimeReferences(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-time"

	// First query to establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
		Timeframe: "yesterday",
	}

	err := cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Test time reference resolution
	testCases := []struct {
		input    string
		expected string
	}{
		{"What happened around that time?", "What happened yesterday?"},
		{"Show me events at that time", "Show me events yesterday"},
		{"What did he do then?", "What did john.doe do yesterday?"},
	}

	for _, tc := range testCases {
		resolved, err := cm.ResolvePronouns(tc.input, sessionID)
		if err != nil {
			t.Errorf("ResolvePronouns() failed for '%s': %v", tc.input, err)
			continue
		}

		if resolved != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, resolved)
		}
	}
}

func TestResolvePronouns_ActionReferences(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-actions"

	// First query to establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
	}

	err := cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Test action reference resolution
	testCases := []struct {
		input    string
		expected string
	}{
		{"When did he do that action?", "When did john.doe do delete?"},
		{"Show me the action details", "Show me delete details"},
		{"What action did he perform?", "What action did john.doe perform?"},
	}

	for _, tc := range testCases {
		resolved, err := cm.ResolvePronouns(tc.input, sessionID)
		if err != nil {
			t.Errorf("ResolvePronouns() failed for '%s': %v", tc.input, err)
			continue
		}

		if resolved != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, resolved)
		}
	}
}

func TestResolvePronouns_NoContext(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-no-context"

	// Test pronoun resolution without any context
	query := "When did he do it?"
	resolved, err := cm.ResolvePronouns(query, sessionID)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	if resolved != query {
		t.Errorf("Expected query to remain unchanged, got %s", resolved)
	}
}

func TestResolvePronouns_ComplexConversation(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-complex"

	// Simulate a complex conversation
	conversation := []struct {
		query    string
		response *types.StructuredQuery
	}{
		{
			"Who deleted the customer CRD yesterday?",
			&types.StructuredQuery{
				LogSource:           "kube-apiserver",
				Verb:                *types.NewStringOrArray("delete"),
				Resource:            *types.NewStringOrArray("customresourcedefinitions"),
				ResourceNamePattern: "customer",
				User:                *types.NewStringOrArray("john.doe"),
				Timeframe:           "yesterday",
			},
		},
		{
			"When did he do it?",
			&types.StructuredQuery{
				LogSource: "kube-apiserver",
				User:      *types.NewStringOrArray("john.doe"),
				Timeframe: "yesterday",
			},
		},
		{
			"What other resources did he modify?",
			&types.StructuredQuery{
				LogSource: "kube-apiserver",
				User:      *types.NewStringOrArray("john.doe"),
				Verb:      *types.NewStringOrArray([]string{"update", "patch"}),
			},
		},
	}

	// Update context with conversation
	for i, conv := range conversation {
		err := cm.UpdateContext(sessionID, conv.query, conv.response)
		if err != nil {
			t.Fatalf("UpdateContext() failed for conversation %d: %v", i, err)
		}
	}

	// Test pronoun resolution with complex context
	testCases := []struct {
		input    string
		expected string
	}{
		{"Show me what he did to that CRD", "Show me what john.doe did to customer"},
		{"When did he modify it?", "When did john.doe modify customresourcedefinitions?"},
		{"What actions did he perform around that time?", "What actions did john.doe perform yesterday?"},
	}

	for _, tc := range testCases {
		resolved, err := cm.ResolvePronouns(tc.input, sessionID)
		if err != nil {
			t.Errorf("ResolvePronouns() failed for '%s': %v", tc.input, err)
			continue
		}

		if resolved != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, resolved)
		}
	}

	// Verify conversation history
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	if len(context.ConversationHistory) != len(conversation) {
		t.Errorf("Expected %d conversation entries, got %d", len(conversation), len(context.ConversationHistory))
	}

	// Verify resolved references
	if len(context.ResolvedReferences) == 0 {
		t.Error("Expected resolved references to be populated")
	}
}

func TestContextEnrichment(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-enrichment"

	query := "Who deleted the customer CRD yesterday?"
	response := &types.StructuredQuery{
		LogSource:           "kube-apiserver",
		Verb:                *types.NewStringOrArray("delete"),
		Resource:            *types.NewStringOrArray("customresourcedefinitions"),
		ResourceNamePattern: "customer",
		User:                *types.NewStringOrArray("john.doe"),
		Timeframe:           "yesterday",
		Limit:               20,
	}

	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Verify context enrichment
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	// Check query patterns
	patterns, exists := context.ContextEnrichment["query_patterns"]
	if !exists {
		t.Error("Expected query_patterns in context enrichment")
	}

	patternsMap, ok := patterns.(map[string]interface{})
	if !ok {
		t.Error("Expected query_patterns to be a map")
	}

	if patternsMap["question_type"] != "who" {
		t.Errorf("Expected question_type 'who', got %v", patternsMap["question_type"])
	}

	if patternsMap["action_type"] != "deleted" {
		t.Errorf("Expected action_type 'deleted', got %v", patternsMap["action_type"])
	}

	// Check response summary
	summary, exists := context.ContextEnrichment["last_response_summary"]
	if !exists {
		t.Error("Expected last_response_summary in context enrichment")
	}

	summaryMap, ok := summary.(map[string]interface{})
	if !ok {
		t.Error("Expected last_response_summary to be a map")
	}

	if summaryMap["log_source"] != "kube-apiserver" {
		t.Errorf("Expected log_source 'kube-apiserver', got %v", summaryMap["log_source"])
	}

	// Check conversation flow
	flow, exists := context.ContextEnrichment["conversation_flow"]
	if !exists {
		t.Error("Expected conversation_flow in context enrichment")
	}

	flowMap, ok := flow.(map[string]interface{})
	if !ok {
		t.Error("Expected conversation_flow to be a map")
	}

	if flowMap["total_interactions"] != 1 {
		t.Errorf("Expected total_interactions 1, got %v", flowMap["total_interactions"])
	}

	if !flowMap["user_focus"].(bool) {
		t.Error("Expected user_focus to be true")
	}
}

func TestSessionExpiration(t *testing.T) {
	cm := NewContextManager()
	contextManager := cm.(*ContextManager) // Cast to concrete type for helper methods

	sessionID := "test-session-expiration"
	query := "Who deleted the customer CRD yesterday?"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
	}

	// Create session
	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Get context and verify it's not expired
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	if context.IsExpired() {
		t.Error("Expected session to not be expired immediately after creation")
	}

	// Verify session count
	count := contextManager.GetSessionCount()
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
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

	// Verify conversation history
	if len(context.ConversationHistory) != 1 {
		t.Errorf("Expected 1 conversation entry, got %d", len(context.ConversationHistory))
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

			// Test pronoun resolution concurrently
			_, err = cm.ResolvePronouns("When did he do it?", sessionID)
			if err != nil {
				// This is expected to fail since there's no user context
				// but it shouldn't cause a panic
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

func TestResolvedReferences(t *testing.T) {
	cm := NewContextManager()
	sessionID := "test-session-references"

	// Create a session with various references
	query := "Who deleted the customer CRD yesterday?"
	response := &types.StructuredQuery{
		LogSource:           "kube-apiserver",
		Verb:                *types.NewStringOrArray("delete"),
		Resource:            *types.NewStringOrArray("customresourcedefinitions"),
		ResourceNamePattern: "customer",
		User:                *types.NewStringOrArray("john.doe"),
		Timeframe:           "yesterday",
		Namespace:           *types.NewStringOrArray("production"),
	}

	err := cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Verify resolved references
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	// Check user reference
	userRef, exists := context.GetResolvedReference("last_user")
	if !exists {
		t.Error("Expected last_user reference to exist")
	}
	if userRef.Value != "john.doe" {
		t.Errorf("Expected last_user 'john.doe', got %s", userRef.Value)
	}
	if userRef.Type != "user" {
		t.Errorf("Expected last_user type 'user', got %s", userRef.Type)
	}

	// Check resource reference
	resourceRef, exists := context.GetResolvedReference("last_resource")
	if !exists {
		t.Error("Expected last_resource reference to exist")
	}
	if resourceRef.Value != "customresourcedefinitions" {
		t.Errorf("Expected last_resource 'customresourcedefinitions', got %s", resourceRef.Value)
	}

	// Check resource name reference
	resourceNameRef, exists := context.GetResolvedReference("last_resource_name")
	if !exists {
		t.Error("Expected last_resource_name reference to exist")
	}
	if resourceNameRef.Value != "customer" {
		t.Errorf("Expected last_resource_name 'customer', got %s", resourceNameRef.Value)
	}

	// Check timeframe reference
	timeRef, exists := context.GetResolvedReference("last_timeframe")
	if !exists {
		t.Error("Expected last_timeframe reference to exist")
	}
	if timeRef.Value != "yesterday" {
		t.Errorf("Expected last_timeframe 'yesterday', got %s", timeRef.Value)
	}

	// Check action reference
	actionRef, exists := context.GetResolvedReference("last_action")
	if !exists {
		t.Error("Expected last_action reference to exist")
	}
	if actionRef.Value != "delete" {
		t.Errorf("Expected last_action 'delete', got %s", actionRef.Value)
	}
}
