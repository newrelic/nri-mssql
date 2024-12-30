// src/queryAnalysis/validation/sql_server_login_test.go
package validation

import (
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
	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled error
	mock.ExpectQuery("SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnError(errQueryError)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
