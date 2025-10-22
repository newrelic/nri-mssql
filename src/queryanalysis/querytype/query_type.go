package querytype

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
)

type QueryType interface {
	Bind(results *[]interface{}, queryIDs *[]models.HexString, rows *sqlx.Rows) error
}
