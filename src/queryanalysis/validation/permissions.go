package validation

import (
	"github.com/newrelic/nri-mssql/src/connection"
)

const checkPermissionsQuery = `
	SELECT 
		CASE 
			WHEN IS_SRVROLEMEMBER('sysadmin') = 1 OR HAS_PERMS_BY_NAME(null, null, 'VIEW SERVER STATE') = 1 
			THEN 1 
			ELSE 0 
		END AS has_permission
`

func checkPermissions(sqlConnection *connection.SQLConnection) (bool, error) {
	var hasPermission bool
	err := sqlConnection.Connection.Get(&hasPermission, checkPermissionsQuery)
	if err != nil {
		return false, err
	}

	return hasPermission, nil
}
