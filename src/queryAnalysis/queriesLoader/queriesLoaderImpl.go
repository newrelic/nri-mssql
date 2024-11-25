package queriesloader

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

var _ QueriesLoader = (*QueriesLoaderImpl)(nil)

type QueriesLoaderImpl struct{}

//go:embed queries.json
var queriesJSON []byte

// LoadQueries loads the queries and maps them to their respective models
func (q *QueriesLoaderImpl) LoadQueries() ([]models.QueryDetailsDto, error) {
	var queries []models.QueryDetailsDto
	if err := json.Unmarshal(queriesJSON, &queries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries configuration: %w", err)
	}
	return queries, nil
}
