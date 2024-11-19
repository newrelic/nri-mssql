package models

type BlockingSessionQueryDetails struct {
	BlockingSPID      *int     `db:"blocking_spid"`
	BlockingStatus    *string  `db:"blocking_status"`
	BlockedSPID       *int     `db:"blocked_spid"`
	BlockedStatus     *string  `db:"blocked_status"`
	WaitType          *string  `db:"wait_type"`
	WaitTimeInSeconds *float64 `db:"wait_time_in_seconds"`
	CommandType       *string  `db:"command_type"`
	DatabaseName      *string  `db:"database_name"`
	BlockingQueryText *string  `db:"blocking_query_text"`
	BlockedQueryText  *string  `db:"blocked_query_text"`
}
