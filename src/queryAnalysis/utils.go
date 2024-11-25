package queryAnalysis

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
)

//go:embed queries.json
var queriesJSON []byte

// LoadQueriesConfig loads the query configuration from the embedded JSON file
func loadQueriesConfig() ([]models.QueryConfig, error) {
	var queries []models.QueryConfig

	// Unmarshal the JSON data into the QueryConfig struct
	if err := json.Unmarshal(queriesJSON, &queries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queries configuration: %w", err)
	}

	return queries, nil
}

// ExecuteQuery executes a given query and returns the resulting rows
func executeQuery(db *sqlx.DB, query string) (*sqlx.Rows, error) {
	rows, err := db.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return rows, nil
}

// BindResults binds query results to the specified data model using `sqlx`
func bindResults(rows *sqlx.Rows, result interface{}) error {
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

// CreateAndAddMetricSet creates and adds a metric set to the integration entity
func createAndAddMetricSet(entity *integration.Entity, results interface{}, metricName string) {
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		log.Println("results should be a slice")
		return
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
				metricSet.SetMetric(fieldName, field.Elem().Interface(), detectMetricType(field.Elem().String()))
			} else if field.Kind() != reflect.Ptr {
				metricSet.SetMetric(fieldName, field.Interface(), metric.GAUGE)
			}
		}
	}
}

func detectMetricType(value string) metric.SourceType {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return metric.ATTRIBUTE
	}

	return metric.GAUGE
}
