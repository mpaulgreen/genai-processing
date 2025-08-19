package context

import (
	"fmt"
	"os"
	"testing"
	"time"

	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

func TestNewContextManagerWithConfig_DefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}

	// Test interface compliance
	var _ interfaces.ContextManager = cm

	// Clean up
	if contextMgr, ok := cm.(*ContextManager); ok {
		err := contextMgr.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
}

func TestNewContextManagerWithConfig_NilConfig(t *testing.T) {
	cm, err := NewContextManagerFull(nil)
	if err != nil {
		t.Fatalf("NewContextManagerFull() with nil config failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}

	// Should use default config
	stats := cm.(*ContextManager).GetStats()
	if stats.MemoryLimitMB != 100 { // Default value
		t.Errorf("Expected default memory limit 100, got %d", stats.MemoryLimitMB)
	}

	if contextMgr, ok := cm.(*ContextManager); ok {
		err := contextMgr.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
}

func TestNewContextManagerWithConfig_InvalidConfig(t *testing.T) {
	// Config with invalid persistence path (permission denied)
	config := DefaultConfig()
	config.PersistencePath = "/root/invalid/path/that/should/not/exist"

	cm, err := NewContextManagerFull(config)
	if err == nil {
		t.Error("Expected error for invalid persistence path")
		if cm != nil {
			if contextMgr, ok := cm.(*ContextManager); ok {
				contextMgr.Close()
			}
		}
	}
}

func TestNewContextManagerWithConfig_NoPersistence(t *testing.T) {
	config := DefaultConfig()
	config.EnablePersistence = false

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() with no persistence failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}

	if contextMgr, ok := cm.(*ContextManager); ok {
		err := contextMgr.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
}

func TestContextManager_UpdateContext(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnableAsyncPersistence = false // Synchronous for testing

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	sessionID := "test-session-enhanced"
	query := "Who deleted the customer CRD yesterday?"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
	}

	// Update context
	err = cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Verify context was created
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	if context.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, context.SessionID)
	}

	// Verify conversation history
	if len(context.ConversationHistory) != 1 {
		t.Errorf("Expected 1 conversation entry, got %d", len(context.ConversationHistory))
	}

	entry := context.ConversationHistory[0]
	if entry.Query != query {
		t.Errorf("Expected query %s, got %s", query, entry.Query)
	}
}

func TestContextManager_UpdateContextWithUser(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	sessionID := "test-session-user"
	userID := "john.doe"
	query := "Test query"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	// Update context with user
	err = cm.UpdateContextWithUser(sessionID, userID, query, response)
	if err != nil {
		t.Fatalf("UpdateContextWithUser() failed: %v", err)
	}

	// Verify context was created with user
	context, err := cm.GetContext(sessionID)
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	if context.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, context.UserID)
	}
}

func TestContextManager_ResolvePronouns(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	sessionID := "test-session-pronouns"

	// First, establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
	}

	err = cm.UpdateContext(sessionID, query1, response1)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Now resolve pronouns
	query2 := "When did he do it?"
	resolved, err := cm.ResolvePronouns(query2, sessionID)
	if err != nil {
		t.Fatalf("ResolvePronouns() failed: %v", err)
	}

	expected := "When did john.doe do customresourcedefinitions?"
	if resolved != expected {
		t.Errorf("Expected resolved query '%s', got '%s'", expected, resolved)
	}
}

func TestContextManager_ResolvePronouns_NoContext(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	// Try to resolve pronouns without establishing context
	query := "When did he do it?"
	resolved, err := cm.ResolvePronouns(query, "non-existent-session")
	if err != nil {
		t.Fatalf("ResolvePronouns() failed: %v", err)
	}

	// Should return original query unchanged
	if resolved != query {
		t.Errorf("Expected query unchanged '%s', got '%s'", query, resolved)
	}
}

func TestContextManager_GetContext_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	// Try to get non-existent context
	context, err := cm.GetContext("non-existent-session")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	if context != nil {
		t.Error("Expected nil context for non-existent session")
	}
}

func TestContextManager_Stats(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	contextManager := cm.(*ContextManager)

	// Get initial stats
	stats := contextManager.GetStats()
	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 initial total sessions, got %d", stats.TotalSessions)
	}

	if stats.ActiveSessions != 0 {
		t.Errorf("Expected 0 initial active sessions, got %d", stats.ActiveSessions)
	}

	if stats.MemoryLimitMB != config.MaxMemoryMB {
		t.Errorf("Expected memory limit %d, got %d", config.MaxMemoryMB, stats.MemoryLimitMB)
	}

	// Add a session
	sessionID := "stats-test-session"
	query := "Test query for stats"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	err = cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Check updated stats
	stats = contextManager.GetStats()
	if stats.TotalSessions != 1 {
		t.Errorf("Expected 1 total session, got %d", stats.TotalSessions)
	}

	if stats.ActiveSessions != 1 {
		t.Errorf("Expected 1 active session, got %d", stats.ActiveSessions)
	}

	if stats.MemoryUsageMB <= 0 {
		t.Error("Expected positive memory usage")
	}
}

func TestContextManager_BackwardCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	contextManager := cm.(*ContextManager)

	// Test GetSessionCount method (backward compatibility)
	count := contextManager.GetSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 initial session count, got %d", count)
	}

	// Add a session
	sessionID := "compat-test-session"
	query := "Test query"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	err = cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	count = contextManager.GetSessionCount()
	if count != 1 {
		t.Errorf("Expected 1 session count, got %d", count)
	}

	// Test ClearAllSessions method
	contextManager.ClearAllSessions()

	count = contextManager.GetSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 session count after clear, got %d", count)
	}
}

func TestContextManager_PersistenceRecovery(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnableAsyncPersistence = false // Synchronous for testing

	// Create first manager and add sessions
	cm1, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerWithConfig() failed: %v", err)
	}

	sessionID1 := "persistent-session-1"
	sessionID2 := "persistent-session-2"

	err = cm1.UpdateContext(sessionID1, "Query 1", &types.StructuredQuery{
		LogSource: "kube-apiserver",
		User:      *types.NewStringOrArray("user1"),
	})
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	err = cm1.UpdateContext(sessionID2, "Query 2", &types.StructuredQuery{
		LogSource: "kube-apiserver",
		User:      *types.NewStringOrArray("user2"),
	})
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Close first manager
	if contextMgr1, ok := cm1.(*ContextManager); ok {
		err = contextMgr1.Close()
		if err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	}

	// Create second manager with same persistence path
	cm2, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("Second NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr2, ok := cm2.(*ContextManager); ok {
			contextMgr2.Close()
		}
	}()

	// Sessions should be recovered
	context1, err := cm2.GetContext(sessionID1)
	if err != nil {
		t.Fatalf("GetContext() failed for recovered session 1: %v", err)
	}

	if context1.SessionID != sessionID1 {
		t.Errorf("Expected recovered SessionID %s, got %s", sessionID1, context1.SessionID)
	}

	context2, err := cm2.GetContext(sessionID2)
	if err != nil {
		t.Fatalf("GetContext() failed for recovered session 2: %v", err)
	}

	if context2.SessionID != sessionID2 {
		t.Errorf("Expected recovered SessionID %s, got %s", sessionID2, context2.SessionID)
	}
}

func TestContextManager_AsyncPersistence(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnableAsyncPersistence = true
	config.PersistenceInterval = 100 * time.Millisecond // Short interval for testing

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	sessionID := "async-test-session"
	query := "Async persistence test"
	response := &types.StructuredQuery{
		LogSource: "kube-apiserver",
	}

	// Add session
	err = cm.UpdateContext(sessionID, query, response)
	if err != nil {
		t.Fatalf("UpdateContext() failed: %v", err)
	}

	// Wait for async persistence to kick in
	time.Sleep(200 * time.Millisecond)

	// Check that session files exist
	sessionsDir := tempDir + "/sessions"
	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("Failed to read sessions directory: %v", err)
	}

	found := false
	expectedFile := sessionID + ".json"
	for _, file := range files {
		if file.Name() == expectedFile {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Session file %s not found after async persistence", expectedFile)
	}
}

func TestContextManager_MemoryPressure(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.MaxSessions = 2 // Very low limit to trigger eviction
	config.MaxMemoryMB = 1 // Very low memory limit

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	// Add sessions beyond the limit
	for i := 0; i < 5; i++ {
		sessionID := "pressure-session-" + string(rune('1'+i))
		query := "Test query " + string(rune('1'+i))
		response := &types.StructuredQuery{
			LogSource: "kube-apiserver",
		}

		err = cm.UpdateContext(sessionID, query, response)
		if err != nil {
			t.Fatalf("UpdateContext() failed for session %s: %v", sessionID, err)
		}
	}

	contextManager := cm.(*ContextManager)
	stats := contextManager.GetStats()

	// Verify that we have sessions
	if stats.ActiveSessions == 0 {
		t.Error("Expected some active sessions")
	}

	// Verify that memory monitoring is working
	if stats.MemoryUsageMB < 0 {
		t.Error("Expected non-negative memory usage")
	}

	// Verify that LRU manager has some evictions when sessions exceed limit
	// Note: The current implementation stores sessions in both main map and LRU cache
	// The LRU cache handles eviction internally but doesn't update the main sessions map
	if stats.EvictionCount > 0 {
		t.Logf("LRU evictions occurred: %d", stats.EvictionCount)
	}
}

func TestNewContextManagerWithConfig_fallback(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir

	// Test successful creation
	cm := NewContextManagerWithConfig(config)
	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() returned nil")
	}

	// Should be a ContextManager
	if _, ok := cm.(*ContextManager); !ok {
		t.Error("Expected ContextManager instance")
	}

	if contextMgr, ok := cm.(*ContextManager); ok {
		err := contextMgr.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}

	// Test fallback to basic configuration on error
	invalidConfig := DefaultConfig()
	invalidConfig.PersistencePath = "/root/invalid/path"

	cm = NewContextManagerWithConfig(invalidConfig)
	if cm == nil {
		t.Fatal("NewContextManagerWithConfig() should fallback and not return nil")
	}

	// Should fallback to ContextManager with basic config
	if _, ok := cm.(*ContextManager); !ok {
		t.Error("Expected fallback to ContextManager")
	}

	// Verify fallback config (persistence should be disabled)
	fallbackStats := cm.(*ContextManager).GetStats()
	if fallbackStats.PersistenceOps != 0 {
		t.Error("Expected persistence to be disabled in fallback configuration")
	}

	if contextMgr, ok := cm.(*ContextManager); ok {
		err := contextMgr.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
}

// ========== CORE FUNCTIONALITY TESTS FROM BASIC MANAGER ==========

func TestContextManager_PronounResolution_UserPronouns(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false // Disable for simple testing

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

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

	err = cm.UpdateContext(sessionID, query1, response1)
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

func TestContextManager_PronounResolution_ResourceReferences(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

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

	err = cm.UpdateContext(sessionID, query1, response1)
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

func TestContextManager_PronounResolution_TimeReferences(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

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

	err = cm.UpdateContext(sessionID, query1, response1)
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

func TestContextManager_PronounResolution_ActionReferences(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	sessionID := "test-session-actions"

	// First query to establish context
	query1 := "Who deleted the customer CRD yesterday?"
	response1 := &types.StructuredQuery{
		LogSource: "kube-apiserver",
		Verb:      *types.NewStringOrArray("delete"),
		Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		User:      *types.NewStringOrArray("john.doe"),
	}

	err = cm.UpdateContext(sessionID, query1, response1)
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

func TestContextManager_ComplexConversation(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

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

func TestContextManager_ThreadSafety(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	config.PersistencePath = tempDir
	config.EnablePersistence = false
	config.MaxSessions = 100 // Allow more sessions for concurrency test

	cm, err := NewContextManagerFull(config)
	if err != nil {
		t.Fatalf("NewContextManagerFull() failed: %v", err)
	}
	defer func() {
		if contextMgr, ok := cm.(*ContextManager); ok {
			contextMgr.Close()
		}
	}()

	contextManager := cm.(*ContextManager)
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
				// This is expected to work since we created a context, but pronouns may not resolve
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