package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

func checkSQLServerLoginEnabled(sqlConnection *connection.SQLConnection) (bool, error) {
	var isLoginEnabled bool
	query := `
		SELECT CASE 
			WHEN SERVERPROPERTY('IsIntegratedSecurityOnly') = 0 
			THEN 1 
			ELSE 0 
		END AS is_login_enabled
	`
	err := sqlConnection.Connection.Get(&isLoginEnabled, query)
	if err != nil {
		return false, err
	}

	if !isLoginEnabled {
		log.Error("You have not enabled SQL Server login. Please refer to the documentation: https://learn.microsoft.com/en-us/sql/database-engine/configure-windows/change-server-authentication-mode?view=sql-server-ver16&tabs=ssms")
	}

	return isLoginEnabled, nil
}
