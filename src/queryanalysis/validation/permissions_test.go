package validation

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).WillReturnError(errQueryError)
	result := checkPermissionsAndLogin(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
