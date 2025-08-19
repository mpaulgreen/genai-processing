package persistence

import (
	"fmt"

	"genai-processing/pkg/types"
)

// PersistenceType defines the type of persistence backend
type PersistenceType string

const (
	// FileStorageType uses file-based storage
	FileStorageType PersistenceType = "file"
	// MemoryStorageType uses in-memory only (no persistence)
	MemoryStorageType PersistenceType = "memory"
	// Future: RedisStorage, DatabaseStorage, etc.
)

// PersistenceConfig holds configuration for creating persistence layers
type PersistenceConfig struct {
	Type   PersistenceType `yaml:"type" json:"type"`
	Path   string          `yaml:"path" json:"path"`
	Format string          `yaml:"format" json:"format"`
}

// NewPersistenceLayer creates a new persistence layer based on configuration.
func NewPersistenceLayer(config PersistenceConfig) (PersistenceLayer, error) {
	switch config.Type {
	case FileStorageType:
		return NewFileStorage(config.Path, config.Format)
	case MemoryStorageType:
		return NewMemoryStorage(), nil
	default:
		return nil, fmt.Errorf("unsupported persistence type: %s", config.Type)
	}
}

// MemoryStorage implements PersistenceLayer with in-memory only storage (no persistence).
// This is useful for testing or when persistence is not desired.
type MemoryStorage struct{}

// NewMemoryStorage creates a new memory-only storage (no persistence).
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

func (ms *MemoryStorage) SaveSession(sessionID string, ctx *types.ConversationContext) error {
	// Memory storage doesn't persist anything
	return nil
}

func (ms *MemoryStorage) LoadSession(sessionID string) (*types.ConversationContext, error) {
	// Memory storage can't load anything
	return nil, fmt.Errorf("session %s not found", sessionID)
}

func (ms *MemoryStorage) DeleteSession(sessionID string) error {
	// Memory storage doesn't need to delete anything
	return nil
}

func (ms *MemoryStorage) LoadAllSessions() (map[string]*types.ConversationContext, error) {
	// Memory storage returns empty map
	return make(map[string]*types.ConversationContext), nil
}

func (ms *MemoryStorage) BatchSaveSessions(sessions map[string]*types.ConversationContext) error {
	// Memory storage doesn't persist anything
	return nil
}

func (ms *MemoryStorage) GetSessionMetadata() (*StorageMetadata, error) {
	return &StorageMetadata{
		TotalSessions: 0,
		StorageSize:   0,
		StorageFormat: "memory",
		StoragePath:   "memory",
	}, nil
}

func (ms *MemoryStorage) Close() error {
	// Memory storage doesn't need cleanup
	return nil
}