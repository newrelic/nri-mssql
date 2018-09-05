package main

// QueryDefinition defines a single query with it's associated
// data model which has struct tags for metric.Set
type QueryDefinition struct {
	query      string
	dataModels interface{}
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
	return qd.dataModels
}
