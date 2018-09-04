package main

import (
	"reflect"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

// databaseNameQuery gets all database names
const databaseNameQuery = "select db_name from sys.databases"

// DatabaseNameRow is a row result in the databaseNameQuery
type DatabaseNameRow struct {
	DBName string `db:"db_name"`
}

func createDatabaseEntities(i *integration.Integration, con *SQLConnection) ([]*integration.Entity, error) {
	databaseRows := make([]*DatabaseNameRow, 0)
	if err := con.Query(&databaseRows, databaseNameQuery); err != nil {
		return nil, err
	}

	dbEntities := make([]*integration.Entity, 0, len(databaseRows))
	for _, row := range databaseRows {
		dbEntity, err := i.Entity(row.DBName, "database")
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

func (l DBMetricSetLookup) getDatabaseName(model interface{}) string {
	v := reflect.ValueOf(model)
	modeler, ok := v.Interface().(DatabaseDataModeler)
	if !ok {
		return ""
	}

	return modeler.GetDBName()
}

// createDBEntitySetLookup creates a look up of Database entity name to a metric.Set
func createDBEntitySetLookup(dbEntities []*integration.Entity) DBMetricSetLookup {
	entitySetLookup := make(map[string]*metric.Set)
	for _, dbEntity := range dbEntities {
		set := dbEntity.NewMetricSet("MssqlDatabaseSample",
			metric.Attribute{Key: "displayName", Value: dbEntity.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: dbEntity.Metadata.Namespace + ":" + dbEntity.Metadata.Name},
		)

		entitySetLookup[dbEntity.Metadata.Name] = set
	}

	return entitySetLookup
}
