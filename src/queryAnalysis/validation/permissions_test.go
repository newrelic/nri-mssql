// src/queryAnalysis/validation/permissions_test.go
package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errQueryError = errors.New("query error")

func TestCheckPermissionsAndLogin(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	mockCheckPermissions(mock, true)
	mockCheckSQLServerLoginEnabled(mock, true)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckPermissionsAndLogin_PermissionsError(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	mock.ExpectQuery("SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnError(errQueryError)

	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
