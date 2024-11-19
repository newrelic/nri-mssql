package queryAnalysis

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"io/ioutil"
	"log"
	"reflect"
)

// LoadQueriesConfig loads the query configuration from a JSON file
func loadQueriesConfig() ([]models.QueryConfig, error) {
	// Read the configuration file from the specified path
	file, err := ioutil.ReadFile("src/queryanalysis/queries.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read queries configuration file: %w", err)
	}

	var queries []models.QueryConfig
	// Unmarshal the JSON data into the QueryConfig struct
	err = json.Unmarshal(file, &queries)
	if err != nil {
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

	// Get the type info of the slice element
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result argument must be a pointer to a slice")
	}

	sliceValue := resultValue.Elem()
	elemType := sliceValue.Type().Elem()

	// Iterate over the result set
	for rows.Next() {
		// Create a new instance of the element type
		elem := reflect.New(elemType).Elem()

		// Scan the current row into the new element
		if err := rows.StructScan(elem.Addr().Interface()); err != nil {
			return err
		}

		// Append the element to the slice
		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	defer rows.Close()
	return rows.Err()
}

func createAndAddMetricSet(entity *integration.Entity, results interface{}, metricName string) {
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		log.Println("results should be a slice")
		return
	}

	for i := 0; i < sliceValue.Len(); i++ {
		result := sliceValue.Index(i).Interface()
		resultValue := reflect.ValueOf(result)

		attributes := []attribute.Attribute{}
		metricSet := entity.NewMetricSet(metricName, attributes...)

		for j := 0; j < resultValue.NumField(); j++ {
			field := resultValue.Field(j)
			fieldType := resultValue.Type().Field(j)
			fieldName := fieldType.Name

			// Set each field as a metric
			if field.Kind() == reflect.Ptr && !field.IsNil() {
				metricSet.SetMetric(fieldName, field.Elem().Interface(), metric.GAUGE)
			} else if field.Kind() != reflect.Ptr {
				metricSet.SetMetric(fieldName, field.Interface(), metric.GAUGE)
			}
		}
	}
}
