package queryAnalysis

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/instance"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

// RunAnalysis runs all types of analyses
func RunAnalysis(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	fmt.Println("Starting query analysis...")

	AnalyzeSlowQueries(instanceEntity, connection, arguments)
	AnalyzeExecutionPlans(instanceEntity, connection, arguments)
	AnalyzeWaits()

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
	var results interface{}

	for _, queryConfig := range queries {
		fmt.Printf("Running query: %s\n", queryConfig.Name)

		// Execute the query and store the results in the executionPlans slice.
		rows, err := executeQuery(sqlConnection.Connection, queryConfig.Query)
		if err != nil {
			log.Error("Could not execute query for execution plan: %s", err.Error())
			return
		}
		defer rows.Close()

		switch queryConfig.Type {
		case "slowQueries":
			var slowQueryResults []models.TopNSlowQueryDetails
			err := bindResults(rows, &slowQueryResults)
			if err != nil {
				log.Error("Failed to bind results: %s", err)
			}
			results = slowQueryResults
			// Process results as needed
			fmt.Println(slowQueryResults)

		case "waitAnalysis":
			var waitAnalysisResults []models.WaitTimeAnalysis
			err := bindResults(rows, &waitAnalysisResults)
			if err != nil {
				log.Error("Failed to bind results: %s", err)
			}
			results = waitAnalysisResults
			// Process results as needed
			fmt.Println(waitAnalysisResults)

		case "executionPlan":
			var executionPlanResults []models.QueryExecutionPlan
			err := bindResults(rows, &executionPlanResults)
			if err != nil {
				log.Error("Failed to bind results: %s", err)
			}
			results = executionPlanResults
			// Process results as needed
			fmt.Println(executionPlanResults)

		case "blockingSessions":
			var blockingSessionsResults []models.BlockingSessionQueryDetails
			err := bindResults(rows, &blockingSessionsResults)
			if err != nil {
				log.Error("Failed to bind results: %s", err)
			}
			results = blockingSessionsResults
			// Process results as needed
			fmt.Println(blockingSessionsResults)

		default:
			log.Info("Query type %s is not supported", queryConfig.Type)
		}

		createAndAddMetricSet(instanceEntity, results, queryConfig.Name)
	}
}
