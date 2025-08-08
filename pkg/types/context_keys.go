package types

// ContextKey is a typed key for storing values in context.Context
type ContextKey string

// ContextKeyUserID is the key used to store authenticated user ID in context
const ContextKeyUserID ContextKey = "user_id"
