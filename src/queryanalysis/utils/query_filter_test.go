package utils

import (
	"testing"

	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"github.com/stretchr/testify/assert"
)

func TestFilterSlowQueriesByThreshold(t *testing.T) {
	// Create test data
	int64Ptr := func(v int64) *int64 { return &v }
	
	enrichedQueries := []EnrichedSlowQueryDetails{
		{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(10),
				TotalWorkerTime:  int64Ptr(50000), // 50ms total = 5ms avg
				TotalElapsedTime: int64Ptr(100000), // 100ms total = 10ms avg
			},
			AvgCPUTimeMS:     5.0,
			AvgElapsedTimeMS: 10.0,
		},
		{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(5),
				TotalWorkerTime:  int64Ptr(250000), // 250ms total = 50ms avg
				TotalElapsedTime: int64Ptr(500000), // 500ms total = 100ms avg
			},
			AvgCPUTimeMS:     50.0,
			AvgElapsedTimeMS: 100.0,
		},
		{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(2),
				TotalWorkerTime:  int64Ptr(10000), // 10ms total = 5ms avg
				TotalElapsedTime: int64Ptr(4000), // 4ms total = 2ms avg
			},
			AvgCPUTimeMS:     5.0,
			AvgElapsedTimeMS: 2.0,
		},
	}

	// Test case 1: Filter with threshold = 5ms, should return 2 queries (10ms and 100ms)
	args1 := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 5,
		QueryMonitoringCountThreshold:        10,
	}

	filtered1 := FilterSlowQueriesByThreshold(enrichedQueries, args1)
	assert.Equal(t, 2, len(filtered1), "Should return 2 queries with AvgElapsedTimeMS >= 5ms")
	
	// Results should be sorted by AvgElapsedTimeMS descending
	assert.Equal(t, 100.0, filtered1[0].AvgElapsedTimeMS, "First result should be slowest (100ms)")
	assert.Equal(t, 10.0, filtered1[1].AvgElapsedTimeMS, "Second result should be next slowest (10ms)")

	// Test case 2: Filter with threshold = 50ms, should return 1 query (100ms)
	args2 := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 50,
		QueryMonitoringCountThreshold:        10,
	}

	filtered2 := FilterSlowQueriesByThreshold(enrichedQueries, args2)
	assert.Equal(t, 1, len(filtered2), "Should return 1 query with AvgElapsedTimeMS >= 50ms")
	assert.Equal(t, 100.0, filtered2[0].AvgElapsedTimeMS, "Result should be the slowest query (100ms)")

	// Test case 3: Filter with count limit = 1, should return only 1 query even if more meet threshold
	args3 := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 1,
		QueryMonitoringCountThreshold:        1,
	}

	filtered3 := FilterSlowQueriesByThreshold(enrichedQueries, args3)
	assert.Equal(t, 1, len(filtered3), "Should return only 1 query due to count limit")
	assert.Equal(t, 100.0, filtered3[0].AvgElapsedTimeMS, "Should return the slowest query")

	// Test case 4: High threshold that excludes all queries
	args4 := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 200,
		QueryMonitoringCountThreshold:        10,
	}

	filtered4 := FilterSlowQueriesByThreshold(enrichedQueries, args4)
	assert.Equal(t, 0, len(filtered4), "Should return 0 queries when threshold is too high")
}

func TestFilterSlowQueriesWithMetrics(t *testing.T) {
	// Create test data
	int64Ptr := func(v int64) *int64 { return &v }
	
	enrichedQueries := []EnrichedSlowQueryDetails{
		{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(10),
				TotalElapsedTime: int64Ptr(100000), // 100ms total = 10ms avg
			},
			AvgElapsedTimeMS: 10.0,
		},
		{
			TopNSlowQueryDetails: models.TopNSlowQueryDetails{
				ExecutionCount:   int64Ptr(5),
				TotalElapsedTime: int64Ptr(500000), // 500ms total = 100ms avg
			},
			AvgElapsedTimeMS: 100.0,
		},
	}

	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 5,
		QueryMonitoringCountThreshold:        10,
	}

	filtered, metrics := FilterSlowQueriesWithMetrics(enrichedQueries, args)

	// Verify filtered results
	assert.Equal(t, 2, len(filtered), "Should return 2 filtered queries")

	// Verify metrics
	assert.Equal(t, 2, metrics.TotalQueriesFromDB, "Total queries from DB should be 2")
	assert.Equal(t, 2, metrics.QueriesAfterFilter, "Queries after filter should be 2")
	assert.Equal(t, 2, metrics.QueriesAfterLimit, "Queries after limit should be 2")
	assert.Equal(t, 5.0, metrics.ThresholdUsed, "Threshold used should be 5ms")
	assert.Equal(t, 10, metrics.CountLimitUsed, "Count limit used should be 10")
	assert.Equal(t, 100.0, metrics.SlowestQueryTime, "Slowest query should be 100ms")
	assert.Equal(t, 10.0, metrics.FastestQueryTime, "Fastest query should be 10ms")
}

func TestFilterSlowQueriesEmptyInput(t *testing.T) {
	args := args.ArgumentList{
		QueryMonitoringResponseTimeThreshold: 5,
		QueryMonitoringCountThreshold:        10,
	}

	// Test with empty slice
	filtered := FilterSlowQueriesByThreshold([]EnrichedSlowQueryDetails{}, args)
	assert.Equal(t, 0, len(filtered), "Should return empty slice for empty input")

	// Test with nil slice
	filtered = FilterSlowQueriesByThreshold(nil, args)
	assert.Equal(t, 0, len(filtered), "Should return empty slice for nil input")
}