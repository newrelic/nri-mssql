package queriesLoader

import "github.com/newrelic/nri-mssql/src/queryAnalysis/models"

type QueriesLoader interface {
	LoadQueries() ([]models.QueryDetailsDto, error)
}
