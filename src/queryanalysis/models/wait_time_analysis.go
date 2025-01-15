package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type WaitTimeAnalysis struct {
	Connection          *sqlx.DB
	QueryID             *HexString `db:"query_id" json:"query_id" metric_name:"query_id" sourceType:"attribute"`
	DatabaseName        *string    `db:"database_name" json:"database_name" metric_name:"database_name" sourceType:"attribute"`
	QueryText           *string    `db:"query_text" json:"query_text" metric_name:"query_text" sourceType:"attribute"`
	WaitCategory        *string    `db:"wait_category" json:"wait_category" metric_name:"wait_category" sourceType:"attribute"`
	TotalWaitTimeMs     *float64   `db:"total_wait_time_ms" json:"total_wait_time_ms" metric_name:"total_wait_time_ms" sourceType:"gauge"`
	AvgWaitTimeMs       *float64   `db:"avg_wait_time_ms" json:"avg_wait_time_ms" metric_name:"avg_wait_time_ms" sourceType:"gauge"`
	WaitEventCount      *int64     `db:"wait_event_count" json:"wait_event_count" metric_name:"wait_event_count" sourceType:"gauge"`
	LastExecutionTime   *time.Time `db:"last_execution_time" json:"last_execution_time" metric_name:"last_execution_time" sourceType:"attribute"`
	CollectionTimestamp time.Time  `db:"collection_timestamp" metric_name:"collection_timestamp" sourceType:"attribute"`
}
