package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConversationContextValidation(t *testing.T) {
	now := time.Now()

	t.Run("valid conversation context", func(t *testing.T) {
		context := ConversationContext{
			SessionID:    "session-123",
			UserID:       "user-456",
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test JSON round-trip
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context: %v", err)
		}

		// Verify round-trip preserved all values
		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("SessionID round-trip failed: expected %s, got %s", context.SessionID, unmarshaled.SessionID)
		}
		if context.UserID != unmarshaled.UserID {
			t.Errorf("UserID round-trip failed: expected %s, got %s", context.UserID, unmarshaled.UserID)
		}
		if !context.CreatedAt.Equal(unmarshaled.CreatedAt) {
			t.Errorf("CreatedAt round-trip failed: expected %v, got %v", context.CreatedAt, unmarshaled.CreatedAt)
		}
		if !context.LastActivity.Equal(unmarshaled.LastActivity) {
			t.Errorf("LastActivity round-trip failed: expected %v, got %v", context.LastActivity, unmarshaled.LastActivity)
		}
	})

	t.Run("conversation context with different timestamps", func(t *testing.T) {
		createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		lastActivity := time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC)

		context := ConversationContext{
			SessionID:    "session-789",
			UserID:       "user-101",
			CreatedAt:    createdAt,
			LastActivity: lastActivity,
		}

		// Test that timestamps are properly handled
		if !context.CreatedAt.Before(context.LastActivity) {
			t.Errorf("CreatedAt should be before LastActivity")
		}

		// Test JSON round-trip
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with different timestamps: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with different timestamps: %v", err)
		}

		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("SessionID round-trip failed: expected %s, got %s", context.SessionID, unmarshaled.SessionID)
		}
		if context.UserID != unmarshaled.UserID {
			t.Errorf("UserID round-trip failed: expected %s, got %s", context.UserID, unmarshaled.UserID)
		}
		if !context.CreatedAt.Equal(unmarshaled.CreatedAt) {
			t.Errorf("CreatedAt round-trip failed: expected %v, got %v", context.CreatedAt, unmarshaled.CreatedAt)
		}
		if !context.LastActivity.Equal(unmarshaled.LastActivity) {
			t.Errorf("LastActivity round-trip failed: expected %v, got %v", context.LastActivity, unmarshaled.LastActivity)
		}
	})

	t.Run("conversation context with empty fields", func(t *testing.T) {
		context := ConversationContext{
			SessionID:    "",
			UserID:       "",
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test JSON round-trip with empty strings
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with empty fields: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with empty fields: %v", err)
		}

		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("SessionID round-trip failed: expected %s, got %s", context.SessionID, unmarshaled.SessionID)
		}
		if context.UserID != unmarshaled.UserID {
			t.Errorf("UserID round-trip failed: expected %s, got %s", context.UserID, unmarshaled.UserID)
		}
	})

	t.Run("conversation context with zero time", func(t *testing.T) {
		context := ConversationContext{
			SessionID:    "session-123",
			UserID:       "user-456",
			CreatedAt:    time.Time{},
			LastActivity: time.Time{},
		}

		// Test that zero times are handled correctly
		if !context.CreatedAt.IsZero() {
			t.Errorf("CreatedAt should be zero time")
		}
		if !context.LastActivity.IsZero() {
			t.Errorf("LastActivity should be zero time")
		}

		// Test JSON round-trip with zero times
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with zero times: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with zero times: %v", err)
		}

		if !unmarshaled.CreatedAt.IsZero() {
			t.Errorf("unmarshaled CreatedAt should be zero time")
		}
		if !unmarshaled.LastActivity.IsZero() {
			t.Errorf("unmarshaled LastActivity should be zero time")
		}
	})
}

func TestConversationContextBusinessLogic(t *testing.T) {
	t.Run("session creation time validation", func(t *testing.T) {
		createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		lastActivity := time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC)

		context := ConversationContext{
			CreatedAt:    createdAt,
			LastActivity: lastActivity,
		}

		// Test that creation time is before or equal to last activity
		if context.CreatedAt.After(context.LastActivity) {
			t.Errorf("CreatedAt should not be after LastActivity")
		}
	})

	t.Run("session creation time equals last activity", func(t *testing.T) {
		now := time.Now()
		context := ConversationContext{
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test that equal times are valid
		if !context.CreatedAt.Equal(context.LastActivity) {
			t.Errorf("CreatedAt should equal LastActivity when session is new")
		}
	})
}

func TestConversationContextEdgeCases(t *testing.T) {
	t.Run("very long session ID", func(t *testing.T) {
		longSessionID := "session-" + string(make([]byte, 1000)) // Very long session ID
		now := time.Now()

		context := ConversationContext{
			SessionID:    longSessionID,
			UserID:       "user-456",
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test that long session ID is handled correctly
		if len(context.SessionID) != len(longSessionID) {
			t.Errorf("long session ID length mismatch: expected %d, got %d", len(longSessionID), len(context.SessionID))
		}

		// Test JSON round-trip with long session ID
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with long session ID: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with long session ID: %v", err)
		}

		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("long SessionID round-trip failed")
		}
	})

	t.Run("special characters in IDs", func(t *testing.T) {
		now := time.Now()
		context := ConversationContext{
			SessionID:    "session-123!@#$%^&*()",
			UserID:       "user-456!@#$%^&*()",
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test JSON round-trip with special characters
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with special characters: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with special characters: %v", err)
		}

		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("SessionID with special characters round-trip failed: expected %s, got %s", context.SessionID, unmarshaled.SessionID)
		}
		if context.UserID != unmarshaled.UserID {
			t.Errorf("UserID with special characters round-trip failed: expected %s, got %s", context.UserID, unmarshaled.UserID)
		}
	})

	t.Run("UUID-like session ID", func(t *testing.T) {
		now := time.Now()
		context := ConversationContext{
			SessionID:    "550e8400-e29b-41d4-a716-446655440000",
			UserID:       "user-456",
			CreatedAt:    now,
			LastActivity: now,
		}

		// Test that UUID-like session ID is handled correctly
		if len(context.SessionID) != 36 {
			t.Errorf("UUID-like session ID should be 36 characters long, got %d", len(context.SessionID))
		}

		// Test JSON round-trip
		data, err := json.Marshal(context)
		if err != nil {
			t.Fatalf("failed to marshal context with UUID-like session ID: %v", err)
		}

		var unmarshaled ConversationContext
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal context with UUID-like session ID: %v", err)
		}

		if context.SessionID != unmarshaled.SessionID {
			t.Errorf("UUID-like SessionID round-trip failed: expected %s, got %s", context.SessionID, unmarshaled.SessionID)
		}
	})
}
