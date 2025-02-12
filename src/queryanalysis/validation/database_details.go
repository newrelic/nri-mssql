package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/config"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

const getDatabaseDetailsQuery = `
	SELECT database_id, name, compatibility_level, is_query_store_on 
	FROM sys.databases
`

// GetDatabaseDetails gets the details of user databases
func GetDatabaseDetails(sqlConnection *connection.SQLConnection) ([]models.DatabaseDetailsDto, error) {
	rows, err := sqlConnection.Queryx(getDatabaseDetailsQuery)
	if err != nil {
		log.Error("Error getting database details:", err)
		return nil, err
	}
	defer rows.Close()

	var databaseDetailsResults []models.DatabaseDetailsDto
	for rows.Next() {
		var model models.DatabaseDetailsDto
		if err := rows.StructScan(&model); err != nil {
			log.Error("Error scanning database details row:", err)
			return nil, err
		}

		// Filter out system databases using their database_id
		if model.DatabaseID > config.MaxSystemDatabaseID {
			databaseDetailsResults = append(databaseDetailsResults, model)
		}
	}
	return databaseDetailsResults, nil
}
