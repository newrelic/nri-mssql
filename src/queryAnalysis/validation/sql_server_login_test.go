package validation

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckSQLServerLoginEnabled(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected rows
	rows := sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true)

	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(rows)

	// Call the function
	isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, isLoginEnabled)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSQLServerLoginEnabled_Disabled(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected rows
	rows := sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(false)

	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(rows)

	// Call the function
	isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)

	// Assertions
	assert.NoError(t, err)
	assert.False(t, isLoginEnabled)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSQLServerLoginEnabled_Error(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected error
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnError(fmt.Errorf("query error"))

	// Call the function
	isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)

	// Assertions
	assert.Error(t, err)
	assert.False(t, isLoginEnabled)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
