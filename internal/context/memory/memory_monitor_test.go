package memory

import (
	"testing"
	"time"
)

func TestNewMemoryMonitor(t *testing.T) {
	maxMemoryMB := 100
	warningThreshold := 0.8
	criticalThreshold := 0.95

	monitor := NewMemoryMonitor(maxMemoryMB, warningThreshold, criticalThreshold)

	if monitor == nil {
		t.Fatal("NewMemoryMonitor() returned nil")
	}

	if monitor.maxMemoryMB != maxMemoryMB {
		t.Errorf("Expected maxMemoryMB %d, got %d", maxMemoryMB, monitor.maxMemoryMB)
	}

	if monitor.warningThreshold != warningThreshold {
		t.Errorf("Expected warningThreshold %f, got %f", warningThreshold, monitor.warningThreshold)
	}

	if monitor.criticalThreshold != criticalThreshold {
		t.Errorf("Expected criticalThreshold %f, got %f", criticalThreshold, monitor.criticalThreshold)
	}

	if monitor.monitorInterval != 30*time.Second {
		t.Errorf("Expected default monitorInterval 30s, got %v", monitor.monitorInterval)
	}

	if monitor.isMonitoring {
		t.Error("Monitor should not be monitoring initially")
	}
}

func TestMemoryMonitor_SetMonitorInterval(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)

	newInterval := 10 * time.Second
	monitor.SetMonitorInterval(newInterval)

	if monitor.monitorInterval != newInterval {
		t.Errorf("Expected monitorInterval %v, got %v", newInterval, monitor.monitorInterval)
	}
}

func TestMemoryMonitor_SetCallbacks(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)

	warningCallback := func(stats MemoryStats) {
		// Callback implementation for testing
	}

	criticalCallback := func(stats MemoryStats) {
		// Callback implementation for testing
	}

	monitor.SetCallbacks(warningCallback, criticalCallback)

	// Verify callbacks are set (we can't directly test the private fields,
	// but we can test that they work by triggering them)
	monitor.mu.Lock()
	if monitor.warningCallback == nil {
		t.Error("Warning callback was not set")
	}
	if monitor.criticalCallback == nil {
		t.Error("Critical callback was not set")
	}
	monitor.mu.Unlock()
}

func TestMemoryMonitor_GetStats(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)

	stats := monitor.GetStats()

	// Basic stats validation
	if stats.TotalMemoryMB != 100 {
		t.Errorf("Expected TotalMemoryMB 100, got %f", stats.TotalMemoryMB)
	}

	if stats.UsedMemoryMB < 0 {
		t.Error("UsedMemoryMB should not be negative")
	}

	if stats.AvailableMemoryMB < 0 {
		t.Error("AvailableMemoryMB should not be negative")
	}

	if stats.MemoryUsagePercent < 0 || stats.MemoryUsagePercent > 100 {
		t.Errorf("MemoryUsagePercent should be between 0-100, got %f", stats.MemoryUsagePercent)
	}

	if stats.HeapAllocMB < 0 {
		t.Error("HeapAllocMB should not be negative")
	}

	if stats.HeapSysMB < 0 {
		t.Error("HeapSysMB should not be negative")
	}

	if stats.Timestamp <= 0 {
		t.Error("Timestamp should be set")
	}

	if stats.SampleCount <= 0 {
		t.Error("SampleCount should be positive")
	}
}

func TestMemoryMonitor_CheckMemory(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)

	// Get initial sample count
	initialStats := monitor.GetStats()
	initialSampleCount := initialStats.SampleCount

	// Wait a moment to ensure timestamp difference
	time.Sleep(1 * time.Millisecond)

	// Check memory explicitly
	stats := monitor.CheckMemory()

	// Sample count should have increased
	if stats.SampleCount <= initialSampleCount {
		t.Error("SampleCount should have increased after CheckMemory()")
	}

	// Stats should be current (use greater than or equal since timestamps might be very close)
	if stats.Timestamp < initialStats.Timestamp {
		t.Error("Timestamp should be updated after CheckMemory()")
	}
}

func TestMemoryMonitor_ForceGC(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)

	before, after := monitor.ForceGC()

	// Verify we got stats
	if before.HeapAllocMB < 0 {
		t.Error("Before stats should be valid")
	}

	if after.HeapAllocMB < 0 {
		t.Error("After stats should be valid")
	}

	// NumGC should have increased
	if after.NumGC <= before.NumGC {
		t.Error("NumGC should have increased after ForceGC()")
	}

	// Timestamps should be different (use greater than or equal since they might be very close)
	if after.Timestamp < before.Timestamp {
		t.Error("After timestamp should be later than or equal to before timestamp")
	}
}

func TestMemoryMonitor_StartStop(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)
	monitor.SetMonitorInterval(50 * time.Millisecond) // Short interval for testing

	// Initially not monitoring
	if monitor.isMonitoring {
		t.Error("Monitor should not be monitoring initially")
	}

	// Start monitoring
	monitor.Start()

	if !monitor.isMonitoring {
		t.Error("Monitor should be monitoring after Start()")
	}

	// Starting again should not cause issues
	monitor.Start()

	if !monitor.isMonitoring {
		t.Error("Monitor should still be monitoring after second Start()")
	}

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop monitoring
	monitor.Stop()

	if monitor.isMonitoring {
		t.Error("Monitor should not be monitoring after Stop()")
	}

	// Stopping again should not cause issues
	monitor.Stop()

	if monitor.isMonitoring {
		t.Error("Monitor should still not be monitoring after second Stop()")
	}
}

func TestMemoryMonitor_GetMemoryPressure(t *testing.T) {
	// Test with different memory configurations to trigger different pressure levels
	testCases := []struct {
		name              string
		maxMemoryMB       int
		warningThreshold  float64
		criticalThreshold float64
		expectedMinPressure MemoryPressure
	}{
		{
			name:              "High memory limit",
			maxMemoryMB:       10000, // Very high limit
			warningThreshold:  0.8,
			criticalThreshold: 0.95,
			expectedMinPressure: LowPressure,
		},
		{
			name:              "Low memory limit",
			maxMemoryMB:       1, // Very low limit
			warningThreshold:  0.1,
			criticalThreshold: 0.2,
			expectedMinPressure: CriticalPressure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			monitor := NewMemoryMonitor(tc.maxMemoryMB, tc.warningThreshold, tc.criticalThreshold)
			pressure := monitor.GetMemoryPressure()

			// For high memory limit, pressure should be low
			// For very low memory limit, pressure should be high
			if tc.name == "High memory limit" && pressure != LowPressure {
				if pressure == CriticalPressure {
					t.Errorf("Expected low pressure with high memory limit, got %s", pressure.String())
				}
				// Moderate or high pressure might be acceptable depending on system state
			}

			if tc.name == "Low memory limit" && pressure == LowPressure {
				t.Errorf("Expected high pressure with very low memory limit, got %s", pressure.String())
			}
		})
	}
}

func TestMemoryPressure_String(t *testing.T) {
	testCases := []struct {
		pressure MemoryPressure
		expected string
	}{
		{LowPressure, "low"},
		{ModeratePressure, "moderate"},
		{HighPressure, "high"},
		{CriticalPressure, "critical"},
		{MemoryPressure(999), "unknown"}, // Invalid value
	}

	for _, tc := range testCases {
		result := tc.pressure.String()
		if result != tc.expected {
			t.Errorf("Expected %s for pressure %d, got %s", tc.expected, tc.pressure, result)
		}
	}
}

func TestMemoryMonitor_ThresholdCallbacks(t *testing.T) {
	// Use very low thresholds to ensure callbacks are triggered
	monitor := NewMemoryMonitor(1, 0.01, 0.02) // 1MB limit, very low thresholds

	warningCalled := false
	criticalCalled := false
	var warningStats, criticalStats MemoryStats

	monitor.SetCallbacks(
		func(stats MemoryStats) {
			warningCalled = true
			warningStats = stats
		},
		func(stats MemoryStats) {
			criticalCalled = true
			criticalStats = stats
		},
	)

	// Check memory to potentially trigger callbacks
	stats := monitor.CheckMemory()

	// With such low limits, callbacks should likely be triggered
	// Note: This test might be flaky depending on actual memory usage
	if stats.IsCriticalLevel && !criticalCalled {
		t.Error("Critical callback should have been called when IsCriticalLevel is true")
	}

	if stats.IsWarningLevel && !warningCalled && !criticalCalled {
		t.Error("Warning callback should have been called when IsWarningLevel is true")
	}

	// If callbacks were called, verify they received valid stats
	if warningCalled {
		if warningStats.Timestamp <= 0 {
			t.Error("Warning callback received invalid stats")
		}
	}

	if criticalCalled {
		if criticalStats.Timestamp <= 0 {
			t.Error("Critical callback received invalid stats")
		}
	}
}

func TestMemoryMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)
	monitor.SetMonitorInterval(10 * time.Millisecond)

	done := make(chan bool, 3)

	// Start monitoring in background
	monitor.Start()
	defer monitor.Stop()

	// Concurrent operations
	go func() {
		for i := 0; i < 10; i++ {
			monitor.GetStats()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			monitor.CheckMemory()
			time.Sleep(7 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			monitor.ForceGC()
			time.Sleep(15 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}

	// Verify monitor is still functional
	stats := monitor.GetStats()
	if stats.SampleCount <= 0 {
		t.Error("Monitor should still be functional after concurrent access")
	}
}

func TestMemoryMonitor_LongRunningMonitoring(t *testing.T) {
	monitor := NewMemoryMonitor(100, 0.8, 0.95)
	monitor.SetMonitorInterval(20 * time.Millisecond)

	initialStats := monitor.GetStats()
	initialSampleCount := initialStats.SampleCount

	// Start monitoring
	monitor.Start()

	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)

	// Stop monitoring
	monitor.Stop()

	finalStats := monitor.GetStats()

	// Sample count should have increased due to background monitoring
	if finalStats.SampleCount <= initialSampleCount {
		t.Error("SampleCount should have increased during monitoring period")
	}

	// Should not be monitoring after stop
	if monitor.isMonitoring {
		t.Error("Monitor should not be monitoring after Stop()")
	}
}