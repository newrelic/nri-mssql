package validation

import (
	"regexp"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/newrelic/nri-mssql/src/connection"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).WillReturnRows(rows)
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	assert.NoError(t, err)
	assert.Len(t, databaseDetails, 2) // Only 2 databases should be returned
	assert.Equal(t, "testdb1", databaseDetails[0].Name)
	assert.Equal(t, 100, databaseDetails[0].Compatibility)
	assert.Equal(t, true, databaseDetails[0].IsQueryStoreOn)
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
	errQueryError := sqlmock.ErrCancelled
	mock.ExpectQuery(regexp.QuoteMeta(getDatabaseDetailsQuery)).
		WillReturnError(errQueryError)
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	assert.Error(t, err)
	assert.Nil(t, databaseDetails)
	assert.NoError(t, mock.ExpectationsWereMet())
}
