package metrics

import (
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/database"
)

// EngineSet is a generic struct that acts as a "bucket" for holding
// the default and Azure-specific implementations for a given resource.
type EngineSet[T any] struct {
	Default                 T
	AzureSQLDatabase        T
	AzureSQLManagedInstance T
}

// Select returns the correct implementation from the set based on the engine edition.
func (s EngineSet[T]) Select(engineEdition int) T {
	switch engineEdition {
	case database.AzureSQLDatabaseEngineEditionNumber:
		return s.AzureSQLDatabase
	case database.AzureSQLManagedInstanceEngineEditionNumber:
		return s.AzureSQLManagedInstance
	default:
		return s.Default
	}
}

// QueryDefinitionType is a custom type for identifying different query sets.
type QueryDefinitionType int

// Enum of the different query definition types.
const (
	StandardQueries = iota
	BufferQueries
	SpecificQueries
	MemoryQueries
)

var queryDefinitionSets = map[QueryDefinitionType]EngineSet[[]*QueryDefinition]{
	StandardQueries: {
		Default:                 databaseDefinitions,
		AzureSQLDatabase:        databaseDefinitionsForAzureSQLDatabase,
		AzureSQLManagedInstance: databaseDefinitionsForAzureSQLManagedInstance,
	},
	BufferQueries: {
		Default:                 databaseBufferDefinitions,
		AzureSQLDatabase:        databaseBufferDefinitionsForAzureSQLDatabase,
		AzureSQLManagedInstance: databaseBufferDefinitions,
	},
	SpecificQueries: {
		Default:                 specificDatabaseDefinitions,
		AzureSQLDatabase:        specificDatabaseDefinitionsForAzureSQLDatabase,
		AzureSQLManagedInstance: specificDatabaseDefinitions,
	},
	MemoryQueries: {
		Default:                 instanceMemoryDefinitions,
		AzureSQLDatabase:        []*QueryDefinition{},
		AzureSQLManagedInstance: instanceMemoryDefinitionsForAzureSQLManagedInstance,
	},
}

func GetQueryDefinitions(defType QueryDefinitionType, engineEdition int) []*QueryDefinition {
	definitionSet, ok := queryDefinitionSets[defType]
	if !ok {
		log.Error("Error: Invalid query definition type provided: %d", defType)
		return nil
	}
	return definitionSet.Select(engineEdition)
}
