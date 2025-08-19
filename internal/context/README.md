# Context Manager Package

The `internal/context` package provides comprehensive conversation context management for the GenAI-powered audit query system. It features enterprise-grade session persistence, memory management, and advanced pronoun resolution capabilities.

## Overview

This package implements a sophisticated context management system that:
- Maintains conversation history across user sessions
- Resolves pronouns and references in natural language queries
- Provides file-based persistence for session recovery
- Implements LRU cache with memory management
- Supports configurable parameters for different environments

## Architecture

```
internal/context/
├── manager.go              # Primary context manager with all features
├── config.go               # Configuration structures and validation
├── persistence/            # Session persistence layer
│   ├── interface.go        # Persistence interface definition
│   ├── file_storage.go     # File-based storage implementation
│   └── factory.go          # Persistence layer factory
└── memory/                 # Memory management components
    ├── lru_manager.go      # LRU cache for session management
    └── memory_monitor.go   # Memory usage monitoring and alerts
```

## Components

### 1. Context Manager (`manager.go`)

The primary (and only) context management component with full feature set:

```go
// Create with custom configuration
config := DefaultConfig()
config.MaxSessions = 5000
config.EnablePersistence = true
config.PersistencePath = "/data/sessions"

cm, err := NewContextManagerWithConfig(config)
if err != nil {
    log.Fatal(err)
}
defer cm.(*ContextManager).Close()

// Or create with basic configuration (no persistence)
cm := NewContextManager()
defer cm.(*ContextManager).Close()
```

**Key Features:**
- File persistence with atomic writes
- LRU cache with automatic eviction
- Memory monitoring with callback alerts
- Configurable session limits and timeouts
- Asynchronous persistence for performance
- Comprehensive statistics and monitoring

### 2. Configuration Management (`config.go`)

Flexible configuration system supporting YAML and environment-specific settings:

```go
type ContextManagerConfig struct {
    CleanupInterval        time.Duration `yaml:"cleanup_interval"`
    SessionTimeout         time.Duration `yaml:"session_timeout"`
    MaxSessions           int           `yaml:"max_sessions"`
    MaxMemoryMB           int           `yaml:"max_memory_mb"`
    EnablePersistence     bool          `yaml:"enable_persistence"`
    PersistencePath       string        `yaml:"persistence_path"`
    PersistenceFormat     string        `yaml:"persistence_format"`
    PersistenceInterval   time.Duration `yaml:"persistence_interval"`
    EnableAsyncPersistence bool         `yaml:"enable_async_persistence"`
}
```

### 3. Persistence Layer (`persistence/`)

Pluggable persistence architecture supporting multiple backends:

```go
// File storage
config := persistence.PersistenceConfig{
    Type:   persistence.FileStorageType,
    Path:   "./sessions",
    Format: "json",
}

persistenceLayer, err := persistence.NewPersistenceLayer(config)
```

**Available Backends:**
- **File Storage**: JSON-based files with atomic writes
- **Memory Storage**: No persistence (testing/development)

**Features:**
- Atomic write operations prevent corruption
- Batch operations for efficiency
- Session metadata and statistics
- Configurable storage formats

### 4. Memory Management (`memory/`)

Advanced memory management with LRU cache and monitoring:

#### LRU Manager
```go
lru := memory.NewLRUManager(maxSessions, maxMemoryMB)

// Put and get sessions
lru.Put(sessionID, context)
context, found := lru.Get(sessionID)

// Get cache statistics
stats := lru.GetStats()
fmt.Printf("Hit rate: %.2f%%", stats.HitRate*100)
```

#### Memory Monitor
```go
monitor := memory.NewMemoryMonitor(maxMemoryMB, 0.8, 0.95)

// Set alert callbacks
monitor.SetCallbacks(
    func(stats memory.MemoryStats) {
        log.Printf("Warning: %.1f%% memory usage", stats.MemoryUsagePercent)
    },
    func(stats memory.MemoryStats) {
        log.Printf("Critical: %.1f%% memory usage", stats.MemoryUsagePercent)
    },
)

monitor.Start()
```

## API Reference

### Basic Operations

```go
// Update conversation context
err := cm.UpdateContext(sessionID, query, response)

// Update with user information
err := cm.UpdateContextWithUser(sessionID, userID, query, response)

// Resolve pronouns in queries
resolved, err := cm.ResolvePronouns("When did he do it?", sessionID)

// Get conversation context
context, err := cm.GetContext(sessionID)
```

### Pronoun Resolution

The system automatically resolves various types of references:

```go
// User pronouns
"When did he do it?" → "When did john.doe do it?"

// Resource references  
"Show me that CRD" → "Show me customer-crd"

// Time references
"What happened around that time?" → "What happened yesterday?"

// Action references
"Who performed that action?" → "Who performed delete?"
```

### Statistics and Monitoring

```go
// Get context manager statistics
stats := cm.(*ContextManager).GetStats()
fmt.Printf("Active sessions: %d", stats.ActiveSessions)
fmt.Printf("Memory usage: %.1f MB", stats.MemoryUsageMB)
fmt.Printf("Cache hit rate: %.2f%%", stats.CacheHitRate*100)
fmt.Printf("Persistence operations: %d", stats.PersistenceOps)

// Get LRU cache statistics  
lruStats := lru.GetStats()
fmt.Printf("Evictions: %d", lruStats.EvictionCount)
fmt.Printf("Oldest session age: %d seconds", lruStats.OldestSessionAge)

// Get memory monitoring stats
memStats := monitor.GetStats()
fmt.Printf("GC count: %d", memStats.NumGC)
fmt.Printf("Heap usage: %.1f MB", memStats.HeapAllocMB)
```

## Configuration Examples

### Development Configuration
```yaml
cleanup_interval: "1m"
session_timeout: "1h"
max_sessions: 100
max_memory_mb: 10
enable_persistence: false
```

### Production Configuration
```yaml
cleanup_interval: "5m"
session_timeout: "24h"
max_sessions: 10000
max_memory_mb: 500
enable_persistence: true
persistence_path: "/data/sessions"
persistence_format: "json"
persistence_interval: "30s"
enable_async_persistence: true
```

## Testing Commands

### Run All Tests
```bash
# Test all context packages
go test ./internal/context/... -v

# Test with coverage
go test ./internal/context/... -v -cover

# Test with race detection
go test ./internal/context/... -v -race
```

### Component-Specific Tests
```bash
# Test core context manager
go test ./internal/context -v

# Test persistence layer
go test ./internal/context/persistence -v

# Test memory management
go test ./internal/context/memory -v

# Test configuration
go test ./internal/context -v -run TestConfig
```

### Performance Testing
```bash
# Run benchmarks
go test ./internal/context/... -bench=. -benchmem

# Test memory management with high load
go test ./internal/context/memory -v -run TestLRUManager_MemoryEstimation

# Test concurrent access
go test ./internal/context -v -run TestContextManager_ThreadSafety
```

### Integration Testing
```bash
# Test persistence recovery
go test ./internal/context -v -run TestContextManager_PersistenceRecovery

# Test async persistence
go test ./internal/context -v -run TestContextManager_AsyncPersistence

# Test memory pressure handling
go test ./internal/context -v -run TestContextManager_MemoryPressure
```

## Performance Characteristics

### Benchmarks (approximate)
- **Session Update**: ~1ms per operation
- **Pronoun Resolution**: ~0.5ms per query
- **File Persistence**: ~5ms per session (async)
- **Memory Usage**: ~1KB per session context
- **LRU Cache Hit Rate**: >90% in typical usage

### Scaling
- **Maximum Sessions**: 10,000+ concurrent sessions
- **Memory Efficiency**: LRU eviction maintains constant memory usage
- **Persistence**: Handles 1000+ sessions/second with async writes
- **Recovery Time**: <2 seconds startup for 1000 persisted sessions

## Error Handling

The package provides comprehensive error handling with detailed error messages:

```go
// Persistence errors
type PersistenceError struct {
    Operation string
    SessionID string
    Cause     error
}

// Memory pressure handling
func handleMemoryWarning(stats MemoryStats) {
    // Trigger cleanup, reduce cache size, etc.
}

func handleMemoryCritical(stats MemoryStats) {
    // Force GC, emergency session cleanup
}
```

## Best Practices

### Configuration
1. **Development**: Disable persistence for faster testing
2. **Staging**: Enable persistence with shorter timeouts
3. **Production**: Use async persistence with appropriate memory limits

### Memory Management
1. Set reasonable session limits based on available memory
2. Configure memory monitoring thresholds (80% warning, 95% critical)
3. Implement memory pressure callbacks for graceful degradation

### Persistence
1. Use atomic writes to prevent corruption
2. Implement proper cleanup of expired sessions
3. Monitor disk usage and implement rotation if needed

### Monitoring
1. Track session creation/expiration rates
2. Monitor cache hit rates and memory usage
3. Set up alerts for persistence failures

## Troubleshooting

### Common Issues

**High Memory Usage**
```bash
# Check LRU cache statistics
go test ./internal/context/memory -v -run TestLRUManager_StatsAccuracy

# Monitor memory pressure
go test ./internal/context/memory -v -run TestMemoryMonitor_GetMemoryPressure
```

**Persistence Failures**
```bash
# Test file storage operations
go test ./internal/context/persistence -v -run TestFileStorage_SaveAndLoadSession

# Check atomic write behavior
go test ./internal/context/persistence -v -run TestFileStorage_AtomicWrites
```

**Session Recovery Issues**
```bash
# Test session recovery
go test ./internal/context -v -run TestContextManager_PersistenceRecovery
```

### Debug Logging
Enable debug logging to troubleshoot issues:

```go
// In production, implement proper logging
log.Printf("Session %s updated: %d history entries", sessionID, len(context.ConversationHistory))
log.Printf("Memory usage: %.1f MB (%.1f%%)", stats.MemoryUsageMB, stats.MemoryUsagePercent)
log.Printf("Cache hit rate: %.2f%% (%d/%d)", stats.HitRate*100, stats.CacheHits, stats.TotalAccesses)
```

## Integration with GenAI System

The context manager integrates with the broader GenAI system:

1. **Query Processing**: Resolves pronouns before LLM processing
2. **Response Handling**: Extracts references from structured responses
3. **Session Management**: Maintains conversation state across interactions
4. **Memory Efficiency**: Prevents memory leaks in long-running services

## Future Enhancements

Potential improvements for future versions:
1. **Redis Backend**: For distributed session storage
2. **Semantic Understanding**: NLP-based reference resolution
3. **Context Compression**: Summarization for long conversations
4. **Multi-tenant Support**: Isolated contexts per organization
5. **Advanced Analytics**: Usage patterns and optimization insights