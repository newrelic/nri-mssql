package validation

import (
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
)

func checkPermissions(sqlConnection *connection.SQLConnection) (bool, error) {
	var hasPermission bool
	query := `
		SELECT 
			CASE 
				WHEN IS_SRVROLEMEMBER('sysadmin') = 1 OR HAS_PERMS_BY_NAME(null, null, 'VIEW SERVER STATE') = 1 
				THEN 1 
				ELSE 0 
			END AS has_permission
	`
	err := sqlConnection.Connection.Get(&hasPermission, query)
	if err != nil {
		return false, err
	}

	return hasPermission, nil
}
