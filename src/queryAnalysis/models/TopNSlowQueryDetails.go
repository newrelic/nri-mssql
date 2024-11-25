package models

type TopNSlowQueryDetails struct {
	QueryID                *string `db:"query_id"`
	QueryText              *string `db:"query_text"`
	DatabaseName           *string `db:"database_name"`
	SchemaName             *string `db:"schema_name"`
	LastExecutionTimestamp *string `db:"last_execution_timestamp"`
	ExecutionCount         *string `db:"execution_count"`
	AvgCPUTimeMS           *string `db:"avg_cpu_time_ms"`
	AvgElapsedTimeMS       *string `db:"avg_elapsed_time_ms"`
	AvgDiskReads           *string `db:"avg_disk_reads"`
	AvgDiskWrites          *string `db:"avg_disk_writes"`
	StatementType          *string `db:"statement_type"`
	CollectionTimestamp    *string `db:"collection_timestamp"`
}
