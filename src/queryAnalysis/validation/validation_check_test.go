package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestValidatePreConditions(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock GetDatabaseDetails
	mock.ExpectQuery("(?i)select name, compatibility_level, is_query_store_on from sys\\.databases").WillReturnRows(
		sqlmock.NewRows([]string{"name", "compatibility_level", "is_query_store_on"}).
			AddRow("TestDB", 100, true),
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

	// Mock GetDatabaseDetails error
	mock.ExpectQuery("(?i)select name, compatibility_level, is_query_store_on from sys\\.databases").WillReturnError(errQueryError)

	result := ValidatePreConditions(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
