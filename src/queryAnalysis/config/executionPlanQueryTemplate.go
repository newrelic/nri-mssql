package config

// ExecutionPlanQuery holds the SQL query for fetching execution plans.
const ExecutionPlanQueryTemplate = `
DECLARE @TopN INT = %d; 
DECLARE @ElapsedTimeThreshold INT = %d;  -- Define the elapsed time threshold in milliseconds
DECLARE @QueryID NVARCHAR(50) = %s;      -- Change the query ID to a string
DECLARE @IntervalSeconds INT = %d;       -- Define the interval in seconds (e.g., 3600 for the last hour)

WITH XMLNAMESPACES (DEFAULT 'http://schemas.microsoft.com/sqlserver/2004/07/showplan'),
TopPlans AS (
    SELECT TOP (@TopN)
        qs.plan_handle,
		qp.query_plan AS query_plan_xml,
        qs.query_hash as query_id,
		qs.query_plan_hash AS query_plan_id,
        st.text AS sql_text,
		qs.execution_count as execution_count,
        (qs.total_elapsed_time / qs.execution_count) / 1000 AS avg_elapsed_time_ms,
        qp.query_plan
    FROM sys.dm_exec_query_stats AS qs
    CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS st
    CROSS APPLY sys.dm_exec_query_plan(qs.plan_handle) AS qp
    WHERE CONVERT(NVARCHAR(50), qs.query_hash) = @QueryID 
    AND qs.last_execution_time BETWEEN DATEADD(SECOND, -@IntervalSeconds, GETDATE()) AND GETDATE() 
    AND (qs.total_elapsed_time / qs.execution_count) / 1000 > @ElapsedTimeThreshold
    ORDER BY avg_elapsed_time_ms DESC
),
PlanNodes AS (
    SELECT
        tp.query_id,
        tp.sql_text,
        tp.plan_handle,
		tp.query_plan_xml,
		tp.query_plan_id,
        tp.avg_elapsed_time_ms,
		tp.execution_count,
        n.value('(@NodeId)[1]', 'INT') AS NodeId,
        n.value('(@PhysicalOp)[1]', 'VARCHAR(50)') AS PhysicalOp,
        n.value('(@LogicalOp)[1]', 'VARCHAR(50)') AS LogicalOp,
        n.value('(@EstimateRows)[1]', 'FLOAT') AS EstimateRows,
        n.value('(@EstimateIO)[1]', 'FLOAT') AS EstimateIO,
        n.value('(@EstimateCPU)[1]', 'FLOAT') AS EstimateCPU,
        n.value('(@AvgRowSize)[1]', 'FLOAT') AS AvgRowSize,
        n.value('(@EstimatedExecutionMode)[1]', 'VARCHAR(50)') AS EstimatedExecutionMode,
        n.value('(@EstimatedTotalSubtreeCost)[1]', 'FLOAT') AS TotalSubtreeCost,
        n.value('(@EstimatedOperatorCost)[1]', 'FLOAT') AS EstimatedOperatorCost,
        n.value('(MemoryGrantInfo/@GrantedMemoryKb)[1]', 'INT') AS GrantedMemoryKb,
        n.value('(Warnings/Warning/@SpillOccurred)[1]', 'BIT') AS SpillOccurred,
        n.value('(Warnings/Warning/@NoJoinPredicate)[1]', 'BIT') AS NoJoinPredicate
    FROM TopPlans AS tp
    CROSS APPLY tp.query_plan.nodes('//RelOp') AS RelOps(n)
)
SELECT *
FROM PlanNodes
ORDER BY plan_handle, NodeId;
`
