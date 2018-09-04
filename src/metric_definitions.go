package main

import (
	"reflect"
)

// QueryDefinition defines a single query with it's associated
// data model which has struct tags for metric.Set
type QueryDefinition struct {
	query      string
	dataModels []interface{}
}

// QueryModifier is a function that takes in a query, does any modification
// and returns the query
type QueryModifier func(string) string

// GetQuery retrieves the query for a QueryDefinition
func (qd QueryDefinition) GetQuery(modifiers ...QueryModifier) string {
	modifiedQuery := qd.query

	for _, modifier := range modifiers {
		modifiedQuery = modifier(modifiedQuery)
	}

	return modifiedQuery
}

// GetDataModels retrieves the DataModels to be passed to the sqlx
// call for results to be martialed into
func (qd QueryDefinition) GetDataModels() interface{} {
	return &qd.dataModels
}

// copyToInterfaceSlice is a utility function to copy a anonymous struct slice
// into an interface slice via reflection. This helps use utilize struct
// field tags for sqlx and the NR SDK.
func copyToInterfaceSlice(models interface{}) []interface{} {
	v := reflect.ValueOf(models)
	interfaceSlice := make([]interface{}, v.Len())

	for i := 0; i < v.Len(); i++ {
		interfaceSlice[i] = v.Index(i).Interface()
	}

	return interfaceSlice
}
