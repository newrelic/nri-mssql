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

	queries, err := loadQueriesConfig()
	if err != nil {
		log.Error("Error loading query configuration: %v", err)
		return
	}

	instanceEntity, err := instance.CreateInstanceEntity(integration, sqlConnection)
	if err != nil {
		log.Error("Error creating instance entity: %s", err.Error())
		return
	}

	var wg sync.WaitGroup
	resultsChannel := make(chan struct {
		queryName string
		results   interface{}
	})

	for _, queryConfig := range queries {
		wg.Add(1)
		go func(queryConfig models.QueryConfig) {
			defer wg.Done()
			fmt.Printf("Running query: %s\n", queryConfig.Name)

			// Execute the query and store the results in the executionPlans slice.
			rows, err := executeQuery(sqlConnection.Connection, queryConfig.Query)
			if err != nil {
				log.Error("Could not execute query for execution plan: %s", err.Error())
				return
			}
			defer rows.Close()

			var results interface{}
			switch queryConfig.Type {
			case "slowQueries":
				var slowQueryResults []models.TopNSlowQueryDetails
				err := bindResults(rows, &slowQueryResults)
				if err != nil {
					log.Error("Failed to bind results: %s", err)
				}
				results = slowQueryResults

			case "waitAnalysis":
				var waitAnalysisResults []models.WaitTimeAnalysis
				err := bindResults(rows, &waitAnalysisResults)
				if err != nil {
					log.Error("Failed to bind results: %s", err)
				}
				results = waitAnalysisResults

			case "executionPlan":
				var executionPlanResults []models.QueryExecutionPlan
				err := bindResults(rows, &executionPlanResults)
				if err != nil {
					log.Error("Failed to bind results: %s", err)
				}
				results = executionPlanResults

			case "blockingSessions":
				var blockingSessionsResults []models.BlockingSessionQueryDetails
				err := bindResults(rows, &blockingSessionsResults)
				if err != nil {
					log.Error("Failed to bind results: %s", err)
				}
				results = blockingSessionsResults

			default:
				log.Info("Query type %s is not supported", queryConfig.Type)
				return
			}
			resultsChannel <- struct {
				queryName string
				results   interface{}
			}{queryName: queryConfig.Name, results: results}
		}(queryConfig)
	}

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	for result := range resultsChannel {
		createAndAddMetricSet(instanceEntity, result.results, result.queryName)
	}
}
