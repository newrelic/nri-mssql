package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

const versionCompatibility = 90

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) bool {
	log.Debug("Starting pre-requisite validation")
	isSupported, err := checkSQLServerVersion(sqlConnection)
	if err != nil {
		log.Error("Error while checking SQL Server: %s", err.Error())
		return false
	}
	if !isSupported {
		log.Error("Unsupported SQL Server version.")
		return false
	}
	databaseDetails, err := GetDatabaseDetails(sqlConnection)
	if err != nil {
		log.Error("Error getting database details: %s", err.Error())
		return false
	}

	if !checkDatabaseCompatibility(databaseDetails) {
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
