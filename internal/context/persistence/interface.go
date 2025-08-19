package persistence

import (
	"genai-processing/pkg/types"
)

// PersistenceLayer defines the interface for session persistence operations.
// This abstraction allows different storage backends (file, database, redis)
// while maintaining a consistent interface for the ContextManager.
type PersistenceLayer interface {
	// SaveSession persists a single session to storage
	SaveSession(sessionID string, ctx *types.ConversationContext) error

	// LoadSession retrieves a single session from storage
	LoadSession(sessionID string) (*types.ConversationContext, error)

	// DeleteSession removes a session from storage
	DeleteSession(sessionID string) error

	// LoadAllSessions retrieves all sessions from storage for startup recovery
	LoadAllSessions() (map[string]*types.ConversationContext, error)

	// BatchSaveSessions saves multiple sessions efficiently
	BatchSaveSessions(sessions map[string]*types.ConversationContext) error

	// GetSessionMetadata returns metadata about stored sessions
	GetSessionMetadata() (*StorageMetadata, error)

	// Close closes the persistence layer and performs cleanup
	Close() error
}

// StorageMetadata contains information about the storage state
type StorageMetadata struct {
	// TotalSessions is the number of sessions in storage
	TotalSessions int `json:"total_sessions"`

	// StorageSize is the total storage size in bytes
	StorageSize int64 `json:"storage_size"`

	// LastBackup is the timestamp of the last backup
	LastBackup int64 `json:"last_backup,omitempty"`

	// StorageFormat is the format used for persistence
	StorageFormat string `json:"storage_format"`

	// StoragePath is the base path for storage
	StoragePath string `json:"storage_path"`
}

// PersistenceStats contains runtime statistics about persistence operations
type PersistenceStats struct {
	// SaveOperations is the total number of save operations
	SaveOperations int64 `json:"save_operations"`

	// LoadOperations is the total number of load operations
	LoadOperations int64 `json:"load_operations"`

	// DeleteOperations is the total number of delete operations
	DeleteOperations int64 `json:"delete_operations"`

	// ErrorCount is the total number of errors
	ErrorCount int64 `json:"error_count"`

	// LastError is the last error encountered
	LastError string `json:"last_error,omitempty"`

	// AverageSaveTime is the average time to save a session in milliseconds
	AverageSaveTime float64 `json:"average_save_time_ms"`

	// AverageLoadTime is the average time to load a session in milliseconds
	AverageLoadTime float64 `json:"average_load_time_ms"`
}