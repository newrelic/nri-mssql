package main

import (
	"fmt"
	"strings"
)

// QueryDefinition defines a single query with it's associated
// data model which has struct tags for metric.Set
type QueryDefinition struct {
	query      string
	dataModels interface{}
}

// QueryModifier is a function that takes in a query, does any modification
// and returns the query
type QueryModifier func(string) string

// GetQuery retrieves the query for a QueryDefinition
func (qd QueryDefinition) GetQuery(modifiers ...QueryModifier) string {
	modifiedQuery := qd.query

	for _, modifier := range modifiers {
		modifiedQuery = modifier(modifiedQuery)
	}

	return modifiedQuery
}

// GetDataModels retrieves the DataModels to load into this query
func (qd QueryDefinition) GetDataModels() interface{} {
	return qd.dataModels
}

// databasePlaceHolder placeholder for Database name in a query
const databasePlaceHolder = "%DATABASE%"

// dbNameReplace inserts the dbName into a query anywhere
// databasePlaceHolder is present
func dbNameReplace(query, dbName string) QueryModifier {
	return func(string) string {
		return strings.Replace(query, databasePlaceHolder, dbName, -1)
	}
}

// databaseDefinitions definitions for Database Queries
var databaseDefinitions = []*QueryDefinition{
	{
		query: `select 
		t1.instance_name as db_name,
		t1.cntr_value as log_growth
		from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE object_name = 'SQLServer:Databases' and counter_name = 'Log Growths' and instance_name NOT IN ('_Total', 'mssqlsystemresource')) t1`,
		dataModels: []struct {
			DBName    string `db:"db_name"`
			LogGrowth int    `db:"log_growth" metric_name:"log.transactionGrowth" source_type:"gauge"`
		}{},
	}, {
		query: `select 
		DB_NAME(database_id) AS db_name,
		SUM(io_stall_write_ms) + SUM(num_of_writes) as io_stalls
		FROM sys.dm_io_virtual_file_stats(null,null)
		GROUP BY database_id`,
		dataModels: []struct {
			DBName   string `db:"db_name"`
			IOStalls int    `db:"io_stalls" metric_name:"io.stallInMilliseconds" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT
		DB_NAME(database_id) AS db_name,
		COUNT_BIG(*) * (8*1024) AS buffer_pool_size
		FROM sys.dm_os_buffer_descriptors WITH (NOLOCK)
		WHERE database_id <> 32767 -- ResourceDB
		GROUP BY database_id`,
		dataModels: []struct {
			DBName   string `db:"db_name"`
			IOStalls int    `db:"buffer_pool_size" metric_name:"bufferpool.sizePerDatabaseInBytes" source_type:"gauge"`
		}{},
	},
	{
		query: fmt.Sprintf(`USE "%s"
		;WITH reserved_space(db_name, reserved_space_kb, reserved_space_not_used_kb)
		AS
		(
		SELECT
			DB_NAME() AS db_name,
			sum(a.total_pages)*8.0 reserved_space_kb,
			sum(a.total_pages)*8.0 -sum(a.used_pages)*8.0 reserved_space_not_used_kb
		FROM sys.partitions p
		INNER JOIN sys.allocation_units a ON p.partition_id = a.container_id
		LEFT JOIN sys.internal_tables it ON p.object_id = it.object_id
		)
		SELECT
		db_name as db_name,
		max(reserved_space_kb) * 1024 AS reserved_space,
		max(reserved_space_not_used_kb) * 1024 AS reserved_space_not_used
		FROM reserved_space
		GROUP BY db_name`, databasePlaceHolder),
		dataModels: []struct {
			DBName               string  `db:"db_name"`
			ReservedSpace        float64 `db:"reserved_space" metric_name:"pageFileTotal" source_type:"gauge"`
			ReservedSpaceNotUsed float64 `db:"reserved_space_not_used" metric_name:"pageFileAvailable" source_type:"gauge"`
		}{},
	},
}

var instanceDefinitions = []*QueryDefinition{
	{
		query: `select 
		t1.cntr_value as buffer_cache_hit_ratio,
		(t1.cntr_value * 1.0 / t2.cntr_value) * 100.0 as buffer_pool_hit_percent,
		t3.cntr_value as sql_compilations,
		t4.cntr_value as sql_recompilations,
		t5.cntr_value as user_connections,
		t6.cntr_value as lock_wait_time_ms,
		t7.cntr_value as page_splits_sec,
		t8.cntr_value as checkpoint_pages_sec,
		t9.cntr_value as deadlocks_sec,
		t10.cntr_value as user_errors,
		t11.cntr_value as kill_connection_errors,
		t12.cntr_value as batch_request_sec,
		(t13.cntr_value * 1000.0) as page_life_expectancy_ms,
		t14.cntr_value as transactions_sec,
		t15.cntr_value as forced_parameterizations_sec
		from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio') t1,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio base') t2,
		(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Compilations/sec') t3,
		(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Re-Compilations/sec') t4,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'User Connections') t5,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Lock Wait Time (ms)' AND instance_name = '_Total') t6,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Page Splits/sec') t7,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Checkpoint pages/sec') t8,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total') t9,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name = 'User Errors') t10,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name like 'Kill Connection Errors%') t11,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Batch Requests/sec') t12,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13,
		(SELECT SUM(cntr_value) as cntr_value FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Transactions/sec') t14,
		(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Forced Parameterizations/sec') t15`,
		dataModels: &[]struct {
			BufferCacheHitRatio *int `db:"buffer_cache_hit_ratio" metric_name:"buffer.cacheHitRatio" source_type:"gauge"`
			BufferPoolHitPercent *float64 `db:"buffer_pool_hit_percent" metric_name:"system.bufferPoolHit" source_type:"gauge"`
			SQLCompilations *int `db:"sql_compilations" metric_name:"stats.sqlCompilationsPerSecond" source_type:"rate"`
			SQLRecompilations *int `db:"sql_recompilations" metric_name:"stats.sqlRecompilationsPerSecond" source_type:"rate"`
			UserConnections *int `db:"user_connections" metric_name:"stats.connections" source_type:"gauge"`
			LockWaitTimeMs *int `db:"lock_wait_time_ms" metric_name:"stats.lockWaitsPerSecond" source_type:"gauge"`
			PageSplitsSec *int `db:"page_splits_sec" metric_name:"access.pageSplitsPerSecond" source_type:"gauge"`
			CheckpointPagesSec *int `db:"checkpoint_pages_sec" metric_name:"buffer.checkpointPagesPerSecond" source_type:"gauge"`
			DeadlocksSec *int `db:"deadlocks_sec" metric_name:"stats.deadlocksPerSecond" source_type:"gauge"`
			UserErrors *int `db:"user_errors" metric_name:"stats.userErrorsPerSecond" source_type:"rate"`
			KillConnectionErrors *int `db:"kill_connection_errors" metric_name:"stats.killConnectionErrorsPerSecond" source_type:"rate"`
			BatchRequestSec *int `db:"batch_request_sec" metric_name:"bufferpool.batchRequestsPerSecond" source_type:"gauge"`
			PageLifeExpectancySec *float64 `db:"page_life_expectancy_ms" metric_name:"bufferpool.pageLifeExpectancyInMilliseconds" source_type:"gauge"`
			TransactionsSec *int `db:"transactions_sec" metric_name:"instance.transactionsPerSecond" source_type:"gauge"`
			ForcedParameterizationsSec *int `db:"forced_parameterizations_sec" metric_name:"instance.forcedParameterizationsPerSecond" source_type:"gauge"`
		}{},
	},
}