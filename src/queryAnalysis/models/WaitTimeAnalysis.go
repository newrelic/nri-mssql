package models

import (
	"github.com/jmoiron/sqlx"
)

type WaitTimeAnalysis struct {
	Connection          *sqlx.DB
	QueryID             string `db:"query_id" json:"query_id"`
	DatabaseName        string `db:"database_name" json:"database_name"`
	QueryText           string `db:"query_text" json:"query_text"`
	CustomQueryType     string `db:"custom_query_type" json:"custom_query_type"`
	WaitCategory        string `db:"wait_category" json:"wait_category"`
	TotalWaitTimeMs     string `db:"total_wait_time_ms" json:"total_wait_time_ms"`
	AvgWaitTimeMs       string `db:"avg_wait_time_ms" json:"avg_wait_time_ms"`
	WaitEventCount      string `db:"wait_event_count" json:"wait_event_count"`
	CollectionTimestamp string `db:"collection_timestamp"`
}
