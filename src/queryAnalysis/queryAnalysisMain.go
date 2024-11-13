package queryanalysis

import (
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/connection"
)

func QueryAnalysisMain(instanceEntity *integration.Entity, connection *connection.SQLConnection, arguments args.ArgumentList) {
	log.Info("Querying SQL Server for query analysis metrics")
	TopNSlowQueryAnalysis(instanceEntity, connection, arguments)
}
