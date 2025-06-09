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
		from (
      SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK)
      WHERE object_name = 'SQLServer:Databases'
        AND counter_name = 'Log Growths'
        AND RTRIM(instance_name) NOT IN ('master', 'tempdb', 'msdb', 'model', 'rdsadmin', 'distribution', 'model_msdb', 'model_replicatedmaster')
        AND instance_name NOT IN ('_Total', 'mssqlsystemresource', 'master', 'tempdb', 'msdb', 'model', 'rdsadmin', 'distribution', 'model_msdb', 'model_replicatedmaster')
    ) t1
    `,
		dataModels: &[]struct {
			database.DataModel
			LogGrowth int `db:"log_growth" metric_name:"log.transactionGrowth" source_type:"gauge"`
		}{},
	}, {
		query: `select
		DB_NAME(database_id) AS db_name,
		SUM(io_stall_write_ms) + SUM(num_of_writes) as io_stalls
		FROM sys.dm_io_virtual_file_stats(null,null)
    WHERE DB_NAME(database_id) NOT IN ('master', 'tempdb', 'msdb', 'model', 'rdsadmin', 'distribution', 'model_msdb', 'model_replicatedmaster')
		GROUP BY database_id`,
		dataModels: &[]struct {
			database.DataModel
			IOStalls int `db:"io_stalls" metric_name:"io.stallInMilliseconds" source_type:"gauge"`
		}{},
	},
}

var databaseDefinitionsForAzureSQLDatabase = []*QueryDefinition{
	{
		query: `
			SELECT 
			    sd.name AS db_name,
			    spc.cntr_value AS log_growth
			FROM sys.dm_os_performance_counters spc
			INNER JOIN sys.databases sd 
			    ON sd.physical_database_name = spc.instance_name
			WHERE spc.counter_name = 'Log Growths'
			    AND spc.object_name LIKE '%:Databases%'
			    AND sd.database_id = DB_ID()
		`,
		dataModels: &[]struct {
			database.DataModel
			LogGrowth int `db:"log_growth" metric_name:"log.transactionGrowth" source_type:"gauge"`
		}{},
	},
	{
		query: `		
			SELECT
			    DB_NAME() AS db_name,
			    SUM(io_stall) AS io_stalls
			FROM sys.dm_io_virtual_file_stats(NULL, NULL)
			WHERE database_id = DB_ID()
		`,
		dataModels: &[]struct {
			database.DataModel
			IOStalls int `db:"io_stalls" metric_name:"io.stallInMilliseconds" source_type:"gauge"`
		}{},
	},
}

func getDatabaseDefinitions(engineEdition int) []*QueryDefinition {
	switch engineEdition {
	case database.AzureSQLDatabaseEngineEditionNumber:
		return databaseDefinitionsForAzureSQLDatabase
	default:
		return databaseDefinitions
	}
}

// databaseBufferDefinitions definitions for Database Queries
var databaseBufferDefinitions = []*QueryDefinition{
	{
		query: `SELECT DB_NAME(database_id) AS db_name, buffer_pool_size * (8*1024) AS buffer_pool_size
		FROM ( SELECT database_id, COUNT_BIG(*) AS buffer_pool_size FROM sys.dm_os_buffer_descriptors a WITH (NOLOCK)
		INNER JOIN sys.sysdatabases b WITH (NOLOCK) ON b.dbid=a.database_id 
		WHERE b.dbid in (SELECT dbid FROM sys.sysdatabases WITH (NOLOCK)
		WHERE name NOT IN ('master', 'tempdb', 'msdb', 'model', 'rdsadmin', 'distribution', 'model_msdb', 'model_replicatedmaster')
		UNION ALL SELECT 32767) GROUP BY database_id) a`,
		dataModels: &[]struct {
			database.DataModel
			IOStalls int `db:"buffer_pool_size" metric_name:"bufferpool.sizePerDatabaseInBytes" source_type:"gauge"`
		}{},
	},
}

var databaseBufferDefinitionsForAzureSQLDatabase = []*QueryDefinition{
	{
		query: `
			SELECT 
			    DB_NAME() AS db_name, 
			    COUNT_BIG(*) * (8 * 1024) AS buffer_pool_size
			FROM sys.dm_os_buffer_descriptors WITH (NOLOCK) 
			WHERE database_id = DB_ID()
		`,
		dataModels: &[]struct {
			database.DataModel
			BufferPoolSize int `db:"buffer_pool_size" metric_name:"bufferpool.sizePerDatabaseInBytes" source_type:"gauge"`
		}{},
	},
}

func getDatabaseBufferDefinitions(engineEdition int) []*QueryDefinition {
	switch engineEdition {
	case database.AzureSQLDatabaseEngineEditionNumber:
		return databaseBufferDefinitionsForAzureSQLDatabase
	default:
		return databaseBufferDefinitions
	}
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

var specificDatabaseDefinitionsForAzureSQLDatabase = []*QueryDefinition{
	{
		query: `
			SELECT
				DB_NAME() AS db_name,
				sum(a.total_pages) * 8.0 * 1024 AS reserved_space,
				(sum(a.total_pages)*8.0 - sum(a.used_pages)*8.0) * 1024 AS reserved_space_not_used
			FROM sys.partitions p with (nolock)
			INNER JOIN sys.allocation_units a WITH (NOLOCK) ON p.partition_id = a.container_id
			LEFT JOIN sys.internal_tables it WITH (NOLOCK) ON p.object_id = it.object_id
		`,
		dataModels: &[]struct {
			database.DataModel
			ReservedSpace        float64 `db:"reserved_space" metric_name:"pageFileTotal" source_type:"gauge"`
			ReservedSpaceNotUsed float64 `db:"reserved_space_not_used" metric_name:"pageFileAvailable" source_type:"gauge"`
		}{},
	},
}

func getSpecificDatabaseDefinitions(engineEdition int) []*QueryDefinition {
	switch engineEdition {
	case database.AzureSQLDatabaseEngineEditionNumber:
		return specificDatabaseDefinitionsForAzureSQLDatabase
	default:
		return specificDatabaseDefinitions
	}
}
