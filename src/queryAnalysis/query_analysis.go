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
	queriesLoader "github.com/newrelic/nri-mssql/src/queryAnalysis/queriesLoader"
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

	var queriesLoader queriesLoader.QueriesLoader = &queriesLoader.QueriesLoaderImpl{}
	var retryMechanism retryMechanism.RetryMechanism = &retryMechanism.RetryMechanismImpl{}
	queriesDetails, err := queriesLoader.LoadQueries()
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

	for _, queryDetailsDto := range queriesDetails {
		wg.Add(1)
		go func(queryDetailsDto models.QueryDetailsDto) {
			defer wg.Done()
			fmt.Printf("Running query: %s\n", queryDetailsDto.Name)
			var results = queryDetailsDto.ResponseDetail
			err := retryMechanism.Retry(func() error {
				rows, err := queryhandler.ExecuteQuery(sqlConnection.Connection, queryDetailsDto)
				if err != nil {
					log.Error("Failed to execute query: %s", err)
					return err
				}
				return queryhandler.BindQueryResults(rows, &results)
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
