package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

const versionCompatibility = 90

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) bool {
	log.Debug("Starting pre-requisite validation")

	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	if err != nil {
		log.Error("Error getting database details:", err)
		return false
	}

	if !checkDatabaseCompatibility(databaseDetails) || !checkQueryStores(databaseDetails) {
		return false
	}

	if !checkPermissionsAndLogin(sqlConnection) {
		return false
	}

	log.Debug("Pre-requisite validation completed successfully")
	return true
}

func checkDatabaseCompatibility(databaseDetails []models.DatabaseDetailsDto) bool {
	allCompatible := true
	for _, database := range databaseDetails {
		if database.Compatibility > versionCompatibility {
			log.Debug("Database %s is compatible with the integration", database.Name)
		} else {
			log.Debug("Database %s is not compatible with the integration", database.Name)
			allCompatible = false
		}
	}
	if !allCompatible {
		log.Error("Some databases are not compatible with the integration. Upgrade the database: https://docs.newrelic.com/install/microsoft-sql/")
	}
	return allCompatible
}

func checkPermissionsAndLogin(sqlConnection *connection.SQLConnection) bool {
	hasPerms, err := checkPermissions(sqlConnection)
	if err != nil {
		log.Error("Error checking permissions:", err)
		return false
	}
	if !hasPerms {
		log.Error("Missing permissions to access sys.dm_exec_query_stats. Obtain permissions: https://docs.newrelic.com/install/microsoft-sql/")
		return false
	}

	isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)
	if err != nil {
		log.Error("Error checking if SQL Server login is enabled:", err)
		return false
	}
	if !isLoginEnabled {
		log.Error("SQL Server login not enabled. Enable login: https://learn.microsoft.com/en-us/sql/database-engine/configure-windows/change-server-authentication-mode?view=sql-server-ver16&tabs=ssms")
		return false
	}
	return true
}

func checkQueryStores(databaseDetails []models.DatabaseDetailsDto) bool {
	allQueryStoresOff := true
	for _, database := range databaseDetails {
		if database.IsQueryStoreOn {
			log.Debug("Query store enabled for database %s and make sure query capture mode is ALL", database.Name)
			allQueryStoresOff = false
		} else {
			log.Debug("Query store disabled for database %s. Turn on with: ALTER DATABASE %s SET QUERY_STORE = ON (QUERY_CAPTURE_MODE = ALL);", database.Name, database.Name)
		}
	}
	if allQueryStoresOff {
		log.Error("Query store is turned off for all databases. Turn on query store and set capture mode to ALL: https://docs.newrelic.com/install/microsoft-sql/")
	}
	return !allQueryStoresOff
}
