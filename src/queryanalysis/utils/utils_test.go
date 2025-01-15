package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/nri-mssql/src/queryanalysis/config"
	"github.com/stretchr/testify/assert"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

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
		Name:  "SlowQueries",
		Query: query,
		Type:  "slowQueries",
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
		Name:  "WaitTimeAnalysisQuery",
		Query: query,
		Type:  "waitAnalysis",
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
		Name:  "BlockingSessionsQuery",
		Query: query,
		Type:  "blockingSessions",
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

	queries, err := LoadQueries(arguments)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	configQueries[slowQueriesIndex].Query = fmt.Sprintf(configQueries[slowQueriesIndex].Query,
		arguments.FetchInterval, arguments.QueryCountThreshold,
		arguments.QueryResponseTimeThreshold, config.TextTruncateLimit)
	if queries[slowQueriesIndex].Query != configQueries[slowQueriesIndex].Query {
		t.Errorf("expected: %s, got: %s", configQueries[slowQueriesIndex].Query, queries[slowQueriesIndex].Query)
	}
}

func TestLoadQueries_WaitAnalysis(t *testing.T) {
	configQueries := config.Queries
	var args args.ArgumentList

	args.FetchInterval = 15
	args.QueryCountThreshold = 10

	waitQueriesIndex := -1
	for i, query := range configQueries {
		if query.Type == "waitAnalysis" {
			waitQueriesIndex = i
			break
		}
	}

	// Ensure the correct query was found
	if waitQueriesIndex == -1 {
		t.Fatalf("could not find 'waitAnalysis' in the list of queries")
	}

	queries, err := LoadQueries(args)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	configQueries[waitQueriesIndex].Query = fmt.Sprintf(
		configQueries[waitQueriesIndex].Query, args.FetchInterval, args.FetchInterval, args.QueryCountThreshold, config.TextTruncateLimit)
	if queries[waitQueriesIndex].Query != configQueries[waitQueriesIndex].Query {
		t.Errorf("expected: %s, got: %s", configQueries[waitQueriesIndex].Query, queries[waitQueriesIndex].Query)
	}
}

func TestLoadQueries_BlockingSessions(t *testing.T) {
	configQueries := config.Queries
	var args args.ArgumentList

	args.FetchInterval = 15
	args.QueryCountThreshold = 10

	blockQueriesIndex := -1
	for i, query := range configQueries {
		if query.Type == "blockingSessions" {
			blockQueriesIndex = i
			break
		}
	}

	// Ensure the correct query was found
	if blockQueriesIndex == -1 {
		t.Fatalf("could not find 'blockingSessions' in the list of queries")
	}

	queries, err := LoadQueries(args)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	configQueries[blockQueriesIndex].Query = fmt.Sprintf(
		configQueries[blockQueriesIndex].Query, config.TextTruncateLimit)
	if queries[blockQueriesIndex].Query != configQueries[blockQueriesIndex].Query {
		t.Errorf("expected: %s, got: %s", configQueries[blockQueriesIndex].Query, queries[blockQueriesIndex].Query)
	}
}

func TestLoadQueries_UnknownType(t *testing.T) {
	config.Queries = []models.QueryDetailsDto{
		{
			Name:  "UnknownTypeQuery",
			Query: "SELECT * FROM mysterious_table",
			Type:  "unknownType",
		},
	}

	var args args.ArgumentList
	args.FetchInterval = 15

	queries, err := LoadQueries(args)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if queries[0].Query != "SELECT * FROM mysterious_table" {
		t.Errorf("unexpected query content for unknown type: %s", queries[0].Query)
	}
}

// utils_test.go
func TestLoadQueries_AllTypes_AllFormats(t *testing.T) {
	// Setup: Ensure config.Queries uses all %d format specifiers as intended
	config.Queries = []models.QueryDetailsDto{
		{
			Name:  "MSSQLTopSlowQueries",
			Type:  "slowQueries",
			Query: "SELECT * FROM slow_queries WHERE condition",
		},
		{
			Name:  "MSSQLWaitTimeAnalysis",
			Type:  "waitAnalysis",
			Query: "SELECT * FROM wait_analysis WHERE condition",
		},
		{
			Name:  "MSSQLBlockingSessionQueries",
			Type:  "blockingSessions",
			Query: "SELECT * FROM blocking_sessions WHERE condition",
		},
	}

	// Setup: Create a sample ArgumentList with realistic values that will be used to replace the %d format specifiers
	sampleArgs := args.ArgumentList{
		FetchInterval:              15,
		QueryCountThreshold:        25,
		QueryResponseTimeThreshold: 35,
	}
	// Expected queries after formatting
	expectedQueries := []models.QueryDetailsDto{
		{
			Name:  "MSSQLTopSlowQueries",
			Type:  "slowQueries",
			Query: fmt.Sprintf(config.Queries[0].Query, sampleArgs.FetchInterval, sampleArgs.QueryCountThreshold, sampleArgs.QueryResponseTimeThreshold, config.TextTruncateLimit),
		},
		{
			Name:  "MSSQLWaitTimeAnalysis",
			Type:  "waitAnalysis",
			Query: fmt.Sprintf(config.Queries[1].Query, sampleArgs.QueryCountThreshold, config.TextTruncateLimit),
		},
		{
			Name:  "MSSQLBlockingSessionQueries",
			Type:  "blockingSessions",
			Query: fmt.Sprintf(config.Queries[2].Query, sampleArgs.QueryCountThreshold, config.TextTruncateLimit),
		},
	}
	// Execute the function
	loadedQueries, err := LoadQueries(sampleArgs)
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
		if loadedQueries[i].Name != expected.Name {
			t.Errorf("query %d: expected name '%s', got '%s'", i, expected.Name, loadedQueries[i].Name)
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
		FetchInterval:              10,
		QueryCountThreshold:        20,
		QueryResponseTimeThreshold: 30,
	}

	// Execute the function
	loadedQueries, err := LoadQueries(sampleArgs)
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

	result := DetectMetricType(value)

	assert.Equal(t, expected, result, "expected GAUGE for a parsable float string")
}

func TestDetectMetricType_AttributeCase(t *testing.T) {
	value := "NotANumber123"
	expected := metric.ATTRIBUTE

	result := DetectMetricType(value)

	assert.Equal(t, expected, result, "expected ATTRIBUTE for a non-parsable float string")
}

func TestDetectMetricType_EmptyString(t *testing.T) {
	value := ""
	expected := metric.ATTRIBUTE

	result := DetectMetricType(value)

	assert.Equal(t, expected, result, "expected ATTRIBUTE for an empty string")
}

func TestDetectMetricType_Integer(t *testing.T) {
	value := "78"
	expected := metric.GAUGE

	result := DetectMetricType(value)

	assert.Equal(t, expected, result, "expected GAUGE for integer string")
}

func TestAnonymizeQueryText_SingleQuotedStrings(t *testing.T) {
	query := "SELECT * FROM users WHERE username = 'admin' AND password = 'secret'"
	expected := "SELECT * FROM users WHERE username = ? AND password = ?"

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should replace single-quoted strings with '?'")
}

func TestAnonymizeQueryText_DoubleQuotedStrings(t *testing.T) {
	query := `SELECT * FROM config WHERE name = "config_value"`
	expected := "SELECT * FROM config WHERE name = ?"

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should replace double-quoted strings with '?'")
}

func TestAnonymizeQueryText_Numbers(t *testing.T) {
	query := "UPDATE orders SET price = 299, quantity = 3 WHERE order_id = 42"
	expected := "UPDATE orders SET price = ?, quantity = ? WHERE order_id = ?"

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should replace numbers with '?'")
}

func TestAnonymizeQueryText_MixedContent(t *testing.T) {
	query := "SELECT name, 'value' FROM table WHERE age > 30 AND id = 2"
	expected := "SELECT name, ? FROM table WHERE age > ? AND id = ?"

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should handle mixed content of strings and numbers")
}

func TestAnonymizeQueryText_EmptyString(t *testing.T) {
	query := ""
	expected := ""

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should handle empty string gracefully")
}

func TestAnonymizeQueryText_NoSensitiveData(t *testing.T) {
	query := "SELECT name FROM users"
	expected := query // No change expected

	AnonymizeQueryText(&query)

	assert.Equal(t, expected, query, "anonymized query should remain unchanged if there is no sensitive data")
}
