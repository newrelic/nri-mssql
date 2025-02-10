package utils

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/metrics"
	"github.com/newrelic/nri-mssql/src/queryanalysis/config"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	ErrQueryExecution = errors.New("query execution error")
)

func TestGenerateAndIngestExecutionPlan_Success(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	// Match using parts of the SQL query
	executionPlanQueryPattern := `(?s)DECLARE @TopN INT =.*?DECLARE @ElapsedTimeThreshold INT =.*?DECLARE @QueryIDs NVARCHAR\(1000\).*?INSERT INTO @QueryIdTable.*?SELECT.*?FROM PlanNodes ORDER BY plan_handle, NodeId;`

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
				[]byte{0x01, 0x02}, "SELECT * FROM table", []byte{0x01, 0x02},
				"some_query_plan_id", 100, 10,
				1, "PhysicalOp1", "LogicalOp1", 100,
				1.0, 0.5, 4.0, "Row",
				3.0, 5.0, 200,
				false, false))

	// Prepare your integration object and arguments list
	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}
	queryIDString := "0102"

	// Call your actual function
	GenerateAndIngestExecutionPlan(argList, integrationObj, sqlConn, queryIDString)

	// Verifying all expectations met ensures your mock was correct.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGenerateAndIngestExecutionPlan_QueryError(t *testing.T) {
	sqlConn, mock := connection.CreateMockSQL(t)
	defer sqlConn.Connection.Close()

	// Simulate an error during the execution of the execution plan query
	mock.ExpectQuery("DECLARE @TopN INT = (.+?);").WillReturnError(ErrQueryExecution)

	integrationObj := &integration.Integration{}
	argList := args.ArgumentList{}
	queryIDString := "0102"

	// Call the function
	GenerateAndIngestExecutionPlan(argList, integrationObj, sqlConn, queryIDString)

	// Ensure all expectations are met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

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

func TestLoadQueries_SlowQueries(t *testing.T) {
	configQueries := config.Queries
	var arguments args.ArgumentList

	slowQueriesIndex := -1
	for i, query := range configQueries {
		if query.Type == "slowQueries" {
			slowQueriesIndex = i
			break
		}
	}

	// Ensure the correct query was found
	if slowQueriesIndex == -1 {
		t.Fatalf("could not find 'slowQueries' in the list of queries")
	}

	queries, err := LoadQueries(config.Queries, arguments)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	configQueries[slowQueriesIndex].Query = fmt.Sprintf(configQueries[slowQueriesIndex].Query,
		arguments.QueryMonitoringFetchInterval, arguments.QueryMonitoringCountThreshold,
		arguments.QueryMonitoringResponseTimeThreshold, config.TextTruncateLimit)
	if queries[slowQueriesIndex].Query != configQueries[slowQueriesIndex].Query {
		t.Errorf("expected: %s, got: %s", configQueries[slowQueriesIndex].Query, queries[slowQueriesIndex].Query)
	}
}

func TestLoadQueries_WaitAnalysis(t *testing.T) {
	// Initial Configuration and Argument Setup
	configQueries := config.Queries
	var args args.ArgumentList

	// Prepare Arguments
	args.QueryMonitoringFetchInterval = 15
	args.QueryMonitoringCountThreshold = 10

	// Locate the index of the "waitAnalysis" query
	waitQueriesIndex := -1
	for i, query := range configQueries {
		if query.Type == "waitAnalysis" {
			waitQueriesIndex = i
			break
		}
	}

	// Ensure the "waitAnalysis" query is found
	if waitQueriesIndex == -1 {
		t.Fatalf("could not find 'waitAnalysis' in the list of queries")
	}

	// Modify the query string in preparation for comparison
	expectedQuery := fmt.Sprintf(
		configQueries[waitQueriesIndex].Query, args.QueryMonitoringCountThreshold, config.TextTruncateLimit)

	// Invoke the function under test
	queries, err := LoadQueries(config.Queries, args)
	assert.Nil(t, err, "expected no error, got an error instead")

	// Verify that the "waitAnalysis" query was modified as expected
	assert.Equal(t, expectedQuery, queries[waitQueriesIndex].Query, "expected query to match the modified query definition")
}

func TestLoadQueries_BlockingSessions(t *testing.T) {
	// Initial Configuration and Argument Setup
	configQueries := config.Queries
	var args args.ArgumentList

	// Prepare Arguments
	args.QueryMonitoringFetchInterval = 15
	args.QueryMonitoringCountThreshold = 10

	// Locate the index of the "blockingSessions" query
	blockQueriesIndex := -1
	for i, query := range configQueries {
		if query.Type == "blockingSessions" {
			blockQueriesIndex = i
			break
		}
	}

	// Ensure the "blockingSessions" query is found
	if blockQueriesIndex == -1 {
		t.Fatalf("could not find 'blockingSessions' in the list of queries")
	}

	// Modify the expected query string in preparation for comparison
	expectedQuery := fmt.Sprintf(
		configQueries[blockQueriesIndex].Query, args.QueryMonitoringCountThreshold, config.TextTruncateLimit)

	// Invoke the function under test
	queries, err := LoadQueries(config.Queries, args)
	assert.Nil(t, err, "expected no error, got an error instead")

	// Verify that the "blockingSessions" query was modified as expected
	assert.Equal(t, expectedQuery, queries[blockQueriesIndex].Query, "expected query to match the modified query definition")
}

func TestLoadQueries_UnknownType(t *testing.T) {
	config.Queries = []models.QueryDetailsDto{
		{
			EventName: "UnknownTypeQuery",
			Query:     "SELECT * FROM mysterious_table",
			Type:      "unknownType",
		},
	}

	args := args.ArgumentList{
		QueryMonitoringFetchInterval:         15,
		QueryMonitoringCountThreshold:        100,
		QueryMonitoringResponseTimeThreshold: 200,
	}

	// Call the function under test
	_, err := LoadQueries(config.Queries, args)
	if err == nil {
		t.Fatalf("expected error for unknown query type, got nil")
	}

	// Verify that the error message is as expected
	expectedError := "unknown query type: unknownType"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

// utils_test.go
func TestLoadQueries_AllTypes_AllFormats(t *testing.T) {
	// Setup: Ensure config.Queries uses all %d format specifiers as intended
	config.Queries = []models.QueryDetailsDto{
		{
			EventName: "MSSQLTopSlowQueries",
			Type:      "slowQueries",
			Query:     "SELECT * FROM slow_queries WHERE condition",
		},
		{
			EventName: "MSSQLWaitTimeAnalysis",
			Type:      "waitAnalysis",
			Query:     "SELECT * FROM wait_analysis WHERE condition",
		},
		{
			EventName: "MSSQLBlockingSessionQueries",
			Type:      "blockingSessions",
			Query:     "SELECT * FROM blocking_sessions WHERE condition",
		},
	}

	// Setup: Create a sample ArgumentList with realistic values that will be used to replace the %d format specifiers
	sampleArgs := args.ArgumentList{
		QueryMonitoringFetchInterval:         15,
		QueryMonitoringCountThreshold:        25,
		QueryMonitoringResponseTimeThreshold: 35,
	}
	// Expected queries after formatting
	expectedQueries := []models.QueryDetailsDto{
		{
			EventName: "MSSQLTopSlowQueries",
			Type:      "slowQueries",
			Query:     fmt.Sprintf(config.Queries[0].Query, sampleArgs.QueryMonitoringFetchInterval, sampleArgs.QueryMonitoringCountThreshold, sampleArgs.QueryMonitoringResponseTimeThreshold, config.TextTruncateLimit),
		},
		{
			EventName: "MSSQLWaitTimeAnalysis",
			Type:      "waitAnalysis",
			Query:     fmt.Sprintf(config.Queries[1].Query, sampleArgs.QueryMonitoringCountThreshold, config.TextTruncateLimit),
		},
		{
			EventName: "MSSQLBlockingSessionQueries",
			Type:      "blockingSessions",
			Query:     fmt.Sprintf(config.Queries[2].Query, sampleArgs.QueryMonitoringCountThreshold, config.TextTruncateLimit),
		},
	}
	// Execute the function
	loadedQueries, err := LoadQueries(config.Queries, sampleArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Assertions
	if len(loadedQueries) != len(expectedQueries) {
		t.Errorf("expected %d queries, got %d", len(expectedQueries), len(loadedQueries))
	}
	for i, expected := range expectedQueries {
		if len(loadedQueries) <= i {
			t.Fatalf("missing query at index %d", i)
		}
		if loadedQueries[i].EventName != expected.EventName {
			t.Errorf("query %d: expected name '%s', got '%s'", i, expected.EventName, loadedQueries[i].EventName)
		}
		if loadedQueries[i].Type != expected.Type {
			t.Errorf("query %d: expected type '%s', got '%s'", i, expected.Type, loadedQueries[i].Type)
		}
		// Compare the formatted queries
		if loadedQueries[i].Query != expected.Query {
			t.Errorf("query %d: \nexpected query:\n%s\ngot query:\n%s", i, expected.Query, loadedQueries[i].Query)
		}
	}
}

func TestLoadQueries_EmptyConfig(t *testing.T) {
	// Setup: Empty config.Queries
	config.Queries = []models.QueryDetailsDto{}

	// Setup: Sample ArgumentList
	sampleArgs := args.ArgumentList{
		QueryMonitoringFetchInterval:         10,
		QueryMonitoringCountThreshold:        20,
		QueryMonitoringResponseTimeThreshold: 30,
	}

	// Execute the function
	loadedQueries, err := LoadQueries(config.Queries, sampleArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assertions
	if len(loadedQueries) != 0 {
		t.Errorf("expected 0 queries, got %d", len(loadedQueries))
	}
}

func TestDetectMetricType_GaugeCase(t *testing.T) {
	value := "123.45"
	expected := metric.GAUGE

	result := metrics.DetectMetricType(value)

	assert.Equal(t, expected, result, "expected GAUGE for a parsable float string")
}

func TestDetectMetricType_AttributeCase(t *testing.T) {
	value := "NotANumber123"
	expected := metric.ATTRIBUTE

	result := metrics.DetectMetricType(value)

	assert.Equal(t, expected, result, "expected ATTRIBUTE for a non-parsable float string")
}

func TestDetectMetricType_EmptyString(t *testing.T) {
	value := ""
	expected := metric.ATTRIBUTE

	result := metrics.DetectMetricType(value)

	assert.Equal(t, expected, result, "expected ATTRIBUTE for an empty string")
}

func TestDetectMetricType_Integer(t *testing.T) {
	value := "78"
	expected := metric.GAUGE

	result := metrics.DetectMetricType(value)

	assert.Equal(t, expected, result, "expected GAUGE for integer string")
}

func TestAnonymizeQueryText_SingleQuotedStrings(t *testing.T) {
	query := "SELECT * FROM users WHERE username = 'admin' AND password = 'secret'"
	expected := "SELECT * FROM users WHERE username = ? AND password = ?"

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should replace single-quoted strings with '?'")
}

func TestAnonymizeQueryText_DoubleQuotedStrings(t *testing.T) {
	query := `SELECT * FROM config WHERE name = "config_value"`
	expected := "SELECT * FROM config WHERE name = ?"

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should replace double-quoted strings with '?'")
}

func TestAnonymizeQueryText_Numbers(t *testing.T) {
	query := "UPDATE orders SET price = 299, quantity = 3 WHERE order_id = 42"
	expected := "UPDATE orders SET price = ?, quantity = ? WHERE order_id = ?"

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should replace numbers with '?'")
}

func TestAnonymizeQueryText_MixedContent(t *testing.T) {
	query := "SELECT name, 'value' FROM table WHERE age > 30 AND id = 2"
	expected := "SELECT name, ? FROM table WHERE age > ? AND id = ?"

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should handle mixed content of strings and numbers")
}

func TestAnonymizeQueryText_EmptyString(t *testing.T) {
	query := ""
	expected := ""

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should handle empty string gracefully")
}

func TestAnonymizeQueryText_NoSensitiveData(t *testing.T) {
	query := "SELECT name FROM users"
	expected := query // No change expected

	query = AnonymizeQueryText(query)

	assert.Equal(t, expected, query, "anonymized query should remain unchanged if there is no sensitive data")
}

func TestAnonymizeQueryText(t *testing.T) {
	query := "SELECT * FROM users WHERE id = 1 AND name = 'John'"
	expected := "SELECT * FROM users WHERE id = ? AND name = ?"
	query = AnonymizeQueryText(query)
	assert.Equal(t, expected, query)
	query = "SELECT * FROM employees WHERE id = 10 OR name <> 'John Doe'   OR name != 'John Doe'   OR age < 30 OR age <= 30   OR salary > 50000OR salary >= 50000  OR department LIKE 'Sales%' OR department ILIKE 'sales%'OR join_date BETWEEN '2023-01-01' AND '2023-12-31' OR department IN ('HR', 'Engineering', 'Marketing') OR department IS NOT NULL OR department IS NULL;"
	expected = "SELECT * FROM employees WHERE id = ? OR name <> ?   OR name != ?   OR age < ? OR age <= ?   OR salary > ?OR salary >= ?  OR department LIKE ? OR department ILIKE ?OR join_date BETWEEN ? AND ? OR department IN (?, ?, ?) OR department IS NOT NULL OR department IS NULL;"
	query = AnonymizeQueryText(query)
	assert.Equal(t, expected, query)
}
