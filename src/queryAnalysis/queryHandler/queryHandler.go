package queryhandler

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

type QueryHandler interface {
	ExecuteQuery(db *sqlx.DB, queryConfig models.QueryDetailsDto) (*sqlx.Rows, error)
	BindQueryResults(rows *sqlx.Rows, result interface{}) error
	IngestMetrics(entity *integration.Entity, results interface{}, metricName string) error
}
