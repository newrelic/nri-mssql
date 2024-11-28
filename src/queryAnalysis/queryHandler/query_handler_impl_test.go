package queryhandler

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func TestExecuteQuery_Success(t *testing.T) {
	// Create sqlmock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"col1"}))

	qh := &QueryHandlerImpl{}
	queryConfig := models.QueryDetailsDto{Query: "SELECT * FROM test"}

	rows, err := qh.ExecuteQuery(sqlxDB, queryConfig)

	require.NoError(t, err)
	assert.NotNil(t, rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteQuery_Error(t *testing.T) {
	// Create sqlmock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectQuery("SELECT .*").WillReturnError(errors.New("query failed"))

	qh := &QueryHandlerImpl{}
	queryConfig := models.QueryDetailsDto{Query: "SELECT * FROM test"}

	rows, err := qh.ExecuteQuery(sqlxDB, queryConfig)

	assert.Error(t, err)
	assert.Nil(t, rows)
	assert.Contains(t, err.Error(), "failed to execute query")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindQueryResults_BlockingSession_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"blocking_spid", "blocking_status", "blocked_spid", "blocked_status", "wait_type", "wait_time_in_seconds", "command_type", "database_name", "blocking_query_text", "blocked_query_text",
	}).
		AddRow(1, "block1", 2, "blocked1", "type1", 10.5, "command1", "db1", "blocking_query1", "blocked_query1").
		AddRow(3, "block2", 4, "blocked2", "type2", 20.5, "command2", "db2", "blocking_query2", "blocked_query2")

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	sqlxRows, err := sqlxDB.Queryx("SELECT * FROM test")
	require.NoError(t, err)

	qh := &QueryHandlerImpl{}

	var result []models.BlockingSessionQueryDetails
	err = qh.BindQueryResults(sqlxRows, &result)
	require.NoError(t, err)

	expected := []models.BlockingSessionQueryDetails{
		{
			BlockingSPID:      int64Ptr(1),
			BlockingStatus:    stringPtr("block1"),
			BlockedSPID:       int64Ptr(2),
			BlockedStatus:     stringPtr("blocked1"),
			WaitType:          stringPtr("type1"),
			WaitTimeInSeconds: float64Ptr(10.5),
			CommandType:       stringPtr("command1"),
			DatabaseName:      stringPtr("db1"),
			BlockingQueryText: stringPtr("blocking_query1"),
			BlockedQueryText:  stringPtr("blocked_query1"),
		},
		{
			BlockingSPID:      int64Ptr(3),
			BlockingStatus:    stringPtr("block2"),
			BlockedSPID:       int64Ptr(4),
			BlockedStatus:     stringPtr("blocked2"),
			WaitType:          stringPtr("type2"),
			WaitTimeInSeconds: float64Ptr(20.5),
			CommandType:       stringPtr("command2"),
			DatabaseName:      stringPtr("db2"),
			BlockingQueryText: stringPtr("blocking_query2"),
			BlockedQueryText:  stringPtr("blocked_query2"),
		},
	}
	assert.Equal(t, expected, result)

	// Validate data types of columns
	assert.IsType(t, int64Ptr(1), result[0].BlockingSPID)
	assert.IsType(t, stringPtr("block1"), result[0].BlockingStatus)
	assert.IsType(t, int64Ptr(2), result[0].BlockedSPID)
	assert.IsType(t, stringPtr("blocked1"), result[0].BlockedStatus)
	assert.IsType(t, stringPtr("type1"), result[0].WaitType)
	assert.IsType(t, float64Ptr(10.5), result[0].WaitTimeInSeconds)
	assert.IsType(t, stringPtr("command1"), result[0].CommandType)
	assert.IsType(t, stringPtr("db1"), result[0].DatabaseName)
	assert.IsType(t, stringPtr("blocking_query1"), result[0].BlockingQueryText)
	assert.IsType(t, stringPtr("blocked_query1"), result[0].BlockedQueryText)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindQueryResults_QueryExecutionPlan_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"sql_text", "query_plan_text", "NodeId", "PhysicalOp", "LogicalOp", "EstimateRows", "EstimateIO", "EstimateCPU",
		"AvgRowSize", "TotalSubtreeCost", "EstimatedOperatorCost", "EstimatedExecutionMode", "total_worker_time", "total_elapsed_time", "total_logical_reads", "total_logical_writes", "execution_count",
	}).
		AddRow("text1", "plan1", 1, "Physical1", "Logical1", 10.5, 20.5, 30.5, 40.5, 50.5, 60.5, "Mode1", 70, 80, 90, 100, 110).
		AddRow("text2", "plan2", 2, "Physical2", "Logical2", 11.5, 21.5, 31.5, 41.5, 51.5, 61.5, "Mode2", 71, 81, 91, 101, 111)

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	sqlxRows, err := sqlxDB.Queryx("SELECT * FROM test")
	require.NoError(t, err)

	qh := &QueryHandlerImpl{}

	var result []models.ExecutionPlanResult
	err = qh.BindQueryResults(sqlxRows, &result)
	require.NoError(t, err)

	expected := []models.ExecutionPlanResult{
		{
			SQLText:                "text1",
			QueryPlanText:          "plan1",
			NodeId:                 1,
			PhysicalOp:             "Physical1",
			LogicalOp:              "Logical1",
			EstimateRows:           10.5,
			EstimateIO:             20.5,
			EstimateCPU:            30.5,
			AvgRowSize:             40.5,
			TotalSubtreeCost:       50.5,
			EstimatedOperatorCost:  60.5,
			EstimatedExecutionMode: "Mode1",
			TotalWorkerTime:        70,
			TotalElapsedTime:       80,
			TotalLogicalReads:      90,
			TotalLogicalWrites:     100,
			ExecutionCount:         110,
		},
		{
			SQLText:                "text2",
			QueryPlanText:          "plan2",
			NodeId:                 2,
			PhysicalOp:             "Physical2",
			LogicalOp:              "Logical2",
			EstimateRows:           11.5,
			EstimateIO:             21.5,
			EstimateCPU:            31.5,
			AvgRowSize:             41.5,
			TotalSubtreeCost:       51.5,
			EstimatedOperatorCost:  61.5,
			EstimatedExecutionMode: "Mode2",
			TotalWorkerTime:        71,
			TotalElapsedTime:       81,
			TotalLogicalReads:      91,
			TotalLogicalWrites:     101,
			ExecutionCount:         111,
		},
	}
	assert.Equal(t, expected, result)

	// Validate data types of columns
	assert.IsType(t, "text1", result[0].SQLText)
	assert.IsType(t, "plan1", result[0].QueryPlanText)
	assert.IsType(t, int64(1), result[0].NodeId)
	assert.IsType(t, "Physical1", result[0].PhysicalOp)
	assert.IsType(t, "Logical1", result[0].LogicalOp)
	assert.IsType(t, 10.5, result[0].EstimateRows)
	assert.IsType(t, 20.5, result[0].EstimateIO)
	assert.IsType(t, 30.5, result[0].EstimateCPU)
	assert.IsType(t, 40.5, result[0].AvgRowSize)
	assert.IsType(t, 50.5, result[0].TotalSubtreeCost)
	assert.IsType(t, 60.5, result[0].EstimatedOperatorCost)
	assert.IsType(t, "Mode1", result[0].EstimatedExecutionMode)
	assert.IsType(t, int64(70), result[0].TotalWorkerTime)
	assert.IsType(t, int64(80), result[0].TotalElapsedTime)
	assert.IsType(t, int64(90), result[0].TotalLogicalReads)
	assert.IsType(t, int64(100), result[0].TotalLogicalWrites)
	assert.IsType(t, int64(110), result[0].ExecutionCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindQueryResults_TopNSlowQueryDetails_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"query_id", "query_text", "database_name", "schema_name", "last_execution_timestamp", "execution_count", "avg_cpu_time_ms", "avg_elapsed_time_ms", "avg_disk_reads", "avg_disk_writes", "statement_type", "collection_timestamp",
	}).
		AddRow("id1", "text1", "db1", "schema1", "timestamp1", 10, 20.5, 30.5, 40.5, 50.5, "type1", "collection1").
		AddRow("id2", "text2", "db2", "schema2", "timestamp2", 11, 21.5, 31.5, 41.5, 51.5, "type2", "collection2")

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	sqlxRows, err := sqlxDB.Queryx("SELECT * FROM test")
	require.NoError(t, err)

	qh := &QueryHandlerImpl{}

	var result []models.TopNSlowQueryDetails
	err = qh.BindQueryResults(sqlxRows, &result)
	require.NoError(t, err)

	expected := []models.TopNSlowQueryDetails{
		{
			QueryID:                stringPtr("id1"),
			QueryText:              stringPtr("text1"),
			DatabaseName:           stringPtr("db1"),
			SchemaName:             stringPtr("schema1"),
			LastExecutionTimestamp: stringPtr("timestamp1"),
			ExecutionCount:         int64Ptr(10),
			AvgCPUTimeMS:           float64Ptr(20.5),
			AvgElapsedTimeMS:       float64Ptr(30.5),
			AvgDiskReads:           float64Ptr(40.5),
			AvgDiskWrites:          float64Ptr(50.5),
			StatementType:          stringPtr("type1"),
			CollectionTimestamp:    stringPtr("collection1"),
		},
		{
			QueryID:                stringPtr("id2"),
			QueryText:              stringPtr("text2"),
			DatabaseName:           stringPtr("db2"),
			SchemaName:             stringPtr("schema2"),
			LastExecutionTimestamp: stringPtr("timestamp2"),
			ExecutionCount:         int64Ptr(11),
			AvgCPUTimeMS:           float64Ptr(21.5),
			AvgElapsedTimeMS:       float64Ptr(31.5),
			AvgDiskReads:           float64Ptr(41.5),
			AvgDiskWrites:          float64Ptr(51.5),
			StatementType:          stringPtr("type2"),
			CollectionTimestamp:    stringPtr("collection2"),
		},
	}
	assert.Equal(t, expected, result)

	// Validate data types of columns
	assert.IsType(t, stringPtr("id1"), result[0].QueryID)
	assert.IsType(t, stringPtr("text1"), result[0].QueryText)
	assert.IsType(t, stringPtr("db1"), result[0].DatabaseName)
	assert.IsType(t, stringPtr("schema1"), result[0].SchemaName)
	assert.IsType(t, stringPtr("timestamp1"), result[0].LastExecutionTimestamp)
	assert.IsType(t, int64Ptr(10), result[0].ExecutionCount)
	assert.IsType(t, float64Ptr(20.5), result[0].AvgCPUTimeMS)
	assert.IsType(t, float64Ptr(30.5), result[0].AvgElapsedTimeMS)
	assert.IsType(t, float64Ptr(40.5), result[0].AvgDiskReads)
	assert.IsType(t, float64Ptr(50.5), result[0].AvgDiskWrites)
	assert.IsType(t, stringPtr("type1"), result[0].StatementType)
	assert.IsType(t, stringPtr("collection1"), result[0].CollectionTimestamp)

	assert.NoError(t, mock.ExpectationsWereMet())
}
