package querytype

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"github.com/newrelic/nri-mssql/src/queryanalysis/utils"
)

// Implementation for SlowQueries
type SlowQueryType struct{}

func (s *SlowQueryType) Bind(results *[]interface{}, queryIDs *[]models.HexString, rows *sqlx.Rows) error {
	var model models.TopNSlowQueryDetails
	if err := rows.StructScan(&model); err != nil {
		return err
	}
	if model.QueryText != nil {
		*model.QueryText = utils.AnonymizeQueryText(*model.QueryText)
	}
	*results = append(*results, model)

	if model.QueryID != nil {
		*queryIDs = append(*queryIDs, *model.QueryID)
	}
	return nil
}
