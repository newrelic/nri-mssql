package models

type BlockingSessionQueryDetails struct {
	BlockingSPID          *int64   `db:"blocking_spid" metric_name:"blocking_spid" sourceType:"gauge"`
	BlockingStatus        *string  `db:"blocking_status" metric_name:"blocking_status" sourceType:"attribute"`
	BlockedSPID           *int64   `db:"blocked_spid" metric_name:"blocked_spid" sourceType:"gauge"`
	BlockedStatus         *string  `db:"blocked_status" metric_name:"blocked_status" sourceType:"attribute"`
	WaitType              *string  `db:"wait_type" metric_name:"wait_type" sourceType:"attribute"`
	WaitTimeInSeconds     *float64 `db:"wait_time_in_seconds" metric_name:"wait_time_in_seconds" sourceType:"gauge"`
	CommandType           *string  `db:"command_type" metric_name:"command_type" sourceType:"attribute"`
	DatabaseName          *string  `db:"database_name" metric_name:"database_name" sourceType:"attribute"`
	BlockingQueryText     *string  `db:"blocking_query_text" metric_name:"blocking_query_text" sourceType:"attribute"`
	BlockedQueryText      *string  `db:"blocked_query_text" metric_name:"blocked_query_text" sourceType:"attribute"`
	BlockedQueryStartTime *string  `db:"blocked_query_start_time" metric_name:"blocked_query_start_time" sourceType:"attribute"`
}
