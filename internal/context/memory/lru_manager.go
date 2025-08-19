package memory

import (
	"sync"
	"time"

	"genai-processing/pkg/types"
)

// LRUManager manages session memory using a Least Recently Used cache strategy.
// It provides memory limits and automatic eviction of old sessions.
type LRUManager struct {
	maxSessions int
	maxMemoryMB int
	sessions    map[string]*SessionEntry
	accessOrder []string // Maintains LRU order
	mu          sync.RWMutex
	stats       LRUStats
}

// SessionEntry wraps a conversation context with metadata for LRU tracking
type SessionEntry struct {
	Context    *types.ConversationContext
	LastAccess time.Time
	MemorySize int64 // Estimated memory size in bytes
}

// LRUStats provides statistics about LRU cache performance
type LRUStats struct {
	TotalSessions   int     `json:"total_sessions"`
	MemoryUsageKB   int64   `json:"memory_usage_kb"`
	MemoryLimitKB   int64   `json:"memory_limit_kb"`
	EvictionCount   int64   `json:"eviction_count"`
	HitRate         float64 `json:"hit_rate"`
	TotalAccesses   int64   `json:"total_accesses"`
	CacheHits       int64   `json:"cache_hits"`
	LastEviction    int64   `json:"last_eviction,omitempty"`
	OldestSessionAge int64  `json:"oldest_session_age_seconds"`
}

// NewLRUManager creates a new LRU manager with specified limits
func NewLRUManager(maxSessions int, maxMemoryMB int) *LRUManager {
	return &LRUManager{
		maxSessions: maxSessions,
		maxMemoryMB: maxMemoryMB,
		sessions:    make(map[string]*SessionEntry),
		accessOrder: make([]string, 0),
		stats: LRUStats{
			MemoryLimitKB: int64(maxMemoryMB * 1024),
		},
	}
}

// Put adds or updates a session in the LRU cache
func (lru *LRUManager) Put(sessionID string, context *types.ConversationContext) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	now := time.Now()
	memSize := lru.estimateMemorySize(context)

	// Update existing session
	if entry, exists := lru.sessions[sessionID]; exists {
		entry.Context = context
		entry.LastAccess = now
		entry.MemorySize = memSize
		lru.moveToFront(sessionID)
		return
	}

	// Add new session
	entry := &SessionEntry{
		Context:    context,
		LastAccess: now,
		MemorySize: memSize,
	}

	lru.sessions[sessionID] = entry
	lru.accessOrder = append([]string{sessionID}, lru.accessOrder...)
	lru.stats.TotalSessions++

	// Enforce limits
	lru.enforceLimits()
}

// Get retrieves a session from the LRU cache
func (lru *LRUManager) Get(sessionID string) (*types.ConversationContext, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.stats.TotalAccesses++

	entry, exists := lru.sessions[sessionID]
	if !exists {
		// Update hit rate even for misses
		lru.stats.HitRate = float64(lru.stats.CacheHits) / float64(lru.stats.TotalAccesses)
		return nil, false
	}

	// Update access time and move to front
	entry.LastAccess = time.Now()
	lru.moveToFront(sessionID)
	lru.stats.CacheHits++
	lru.stats.HitRate = float64(lru.stats.CacheHits) / float64(lru.stats.TotalAccesses)

	return entry.Context, true
}

// Remove removes a session from the LRU cache
func (lru *LRUManager) Remove(sessionID string) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if _, exists := lru.sessions[sessionID]; !exists {
		return false
	}

	delete(lru.sessions, sessionID)
	lru.removeFromOrder(sessionID)
	lru.stats.TotalSessions--

	return true
}

// GetAllSessions returns all sessions in the cache
func (lru *LRUManager) GetAllSessions() map[string]*types.ConversationContext {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	result := make(map[string]*types.ConversationContext)
	for sessionID, entry := range lru.sessions {
		result[sessionID] = entry.Context
	}

	return result
}

// Clear removes all sessions from the cache
func (lru *LRUManager) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.sessions = make(map[string]*SessionEntry)
	lru.accessOrder = make([]string, 0)
	lru.stats.TotalSessions = 0
}

// GetStats returns current LRU cache statistics
func (lru *LRUManager) GetStats() LRUStats {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	stats := lru.stats
	stats.MemoryUsageKB = lru.calculateMemoryUsage()
	stats.TotalSessions = len(lru.sessions)

	if len(lru.accessOrder) > 0 {
		oldestSessionID := lru.accessOrder[len(lru.accessOrder)-1]
		if entry, exists := lru.sessions[oldestSessionID]; exists {
			stats.OldestSessionAge = int64(time.Since(entry.LastAccess).Seconds())
		}
	}

	return stats
}

// enforceLimits removes old sessions if limits are exceeded
func (lru *LRUManager) enforceLimits() {
	// Enforce session count limit
	for len(lru.sessions) > lru.maxSessions && len(lru.accessOrder) > 0 {
		lru.evictOldest()
	}

	// Enforce memory limit
	memoryUsageKB := lru.calculateMemoryUsage()
	memoryLimitKB := int64(lru.maxMemoryMB * 1024)

	for memoryUsageKB > memoryLimitKB && len(lru.accessOrder) > 0 {
		lru.evictOldest()
		memoryUsageKB = lru.calculateMemoryUsage()
	}
}

// evictOldest removes the least recently used session
func (lru *LRUManager) evictOldest() {
	if len(lru.accessOrder) == 0 {
		return
	}

	// Remove the last (oldest) session
	oldestSessionID := lru.accessOrder[len(lru.accessOrder)-1]
	delete(lru.sessions, oldestSessionID)
	lru.accessOrder = lru.accessOrder[:len(lru.accessOrder)-1]

	lru.stats.EvictionCount++
	lru.stats.LastEviction = time.Now().Unix()
	lru.stats.TotalSessions--
}

// moveToFront moves a session to the front of the access order
func (lru *LRUManager) moveToFront(sessionID string) {
	// Remove from current position
	lru.removeFromOrder(sessionID)
	// Add to front
	lru.accessOrder = append([]string{sessionID}, lru.accessOrder...)
}

// removeFromOrder removes a session from the access order slice
func (lru *LRUManager) removeFromOrder(sessionID string) {
	for i, id := range lru.accessOrder {
		if id == sessionID {
			lru.accessOrder = append(lru.accessOrder[:i], lru.accessOrder[i+1:]...)
			break
		}
	}
}

// calculateMemoryUsage estimates total memory usage in KB
func (lru *LRUManager) calculateMemoryUsage() int64 {
	var total int64
	for _, entry := range lru.sessions {
		total += entry.MemorySize
	}
	return total / 1024 // Convert to KB
}

// estimateMemorySize estimates the memory size of a conversation context
func (lru *LRUManager) estimateMemorySize(ctx *types.ConversationContext) int64 {
	if ctx == nil {
		return 0
	}

	var size int64

	// Base struct size
	size += 200 // Approximate base struct overhead

	// Session ID and User ID
	size += int64(len(ctx.SessionID) + len(ctx.UserID))

	// Conversation history
	for _, entry := range ctx.ConversationHistory {
		size += int64(len(entry.Query))
		size += 100 // Approximate struct overhead for response
		
		// Estimate response size (basic estimation)
		if entry.Response != nil {
			size += int64(len(entry.Response.LogSource))
			size += int64(len(entry.Response.Timeframe))
			size += int64(len(entry.Response.ResourceNamePattern))
			size += 200 // Other fields overhead
		}
	}

	// Resolved references
	for key, ref := range ctx.ResolvedReferences {
		size += int64(len(key) + len(ref.Value) + len(ref.Type))
		size += 50 // Struct overhead
	}

	// Context enrichment
	size += int64(len(ctx.ContextEnrichment) * 100) // Rough estimate

	return size
}