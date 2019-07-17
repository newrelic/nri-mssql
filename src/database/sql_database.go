// Package database contains helper methods for retrieving data about each database in the target environment
package database

import (
	"reflect"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
)

// databaseNameQuery gets all database names
const databaseNameQuery = "select name as db_name from sys.databases"

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
			metric.Attribute{Key: "displayName", Value: dbEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: dbEntity.Metadata.Namespace + ":" + dbEntity.Metadata.Name},
			metric.Attribute{Key: "instance", Value: instanceName},
			metric.Attribute{Key: "host", Value: hostname},
		)

		entitySetLookup[dbEntity.Metadata.Name] = set
	}

	return entitySetLookup
}
