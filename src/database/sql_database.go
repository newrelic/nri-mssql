// Package database contains helper methods for retrieving data about each database in the target environment
package database

import (
	"fmt"
	"reflect"

	"github.com/newrelic/infra-integrations-sdk/v3/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/v3/data/metric"
	"github.com/newrelic/infra-integrations-sdk/v3/integration"
	"github.com/newrelic/infra-integrations-sdk/v3/log"
	"github.com/newrelic/nri-mssql/src/connection"
)

const (
	// databaseNameQuery gets all database names
	databaseNameQuery                          = "select name as db_name from sys.databases where name not in ('master', 'tempdb', 'msdb', 'model', 'rdsadmin', 'distribution', 'model_msdb', 'model_replicatedmaster')"
	engineEditionQuery                         = "SELECT SERVERPROPERTY('EngineEdition') AS EngineEdition;"
	AzureSQLDatabaseEngineEditionNumber        = 5
	AzureSQLManagedInstanceEngineEditionNumber = 8
)

// NameRow is a row result in the databaseNameQuery
type NameRow struct {
	DBName string `db:"db_name"`
}

// DataModeler represents a data model for a database query
type DataModeler interface {
	GetDBName() string
}

// DataModel implements DatabaseDataModeler interface
type DataModel struct {
	DBName string `db:"db_name"`
}

// GetDBName retrieves the DBName field
func (dm DataModel) GetDBName() string {
	return dm.DBName
}

// CreateDatabaseEntities instantiates an entity for each database we're collecting
func CreateDatabaseEntities(i *integration.Integration, con *connection.SQLConnection, instanceName string) ([]*integration.Entity, error) {
	databaseRows := make([]*NameRow, 0)
	if err := con.Query(&databaseRows, databaseNameQuery); err != nil {
		return nil, err
	}

	instanceIDAttr := integration.NewIDAttribute("instance", instanceName)
	dbEntities := make([]*integration.Entity, 0, len(databaseRows))
	for _, row := range databaseRows {
		databaseIDAttr := integration.NewIDAttribute("database", row.DBName)
		dbEntity, err := i.EntityReportedVia(con.Host, row.DBName, "ms-database", instanceIDAttr, databaseIDAttr)
		if err != nil {
			return nil, err
		}

		dbEntities = append(dbEntities, dbEntity)
	}

	return dbEntities, nil
}

// DBMetricSetLookup represents a cache of Database entitiy names
// to their corresponding metric set
type DBMetricSetLookup map[string]*metric.Set

// MetricSetFromModel given a data model that implements DatabaseDataModeler
// retrieve the metric set associated with the database.
//
// ok will be false in tow cases, either model does not implement DatabaseDataModeler
// or a metric set does not exist for the database.
func (l DBMetricSetLookup) MetricSetFromModel(model interface{}) (set *metric.Set, ok bool) {
	dbName := l.getDatabaseName(model)

	if dbName != "" {
		set, ok = l[dbName]
	}

	return
}

// GetDBNames retrieves all names of databases in lookup
func (l DBMetricSetLookup) GetDBNames() []string {
	dbNames := make([]string, 0, len(l))
	for name := range l {
		dbNames = append(dbNames, name)
	}

	return dbNames
}

// getDatabaseName takes in a model and if it implements DatabaseDataModeler
// then retrieve the name of the database from that model
func (l DBMetricSetLookup) getDatabaseName(model interface{}) string {
	v := reflect.ValueOf(model)
	modeler, ok := v.Interface().(DataModeler)
	if !ok {
		return ""
	}

	return modeler.GetDBName()
}

// CreateDBEntitySetLookup creates a look up of Database entity name to a metric.Set
func CreateDBEntitySetLookup(dbEntities []*integration.Entity, instanceName, hostname string) DBMetricSetLookup {
	entitySetLookup := make(DBMetricSetLookup)
	for _, dbEntity := range dbEntities {
		set := dbEntity.NewMetricSet("MssqlDatabaseSample",
			attribute.Attribute{Key: "displayName", Value: dbEntity.Metadata.Name},
			attribute.Attribute{Key: "entityName", Value: dbEntity.Metadata.Namespace + ":" + dbEntity.Metadata.Name},
			attribute.Attribute{Key: "instance", Value: instanceName},
			attribute.Attribute{Key: "host", Value: hostname},
		)

		entitySetLookup[dbEntity.Metadata.Name] = set
	}

	return entitySetLookup
}

// GetEngineEdition retrieves the engine edition from the database.
func GetEngineEdition(connection *connection.SQLConnection) (int, error) {
	var engineEdition []int
	if err := connection.Query(&engineEdition, engineEditionQuery); err != nil {
		return 0, fmt.Errorf("error querying EngineEdition: %w", err)
	}
	if len(engineEdition) == 0 {
		log.Debug("EngineEdition query returned empty output.")
		return 0, nil
	} else {
		log.Debug("Detected EngineEdition: %d", engineEdition[0])
		return engineEdition[0], nil
	}
}

// IsAzureSQLDatabase checks if the given engine edition corresponds to Azure SQL Database with EngineEdition value of 5
func IsAzureSQLDatabase(engineEdition int) bool {
	return engineEdition == AzureSQLDatabaseEngineEditionNumber
}
