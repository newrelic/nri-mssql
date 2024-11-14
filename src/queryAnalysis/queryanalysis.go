package queryAnalysis

import (
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/instance"
)

// RunAnalysis runs all types of analyses
func RunAnalysis(integration *integration.Integration, arguments args.ArgumentList) {
	fmt.Println("Starting query analysis...")

	// Create a new connection
	con, err := connection.NewConnection(&arguments)
	if err != nil {
		log.Error("Error creating connection to SQL Server: %s", err.Error())
		return
	}

	// Create the entity for the instance
	instanceEntity, err := instance.CreateInstanceEntity(integration, con)
	if err != nil {
		log.Error("Unable to create entity for instance: %s", err.Error())
		return
	}

	AnalyzeSlowQueries(instanceEntity, con, arguments)
	AnalyzeExecutionPlans(instanceEntity, con, arguments)
	AnalyzeWaits()

	fmt.Println("Query analysis completed.")
}
