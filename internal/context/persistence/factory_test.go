package persistence

import (
	"testing"

	"genai-processing/pkg/types"
)

func TestNewPersistenceLayer_FileStorage(t *testing.T) {
	config := PersistenceConfig{
		Type:   FileStorageType,
		Path:   t.TempDir(),
		Format: "json",
	}

	persistenceLayer, err := NewPersistenceLayer(config)
	if err != nil {
		t.Fatalf("NewPersistenceLayer() failed: %v", err)
	}

	if persistenceLayer == nil {
		t.Fatal("NewPersistenceLayer() returned nil")
	}

	// Verify it's actually a FileStorage instance
	if _, ok := persistenceLayer.(*FileStorage); !ok {
		t.Error("Expected FileStorage instance")
	}

	// Test basic operations
	sessionID := "test-session"
	context := types.NewConversationContext(sessionID, "test-user")

	err = persistenceLayer.SaveSession(sessionID, context)
	if err != nil {
		t.Errorf("SaveSession() failed: %v", err)
	}

	loadedContext, err := persistenceLayer.LoadSession(sessionID)
	if err != nil {
		t.Errorf("LoadSession() failed: %v", err)
	}

	if loadedContext.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, loadedContext.SessionID)
	}

	err = persistenceLayer.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestNewPersistenceLayer_MemoryStorage(t *testing.T) {
	config := PersistenceConfig{
		Type:   MemoryStorageType,
		Path:   "/unused",
		Format: "unused",
	}

	persistenceLayer, err := NewPersistenceLayer(config)
	if err != nil {
		t.Fatalf("NewPersistenceLayer() failed: %v", err)
	}

	if persistenceLayer == nil {
		t.Fatal("NewPersistenceLayer() returned nil")
	}

	// Verify it's actually a MemoryStorage instance
	if _, ok := persistenceLayer.(*MemoryStorage); !ok {
		t.Error("Expected MemoryStorage instance")
	}

	// Test basic operations - memory storage should not persist anything
	sessionID := "test-session"
	context := types.NewConversationContext(sessionID, "test-user")

	// Save should succeed but not actually persist
	err = persistenceLayer.SaveSession(sessionID, context)
	if err != nil {
		t.Errorf("SaveSession() failed: %v", err)
	}

	// Load should fail since memory storage doesn't persist
	_, err = persistenceLayer.LoadSession(sessionID)
	if err == nil {
		t.Error("Expected error when loading from memory storage")
	}

	// LoadAllSessions should return empty map
	sessions, err := persistenceLayer.LoadAllSessions()
	if err != nil {
		t.Errorf("LoadAllSessions() failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions from memory storage, got %d", len(sessions))
	}

	err = persistenceLayer.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestNewPersistenceLayer_UnsupportedType(t *testing.T) {
	config := PersistenceConfig{
		Type:   "unsupported",
		Path:   "/unused",
		Format: "unused",
	}

	persistenceLayer, err := NewPersistenceLayer(config)
	if err == nil {
		t.Error("Expected error for unsupported persistence type")
	}

	if persistenceLayer != nil {
		t.Error("Expected nil persistence layer for unsupported type")
	}

	if err.Error() != "unsupported persistence type: unsupported" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestMemoryStorage_AllOperations(t *testing.T) {
	ms := NewMemoryStorage()

	if ms == nil {
		t.Fatal("NewMemoryStorage() returned nil")
	}

	sessionID := "memory-test-session"
	context := types.NewConversationContext(sessionID, "test-user")

	// Test SaveSession - should not error but not persist
	err := ms.SaveSession(sessionID, context)
	if err != nil {
		t.Errorf("SaveSession() failed: %v", err)
	}

	// Test LoadSession - should return not found error
	loadedContext, err := ms.LoadSession(sessionID)
	if err == nil {
		t.Error("Expected error when loading from memory storage")
	}

	if loadedContext != nil {
		t.Error("Expected nil context from memory storage")
	}

	expectedErrorMsg := "session " + sessionID + " not found"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}

	// Test DeleteSession - should not error
	err = ms.DeleteSession(sessionID)
	if err != nil {
		t.Errorf("DeleteSession() failed: %v", err)
	}

	// Test LoadAllSessions - should return empty map
	sessions, err := ms.LoadAllSessions()
	if err != nil {
		t.Errorf("LoadAllSessions() failed: %v", err)
	}

	if sessions == nil {
		t.Error("LoadAllSessions() returned nil map")
	}

	if len(sessions) != 0 {
		t.Errorf("Expected empty map from LoadAllSessions(), got %d entries", len(sessions))
	}

	// Test BatchSaveSessions - should not error
	batchSessions := map[string]*types.ConversationContext{
		"batch-1": types.NewConversationContext("batch-1", "user1"),
		"batch-2": types.NewConversationContext("batch-2", "user2"),
	}

	err = ms.BatchSaveSessions(batchSessions)
	if err != nil {
		t.Errorf("BatchSaveSessions() failed: %v", err)
	}

	// Test GetSessionMetadata - should return memory metadata
	metadata, err := ms.GetSessionMetadata()
	if err != nil {
		t.Errorf("GetSessionMetadata() failed: %v", err)
	}

	if metadata == nil {
		t.Fatal("GetSessionMetadata() returned nil")
	}

	if metadata.TotalSessions != 0 {
		t.Errorf("Expected 0 total sessions, got %d", metadata.TotalSessions)
	}

	if metadata.StorageSize != 0 {
		t.Errorf("Expected 0 storage size, got %d", metadata.StorageSize)
	}

	if metadata.StorageFormat != "memory" {
		t.Errorf("Expected storage format 'memory', got %s", metadata.StorageFormat)
	}

	if metadata.StoragePath != "memory" {
		t.Errorf("Expected storage path 'memory', got %s", metadata.StoragePath)
	}

	// Test Close - should not error
	err = ms.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestPersistenceTypes(t *testing.T) {
	// Test that persistence type constants are defined correctly
	if FileStorageType != "file" {
		t.Errorf("Expected FileStorageType to be 'file', got %s", FileStorageType)
	}

	if MemoryStorageType != "memory" {
		t.Errorf("Expected MemoryStorageType to be 'memory', got %s", MemoryStorageType)
	}
}

func TestPersistenceConfig_Struct(t *testing.T) {
	// Test that PersistenceConfig can be created and used properly
	config := PersistenceConfig{
		Type:   FileStorageType,
		Path:   "/test/path",
		Format: "json",
	}

	if config.Type != FileStorageType {
		t.Errorf("Expected Type %s, got %s", FileStorageType, config.Type)
	}

	if config.Path != "/test/path" {
		t.Errorf("Expected Path '/test/path', got %s", config.Path)
	}

	if config.Format != "json" {
		t.Errorf("Expected Format 'json', got %s", config.Format)
	}
}