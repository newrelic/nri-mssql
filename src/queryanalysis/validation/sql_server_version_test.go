package validation

import (
	"regexp"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckSqlServerVersion_SupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a supported SQL Server version response
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2019 (RTM) - 15.0.2000.5"))

	result, _ := checkSQLServerVersion(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_UnsupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking an unsupported SQL Server version response
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2014 - 12.0.2000.8"))

	result, _ := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckSqlServerVersion_EmptyVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(""))
	result, _ := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckSqlServerVersion_InvalidVersionString(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a malformed version string
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server  - version unknown"))

	result, _ := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestCheckSqlServerVersion_QueryError(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()
	mock.ExpectQuery(regexp.QuoteMeta(getSQLServerVersionQuery)).WillReturnError(errQueryError)
	result, _ := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSQLServerVersion(t *testing.T) {
	version, err := parseSQLServerVersion("Microsoft SQL Server 2019 (RTM) - 15.0.2000.5")
	assert.NoError(t, err)
	if version.Major > uint64(^uint(0)>>1) {
		t.Fatalf("version.Major value %d is too large to convert to int", version.Major)
	}
	assert.Equal(t, 15, int(version.Major))
}

func TestGetSQLServerVersion_EmptyVersion(t *testing.T) {
	_, err := parseSQLServerVersion("")
	assert.Error(t, err)
}

func TestGetSQLServerVersion_InvalidVersionString(t *testing.T) {
	_, err := parseSQLServerVersion("Microsoft SQL Server  - version unknown")
	assert.Error(t, err)
}

func TestGetSQLServerVersion_ParseError(t *testing.T) {
	_, err := parseSQLServerVersion("Invalid SQL Server version string")
	assert.Error(t, err)
}

func TestIsSQLServerVersionSupported_SupportedVersion(t *testing.T) {
	version := semver.Version{Major: 15}
	assert.True(t, isSQLServerVersionSupported(version))
}

func TestIsSQLServerVersionSupported_UnsupportedVersion(t *testing.T) {
	version := semver.Version{Major: 13}
	assert.False(t, isSQLServerVersionSupported(version))
}
