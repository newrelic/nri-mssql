package config

import "github.com/newrelic/nri-mssql/src/queryanalysis/models"

// Documentation: https:https://newrelic.atlassian.net/wiki/x/SYFq6g
// The above link contains all the queries, data models, and query details for QueryAnalysis.
var Queries = []models.QueryDetailsDto{
	{
		EventName: "MSSQLTopSlowQueries",
		Query: `DECLARE @IntervalSeconds INT = %d; 		-- Define the interval in seconds
				DECLARE @TopN INT = %d; 				-- Number of top queries to retrieve
				DECLARE @ElapsedTimeThreshold INT = %d; -- Elapsed time threshold in milliseconds
				DECLARE @TextTruncateLimit INT = %d; 	-- Truncate limit for query_text
				
				WITH RecentQueryIds AS (
					SELECT  
						qs.query_hash as query_id
					FROM 
						sys.dm_exec_query_stats qs
					WHERE 
						qs.execution_count > 0
						AND qs.last_execution_time >= DATEADD(SECOND, -@IntervalSeconds, SYSDATETIME())
						AND qs.sql_handle IS NOT NULL
				),
				QueryStats AS (
					SELECT
						qs.plan_handle,
						qs.sql_handle,
						LEFT(SUBSTRING(
							qt.text,
							(qs.statement_start_offset / 2) + 1,
							(
								CASE
									qs.statement_end_offset
									WHEN -1 THEN DATALENGTH(qt.text)
									ELSE qs.statement_end_offset
								END - qs.statement_start_offset
							) / 2 + 1
						), @TextTruncateLimit) AS query_text, 
						qs.query_hash AS query_id,
						qs.last_execution_time,
						qs.execution_count,
						(qs.total_worker_time / qs.execution_count) / 1000.0 AS avg_cpu_time_ms,
						(qs.total_elapsed_time / qs.execution_count) / 1000.0 AS avg_elapsed_time_ms,
						(qs.total_logical_reads / qs.execution_count) AS avg_disk_reads,
						(qs.total_logical_writes / qs.execution_count) AS avg_disk_writes,
						CASE
							WHEN UPPER(
								LTRIM(
									SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6)
								)
							) LIKE 'SELECT' THEN 'SELECT'
							WHEN UPPER(
								LTRIM(
									SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6)
								)
							) LIKE 'INSERT' THEN 'INSERT'
							WHEN UPPER(
								LTRIM(
									SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6)
								)
							) LIKE 'UPDATE' THEN 'UPDATE'
							WHEN UPPER(
								LTRIM(
									SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6)
								)
							) LIKE 'DELETE' THEN 'DELETE'
							ELSE 'OTHER'
						END AS statement_type,
						CONVERT(INT, pa.value) AS database_id,
						qt.objectid
					FROM
						sys.dm_exec_query_stats qs
						CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS qt
						JOIN sys.dm_exec_cached_plans cp ON qs.plan_handle = cp.plan_handle
						CROSS APPLY sys.dm_exec_plan_attributes(cp.plan_handle) AS pa
					WHERE
						qs.query_hash IN (SELECT DISTINCT(query_id) FROM RecentQueryIds)
						AND qs.execution_count > 0
						AND pa.attribute = 'dbid'
						AND DB_NAME(CONVERT(INT, pa.value)) NOT IN ('master', 'model', 'msdb', 'tempdb')
						AND qt.text NOT LIKE '%%sys.%%'
						AND qt.text NOT LIKE '%%INFORMATION_SCHEMA%%'
						AND qt.text NOT LIKE '%%schema_name()%%'
						AND qt.text IS NOT NULL
						AND LTRIM(RTRIM(qt.text)) <> ''
						-- ✅ Removed Query Store dependency for broader compatibility
				)
				SELECT
					TOP (@TopN) qs.query_id,
					MIN(qs.query_text) AS query_text,
					DB_NAME(MIN(qs.database_id)) AS database_name,
					COALESCE(
						OBJECT_SCHEMA_NAME(MIN(qs.objectid), MIN(qs.database_id)),
						'N/A'
					) AS schema_name,
					FORMAT(
						MAX(qs.last_execution_time) AT TIME ZONE 'UTC',
						'yyyy-MM-ddTHH:mm:ssZ'
					) AS last_execution_timestamp,
					SUM(qs.execution_count) AS execution_count,
					AVG(qs.avg_cpu_time_ms) AS avg_cpu_time_ms,
					AVG(qs.avg_elapsed_time_ms) AS avg_elapsed_time_ms,
					AVG(qs.avg_disk_reads) AS avg_disk_reads,
					AVG(qs.avg_disk_writes) AS avg_disk_writes,
					 MAX(qs.statement_type) AS statement_type,
					FORMAT(
						SYSDATETIMEOFFSET() AT TIME ZONE 'UTC',
						'yyyy-MM-ddTHH:mm:ssZ'
					) AS collection_timestamp
				FROM
					QueryStats qs
				GROUP BY
					qs.query_id
				HAVING
					AVG(qs.avg_elapsed_time_ms) > @ElapsedTimeThreshold
				ORDER BY
					avg_elapsed_time_ms DESC;`,
		Type: "slowQueries",
	},
	{
		EventName: "MSSQLWaitTimeAnalysis",
		Query: `DECLARE @TopN INT = %d; 				-- Number of results to retrieve
				DECLARE @TextTruncateLimit INT = %d; 	-- Truncate limit for query_text
				
				-- Get current waiting requests and recent wait statistics
				WITH ActiveRequests AS (
					SELECT 
						req.session_id,
						req.request_id,
						req.wait_type,
						req.wait_time,
						req.total_elapsed_time,
						req.database_id,
						COALESCE(
							LEFT(qt.text, @TextTruncateLimit),
							'Query text not available'
						) AS query_text,
						req.sql_handle,
						req.start_time
					FROM 
						sys.dm_exec_requests req WITH (NOLOCK)
					OUTER APPLY 
						sys.dm_exec_sql_text(req.sql_handle) qt
					WHERE 
						req.database_id > 4  -- Exclude system databases
						AND req.session_id > 50  -- Exclude system sessions
						AND (
							req.wait_time > 100  -- Currently waiting > 100ms
							OR req.total_elapsed_time > 1000  -- Long running queries > 1s
						)
						AND COALESCE(qt.text, '') NOT LIKE '%sp_reset_connection%'
						AND COALESCE(qt.text, '') NOT LIKE '%sys.%'
						AND COALESCE(qt.text, '') NOT LIKE '%INFORMATION_SCHEMA%'
						AND COALESCE(qt.text, '') <> ''
				),
				-- Also get wait stats from historical data
				WaitStats AS (
					SELECT 
						ws.wait_type,
						ws.waiting_tasks_count,
						ws.wait_time_ms,
						ws.signal_wait_time_ms,
						(ws.wait_time_ms - ws.signal_wait_time_ms) AS resource_wait_time_ms
					FROM 
						sys.dm_os_wait_stats ws WITH (NOLOCK)
					WHERE 
						ws.wait_time_ms > 0
						AND ws.wait_type NOT IN (
							'CLR_SEMAPHORE', 'LAZYWRITER_SLEEP', 'RESOURCE_QUEUE', 
							'SLEEP_TASK', 'SLEEP_SYSTEMTASK', 'SQLTRACE_BUFFER_FLUSH',
							'WAITFOR', 'LOGMGR_QUEUE', 'CHECKPOINT_QUEUE', 'REQUEST_FOR_DEADLOCK_SEARCH',
							'XE_TIMER_EVENT', 'BROKER_TO_FLUSH', 'BROKER_TASK_STOP', 'CLR_MANUAL_EVENT',
							'CLR_AUTO_EVENT', 'DISPATCHER_QUEUE_SEMAPHORE', 'FT_IFTS_SCHEDULER_IDLE_WAIT',
							'XE_DISPATCHER_WAIT', 'XE_DISPATCHER_JOIN', 'SQLTRACE_INCREMENTAL_FLUSH_SLEEP'
						)
				),
				CombinedResults AS (
					-- Current active waiting requests
					SELECT 
						CONVERT(VARBINARY(255), ar.sql_handle) AS query_id,
						DB_NAME(ar.database_id) AS database_name,
						ar.query_text,
						ar.wait_type AS wait_category,
						CAST(ar.wait_time AS FLOAT) AS total_wait_time_ms,
						CAST(ar.wait_time AS FLOAT) AS avg_wait_time_ms,
						1 AS wait_event_count,
						COALESCE(ar.start_time, SYSDATETIME()) AS last_execution_time,
						SYSDATETIME() AS collection_timestamp
					FROM 
						ActiveRequests ar
					WHERE 
						ar.query_text IS NOT NULL
						AND LEN(LTRIM(RTRIM(ar.query_text))) > 10
						AND DB_NAME(ar.database_id) IS NOT NULL  -- Ensure valid database name
				)
				SELECT TOP (@TopN)
					query_id,
					database_name,
					query_text,
					wait_category,
					total_wait_time_ms,
					avg_wait_time_ms,
					wait_event_count,
					last_execution_time,
					collection_timestamp
				FROM 
					CombinedResults
				WHERE
					database_name IS NOT NULL  -- Exclude any NULL database names
				ORDER BY 
					total_wait_time_ms DESC
				OPTION (MAXDOP 1);`,
		Type: "waitAnalysis",
	},
	{
		EventName: "MSSQLBlockingSessionQueries",
		Query: `DECLARE @Limit INT = %d; -- Define the limit for the number of rows returned
				DECLARE @TextTruncateLimit INT = %d; -- Define the truncate limit for the query text
				
				-- Direct blocking detection - simplified and more reliable approach
				WITH DirectBlocking AS (
					SELECT
						req.blocking_session_id AS blocking_spid,
						req.session_id AS blocked_spid,
						req.wait_type AS wait_type,
						req.wait_time / 1000.0 AS wait_time_in_seconds,
						req.start_time AS start_time,
						sess.status AS blocked_status,
						req.command AS command_type,
						req.database_id AS database_id,
						req.sql_handle AS blocked_sql_handle,
						blocking_req.sql_handle AS blocking_sql_handle,
						blocking_req.start_time AS blocking_start_time,
						blocking_sess.status AS blocking_status,
						req.total_elapsed_time / 1000.0 AS total_elapsed_time_seconds,
						req.cpu_time / 1000.0 AS cpu_time_seconds,
						req.logical_reads,
						req.writes
					FROM
						sys.dm_exec_requests AS req WITH (NOLOCK)
					LEFT JOIN sys.dm_exec_requests AS blocking_req WITH (NOLOCK) 
						ON blocking_req.session_id = req.blocking_session_id
					LEFT JOIN sys.dm_exec_sessions AS sess WITH (NOLOCK) 
						ON sess.session_id = req.session_id
					LEFT JOIN sys.dm_exec_sessions AS blocking_sess WITH (NOLOCK) 
						ON blocking_sess.session_id = req.blocking_session_id
					WHERE
						req.blocking_session_id > 0  -- Changed from != 0 to > 0 for clarity
						AND req.session_id != req.blocking_session_id
						AND req.database_id > 4
						AND req.wait_time > 100  -- Only significant waits > 100ms
				)
				SELECT TOP (@Limit)
					db.blocking_spid,
					db.blocking_status,
					db.blocked_spid,
					db.blocked_status,
					db.wait_type,
					db.wait_time_in_seconds,
					db.command_type,
					db.start_time AS blocked_query_start_time,
					DB_NAME(db.database_id) AS database_name,
					CASE
						WHEN blocking_sql.text IS NOT NULL THEN 
							LEFT(blocking_sql.text, @TextTruncateLimit)
						WHEN input_buffer.event_info IS NOT NULL THEN 
							LEFT(input_buffer.event_info, @TextTruncateLimit)
						ELSE 'No blocking query text available'
					END AS blocking_query_text,
					LEFT(COALESCE(blocked_sql.text, 'No blocked query text available'), @TextTruncateLimit) AS blocked_query_text,
					db.total_elapsed_time_seconds,
					db.cpu_time_seconds,
					db.logical_reads,
					db.writes,
					SYSDATETIME() AS collection_timestamp
				FROM
					DirectBlocking db
				OUTER APPLY sys.dm_exec_sql_text(db.blocking_sql_handle) AS blocking_sql
				OUTER APPLY sys.dm_exec_sql_text(db.blocked_sql_handle) AS blocked_sql
				OUTER APPLY sys.dm_exec_input_buffer(db.blocking_spid, NULL) AS input_buffer
				WHERE
					DB_NAME(db.database_id) IS NOT NULL  -- Ensure valid database name
				ORDER BY
					db.wait_time_in_seconds DESC,  -- Longest waits first
					db.start_time ASC  -- Oldest requests first
				OPTION (MAXDOP 1);`,
		Type: "blockingSessions",
	},
}

// ExecutionPlanQuery holds the SQL query for fetching execution plans.
const ExecutionPlanQueryTemplate = `
DECLARE @TopN INT = %d; 
DECLARE @ElapsedTimeThreshold INT = %d;  -- Define the elapsed time threshold in milliseconds
DECLARE @QueryIDs NVARCHAR(1000) = '%s';      -- Change the query ID to a string
DECLARE @IntervalSeconds INT = %d;       -- Define the interval in seconds (e.g., 3600 for the last hour)
DECLARE @TextTruncateLimit INT = %d;     -- Define the dynamic limit for truncation of SQL text

-- Declare and fill the temporary table
DECLARE @QueryIdTable TABLE (QueryId BINARY(8));

-- Use a conversion that properly removes the 0x prefix and casts to BINARY
INSERT INTO @QueryIdTable (QueryId)
SELECT CONVERT(BINARY(8), value, 1)
FROM STRING_SPLIT(@QueryIDs, ',');

WITH XMLNAMESPACES (DEFAULT 'http://schemas.microsoft.com/sqlserver/2004/07/showplan'),
TopPlans AS (
    SELECT TOP (@TopN)
        qs.plan_handle,
        qs.query_hash as query_id,
        qs.query_plan_hash AS query_plan_id,
        LEFT(st.text, @TextTruncateLimit) AS sql_text,
        qs.execution_count as execution_count,
        COALESCE((qs.total_elapsed_time / NULLIF(qs.execution_count, 0)) / 1000, 0) AS avg_elapsed_time_ms,
        qp.query_plan
    FROM sys.dm_exec_query_stats AS qs
    CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS st
    CROSS APPLY sys.dm_exec_query_plan(qs.plan_handle) AS qp
	WHERE qs.query_hash IN (SELECT QueryId FROM @QueryIdTable)
	AND qs.last_execution_time BETWEEN DATEADD(SECOND, -@IntervalSeconds, SYSDATETIME()) AND SYSDATETIME()
    AND COALESCE((qs.total_elapsed_time / NULLIF(qs.execution_count, 0)) / 1000, 0) > @ElapsedTimeThreshold
    ORDER BY avg_elapsed_time_ms DESC
),
PlanNodes AS (
    SELECT
        tp.query_id,
        tp.sql_text,
        tp.plan_handle,
        tp.query_plan_id,
        tp.avg_elapsed_time_ms,
        tp.execution_count,
        COALESCE(n.value('(@NodeId)[1]', 'INT'), 0) AS NodeId,
        COALESCE(n.value('(@PhysicalOp)[1]', 'VARCHAR(50)'), 'N/A') AS PhysicalOp,
        COALESCE(n.value('(@LogicalOp)[1]', 'VARCHAR(50)'), 'N/A') AS LogicalOp,
        COALESCE(n.value('(@EstimateRows)[1]', 'FLOAT'), 0.0) AS EstimateRows,
        COALESCE(n.value('(@EstimateIO)[1]', 'FLOAT'), 0.0) AS EstimateIO,
        COALESCE(n.value('(@EstimateCPU)[1]', 'FLOAT'), 0.0) AS EstimateCPU,
        COALESCE(n.value('(@AvgRowSize)[1]', 'FLOAT'), 0.0) AS AvgRowSize,
        COALESCE(n.value('(@EstimatedExecutionMode)[1]', 'VARCHAR(50)'), 'N/A') AS EstimatedExecutionMode,
        COALESCE(n.value('(@EstimatedTotalSubtreeCost)[1]', 'FLOAT'), 0.0) AS TotalSubtreeCost,
        COALESCE(n.value('(@EstimatedOperatorCost)[1]', 'FLOAT'), 0.0) AS EstimatedOperatorCost,
        COALESCE(n.value('(MemoryGrantInfo/@GrantedMemoryKb)[1]', 'INT'), 0) AS GrantedMemoryKb,
        COALESCE(n.value('(Warnings/Warning/@SpillOccurred)[1]', 'BIT'), 0) AS SpillOccurred,
        COALESCE(n.value('(Warnings/Warning/@NoJoinPredicate)[1]', 'BIT'), 0) AS NoJoinPredicate
    FROM TopPlans AS tp
    CROSS APPLY tp.query_plan.nodes('//RelOp') AS RelOps(n)
)
SELECT *
FROM PlanNodes
ORDER BY plan_handle, NodeId;
`

// We need to use this limit of long strings that we are injesting because the logs datastore in New Relic limits the field length to 4,094 characters. Any data longer than that is truncated during ingestion.
const TextTruncateLimit = 4094

const (
	// QueryResponseTimeThresholdDefault defines the default threshold in milliseconds
	// for determining if a query is considered slow based on its response time.
	QueryResponseTimeThresholdDefault = 500

	// SlowQueryCountThresholdDefault sets the default maximum number of slow queries
	// that is ingested in an analysis cycle/interval.
	SlowQueryCountThresholdDefault = 20

	// IndividualQueryCountMax represents the maximum number of individual queries
	// that is ingested at one time for any grouped query in detailed analysis.
	IndividualQueryCountMax = 10

	// GroupedQueryCountMax specifies the maximum number of grouped queries
	// that is ingested in  an analysis cycle/interval.
	GroupedQueryCountMax = 30

	// MaxSystemDatabaseID indicates the highest database ID value considered
	// a system database, used to filter out system databases from certain operations.
	MaxSystemDatabaseID = 4
	BatchSize           = 600 // New Relic's Integration SDK imposes a limit of 1000 metrics per ingestion.To handle metric sets exceeding this limit, we process and ingest metrics in smaller chunks to ensure all data is successfully reported without exceeding the limit.

)
