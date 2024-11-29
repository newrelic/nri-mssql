package validation

import "github.com/newrelic/nri-mssql/src/queryAnalysis/connection"

// checkDatabaseVersionCompatibilityLevel checks if the database version is compatible
func checkDatabaseVersionCompatibilityLevel(sqlConnection *connection.SQLConnection) (bool, error) {
	rows, err := sqlConnection.Queryx("SELECT compatibility_level FROM sys.databases")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var isCompatible bool

	for rows.Next() {
		var level int
		if err := rows.Scan(&level); err != nil {
			return false, err
		}
		if level >= 90 {
			isCompatible = true
			break
		}
	}

	return isCompatible, nil
}
