package queryAnalysis

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/args"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/config"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/connection"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/instance"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"regexp"
	"strconv"
)

func LoadQueries(arguments args.ArgumentList) ([]models.QueryDetailsDto, error) {
	var queries []models.QueryDetailsDto = config.Queries

	for i := range queries {
		switch queries[i].Type {
		case "slowQueries":
			queries[i].Query = fmt.Sprintf(queries[i].Query, arguments.FetchInterval, arguments.QueryCountThreshold, arguments.QueryResponseTimeThreshold)
		case "waitAnalysis":
			queries[i].Query = fmt.Sprintf(queries[i].Query, arguments.FetchInterval, arguments.FetchInterval, arguments.QueryCountThreshold)
		case "blockingSessions":
			continue
		default:
			fmt.Println("Unknown query type:", queries[i].Type)
		}
	}

	return queries, nil
}

func ExecuteQuery(interval int,
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection) ([]interface{}, error) {
	fmt.Println("Executing query...", queryDetailsDto.Name)

	rows, err := sqlConnection.Connection.Queryx(queryDetailsDto.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return BindQueryResults(interval, rows, queryDetailsDto, integration, sqlConnection)
}

// BindQueryResults binds query results to the specified data model using `sqlx`
func BindQueryResults(interval int,
	rows *sqlx.Rows,
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection) ([]interface{}, error) {

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
			AnonymizeQueryText(model.QueryText)

			results = append(results, model)

			// fetch and generate execution plan
			GenerateAndInjestExecutionPlan(interval, integration, sqlConnection, *model.QueryID)

		case "waitAnalysis":
			var model models.WaitTimeAnalysis
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			AnonymizeQueryText(model.QueryText)

			results = append(results, model)
		case "blockingSessions":
			var model models.BlockingSessionQueryDetails
			if err := rows.StructScan(&model); err != nil {
				fmt.Println("Could not scan row: ", err)
				continue
			}
			AnonymizeQueryText(model.BlockedQueryText)
			AnonymizeQueryText(model.BlockingQueryText)
			results = append(results, model)
		default:
			return nil, fmt.Errorf("unknown query type: %s", queryDetailsDto.Type)
		}
	}
	return results, nil

}

func GenerateAndInjestExecutionPlan(interval int,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection,
	queryId models.HexString) {

	hexQueryId := fmt.Sprintf("%s", queryId)
	executionPlanQuery := fmt.Sprintf(config.ExecutionPlanQueryTemplate, hexQueryId, interval)

	var model models.ExecutionPlanResult

	rows, err := sqlConnection.Connection.Queryx(executionPlanQuery)
	if err != nil {
		log.Error("Failed to execute query: %s", err)
		return
	}
	defer rows.Close()

	results := make([]interface{}, 0)

	for rows.Next() {
		if err := rows.StructScan(&model); err != nil {
			log.Error("Could not scan row: %s", err)
			return
		}
		AnonymizeQueryText(model.SQLText)
		results = append(results, model)
	}

	queryDetailsDto := models.QueryDetailsDto{
		Name:  "MSSQLQueryExecutionPlans",
		Query: "",
		Type:  "executionPlan",
	}

	// Ingest the execution plan
	if err := IngestQueryMetricsInBatches(results, queryDetailsDto, integration, sqlConnection); err != nil {
		log.Error("Failed to ingest execution plan: %s", err)
	}
}

func IngestQueryMetricsInBatches(results []interface{},
	queryDetailsDto models.QueryDetailsDto,
	integration *integration.Integration,
	sqlConnection *connection.SQLConnection) error {

	const batchSize = 100

	for start := 0; start < len(results); start += batchSize {
		end := start + batchSize
		if end > len(results) {
			end = len(results)
		}

		batchResult := results[start:end]
		fmt.Printf("Processing batch of %s: startIndex: %d to endIndex: %d totalLength: %d \n", queryDetailsDto.Name, start, end, len(results))

		if err := IngestQueryMetrics(batchResult, queryDetailsDto, integration, sqlConnection); err != nil {
			return fmt.Errorf("error ingesting batch from %d to %d: %w", start, end, err)
		}
	}

	return nil
}

// IngestQueryMetrics processes and ingests query metrics into the New Relic entity
func IngestQueryMetrics(results []interface{}, queryDetailsDto models.QueryDetailsDto, integration *integration.Integration, sqlConnection *connection.SQLConnection) error {

	instanceEntity, err := instance.CreateInstanceEntity(integration, sqlConnection)

	for _, result := range results {
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
		metricSet := instanceEntity.NewMetricSet(queryDetailsDto.Name)

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
	}
	err = integration.Publish()
	if err != nil {
		return err
	}
	integration.Clear()

	return nil
}

func DetectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}

func AnonymizeQueryText(query *string) {

	re := regexp.MustCompile(`'[^']*'|\d+|".*?"`)

	anonymizedQuery := re.ReplaceAllString(*query, "?")

	*query = anonymizedQuery
}
