package queryAnalysis

import (
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
)

// RunAnalysis runs all types of analyses
func RunAnalysis(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	fmt.Println("Starting query analysis...")

	AnalyzeSlowQueries(instanceEntity, connection, arguments)
	AnalyzeExecutionPlans(instanceEntity, connection, arguments)
	AnalyzeWaits()

	fmt.Println("Query analysis completed.")

}
