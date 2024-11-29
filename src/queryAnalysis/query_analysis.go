package queryAnalysis

import (
	"fmt"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/instance"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	queryhandler "github.com/newrelic/nri-mssql/src/queryAnalysis/queryHandler"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/retryMechanism"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/validation"
)

// RunAnalysis runs all types of analyses
func RunAnalysis(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	fmt.Println("Starting query analysis...")

	// Create a new connection
	sqlConnection, err := connection.NewConnection(&arguments)
	if err != nil {
		log.Error("Error creating connection to SQL Server: %s", err.Error())
		return
	}
	validation.ValidatePreConditions(sqlConnection)

	// create a instanceEntity
	instanceEntity, err := instance.CreateInstanceEntity(integration, sqlConnection)
	if err != nil {
		log.Error("Error creating instance entity: %s", err.Error())
		return
	}

	var queryhandler queryhandler.QueryHandler = &queryhandler.QueryHandlerImpl{}
	var retryMechanism retryMechanism.RetryMechanism = &retryMechanism.RetryMechanismImpl{}

	queriesDetails, err := queryhandler.LoadQueries()
	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	var wg sync.WaitGroup

	for _, queryDetailsDto := range queriesDetails {
		wg.Add(1)

		// Launch a goroutine for each queryDetailsDto
		go func(queryDetailsDto models.QueryDetailsDto) {
			defer wg.Done()

			err := retryMechanism.Retry(func() error {
				results, err := queryhandler.ExecuteQuery(sqlConnection.Connection, queryDetailsDto)
				if err != nil {
					log.Error("Failed to execute query: %s", err)
					return err
				}
				err = queryhandler.IngestQueryMetrics(instanceEntity, results, queryDetailsDto)
				if err != nil {
					log.Error("Failed to ingest metrics: %s", err)
					return err
				}
				return nil
			})

			if err != nil {

				log.Error("Failed after retries: %s", err)
			}
		}(queryDetailsDto) // Pass queryDetailsDto as a parameter to avoid closure capture issues
	}

	// Wait for all goroutines to complete
	wg.Wait()

}
