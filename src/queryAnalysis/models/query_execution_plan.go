package models

import (
	"encoding/hex"
	"fmt"
)

type HexString string

// Scan implements the sql.Scanner interface for HexString
func (h *HexString) Scan(value interface{}) error {
	bytes, ok := value.([]uint8)
	if !ok {
		return fmt.Errorf("HexString: expected []uint8, got %T", value)
	}

	hexString := "0x" + hex.EncodeToString(bytes)
	*h = HexString(hexString)
	return nil
}

type ExecutionPlanResult struct {
	SQLText                *string    `db:"sql_text"`
	QueryPlanXML           *string    `db:"query_plan_xml"`
	QueryID                *HexString `db:"query_id"`
	QueryPlanID            *HexString `db:"query_plan_id"`
	NodeId                 *int       `db:"NodeId"`
	PhysicalOp             *string    `db:"PhysicalOp"`
	LogicalOp              *string    `db:"LogicalOp"`
	EstimateRows           *float64   `db:"EstimateRows"`
	EstimateIO             *float64   `db:"EstimateIO"`
	EstimateCPU            *float64   `db:"EstimateCPU"`
	AvgRowSize             *float64   `db:"AvgRowSize"`
	TotalSubtreeCost       *float64   `db:"TotalSubtreeCost"`
	EstimatedOperatorCost  *float64   `db:"EstimatedOperatorCost"`
	EstimatedExecutionMode *string    `db:"EstimatedExecutionMode"`
	GrantedMemoryKb        *int       `db:"GrantedMemoryKb"`
	SpillOccurred          *bool      `db:"SpillOccurred"`
	NoJoinPredicate        *bool      `db:"NoJoinPredicate"`
	TotalWorkerTime        *int64     `db:"total_worker_time"`
	TotalElapsedTime       *int64     `db:"total_elapsed_time"`
	TotalLogicalReads      *int64     `db:"total_logical_reads"`
	TotalLogicalWrites     *int64     `db:"total_logical_writes"`
	ExecutionCount         *int64     `db:"execution_count"`
}
