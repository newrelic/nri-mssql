package validation

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestGetDatabaseDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}
	rows := sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
		AddRow(100, "testdb1", 100, true).
		AddRow(101, "testdb2", 110, false).
		AddRow(1, "master", 100, true) // Should be filtered out based on database_id
	mock.ExpectQuery("SELECT @@VERSION\n").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("Microsoft SQL Server 2019 - 15.0.2000.5 (X64) \n\tSep 24 2019 13:48:23 \n\tCopyright (c) 1988-2019, Microsoft Corporation \n\tDeveloper Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 18363: )"))
	mock.ExpectQuery("(?i)^SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases$").WillReturnRows(rows)
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	assert.NoError(t, err)
	assert.Len(t, databaseDetails, 2) // Only 2 databases should be returned
	assert.Equal(t, 100, databaseDetails[0].DatabaseID)
	assert.Equal(t, "testdb1", databaseDetails[0].Name)
	assert.Equal(t, 100, databaseDetails[0].Compatibility)
	assert.Equal(t, true, databaseDetails[0].IsQueryStoreOn)
	assert.Equal(t, 101, databaseDetails[1].DatabaseID)
	assert.Equal(t, "testdb2", databaseDetails[1].Name)
	assert.Equal(t, 110, databaseDetails[1].Compatibility)
	assert.Equal(t, false, databaseDetails[1].IsQueryStoreOn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatabaseDetails_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}
	errQueryError := sqlmock.ErrCancelled // or use the appropriate error you expect
	mock.ExpectQuery("SELECT @@VERSION\n").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("Microsoft SQL Server 2019 - 15.0.2000.5 (X64) \n\tSep 24 2019 13:48:23 \n\tCopyright (c) 1988-2019, Microsoft Corporation \n\tDeveloper Edition (64-bit) on Windows 10 Pro 10.0 <X64> (Build 18363: )"))
	mock.ExpectQuery("(?i)^SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases$").
		WillReturnError(errQueryError)
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	assert.Error(t, err)
	assert.Nil(t, databaseDetails)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatabaseDetails_UnsupportedVersion(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	assert.Nil(t, err)
	assert.Nil(t, databaseDetails)
}
