package queryAnalysis

import (
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/retryMechanism"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/validation"
)

// queryPerformanceMain runs all types of analyses
func QueryPerformanceMain(integration *integration.Integration, arguments args.ArgumentList) {

	fmt.Println("Starting query analysis...")

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

	var retryMechanism retryMechanism.RetryMechanism = &retryMechanism.RetryMechanismImpl{}

	queryDetails, err := LoadQueries(arguments)

	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	for _, queryDetailsDto := range queryDetails {
		err := retryMechanism.Retry(func() error {
			queryResults, err := ExecuteQuery(arguments, queryDetailsDto, integration, sqlConnection)
			if err != nil {
				log.Error("Failed to execute query: %s", err)
				return err
			}
			err = IngestQueryMetricsInBatches(queryResults, queryDetailsDto, integration, sqlConnection)
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
