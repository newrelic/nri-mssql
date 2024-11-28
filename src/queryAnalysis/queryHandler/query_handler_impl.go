package queryhandler

import (
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/constants"
	"reflect"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

var _ QueryHandler = (*QueryHandlerImpl)(nil)

type QueryHandlerImpl struct{}

// ExecuteQuery executes a given query and returns the resulting rows
func (q *QueryHandlerImpl) ExecuteQuery(db *sqlx.DB, queryConfig models.QueryDetailsDto) (*sqlx.Rows, error) {
	rows, err := db.Queryx(queryConfig.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return rows, nil
}

// BindQueryResults binds query results to the specified data model using `sqlx`
func (q *QueryHandlerImpl) BindQueryResults(rows *sqlx.Rows, result interface{}) error {
	defer rows.Close()

	// Ensure result is a pointer to a slice
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result argument must be a pointer to a slice, got %T", result)
	}

	sliceValue := resultValue.Elem()
	elemType := sliceValue.Type().Elem()

	// Ensure that the element type is a struct
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("element type must be a struct, got %s", elemType.Kind())
	}

	// Iterate over the result set
	for rows.Next() {
		// Create a new instance of the element type
		elem := reflect.New(elemType).Elem()

		// Scan the current row into the new element
		if err := rows.StructScan(elem.Addr().Interface()); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Append the element to the slice
		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over rows: %w", err)
	}

	return nil
}

func (q *QueryHandlerImpl) IngestMetrics(entity *integration.Entity, results interface{}, metricName string) error {
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return fmt.Errorf("results should be a slice")
	}

	for i := 0; i < sliceValue.Len(); i++ {
		result := sliceValue.Index(i).Interface()
		resultValue := reflect.ValueOf(result)

		metricSet := entity.NewMetricSet(metricName)

		for j := 0; j < resultValue.NumField(); j++ {
			field := resultValue.Field(j)
			fieldType := resultValue.Type().Field(j)
			fieldName := fieldType.Name

			if field.Kind() == reflect.Ptr && !field.IsNil() {
				metricSet.SetMetric(fieldName, field.Elem().Interface(), DetectMetricType(field.Elem().String()))
			} else if field.Kind() != reflect.Ptr {
				metricSet.SetMetric(fieldName, field.Interface(), metric.GAUGE)
			}
		}
	}
	return nil
}

// fetchAndIngestExecutionPlan fetches the execution plan for a given query_id and ingests the data.
func fetchAndIngestExecutionPlan(db *sqlx.DB, queryID string, entity *integration.Entity, queryHandler QueryHandler) error {
	query := fmt.Sprintf(constants.ExecutionPlanQueryTemplate, queryID)

	rows, err := db.Queryx(query)
	if err != nil {
		return fmt.Errorf("failed to execute execution plan query: %w", err)
	}
	defer rows.Close()

	var executionPlanResults []models.ExecutionPlanResult
	err = queryHandler.BindQueryResults(rows, &executionPlanResults)
	if err != nil {
		return fmt.Errorf("failed to bind execution plan results: %w", err)
	}

	err = queryHandler.IngestMetrics(entity, executionPlanResults, "MssqlExecutionPlan")
	if err != nil {
		return fmt.Errorf("failed to ingest execution plan metrics: %w", err)
	}
	return nil
}

// ProcessSlowQueries processes the slow queries and fetches their execution plans.
func (q *QueryHandlerImpl) ProcessSlowQueries(db *sqlx.DB, slowQueryResults []models.TopNSlowQueryDetails, entity *integration.Entity, queryHandler QueryHandler) error {
	for _, slowQuery := range slowQueryResults {
		err := fetchAndIngestExecutionPlan(db, *slowQuery.QueryID, entity, queryHandler)
		if err != nil {
			log.Error("Failed to fetch and ingest execution plan for query_id %s: %s", slowQuery.QueryID, err)
			return err
		}
	}
	return nil
}

func DetectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}
