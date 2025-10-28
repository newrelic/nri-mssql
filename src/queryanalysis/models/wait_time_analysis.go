package models

import (
	"time"
)

// WaitTimeAnalysis represents the structure for analyzing current waiting sessions
// This has been updated to match the simplified query that shows currently waiting sessions
// instead of the previous complex query store-based analysis
type WaitTimeAnalysis struct {
	SessionID           *int64     `db:"session_id" json:"session_id" metric_name:"session_id" source_type:"attribute"`
	DatabaseName        *string    `db:"database_name" json:"database_name" metric_name:"database_name" source_type:"attribute"`
	QueryText           *string    `db:"query_text" json:"query_text" metric_name:"query_text" source_type:"attribute"`
	WaitCategory        *string    `db:"wait_category" json:"wait_category" metric_name:"wait_category" source_type:"attribute"`
	TotalWaitTimeMs     *float64   `db:"total_wait_time_ms" json:"total_wait_time_ms" metric_name:"total_wait_time_ms" source_type:"gauge"`
	RequestStartTime    *time.Time `db:"request_start_time" json:"request_start_time" metric_name:"request_start_time" source_type:"attribute"`
	CollectionTimestamp time.Time  `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}
