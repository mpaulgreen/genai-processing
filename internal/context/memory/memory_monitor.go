package memory

import (
	"runtime"
	"sync"
	"time"
)

// MemoryMonitor tracks system memory usage and provides alerts when limits are approached.
// It helps prevent OOM conditions and provides visibility into memory consumption patterns.
type MemoryMonitor struct {
	maxMemoryMB     int
	warningThreshold float64 // Percentage (0.0-1.0) at which to warn
	criticalThreshold float64 // Percentage (0.0-1.0) at which to take action
	
	// Monitoring state
	isMonitoring    bool
	monitorInterval time.Duration
	stopChan        chan bool
	mu              sync.RWMutex
	
	// Statistics
	stats MemoryStats
	
	// Callbacks
	warningCallback  func(stats MemoryStats)
	criticalCallback func(stats MemoryStats)
}

// MemoryStats contains detailed memory usage statistics
type MemoryStats struct {
	// Current memory usage
	UsedMemoryMB     float64 `json:"used_memory_mb"`
	TotalMemoryMB    float64 `json:"total_memory_mb"`
	AvailableMemoryMB float64 `json:"available_memory_mb"`
	
	// Usage percentages
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	
	// Go runtime memory stats
	HeapAllocMB      float64 `json:"heap_alloc_mb"`
	HeapSysMB        float64 `json:"heap_sys_mb"`
	HeapIdleMB       float64 `json:"heap_idle_mb"`
	HeapInuseMB      float64 `json:"heap_inuse_mb"`
	
	// GC statistics
	NumGC           uint32  `json:"num_gc"`
	LastGCTime      int64   `json:"last_gc_time"`
	GCPausePercent  float64 `json:"gc_pause_percent"`
	
	// Monitoring metadata
	Timestamp       int64   `json:"timestamp"`
	SampleCount     int64   `json:"sample_count"`
	
	// Alert levels
	IsWarningLevel  bool    `json:"is_warning_level"`
	IsCriticalLevel bool    `json:"is_critical_level"`
}

// NewMemoryMonitor creates a new memory monitor with specified limits and thresholds.
func NewMemoryMonitor(maxMemoryMB int, warningThreshold, criticalThreshold float64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemoryMB:       maxMemoryMB,
		warningThreshold:  warningThreshold,
		criticalThreshold: criticalThreshold,
		monitorInterval:   30 * time.Second,
		stopChan:          make(chan bool),
		stats:             MemoryStats{},
	}
}

// SetCallbacks sets callback functions for warning and critical memory conditions.
func (mm *MemoryMonitor) SetCallbacks(warningCallback, criticalCallback func(MemoryStats)) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.warningCallback = warningCallback
	mm.criticalCallback = criticalCallback
}

// SetMonitorInterval sets how often memory is checked.
func (mm *MemoryMonitor) SetMonitorInterval(interval time.Duration) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.monitorInterval = interval
}

// Start begins memory monitoring in a background goroutine.
func (mm *MemoryMonitor) Start() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if mm.isMonitoring {
		return
	}
	
	// Create a new channel if it was closed
	select {
	case <-mm.stopChan:
		// Channel was closed, create a new one
		mm.stopChan = make(chan bool)
	default:
		// Channel is still open, use existing one
	}
	
	mm.isMonitoring = true
	go mm.monitorLoop()
}

// Stop stops memory monitoring.
func (mm *MemoryMonitor) Stop() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if !mm.isMonitoring {
		return
	}
	
	mm.isMonitoring = false
	close(mm.stopChan)
}

// GetStats returns current memory statistics.
func (mm *MemoryMonitor) GetStats() MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	mm.updateStats()
	return mm.stats
}

// CheckMemory performs an immediate memory check and returns current stats.
func (mm *MemoryMonitor) CheckMemory() MemoryStats {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.updateStats()
	mm.checkThresholds()
	
	return mm.stats
}

// ForceGC triggers garbage collection and returns memory stats before and after.
func (mm *MemoryMonitor) ForceGC() (before, after MemoryStats) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Get stats before GC
	mm.updateStats()
	before = mm.stats
	
	// Force garbage collection
	runtime.GC()
	runtime.GC() // Call twice to ensure full collection
	
	// Get stats after GC
	mm.updateStats()
	after = mm.stats
	
	return before, after
}

// monitorLoop runs the continuous memory monitoring.
func (mm *MemoryMonitor) monitorLoop() {
	// Get the monitoring interval and channel under lock
	mm.mu.RLock()
	interval := mm.monitorInterval
	stopChan := mm.stopChan
	mm.mu.RUnlock()
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mm.mu.Lock()
			// Check if we should still be monitoring
			if !mm.isMonitoring {
				mm.mu.Unlock()
				return
			}
			mm.updateStats()
			mm.checkThresholds()
			mm.mu.Unlock()
			
		case <-stopChan:
			return
		}
	}
}

// updateStats updates the internal memory statistics.
func (mm *MemoryMonitor) updateStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	now := time.Now()
	
	// Convert bytes to MB
	bytesToMB := func(b uint64) float64 {
		return float64(b) / 1024 / 1024
	}
	
	// Update basic memory stats
	mm.stats.HeapAllocMB = bytesToMB(m.HeapAlloc)
	mm.stats.HeapSysMB = bytesToMB(m.HeapSys)
	mm.stats.HeapIdleMB = bytesToMB(m.HeapIdle)
	mm.stats.HeapInuseMB = bytesToMB(m.HeapInuse)
	
	// Calculate usage against configured limit
	mm.stats.UsedMemoryMB = mm.stats.HeapAllocMB
	mm.stats.TotalMemoryMB = float64(mm.maxMemoryMB)
	mm.stats.AvailableMemoryMB = mm.stats.TotalMemoryMB - mm.stats.UsedMemoryMB
	mm.stats.MemoryUsagePercent = (mm.stats.UsedMemoryMB / mm.stats.TotalMemoryMB) * 100
	
	// GC statistics
	mm.stats.NumGC = m.NumGC
	if m.NumGC > 0 {
		mm.stats.LastGCTime = now.Unix()
		
		// Calculate GC pause percentage (last 256 pauses)
		var totalPause uint64
		numSamples := int(m.NumGC)
		if numSamples > 256 {
			numSamples = 256
		}
		
		for i := 0; i < numSamples; i++ {
			totalPause += m.PauseNs[(m.NumGC+uint32(255-i))%256]
		}
		
		if numSamples > 0 {
			avgPauseNs := totalPause / uint64(numSamples)
			// Calculate pause percentage over monitoring interval
			intervalNs := uint64(mm.monitorInterval.Nanoseconds())
			mm.stats.GCPausePercent = (float64(avgPauseNs) / float64(intervalNs)) * 100
		}
	}
	
	// Monitoring metadata
	mm.stats.Timestamp = now.Unix()
	mm.stats.SampleCount++
	
	// Check alert levels
	mm.stats.IsWarningLevel = mm.stats.MemoryUsagePercent >= (mm.warningThreshold * 100)
	mm.stats.IsCriticalLevel = mm.stats.MemoryUsagePercent >= (mm.criticalThreshold * 100)
}

// checkThresholds checks if memory usage has crossed warning or critical thresholds.
func (mm *MemoryMonitor) checkThresholds() {
	usagePercent := mm.stats.MemoryUsagePercent / 100
	
	if usagePercent >= mm.criticalThreshold && mm.criticalCallback != nil {
		mm.criticalCallback(mm.stats)
	} else if usagePercent >= mm.warningThreshold && mm.warningCallback != nil {
		mm.warningCallback(mm.stats)
	}
}

// GetMemoryPressure returns a simplified memory pressure indicator.
func (mm *MemoryMonitor) GetMemoryPressure() MemoryPressure {
	stats := mm.GetStats()
	
	if stats.IsCriticalLevel {
		return CriticalPressure
	} else if stats.IsWarningLevel {
		return HighPressure
	} else if stats.MemoryUsagePercent > 50 {
		return ModeratePressure
	} else {
		return LowPressure
	}
}

// MemoryPressure represents different levels of memory pressure.
type MemoryPressure int

const (
	LowPressure MemoryPressure = iota
	ModeratePressure
	HighPressure
	CriticalPressure
)

// String returns a string representation of memory pressure.
func (mp MemoryPressure) String() string {
	switch mp {
	case LowPressure:
		return "low"
	case ModeratePressure:
		return "moderate"
	case HighPressure:
		return "high"
	case CriticalPressure:
		return "critical"
	default:
		return "unknown"
	}
}