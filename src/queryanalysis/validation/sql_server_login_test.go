// src/queryAnalysis/validation/sql_server_login_test.go
package validation

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckPermissionsAndLogin_LoginEnabled(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	mockCheckPermissions(mock, true)
	mockCheckSQLServerLoginEnabled(mock, true)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_LoginEnabledError(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock checkPermissions
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled error
	mock.ExpectQuery(regexp.QuoteMeta(checkSQLServerLoginEnabledQuery)).
		WillReturnError(errQueryError)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
