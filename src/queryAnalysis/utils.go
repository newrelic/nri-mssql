package queryAnalysis

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"io/ioutil"
	"reflect"
)

// LoadQueriesConfig loads the query configuration from a JSON file
func loadQueriesConfig() []models.QueryConfig {
	file, err := ioutil.ReadFile("src/queryanalysis/queries.json")
	if err != nil {
		panic(err)
	}

	var queries []models.QueryConfig
	err = json.Unmarshal(file, &queries)
	if err != nil {
		panic(err)
	}

	return queries
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

	return rows.Err()
}
