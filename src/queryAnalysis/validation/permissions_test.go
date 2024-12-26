package validation

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckPermissions(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected rows
	rows := sqlmock.NewRows([]string{"has_permission"}).AddRow(true)

	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(rows)

	// Call the function
	hasPermission, err := checkPermissions(sqlConnection)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, hasPermission)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissions_NoPermission(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected rows
	rows := sqlmock.NewRows([]string{"has_permission"}).AddRow(false)

	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(rows)

	// Call the function
	hasPermission, err := checkPermissions(sqlConnection)

	// Assertions
	assert.NoError(t, err)
	assert.False(t, hasPermission)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissions_Error(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected error
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnError(fmt.Errorf("query error"))

	// Call the function
	hasPermission, err := checkPermissions(sqlConnection)

	// Assertions
	assert.Error(t, err)
	assert.False(t, hasPermission)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
