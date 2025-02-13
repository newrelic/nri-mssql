package models

type ExecutionPlanResult struct {
	SQLText                *string      `db:"sql_text" metric_name:"sql_text" source_type:"attribute"`
	QueryID                *HexString   `db:"query_id" metric_name:"query_id" source_type:"attribute"`
	QueryPlanID            *HexString   `db:"query_plan_id" metric_name:"query_plan_id" source_type:"attribute"`
	NodeID                 *int         `db:"NodeId" metric_name:"NodeId" source_type:"gauge"`
	PhysicalOp             *string      `db:"PhysicalOp" metric_name:"PhysicalOp" source_type:"attribute"`
	LogicalOp              *string      `db:"LogicalOp" metric_name:"LogicalOp" source_type:"attribute"`
	EstimateRows           *float64     `db:"EstimateRows" metric_name:"EstimateRows" source_type:"gauge"`
	EstimateIO             *float64     `db:"EstimateIO" metric_name:"EstimateIO" source_type:"gauge"`
	EstimateCPU            *float64     `db:"EstimateCPU" metric_name:"EstimateCPU" source_type:"gauge"`
	AvgRowSize             *float64     `db:"AvgRowSize" metric_name:"AvgRowSize" source_type:"gauge"`
	TotalSubtreeCost       *float64     `db:"TotalSubtreeCost" metric_name:"TotalSubtreeCost" source_type:"gauge"`
	EstimatedOperatorCost  *float64     `db:"EstimatedOperatorCost" metric_name:"EstimatedOperatorCost" source_type:"gauge"`
	EstimatedExecutionMode *string      `db:"EstimatedExecutionMode" metric_name:"EstimatedExecutionMode" source_type:"attribute"`
	GrantedMemoryKb        *int         `db:"GrantedMemoryKb" metric_name:"GrantedMemoryKb" source_type:"gauge"`
	SpillOccurred          *bool        `db:"SpillOccurred" metric_name:"SpillOccurred" source_type:"attribute"`
	NoJoinPredicate        *bool        `db:"NoJoinPredicate" metric_name:"NoJoinPredicate" source_type:"attribute"`
	TotalWorkerTime        *int64       `db:"total_worker_time" metric_name:"total_worker_time" source_type:"gauge"`
	TotalElapsedTime       *int64       `db:"total_elapsed_time" metric_name:"total_elapsed_time" source_type:"gauge"`
	TotalLogicalReads      *int64       `db:"total_logical_reads" metric_name:"total_logical_reads" source_type:"gauge"`
	TotalLogicalWrites     *int64       `db:"total_logical_writes" metric_name:"total_logical_writes" source_type:"gauge"`
	ExecutionCount         *int64       `db:"execution_count" metric_name:"execution_count" source_type:"gauge"`
	PlanHandle             *VarBinary64 `db:"plan_handle" metric_name:"plan_handle" source_type:"attribute"`
	AvgElapsedTimeMs       *float64     `db:"avg_elapsed_time_ms" metric_name:"avg_elapsed_time_ms" source_type:"gauge"`
}
