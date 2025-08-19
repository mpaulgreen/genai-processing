package context

import (
	"time"
)

// ContextManagerConfig defines configuration parameters for the ContextManager.
// This structure allows flexible configuration of session management, persistence,
// and memory management settings for different environments.
type ContextManagerConfig struct {
	// CleanupInterval defines how often expired sessions are cleaned up
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval" default:"5m"`

	// SessionTimeout defines the default timeout for sessions
	SessionTimeout time.Duration `yaml:"session_timeout" json:"session_timeout" default:"24h"`

	// MaxSessions defines the maximum number of active sessions allowed
	MaxSessions int `yaml:"max_sessions" json:"max_sessions" default:"10000"`

	// MaxMemoryMB defines the maximum memory usage in megabytes
	MaxMemoryMB int `yaml:"max_memory_mb" json:"max_memory_mb" default:"100"`

	// EnablePersistence enables session persistence to storage
	EnablePersistence bool `yaml:"enable_persistence" json:"enable_persistence" default:"true"`

	// PersistencePath defines the path for session storage
	PersistencePath string `yaml:"persistence_path" json:"persistence_path" default:"./sessions"`

	// PersistenceFormat defines the storage format (json, gob)
	PersistenceFormat string `yaml:"persistence_format" json:"persistence_format" default:"json"`

	// PersistenceInterval defines how often to persist dirty sessions
	PersistenceInterval time.Duration `yaml:"persistence_interval" json:"persistence_interval" default:"30s"`

	// EnableAsyncPersistence enables asynchronous session persistence
	EnableAsyncPersistence bool `yaml:"enable_async_persistence" json:"enable_async_persistence" default:"true"`
}

// DefaultConfig returns a default configuration for the ContextManager.
func DefaultConfig() *ContextManagerConfig {
	return &ContextManagerConfig{
		CleanupInterval:        5 * time.Minute,
		SessionTimeout:         24 * time.Hour,
		MaxSessions:            10000,
		MaxMemoryMB:            100,
		EnablePersistence:      true,
		PersistencePath:        "./sessions",
		PersistenceFormat:      "json",
		PersistenceInterval:    30 * time.Second,
		EnableAsyncPersistence: true,
	}
}

// Validate checks if the configuration values are valid and sets defaults if needed.
func (c *ContextManagerConfig) Validate() error {
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = 5 * time.Minute
	}

	if c.SessionTimeout <= 0 {
		c.SessionTimeout = 24 * time.Hour
	}

	if c.MaxSessions <= 0 {
		c.MaxSessions = 10000
	}

	if c.MaxMemoryMB <= 0 {
		c.MaxMemoryMB = 100
	}

	if c.PersistencePath == "" {
		c.PersistencePath = "./sessions"
	}

	if c.PersistenceFormat != "json" && c.PersistenceFormat != "gob" {
		c.PersistenceFormat = "json"
	}

	if c.PersistenceInterval <= 0 {
		c.PersistenceInterval = 30 * time.Second
	}

	return nil
}