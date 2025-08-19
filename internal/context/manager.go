package context

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"genai-processing/internal/context/memory"
	"genai-processing/internal/context/persistence"
	"genai-processing/pkg/interfaces"
	"genai-processing/pkg/types"
)

// ContextManager implements the ContextManager interface with persistence and memory management.
// This implementation provides file persistence, LRU cache management, memory monitoring,
// and configurable parameters for production deployments.
type ContextManager struct {
	// Configuration
	config *ContextManagerConfig

	// Core session storage
	sessions map[string]*types.ConversationContext
	mu       sync.RWMutex

	// Memory management
	lruManager    *memory.LRUManager
	memoryMonitor *memory.MemoryMonitor

	// Persistence layer
	persistence persistence.PersistenceLayer

	// Background processes
	stopCleanup      chan bool
	stopPersistence  chan bool
	dirtySessions    map[string]bool // Track sessions needing persistence
	dirtySessionsMu  sync.RWMutex

	// Statistics
	stats ContextManagerStats
}

// ContextManagerStats provides comprehensive statistics about the context manager
type ContextManagerStats struct {
	// Session statistics
	TotalSessions    int     `json:"total_sessions"`
	ActiveSessions   int     `json:"active_sessions"`
	ExpiredSessions  int64   `json:"expired_sessions"`
	SessionsPerMinute float64 `json:"sessions_per_minute"`

	// Memory statistics
	MemoryUsageMB    float64 `json:"memory_usage_mb"`
	MemoryLimitMB    int     `json:"memory_limit_mb"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	EvictionCount    int64   `json:"eviction_count"`

	// Persistence statistics
	PersistenceOps   int64   `json:"persistence_ops"`
	PersistenceErrors int64  `json:"persistence_errors"`
	LastPersistTime  int64   `json:"last_persist_time"`
	PersistenceRate  float64 `json:"persistence_rate_per_minute"`

	// Performance statistics
	AvgUpdateTime    float64 `json:"avg_update_time_ms"`
	AvgResolveTime   float64 `json:"avg_resolve_time_ms"`
	UptimeSeconds    int64   `json:"uptime_seconds"`

	// Error statistics
	TotalErrors      int64   `json:"total_errors"`
	RecentErrors     []string `json:"recent_errors,omitempty"`
}

// NewContextManagerFull creates a new context manager with full feature set.
func NewContextManagerFull(config *ContextManagerConfig) (interfaces.ContextManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create persistence layer
	var persistenceLayer persistence.PersistenceLayer
	if config.EnablePersistence {
		persistenceConfig := persistence.PersistenceConfig{
			Type:   persistence.FileStorageType,
			Path:   config.PersistencePath,
			Format: config.PersistenceFormat,
		}
		
		var err error
		persistenceLayer, err = persistence.NewPersistenceLayer(persistenceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistence layer: %w", err)
		}
	}

	// Create LRU manager
	lruManager := memory.NewLRUManager(config.MaxSessions, config.MaxMemoryMB)

	// Create memory monitor
	memoryMonitor := memory.NewMemoryMonitor(config.MaxMemoryMB, 0.80, 0.95) // 80% warning, 95% critical

	cm := &ContextManager{
		config:           config,
		sessions:         make(map[string]*types.ConversationContext),
		lruManager:       lruManager,
		memoryMonitor:    memoryMonitor,
		persistence:      persistenceLayer,
		stopCleanup:      make(chan bool),
		stopPersistence:  make(chan bool),
		dirtySessions:    make(map[string]bool),
		stats:            ContextManagerStats{MemoryLimitMB: config.MaxMemoryMB},
	}

	// Set memory monitor callbacks
	memoryMonitor.SetCallbacks(
		cm.handleMemoryWarning,
		cm.handleMemoryCritical,
	)

	// Load persisted sessions on startup
	if err := cm.loadPersistedSessions(); err != nil {
		log.Printf("Warning: failed to load persisted sessions: %v", err)
	}

	// Start background processes
	go cm.runCleanup()
	if config.EnableAsyncPersistence && config.EnablePersistence {
		go cm.runAsyncPersistence()
	}
	memoryMonitor.Start()

	return cm, nil
}

// NewContextManagerWithConfig creates a new context manager with configuration (backward compatible).
func NewContextManagerWithConfig(config *ContextManagerConfig) interfaces.ContextManager {
	cm, err := NewContextManagerFull(config)
	if err != nil {
		log.Printf("Failed to create context manager: %v, falling back to basic configuration", err)
		// Fallback to basic configuration instead of basic manager
		fallbackConfig := DefaultConfig()
		fallbackConfig.EnablePersistence = false
		fallbackCM, fallbackErr := NewContextManagerFull(fallbackConfig)
		if fallbackErr != nil {
			log.Printf("Failed to create fallback context manager: %v", fallbackErr)
			return nil
		}
		return fallbackCM
	}
	return cm
}

// NewContextManager creates a new context manager (now uses clean implementation).
func NewContextManager() interfaces.ContextManager {
	config := DefaultConfig()
	config.EnablePersistence = false // Basic mode: no persistence
	cm, err := NewContextManagerFull(config)
	if err != nil {
		log.Printf("Failed to create context manager: %v", err)
		return nil
	}
	return cm
}

// UpdateContext updates the conversation context with new query and response data.
func (cm *ContextManager) UpdateContext(sessionID string, query string, response *types.StructuredQuery) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		cm.updatePerformanceStats("update", duration)
	}()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get or create session context
	context, exists := cm.sessions[sessionID]
	if !exists {
		context = types.NewConversationContext(sessionID, "")
		cm.sessions[sessionID] = context
		cm.stats.TotalSessions++
	} else {
		context.LastActivity = time.Now()
		context.ExtendExpiration(cm.config.SessionTimeout)
	}

	// Update LRU cache
	cm.lruManager.Put(sessionID, context)

	// Extract and store resolved references
	resolvedRefs := cm.extractReferencesFromResponse(response)

	// Add conversation entry
	context.AddConversationEntry(query, response, resolvedRefs)

	// Update resolved references
	cm.updateResolvedReferences(context, resolvedRefs)

	// Enrich context
	cm.enrichContext(context, query, response)

	// Mark session as dirty for persistence
	cm.markSessionDirty(sessionID)

	// Synchronous persistence if async is disabled
	if cm.config.EnablePersistence && !cm.config.EnableAsyncPersistence {
		if err := cm.persistence.SaveSession(sessionID, context); err != nil {
			cm.recordError(fmt.Sprintf("Failed to persist session %s: %v", sessionID, err))
		}
	}

	return nil
}

// UpdateContextWithUser updates context with user information.
func (cm *ContextManager) UpdateContextWithUser(sessionID string, userID string, query string, response *types.StructuredQuery) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	sanitizedUserID := sanitizeUserID(userID)

	context, exists := cm.sessions[sessionID]
	if !exists {
		context = types.NewConversationContext(sessionID, sanitizedUserID)
		cm.sessions[sessionID] = context
		cm.stats.TotalSessions++
	} else {
		if sanitizedUserID != "" && context.UserID != sanitizedUserID {
			context.UserID = sanitizedUserID
		}
		context.LastActivity = time.Now()
		context.ExtendExpiration(cm.config.SessionTimeout)
	}

	// Update LRU cache
	cm.lruManager.Put(sessionID, context)

	// Continue with normal update logic
	resolvedRefs := cm.extractReferencesFromResponse(response)
	context.AddConversationEntry(query, response, resolvedRefs)
	cm.updateResolvedReferences(context, resolvedRefs)
	cm.enrichContext(context, query, response)
	cm.markSessionDirty(sessionID)

	return nil
}

// ResolvePronouns resolves pronouns using conversation context.
func (cm *ContextManager) ResolvePronouns(query string, sessionID string) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		cm.updatePerformanceStats("resolve", duration)
	}()

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try LRU cache first
	if context, found := cm.lruManager.Get(sessionID); found {
		return cm.performPronounResolution(query, context), nil
	}

	// Fallback to main sessions map
	context, exists := cm.sessions[sessionID]
	if !exists {
		return query, nil
	}

	return cm.performPronounResolution(query, context), nil
}

// GetContext retrieves conversation context.
func (cm *ContextManager) GetContext(sessionID string) (*types.ConversationContext, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try LRU cache first
	if context, found := cm.lruManager.Get(sessionID); found {
		return context, nil
	}

	// Fallback to main sessions map
	context, exists := cm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return context, nil
}

// GetStats returns comprehensive statistics about the context manager.
func (cm *ContextManager) GetStats() ContextManagerStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := cm.stats
	stats.ActiveSessions = len(cm.sessions)

	// Get LRU statistics
	lruStats := cm.lruManager.GetStats()
	stats.CacheHitRate = lruStats.HitRate
	stats.EvictionCount = lruStats.EvictionCount
	stats.MemoryUsageMB = float64(lruStats.MemoryUsageKB) / 1024

	// Get memory statistics
	memStats := cm.memoryMonitor.GetStats()
	stats.MemoryUsageMB = memStats.HeapAllocMB

	return stats
}

// Close stops all background processes and cleans up resources.
func (cm *ContextManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Stop background processes
	close(cm.stopCleanup)
	if cm.config.EnableAsyncPersistence {
		close(cm.stopPersistence)
	}
	cm.memoryMonitor.Stop()

	// Final persistence of dirty sessions
	if cm.config.EnablePersistence {
		cm.persistDirtySessions()
		if cm.persistence != nil {
			cm.persistence.Close()
		}
	}

	return nil
}

// Private helper methods

func (cm *ContextManager) loadPersistedSessions() error {
	if cm.persistence == nil {
		return nil
	}

	sessions, err := cm.persistence.LoadAllSessions()
	if err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for sessionID, context := range sessions {
		cm.sessions[sessionID] = context
		cm.lruManager.Put(sessionID, context)
	}

	log.Printf("Loaded %d persisted sessions", len(sessions))
	return nil
}

func (cm *ContextManager) markSessionDirty(sessionID string) {
	cm.dirtySessionsMu.Lock()
	defer cm.dirtySessionsMu.Unlock()
	cm.dirtySessions[sessionID] = true
}

func (cm *ContextManager) persistDirtySessions() {
	if cm.persistence == nil {
		return
	}

	cm.dirtySessionsMu.Lock()
	dirtyList := make([]string, 0, len(cm.dirtySessions))
	for sessionID := range cm.dirtySessions {
		dirtyList = append(dirtyList, sessionID)
	}
	cm.dirtySessions = make(map[string]bool) // Clear dirty list
	cm.dirtySessionsMu.Unlock()

	for _, sessionID := range dirtyList {
		if context, exists := cm.sessions[sessionID]; exists {
			if err := cm.persistence.SaveSession(sessionID, context); err != nil {
				cm.recordError(fmt.Sprintf("Failed to persist session %s: %v", sessionID, err))
				// Re-mark as dirty
				cm.dirtySessionsMu.Lock()
				cm.dirtySessions[sessionID] = true
				cm.dirtySessionsMu.Unlock()
			} else {
				cm.stats.PersistenceOps++
				cm.stats.LastPersistTime = time.Now().Unix()
			}
		}
	}
}

func (cm *ContextManager) runAsyncPersistence() {
	ticker := time.NewTicker(cm.config.PersistenceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.persistDirtySessions()
		case <-cm.stopPersistence:
			return
		}
	}
}

func (cm *ContextManager) runCleanup() {
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanupExpiredSessions()
		case <-cm.stopCleanup:
			return
		}
	}
}

func (cm *ContextManager) cleanupExpiredSessions() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	expiredSessions := make([]string, 0)
	for sessionID, context := range cm.sessions {
		if context.IsExpired() {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		delete(cm.sessions, sessionID)
		cm.lruManager.Remove(sessionID)
		
		// Clean up from persistence
		if cm.persistence != nil {
			cm.persistence.DeleteSession(sessionID)
		}
	}

	if len(expiredSessions) > 0 {
		cm.stats.ExpiredSessions += int64(len(expiredSessions))
		log.Printf("Cleaned up %d expired sessions", len(expiredSessions))
	}
}

func (cm *ContextManager) handleMemoryWarning(stats memory.MemoryStats) {
	log.Printf("Memory warning: %.1f%% usage (%.1fMB)", stats.MemoryUsagePercent, stats.UsedMemoryMB)
	// Trigger more aggressive cleanup
	cm.cleanupExpiredSessions()
}

func (cm *ContextManager) handleMemoryCritical(stats memory.MemoryStats) {
	log.Printf("Critical memory usage: %.1f%% usage (%.1fMB)", stats.MemoryUsagePercent, stats.UsedMemoryMB)
	// Force garbage collection and aggressive session eviction
	cm.memoryMonitor.ForceGC()
	cm.cleanupExpiredSessions()
}

func (cm *ContextManager) updatePerformanceStats(operation string, duration time.Duration) {
	durationMs := float64(duration.Nanoseconds()) / 1000000

	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch operation {
	case "update":
		if cm.stats.AvgUpdateTime == 0 {
			cm.stats.AvgUpdateTime = durationMs
		} else {
			cm.stats.AvgUpdateTime = (cm.stats.AvgUpdateTime + durationMs) / 2
		}
	case "resolve":
		if cm.stats.AvgResolveTime == 0 {
			cm.stats.AvgResolveTime = durationMs
		} else {
			cm.stats.AvgResolveTime = (cm.stats.AvgResolveTime + durationMs) / 2
		}
	}
}

func (cm *ContextManager) recordError(errorMsg string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats.TotalErrors++
	cm.stats.PersistenceErrors++

	// Keep last 10 errors
	cm.stats.RecentErrors = append(cm.stats.RecentErrors, errorMsg)
	if len(cm.stats.RecentErrors) > 10 {
		cm.stats.RecentErrors = cm.stats.RecentErrors[1:]
	}

	log.Printf("Context manager error: %s", errorMsg)
}

func (cm *ContextManager) performPronounResolution(query string, context *types.ConversationContext) string {
	resolved := query

	// Use the same resolution logic from the original manager
	resolved = cm.resolveUserPronouns(resolved, context)
	resolved = cm.resolveResourceReferences(resolved, context)
	resolved = cm.resolveTimeReferences(resolved, context)
	resolved = cm.resolveActionReferences(resolved, context)

	return resolved
}

// Include all the helper methods from the original manager
func (cm *ContextManager) extractReferencesFromResponse(response *types.StructuredQuery) map[string]string {
	refs := make(map[string]string)

	if !response.User.IsEmpty() {
		if response.User.IsString() {
			refs["last_user"] = response.User.GetString()
		} else if response.User.IsArray() && len(response.User.GetArray()) > 0 {
			refs["last_user"] = response.User.GetArray()[0]
		}
	}

	if !response.Resource.IsEmpty() {
		if response.Resource.IsString() {
			refs["last_resource"] = response.Resource.GetString()
		} else if response.Resource.IsArray() && len(response.Resource.GetArray()) > 0 {
			refs["last_resource"] = response.Resource.GetArray()[0]
		}
	}

	if !response.Namespace.IsEmpty() {
		if response.Namespace.IsString() {
			refs["last_namespace"] = response.Namespace.GetString()
		} else if response.Namespace.IsArray() && len(response.Namespace.GetArray()) > 0 {
			refs["last_namespace"] = response.Namespace.GetArray()[0]
		}
	}

	if !response.Verb.IsEmpty() {
		if response.Verb.IsString() {
			refs["last_action"] = response.Verb.GetString()
		} else if response.Verb.IsArray() && len(response.Verb.GetArray()) > 0 {
			refs["last_action"] = response.Verb.GetArray()[0]
		}
	}

	if response.Timeframe != "" {
		refs["last_timeframe"] = response.Timeframe
	}

	if response.ResourceNamePattern != "" {
		refs["last_resource_name"] = response.ResourceNamePattern
	}

	return refs
}

func (cm *ContextManager) updateResolvedReferences(context *types.ConversationContext, refs map[string]string) {
	for key, value := range refs {
		if value != "" {
			refType := "unknown"
			switch {
			case strings.HasPrefix(key, "last_user"):
				refType = "user"
			case strings.HasPrefix(key, "last_resource"):
				refType = "resource"
			case strings.HasPrefix(key, "last_namespace"):
				refType = "namespace"
			case strings.HasPrefix(key, "last_action"):
				refType = "action"
			case strings.HasPrefix(key, "last_timeframe"):
				refType = "time"
			case strings.HasPrefix(key, "last_resource_name"):
				refType = "resource_name"
			}

			context.UpdateResolvedReference(key, refType, value, 0.9)
		}
	}
}

func (cm *ContextManager) enrichContext(context *types.ConversationContext, query string, response *types.StructuredQuery) {
	context.ContextEnrichment["query_patterns"] = cm.extractQueryPatterns(query)
	context.ContextEnrichment["last_response_summary"] = cm.createResponseSummary(response)
	context.ContextEnrichment["conversation_flow"] = cm.analyzeConversationFlow(context)
}

func (cm *ContextManager) resolveUserPronouns(query string, context *types.ConversationContext) string {
	resolved := query

	userPatterns := map[string]string{
		`\bhe\b`:            "last_user",
		`\bshe\b`:           "last_user",
		`\bthat user\b`:     "last_user",
		`\bthe user\b`:      "last_user",
		`\bthis user\b`:     "last_user",
		`\bthe same user\b`: "last_user",
	}

	for pattern, refKey := range userPatterns {
		if ref, exists := context.GetResolvedReference(refKey); exists && ref.Value != "" {
			re := regexp.MustCompile(pattern)
			resolved = re.ReplaceAllString(resolved, ref.Value)
		}
	}

	return resolved
}

func (cm *ContextManager) resolveResourceReferences(query string, context *types.ConversationContext) string {
	resolved := query

	resourcePatterns := map[string]string{
		`\bit\b`:                "last_resource",
		`\bthat resource\b`:     "last_resource",
		`\bthe resource\b`:      "last_resource",
		`\bthis resource\b`:     "last_resource",
		`\bthe same resource\b`: "last_resource",
		`\bthat CRD\b`:          "last_resource_name",
		`\bthe CRD\b`:           "last_resource_name",
		`\bthis CRD\b`:          "last_resource_name",
	}

	for pattern, refKey := range resourcePatterns {
		if ref, exists := context.GetResolvedReference(refKey); exists && ref.Value != "" {
			re := regexp.MustCompile(pattern)
			resolved = re.ReplaceAllString(resolved, ref.Value)
		}
	}

	return resolved
}

func (cm *ContextManager) resolveTimeReferences(query string, context *types.ConversationContext) string {
	resolved := query

	timePatterns := map[string]string{
		`\baround that time\b`: "last_timeframe",
		`\bat that time\b`:     "last_timeframe",
		`\bthen\b`:             "last_timeframe",
	}

	for pattern, refKey := range timePatterns {
		if ref, exists := context.GetResolvedReference(refKey); exists && ref.Value != "" {
			re := regexp.MustCompile(pattern)
			resolved = re.ReplaceAllString(resolved, ref.Value)
		}
	}

	return resolved
}

func (cm *ContextManager) resolveActionReferences(query string, context *types.ConversationContext) string {
	resolved := query

	actionPatterns := map[string]string{
		`\bthat action\b`: "last_action",
		`\bthe action\b`:  "last_action",
		`\bthis action\b`: "last_action",
	}

	for pattern, refKey := range actionPatterns {
		if ref, exists := context.GetResolvedReference(refKey); exists && ref.Value != "" {
			re := regexp.MustCompile(pattern)
			resolved = re.ReplaceAllString(resolved, ref.Value)
		}
	}

	return resolved
}

func (cm *ContextManager) extractQueryPatterns(query string) map[string]interface{} {
	patterns := make(map[string]interface{})

	questionWords := []string{"who", "what", "when", "where", "why", "how"}
	for _, word := range questionWords {
		if strings.Contains(strings.ToLower(query), word) {
			patterns["question_type"] = word
			break
		}
	}

	actionWords := []string{"deleted", "created", "modified", "accessed", "failed", "succeeded"}
	for _, word := range actionWords {
		if strings.Contains(strings.ToLower(query), word) {
			patterns["action_type"] = word
			break
		}
	}

	timeIndicators := []string{"yesterday", "today", "last week", "this week", "hour", "day", "week"}
	for _, indicator := range timeIndicators {
		if strings.Contains(strings.ToLower(query), indicator) {
			patterns["time_indicator"] = indicator
			break
		}
	}

	return patterns
}

func (cm *ContextManager) createResponseSummary(response *types.StructuredQuery) map[string]interface{} {
	summary := make(map[string]interface{})

	if response.LogSource != "" {
		summary["log_source"] = response.LogSource
	}

	if !response.Verb.IsEmpty() {
		summary["verb"] = response.Verb.GetValue()
	}

	if !response.Resource.IsEmpty() {
		summary["resource"] = response.Resource.GetValue()
	}

	if response.Timeframe != "" {
		summary["timeframe"] = response.Timeframe
	}

	if response.Limit > 0 {
		summary["limit"] = response.Limit
	}

	return summary
}

func (cm *ContextManager) analyzeConversationFlow(context *types.ConversationContext) map[string]interface{} {
	flow := make(map[string]interface{})

	flow["total_interactions"] = len(context.ConversationHistory)
	flow["session_duration"] = time.Since(context.CreatedAt).String()

	if len(context.ConversationHistory) > 0 {
		flow["last_interaction"] = context.ConversationHistory[len(context.ConversationHistory)-1].Timestamp
	}

	userQueries := 0
	resourceQueries := 0
	timeQueries := 0

	for _, entry := range context.ConversationHistory {
		query := strings.ToLower(entry.Query)
		if strings.Contains(query, "who") || strings.Contains(query, "user") {
			userQueries++
		}
		if strings.Contains(query, "what") || strings.Contains(query, "resource") || strings.Contains(query, "crd") {
			resourceQueries++
		}
		if strings.Contains(query, "when") || strings.Contains(query, "time") {
			timeQueries++
		}
	}

	flow["user_focus"] = userQueries > 0
	flow["resource_focus"] = resourceQueries > 0
	flow["time_focus"] = timeQueries > 0

	return flow
}

// Backward compatibility functions to maintain existing interface

// GetSessionCount returns the number of active sessions for monitoring purposes.
func (cm *ContextManager) GetSessionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.sessions)
}

// ClearAllSessions removes all sessions from memory and persistence.
func (cm *ContextManager) ClearAllSessions() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.sessions = make(map[string]*types.ConversationContext)
	cm.lruManager.Clear()
	
	cm.dirtySessionsMu.Lock()
	cm.dirtySessions = make(map[string]bool)
	cm.dirtySessionsMu.Unlock()
}

// sanitizeUserID performs basic validation and sanitization of a user ID.
// It strips whitespace and disallows control characters; returns empty string if invalid.
func sanitizeUserID(userID string) string {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return ""
	}
	// Basic sanity checks: length and allowed characters (alphanumerics plus .-_@)
	if len(trimmed) > 256 {
		return ""
	}
	for _, r := range trimmed {
		if r <= 31 || r == 127 { // control characters
			return ""
		}
		if !(r == '.' || r == '-' || r == '_' || r == '@' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return ""
		}
	}
	return trimmed
}
