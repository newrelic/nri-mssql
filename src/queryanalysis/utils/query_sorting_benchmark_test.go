package utils

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

// generateTestQueries creates a slice of test queries with random AvgElapsedTimeMS values
func generateTestQueries(count int) []EnrichedSlowQueryDetails {
	rand.Seed(time.Now().UnixNano())
	queries := make([]EnrichedSlowQueryDetails, count)
	
	int64Ptr := func(v int64) *int64 { return &v }
	
	for i := 0; i < count; i++ {
		avgElapsed := rand.Float64() * 1000 // 0-1000ms
		queries[i] = EnrichedSlowQueryDetails{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(int64(rand.Intn(100) + 1)),
				TotalElapsedTime: int64Ptr(int64(avgElapsed * 1000)), // Convert to microseconds
			},
			AvgElapsedTimeMS: avgElapsed,
		}
	}
	return queries
}

// Benchmark the original sorting approach
func BenchmarkOriginalSort(b *testing.B) {
	queries := generateTestQueries(10000)
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 10,
		QueryMonitoringCountThreshold:        20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		testQueries := make([]EnrichedSlowQueryDetails, len(queries))
		copy(testQueries, queries)
		FilterSlowQueriesByThreshold(testQueries, args)
	}
}

// Benchmark the heap-based approach
func BenchmarkHeapSort(b *testing.B) {
	queries := generateTestQueries(10000)
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 10,
		QueryMonitoringCountThreshold:        20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		testQueries := make([]EnrichedSlowQueryDetails, len(queries))
		copy(testQueries, queries)
		FilterSlowQueriesByThresholdHeap(testQueries, args)
	}
}

// Benchmark the partial sort approach
func BenchmarkPartialSort(b *testing.B) {
	queries := generateTestQueries(10000)
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 10,
		QueryMonitoringCountThreshold:        20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		testQueries := make([]EnrichedSlowQueryDetails, len(queries))
		copy(testQueries, queries)
		FilterSlowQueriesByThresholdPartialSort(testQueries, args)
	}
}

// Benchmark the QuickSelect approach
func BenchmarkQuickSelect(b *testing.B) {
	queries := generateTestQueries(10000)
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 10,
		QueryMonitoringCountThreshold:        20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		testQueries := make([]EnrichedSlowQueryDetails, len(queries))
		copy(testQueries, queries)
		FilterSlowQueriesByThresholdQuickSelect(testQueries, args)
	}
}

// Test to verify all approaches produce the same results
func TestSortingApproachesEquivalence(t *testing.T) {
	queries := generateTestQueries(1000)
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 50,
		QueryMonitoringCountThreshold:        10,
	}

	// Test all approaches
	result1 := FilterSlowQueriesByThreshold(copyQueries(queries), args)
	result2 := FilterSlowQueriesByThresholdHeap(copyQueries(queries), args)
	result3 := FilterSlowQueriesByThresholdPartialSort(copyQueries(queries), args)
	result4 := FilterSlowQueriesByThresholdQuickSelect(copyQueries(queries), args)

	// All should return the same number of results
	if len(result1) != len(result2) || len(result1) != len(result3) || len(result1) != len(result4) {
		t.Errorf("Different result lengths: %d, %d, %d, %d", len(result1), len(result2), len(result3), len(result4))
	}

	// All should have the same top query (slowest)
	if len(result1) > 0 {
		if result1[0].AvgElapsedTimeMS != result2[0].AvgElapsedTimeMS ||
		   result1[0].AvgElapsedTimeMS != result3[0].AvgElapsedTimeMS ||
		   result1[0].AvgElapsedTimeMS != result4[0].AvgElapsedTimeMS {
			t.Errorf("Different slowest queries: %.2f, %.2f, %.2f, %.2f", 
				result1[0].AvgElapsedTimeMS, result2[0].AvgElapsedTimeMS, 
				result3[0].AvgElapsedTimeMS, result4[0].AvgElapsedTimeMS)
		}
	}

	fmt.Printf("✅ All approaches returned %d queries with slowest at %.2fms\n", 
		len(result1), result1[0].AvgElapsedTimeMS)
}

func copyQueries(queries []EnrichedSlowQueryDetails) []EnrichedSlowQueryDetails {
	result := make([]EnrichedSlowQueryDetails, len(queries))
	copy(result, queries)
	return result
}

// Example function to demonstrate performance characteristics
func TestPerformanceComparison(t *testing.T) {
	sizes := []int{100, 1000, 10000, 50000}
	topK := 20

	for _, size := range sizes {
		fmt.Printf("\n📊 Performance for %d queries (top %d):\n", size, topK)
		
		queries := generateTestQueries(size)
		args := args.ArgumentList{
			QueryMonitoringResponseTimeThreshold: 10,
			QueryMonitoringCountThreshold:        topK,
		}

		// Measure Original Sort
		start := time.Now()
		FilterSlowQueriesByThreshold(copyQueries(queries), args)
		originalTime := time.Since(start)

		// Measure Heap Sort
		start = time.Now()
		FilterSlowQueriesByThresholdHeap(copyQueries(queries), args)
		heapTime := time.Since(start)

		fmt.Printf("  Original Sort: %v\n", originalTime)
		fmt.Printf("  Heap Sort:     %v (%.1fx faster)\n", heapTime, float64(originalTime)/float64(heapTime))
	}
}