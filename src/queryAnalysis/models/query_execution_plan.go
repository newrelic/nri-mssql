package models

type QueryExecutionPlan struct {
	SQLText                string  `json:"sql_text" db:"sql_text"`
	QueryPlanText          string  `json:"query_plan_text" db:"query_plan_text"`
	NodeId                 int64   `json:"node_id" db:"NodeId"`
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
	ExecutionCount         int64   `json:"execution_count" db:"execution_count"`
}
