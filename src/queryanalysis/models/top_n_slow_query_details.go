package models

type TopNSlowQueryDetails struct {
	QueryID                *HexString `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	QueryText              *string    `db:"query_text" metric_name:"query_text" source_type:"attribute"`
	DatabaseName           *string    `db:"database_name" metric_name:"database_name" source_type:"attribute"`
	SchemaName             *string    `db:"schema_name" metric_name:"schema_name" source_type:"attribute"`
	LastExecutionTimestamp *string    `db:"last_execution_timestamp" metric_name:"last_execution_timestamp" source_type:"attribute"`
	ExecutionCount         *int64     `db:"execution_count" metric_name:"execution_count" source_type:"gauge"`
	AvgCPUTimeMS           *float64   `db:"avg_cpu_time_ms" metric_name:"avg_cpu_time_ms" source_type:"gauge"`
	AvgElapsedTimeMS       *float64   `db:"avg_elapsed_time_ms" metric_name:"avg_elapsed_time_ms" source_type:"gauge"`
	AvgDiskReads           *float64   `db:"avg_disk_reads" metric_name:"avg_disk_reads" source_type:"gauge"`
	AvgDiskWrites          *float64   `db:"avg_disk_writes" metric_name:"avg_disk_writes" source_type:"gauge"`
	StatementType          *string    `db:"statement_type" metric_name:"statement_type" source_type:"attribute"`
	CollectionTimestamp    *string    `db:"collection_timestamp" metric_name:"collection_timestamp" source_type:"attribute"`
}
