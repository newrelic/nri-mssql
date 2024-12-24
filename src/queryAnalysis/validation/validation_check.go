package validation

import (
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

const versionCompatibility = 90

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) bool {
	// Database version compatibility check
	databaseDetails, err := GetDatabaseDetails(sqlConnection)

	if err != nil {
		log.Error("Error getting database details:", err)
		return false
	}

	allQueryStoresOff := true
	allDatabaseVersionsCompatible := true

	for _, database := range databaseDetails {

		if database.Compatibility > versionCompatibility {
			log.Info("Database %s is compatible with the integration", database.Name)
		} else {
			log.Warn("Database %s is not compatible with the integration", database.Name)
			allDatabaseVersionsCompatible = false
		}

		if database.IsQueryStoreOn {
			log.Info("Query store for this database is turned on: %s", database.Name)
			allQueryStoresOff = false
		} else {
			log.Warn("Query store disabled for database %s. Please use this command to turn on:ALTER DATABASE %s SET QUERY_STORE = ON;", database.Name, database.Name)
		}
	}
	if !allDatabaseVersionsCompatible {
		log.Error("some databases are not compatible with the integration. Please refer to the documentation and complete the steps to upgrade the database: https://docs.newrelic.com/install/microsoft-sql/")
		return false
	}
	if allQueryStoresOff {
		log.Error("query store is turned off for all databases. Please refer to the documentation and complete the steps to turn on query store: https://docs.newrelic.com/install/microsoft-sql/")
		return false
	}

	// Permissions check
	hasPerms, err := checkPermissions(sqlConnection)
	if err != nil {
		log.Error("Error checking permissions:", err)
		return false
	}
	if !hasPerms {
		log.Error("you do not have the necessary permissions to access sys.dm_exec_query_stats. Please refer to the documentation and complete the steps to obtain the required permissions: https://docs.newrelic.com/install/microsoft-sql/")
		return false
	}

	// SQL Server login check
	isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)
	if err != nil {
		log.Error("Error checking if SQL Server login is enabled:", err)
		return false
	}
	if !isLoginEnabled {
		log.Error("you have not enabled SQL Server login. Please refer to the documentation: https://learn.microsoft.com/en-us/sql/database-engine/configure-windows/change-server-authentication-mode?view=sql-server-ver16&tabs=ssms")
		return false
	}

	fmt.Println("All validation checks have passed successfully")

	// Return nil if all checks passed without errors
	return true
}
