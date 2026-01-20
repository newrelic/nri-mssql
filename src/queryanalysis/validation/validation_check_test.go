package validation

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestValidatePreConditions(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("Microsoft SQL Server 2019 - 15.0.2000.5 (X64) \n\tSep 24 2019 13:48:23 \n\tCopyright (c) 1988-2019, Microsoft Corporation \n\tDeveloper Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 18363: )"))
	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
			AddRow(100, "TestDB", 100, true),
	)

	// Mock checkPermissions
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery(regexp.QuoteMeta(checkSQLServerLoginEnabledQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := ValidatePreConditions(sqlConnection, false)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_ErrorGettingDatabaseDetails(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("Microsoft SQL Server 2019 - 15.0.2000.5 (X64) \n\tSep 24 2019 13:48:23 \n\tCopyright (c) 1988-2019, Microsoft Corporation \n\tDeveloper Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 18363: )"))
	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).
		WillReturnError(errQueryError)
	result := ValidatePreConditions(sqlConnection, false)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// New tests for DMV-only mode and SQL Server 2016
func TestValidatePreConditions_DMVOnlyMode_SQL2016(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock SQL Server 2016 version
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).
			AddRow("Microsoft SQL Server 2016 (RTM) - 13.0.1601.5 (X64)"))

	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
			AddRow(100, "TestDB", 130, true),
	)

	// Mock checkPermissions
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery(regexp.QuoteMeta(checkSQLServerLoginEnabledQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := ValidatePreConditions(sqlConnection, true)
	assert.True(t, result, "SQL Server 2016 should pass validation in DMV-only mode")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_QueryStoreMode_SQL2016(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock SQL Server 2016 version
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).
			AddRow("Microsoft SQL Server 2016 (RTM) - 13.0.1601.5 (X64)"))

	result := ValidatePreConditions(sqlConnection, false)
	assert.False(t, result, "SQL Server 2016 should fail validation in Query Store mode")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_QueryStoreMode_SQL2017(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mock SQL Server 2017 version (should pass with Query Store mode)
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).
			AddRow("Microsoft SQL Server 2017 - 14.0.1000.169"))

	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
			AddRow(100, "TestDB", 140, true),
	)

	// Mock checkPermissions
	mock.ExpectQuery(regexp.QuoteMeta(checkPermissionsQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	// Mock checkSQLServerLoginEnabled
	mock.ExpectQuery(regexp.QuoteMeta(checkSQLServerLoginEnabledQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"is_login_enabled"}).AddRow(true))

	result := ValidatePreConditions(sqlConnection, false)
	assert.True(t, result, "SQL Server 2017 should pass validation in Query Store mode")
	assert.NoError(t, mock.ExpectationsWereMet())
}
