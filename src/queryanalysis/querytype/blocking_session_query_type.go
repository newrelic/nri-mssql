package querytype

import (
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryanalysis/models"
	"github.com/newrelic/nri-mssql/src/queryanalysis/utils"
)

type BlockingSessionsType struct{}

func (b *BlockingSessionsType) Bind(results *[]interface{}, queryIDs *[]models.HexString, rows *sqlx.Rows) error {
	var model models.BlockingSessionQueryDetails
	if err := rows.StructScan(&model); err != nil {
		return err
	}
	if model.BlockingQueryText != nil {
		*model.BlockingQueryText = utils.AnonymizeQueryText(*model.BlockingQueryText)
	}
	if model.BlockedQueryText != nil {
		*model.BlockedQueryText = utils.AnonymizeQueryText(*model.BlockedQueryText)
	}
	*results = append(*results, model)
	return nil
}
