package metrics

import (
	"fmt"
	"strings"

	"github.com/newrelic/nri-mssql/src/database"
)

// databasePlaceHolder placeholder for Database name in a query
const databasePlaceHolder = "%DATABASE%"

// dbNameReplace inserts the dbName into a query anywhere
// databasePlaceHolder is present
func dbNameReplace(dbName string) QueryModifier {
	return func(query string) string {
		return strings.Replace(query, databasePlaceHolder, dbName, -1)
	}
}

// databaseDefinitions definitions for Database Queries
var databaseDefinitions = []*QueryDefinition{
	{
		query: `select 
		RTRIM(t1.instance_name) as db_name,
		t1.cntr_value as log_growth
		from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE object_name = 'SQLServer:Databases' and counter_name = 'Log Growths' and instance_name NOT IN ('_Total', 'mssqlsystemresource')) t1`,
		dataModels: &[]struct {
			database.DataModel
			LogGrowth int `db:"log_growth" metric_name:"log.transactionGrowth" source_type:"gauge"`
		}{},
	}, {
		query: `select 
		DB_NAME(database_id) AS db_name,
		SUM(io_stall_write_ms) + SUM(num_of_writes) as io_stalls
		FROM sys.dm_io_virtual_file_stats(null,null)
		GROUP BY database_id`,
		dataModels: &[]struct {
			database.DataModel
			IOStalls int `db:"io_stalls" metric_name:"io.stallInMilliseconds" source_type:"gauge"`
		}{},
	},
	{
		query: `SELECT
		DB_NAME(database_id) AS db_name,
		COUNT_BIG(*) * (8*1024) AS buffer_pool_size
		FROM sys.dm_os_buffer_descriptors WITH (NOLOCK)
		WHERE database_id <> 32767 -- ResourceDB
		GROUP BY database_id`,
		dataModels: &[]struct {
			database.DataModel
			IOStalls int `db:"buffer_pool_size" metric_name:"bufferpool.sizePerDatabaseInBytes" source_type:"gauge"`
		}{},
	},
}

var specificDatabaseDefinitions = []*QueryDefinition{
	{
		query: fmt.Sprintf(`USE "%s"
		;WITH reserved_space(db_name, reserved_space_kb, reserved_space_not_used_kb)
		AS
		(
		SELECT
			DB_NAME() AS db_name,
			sum(a.total_pages)*8.0 reserved_space_kb,
			sum(a.total_pages)*8.0 -sum(a.used_pages)*8.0 reserved_space_not_used_kb
		FROM sys.partitions p with (nolock)
		INNER JOIN sys.allocation_units a WITH (NOLOCK) ON p.partition_id = a.container_id
		LEFT JOIN sys.internal_tables it WITH (NOLOCK) ON p.object_id = it.object_id
		)
		SELECT
		db_name as db_name,
		max(reserved_space_kb) * 1024 AS reserved_space,
		max(reserved_space_not_used_kb) * 1024 AS reserved_space_not_used
		FROM reserved_space
		GROUP BY db_name`, databasePlaceHolder),
		dataModels: &[]struct {
			database.DataModel
			ReservedSpace        float64 `db:"reserved_space" metric_name:"pageFileTotal" source_type:"gauge"`
			ReservedSpaceNotUsed float64 `db:"reserved_space_not_used" metric_name:"pageFileAvailable" source_type:"gauge"`
		}{},
	},
}
