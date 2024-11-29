package queryhandler

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

type QueryHandler interface {
	BindQueryResults(rows *sqlx.Rows, queryDetailsDto models.QueryDetailsDto) ([]interface{}, error)
	LoadQueries() ([]models.QueryDetailsDto, error)
	ExecuteQuery(db *sqlx.DB, queryConfig models.QueryDetailsDto) ([]interface{}, error)
	IngestQueryMetrics(entity *integration.Entity, results []interface{}, queryDetailsDto models.QueryDetailsDto) error
}
