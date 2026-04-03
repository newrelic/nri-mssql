package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/connection"
)

const checkSQLServerLoginEnabledQuery = `
	SELECT CAST(SERVERPROPERTY('IsIntegratedSecurityOnly') AS INT) AS is_windows_only_mode
`

// checkSQLServerLoginEnabled verifies that the SQL Server authentication mode is compatible with
// query monitoring. Both Mixed Mode (IsIntegratedSecurityOnly=0) and Windows Authentication Only
// mode (IsIntegratedSecurityOnly=1) are valid — in Windows-only mode the connection was already
// established via Windows Authentication, so query monitoring can proceed.
func checkSQLServerLoginEnabled(sqlConnection *connection.SQLConnection) (bool, error) {
	var isWindowsOnlyMode int
	err := sqlConnection.Connection.Get(&isWindowsOnlyMode, checkSQLServerLoginEnabledQuery)
	if err != nil {
		return false, err
	}
	if isWindowsOnlyMode == 1 {
		log.Debug("SQL Server is configured for Windows Authentication Only mode. Windows auth connection is valid for query monitoring.")
	}
	return true, nil
}
