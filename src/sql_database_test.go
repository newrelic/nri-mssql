package main

import (
	"errors"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Test_createDatabaseEntities_QueryError(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)
	defer conn.Close()

	mock.ExpectQuery(databaseNameQuery).WillReturnError(errors.New("error"))

	if _, err := createDatabaseEntities(i, conn); err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_createDatabaseEntities(t *testing.T) {
	i, err := integration.New("test", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
		t.FailNow()
	}

	conn, mock := createMockSQL(t)
	defer conn.Close()

	rows := sqlmock.NewRows([]string{"db_name"}).
		AddRow("master").
		AddRow("tempdb")
	mock.ExpectQuery(databaseNameQuery).WillReturnRows(rows)

	dbEntities, err := createDatabaseEntities(i, conn)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	for _, entity := range dbEntities {
		entityName := entity.Metadata.Name
		if entityName != "master" && entityName != "tempdb" {
			t.Errorf("Incorrect entity name '%s'", entityName)
		} else if entity.Metadata.Namespace != "database" {
			t.Errorf("Incorrect entity namespace '%s'", entity.Metadata.Namespace)
		}
	}
}
