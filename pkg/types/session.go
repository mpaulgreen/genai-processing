package types

import "time"

// ConversationContext represents the context for a conversation session.
// This struct maintains the basic state and context information needed for multi-turn conversations.
type ConversationContext struct {
	// SessionID is the unique identifier for this conversation session
	SessionID string `json:"session_id"`

	// UserID is the identifier of the user participating in the conversation
	UserID string `json:"user_id"`

	// CreatedAt is when this conversation session was created
	CreatedAt time.Time `json:"created_at"`

	// LastActivity is the timestamp of the last activity in this session
	LastActivity time.Time `json:"last_activity"`
}
