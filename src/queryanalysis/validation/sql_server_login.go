package validation

import (
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
)

const checkSQLServerLoginEnabledQuery = `
	SELECT CASE 
		WHEN SERVERPROPERTY('IsIntegratedSecurityOnly') = 0 
		THEN 1 
		ELSE 0 
	END AS is_login_enabled
`

func checkSQLServerLoginEnabled(sqlConnection *connection.SQLConnection) (bool, error) {
	var isLoginEnabled bool
	err := sqlConnection.Connection.Get(&isLoginEnabled, checkSQLServerLoginEnabledQuery)
	if err != nil {
		return false, err
	}
	return isLoginEnabled, nil
}
