package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"genai-processing/pkg/types"
)

func TestNewFileStorage(t *testing.T) {
	tempDir := t.TempDir()

	// Test with valid parameters
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	if fs == nil {
		t.Fatal("NewFileStorage() returned nil")
	}

	if fs.basePath != tempDir {
		t.Errorf("Expected basePath %s, got %s", tempDir, fs.basePath)
	}

	if fs.format != "json" {
		t.Errorf("Expected format 'json', got %s", fs.format)
	}

	// Test directory creation
	sessionsDir := filepath.Join(tempDir, "sessions")
	backupsDir := filepath.Join(tempDir, "backups")

	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Error("Sessions directory was not created")
	}

	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		t.Error("Backups directory was not created")
	}
}

func TestNewFileStorage_InvalidFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Test with invalid format - should default to json
	fs, err := NewFileStorage(tempDir, "invalid")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	if fs.format != "json" {
		t.Errorf("Expected format to default to 'json', got %s", fs.format)
	}
}

func TestFileStorage_SaveAndLoadSession(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Create test session
	sessionID := "test-session-1"
	originalContext := types.NewConversationContext(sessionID, "test-user")
	originalContext.AddConversationEntry(
		"Who deleted the CRD?",
		&types.StructuredQuery{
			LogSource: "kube-apiserver",
			Verb:      *types.NewStringOrArray("delete"),
			Resource:  *types.NewStringOrArray("customresourcedefinitions"),
		},
		map[string]string{"last_user": "john.doe"},
	)

	// Test SaveSession
	err = fs.SaveSession(sessionID, originalContext)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Verify file was created
	expectedPath := fs.getSessionFilePath(sessionID)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// Test LoadSession
	loadedContext, err := fs.LoadSession(sessionID)
	if err != nil {
		t.Fatalf("LoadSession() failed: %v", err)
	}

	if loadedContext == nil {
		t.Fatal("LoadSession() returned nil context")
	}

	// Verify loaded context matches original
	if loadedContext.SessionID != originalContext.SessionID {
		t.Errorf("Expected SessionID %s, got %s", originalContext.SessionID, loadedContext.SessionID)
	}

	if loadedContext.UserID != originalContext.UserID {
		t.Errorf("Expected UserID %s, got %s", originalContext.UserID, loadedContext.UserID)
	}

	if len(loadedContext.ConversationHistory) != len(originalContext.ConversationHistory) {
		t.Errorf("Expected %d conversation entries, got %d",
			len(originalContext.ConversationHistory), len(loadedContext.ConversationHistory))
	}
}

func TestFileStorage_LoadNonExistentSession(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Try to load non-existent session
	context, err := fs.LoadSession("non-existent-session")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	if context != nil {
		t.Error("Expected nil context for non-existent session")
	}

	if !containsString(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %s", err.Error())
	}
}

func TestFileStorage_DeleteSession(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	sessionID := "test-session-delete"
	context := types.NewConversationContext(sessionID, "test-user")

	// Save session first
	err = fs.SaveSession(sessionID, context)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Verify file exists
	sessionPath := fs.getSessionFilePath(sessionID)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Fatal("Session file was not created")
	}

	// Delete session
	err = fs.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession() failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("Session file was not deleted")
	}

	// Deleting non-existent session should not error
	err = fs.DeleteSession("non-existent-session")
	if err != nil {
		t.Errorf("DeleteSession() should not error for non-existent session: %v", err)
	}
}

func TestFileStorage_LoadAllSessions(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Create multiple test sessions
	sessions := map[string]*types.ConversationContext{
		"session-1": types.NewConversationContext("session-1", "user1"),
		"session-2": types.NewConversationContext("session-2", "user2"),
		"session-3": types.NewConversationContext("session-3", "user3"),
	}

	// Save all sessions
	for sessionID, context := range sessions {
		err = fs.SaveSession(sessionID, context)
		if err != nil {
			t.Fatalf("SaveSession() failed for %s: %v", sessionID, err)
		}
	}

	// Load all sessions
	loadedSessions, err := fs.LoadAllSessions()
	if err != nil {
		t.Fatalf("LoadAllSessions() failed: %v", err)
	}

	if len(loadedSessions) != len(sessions) {
		t.Errorf("Expected %d sessions, got %d", len(sessions), len(loadedSessions))
	}

	// Verify each session was loaded correctly
	for sessionID, originalContext := range sessions {
		loadedContext, exists := loadedSessions[sessionID]
		if !exists {
			t.Errorf("Session %s was not loaded", sessionID)
			continue
		}

		if loadedContext.SessionID != originalContext.SessionID {
			t.Errorf("SessionID mismatch for %s: expected %s, got %s",
				sessionID, originalContext.SessionID, loadedContext.SessionID)
		}

		if loadedContext.UserID != originalContext.UserID {
			t.Errorf("UserID mismatch for %s: expected %s, got %s",
				sessionID, originalContext.UserID, loadedContext.UserID)
		}
	}
}

func TestFileStorage_LoadAllSessions_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Load from empty directory
	sessions, err := fs.LoadAllSessions()
	if err != nil {
		t.Fatalf("LoadAllSessions() failed on empty directory: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions from empty directory, got %d", len(sessions))
	}
}

func TestFileStorage_BatchSaveSessions(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Create multiple sessions
	sessions := map[string]*types.ConversationContext{
		"batch-session-1": types.NewConversationContext("batch-session-1", "user1"),
		"batch-session-2": types.NewConversationContext("batch-session-2", "user2"),
		"batch-session-3": types.NewConversationContext("batch-session-3", "user3"),
	}

	// Batch save sessions
	err = fs.BatchSaveSessions(sessions)
	if err != nil {
		t.Fatalf("BatchSaveSessions() failed: %v", err)
	}

	// Verify all sessions were saved
	for sessionID := range sessions {
		sessionPath := fs.getSessionFilePath(sessionID)
		if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
			t.Errorf("Session file for %s was not created", sessionID)
		}
	}

	// Load and verify sessions
	for sessionID, originalContext := range sessions {
		loadedContext, err := fs.LoadSession(sessionID)
		if err != nil {
			t.Errorf("Failed to load batch-saved session %s: %v", sessionID, err)
			continue
		}

		if loadedContext.SessionID != originalContext.SessionID {
			t.Errorf("SessionID mismatch for %s", sessionID)
		}
	}
}

func TestFileStorage_GetSessionMetadata(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Get metadata from empty storage
	metadata, err := fs.GetSessionMetadata()
	if err != nil {
		t.Fatalf("GetSessionMetadata() failed: %v", err)
	}

	if metadata.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions in empty storage, got %d", metadata.TotalSessions)
	}

	if metadata.StorageSize != 0 {
		t.Errorf("Expected 0 storage size in empty storage, got %d", metadata.StorageSize)
	}

	if metadata.StorageFormat != "json" {
		t.Errorf("Expected StorageFormat 'json', got %s", metadata.StorageFormat)
	}

	if metadata.StoragePath != tempDir {
		t.Errorf("Expected StoragePath %s, got %s", tempDir, metadata.StoragePath)
	}

	// Add some sessions
	sessions := map[string]*types.ConversationContext{
		"meta-session-1": types.NewConversationContext("meta-session-1", "user1"),
		"meta-session-2": types.NewConversationContext("meta-session-2", "user2"),
	}

	for sessionID, context := range sessions {
		err = fs.SaveSession(sessionID, context)
		if err != nil {
			t.Fatalf("SaveSession() failed for %s: %v", sessionID, err)
		}
	}

	// Get metadata after adding sessions
	metadata, err = fs.GetSessionMetadata()
	if err != nil {
		t.Fatalf("GetSessionMetadata() failed: %v", err)
	}

	if metadata.TotalSessions != len(sessions) {
		t.Errorf("Expected %d sessions, got %d", len(sessions), metadata.TotalSessions)
	}

	if metadata.StorageSize <= 0 {
		t.Error("Expected positive storage size")
	}
}

func TestFileStorage_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Get initial stats
	stats := fs.GetStats()
	if stats.SaveOperations != 0 {
		t.Errorf("Expected 0 save operations initially, got %d", stats.SaveOperations)
	}

	if stats.LoadOperations != 0 {
		t.Errorf("Expected 0 load operations initially, got %d", stats.LoadOperations)
	}

	// Perform some operations
	sessionID := "stats-session"
	context := types.NewConversationContext(sessionID, "test-user")

	err = fs.SaveSession(sessionID, context)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	_, err = fs.LoadSession(sessionID)
	if err != nil {
		t.Fatalf("LoadSession() failed: %v", err)
	}

	err = fs.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession() failed: %v", err)
	}

	// Check updated stats
	stats = fs.GetStats()
	if stats.SaveOperations != 1 {
		t.Errorf("Expected 1 save operation, got %d", stats.SaveOperations)
	}

	if stats.LoadOperations != 1 {
		t.Errorf("Expected 1 load operation, got %d", stats.LoadOperations)
	}

	if stats.DeleteOperations != 1 {
		t.Errorf("Expected 1 delete operation, got %d", stats.DeleteOperations)
	}

	if stats.AverageSaveTime <= 0 {
		t.Error("Expected positive average save time")
	}

	if stats.AverageLoadTime <= 0 {
		t.Error("Expected positive average load time")
	}
}

func TestFileStorage_ValidateSessionID(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	validIDs := []string{
		"valid-session-id",
		"session123",
		"user@domain.com",
		"session_with_underscores",
		"session-with-dashes",
	}

	for _, sessionID := range validIDs {
		err := fs.validateSessionID(sessionID)
		if err != nil {
			t.Errorf("validateSessionID() failed for valid ID '%s': %v", sessionID, err)
		}
	}

	invalidIDs := []string{
		"",                                    // Empty
		"session/with/slashes",               // Forward slashes
		"session\\with\\backslashes",         // Backslashes
		"session..with..dots",                // Double dots
		string(make([]byte, 300)),            // Too long
	}

	for _, sessionID := range invalidIDs {
		err := fs.validateSessionID(sessionID)
		if err == nil {
			t.Errorf("validateSessionID() should have failed for invalid ID '%s'", sessionID)
		}
	}
}

func TestFileStorage_AtomicWrites(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	sessionID := "atomic-session"
	context := types.NewConversationContext(sessionID, "test-user")

	// Save session
	err = fs.SaveSession(sessionID, context)
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Verify no temporary files are left behind
	sessionsDir := filepath.Join(tempDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("Failed to read sessions directory: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("Temporary file left behind: %s", entry.Name())
		}
	}

	// Verify the actual session file exists
	expectedFile := sessionID + ".json"
	found := false
	for _, entry := range entries {
		if entry.Name() == expectedFile {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Session file %s not found", expectedFile)
	}
}

func TestFileStorage_Close(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir, "json")
	if err != nil {
		t.Fatalf("NewFileStorage() failed: %v", err)
	}

	// Close should not error for file storage
	err = fs.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || len(substr) == 0 || 
		    (len(substr) <= len(s) && s[0:len(substr)] == substr) ||
		    (len(substr) <= len(s) && s[len(s)-len(substr):] == substr) ||
		    (len(substr) < len(s) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}