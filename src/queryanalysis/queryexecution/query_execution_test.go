package queryexecution

import (
	"errors"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	ErrQueryExecution = errors.New("query execution error")
)

func TestProcessExecutionPlans_Success(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	// Setup your specific execution plan SQL query pattern
	executionPlanQueryPattern := `(?s)DECLARE @TopN INT =.*?DECLARE @ElapsedTimeThreshold INT =.*?DECLARE @QueryIDs NVARCHAR\(1000\).*?INSERT INTO @QueryIdTable.*?SELECT.*?FROM PlanNodes ORDER BY plan_handle, NodeId;`

	// Mocking SQL response to match expected output
	mock.ExpectQuery(executionPlanQueryPattern).
		WillReturnRows(sqlmock.NewRows([]string{
			"query_id", "sql_text", "plan_handle", "query_plan_id",
			"avg_elapsed_time_ms", "execution_count", "NodeId",
			"PhysicalOp", "LogicalOp", "EstimateRows",
			"EstimateIO", "EstimateCPU", "AvgRowSize",
			"EstimatedExecutionMode", "TotalSubtreeCost",
			"EstimatedOperatorCost", "GrantedMemoryKb",
			"SpillOccurred", "NoJoinPredicate",
		}).
			AddRow(
				[]byte{0x01, 0x02}, "SELECT * FROM some_table", "some_plan_handle",
				"some_query_plan_id", 100, 10, // Replace with realistic/mock values
				1, "PhysicalOp1", "LogicalOp1", 100,
				1.0, 0.5, 4.0, "Row",
				3.0, 5.0, 200,
				false, false))

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}
	queryIDs := []models.HexString{"0x0102"}

	// Call the target function
	ProcessExecutionPlans(argList, integrationObj, sqlConn, queryIDs)

	// Ensure all expectations are met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestProcessExecutionPlans_NoQueryIDs(t *testing.T) {
	// Initialize a mock SQL connection
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	// There shouldn't be any SQL query execution when there are no query IDs
	// Hence, no `ExpectQuery` call is needed when expecting zero interactions

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}
	queryIDs := []models.HexString{} // Empty query IDs

	// Call the function, which should ideally do nothing
	ProcessExecutionPlans(argList, integrationObj, sqlConn, queryIDs)

	// Verify that no SQL expectations were set (and consequently met)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected SQL execution: %v", err)
	}
}

func TestExecuteQuery_SlowQueriesSuccess(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	query := "SELECT * FROM slow_queries WHERE condition"
	mock.ExpectQuery("SELECT \\* FROM slow_queries WHERE condition").
		WillReturnRows(sqlmock.NewRows([]string{
			"query_id", "query_text", "database_name",
		}).
			AddRow(
				[]byte{0x01, 0x02},
				"SELECT * FROM something",
				"example_db",
			))

	queryDetails := models.QueryDetailsDto{
		EventName: "SlowQueries",
		Query:     query,
		Type:      "slowQueries",
	}

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}

	results, err := ExecuteQuery(argList, queryDetails, integrationObj, sqlConn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	slowQuery, ok := results[0].(models.TopNSlowQueryDetails)
	if !ok {
		t.Fatalf("expected type models.TopNSlowQueryDetails, got %T", results[0])
	}

	expectedQueryID := models.HexString("0x0102")
	if slowQuery.QueryID == nil || *slowQuery.QueryID != expectedQueryID {
		t.Errorf("expected QueryID %v, got %v", expectedQueryID, slowQuery.QueryID)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestExecuteQuery_WaitTimeAnalysis(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	query := "SELECT * FROM wait_analysis WHERE condition"
	mock.ExpectQuery("SELECT \\* FROM wait_analysis WHERE condition").
		WillReturnRows(sqlmock.NewRows([]string{
			"query_id", "database_name", "query_text", "wait_category",
			"total_wait_time_ms", "avg_wait_time_ms", "wait_event_count",
			"last_execution_time", "collection_timestamp",
		}).
			AddRow(
				[]byte{0x01, 0x02},
				"example_db",
				"SELECT * FROM waits",
				"CPU",
				100.5,
				50.25,
				10,
				time.Now(),
				time.Now(),
			))

	queryDetails := models.QueryDetailsDto{
		EventName: "WaitTimeAnalysisQuery",
		Query:     query,
		Type:      "waitAnalysis",
	}

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}

	results, err := ExecuteQuery(argList, queryDetails, integrationObj, sqlConn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	waitTimeAnalysis, ok := results[0].(models.WaitTimeAnalysis)
	if !ok {
		t.Fatalf("expected type models.WaitTimeAnalysis, got %T", results[0])
	}

	expectedQueryID := models.HexString("0x0102")
	if waitTimeAnalysis.QueryID == nil || *waitTimeAnalysis.QueryID != expectedQueryID {
		t.Errorf("expected QueryID %v, got %v", expectedQueryID, waitTimeAnalysis.QueryID)
	}

	expectedDatabaseName := "example_db"
	if waitTimeAnalysis.DatabaseName == nil || *waitTimeAnalysis.DatabaseName != expectedDatabaseName {
		t.Errorf("expected DatabaseName %s, got %v", expectedDatabaseName, waitTimeAnalysis.DatabaseName)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestExecuteQuery_BlockingSessionsSuccess(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	query := "SELECT * FROM blocking_sessions WHERE condition"
	mock.ExpectQuery("SELECT \\* FROM blocking_sessions WHERE condition").
		WillReturnRows(sqlmock.NewRows([]string{
			"blocking_spid", "blocking_status", "blocked_spid", "blocked_status",
			"wait_type", "wait_time_in_seconds", "command_type", "database_name",
			"blocking_query_text", "blocked_query_text",
		}).
			AddRow(
				int64(101),
				"Running",
				int64(202),
				"Suspended",
				"LCK_M_U",
				3.5,
				"SELECT",
				"example_db",
				"SELECT * FROM source",
				"INSERT INTO destination",
			))

	queryDetails := models.QueryDetailsDto{
		EventName: "BlockingSessionsQuery",
		Query:     query,
		Type:      "blockingSessions",
	}

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}

	results, err := ExecuteQuery(argList, queryDetails, integrationObj, sqlConn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	validateBlockingSession(t, results[0])

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Continue with other test functions and validation functions as needed...

// Helper function for validating blocking session results
func validateBlockingSession(t *testing.T, result interface{}) {
	blockingSession, ok := result.(models.BlockingSessionQueryDetails)
	if !ok {
		t.Fatalf("expected type models.BlockingSessionQueryDetails, got %T", result)
	}

	checkInt64Field(t, "BlockingSPID", blockingSession.BlockingSPID, 101)
	checkInt64Field(t, "BlockedSPID", blockingSession.BlockedSPID, 202)
	checkStringField(t, "DatabaseName", blockingSession.DatabaseName, "example_db")
	checkStringField(t, "BlockingQueryText", blockingSession.BlockingQueryText, "SELECT * FROM source")
}

// Helper functions to check fields
func checkInt64Field(t *testing.T, name string, field *int64, expected int64) {
	if field == nil || *field != expected {
		t.Errorf("expected %s %v, got %v", name, expected, field)
	}
}

func checkStringField(t *testing.T, name string, field *string, expected string) {
	if field == nil || *field != expected {
		t.Errorf("expected %s %v, got %v", name, expected, field)
	}
}
