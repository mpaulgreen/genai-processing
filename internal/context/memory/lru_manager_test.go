package memory

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestNewLRUManager(t *testing.T) {
	maxSessions := 100
	maxMemoryMB := 50

	lru := NewLRUManager(maxSessions, maxMemoryMB)

	if lru == nil {
		t.Fatal("NewLRUManager() returned nil")
	}

	if lru.maxSessions != maxSessions {
		t.Errorf("Expected maxSessions %d, got %d", maxSessions, lru.maxSessions)
	}

	if lru.maxMemoryMB != maxMemoryMB {
		t.Errorf("Expected maxMemoryMB %d, got %d", maxMemoryMB, lru.maxMemoryMB)
	}

	if lru.sessions == nil {
		t.Error("Sessions map not initialized")
	}

	if lru.accessOrder == nil {
		t.Error("Access order slice not initialized")
	}

	stats := lru.GetStats()
	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 initial sessions, got %d", stats.TotalSessions)
	}

	if stats.MemoryLimitKB != int64(maxMemoryMB*1024) {
		t.Errorf("Expected memory limit %d KB, got %d KB", maxMemoryMB*1024, stats.MemoryLimitKB)
	}
}

func TestLRUManager_PutAndGet(t *testing.T) {
	lru := NewLRUManager(10, 100)

	sessionID := "test-session-1"
	context := types.NewConversationContext(sessionID, "test-user")

	// Test Put
	lru.Put(sessionID, context)

	stats := lru.GetStats()
	if stats.TotalSessions != 1 {
		t.Errorf("Expected 1 session after Put, got %d", stats.TotalSessions)
	}

	// Test Get
	retrievedContext, found := lru.Get(sessionID)
	if !found {
		t.Error("Session not found after Put")
	}

	if retrievedContext == nil {
		t.Fatal("Retrieved context is nil")
	}

	if retrievedContext.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, retrievedContext.SessionID)
	}

	if retrievedContext.UserID != context.UserID {
		t.Errorf("Expected UserID %s, got %s", context.UserID, retrievedContext.UserID)
	}

	// Test stats after access
	stats = lru.GetStats()
	if stats.TotalAccesses != 1 {
		t.Errorf("Expected 1 total access, got %d", stats.TotalAccesses)
	}

	if stats.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.CacheHits)
	}

	if stats.HitRate != 1.0 {
		t.Errorf("Expected hit rate 1.0, got %f", stats.HitRate)
	}
}

func TestLRUManager_GetNonExistent(t *testing.T) {
	lru := NewLRUManager(10, 100)

	// Try to get non-existent session
	context, found := lru.Get("non-existent")
	if found {
		t.Error("Expected session not to be found")
	}

	if context != nil {
		t.Error("Expected nil context for non-existent session")
	}

	stats := lru.GetStats()
	if stats.TotalAccesses != 1 {
		t.Errorf("Expected 1 total access, got %d", stats.TotalAccesses)
	}

	if stats.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits, got %d", stats.CacheHits)
	}

	if stats.HitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0, got %f", stats.HitRate)
	}
}

func TestLRUManager_UpdateExisting(t *testing.T) {
	lru := NewLRUManager(10, 100)

	sessionID := "test-session-update"
	context1 := types.NewConversationContext(sessionID, "user1")
	context2 := types.NewConversationContext(sessionID, "user2")

	// Put initial context
	lru.Put(sessionID, context1)

	// Update with new context
	lru.Put(sessionID, context2)

	stats := lru.GetStats()
	if stats.TotalSessions != 1 {
		t.Errorf("Expected 1 session after update, got %d", stats.TotalSessions)
	}

	// Retrieve and verify it's the updated context
	retrievedContext, found := lru.Get(sessionID)
	if !found {
		t.Error("Session not found after update")
	}

	if retrievedContext.UserID != "user2" {
		t.Errorf("Expected updated UserID 'user2', got %s", retrievedContext.UserID)
	}
}

func TestLRUManager_LRUEviction_SessionLimit(t *testing.T) {
	maxSessions := 3
	lru := NewLRUManager(maxSessions, 1000) // High memory limit to test session limit

	// Add sessions up to the limit
	for i := 0; i < maxSessions; i++ {
		sessionID := "session-" + string(rune('1'+i))
		context := types.NewConversationContext(sessionID, "user")
		lru.Put(sessionID, context)
	}

	stats := lru.GetStats()
	if stats.TotalSessions != maxSessions {
		t.Errorf("Expected %d sessions, got %d", maxSessions, stats.TotalSessions)
	}

	// Add one more session - should trigger eviction
	extraSessionID := "session-extra"
	extraContext := types.NewConversationContext(extraSessionID, "user")
	lru.Put(extraSessionID, extraContext)

	stats = lru.GetStats()
	if stats.TotalSessions != maxSessions {
		t.Errorf("Expected %d sessions after eviction, got %d", maxSessions, stats.TotalSessions)
	}

	if stats.EvictionCount != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.EvictionCount)
	}

	// The oldest session (session-1) should be evicted
	_, found := lru.Get("session-1")
	if found {
		t.Error("Oldest session should have been evicted")
	}

	// The newest session should still be there
	_, found = lru.Get(extraSessionID)
	if !found {
		t.Error("Newest session should not have been evicted")
	}
}

func TestLRUManager_AccessOrder(t *testing.T) {
	lru := NewLRUManager(3, 1000)

	// Add three sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, sessionID := range sessions {
		context := types.NewConversationContext(sessionID, "user")
		lru.Put(sessionID, context)
	}

	// Access session-1 to make it most recently used
	lru.Get("session-1")

	// Add a new session - session-2 should be evicted (oldest unaccessed)
	newContext := types.NewConversationContext("session-4", "user")
	lru.Put("session-4", newContext)

	// Verify session-2 was evicted
	_, found := lru.Get("session-2")
	if found {
		t.Error("session-2 should have been evicted")
	}

	// Verify session-1 is still there (was recently accessed)
	_, found = lru.Get("session-1")
	if !found {
		t.Error("session-1 should not have been evicted")
	}

	// Verify session-3 is still there
	_, found = lru.Get("session-3")
	if !found {
		t.Error("session-3 should not have been evicted")
	}

	// Verify session-4 is there
	_, found = lru.Get("session-4")
	if !found {
		t.Error("session-4 should be present")
	}
}

func TestLRUManager_Remove(t *testing.T) {
	lru := NewLRUManager(10, 100)

	sessionID := "test-session-remove"
	context := types.NewConversationContext(sessionID, "test-user")

	// Add session
	lru.Put(sessionID, context)

	// Verify it exists
	_, found := lru.Get(sessionID)
	if !found {
		t.Error("Session should exist before removal")
	}

	// Remove session
	removed := lru.Remove(sessionID)
	if !removed {
		t.Error("Remove should return true for existing session")
	}

	// Verify it's gone
	_, found = lru.Get(sessionID)
	if found {
		t.Error("Session should not exist after removal")
	}

	stats := lru.GetStats()
	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions after removal, got %d", stats.TotalSessions)
	}

	// Try to remove non-existent session
	removed = lru.Remove("non-existent")
	if removed {
		t.Error("Remove should return false for non-existent session")
	}
}

func TestLRUManager_GetAllSessions(t *testing.T) {
	lru := NewLRUManager(10, 100)

	// Add multiple sessions
	expectedSessions := map[string]string{
		"session-1": "user1",
		"session-2": "user2",
		"session-3": "user3",
	}

	for sessionID, userID := range expectedSessions {
		context := types.NewConversationContext(sessionID, userID)
		lru.Put(sessionID, context)
	}

	// Get all sessions
	allSessions := lru.GetAllSessions()

	if len(allSessions) != len(expectedSessions) {
		t.Errorf("Expected %d sessions, got %d", len(expectedSessions), len(allSessions))
	}

	// Verify each session
	for sessionID, expectedUserID := range expectedSessions {
		context, exists := allSessions[sessionID]
		if !exists {
			t.Errorf("Session %s not found in GetAllSessions result", sessionID)
			continue
		}

		if context.UserID != expectedUserID {
			t.Errorf("Expected UserID %s for session %s, got %s",
				expectedUserID, sessionID, context.UserID)
		}
	}
}

func TestLRUManager_Clear(t *testing.T) {
	lru := NewLRUManager(10, 100)

	// Add some sessions
	for i := 0; i < 5; i++ {
		sessionID := "session-" + string(rune('1'+i))
		context := types.NewConversationContext(sessionID, "user")
		lru.Put(sessionID, context)
	}

	stats := lru.GetStats()
	if stats.TotalSessions != 5 {
		t.Errorf("Expected 5 sessions before clear, got %d", stats.TotalSessions)
	}

	// Clear all sessions
	lru.Clear()

	stats = lru.GetStats()
	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions after clear, got %d", stats.TotalSessions)
	}

	// Verify sessions are gone
	_, found := lru.Get("session-1")
	if found {
		t.Error("Sessions should not exist after clear")
	}

	allSessions := lru.GetAllSessions()
	if len(allSessions) != 0 {
		t.Errorf("Expected 0 sessions from GetAllSessions after clear, got %d", len(allSessions))
	}
}

func TestLRUManager_MemoryEstimation(t *testing.T) {
	lru := NewLRUManager(10, 100)

	sessionID := "memory-test-session"
	context := types.NewConversationContext(sessionID, "test-user")

	// Add some conversation history to increase memory footprint
	for i := 0; i < 10; i++ {
		query := "Test query " + string(rune('1'+i))
		response := &types.StructuredQuery{
			LogSource: "kube-apiserver",
			Verb:      *types.NewStringOrArray("get"),
		}
		context.AddConversationEntry(query, response, nil)
	}

	// Put session and check memory estimation
	lru.Put(sessionID, context)

	stats := lru.GetStats()
	if stats.MemoryUsageKB <= 0 {
		t.Error("Expected positive memory usage")
	}

	// Memory usage should increase with more data
	largerContext := types.NewConversationContext("larger-session", "test-user-with-longer-id")
	for i := 0; i < 100; i++ {
		query := "Much longer test query with more text to increase memory footprint " + string(rune('1'+(i%10)))
		response := &types.StructuredQuery{
			LogSource:           "kube-apiserver-with-longer-name",
			Verb:                *types.NewStringOrArray("get"),
			Resource:            *types.NewStringOrArray("pods"),
			ResourceNamePattern: "large-resource-name-pattern",
			Timeframe:          "yesterday-with-extended-timeframe",
		}
		largerContext.AddConversationEntry(query, response, nil)
	}

	lru.Put("larger-session", largerContext)

	newStats := lru.GetStats()
	if newStats.MemoryUsageKB <= stats.MemoryUsageKB {
		t.Error("Memory usage should increase with larger context")
	}
}

func TestLRUManager_StatsAccuracy(t *testing.T) {
	lru := NewLRUManager(10, 100)

	// Test initial stats
	stats := lru.GetStats()
	if stats.TotalSessions != 0 {
		t.Error("Expected 0 initial sessions")
	}
	if stats.TotalAccesses != 0 {
		t.Error("Expected 0 initial accesses")
	}
	if stats.CacheHits != 0 {
		t.Error("Expected 0 initial cache hits")
	}
	if stats.HitRate != 0 {
		t.Error("Expected 0 initial hit rate")
	}
	if stats.EvictionCount != 0 {
		t.Error("Expected 0 initial evictions")
	}

	// Add a session
	sessionID := "stats-session"
	context := types.NewConversationContext(sessionID, "user")
	lru.Put(sessionID, context)

	// Access existing session - should be a hit
	_, found := lru.Get(sessionID)
	if !found {
		t.Error("Session should be found")
	}

	// Check stats after first access
	stats = lru.GetStats()
	t.Logf("After first access: TotalAccesses=%d, CacheHits=%d, HitRate=%f", 
		stats.TotalAccesses, stats.CacheHits, stats.HitRate)

	// Access non-existent session - should be a miss
	_, found = lru.Get("non-existent")
	if found {
		t.Error("Non-existent session should not be found")
	}

	// Check final stats
	stats = lru.GetStats()
	t.Logf("After second access: TotalAccesses=%d, CacheHits=%d, HitRate=%f", 
		stats.TotalAccesses, stats.CacheHits, stats.HitRate)

	if stats.TotalSessions != 1 {
		t.Errorf("Expected 1 total session, got %d", stats.TotalSessions)
	}
	if stats.TotalAccesses != 2 { // 1 hit + 1 miss
		t.Errorf("Expected 2 total accesses, got %d", stats.TotalAccesses)
	}
	if stats.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.CacheHits)
	}

	// The actual hit rate calculation should be correct
	if stats.TotalAccesses > 0 {
		actualHitRate := float64(stats.CacheHits) / float64(stats.TotalAccesses)
		if abs(stats.HitRate-actualHitRate) > 0.01 {
			t.Errorf("Hit rate calculation inconsistent: reported %.3f, calculated %.3f", 
				stats.HitRate, actualHitRate)
		}
	}
}

func TestLRUManager_OldestSessionAge(t *testing.T) {
	lru := NewLRUManager(10, 100)

	// Test that oldest session age calculation doesn't crash
	// and that we can at least access the field
	stats := lru.GetStats()
	if stats.OldestSessionAge < 0 {
		t.Errorf("Oldest session age should not be negative, got %d", stats.OldestSessionAge)
	}

	// Add a session
	sessionID := "age-test-session"
	context := types.NewConversationContext(sessionID, "user")
	lru.Put(sessionID, context)

	stats = lru.GetStats()
	// The session age should be 0 or very small since it was just created
	if stats.OldestSessionAge < 0 {
		t.Errorf("Oldest session age should not be negative, got %d", stats.OldestSessionAge)
	}

	// Add multiple sessions and verify we don't get errors
	for i := 0; i < 5; i++ {
		sid := "session-" + string(rune('1'+i))
		ctx := types.NewConversationContext(sid, "user")
		lru.Put(sid, ctx)
	}

	stats = lru.GetStats()
	if stats.OldestSessionAge < 0 {
		t.Errorf("Oldest session age should not be negative with multiple sessions, got %d", stats.OldestSessionAge)
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}