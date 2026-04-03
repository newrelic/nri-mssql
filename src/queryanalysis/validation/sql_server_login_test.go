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
	mockCheckSQLServerLoginEnabled(mock, 0) // Mixed mode (SQL Server + Windows auth)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_WindowsOnlyAuthMode(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	mockCheckPermissions(mock, true)
	mockCheckSQLServerLoginEnabled(mock, 1) // Windows Authentication Only mode

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result) // Windows-only mode must be valid for query monitoring
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
