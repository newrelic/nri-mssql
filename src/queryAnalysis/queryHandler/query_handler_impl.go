package queryhandler

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

var _ QueryHandler = (*QueryHandlerImpl)(nil)

type QueryHandlerImpl struct{}

//go:embed queries.json
var queriesJSON []byte

func (q *QueryHandlerImpl) LoadQueries() ([]models.QueryDetailsDto, error) {
	var queries []models.QueryDetailsDto
	if err := json.Unmarshal(queriesJSON, &queries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries configuration: %w", err)
	}
	return queries, nil
}

func LoadQueryResponseModel(queryType string) (interface{}, error) {
	switch queryType {
	case "slowQueries":
		return &models.TopNSlowQueryDetails{}, nil
	case "waitAnalysis":
		return &models.WaitTimeAnalysis{}, nil
	case "executionPlan":
		return &models.ExecutionPlanResult{}, nil
	case "blockingSessions":
		return &models.BlockingSessionQueryDetails{}, nil
	default:
		return nil, fmt.Errorf("unknown query type: %s", queryType)
	}
}

func (q *QueryHandlerImpl) ExecuteQuery(db *sqlx.DB, queryDetailsDto models.QueryDetailsDto) ([]interface{}, error) {
	fmt.Println("Executing query...", queryDetailsDto.Name)

	rows, err := db.Queryx(queryDetailsDto.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	results := make([]interface{}, 0)

	for rows.Next() {
		switch queryDetailsDto.Type {
		case "slowQueries":
			var model models.TopNSlowQueryDetails
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			results = append(results, model)
		case "waitAnalysis":
			var model models.WaitTimeAnalysis
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			results = append(results, model)
		case "executionPlan":
			var model models.ExecutionPlanResult
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			results = append(results, model)
		case "blockingSessions":
			var model models.BlockingSessionQueryDetails
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			results = append(results, model)
		default:
			return nil, fmt.Errorf("unknown query type: %s", queryDetailsDto.Type)
		}
	}
	return results, nil

}

// IngestQueryMetrics processes and ingests query metrics into the New Relic entity
func (q *QueryHandlerImpl) IngestQueryMetrics(entity *integration.Entity, results []interface{}, queryDetailsDto models.QueryDetailsDto) error {
	for i, result := range results {
		// Convert the result into a map[string]interface{} for dynamic key-value access
		var resultMap map[string]interface{}
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("error marshalling to JSON: %w", err)
		}
		err = json.Unmarshal(data, &resultMap)
		if err != nil {
			return fmt.Errorf("error unmarshalling to map: %w", err)
		}

		// Create a new metric set with the query name
		metricSet := entity.NewMetricSet(queryDetailsDto.Name)

		// Iterate over the map and add each key-value pair as a metric
		for key, value := range resultMap {
			strValue := fmt.Sprintf("%v", value) // Convert the value to a string representation
			metricType := DetectMetricType(strValue)
			if metricType == metric.GAUGE {
				if floatValue, err := strconv.ParseFloat(strValue, 64); err == nil {
					metricSet.SetMetric(key, floatValue, metric.GAUGE)
				}
			} else {
				metricSet.SetMetric(key, strValue, metric.ATTRIBUTE)
			}
		}

		fmt.Println("Ingested Row:", i, string(data))
	}
	return nil
}

func DetectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}
