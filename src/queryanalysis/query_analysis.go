package queryanalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryanalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryanalysis/retrymechanism"
	"github.com/newrelic/nri-mssql/src/queryanalysis/utils"
	"github.com/newrelic/nri-mssql/src/queryanalysis/validation"
)

// queryPerformanceMain runs all types of analyzes
func QueryPerformanceMain(integration *integration.Integration, arguments args.ArgumentList) {
	// Create a new connection
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

	var retryMechanism retrymechanism.RetryMechanism = &retrymechanism.RetryMechanismImpl{}

	queryDetails, err := utils.LoadQueries(arguments)

	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	for _, queryDetailsDto := range queryDetails {
		err := retryMechanism.Retry(func() error {
			queryResults, err := utils.ExecuteQuery(arguments, queryDetailsDto, integration, sqlConnection)
			if err != nil {
				log.Error("Failed to execute query: %s", err)
				return err
			}
			err = utils.IngestQueryMetricsInBatches(queryResults, queryDetailsDto, integration, sqlConnection)
			if err != nil {
				log.Error("Failed to ingest metrics: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.Error("Failed after retries: %s", err)
		}
	}
}
