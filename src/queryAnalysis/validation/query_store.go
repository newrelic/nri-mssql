package validation

import "github.com/newrelic/nri-mssql/src/queryAnalysis/connection"

// checkQueryStoreEnabled verifies if the Query Store is enabled in the database
func checkQueryStoreEnabled(sqlConnection *connection.SQLConnection, name string) (bool, error) {
	rows, err := sqlConnection.Queryx("ALTER DATABASE <database_name>\nSET QUERY_STORE = ON;")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var isQueryStoreEnabled bool
	for rows.Next() {
		var isEnabled bool
		if err := rows.Scan(&isEnabled); err != nil {
			return false, err
		}
		if isEnabled {
			isQueryStoreEnabled = true
			break
		}
	}

	return isQueryStoreEnabled, nil
}
