package queryanalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/config"
	"github.com/newrelic/nri-mssql/src/queryanalysis/utils"
	"github.com/newrelic/nri-mssql/src/queryanalysis/validation"
)

// queryPerformanceMain runs all types of analyzes
func PopulateQueryPerformanceMetrics(integration *integration.Integration, arguments args.ArgumentList) {
	// Create a new connection
	log.Debug("Starting query analysis...")

	sqlConnection, err := connection.NewConnection(&arguments)
	if err != nil {
		log.Error("Error creating connection to SQL Server: %s", err.Error())
		return
	}
	defer sqlConnection.Close()

	// Validate preconditions
	isPreconditionPassed := validation.ValidatePreConditions(sqlConnection)
	if !isPreconditionPassed {
		return
	}

	utils.ValidateAndSetDefaults(&arguments)

	queries := config.Queries
	queryDetails, err := utils.LoadQueries(queries, arguments)
	if err != nil {
		log.Error("Error loading query configuration: %s", err.Error())
		return
	}

	for _, queryDetailsDto := range queryDetails {
		queryResults, err := utils.ExecuteQuery(arguments, queryDetailsDto, integration, sqlConnection)
		if err != nil {
			log.Error("Failed to execute query %s : %s", queryDetailsDto.Type, err.Error())
			continue
		}
		err = utils.IngestQueryMetricsInBatches(queryResults, queryDetailsDto, integration, sqlConnection)
		if err != nil {
			log.Error("Failed to ingest metrics: %s", err.Error())
			continue
		}
	}
	log.Debug("Query analysis completed")
}
