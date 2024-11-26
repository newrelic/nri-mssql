package queryhandler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

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
	fmt.Println("TestExecuteQuery_Success")
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

func TestBindQueryResults_Success(t *testing.T) {
	// Create sqlmock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Field1", "Field2"}).
		AddRow("value1", "value2").
		AddRow("value3", "value4")

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	sqlxRows, err := sqlxDB.Queryx("SELECT * FROM test")
	require.NoError(t, err)

	qh := &QueryHandlerImpl{}

	type TestStruct struct {
		Field1 string `db:"Field1"`
		Field2 string `db:"Field2"`
	}

	var result []TestStruct
	err = qh.BindQueryResults(sqlxRows, &result)
	require.NoError(t, err)

	expected := []TestStruct{
		{Field1: "value1", Field2: "value2"},
		{Field1: "value3", Field2: "value4"},
	}
	assert.Equal(t, expected, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBindQueryResults_InvalidResultType(t *testing.T) {
	// Create sqlmock database connection
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Field1", "Field2"}).
		AddRow("value1", "value2")

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	sqlxRows, err := sqlxDB.Queryx("SELECT * FROM test")
	require.NoError(t, err)

	qh := &QueryHandlerImpl{}

	result := "InvalidType"
	err = qh.BindQueryResults(sqlxRows, &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "result argument must be a pointer to a slice")
}
