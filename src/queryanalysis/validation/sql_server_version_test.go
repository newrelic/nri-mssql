package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCheckSqlServerVersion_SupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a supported SQL Server version response
	mock.ExpectQuery("SELECT @@VERSION").
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2019 (RTM) - 15.0.2000.5"))

	result := checkSQLServerVersion(sqlConnection)
	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_UnsupportedVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking an unsupported SQL Server version response
	mock.ExpectQuery("SELECT @@VERSION").
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server 2014 - 12.0.2000.8"))

	result := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_EmptyVersion(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking an empty SQL Server version response
	mock.ExpectQuery("SELECT @@VERSION").
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow(""))

	result := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_InvalidVersionString(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking a malformed version string
	mock.ExpectQuery("SELECT @@VERSION").
		WillReturnRows(sqlmock.NewRows([]string{"@@VERSION"}).AddRow("Microsoft SQL Server  - version unknown"))

	result := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSqlServerVersion_QueryError(t *testing.T) {
	sqlConnection, mock := setupMockDB(t)
	defer sqlConnection.Connection.Close()

	// Mocking an error on querying for SQL Server version
	mock.ExpectQuery("SELECT @@VERSION").WillReturnError(errQueryError)

	result := checkSQLServerVersion(sqlConnection)
	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
