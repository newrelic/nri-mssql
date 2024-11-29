package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) (bool, error) {
	// Get database name
	databaseName, err := GetDatabaseName(sqlConnection)
	if err != nil {
		log.Error("Error getting database name:", err)
		return false, err
	}

	if databaseName != "" {

		// Database version compatibility check
		isDatabaseVersionCompatible, err := checkDatabaseVersionCompatibilityLevel(sqlConnection)
		if err != nil {
			log.Error("Error checking compatibility level:", err)
			return false, err
		}

		// Query Store check
		isQueryStoreEnabled, err := checkQueryStoreEnabled(sqlConnection, databaseName)
		if err != nil {

			log.Error("Error checking if Query Store is enabled:", err)
			return false, err
		}
		// Permissions check
		hasPermissions, err := checkPermissions(sqlConnection)
		if err != nil {
			log.Error("Error checking permissions:", err)
		}

		// SQL Server login check
		isSQLServerLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)
		if err != nil {
			log.Error("Error checking if SQL Server login is enabled:", err)
		}
		//Tcp
		isTcpEnabled, err := checkTcpEnabled(sqlConnection)
		if err != nil {
			log.Error("Error checking if TCP is enabled:", err)
		}

		return isDatabaseVersionCompatible && isQueryStoreEnabled && hasPermissions && isSQLServerLoginEnabled && isTcpEnabled, nil
	}

	return false, nil
}

// GetDatabaseName gets the name of the database
func GetDatabaseName(sqlConnection *connection.SQLConnection) (string, error) {
	var databaseName string
	err := sqlConnection.Connection.Get(&databaseName, "SELECT DB_NAME()")
	if err != nil {
		return "", err
	}
	return databaseName, nil

}
