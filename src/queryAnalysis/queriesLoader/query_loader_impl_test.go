package queriesLoader

import (
	"testing"

	"github.com/newrelic/nri-mssql/src/queryAnalysis/models"
	"github.com/stretchr/testify/assert"
)

// Mock queries JSON data for testing
var mockQueriesJSON = []byte(`[
	{
		"name": "MSSQLTopSlowQueries",
		"query": "select * from sys.databases",
		"type": "slowQueries"
	},
	{
		"name": "MSSQLWaitTimeAnalysis",
		"query": "select * from sys.databases",
		"type": "waitAnalysis"
	},
	{
		"name": "MSSQLQueryExecutionPlan",
		"query": "select * from sys.databases",
		"type": "executionPlan"
	},
	{
		"name": "MSSQLBlockingSessionQueries",
		"query": "select * from sys.databases",
		"type": "blockingSessions"
	}
]`)

func TestLoadQueries_Success(t *testing.T) {
	queriesLoader := &QueriesLoaderImpl{}

	// Override the embedded queries.json with mock data
	queriesJSON = mockQueriesJSON

	queries, err := queriesLoader.LoadQueries()
	assert.NoError(t, err)

	assert.Equal(t, 4, len(queries))

	// Validate each response detail type
	for _, query := range queries {
		switch query.Type {
		case "slowQueries":
			assert.IsType(t, &models.TopNSlowQueryDetails{}, query.ResponseDetail)
		case "waitAnalysis":
			assert.IsType(t, &models.WaitTimeAnalysis{}, query.ResponseDetail)
		case "executionPlan":
			assert.IsType(t, &models.ExecutionPlanResult{}, query.ResponseDetail)
		case "blockingSessions":
			assert.IsType(t, &models.BlockingSessionQueryDetails{}, query.ResponseDetail)
		default:
			t.Fatalf("unexpected query type: %s", query.Type)
		}
	}
}

func TestLoadQueries_UnmarshalError(t *testing.T) {
	queriesLoader := &QueriesLoaderImpl{}

	// Introduce incorrect JSON format to induce an unmarshalling error
	badJSON := []byte(`[{
		"name": "BadQuery",
		"query": "select * from sys.databases"
		"type": "invalidType",`)

	queriesJSON = badJSON

	_, err := queriesLoader.LoadQueries()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal queries configuration")
}

func TestLoadQueries_UnknownQueryType(t *testing.T) {
	queriesLoader := &QueriesLoaderImpl{}

	// Add an invalid type to the mock JSON data
	invalidJSON := []byte(`[
		{
			"name": "UnknownTypeQuery",
			"query": "select * from sys.databases",
			"type": "unknownType"
		}
	]`)

	queriesJSON = invalidJSON

	_, err := queriesLoader.LoadQueries()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown query type")
}
