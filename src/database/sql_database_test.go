package database

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-mssql/src/connection"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Test_createDatabaseEntities_QueryError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := connection.CreateMockSQL(t)

	mock.ExpectQuery(databaseNameQuery).WillReturnError(errors.New("error"))

	instanceName := "testInstanceName"
	if _, err := CreateDatabaseEntities(i, conn, instanceName); err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_createDatabaseEntities(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := connection.CreateMockSQL(t)

	rows := sqlmock.NewRows([]string{"db_name"}).
		AddRow("master").
		AddRow("tempdb")
	mock.ExpectQuery(databaseNameQuery).WillReturnRows(rows)

	instanceName := "testInstanceName"
	dbEntities, err := CreateDatabaseEntities(i, conn, instanceName)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	expectedEntities := []string{"master", "tempdb"}

	for i, entity := range dbEntities {
		entityName := entity.Metadata.Name
		if entityName != expectedEntities[i] {
			t.Errorf("Incorrect entity name '%s'", entityName)
		} else if entity.Metadata.Namespace != "ms-database" {
			t.Errorf("Incorrect entity namespace '%s'", entity.Metadata.Namespace)
		}
	}
}

func Test_DBMetricSetLookup_GetDBNames(t *testing.T) {
	expected := []string{"one", "three", "two"}

	lookup := make(DBMetricSetLookup)

	for _, dbName := range expected {
		lookup[dbName] = nil
	}

	out := lookup.GetDBNames()
	sort.Strings(out)

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}

func Test_DBMetricSetLookup_MetricSetFromModel_NotFound(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("one", "database")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	model := struct {
		Metric int
	}{
		1,
	}

	lookup := DBMetricSetLookup{"one": e.NewMetricSet("testSample")}

	set, ok := lookup.MetricSetFromModel(model)
	if ok || set != nil {
		t.Errorf("Expected ok 'false' and nil set got, ok '%t' and '%+v' set", ok, set)
	}
}

func Test_DBMetricSetLookup_MetricSetFromModel_Found(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("one", "database")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	model := struct {
		DataModel
	}{
		DataModel{
			DBName: "one",
		},
	}

	expectedSet := e.NewMetricSet("testSample")

	lookup := DBMetricSetLookup{"one": expectedSet}

	set, ok := lookup.MetricSetFromModel(model)
	if !ok {
		t.Errorf("Expected ok 'true' got %t", ok)
	} else if !reflect.DeepEqual(set, expectedSet) {
		t.Errorf("Expected %+v got %+v", expectedSet, set)
	}
}

func Test_createDBEntitySetLookUp(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	entities := make([]*integration.Entity, 0, 2)

	masterEntity, err := i.Entity("master", "database")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}
	tempdbEntity, err := i.Entity("tempdb", "database")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	entities = append(entities, masterEntity, tempdbEntity)

	expected := DBMetricSetLookup{
		"master": masterEntity.NewMetricSet("MssqlDatabaseSample",
			metric.Attribute{Key: "displayName", Value: "master"},
			metric.Attribute{Key: "entityName", Value: "database:master"},
			metric.Attribute{Key: "instance", Value: "MSSQL"},
			metric.Attribute{Key: "host", Value: "myHost"},
		),
		"tempdb": tempdbEntity.NewMetricSet("MssqlDatabaseSample",
			metric.Attribute{Key: "displayName", Value: "tempdb"},
			metric.Attribute{Key: "entityName", Value: "database:tempdb"},
			metric.Attribute{Key: "instance", Value: "MSSQL"},
			metric.Attribute{Key: "host", Value: "myHost"},
		),
	}

	out := CreateDBEntitySetLookup(entities, "MSSQL", "myHost")
	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}
