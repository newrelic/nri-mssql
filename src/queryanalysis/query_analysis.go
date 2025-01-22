package queryanalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
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

	// Validate preconditions
	isPreconditionPassed := validation.ValidatePreConditions(sqlConnection)
	if !isPreconditionPassed {
		log.Error("Error validating preconditions")
		return
	}

	utils.ValidateAndSetDefaults(&arguments)

	queryDetails, err := utils.LoadQueries(arguments)
	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	for _, queryDetailsDto := range queryDetails {
		queryResults, err := utils.ExecuteQuery(arguments, queryDetailsDto, integration, sqlConnection)
		if err != nil {
			log.Error("Failed to execute query: %s", err)
			continue
		}
		err = utils.IngestQueryMetricsInBatches(queryResults, queryDetailsDto, integration, sqlConnection)
		if err != nil {
			log.Error("Failed to ingest metrics: %s", err)
			continue
		}

		if err != nil {
			log.Error("Failed after retries: %s", err)
		}
	}
	log.Debug("Query analysis completed")

}
