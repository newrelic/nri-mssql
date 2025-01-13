package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestValidatePreConditions(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock GetDatabaseDetails to match the updated query
	mock.ExpectQuery("(?i)SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases").WillReturnRows(
		sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
			AddRow(100, "TestDB", 100, true),
	)

	// Mock checkPermissions
	mock.ExpectQuery("(?i)SELECT CASE WHEN IS_SRVROLEMEMBER\\('sysadmin'\\) = 1 OR HAS_PERMS_BY_NAME\\(null, null, 'VIEW SERVER STATE'\\) = 1 THEN 1 ELSE 0 END AS has_permission").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery("(?i)SELECT CASE WHEN SERVERPROPERTY\\('IsIntegratedSecurityOnly'\\) = 0 THEN 1 ELSE 0 END AS is_login_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := ValidatePreConditions(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_ErrorGettingDatabaseDetails(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Assume errQueryError is predefined somewhere in your test suite
	errQueryError := errors.New("mock query error")

	// Mock GetDatabaseDetails error
	mock.ExpectQuery("(?i)SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases").
		WillReturnError(errQueryError)

	result := ValidatePreConditions(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
