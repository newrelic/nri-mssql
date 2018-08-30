package main

// QueryDefinition defines a single query with it's associated
// data model which has struct tags for metric.Set
type QueryDefinition struct {
	Query      string
	DataModels interface{}
}

var databaseDefinitions = []*QueryDefinition{
	{
		Query: `select 
		t1.instance_name as db_name,
		t1.cntr_value as log_growth
		from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE object_name = 'SQLServer:Databases' and counter_name = 'Log Growths' and instance_name NOT IN ('_Total', 'mssqlsystemresource')) t1`,
		DataModels: []struct {
			DBName    string `db:"db_name"`
			LogGrowth int    `db:"log_growth" metric_name:"log.transactionGrowth" source_type:"gauge"`
		}{},
	}, {
		Query: `select 
		DB_NAME(database_id) AS db_name,
		SUM(io_stall_write_ms) + SUM(num_of_writes) as io_stalls
		FROM sys.dm_io_virtual_file_stats(null,null)
		GROUP BY database_id`,
		DataModels: []struct {
			DBName   string `db:"db_name"`
			IOStalls int    `db:"io_stalls" metric_name:"io.stallInMilliseconds" source_type:"gauge"`
		}{},
	},
	{
		Query: `SELECT
		DB_NAME(database_id) AS db_name,
		COUNT_BIG(*) * (8*1024) AS buffer_pool_size
		FROM sys.dm_os_buffer_descriptors WITH (NOLOCK)
		WHERE database_id <> 32767 -- ResourceDB
		GROUP BY database_id`,
		DataModels: []struct {
			DBName   string `db:"db_name"`
			IOStalls int    `db:"buffer_pool_size" metric_name:"bufferpool.sizePerDatabaseInBytes" source_type:"gauge"`
		}{},
	},
}
