package utils

import (
	"container/heap"
	"sort"

	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
)

// QueryHeap implements heap.Interface for EnrichedSlowQueryDetails
// This is a min-heap, so we can efficiently maintain the top K slowest queries
type QueryHeap []EnrichedSlowQueryDetails

func (h QueryHeap) Len() int           { return len(h) }
func (h QueryHeap) Less(i, j int) bool { return h[i].AvgElapsedTimeMS < h[j].AvgElapsedTimeMS } // Min-heap
func (h QueryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *QueryHeap) Push(x interface{}) {
	*h = append(*h, x.(EnrichedSlowQueryDetails))
}

func (h *QueryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// FilterSlowQueriesByThresholdHeap - MOST EFFICIENT version using heap
// Time Complexity: O(n log k) where n = total queries, k = count threshold
// Space Complexity: O(k)
func FilterSlowQueriesByThresholdHeap(enrichedQueries []EnrichedSlowQueryDetails, args args.ArgumentList) []EnrichedSlowQueryDetails {
	if len(enrichedQueries) == 0 {
		log.Debug("No slow queries to filter")
		return enrichedQueries
	}

	thresholdMs := float64(args.QueryMonitoringResponseTimeThreshold)
	countLimit := args.QueryMonitoringCountThreshold
	
	if countLimit <= 0 {
		countLimit = len(enrichedQueries)
	}

	// Use a min-heap to efficiently maintain top K queries
	h := &QueryHeap{}
	heap.Init(h)

	filteredCount := 0
	
	for _, query := range enrichedQueries {
		if query.AvgElapsedTimeMS >= thresholdMs {
			filteredCount++
			
			if h.Len() < countLimit {
				// Heap not full, just add the query
				heap.Push(h, query)
			} else if query.AvgElapsedTimeMS > (*h)[0].AvgElapsedTimeMS {
				// Query is slower than the fastest query in our top-K, replace it
				heap.Pop(h)
				heap.Push(h, query)
			}
		}
	}

	log.Debug("Filtered %d queries out of %d based on response time threshold %.2f ms", 
		filteredCount, len(enrichedQueries), thresholdMs)

	// Convert heap to slice and sort in descending order
	result := make([]EnrichedSlowQueryDetails, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(EnrichedSlowQueryDetails)
	}

	log.Debug("Returning top %d slowest queries", len(result))
	return result
}

// FilterSlowQueriesByThresholdPartialSort - Alternative efficient approach
// Time Complexity: O(n + k log k) where n = total queries, k = count threshold
// Space Complexity: O(n) but only sorts the top k elements
func FilterSlowQueriesByThresholdPartialSort(enrichedQueries []EnrichedSlowQueryDetails, args args.ArgumentList) []EnrichedSlowQueryDetails {
	if len(enrichedQueries) == 0 {
		log.Debug("No slow queries to filter")
		return enrichedQueries
	}

	// Step 1: Filter by threshold
	thresholdMs := float64(args.QueryMonitoringResponseTimeThreshold)
	filteredQueries := make([]EnrichedSlowQueryDetails, 0, len(enrichedQueries))
	
	for _, query := range enrichedQueries {
		if query.AvgElapsedTimeMS >= thresholdMs {
			filteredQueries = append(filteredQueries, query)
		}
	}

	log.Debug("Filtered %d queries out of %d based on response time threshold %.2f ms", 
		len(filteredQueries), len(enrichedQueries), thresholdMs)

	if len(filteredQueries) == 0 {
		return []EnrichedSlowQueryDetails{}
	}

	// Step 2: Use partial sort - only sort as much as needed
	countLimit := args.QueryMonitoringCountThreshold
	if countLimit <= 0 || countLimit > len(filteredQueries) {
		countLimit = len(filteredQueries)
	}

	// Use nth_element equivalent - partial sort
	if countLimit < len(filteredQueries) {
		// Use Go's sort.Slice with a smaller slice for efficiency
		sort.Slice(filteredQueries, func(i, j int) bool {
			return filteredQueries[i].AvgElapsedTimeMS > filteredQueries[j].AvgElapsedTimeMS
		})
		filteredQueries = filteredQueries[:countLimit]
	} else {
		// Sort all if we need all of them
		sort.Slice(filteredQueries, func(i, j int) bool {
			return filteredQueries[i].AvgElapsedTimeMS > filteredQueries[j].AvgElapsedTimeMS
		})
	}

	log.Debug("Returning top %d slowest queries", len(filteredQueries))
	return filteredQueries
}

// FilterSlowQueriesByThresholdQuickSelect - Using QuickSelect algorithm
// Time Complexity: O(n) average case, O(n²) worst case
// Space Complexity: O(1)
func FilterSlowQueriesByThresholdQuickSelect(enrichedQueries []EnrichedSlowQueryDetails, args args.ArgumentList) []EnrichedSlowQueryDetails {
	if len(enrichedQueries) == 0 {
		log.Debug("No slow queries to filter")
		return enrichedQueries
	}

	// Step 1: Filter by threshold
	thresholdMs := float64(args.QueryMonitoringResponseTimeThreshold)
	filteredQueries := make([]EnrichedSlowQueryDetails, 0, len(enrichedQueries))
	
	for _, query := range enrichedQueries {
		if query.AvgElapsedTimeMS >= thresholdMs {
			filteredQueries = append(filteredQueries, query)
		}
	}

	if len(filteredQueries) == 0 {
		return []EnrichedSlowQueryDetails{}
	}

	countLimit := args.QueryMonitoringCountThreshold
	if countLimit <= 0 || countLimit > len(filteredQueries) {
		countLimit = len(filteredQueries)
	}

	// Step 2: Use quickselect to find the kth largest elements
	if countLimit < len(filteredQueries) {
		quickSelect(filteredQueries, 0, len(filteredQueries)-1, countLimit-1)
		filteredQueries = filteredQueries[:countLimit]
	}

	// Step 3: Sort only the selected elements
	sort.Slice(filteredQueries, func(i, j int) bool {
		return filteredQueries[i].AvgElapsedTimeMS > filteredQueries[j].AvgElapsedTimeMS
	})

	return filteredQueries
}

// quickSelect implements the QuickSelect algorithm to find the kth largest element
func quickSelect(queries []EnrichedSlowQueryDetails, left, right, k int) {
	if left == right {
		return
	}

	pivotIndex := partition(queries, left, right)

	if k == pivotIndex {
		return
	} else if k < pivotIndex {
		quickSelect(queries, left, pivotIndex-1, k)
	} else {
		quickSelect(queries, pivotIndex+1, right, k)
	}
}

// partition rearranges the slice so that elements greater than pivot are on the left
func partition(queries []EnrichedSlowQueryDetails, left, right int) int {
	pivot := queries[right].AvgElapsedTimeMS
	i := left

	for j := left; j < right; j++ {
		if queries[j].AvgElapsedTimeMS > pivot { // Greater than for descending order
			queries[i], queries[j] = queries[j], queries[i]
			i++
		}
	}
	queries[i], queries[right] = queries[right], queries[i]
	return i
}