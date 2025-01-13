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
	rows := sqlmock.NewRows([]string{"database_id", "name", "compatibility_level", "is_query_store_on"}).
		AddRow(100, "testdb1", 100, true).
		AddRow(101, "testdb2", 110, false).
		AddRow(1, "master", 100, true) // Should be filtered out based on database_id

	// Make the regular expression for the SQL query case-insensitive
	mock.ExpectQuery("(?i)^SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases$").WillReturnRows(rows)

	// Call the function
	databaseDetails, err := GetDatabaseDetails(sqlConnection)

	// Assertions
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

	// Update the mocked query to match the new SQL structure with "database_id"
	mock.ExpectQuery("(?i)^SELECT database_id, name, compatibility_level, is_query_store_on FROM sys\\.databases$").
		WillReturnError(errQueryError)

	// Call the function
	databaseDetails, err := GetDatabaseDetails(sqlConnection)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, databaseDetails)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
