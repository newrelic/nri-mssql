package config

// ExecutionPlanQuery holds the SQL query for fetching execution plans.
const ExecutionPlanQueryTemplate = `WITH XMLNAMESPACES (DEFAULT 'http://schemas.microsoft.com/sqlserver/2004/07/showplan')
SELECT 
    st.text AS sql_text, 
    qp.query_plan AS query_plan_xml, 
	qs.query_hash AS query_id,
    COALESCE(n.value('(@NodeId)[1]', 'INT'), 0) AS NodeId, 
    COALESCE(n.value('(@PhysicalOp)[1]', 'VARCHAR(50)'), 'N/A') AS PhysicalOp, 
    COALESCE(n.value('(@LogicalOp)[1]', 'VARCHAR(50)'), 'N/A') AS LogicalOp, 
    COALESCE(n.value('(@EstimateRows)[1]', 'FLOAT'), 0.0) AS EstimateRows, 
    COALESCE(n.value('(@EstimateIO)[1]', 'FLOAT'), 0.0) AS EstimateIO, 
    COALESCE(n.value('(@EstimateCPU)[1]', 'FLOAT'), 0.0) AS EstimateCPU, 
    COALESCE(n.value('(@AvgRowSize)[1]', 'FLOAT'), 0.0) AS AvgRowSize, 
    COALESCE(n.value('(@TotalSubtreeCost)[1]', 'FLOAT'), 0.0) AS TotalSubtreeCost, 
    COALESCE(n.value('(@EstimatedOperatorCost)[1]', 'FLOAT'), 0.0) AS EstimatedOperatorCost, 
    COALESCE(n.value('(@EstimatedExecutionMode)[1]', 'VARCHAR(50)'), 'N/A') AS EstimatedExecutionMode, 
    COALESCE(n.value('(MemoryGrantInfo/@GrantedMemoryKb)[1]', 'INT'), 0) AS GrantedMemoryKb, 
    COALESCE(n.value('(Warnings/@SpillOccurred)[1]', 'BIT'), 0) AS SpillOccurred, 
    COALESCE(n.value('(Warnings/@NoJoinPredicate)[1]', 'BIT'), 0) AS NoJoinPredicate, 
    qs.total_worker_time, 
    qs.total_elapsed_time, 
    qs.total_logical_reads, 
    qs.total_logical_writes, 
    qs.execution_count
FROM sys.dm_exec_query_stats AS qs
CROSS APPLY sys.dm_exec_sql_text(qs.sql_handle) AS st
CROSS APPLY sys.dm_exec_query_plan(qs.plan_handle) AS qp
CROSS APPLY qp.query_plan.nodes('//RelOp') AS RelOps(n)
WHERE qs.query_hash = %s
ORDER BY qs.total_worker_time DESC;`
