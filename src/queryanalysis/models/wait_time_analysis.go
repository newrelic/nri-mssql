package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type WaitTimeAnalysis struct {
	Connection          *sqlx.DB
	QueryID             *HexString `db:"query_id" json:"query_id"`
	DatabaseName        *string    `db:"database_name" json:"database_name"`
	QueryText           *string    `db:"query_text" json:"query_text"`
	WaitCategory        *string    `db:"wait_category" json:"wait_category"`
	TotalWaitTimeMs     *float64   `db:"total_wait_time_ms" json:"total_wait_time_ms"`
	AvgWaitTimeMs       *float64   `db:"avg_wait_time_ms" json:"avg_wait_time_ms"`
	WaitEventCount      *int64     `db:"wait_event_count" json:"wait_event_count"`
	LastExecutionTime   *time.Time `db:"last_execution_time" json:"last_execution_time"`
	CollectionTimestamp time.Time  `db:"collection_timestamp"`
}
