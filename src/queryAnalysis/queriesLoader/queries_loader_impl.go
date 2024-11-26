package queriesloader

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

var _ QueriesLoader = (*QueriesLoaderImpl)(nil)

type QueriesLoaderImpl struct{}

//go:embed queries.json
var queriesJSON []byte

var queryTypeToResponseDetailMap = map[string]interface{}{
	"slowQueries":      models.TopNSlowQueryDetails{},
	"waitAnalysis":     models.WaitTimeAnalysis{},
	"executionPlan":    models.QueryExecutionPlan{},
	"blockingSessions": models.BlockingSessionQueryDetails{},
}

// LoadQueries loads the queries and maps them to their respective models
func (q *QueriesLoaderImpl) LoadQueries() ([]models.QueryDetailsDto, error) {
	var queries []models.QueryDetailsDto
	if err := json.Unmarshal(queriesJSON, &queries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries configuration: %w", err)
	}
	for i, query := range queries {
		if responseDetail, ok := queryTypeToResponseDetailMap[query.Type]; ok {
			queries[i].ResponseDetail = reflect.New(reflect.TypeOf(responseDetail)).Interface()
		} else {
			return nil, fmt.Errorf("unknown query type: %s", query.Type)
		}
	}

	return queries, nil
}
