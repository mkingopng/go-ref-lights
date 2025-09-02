//go:build benchmark
// +build benchmark

package logger

import (
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkLoggingSystemComparison provides comprehensive performance comparison
// between old and new logging systems
func BenchmarkLoggingSystemComparison(b *testing.B) {
	b.Run("OldSystem_SimpleLogging", benchmarkOldSystemSimple)
	b.Run("NewSystem_StructuredLogging_Enabled", benchmarkNewSystemStructuredEnabled)
	b.Run("NewSystem_StructuredLogging_Disabled", benchmarkNewSystemStructuredDisabled)
	b.Run("NewSystem_LazyLogging_Enabled", benchmarkNewSystemLazyEnabled)
	b.Run("NewSystem_LazyLogging_Disabled", benchmarkNewSystemLazyDisabled)
	b.Run("NewSystem_ConditionalLogging", benchmarkNewSystemConditional)
	b.Run("ConcurrentLogging_OldSystem", benchmarkConcurrentOldSystem)
	b.Run("ConcurrentLogging_NewSystem", benchmarkConcurrentNewSystem)
}

// benchmarkOldSystemSimple benchmarks the old simple logging system
func benchmarkOldSystemSimple(b *testing.B) {
	logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Printf("Old system message %d with some data", i)
	}
}

// benchmarkNewSystemStructuredEnabled benchmarks new system with structured logging enabled
func benchmarkNewSystemStructuredEnabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(INFO)
	logger.infoLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "TestMeet",
		"refereeId": "left",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogWithContext(INFO, context, "New system message %d with some data", i)
	}
}

// benchmarkNewSystemStructuredDisabled benchmarks new system with structured logging disabled
func benchmarkNewSystemStructuredDisabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // INFO messages will be filtered out

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "TestMeet",
		"refereeId": "left",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogWithContext(INFO, context, "Disabled message %d with some data", i)
	}
}

// benchmarkNewSystemLazyEnabled benchmarks lazy logging when enabled
func benchmarkNewSystemLazyEnabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(INFO)
	logger.infoLogger = log.New(io.Discard, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogLazy(INFO, func() (string, map[string]interface{}) {
			return fmt.Sprintf("Lazy message %d", i), map[string]interface{}{
				"component": "benchmark",
				"iteration": i,
			}
		})
	}
}

// benchmarkNewSystemLazyDisabled benchmarks lazy logging when disabled
func benchmarkNewSystemLazyDisabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // INFO messages will be filtered out

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogLazy(INFO, func() (string, map[string]interface{}) {
			// This expensive operation should not be called
			time.Sleep(1 * time.Microsecond)
			return fmt.Sprintf("Lazy message %d", i), map[string]interface{}{
				"component": "benchmark",
				"iteration": i,
			}
		})
	}
}

// benchmarkNewSystemConditional benchmarks conditional logging optimization
func benchmarkNewSystemConditional(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // INFO messages will be filtered out

	expensiveOperation := func() string {
		// Simulate expensive string formatting
		var result strings.Builder
		for j := 0; j < 10; j++ {
			result.WriteString(fmt.Sprintf("data_%d_", j))
		}
		return result.String()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if logger.ShouldLog(INFO) {
			logger.LogWithContext(INFO, nil, "Expensive: %s", expensiveOperation())
		}
	}
}

// benchmarkConcurrentOldSystem benchmarks concurrent access to old logging system
func benchmarkConcurrentOldSystem(b *testing.B) {
	logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Printf("Concurrent old system message %d", i)
			i++
		}
	})
}

// benchmarkConcurrentNewSystem benchmarks concurrent access to new logging system
func benchmarkConcurrentNewSystem(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(INFO)
	logger.infoLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "benchmark",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.LogWithContext(INFO, context, "Concurrent new system message %d", i)
			i++
		}
	})
}

// TestPerformanceComparisonAnalysis provides detailed performance analysis
func TestPerformanceComparisonAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	// Test parameters
	iterations := 10000

	// Benchmark old system
	oldResults := benchmarkOldSystemPerformance(t, iterations)

	// Benchmark new system (enabled)
	newEnabledResults := benchmarkNewSystemPerformance(t, iterations, true)

	// Benchmark new system (disabled)
	newDisabledResults := benchmarkNewSystemPerformance(t, iterations, false)

	// Analyze and report results
	analyzePerformanceResults(t, oldResults, newEnabledResults, newDisabledResults)
}

// PerformanceResults holds performance test results
type PerformanceResults struct {
	Name            string
	Duration        time.Duration
	MemoryAllocated uint64
	MemoryAllocs    uint64
	MessagesPerSec  float64
}

// benchmarkOldSystemPerformance benchmarks the old logging system
func benchmarkOldSystemPerformance(t *testing.T, iterations int) PerformanceResults {
	logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Measure memory before
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < iterations; i++ {
		logger.Printf("Old system message %d with data", i)
	}

	duration := time.Since(start)

	// Measure memory after
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	return PerformanceResults{
		Name:            "OldSystem",
		Duration:        duration,
		MemoryAllocated: memAfter.TotalAlloc - memBefore.TotalAlloc,
		MemoryAllocs:    memAfter.Mallocs - memBefore.Mallocs,
		MessagesPerSec:  float64(iterations) / duration.Seconds(),
	}
}

// benchmarkNewSystemPerformance benchmarks the new logging system
func benchmarkNewSystemPerformance(t *testing.T, iterations int, enabled bool) PerformanceResults {
	logger := NewLogger()
	if enabled {
		logger.SetLevel(INFO)
		logger.infoLogger = log.New(io.Discard, "", 0)
	} else {
		logger.SetLevel(ERROR) // INFO messages will be filtered out
	}

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "TestMeet",
		"refereeId": "left",
	}

	// Measure memory before
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < iterations; i++ {
		logger.LogWithContext(INFO, context, "New system message %d with data", i)
	}

	duration := time.Since(start)

	// Measure memory after
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	name := "NewSystem_Enabled"
	if !enabled {
		name = "NewSystem_Disabled"
	}

	return PerformanceResults{
		Name:            name,
		Duration:        duration,
		MemoryAllocated: memAfter.TotalAlloc - memBefore.TotalAlloc,
		MemoryAllocs:    memAfter.Mallocs - memBefore.Mallocs,
		MessagesPerSec:  float64(iterations) / duration.Seconds(),
	}
}

// analyzePerformanceResults analyzes and reports performance comparison results
func analyzePerformanceResults(t *testing.T, old, newEnabled, newDisabled PerformanceResults) {
	t.Logf("Performance Comparison Results:")
	t.Logf("================================")

	// Report individual results
	results := []PerformanceResults{old, newEnabled, newDisabled}
	for _, result := range results {
		t.Logf("%s:", result.Name)
		t.Logf("  Duration: %v", result.Duration)
		t.Logf("  Messages/sec: %.2f", result.MessagesPerSec)
		t.Logf("  Memory allocated: %d bytes", result.MemoryAllocated)
		t.Logf("  Memory allocations: %d", result.MemoryAllocs)
		t.Logf("")
	}

	// Calculate overhead ratios
	enabledOverhead := float64(newEnabled.Duration) / float64(old.Duration)
	disabledSpeedup := float64(old.Duration) / float64(newDisabled.Duration)

	t.Logf("Performance Analysis:")
	t.Logf("====================")
	t.Logf("New system (enabled) overhead: %.2fx", enabledOverhead)
	t.Logf("New system (disabled) speedup: %.2fx", disabledSpeedup)

	// Memory comparison
	enabledMemoryRatio := float64(newEnabled.MemoryAllocated) / float64(old.MemoryAllocated)
	disabledMemoryRatio := float64(newDisabled.MemoryAllocated) / float64(old.MemoryAllocated)

	t.Logf("Memory usage (enabled): %.2fx", enabledMemoryRatio)
	t.Logf("Memory usage (disabled): %.2fx", disabledMemoryRatio)

	// Performance requirements validation
	t.Logf("Performance Requirements Validation:")
	t.Logf("===================================")

	// Requirement: New system should not be more than 10x slower when enabled
	if enabledOverhead > 10.0 {
		t.Errorf("FAIL: New system overhead too high: %.2fx (should be < 10x)", enabledOverhead)
	} else {
		t.Logf("PASS: New system overhead acceptable: %.2fx", enabledOverhead)
	}

	// Requirement: Disabled logging should be faster than old system
	if disabledSpeedup < 1.0 {
		t.Errorf("FAIL: Disabled logging should be faster than old system: %.2fx", disabledSpeedup)
	} else {
		t.Logf("PASS: Disabled logging is faster: %.2fx speedup", disabledSpeedup)
	}

	// Requirement: Memory usage should be reasonable
	if enabledMemoryRatio > 5.0 {
		t.Errorf("FAIL: Memory usage too high when enabled: %.2fx", enabledMemoryRatio)
	} else {
		t.Logf("PASS: Memory usage acceptable when enabled: %.2fx", enabledMemoryRatio)
	}

	if disabledMemoryRatio > 2.0 {
		t.Errorf("FAIL: Memory usage too high when disabled: %.2fx", disabledMemoryRatio)
	} else {
		t.Logf("PASS: Memory usage acceptable when disabled: %.2fx", disabledMemoryRatio)
	}
}

// TestConcurrentPerformanceComparison tests concurrent logging performance
func TestConcurrentPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance test in short mode")
	}

	numGoroutines := 10
	messagesPerGoroutine := 1000

	// Test old system concurrent performance
	oldDuration := testConcurrentOldSystem(t, numGoroutines, messagesPerGoroutine)

	// Test new system concurrent performance (enabled)
	newEnabledDuration := testConcurrentNewSystem(t, numGoroutines, messagesPerGoroutine, true)

	// Test new system concurrent performance (disabled)
	newDisabledDuration := testConcurrentNewSystem(t, numGoroutines, messagesPerGoroutine, false)

	// Analyze results
	t.Logf("Concurrent Performance Comparison:")
	t.Logf("=================================")
	t.Logf("Goroutines: %d", numGoroutines)
	t.Logf("Messages per goroutine: %d", messagesPerGoroutine)
	t.Logf("Total messages: %d", numGoroutines*messagesPerGoroutine)
	t.Logf("")
	t.Logf("Old system: %v", oldDuration)
	t.Logf("New system (enabled): %v", newEnabledDuration)
	t.Logf("New system (disabled): %v", newDisabledDuration)
	t.Logf("")

	enabledOverhead := float64(newEnabledDuration) / float64(oldDuration)
	disabledSpeedup := float64(oldDuration) / float64(newDisabledDuration)

	t.Logf("Concurrent overhead (enabled): %.2fx", enabledOverhead)
	t.Logf("Concurrent speedup (disabled): %.2fx", disabledSpeedup)

	// Validate concurrent performance requirements
	if enabledOverhead > 5.0 {
		t.Errorf("Concurrent logging overhead too high: %.2fx", enabledOverhead)
	}

	if disabledSpeedup < 0.5 {
		t.Errorf("Concurrent disabled logging should not be much slower: %.2fx", disabledSpeedup)
	}
}

// testConcurrentOldSystem tests concurrent performance of old logging system
func testConcurrentOldSystem(t *testing.T, numGoroutines, messagesPerGoroutine int) time.Duration {
	logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Printf("Concurrent old message %d from goroutine %d", j, goroutineID)
			}
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

// testConcurrentNewSystem tests concurrent performance of new logging system
func testConcurrentNewSystem(t *testing.T, numGoroutines, messagesPerGoroutine int, enabled bool) time.Duration {
	logger := NewLogger()
	if enabled {
		logger.SetLevel(INFO)
		logger.infoLogger = log.New(io.Discard, "", 0)
	} else {
		logger.SetLevel(ERROR) // INFO messages will be filtered out
	}

	context := map[string]interface{}{
		"component": "concurrent_test",
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.LogWithContext(INFO, context, "Concurrent new message %d from goroutine %d", j, goroutineID)
			}
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

// TestLazyLoggingPerformanceValidation validates lazy logging performance benefits
func TestLazyLoggingPerformanceValidation(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // INFO messages will be filtered out

	iterations := 1000

	// Test regular logging with expensive operations (should be slow if not optimized)
	expensiveOperationCount := 0
	expensiveOperation := func() string {
		expensiveOperationCount++
		time.Sleep(100 * time.Microsecond) // Simulate expensive operation
		return "expensive result"
	}

	// Test 1: Regular logging without conditional check (bad practice)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		logger.LogWithContext(INFO, nil, "Message: %s", expensiveOperation())
	}
	regularDuration := time.Since(start)

	// Reset counter
	expensiveOperationCount = 0

	// Test 2: Conditional logging (good practice)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		if logger.ShouldLog(INFO) {
			logger.LogWithContext(INFO, nil, "Message: %s", expensiveOperation())
		}
	}
	conditionalDuration := time.Since(start)

	// Reset counter
	expensiveOperationCount = 0

	// Test 3: Lazy logging (best practice)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		logger.LogLazy(INFO, func() (string, map[string]interface{}) {
			return fmt.Sprintf("Message: %s", expensiveOperation()), nil
		})
	}
	lazyDuration := time.Since(start)

	t.Logf("Lazy Logging Performance Validation:")
	t.Logf("===================================")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Regular logging: %v", regularDuration)
	t.Logf("Conditional logging: %v", conditionalDuration)
	t.Logf("Lazy logging: %v", lazyDuration)
	t.Logf("")

	conditionalSpeedup := float64(regularDuration) / float64(conditionalDuration)
	lazySpeedup := float64(regularDuration) / float64(lazyDuration)

	t.Logf("Conditional speedup: %.2fx", conditionalSpeedup)
	t.Logf("Lazy speedup: %.2fx", lazySpeedup)

	// Validate performance improvements
	if conditionalSpeedup < 10.0 {
		t.Errorf("Conditional logging should be much faster: %.2fx speedup", conditionalSpeedup)
	}

	if lazySpeedup < 10.0 {
		t.Errorf("Lazy logging should be much faster: %.2fx speedup", lazySpeedup)
	}

	// Validate that expensive operations were avoided
	if expensiveOperationCount > 0 {
		t.Errorf("Expensive operations should be avoided with lazy logging, called %d times", expensiveOperationCount)
	}
}

// TestMemoryUsageValidation validates memory usage patterns
func TestMemoryUsageValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	// Test memory usage with different logging patterns
	iterations := 10000

	// Test 1: Old system memory usage
	oldMemory := measureMemoryUsage(func() {
		logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		for i := 0; i < iterations; i++ {
			logger.Printf("Old system message %d", i)
		}
	})

	// Test 2: New system memory usage (enabled)
	newEnabledMemory := measureMemoryUsage(func() {
		logger := NewLogger()
		logger.SetLevel(INFO)
		logger.infoLogger = log.New(io.Discard, "", 0)

		context := map[string]interface{}{
			"component": "test",
			"meetName":  "TestMeet",
		}

		for i := 0; i < iterations; i++ {
			logger.LogWithContext(INFO, context, "New system message %d", i)
		}
	})

	// Test 3: New system memory usage (disabled)
	newDisabledMemory := measureMemoryUsage(func() {
		logger := NewLogger()
		logger.SetLevel(ERROR) // INFO messages filtered out

		context := map[string]interface{}{
			"component": "test",
			"meetName":  "TestMeet",
		}

		for i := 0; i < iterations; i++ {
			logger.LogWithContext(INFO, context, "New system message %d", i)
		}
	})

	t.Logf("Memory Usage Validation:")
	t.Logf("=======================")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Old system: %d bytes", oldMemory)
	t.Logf("New system (enabled): %d bytes", newEnabledMemory)
	t.Logf("New system (disabled): %d bytes", newDisabledMemory)
	t.Logf("")

	enabledRatio := float64(newEnabledMemory) / float64(oldMemory)
	disabledRatio := float64(newDisabledMemory) / float64(oldMemory)

	t.Logf("Memory ratio (enabled): %.2fx", enabledRatio)
	t.Logf("Memory ratio (disabled): %.2fx", disabledRatio)

	// Validate memory usage requirements
	if enabledRatio > 3.0 {
		t.Errorf("Memory usage too high when enabled: %.2fx", enabledRatio)
	}

	if disabledRatio > 1.5 {
		t.Errorf("Memory usage too high when disabled: %.2fx", disabledRatio)
	}

	// Disabled logging should use less memory than enabled
	if newDisabledMemory > newEnabledMemory {
		t.Errorf("Disabled logging should use less memory: %d vs %d", newDisabledMemory, newEnabledMemory)
	}
}

// measureMemoryUsage measures memory allocated during function execution
func measureMemoryUsage(fn func()) uint64 {
	var memBefore, memAfter runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	fn()

	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	return memAfter.TotalAlloc - memBefore.TotalAlloc
}
