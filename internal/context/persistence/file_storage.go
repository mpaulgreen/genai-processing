package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"genai-processing/pkg/types"
)

// FileStorage implements the PersistenceLayer interface using file-based storage.
// It stores each session as a separate file in JSON format for durability and
// human readability. Supports atomic writes to prevent corruption.
type FileStorage struct {
	basePath string
	format   string // "json" or "gob"
	mu       sync.RWMutex
	stats    PersistenceStats
}

// NewFileStorage creates a new file-based persistence layer.
func NewFileStorage(basePath, format string) (*FileStorage, error) {
	if format != "json" && format != "gob" {
		format = "json"
	}

	fs := &FileStorage{
		basePath: basePath,
		format:   format,
		stats:    PersistenceStats{},
	}

	// Create directory structure
	if err := fs.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create storage directories: %w", err)
	}

	return fs, nil
}

// ensureDirectories creates the necessary directory structure for file storage.
func (fs *FileStorage) ensureDirectories() error {
	directories := []string{
		fs.basePath,
		filepath.Join(fs.basePath, "sessions"),
		filepath.Join(fs.basePath, "backups"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// SaveSession saves a single session to a file with atomic write operation.
func (fs *FileStorage) SaveSession(sessionID string, ctx *types.ConversationContext) error {
	start := time.Now()
	defer func() {
		fs.mu.Lock()
		fs.stats.SaveOperations++
		duration := time.Since(start)
		if fs.stats.SaveOperations == 1 {
			fs.stats.AverageSaveTime = float64(duration.Nanoseconds()) / 1000000
		} else {
			fs.stats.AverageSaveTime = (fs.stats.AverageSaveTime + float64(duration.Nanoseconds())/1000000) / 2
		}
		fs.mu.Unlock()
	}()

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.validateSessionID(sessionID); err != nil {
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return err
	}

	filePath := fs.getSessionFilePath(sessionID)
	tempPath := filePath + ".tmp"

	// Write to temporary file first for atomic operation
	data, err := fs.serializeSession(ctx)
	if err != nil {
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return fmt.Errorf("failed to serialize session %s: %w", sessionID, err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return fmt.Errorf("failed to write session file %s: %w", sessionID, err)
	}

	// Atomic rename from temp to final location
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return fmt.Errorf("failed to rename session file %s: %w", sessionID, err)
	}

	return nil
}

// LoadSession loads a single session from storage.
func (fs *FileStorage) LoadSession(sessionID string) (*types.ConversationContext, error) {
	start := time.Now()
	defer func() {
		fs.mu.Lock()
		fs.stats.LoadOperations++
		duration := time.Since(start)
		if fs.stats.LoadOperations == 1 {
			fs.stats.AverageLoadTime = float64(duration.Nanoseconds()) / 1000000
		} else {
			fs.stats.AverageLoadTime = (fs.stats.AverageLoadTime + float64(duration.Nanoseconds())/1000000) / 2
		}
		fs.mu.Unlock()
	}()

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if err := fs.validateSessionID(sessionID); err != nil {
		fs.mu.RUnlock()
		fs.mu.Lock()
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		fs.mu.Unlock()
		fs.mu.RLock()
		return nil, err
	}

	filePath := fs.getSessionFilePath(sessionID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		fs.mu.RUnlock()
		fs.mu.Lock()
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		fs.mu.Unlock()
		fs.mu.RLock()
		return nil, fmt.Errorf("failed to read session file %s: %w", sessionID, err)
	}

	session, err := fs.deserializeSession(data)
	if err != nil {
		fs.mu.RUnlock()
		fs.mu.Lock()
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		fs.mu.Unlock()
		fs.mu.RLock()
		return nil, fmt.Errorf("failed to deserialize session %s: %w", sessionID, err)
	}

	return session, nil
}

// DeleteSession removes a session from storage.
func (fs *FileStorage) DeleteSession(sessionID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.stats.DeleteOperations++

	if err := fs.validateSessionID(sessionID); err != nil {
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return err
	}

	filePath := fs.getSessionFilePath(sessionID)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		fs.stats.ErrorCount++
		fs.stats.LastError = err.Error()
		return fmt.Errorf("failed to delete session file %s: %w", sessionID, err)
	}

	return nil
}

// LoadAllSessions loads all sessions from storage for startup recovery.
func (fs *FileStorage) LoadAllSessions() (map[string]*types.ConversationContext, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	sessionsDir := filepath.Join(fs.basePath, "sessions")
	sessions := make(map[string]*types.ConversationContext)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return sessions, nil // Return empty map if directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".json") {
			continue
		}

		sessionID := strings.TrimSuffix(fileName, ".json")
		
		// Read and deserialize session
		filePath := filepath.Join(sessionsDir, fileName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Log error but continue with other sessions
			continue
		}

		session, err := fs.deserializeSession(data)
		if err != nil {
			// Log error but continue with other sessions
			continue
		}

		sessions[sessionID] = session
	}

	return sessions, nil
}

// BatchSaveSessions saves multiple sessions efficiently.
func (fs *FileStorage) BatchSaveSessions(sessions map[string]*types.ConversationContext) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for sessionID, session := range sessions {
		if err := fs.validateSessionID(sessionID); err != nil {
			fs.stats.ErrorCount++
			fs.stats.LastError = err.Error()
			continue // Skip invalid session IDs
		}

		filePath := fs.getSessionFilePath(sessionID)
		tempPath := filePath + ".tmp"

		data, err := fs.serializeSession(session)
		if err != nil {
			fs.stats.ErrorCount++
			fs.stats.LastError = err.Error()
			continue // Skip sessions that can't be serialized
		}

		if err := os.WriteFile(tempPath, data, 0644); err != nil {
			fs.stats.ErrorCount++
			fs.stats.LastError = err.Error()
			continue
		}

		if err := os.Rename(tempPath, filePath); err != nil {
			os.Remove(tempPath)
			fs.stats.ErrorCount++
			fs.stats.LastError = err.Error()
			continue
		}

		fs.stats.SaveOperations++
	}

	return nil
}

// GetSessionMetadata returns metadata about the storage.
func (fs *FileStorage) GetSessionMetadata() (*StorageMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	sessionsDir := filepath.Join(fs.basePath, "sessions")
	
	var totalSessions int
	var totalSize int64

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &StorageMetadata{
				TotalSessions: 0,
				StorageSize:   0,
				StorageFormat: fs.format,
				StoragePath:   fs.basePath,
			}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".json") {
			totalSessions++
			
			info, err := entry.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
	}

	return &StorageMetadata{
		TotalSessions: totalSessions,
		StorageSize:   totalSize,
		StorageFormat: fs.format,
		StoragePath:   fs.basePath,
	}, nil
}

// GetStats returns runtime statistics about persistence operations.
func (fs *FileStorage) GetStats() PersistenceStats {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.stats
}

// Close performs cleanup operations.
func (fs *FileStorage) Close() error {
	// File storage doesn't require explicit closing
	return nil
}

// getSessionFilePath returns the file path for a session.
func (fs *FileStorage) getSessionFilePath(sessionID string) string {
	fileName := sessionID + ".json"
	return filepath.Join(fs.basePath, "sessions", fileName)
}

// validateSessionID validates a session ID for file system safety.
func (fs *FileStorage) validateSessionID(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if strings.Contains(sessionID, "..") || strings.Contains(sessionID, "/") || strings.Contains(sessionID, "\\") {
		return fmt.Errorf("session ID contains invalid characters: %s", sessionID)
	}

	if len(sessionID) > 255 {
		return fmt.Errorf("session ID too long: %d characters", len(sessionID))
	}

	return nil
}

// serializeSession converts a session to bytes for storage.
func (fs *FileStorage) serializeSession(session *types.ConversationContext) ([]byte, error) {
	return json.MarshalIndent(session, "", "  ")
}

// deserializeSession converts bytes back to a session.
func (fs *FileStorage) deserializeSession(data []byte) (*types.ConversationContext, error) {
	var session types.ConversationContext
	err := json.Unmarshal(data, &session)
	return &session, err
}