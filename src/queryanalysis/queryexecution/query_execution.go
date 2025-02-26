package queryexecution

import (
	"strings"

	"github.com/newrelic/nri-mssql/src/connection"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"

	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"github.com/newrelic/nri-mssql/src/queryanalysis/querytype"
	"github.com/newrelic/nri-mssql/src/queryanalysis/utils"
)

func ExecuteQuery(arguments args.ArgumentList, queryDetailsDto models.QueryDetailsDto, integration *integration.Integration, sqlConnection *connection.SQLConnection) ([]interface{}, error) {
	log.Debug("Executing query: %s", queryDetailsDto.Query)
	rows, err := sqlConnection.Connection.Queryx(queryDetailsDto.Query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	log.Debug("Query executed: %s", queryDetailsDto.Query)
	result, queryIDs, err := BindQueryResults(arguments, rows, queryDetailsDto, integration, sqlConnection)
	rows.Close()

	if err != nil {
		return nil, err
	}

	// Process collected query IDs for execution plan
	if len(queryIDs) > 0 {
		ProcessExecutionPlans(arguments, integration, sqlConnection, queryIDs)
	}
	return result, err
}

// BindQueryResults binds query results to the specified data model using `sqlx`
// nolint:gocyclo
func BindQueryResults(arguments args.ArgumentList,
	rows *sqlx.Rows,
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection) ([]interface{}, []models.HexString, error) {

	results := make([]interface{}, 0)
	queryIDs := make([]models.HexString, 0)

	queryType, err := querytype.CreateQueryType(queryDetailsDto.Type)
	if err != nil {
		return nil, queryIDs, err
	}

	for rows.Next() {
		if err := queryType.Bind(&results, &queryIDs, rows); err != nil {
			continue
		}
	}
	return results, queryIDs, nil
}

// ProcessExecutionPlans processes execution plans for all collected queryIDs
func ProcessExecutionPlans(arguments args.ArgumentList, integration *integration.Integration, sqlConnection *connection.SQLConnection, queryIDs []models.HexString) {
	if len(queryIDs) == 0 {
		return
	}
	stringIDs := make([]string, len(queryIDs))
	for i, qid := range queryIDs {
		stringIDs[i] = string(qid) // Cast HexString to string
	}

	// Join the converted string slice into a comma-separated list
	queryIDString := strings.Join(stringIDs, ",")

	utils.GenerateAndIngestExecutionPlan(arguments, integration, sqlConnection, queryIDString)
}
