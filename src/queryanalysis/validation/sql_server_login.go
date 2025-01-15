package validation

import (
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
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
	return isLoginEnabled, nil
}
