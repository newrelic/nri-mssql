package queryAnalysis

//
//import (
//	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
//	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
//	"github.com/newrelic/infra-integrations-sdk/v3/integration"
//	"github.com/newrelic/infra-integrations-sdk/v3/log"
//	"github.com/newrelic/nri-mssql/src/args"
//	connection2 "github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
//	"time"
//)
//
//// AnalyzeWaits performs wait analysis of queries
//func AnalyzeWaits(instanceEntity *integration.Entity, con *connection2.SQLConnection, arguments args.ArgumentList) {
//
//	query := "DECLARE @sql NVARCHAR(MAX) = '';DECLARE @dbName NVARCHAR(128);DECLARE @resultTable TABLE (query_id NVARCHAR(64), database_name NVARCHAR(128), query_text NVARCHAR(MAX), custom_query_type NVARCHAR(64), wait_category NVARCHAR(128), total_wait_time_ms FLOAT, avg_wait_time_ms FLOAT, wait_event_count BIGINT, collection_timestamp DATETIME);DECLARE db_cursor CURSOR FOR SELECT name FROM sys.databases WHERE state_desc = 'ONLINE' AND is_query_store_on = 1 AND database_id > 4;OPEN db_cursor;FETCH NEXT FROM db_cursor INTO @dbName;WHILE @@FETCH_STATUS = 0 BEGIN SET @sql = N'USE ' + QUOTENAME(@dbName) + ';WITH WaitStats AS (SELECT ''' + QUOTENAME(@dbName) + ''' AS database_name, qs.query_sql_text AS query_text, CONVERT(VARCHAR(32), HASHBYTES(''SHA2_256'', qs.query_sql_text), 2) AS query_id, ws.wait_category_desc AS wait_category, CAST(SUM(ws.total_query_wait_time_ms) AS FLOAT) AS total_wait_time_ms, CASE WHEN SUM(rs.count_executions) > 0 THEN CAST(SUM(ws.total_query_wait_time_ms) / SUM(rs.count_executions) AS FLOAT) ELSE 0 END AS avg_wait_time_ms, SUM(rs.count_executions) AS wait_event_count, GETUTCDATE() AS collection_timestamp, qsq.query_hash FROM sys.query_store_wait_stats ws INNER JOIN sys.query_store_plan qsp ON ws.plan_id = qsp.plan_id INNER JOIN sys.query_store_query qsq ON qsp.query_id = qsq.query_id INNER JOIN sys.query_store_query_text qs ON qsq.query_text_id = qs.query_text_id INNER JOIN sys.query_store_runtime_stats rs ON ws.plan_id = rs.plan_id AND ws.runtime_stats_interval_id = rs.runtime_stats_interval_id WHERE rs.last_execution_time >= DATEADD(hour, -1, GETUTCDATE()) GROUP BY qs.query_sql_text, qsq.query_hash, ws.wait_category_desc), DatabaseInfo AS (SELECT DISTINCT cp.plan_handle, DB_NAME(t.dbid) AS database_name, th.query_hash FROM sys.dm_exec_cached_plans cp CROSS APPLY sys.dm_exec_query_plan(cp.plan_handle) p CROSS APPLY sys.dm_exec_sql_text(cp.plan_handle) t CROSS APPLY (SELECT query_hash FROM sys.dm_exec_query_stats qs WHERE qs.plan_handle = cp.plan_handle) th WHERE DB_NAME(t.dbid) NOT IN (''master'', ''tempdb'', ''model'', ''msdb'')) SELECT ws.query_id, di.database_name, ws.query_text, ''waitTypesDetails'' AS custom_query_type, ws.wait_category, ws.total_wait_time_ms, ws.avg_wait_time_ms, ws.wait_event_count, ws.collection_timestamp FROM WaitStats ws LEFT JOIN DatabaseInfo di ON ws.query_hash = di.query_hash WHERE di.database_name IS NOT NULL;';INSERT INTO @resultTable EXEC sp_executesql @sql;FETCH NEXT FROM db_cursor INTO @dbName;END CLOSE db_cursor;DEALLOCATE db_cursor;SELECT TOP 10 * FROM @resultTable ORDER BY total_wait_time_ms DESC;"
//
//	// Capture the rows from the query
//
//	rows, err := con.Queryx(query)
//	if err != nil {
//		log.Error("Query execution failed: %v", err)
//		return
//	}
//	defer rows.Close()
//
//	// Process and log each row
//	for rows.Next() {
//		var waitQueryInfo WaitQuery
//		if err := rows.StructScan(&waitQueryInfo); err != nil {
//			log.Error("Error scanning row: %v", err)
//			continue
//		}
//
//		// Create a new metric set for each row
//		metricSet := instanceEntity.NewMetricSet("MssqlWaitAnalysisSample",
//			attribute.Attribute{Key: "queryID", Value: waitQueryInfo.QueryID},
//			attribute.Attribute{Key: "databaseName", Value: waitQueryInfo.DatabaseName},
//			attribute.Attribute{Key: "queryText", Value: waitQueryInfo.QueryText},
//		)
//
//		// Add all the fields to the metric set
//
//		if err := metricSet.SetMetric("customQueryType", waitQueryInfo.CustomQueryType, metric.ATTRIBUTE); err != nil {
//			log.Error("Error setting metric customQueryType: %v", err)
//		}
//		if err := metricSet.SetMetric("waitCategory", waitQueryInfo.WaitCategory, metric.ATTRIBUTE); err != nil {
//			log.Error("Error setting metric waitCategory: %v", err)
//		}
//		if err := metricSet.SetMetric("totalWaitTimeMs", waitQueryInfo.TotalWaitTimeMs, metric.GAUGE); err != nil {
//			log.Error("Error setting metric totalWaitTimeMs: %v", err)
//		}
//		if err := metricSet.SetMetric("avgWaitTimeMs", waitQueryInfo.AvgWaitTimeMs, metric.GAUGE); err != nil {
//			log.Error("Error setting metric avgWaitTimeMs: %v", err)
//		}
//		if err := metricSet.SetMetric("waitEventCount", waitQueryInfo.WaitEventCount, metric.GAUGE); err != nil {
//			log.Error("Error setting metric waitEventCount: %v", err)
//		}
//		if err := metricSet.SetMetric("collectionTimestamp", waitQueryInfo.CollectionTimestamp.Format(time.RFC3339), metric.ATTRIBUTE); err != nil {
//			log.Error("Error setting metric collectionTimestamp: %v", err)
//		}
//
//		log.Info("Metrics set for wait analysis: QueryID: %s, QueryText: %s   Database: %s, WaitCategory: %s, TotalWaitTimeMs: %f, AvgWaitTimeMs: %f, WaitEventCount: %d, CollectionTimestamp: %s",
//			waitQueryInfo.QueryID, waitQueryInfo.QueryText, waitQueryInfo.DatabaseName, waitQueryInfo.WaitCategory, waitQueryInfo.TotalWaitTimeMs, waitQueryInfo.AvgWaitTimeMs, waitQueryInfo.WaitEventCount, waitQueryInfo.CollectionTimestamp.Format(time.RFC3339))
//	}
//
//	// Check for any error encountered during iteration
//	if err = rows.Err(); err != nil {
//		log.Error("Error iterating rows: %v", err)
//	}
//}
