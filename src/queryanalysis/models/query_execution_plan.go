package models

type ExecutionPlanResult struct {
	SQLText                *string      `db:"sql_text" metric_name:"sql_text" sourceType:"attribute"`
	QueryID                *HexString   `db:"query_id" metric_name:"query_id" sourceType:"attribute"`
	QueryPlanID            *HexString   `db:"query_plan_id" metric_name:"query_plan_id" sourceType:"attribute"`
	NodeID                 *int         `db:"NodeId" metric_name:"NodeId" sourceType:"gauge"`
	PhysicalOp             *string      `db:"PhysicalOp" metric_name:"PhysicalOp" sourceType:"attribute"`
	LogicalOp              *string      `db:"LogicalOp" metric_name:"LogicalOp" sourceType:"attribute"`
	EstimateRows           *float64     `db:"EstimateRows" metric_name:"EstimateRows" sourceType:"gauge"`
	EstimateIO             *float64     `db:"EstimateIO" metric_name:"EstimateIO" sourceType:"gauge"`
	EstimateCPU            *float64     `db:"EstimateCPU" metric_name:"EstimateCPU" sourceType:"gauge"`
	AvgRowSize             *float64     `db:"AvgRowSize" metric_name:"AvgRowSize" sourceType:"gauge"`
	TotalSubtreeCost       *float64     `db:"TotalSubtreeCost" metric_name:"TotalSubtreeCost" sourceType:"gauge"`
	EstimatedOperatorCost  *float64     `db:"EstimatedOperatorCost" metric_name:"EstimatedOperatorCost" sourceType:"gauge"`
	EstimatedExecutionMode *string      `db:"EstimatedExecutionMode" metric_name:"EstimatedExecutionMode" sourceType:"attribute"`
	GrantedMemoryKb        *int         `db:"GrantedMemoryKb" metric_name:"GrantedMemoryKb" sourceType:"gauge"`
	SpillOccurred          *bool        `db:"SpillOccurred" metric_name:"SpillOccurred" sourceType:"attribute"`
	NoJoinPredicate        *bool        `db:"NoJoinPredicate" metric_name:"NoJoinPredicate" sourceType:"attribute"`
	TotalWorkerTime        *int64       `db:"total_worker_time" metric_name:"total_worker_time" sourceType:"gauge"`
	TotalElapsedTime       *int64       `db:"total_elapsed_time" metric_name:"total_elapsed_time" sourceType:"gauge"`
	TotalLogicalReads      *int64       `db:"total_logical_reads" metric_name:"total_logical_reads" sourceType:"gauge"`
	TotalLogicalWrites     *int64       `db:"total_logical_writes" metric_name:"total_logical_writes" sourceType:"gauge"`
	ExecutionCount         *int64       `db:"execution_count" metric_name:"execution_count" sourceType:"gauge"`
	PlanHandle             *VarBinary64 `db:"plan_handle" metric_name:"plan_handle" sourceType:"attribute"`
	AvgElapsedTimeMs       *float64     `db:"avg_elapsed_time_ms" metric_name:"avg_elapsed_time_ms" sourceType:"gauge"`
}
