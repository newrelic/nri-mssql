package validation

import (
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"sync"
)

// ValidatePreConditions checks if the database is compatible with the integration
func ValidatePreConditions(sqlConnection *connection.SQLConnection) error {

	var wg sync.WaitGroup
	errorChan := make(chan error, 3) // Buffered channel to collect errors from goroutines

	// Database version compatibility check
	wg.Add(1)
	go func() {
		defer wg.Done()
		databaseDetails, err := GetDatabaseDetails(sqlConnection)
		if err != nil {
			log.Error("Error getting database details:", err)
			errorChan <- err
			return
		}

		for _, database := range databaseDetails {
			if database.Compatibility < 90 {
				errorChan <- fmt.Errorf("Database version is not compatible with the integration for this Database: %s", database.Name)
				return
			}

			if !database.IsQueryStoreOn {
				errorChan <- fmt.Errorf("Query Store is not enabled to this database : %s", database.Name)
				return
			}
		}
	}()

	// Permissions check
	wg.Add(1)
	go func() {
		defer wg.Done()
		hasPerms, err := checkPermissions(sqlConnection)
		if err != nil {
			log.Error("Error checking permissions:", err)
			errorChan <- err
			return
		}
		if hasPerms == false {
			errorChan <- fmt.Errorf("You do not have the necessary permissions to access sys.dm_exec_query_stats. Please refer to the documentation and complete the steps to obtain the required permissions: https://docs.newrelic.com/install/microsoft-sql/")
		}
	}()

	// SQL Server login check
	wg.Add(1)
	go func() {
		defer wg.Done()
		isLoginEnabled, err := checkSQLServerLoginEnabled(sqlConnection)
		if err != nil {
			log.Error("Error checking if SQL Server login is enabled:", err)
			errorChan <- err
			return
		}
		if !isLoginEnabled {
			errorChan <- fmt.Errorf("You have not enabled SQL Server login. Please refer to the documentation: https://learn.microsoft.com/en-us/sql/database-engine/configure-windows/change-server-authentication-mode?view=sql-server-ver16&tabs=ssms")
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan) // Close the channel after all goroutines have finished sending

	// Process and return the first error encountered
	for err := range errorChan {
		if err != nil {
			return err
		}
	}
	fmt.Println("All validation checks have passed successfully")

	// Return nil if all checks passed without errors
	return nil
}
