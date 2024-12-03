package queryAnalysis

import (
	"fmt"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/config"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/validation"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/instance"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	queryhandler "github.com/newrelic/nri-mssql/src/queryAnalysis/queryHandler"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/retryMechanism"
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
	validation.ValidatePreConditions(sqlConnection)

	// create a instanceEntity
	instanceEntity, err := instance.CreateInstanceEntity(integration, sqlConnection)
	if err != nil {
		log.Error("Error creating instance entity: %s", err.Error())
		return
	}

	// Validate preconditions
	validate, err := validation.ValidatePreConditions(sqlConnection)
	if err != nil || !validate {
		log.Error("Error validating preconditions: %s", err.Error())
		return // Abort further operations if validations fail
	}

	var retryMechanism retryMechanism.RetryMechanism = &retryMechanism.RetryMechanismImpl{}

	queryDetails, err := LoadQueries()
	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	var wg sync.WaitGroup
	resultsChannel := make(chan struct {
		queryName string
		results   interface{}
	})

	var queryhandler queryhandler.QueryHandler = &queryhandler.QueryHandlerImpl{}

	for _, queryDetailsDto := range queryDetails {
		wg.Add(1)
		go func(queryDetailsDto models.QueryDetailsDto) {
			defer wg.Done()
			fmt.Printf("Running query: %s\n", queryDetailsDto.Name)
			var results = queryDetailsDto.ResponseDetail
			err := retryMechanism.Retry(func() error {
				queryResults, err := ExecuteQuery(sqlConnection.Connection, queryDetailsDto)
				if err != nil {
					log.Error("Failed to execute query: %s", err)
					return err
				}
				//Anonymize query results
				//anonymizedQuery, err := AnonymizeQuery(queryResults)
				err = IngestQueryMetrics(instanceEntity, queryResults, queryDetailsDto)
				if err != nil {
					log.Error("Failed to ingest metrics: %s", err)
					return err
				}

				if queryDetailsDto.Name == "MSSQLTopSlowQueries" {
					for _, result := range queryResults {
						slowQuery, ok := result.(models.TopNSlowQueryDetails)
						if ok && slowQuery.QueryID != nil {
							newQueryDetails := models.QueryDetailsDto{
								Type:  "executionPlan",
								Name:  "MSSQLExecutionPlans",
								Query: fmt.Sprintf(config.ExecutionPlanQueryTemplate, *slowQuery.QueryID),
							}
							queryDetails = append(queryDetails, newQueryDetails)
						} else {
							log.Error("Failed to cast result to models.TopNSlowQueryDetails or QueryID is nil")
						}
					}
				}

				return nil
			})
			if err != nil {
				log.Error("Failed to execute and bind query results after retries: %s", err)
				return
			}
			resultsChannel <- struct {
				queryName string
				results   interface{}
			}{queryName: queryDetailsDto.Name, results: results}
		}(queryDetailsDto)
	}

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	for result := range resultsChannel {
		err := retryMechanism.Retry(func() error {
			return queryhandler.IngestMetrics(instanceEntity, result.results, result.queryName)
		})
		if err != nil {
			log.Error("Failed to ingest metrics after retries: %s", err)
		}
	}
}
