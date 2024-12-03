package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"sync"
)

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) (bool, error) {
	// Get database name
	databaseName, err := GetDatabaseName(sqlConnection)
	if err != nil {
		log.Error("Error getting database name:", err)
		return false, err
	}

	// If the database name is empty, return early
	if databaseName == "" {
		log.Error("Database name is empty.")
		return false, nil
	}

	var wg sync.WaitGroup

	// Using separate variables for each check result
	var isDatabaseVersionCompatible, isQueryStoreEnabled, hasPermissions, isSQLServerLoginEnabled, isTcpEnabled bool

	// Database version compatibility check
	wg.Add(1)
	go func() {
		defer wg.Done()
		isCompatible, err := checkDatabaseVersionCompatibilityLevel(sqlConnection)
		if err != nil {
			log.Error("Error checking database version compatibility level:", err)
			return
		}
		isDatabaseVersionCompatible = isCompatible
	}()

	// Query Store check
	wg.Add(1)
	go func() {
		defer wg.Done()
		isEnabled, err := checkQueryStoreEnabled(sqlConnection, databaseName)
		if err != nil {
			log.Error("Error checking if Query Store is enabled:", err)
			return
		}
		isQueryStoreEnabled = isEnabled
	}()

	// Permissions check
	wg.Add(1)
	go func() {
		defer wg.Done()
		hasPerms, err := checkPermissions(sqlConnection)
		if err != nil {
			log.Error("Error checking permissions:", err)
			return
		}
		hasPermissions = hasPerms
	}()

	// SQL Server login check
	wg.Add(1)
	go func() {
		defer wg.Done()
		isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)
		if err != nil {
			log.Error("Error checking if SQL Server login is enabled:", err)
			return
		}
		isSQLServerLoginEnabled = isLoginEnabled
	}()

	// TCP check
	wg.Add(1)
	go func() {
		defer wg.Done()
		isTcp, err := checkTcpEnabled(sqlConnection)
		if err != nil {
			log.Error("Error checking if TCP is enabled:", err)
			return
		}
		isTcpEnabled = isTcp
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Return consolidated boolean result
	return isDatabaseVersionCompatible &&
		isQueryStoreEnabled &&
		hasPermissions &&
		isSQLServerLoginEnabled &&
		isTcpEnabled, nil
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
