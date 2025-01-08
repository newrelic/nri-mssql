package validation

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestGetDatabaseDetails(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected rows
	rows := sqlmock.NewRows([]string{"name", "compatibility_level", "is_query_store_on"}).
		AddRow("testdb1", 100, true).
		AddRow("testdb2", 110, false).
		AddRow("master", 100, true) // This should be filtered out

	// Make the regular expression for the SQL query case-insensitive
	mock.ExpectQuery("(?i)^SELECT name, compatibility_level, is_query_store_on FROM sys\\.databases$").WillReturnRows(rows)

	// Call the function
	databaseDetails, err := GetDatabaseDetails(sqlConnection)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, databaseDetails, 2) // Only 2 databases should be returned
	assert.Equal(t, "testdb1", databaseDetails[0].Name)
	assert.Equal(t, 100, databaseDetails[0].Compatibility)
	assert.Equal(t, true, databaseDetails[0].IsQueryStoreOn)
	assert.Equal(t, "testdb2", databaseDetails[1].Name)
	assert.Equal(t, 110, databaseDetails[1].Compatibility)
	assert.Equal(t, false, databaseDetails[1].IsQueryStoreOn)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDatabaseDetails_Error(t *testing.T) {
	// Create a new mock database connection
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlConnection := &connection.SQLConnection{Connection: sqlx.NewDb(db, "sqlmock")}

	// Define the expected error
	errQueryError := sqlmock.ErrCancelled // or use the appropriate error you expect

	mock.ExpectQuery("(?i)^SELECT name, compatibility_level, is_query_store_on FROM sys\\.databases$").
		WillReturnError(errQueryError)

	// Call the function
	databaseDetails, err := GetDatabaseDetails(sqlConnection)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, databaseDetails)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
