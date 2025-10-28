package config

import "github.com/newrelic/nri-mssql/src/queryanalysis/models"

// Documentation: https:https://newrelic.atlassian.net/wiki/x/SYFq6g
// The above link contains all the queries, data models, and query details for QueryAnalysis.
var Queries = []models.QueryDetailsDto{
	{
		EventName: "MSSQLTopSlowQueries",
		Query: `DECLARE @IntervalSeconds INT = %d;      -- Define the interval in seconds
DECLARE @TextTruncateLimit INT = %d;  -- Truncate limit for query_text
DECLARE @Limit INT = 10000; -- Number of top aggregated groups to select

---------------------------------------------------------------------------------------------------------------------

WITH AggregatedStats AS (
    -- This CTE performs the GROUP BY and SUM aggregation.
    SELECT
        qs.query_hash AS query_id,
        qs.query_plan_hash,
        
        -- CAPTURE REPRESENTATIVE PLAN INFO AND OFFSETS (MAX picks one plan/offset set to represent the group)
        MAX(CONCAT(
            CONVERT(VARCHAR(64), CONVERT(binary(64), qs.plan_handle), 1),
            CONVERT(VARCHAR(10), CONVERT(varbinary(4), qs.statement_start_offset), 1),
            CONVERT(VARCHAR(10), CONVERT(varbinary(4), qs.statement_end_offset), 1)
        )) AS plan_handle_and_offsets,
        
        -- Use MAX for representative non-numeric columns
        MAX(qs.last_execution_time) AS last_execution_time,
        MAX(CONVERT(INT, pa.value)) AS database_id,
        
        -- *** ADDED: MIN(qt.objectid) is needed for the schema_name lookup ***
        MIN(qt.objectid) AS objectid,
        
        -- SUM for all numerical metrics
        SUM(qs.execution_count) AS execution_count,
        SUM(qs.total_worker_time) AS total_worker_time,
        SUM(qs.total_elapsed_time) AS total_elapsed_time,
        SUM(qs.total_logical_reads) AS total_logical_reads,
        SUM(qs.total_logical_writes) AS total_logical_writes,
        
        -- Representative Statement Type (MAX attempts to pick one type from the group)
        MAX(CASE
            WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'SELECT' THEN 'SELECT'
            WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'INSERT' THEN 'INSERT'
            WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'UPDATE' THEN 'UPDATE'
            WHEN UPPER(LTRIM(SUBSTRING(qt.text, (qs.statement_start_offset / 2) + 1, 6))) LIKE 'DELETE' THEN 'DELETE'
            ELSE 'OTHER'
        END) AS statement_type

    FROM
        sys.dm_exec_query_stats qs
        CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS qt
        CROSS APPLY sys.dm_exec_plan_attributes(qs.plan_handle) AS pa
    WHERE
        -- KEY FILTER: Only plans that ran in the last @IntervalSeconds (e.g., 15)
        qs.last_execution_time >= DATEADD(SECOND, -@IntervalSeconds, GETUTCDATE())
        AND qs.execution_count > 0
        AND pa.attribute = 'dbid'
        AND qt.text IS NOT NULL
        AND LTRIM(RTRIM(qt.text)) <> ''
        AND EXISTS (
            SELECT 1
            FROM sys.databases d
            WHERE d.database_id = CONVERT(INT, pa.value) 
        )
    -- *** GROUP BY THE HASHES ***
    GROUP BY
        qs.query_hash,
        qs.query_plan_hash
)
-- Select the final top @Limit aggregated query groups, performing text extraction.
SELECT TOP (@Limit) 
    s.query_id,
    
    -- EXTRACT the statement text using the decoded offsets and plan handle text
    LEFT(SUBSTRING(
        qt_final.text, 
        (T.statement_start_offset / 2) + 1,
        (
            (CASE T.statement_end_offset 
             WHEN -1 THEN DATALENGTH(qt_final.text)
             ELSE T.statement_end_offset
             END - T.statement_start_offset) / 2
        ) + 1
    ), @TextTruncateLimit) AS query_text,
    
    DB_NAME(s.database_id) AS database_name,

    -- *** ADDED: Schema Name lookup using objectid and database_id ***
    COALESCE(OBJECT_SCHEMA_NAME(s.objectid, s.database_id), 'N/A') AS schema_name,

    FORMAT(
        s.last_execution_time AT TIME ZONE 'UTC',
        'yyyy-MM-ddTHH:mm:ssZ'
    ) AS last_execution_timestamp,
    s.execution_count,
    s.total_worker_time,
    s.total_elapsed_time,
    s.total_logical_reads,
    s.total_logical_writes,
    s.statement_type,

    FORMAT(
        SYSDATETIMEOFFSET() AT TIME ZONE 'UTC',
        'yyyy-MM-ddTHH:mm:ssZ'
    ) AS collection_timestamp
FROM
    AggregatedStats s
-- Decode the representative plan info captured in the CTE
CROSS APPLY (
    SELECT
        CONVERT(VARBINARY(64), CONVERT(BINARY(64), SUBSTRING(s.plan_handle_and_offsets, 1, 64), 1)) AS plan_handle,
        CONVERT(INT, CONVERT(VARBINARY(10), SUBSTRING(s.plan_handle_and_offsets, 65, 10), 1)) AS statement_start_offset,
        CONVERT(INT, CONVERT(VARBINARY(10), SUBSTRING(s.plan_handle_and_offsets, 75, 10), 1)) AS statement_end_offset
) T
-- Join to get the full query batch text for extraction
CROSS APPLY sys.dm_exec_sql_text(T.plan_handle) qt_final


ORDER BY
    s.last_execution_time DESC`,
		Type: "slowQueries",
	},
	{
		EventName: "MSSQLWaitTimeAnalysis",
		Query: `-- Original complex query with Query Store and cursors (commented out for performance reasons)
				-- DECLARE @TopN INT = %d; 				-- Number of results to retrieve
				-- DECLARE @TextTruncateLimit INT = %d; 	-- Truncate limit for query_text
				-- DECLARE @sql NVARCHAR(MAX) = '';
				-- DECLARE @dbName NVARCHAR(128);
				-- DECLARE @resultTable TABLE(
				--   query_id VARBINARY(255),
				--   database_name NVARCHAR(128),
				--   query_text NVARCHAR(MAX),
				--   wait_category NVARCHAR(128),
				--   total_wait_time_ms FLOAT,
				--   avg_wait_time_ms FLOAT,
				--   wait_event_count INT,
				--   last_execution_time DATETIME,
				--   collection_timestamp DATETIME
				-- );
				-- 
				-- IF CURSOR_STATUS('global', 'db_cursor') > -1
				-- BEGIN
				--   CLOSE db_cursor;
				--   DEALLOCATE db_cursor;
				-- END
				-- 
				-- DECLARE db_cursor CURSOR FOR
				-- SELECT name FROM sys.databases
				-- WHERE state_desc = 'ONLINE'
				-- AND database_id > 4;
				-- 
				-- OPEN db_cursor;
				-- FETCH NEXT FROM db_cursor INTO @dbName;
				-- 
				-- WHILE @@FETCH_STATUS = 0
				-- BEGIN
				--   SET @sql = N'USE ' + QUOTENAME(@dbName) + ';
				--   WITH LatestInterval AS (
				-- 	SELECT 
				-- 	  qsqt.query_sql_text, 
				-- 	  MAX(ws.runtime_stats_interval_id) AS max_runtime_stats_interval_id
				-- 	FROM 
				-- 	  sys.query_store_wait_stats ws
				-- 	INNER JOIN 
				-- 	  sys.query_store_plan qsp ON ws.plan_id = qsp.plan_id
				-- 	INNER JOIN 
				-- 	  sys.query_store_query AS qsq ON qsp.query_id = qsq.query_id
				-- 	INNER JOIN 
				-- 	  sys.query_store_query_text AS qsqt ON qsqt.query_text_id = qsq.query_text_id
				-- 	WHERE 
				-- 	  qsqt.query_sql_text NOT LIKE ''%%sys.%%''
				-- 	  AND qsqt.query_sql_text NOT LIKE ''%%INFORMATION_SCHEMA%%''
				-- 	GROUP BY 
				-- 	  qsqt.query_sql_text 
				--   ),
				--   WaitStates AS (
				-- 	SELECT 
				-- 	  ws.runtime_stats_interval_id,
				-- 	  LEFT(qsqt.query_sql_text, ' + CAST(@TextTruncateLimit AS NVARCHAR(4)) + ') AS query_text, -- Truncate query text for the output
				-- 	  qsq.last_execution_time,
				-- 	  ws.wait_category_desc AS wait_category,
				-- 	  ws.total_query_wait_time_ms AS total_wait_time_ms,
				-- 	  ws.avg_query_wait_time_ms AS avg_wait_time_ms,
				-- 	  CASE 
				-- 		WHEN ws.avg_query_wait_time_ms > 0 THEN 
				-- 		  ws.total_query_wait_time_ms / ws.avg_query_wait_time_ms
				-- 		ELSE 
				-- 		  0 
				-- 	  END AS wait_event_count,
				-- 	  qsq.query_hash AS query_id,
				-- 	  SYSDATETIME() AS collection_timestamp,
				-- 	  ''' + @dbName + ''' AS database_name
				-- 	FROM 
				-- 	  sys.query_store_wait_stats ws
				-- 	INNER JOIN 
				-- 	  sys.query_store_plan qsp ON ws.plan_id = qsp.plan_id
				-- 	INNER JOIN 
				-- 	  sys.query_store_query AS qsq ON qsp.query_id = qsq.query_id
				-- 	INNER JOIN 
				-- 	  sys.query_store_query_text AS qsqt ON qsqt.query_text_id = qsq.query_text_id
				-- 	INNER JOIN 
				-- 	  LatestInterval li ON qsqt.query_sql_text = li.query_sql_text 
				-- 			  AND ws.runtime_stats_interval_id = li.max_runtime_stats_interval_id
				-- 	WHERE 
				-- 	  qsqt.query_sql_text NOT LIKE ''%%WITH%%''
				-- 	  AND qsqt.query_sql_text NOT LIKE ''%%sys.%%''
				-- 	  AND qsqt.query_sql_text NOT LIKE ''%%INFORMATION_SCHEMA%%''
				--   )
				--   SELECT
				-- 	query_id,
				-- 	database_name, 
				-- 	query_text,
				-- 	wait_category,
				-- 	total_wait_time_ms,
				-- 	avg_wait_time_ms,
				-- 	wait_event_count,
				-- 	last_execution_time,
				-- 	collection_timestamp
				--   FROM
				-- 	WaitStates;';
				--   
				--   INSERT INTO @resultTable
				-- 	EXEC sp_executesql @sql;
				-- 
				--   FETCH NEXT FROM db_cursor INTO @dbName;
				-- END
				-- CLOSE db_cursor;
				-- DEALLOCATE db_cursor;
				-- SELECT TOP (@TopN) * FROM @resultTable 
				-- ORDER BY total_wait_time_ms DESC;

				-- Optimized query for current waiting sessions with proper filtering and sorting
				SELECT TOP 1000
					r.session_id,
					DB_NAME(r.database_id) AS database_name,
					LEFT(st.text, 4096) AS query_text, -- 4096-character limit applied here
					r.wait_type as wait_category,
					r.wait_time AS total_wait_time_ms,
					r.start_time AS request_start_time,
					SYSDATETIME() AS collection_timestamp
				FROM
					sys.dm_exec_requests AS r
				CROSS APPLY
					sys.dm_exec_sql_text(r.sql_handle) AS st
				WHERE
					r.session_id > 50          -- Ignore system sessions
					AND r.wait_time > 0        -- Ignore if wait time is 0
					AND r.database_id > 4      -- Ignore system databases (master, model, msdb, tempdb)
					AND r.wait_type IS NOT NULL -- Only show sessions currently waiting
				`,
		Type: "waitAnalysis",
	},
	{
		EventName: "MSSQLBlockingSessionQueries",
		Query: `DECLARE @Limit INT = %d; -- Define the limit for the number of rows returned
				DECLARE @TextTruncateLimit INT = %d; -- Define the truncate limit for the query text
				WITH blocking_info AS (
					SELECT
						req.blocking_session_id AS blocking_spid,
						req.session_id AS blocked_spid,
						req.wait_type AS wait_type,
						req.wait_time / 1000.0 AS wait_time_in_seconds,
						req.start_time AS start_time,
						sess.status AS status,
						req.command AS command_type,
						req.database_id AS database_id,
						req.sql_handle AS blocked_sql_handle,
						blocking_req.sql_handle AS blocking_sql_handle,
						blocking_req.start_time AS blocking_start_time
					FROM
						sys.dm_exec_requests AS req
					LEFT JOIN sys.dm_exec_requests AS blocking_req ON blocking_req.session_id = req.blocking_session_id
					LEFT JOIN sys.dm_exec_sessions AS sess ON sess.session_id = req.session_id
					WHERE
						req.blocking_session_id != 0
				)
				SELECT TOP (@Limit)
					blocking_info.blocking_spid,
					blocking_sessions.status AS blocking_status,
					blocking_info.blocked_spid,
					blocked_sessions.status AS blocked_status,
					blocking_info.wait_type,
					blocking_info.wait_time_in_seconds,
					blocking_info.command_type,
					blocking_info.start_time AS blocked_query_start_time,
					DB_NAME(blocking_info.database_id) AS database_name,
					CASE
						WHEN blocking_sql.text IS NULL THEN LEFT(input_buffer.event_info, @TextTruncateLimit)
						ELSE LEFT(blocking_sql.text, @TextTruncateLimit)
					END AS blocking_query_text,
					LEFT(blocked_sql.text, @TextTruncateLimit) AS blocked_query_text -- Truncate blocked query text
				FROM
					blocking_info
				JOIN sys.dm_exec_sessions AS blocking_sessions ON blocking_sessions.session_id = blocking_info.blocking_spid
				JOIN sys.dm_exec_sessions AS blocked_sessions ON blocked_sessions.session_id = blocking_info.blocked_spid
				OUTER APPLY sys.dm_exec_sql_text(blocking_info.blocking_sql_handle) AS blocking_sql
				OUTER APPLY sys.dm_exec_sql_text(blocking_info.blocked_sql_handle) AS blocked_sql
				OUTER APPLY sys.dm_exec_input_buffer(blocking_info.blocking_spid, NULL) AS input_buffer
				JOIN sys.databases AS db ON db.database_id = blocking_info.database_id
				ORDER BY
    				blocking_info.start_time;`,
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
	QueryResponseTimeThresholdDefault = 1

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
