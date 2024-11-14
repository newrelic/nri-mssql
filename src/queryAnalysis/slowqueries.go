package queryAnalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

// topNSlowQueryDetails struct to hold query results.
type topNSlowQueryDetails struct {
	QueryID                *string  `db:"query_id"`
	QueryText              *string  `db:"query_text"`
	DatabaseName           *string  `db:"database_name"`
	SchemaName             *string  `db:"schema_name"`
	LastExecutionTimestamp *string  `db:"last_execution_timestamp"`
	ExecutionCount         *int64   `db:"execution_count"`
	AvgCPUTimeMS           *float64 `db:"avg_cpu_time_ms"`
	AvgElapsedTimeMS       *float64 `db:"avg_elapsed_time_ms"`
	AvgDiskReads           *float64 `db:"avg_disk_reads"`
	AvgDiskWrites          *float64 `db:"avg_disk_writes"`
	StatementType          *string  `db:"statement_type"`
	CollectionTimestamp    *string  `db:"collection_timestamp"`
}

// AnalyzeSlowQueries analyzes the top 10 slowest queries
func AnalyzeSlowQueries(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	log.Info("Querying SQL Server for top N slow queries")

	var getTopNSlowQueryDetailsQuery = `DECLARE @TopN INT = 5; WITH QueryStats AS(SELECT TOP (@TopN) qs.plan_handle, qs.sql_handle, SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, (CASE qs.statement_end_offset WHEN -1 THEN DATALENGTH(qt.text) ELSE qs.statement_end_offset END - qs.statement_start_offset) / 2 + 1) AS query_text, CONVERT(VARCHAR(32), HASHBYTES('SHA2_256', SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, (CASE qs.statement_end_offset WHEN -1 THEN DATALENGTH(qt.text) ELSE qs.statement_end_offset END - qs.statement_start_offset) / 2 + 1)), 2) AS query_id, qs.last_execution_time, qs.execution_count, (qs.total_worker_time / qs.execution_count) / 1000.0 AS avg_cpu_time_ms, (qs.total_elapsed_time / qs.execution_count) / 1000.0 AS avg_elapsed_time_ms, (qs.total_logical_reads / qs.execution_count) AS avg_disk_reads, (qs.total_logical_writes / qs.execution_count) AS avg_disk_writes, CASE WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'SELECT' THEN 'SELECT' WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'INSERT' THEN 'INSERT' WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'UPDATE' THEN 'UPDATE' WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'DELETE' THEN 'DELETE' ELSE 'OTHER' END AS statement_type, CONVERT(INT, pa.value) AS database_id, qt.text AS anonymized_query_text FROM sys.dm_exec_query_stats qs CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS qt JOIN sys.dm_exec_cached_plans cp ON qs.plan_handle = cp.plan_handle CROSS APPLY sys.dm_exec_plan_attributes(cp.plan_handle) AS pa WHERE qs.execution_count > 0 AND pa.attribute = 'dbid' AND DB_NAME(CONVERT(INT, pa.value)) NOT IN ('master', 'model', 'msdb', 'tempdb') AND qs.last_execution_time >= DATEADD(SECOND, -15, GETUTCDATE()) AND qt.text NOT LIKE '%sys.%' AND qt.text NOT LIKE '%INFORMATION_SCHEMA%' AND qt.text NOT LIKE '%schema_name()%' AND qt.text IS NOT NULL AND LTRIM(RTRIM(qt.text)) <> '' ORDER BY avg_elapsed_time_ms DESC) SELECT query_id, anonymized_query_text as query_text, DB_NAME(database_id) AS database_name, COALESCE(OBJECT_SCHEMA_NAME(qt.objectid, database_id), 'N/A') AS schema_name, FORMAT(qs.last_execution_time AT TIME ZONE 'UTC', 'yyyy-MM-ddTHH:mm:ssZ') AS last_execution_timestamp, execution_count, avg_cpu_time_ms, avg_elapsed_time_ms, avg_disk_reads, avg_disk_writes, statement_type, FORMAT(SYSDATETIMEOFFSET() AT TIME ZONE 'UTC', 'yyyy-MM-ddTHH:mm:ssZ') AS collection_timestamp FROM QueryStats qs CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) qt;`

	log.Info("Executing query to get top N slow query details.")

	// Slice to hold query results.
	slowQueryModels := make([]topNSlowQueryDetails, 0)

	// Execute the query and store the results in the slowQueryModels slice.
	rows, err := connection.Queryx(getTopNSlowQueryDetailsQuery)
	if err != nil {
		log.Error("Could not execute query: %s", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var model topNSlowQueryDetails
		if err := rows.StructScan(&model); err != nil {
			log.Error("Could not scan row: %s", err.Error())
			continue
		}
		slowQueryModels = append(slowQueryModels, model)
	}

	log.Info("Number of records retrieved: %d", len(slowQueryModels))

	// Log and report each result from the query.
	for _, model := range slowQueryModels {
		if model.DatabaseName == nil || model.QueryText == nil || model.QueryID == nil {
			log.Warn("Skipping entry with nil field: DatabaseName, QueryText, or QueryID")
			continue // Skip this entry if any critical field is nil
		}

		queryID := *model.QueryID
		databaseName := *model.DatabaseName
		queryText := *model.QueryText
		schemaName := *model.SchemaName
		lastExecutionTimestamp := *model.LastExecutionTimestamp
		executionCount := *model.ExecutionCount
		avgCPUTimeMS := *model.AvgCPUTimeMS
		avgElapsedTimeMS := *model.AvgElapsedTimeMS
		avgDiskReads := *model.AvgDiskReads
		avgDiskWrites := *model.AvgDiskWrites
		statementType := *model.StatementType
		collectionTimestamp := *model.CollectionTimestamp

		log.Info("Metrics set for slow query: QueryID: %s, QueryText: %s, Database: %s, Schema: %s, LastExecution: %s, ExecutionCount: %d, AvgCPUTimeMS: %f, AvgElapsedTimeMS: %f, AvgDiskReads: %f, AvgDiskWrites: %f, StatementType: %s, CollectionTimestamp: %s",
			queryID,
			queryText,
			databaseName,
			schemaName,
			lastExecutionTimestamp,
			executionCount,
			avgCPUTimeMS,
			avgElapsedTimeMS,
			avgDiskReads,
			avgDiskWrites,
			statementType,
			collectionTimestamp)

		metricSet := instanceEntity.NewMetricSet("MssqlSlowQueries",
			attribute.Attribute{Key: "queryID", Value: queryID},
			attribute.Attribute{Key: "databaseName", Value: databaseName},
			attribute.Attribute{Key: "queryText", Value: queryText},
		)

		// Add all the fields to the metric set.
		if model.SchemaName != nil {
			metricSet.SetMetric("schemaName", *model.SchemaName, metric.GAUGE)
		}
		if model.LastExecutionTimestamp != nil {
			metricSet.SetMetric("lastExecutionTimestamp", *model.LastExecutionTimestamp, metric.GAUGE)
		}
		if model.ExecutionCount != nil {
			metricSet.SetMetric("executionCount", *model.ExecutionCount, metric.GAUGE)
		}
		if model.AvgCPUTimeMS != nil {
			metricSet.SetMetric("avgCPUTimeMS", *model.AvgCPUTimeMS, metric.GAUGE)
		}
		if model.AvgElapsedTimeMS != nil {
			metricSet.SetMetric("avgElapsedTimeMS", *model.AvgElapsedTimeMS, metric.GAUGE)
		}
		if model.AvgDiskReads != nil {
			metricSet.SetMetric("avgDiskReads", *model.AvgDiskReads, metric.GAUGE)
		}
		if model.AvgDiskWrites != nil {
			metricSet.SetMetric("avgDiskWrites", *model.AvgDiskWrites, metric.GAUGE)
		}
		if model.StatementType != nil {
			metricSet.SetMetric("statementType", *model.StatementType, metric.GAUGE)
		}
		if model.CollectionTimestamp != nil {
			metricSet.SetMetric("collectionTimestamp", *model.CollectionTimestamp, metric.GAUGE)
		}
		if model.QueryText != nil {
			metricSet.SetMetric("queryText", *model.QueryText, metric.GAUGE)
		}

		log.Info("Metrics set for slow query: %s in database: %s", queryID, databaseName)
	}

	log.Info("Completed processing all slow query entries.")
}
