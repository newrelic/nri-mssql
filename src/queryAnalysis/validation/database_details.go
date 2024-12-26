package validation

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

// GetDatabaseDetails gets the details of user databases
func GetDatabaseDetails(sqlConnection *connection.SQLConnection) ([]models.DatabaseDetailsDto, error) {
	rows, err := sqlConnection.Queryx("SELECT name, compatibility_level, is_query_store_on FROM sys.databases")
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

		if model.Name != "master" && model.Name != "tempdb" && model.Name != "model" && model.Name != "msdb" {
			databaseDetailsResults = append(databaseDetailsResults, model)
		}
	}

	return databaseDetailsResults, nil
}
