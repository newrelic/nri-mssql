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

	result := ValidatePreConditions(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidatePreConditions_ErrorGettingDatabaseDetails(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("Microsoft SQL Server 2019 - 15.0.2000.5 (X64) \n\tSep 24 2019 13:48:23 \n\tCopyright (c) 1988-2019, Microsoft Corporation \n\tDeveloper Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 18363: )"))
	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).
		WillReturnError(errQueryError)
	result := ValidatePreConditions(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
