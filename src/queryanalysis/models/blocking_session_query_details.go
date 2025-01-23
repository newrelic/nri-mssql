package models

type BlockingSessionQueryDetails struct {
	BlockingSPID          *int64   `db:"blocking_spid" metric_name:"blocking_spid" source_type:"gauge"`
	BlockingStatus        *string  `db:"blocking_status" metric_name:"blocking_status" source_type:"attribute"`
	BlockedSPID           *int64   `db:"blocked_spid" metric_name:"blocked_spid" source_type:"gauge"`
	BlockedStatus         *string  `db:"blocked_status" metric_name:"blocked_status" source_type:"attribute"`
	WaitType              *string  `db:"wait_type" metric_name:"wait_type" source_type:"attribute"`
	WaitTimeInSeconds     *float64 `db:"wait_time_in_seconds" metric_name:"wait_time_in_seconds" source_type:"gauge"`
	CommandType           *string  `db:"command_type" metric_name:"command_type" source_type:"attribute"`
	DatabaseName          *string  `db:"database_name" metric_name:"database_name" source_type:"attribute"`
	BlockingQueryText     *string  `db:"blocking_query_text" metric_name:"blocking_query_text" source_type:"attribute"`
	BlockedQueryText      *string  `db:"blocked_query_text" metric_name:"blocked_query_text" source_type:"attribute"`
	BlockedQueryStartTime *string  `db:"blocked_query_start_time" metric_name:"blocked_query_start_time" source_type:"attribute"`
}
