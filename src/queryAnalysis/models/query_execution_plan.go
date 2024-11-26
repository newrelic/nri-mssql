package models

type QueryExecutionPlan struct {
	SQLText                string `json:"sql_text" db:"sql_text"`
	QueryPlanText          string `json:"query_plan_text" db:"query_plan_text"`
	NodeId                 string `json:"node_id" db:"NodeId"`
	PhysicalOp             string `json:"physical_op" db:"PhysicalOp"`
	LogicalOp              string `json:"logical_op" db:"LogicalOp"`
	EstimateRows           string `json:"estimate_rows" db:"EstimateRows"`
	EstimateIO             string `json:"estimate_io" db:"EstimateIO"`
	EstimateCPU            string `json:"estimate_cpu" db:"EstimateCPU"`
	AvgRowSize             string `json:"avg_row_size" db:"AvgRowSize"`
	TotalSubtreeCost       string `json:"total_subtree_cost" db:"TotalSubtreeCost"`
	EstimatedOperatorCost  string `json:"estimated_operator_cost" db:"EstimatedOperatorCost"`
	EstimatedExecutionMode string `json:"estimated_execution_mode" db:"EstimatedExecutionMode"`
	TotalWorkerTime        string `json:"total_worker_time" db:"total_worker_time"`
	TotalElapsedTime       string `json:"total_elapsed_time" db:"total_elapsed_time"`
	TotalLogicalReads      string `json:"total_logical_reads" db:"total_logical_reads"`
	TotalLogicalWrites     string `json:"total_logical_writes" db:"total_logical_writes"`
	ExecutionCount         string `json:"execution_count" db:"execution_count"`
}
