package context

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test default values
	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("Expected CleanupInterval 5m, got %v", config.CleanupInterval)
	}

	if config.SessionTimeout != 24*time.Hour {
		t.Errorf("Expected SessionTimeout 24h, got %v", config.SessionTimeout)
	}

	if config.MaxSessions != 10000 {
		t.Errorf("Expected MaxSessions 10000, got %d", config.MaxSessions)
	}

	if config.MaxMemoryMB != 100 {
		t.Errorf("Expected MaxMemoryMB 100, got %d", config.MaxMemoryMB)
	}

	if !config.EnablePersistence {
		t.Error("Expected EnablePersistence to be true")
	}

	if config.PersistencePath != "./sessions" {
		t.Errorf("Expected PersistencePath './sessions', got %s", config.PersistencePath)
	}

	if config.PersistenceFormat != "json" {
		t.Errorf("Expected PersistenceFormat 'json', got %s", config.PersistenceFormat)
	}

	if config.PersistenceInterval != 30*time.Second {
		t.Errorf("Expected PersistenceInterval 30s, got %v", config.PersistenceInterval)
	}

	if !config.EnableAsyncPersistence {
		t.Error("Expected EnableAsyncPersistence to be true")
	}
}

func TestConfigValidate_ValidConfig(t *testing.T) {
	config := &ContextManagerConfig{
		CleanupInterval:        10 * time.Minute,
		SessionTimeout:         12 * time.Hour,
		MaxSessions:            5000,
		MaxMemoryMB:            50,
		EnablePersistence:      false,
		PersistencePath:        "/tmp/sessions",
		PersistenceFormat:      "json",
		PersistenceInterval:    60 * time.Second,
		EnableAsyncPersistence: false,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() failed for valid config: %v", err)
	}

	// Values should remain unchanged
	if config.CleanupInterval != 10*time.Minute {
		t.Errorf("CleanupInterval changed during validation")
	}
	if config.SessionTimeout != 12*time.Hour {
		t.Errorf("SessionTimeout changed during validation")
	}
	if config.MaxSessions != 5000 {
		t.Errorf("MaxSessions changed during validation")
	}
	if config.MaxMemoryMB != 50 {
		t.Errorf("MaxMemoryMB changed during validation")
	}
	if config.PersistencePath != "/tmp/sessions" {
		t.Errorf("PersistencePath changed during validation")
	}
	if config.PersistenceFormat != "json" {
		t.Errorf("PersistenceFormat changed during validation")
	}
}

func TestConfigValidate_InvalidValues(t *testing.T) {
	testCases := []struct {
		name           string
		config         *ContextManagerConfig
		expectedFields map[string]interface{}
	}{
		{
			name: "Zero and negative values",
			config: &ContextManagerConfig{
				CleanupInterval:     0,
				SessionTimeout:      -1 * time.Hour,
				MaxSessions:         -1,
				MaxMemoryMB:         0,
				PersistencePath:     "",
				PersistenceFormat:   "invalid",
				PersistenceInterval: -1 * time.Second,
			},
			expectedFields: map[string]interface{}{
				"CleanupInterval":     5 * time.Minute,
				"SessionTimeout":      24 * time.Hour,
				"MaxSessions":         10000,
				"MaxMemoryMB":         100,
				"PersistencePath":     "./sessions",
				"PersistenceFormat":   "json",
				"PersistenceInterval": 30 * time.Second,
			},
		},
		{
			name: "Empty and invalid format",
			config: &ContextManagerConfig{
				CleanupInterval:     1 * time.Minute,   // Valid
				SessionTimeout:      1 * time.Hour,     // Valid
				MaxSessions:         1000,              // Valid
				MaxMemoryMB:         10,                // Valid
				PersistencePath:     "",                // Invalid - empty
				PersistenceFormat:   "xml",             // Invalid - not json/gob
				PersistenceInterval: 10 * time.Second,  // Valid
			},
			expectedFields: map[string]interface{}{
				"PersistencePath":   "./sessions",
				"PersistenceFormat": "json",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if err != nil {
				t.Errorf("Validate() returned error: %v", err)
			}

			// Check that invalid values were corrected
			for field, expected := range tc.expectedFields {
				switch field {
				case "CleanupInterval":
					if tc.config.CleanupInterval != expected.(time.Duration) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.CleanupInterval)
					}
				case "SessionTimeout":
					if tc.config.SessionTimeout != expected.(time.Duration) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.SessionTimeout)
					}
				case "MaxSessions":
					if tc.config.MaxSessions != expected.(int) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.MaxSessions)
					}
				case "MaxMemoryMB":
					if tc.config.MaxMemoryMB != expected.(int) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.MaxMemoryMB)
					}
				case "PersistencePath":
					if tc.config.PersistencePath != expected.(string) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.PersistencePath)
					}
				case "PersistenceFormat":
					if tc.config.PersistenceFormat != expected.(string) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.PersistenceFormat)
					}
				case "PersistenceInterval":
					if tc.config.PersistenceInterval != expected.(time.Duration) {
						t.Errorf("Expected %s to be corrected to %v, got %v", field, expected, tc.config.PersistenceInterval)
					}
				}
			}
		})
	}
}

func TestConfigValidate_GobFormat(t *testing.T) {
	config := &ContextManagerConfig{
		CleanupInterval:        5 * time.Minute,
		SessionTimeout:         24 * time.Hour,
		MaxSessions:            10000,
		MaxMemoryMB:            100,
		EnablePersistence:      true,
		PersistencePath:        "./sessions",
		PersistenceFormat:      "gob",
		PersistenceInterval:    30 * time.Second,
		EnableAsyncPersistence: true,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() failed for config with gob format: %v", err)
	}

	if config.PersistenceFormat != "gob" {
		t.Errorf("Expected PersistenceFormat to remain 'gob', got %s", config.PersistenceFormat)
	}
}