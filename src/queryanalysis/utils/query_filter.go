package utils

import (
	"sort"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
)

// FilterSlowQueriesByThreshold filters and limits slow queries based on response time threshold and count limit
// This function:
// 1. Filters queries where AvgElapsedTimeMS >= QueryMonitoringResponseTimeThreshold
// 2. Sorts filtered queries by AvgElapsedTimeMS in descending order (slowest first)
// 3. Returns top QueryMonitoringCountThreshold queries
func FilterSlowQueriesByThreshold(enrichedQueries []EnrichedSlowQueryDetails, args args.ArgumentList) []EnrichedSlowQueryDetails {
	if len(enrichedQueries) == 0 {
		log.Debug("No slow queries to filter")
		return enrichedQueries
	}

	// Step 1: Filter queries based on AvgElapsedTimeMS threshold
	filteredQueries := make([]EnrichedSlowQueryDetails, 0)
	thresholdMs := float64(args.QueryMonitoringResponseTimeThreshold)
	
	for _, query := range enrichedQueries {
		if query.AvgElapsedTimeMS >= thresholdMs {
			filteredQueries = append(filteredQueries, query)
		}
	}

	log.Debug("Filtered %d queries out of %d based on response time threshold %.2f ms", 
		len(filteredQueries), len(enrichedQueries), thresholdMs)

	// If no queries meet the threshold, return empty slice
	if len(filteredQueries) == 0 {
		log.Debug("No queries meet the response time threshold of %.2f ms", thresholdMs)
		return []EnrichedSlowQueryDetails{}
	}

	// Step 2: Sort by AvgElapsedTimeMS in descending order (slowest first)
	sort.Slice(filteredQueries, func(i, j int) bool {
		return filteredQueries[i].AvgElapsedTimeMS > filteredQueries[j].AvgElapsedTimeMS
	})

	// Step 3: Limit to QueryMonitoringCountThreshold
	countLimit := args.QueryMonitoringCountThreshold
	if countLimit <= 0 || countLimit > len(filteredQueries) {
		countLimit = len(filteredQueries)
	}

	finalQueries := filteredQueries[:countLimit]
	
	log.Debug("Returning top %d slowest queries out of %d filtered queries", 
		len(finalQueries), len(filteredQueries))

	return finalQueries
}

// FilterSlowQueriesWithMetrics filters queries and provides detailed metrics about the filtering process
type FilterMetrics struct {
	TotalQueriesFromDB    int     `json:"total_queries_from_db"`
	QueriesAfterFilter    int     `json:"queries_after_filter"`
	QueriesAfterLimit     int     `json:"queries_after_limit"`
	ThresholdUsed         float64 `json:"threshold_used_ms"`
	CountLimitUsed        int     `json:"count_limit_used"`
	SlowestQueryTime      float64 `json:"slowest_query_time_ms"`
	FastestQueryTime      float64 `json:"fastest_query_time_ms"`
}

// FilterSlowQueriesWithMetrics does the same filtering but also returns detailed metrics
func FilterSlowQueriesWithMetrics(enrichedQueries []EnrichedSlowQueryDetails, args args.ArgumentList) ([]EnrichedSlowQueryDetails, FilterMetrics) {
	metrics := FilterMetrics{
		TotalQueriesFromDB: len(enrichedQueries),
		ThresholdUsed:      float64(args.QueryMonitoringResponseTimeThreshold),
		CountLimitUsed:     args.QueryMonitoringCountThreshold,
	}

	if len(enrichedQueries) == 0 {
		return enrichedQueries, metrics
	}

	// Filter and get results using the optimized heap approach for better performance
	filteredQueries := FilterSlowQueriesByThresholdHeap(enrichedQueries, args)
	
	metrics.QueriesAfterFilter = len(filteredQueries)
	metrics.QueriesAfterLimit = len(filteredQueries)

	// Calculate slowest and fastest query times from the final result set
	if len(filteredQueries) > 0 {
		metrics.SlowestQueryTime = filteredQueries[0].AvgElapsedTimeMS // Already sorted, first is slowest
		metrics.FastestQueryTime = filteredQueries[len(filteredQueries)-1].AvgElapsedTimeMS // Last is fastest
	}

	return filteredQueries, metrics
}

// LogFilterMetrics logs the filtering metrics for debugging
func LogFilterMetrics(metrics FilterMetrics) {
	log.Info("Query Filtering Metrics:")
	log.Info("  - Total queries from DB: %d", metrics.TotalQueriesFromDB)
	log.Info("  - Queries after threshold filter (>= %.2f ms): %d", metrics.ThresholdUsed, metrics.QueriesAfterFilter)
	log.Info("  - Final queries sent to New Relic: %d", metrics.QueriesAfterLimit)
	log.Info("  - Count limit used: %d", metrics.CountLimitUsed)
	if metrics.QueriesAfterLimit > 0 {
		log.Info("  - Slowest query time: %.2f ms", metrics.SlowestQueryTime)
		log.Info("  - Fastest query time: %.2f ms", metrics.FastestQueryTime)
	}
}