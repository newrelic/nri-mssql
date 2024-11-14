package queryAnalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

type ExecutionPlan struct {
	SQLText                string  `json:"sql_text" db:"sql_text"`
	QueryPlanText          string  `json:"query_plan_text" db:"query_plan_text"`
	NodeId                 int     `json:"node_id" db:"NodeId"`
	PhysicalOp             string  `json:"physical_op" db:"PhysicalOp"`
	LogicalOp              string  `json:"logical_op" db:"LogicalOp"`
	EstimateRows           float64 `json:"estimate_rows" db:"EstimateRows"`
	EstimateIO             float64 `json:"estimate_io" db:"EstimateIO"`
	EstimateCPU            float64 `json:"estimate_cpu" db:"EstimateCPU"`
	AvgRowSize             float64 `json:"avg_row_size" db:"AvgRowSize"`
	TotalSubtreeCost       float64 `json:"total_subtree_cost" db:"TotalSubtreeCost"`
	EstimatedOperatorCost  float64 `json:"estimated_operator_cost" db:"EstimatedOperatorCost"`
	EstimatedExecutionMode string  `json:"estimated_execution_mode" db:"EstimatedExecutionMode"`
	TotalWorkerTime        int64   `json:"total_worker_time" db:"total_worker_time"`
	TotalElapsedTime       int64   `json:"total_elapsed_time" db:"total_elapsed_time"`
	TotalLogicalReads      int64   `json:"total_logical_reads" db:"total_logical_reads"`
	TotalLogicalWrites     int64   `json:"total_logical_writes" db:"total_logical_writes"`
	ExecutionCount         int     `json:"execution_count" db:"execution_count"`
}

// AnalyzeExecutionPlans examines the execution plans of queries
func AnalyzeExecutionPlans(entity *integration.Entity, sqlConnection *connection.SQLConnection, arguments args.ArgumentList) {
	log.Info("Analyzing execution plans of queries...")

	query := `
        WITH XMLNAMESPACES (DEFAULT 'http://schemas.microsoft.com/sqlserver/2004/07/showplan')
        SELECT TOP 10
            st.text AS sql_text,
            CAST(qp.query_plan AS NVARCHAR(MAX)) AS query_plan_text,
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
            COALESCE(qs.total_worker_time, 0) AS total_worker_time,
            COALESCE(qs.total_elapsed_time, 0) AS total_elapsed_time,
            COALESCE(qs.total_logical_reads, 0) AS total_logical_reads,
            COALESCE(qs.total_logical_writes, 0) AS total_logical_writes,
            COALESCE(qs.execution_count, 0) AS execution_count
        FROM
            sys.dm_exec_query_stats AS qs
        CROSS APPLY
            sys.dm_exec_sql_text(qs.sql_handle) AS st
        CROSS APPLY
            sys.dm_exec_query_plan(qs.plan_handle) AS qp
        CROSS APPLY
            qp.query_plan.nodes('//RelOp') AS RelOps(n)
        WHERE
            st.text IS NOT NULL AND LTRIM(RTRIM(st.text)) <> ''
        ORDER BY
            qs.total_worker_time DESC;
    `

	// Slice to hold query results.
	executionPlans := make([]ExecutionPlan, 0)

	// Execute the query and store the results in the executionPlans slice.
	rows, err := sqlConnection.Queryx(query)
	if err != nil {
		log.Error("Could not execute query for execution plan: %s", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var plan ExecutionPlan
		if err := rows.StructScan(&plan); err != nil {
			log.Error("Could not scan row: %s", err.Error())
			continue
		}
		executionPlans = append(executionPlans, plan)
	}

	log.Info("Number of records retrieved: %d", len(executionPlans))

	// Log and report each result from the query.
	for _, plan := range executionPlans {
		metricSet := entity.NewMetricSet("MssqlExecutionPlans",
			attribute.Attribute{Key: "sqlText", Value: plan.SQLText},
			attribute.Attribute{Key: "queryPlanText", Value: plan.QueryPlanText},
		)

		// Add all the fields to the metric set.
		metricSet.SetMetric("nodeId", plan.NodeId, metric.GAUGE)
		metricSet.SetMetric("physicalOp", plan.PhysicalOp, metric.GAUGE)
		metricSet.SetMetric("logicalOp", plan.LogicalOp, metric.GAUGE)
		metricSet.SetMetric("estimateRows", plan.EstimateRows, metric.GAUGE)
		metricSet.SetMetric("estimateIO", plan.EstimateIO, metric.GAUGE)
		metricSet.SetMetric("estimateCPU", plan.EstimateCPU, metric.GAUGE)
		metricSet.SetMetric("avgRowSize", plan.AvgRowSize, metric.GAUGE)
		metricSet.SetMetric("totalSubtreeCost", plan.TotalSubtreeCost, metric.GAUGE)
		metricSet.SetMetric("estimatedOperatorCost", plan.EstimatedOperatorCost, metric.GAUGE)
		metricSet.SetMetric("estimatedExecutionMode", plan.EstimatedExecutionMode, metric.GAUGE)
		metricSet.SetMetric("totalWorkerTime", plan.TotalWorkerTime, metric.GAUGE)
		metricSet.SetMetric("totalElapsedTime", plan.TotalElapsedTime, metric.GAUGE)
		metricSet.SetMetric("totalLogicalReads", plan.TotalLogicalReads, metric.GAUGE)
		metricSet.SetMetric("totalLogicalWrites", plan.TotalLogicalWrites, metric.GAUGE)
		metricSet.SetMetric("executionCount", plan.ExecutionCount, metric.GAUGE)

		log.Info("Metrics set for execution plan: SQLText: %s", plan.SQLText)
	}

	log.Info("Completed processing all execution plan entries.")
}
