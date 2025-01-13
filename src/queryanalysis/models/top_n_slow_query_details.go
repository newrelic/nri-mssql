package models

type TopNSlowQueryDetails struct {
	QueryID                *HexString `db:"query_id" metric_name:"query_id" sourceType:"attribute"`
	QueryText              *string    `db:"query_text" metric_name:"query_text" sourceType:"attribute"`
	DatabaseName           *string    `db:"database_name" metric_name:"database_name" sourceType:"attribute"`
	SchemaName             *string    `db:"schema_name" metric_name:"schema_name" sourceType:"attribute"`
	LastExecutionTimestamp *string    `db:"last_execution_timestamp" metric_name:"last_execution_timestamp" sourceType:"attribute"`
	ExecutionCount         *int64     `db:"execution_count" metric_name:"execution_count" sourceType:"gauge"`
	AvgCPUTimeMS           *float64   `db:"avg_cpu_time_ms" metric_name:"avg_cpu_time_ms" sourceType:"gauge"`
	AvgElapsedTimeMS       *float64   `db:"avg_elapsed_time_ms" metric_name:"avg_elapsed_time_ms" sourceType:"gauge"`
	AvgDiskReads           *float64   `db:"avg_disk_reads" metric_name:"avg_disk_reads" sourceType:"gauge"`
	AvgDiskWrites          *float64   `db:"avg_disk_writes" metric_name:"avg_disk_writes" sourceType:"gauge"`
	StatementType          *string    `db:"statement_type" metric_name:"statement_type" sourceType:"attribute"`
	CollectionTimestamp    *string    `db:"collection_timestamp" metric_name:"collection_timestamp" sourceType:"attribute"`
}
